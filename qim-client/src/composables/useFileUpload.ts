import { ref } from 'vue'
import { fileApi, type InitUploadResponse } from '../api/file'
import { useUploadStore } from '../stores/upload'
import SparkMD5 from 'spark-md5'

/**
 * 文件上传 Composable
 * 整合 MD5 计算、分片上传、进度管理等功能
 */

// MD5 Worker 实例
let md5Worker: Worker | null = null

// 获取或创建 MD5 Worker
function getMD5Worker(): Worker {
  if (!md5Worker) {
    md5Worker = new Worker(new URL('../workers/md5.worker.ts', import.meta.url), {
      type: 'module'
    })
  }
  return md5Worker
}

// 分片策略配置
interface ChunkStrategy {
  chunkSize: number
  description: string
}

/**
 * 智能分片策略
 * 根据文件大小自动选择合适的分片大小
 */
export function getChunkStrategy(fileSize: number): ChunkStrategy {
  const MB = 1024 * 1024
  const GB = 1024 * MB

  if (fileSize < 10 * MB) {
    // 小于 10MB：不分片
    return { chunkSize: fileSize, description: '小文件不分片' }
  } else if (fileSize < 100 * MB) {
    // 10MB - 100MB：2MB 分片
    return { chunkSize: 2 * MB, description: '2MB 分片' }
  } else if (fileSize < 500 * MB) {
    // 100MB - 500MB：5MB 分片
    return { chunkSize: 5 * MB, description: '5MB 分片' }
  } else if (fileSize < GB) {
    // 500MB - 1GB：10MB 分片
    return { chunkSize: 10 * MB, description: '10MB 分片' }
  } else {
    // 大于 1GB：20MB 分片
    return { chunkSize: 20 * MB, description: '20MB 分片' }
  }
}

/**
 * 文件分片
 * @param file 要分片的文件
 * @param chunkSize 分片大小
 * @returns 分片数组
 */
export function splitFile(file: File, chunkSize: number): Blob[] {
  const chunks: Blob[] = []
  let start = 0

  while (start < file.size) {
    const end = Math.min(start + chunkSize, file.size)
    chunks.push(file.slice(start, end))
    start = end
  }

  return chunks
}

/**
 * 计算分片 MD5
 * @param chunk 分片数据
 * @returns MD5 哈希值
 */
export async function calculateChunkMD5(chunk: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    const spark = new SparkMD5.ArrayBuffer()

    reader.onload = (e) => {
      if (e.target?.result instanceof ArrayBuffer) {
        spark.append(e.target.result)
        resolve(spark.end())
      } else {
        reject(new Error('读取分片失败'))
      }
    }

    reader.onerror = () => {
      reject(new Error('读取分片失败'))
    }

    reader.readAsArrayBuffer(chunk)
  })
}

/**
 * 计算文件 MD5
 * 使用 Web Worker 在后台线程计算，避免阻塞主线程
 * @param file 要计算的文件
 * @returns MD5 哈希值
 */
export async function calculateMD5(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const worker = getMD5Worker()

    const handleMessage = (e: MessageEvent) => {
      const data = e.data

      if (data.error) {
        worker.removeEventListener('message', handleMessage)
        reject(new Error(data.error))
      } else if (data.hash) {
        worker.removeEventListener('message', handleMessage)
        resolve(data.hash)
      }
      // 进度消息可以忽略，或者通过回调通知
    }

    worker.addEventListener('message', handleMessage)
    worker.postMessage({ file })
  })
}

/**
 * 上传任务管理器
 */
interface UploadManager {
  uploadId: string
  file: File
  folderId: number | null
  fileHash: string
  chunks: Blob[]
  uploadedChunks: Set<number>
  retryCount: Map<number, number>
  abortController: AbortController
}

// 活跃的上传任务
const activeUploads = new Map<string, UploadManager>()

// 最大并发数
const MAX_CONCURRENT_UPLOADS = 3

// 最大重试次数
const MAX_RETRY_COUNT = 3

/**
 * 初始化上传
 * @param file 要上传的文件
 * @param folderId 目标文件夹 ID
 * @returns 初始化响应
 */
export async function initUpload(
  file: File,
  folderId?: number
): Promise<InitUploadResponse> {
  // 计算 MD5
  const fileHash = await calculateMD5(file)

  // 调用初始化 API
  const response = await fileApi.initUpload({
    filename: file.name,
    file_size: file.size,
    file_hash: fileHash,
    folder_id: folderId ?? null,
    mime_type: file.type || 'application/octet-stream'
  })

  if (response.data.code !== 200) {
    throw new Error('初始化上传失败')
  }

  return response.data.data
}

/**
 * 上传单个分片
 */
async function uploadSingleChunk(
  uploadId: string,
  chunk: Blob,
  chunkIndex: number,
  chunkHash: string
): Promise<void> {
  const formData = new FormData()
  formData.append('upload_id', uploadId)
  formData.append('chunk_index', String(chunkIndex))
  formData.append('chunk_hash', chunkHash)
  formData.append('chunk', chunk)

  const response = await fileApi.uploadChunk(formData)

  if (response.data.code !== 200) {
    throw new Error(`分片 ${chunkIndex} 上传失败`)
  }
}

/**
 * 上传分片（带重试）
 */
async function uploadChunkWithRetry(
  manager: UploadManager,
  chunkIndex: number,
  onProgress?: (uploadedChunks: number) => void
): Promise<void> {
  const chunk = manager.chunks[chunkIndex]
  const chunkHash = await calculateChunkMD5(chunk)

  let retryCount = manager.retryCount.get(chunkIndex) || 0

  while (retryCount < MAX_RETRY_COUNT) {
    try {
      await uploadSingleChunk(
        manager.uploadId,
        chunk,
        chunkIndex,
        chunkHash
      )

      // 上传成功
      manager.uploadedChunks.add(chunkIndex)
      manager.retryCount.delete(chunkIndex)

      if (onProgress) {
        onProgress(manager.uploadedChunks.size)
      }

      return
    } catch (error) {
      retryCount++
      manager.retryCount.set(chunkIndex, retryCount)

      if (retryCount >= MAX_RETRY_COUNT) {
        throw new Error(`分片 ${chunkIndex} 上传失败，已重试 ${MAX_RETRY_COUNT} 次`)
      }

      // 等待一段时间后重试
      await new Promise(resolve => setTimeout(resolve, 1000 * retryCount))
    }
  }
}

/**
 * 并发上传分片
 */
async function uploadChunksConcurrently(
  manager: UploadManager,
  onProgress?: (uploadedChunks: number) => void
): Promise<void> {
  const pendingChunks: number[] = []

  // 找出需要上传的分片
  for (let i = 0; i < manager.chunks.length; i++) {
    if (!manager.uploadedChunks.has(i)) {
      pendingChunks.push(i)
    }
  }

  // 并发上传，最多 3 个
  const uploadQueue: Promise<void>[] = []

  for (const chunkIndex of pendingChunks) {
    // 如果已经有 3 个在上传，等待其中一个完成
    if (uploadQueue.length >= MAX_CONCURRENT_UPLOADS) {
      await Promise.race(uploadQueue)
    }

    const uploadPromise = uploadChunkWithRetry(manager, chunkIndex, onProgress)
      .then(() => {
        // 从队列中移除已完成的
        const index = uploadQueue.indexOf(uploadPromise)
        if (index > -1) {
          uploadQueue.splice(index, 1)
        }
      })
      .catch(error => {
        // 从队列中移除失败的
        const index = uploadQueue.indexOf(uploadPromise)
        if (index > -1) {
          uploadQueue.splice(index, 1)
        }
        throw error
      })

    uploadQueue.push(uploadPromise)
  }

  // 等待所有上传完成
  await Promise.all(uploadQueue)
}

/**
 * 完成上传
 * @param uploadId 上传 ID
 * @param fileHash 文件 MD5
 * @param totalChunks 总分片数
 * @returns 文件信息
 */
export async function completeUpload(
  uploadId: string,
  fileHash: string,
  totalChunks: number
) {
  const response = await fileApi.completeUpload({
    upload_id: uploadId,
    file_hash: fileHash,
    total_chunks: totalChunks
  })

  if (response.data.code !== 200) {
    throw new Error('完成上传失败')
  }

  return response.data.data
}

/**
 * 取消上传
 * @param uploadId 上传 ID
 */
export async function cancelUpload(uploadId: string): Promise<void> {
  const manager = activeUploads.get(uploadId)

  if (manager) {
    // 取消正在进行的请求
    manager.abortController.abort()

    // 通知服务器
    try {
      await fileApi.cancelUpload({ upload_id: uploadId })
    } catch (error) {
      console.error('取消上传失败:', error)
    }

    // 清理
    activeUploads.delete(uploadId)
  }
}

/**
 * 完整上传流程
 * @param file 要上传的文件
 * @param folderId 目标文件夹 ID
 * @returns 文件信息
 */
export async function uploadFile(
  file: File,
  folderId?: number
) {
  const uploadStore = useUploadStore()

  // 添加上传任务到 store
  const uploadId = uploadStore.addTask(file, folderId)

  try {
    // 更新状态为上传中
    uploadStore.updateTask(uploadId, { status: 'uploading' })

    // 1. 计算文件 MD5
    const fileHash = await calculateMD5(file)

    // 2. 初始化上传
    const initResponse = await initUpload(file, folderId)

    // 检查是否秒传
    if (initResponse.is_quick_upload && initResponse.file_id) {
      // 秒传成功
      uploadStore.markCompleted(uploadId, initResponse.file_id)
      return { uploadId, fileId: initResponse.file_id, isQuickUpload: true }
    }

    // 3. 分片上传
    const { chunk_size, total_chunks, uploaded_chunks } = initResponse

    // 获取分片策略
    const strategy = getChunkStrategy(file.size)

    // 分片
    const chunks = splitFile(file, strategy.chunkSize)

    // 创建上传管理器
    const manager: UploadManager = {
      uploadId: initResponse.upload_id,
      file,
      folderId: folderId ?? null,
      fileHash,
      chunks,
      uploadedChunks: new Set(uploaded_chunks),
      retryCount: new Map(),
      abortController: new AbortController()
    }

    activeUploads.set(uploadId, manager)

    // 更新任务信息
    uploadStore.updateTask(uploadId, {
      totalChunks: total_chunks,
      uploadedChunks: uploaded_chunks
    })

    // 上传分片
    await uploadChunksConcurrently(manager, (uploadedCount) => {
      // 更新进度
      const progress = Math.round((uploadedCount / total_chunks) * 100)
      const uploadedSize = uploadedCount * chunk_size
      uploadStore.updateProgress(uploadId, progress, uploadedSize)
      uploadStore.updateTask(uploadId, {
        uploadedChunks: Array.from(manager.uploadedChunks)
      })
    })

    // 4. 完成上传
    const fileInfo = await completeUpload(
      initResponse.upload_id,
      fileHash,
      total_chunks
    )

    // 标记完成
    uploadStore.markCompleted(uploadId, fileInfo.id)

    // 清理
    activeUploads.delete(uploadId)

    return { uploadId, fileId: fileInfo.id, isQuickUpload: false }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : '上传失败'
    uploadStore.markFailed(uploadId, errorMessage)
    activeUploads.delete(uploadId)
    throw error
  }
}

/**
 * useFileUpload Composable
 * 提供文件上传相关的状态和方法
 */
export function useFileUpload() {
  const uploadStore = useUploadStore()
  const isUploading = ref(false)

  return {
    // 状态
    isUploading,
    tasks: uploadStore.tasks,
    activeTasks: uploadStore.activeTasks,
    completedTasks: uploadStore.completedTasks,
    failedTasks: uploadStore.failedTasks,
    totalProgress: uploadStore.totalProgress,
    isExpanded: uploadStore.isExpanded,

    // 方法
    calculateMD5,
    getChunkStrategy,
    splitFile,
    calculateChunkMD5,
    initUpload,
    uploadFile,
    cancelUpload,
    removeTask: uploadStore.removeTask,
    clearCompleted: uploadStore.clearCompleted,
    toggleExpanded: uploadStore.toggleExpanded
  }
}
