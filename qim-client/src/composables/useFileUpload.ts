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

/**
 * 动态计算最大并发数
 * 基于 CPU 核心数，最多 5 个并发
 */
function getMaxConcurrentUploads(): number {
  // 获取 CPU 核心数，默认为 4
  const cpuCores = navigator.hardwareConcurrency || 4
  // 根据核心数计算并发数，最多 5 个
  const maxConcurrent = Math.min(Math.max(Math.floor(cpuCores / 2), 2), 5)
  return maxConcurrent
}

// 最大并发数（动态计算）
const MAX_CONCURRENT_UPLOADS = getMaxConcurrentUploads()

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

  if (response.data.code !== 0) {
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

  if (response.data.code !== 0) {
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
 * 上传队列管理器
 * 使用队列管理并发上传，优化内存使用
 */
class UploadQueueManager {
  private queue: number[] = []
  private activeCount = 0
  private maxConcurrent: number
  private manager: UploadManager
  private onProgress?: (uploadedChunks: number) => void
  private resolve?: () => void
  private reject?: (error: Error) => void
  private hasError = false

  constructor(
    manager: UploadManager,
    maxConcurrent: number,
    onProgress?: (uploadedChunks: number) => void
  ) {
    this.manager = manager
    this.maxConcurrent = maxConcurrent
    this.onProgress = onProgress
  }

  /**
   * 添加分片到队列
   */
  addChunk(chunkIndex: number): void {
    this.queue.push(chunkIndex)
  }

  /**
   * 启动队列处理
   */
  async start(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.resolve = resolve
      this.reject = reject
      this.processQueue()
    })
  }

  /**
   * 处理队列
   */
  private processQueue(): void {
    // 如果有错误，停止处理
    if (this.hasError) {
      return
    }

    // 如果队列为空且没有活跃的上传，完成
    if (this.queue.length === 0 && this.activeCount === 0) {
      this.resolve?.()
      return
    }

    // 启动新的上传任务，直到达到最大并发数
    while (this.queue.length > 0 && this.activeCount < this.maxConcurrent) {
      const chunkIndex = this.queue.shift()!
      this.activeCount++
      
      this.uploadChunk(chunkIndex)
        .then(() => {
          this.activeCount--
          // 上传完成后，释放分片引用以优化内存
          this.manager.chunks[chunkIndex] = new Blob([])
          // 继续处理队列
          this.processQueue()
        })
        .catch((error) => {
          this.hasError = true
          this.reject?.(error)
        })
    }
  }

  /**
   * 上传单个分片
   */
  private async uploadChunk(chunkIndex: number): Promise<void> {
    await uploadChunkWithRetry(this.manager, chunkIndex, this.onProgress)
  }
}

/**
 * 并发上传分片
 * 使用队列管理器优化并发控制和内存使用
 */
async function uploadChunksConcurrently(
  manager: UploadManager,
  onProgress?: (uploadedChunks: number) => void
): Promise<void> {
  // 创建队列管理器
  const queueManager = new UploadQueueManager(
    manager,
    MAX_CONCURRENT_UPLOADS,
    onProgress
  )

  // 找出需要上传的分片并添加到队列
  for (let i = 0; i < manager.chunks.length; i++) {
    if (!manager.uploadedChunks.has(i)) {
      queueManager.addChunk(i)
    }
  }

  // 启动队列处理
  await queueManager.start()
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

  if (response.data.code !== 0) {
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
