import { ref } from 'vue'
import { fileApi, type InitUploadResponse } from '../api/file'
import { useUploadStore } from '../stores/upload'
import SparkMD5 from 'spark-md5'

/**
 * 文件上传 Composable
 * 整合分片上传、进度管理等功能
 *
 * 秒传功能已移除（存在越权风险且前端算 MD5 性能开销大、命中率低）。
 * 不再计算文件级 MD5，init 时不再传 file_hash。
 * 分片级 MD5 仍保留用于分片完整性校验。
 */

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
 * 上传任务管理器
 */
interface UploadManager {
  uploadId: string
  file: File
  folderId: number | null
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
  // 调用初始化 API（秒传已移除，不再传 file_hash）
  const response = await fileApi.initUpload({
    filename: file.name,
    file_size: file.size,
    folder_id: folderId ?? null
  })

  if (response.data.code !== 0) {
    throw new Error('初始化上传失败')
  }

  return response.data.data
}

/**
 * 上传单个分片
 * signal 用于支持取消：调用 abortController.abort() 后正在进行的 HTTP 请求会被中止
 */
async function uploadSingleChunk(
  uploadId: string,
  chunk: Blob,
  chunkIndex: number,
  chunkHash: string,
  signal?: AbortSignal
): Promise<void> {
  const formData = new FormData()
  formData.append('upload_id', uploadId)
  formData.append('chunk_index', String(chunkIndex))
  formData.append('chunk_hash', chunkHash)
  formData.append('chunk', chunk)

  const response = await fileApi.uploadChunk(formData, signal)

  if (response.data.code !== 0) {
    throw new Error(`分片 ${chunkIndex} 上传失败`)
  }
}

/**
 * 上传分片（带重试）
 * 支持 abort 取消：signal.aborted 后立即 reject 不再重试
 */
async function uploadChunkWithRetry(
  manager: UploadManager,
  chunkIndex: number,
  onProgress?: (uploadedChunks: number) => void
): Promise<void> {
  const chunk = manager.chunks[chunkIndex]
  const chunkHash = await calculateChunkMD5(chunk)
  const signal = manager.abortController.signal

  // 进入前先检查是否已取消
  if (signal.aborted) {
    throw new Error('上传已取消')
  }

  let retryCount = manager.retryCount.get(chunkIndex) || 0

  while (retryCount < MAX_RETRY_COUNT) {
    // 每次重试前检查取消状态
    if (signal.aborted) {
      throw new Error('上传已取消')
    }

    try {
      await uploadSingleChunk(
        manager.uploadId,
        chunk,
        chunkIndex,
        chunkHash,
        signal
      )

      // 上传成功
      manager.uploadedChunks.add(chunkIndex)
      manager.retryCount.delete(chunkIndex)

      if (onProgress) {
        onProgress(manager.uploadedChunks.size)
      }

      return
    } catch (error) {
      // 如果是取消导致的错误，直接抛出不再重试
      if (signal.aborted || (error instanceof Error && error.name === 'CanceledError')) {
        throw new Error('上传已取消')
      }

      retryCount++
      manager.retryCount.set(chunkIndex, retryCount)

      if (retryCount >= MAX_RETRY_COUNT) {
        throw new Error(`分片 ${chunkIndex} 上传失败，已重试 ${MAX_RETRY_COUNT} 次`)
      }

      // 等待一段时间后重试，等待期间也要支持取消
      await new Promise<void>((resolve, reject) => {
        const timer = setTimeout(resolve, 1000 * retryCount)
        // abort 时清除定时器并 reject
        if (signal.aborted) {
          clearTimeout(timer)
          reject(new Error('上传已取消'))
          return
        }
        signal.addEventListener('abort', () => {
          clearTimeout(timer)
          reject(new Error('上传已取消'))
        }, { once: true })
      })
    }
  }
}

/**
 * 上传队列管理器
 * 使用队列管理并发上传，优化内存使用
 * 支持 abort 取消：检测到 signal.aborted 后立即停止队列并 resolve
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

    // 取消检查：如果已 abort，立即停止队列
    if (this.manager.abortController.signal.aborted) {
      this.resolve?.()
      return
    }

    // 如果队列为空且没有活跃的上传，完成
    if (this.queue.length === 0 && this.activeCount === 0) {
      this.resolve?.()
      return
    }

    // 启动新的上传任务，直到达到最大并发数
    while (this.queue.length > 0 && this.activeCount < this.maxConcurrent) {
      // 再次检查取消状态，避免 abort 后继续 shift 新分片
      if (this.manager.abortController.signal.aborted) {
        this.resolve?.()
        return
      }
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
          this.activeCount--
          // 取消导致的错误不算失败，直接 resolve 停止队列
          if (this.manager.abortController.signal.aborted) {
            this.resolve?.()
            return
          }
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
 * @returns 文件信息
 */
export async function completeUpload(
  uploadId: string
) {
  const response = await fileApi.completeUpload({
    upload_id: uploadId
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

    // 1. 初始化上传（秒传已移除，不再计算文件级 MD5）
    const initResponse = await initUpload(file, folderId)

    // 2. 分片上传
    const { chunk_size, total_chunks, uploaded_chunks } = initResponse

    // 使用后端返回的分片大小进行分片，确保前后端分片策略一致
    const chunks = splitFile(file, chunk_size)

    // 创建上传管理器
    const manager: UploadManager = {
      uploadId: initResponse.upload_id,
      file,
      folderId: folderId ?? null,
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

    // 3. 完成上传
    const fileInfo = await completeUpload(initResponse.upload_id)

    // 标记完成
    uploadStore.markCompleted(uploadId, fileInfo.id)

    // 清理
    activeUploads.delete(uploadId)

    return { uploadId, fileId: fileInfo.id }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : '上传失败'
    uploadStore.markFailed(uploadId, errorMessage)
    // 释放内存：清空 manager 持有的 chunks Blob 数组和 file 引用
    const manager = activeUploads.get(uploadId)
    if (manager) {
      manager.chunks = []
      // file 引用清空需谨慎，File 对象无法直接置 null（类型为 File）
      // 但 chunks 已清空，主要内存占用已释放
    }
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

/**
 * 文件级并发上传限制器
 * 限制同时上传的文件数量，避免浏览器并发连接数限制（HTTP/1.1 通常 6 个/域名）
 * 和过多并发导致的内存压力。
 */
const DEFAULT_MAX_CONCURRENT_FILES = 3

/**
 * 批量上传文件，限制并发数
 * @param files 要上传的文件列表
 * @param folderId 目标文件夹 ID
 * @param options 可选配置：maxConcurrent 最大并发数；onFileUploaded 单个文件上传成功后的回调（回调内抛错会记为失败）
 * @returns 每个文件的上传结果（成功返回 fileId，失败返回 null）
 */
export async function uploadFilesWithLimit(
  files: File[] | FileList,
  folderId?: number,
  options?: {
    maxConcurrent?: number
    onFileUploaded?: (file: File, fileId: number) => Promise<void>
  }
): Promise<Array<{ file: File; success: boolean; fileId?: number }>> {
  const list = Array.from(files)
  if (list.length === 0) return []

  const maxConcurrent = options?.maxConcurrent ?? DEFAULT_MAX_CONCURRENT_FILES
  const onFileUploaded = options?.onFileUploaded
  const results: Array<{ file: File; success: boolean; fileId?: number }> = []
  let currentIndex = 0

  // 工作函数：从队列中取下一个文件上传
  async function worker() {
    while (currentIndex < list.length) {
      const index = currentIndex++
      const file = list[index]
      try {
        const result = await uploadFile(file, folderId)
        // 上传成功后执行回调（如挂载到群文件），回调失败也算整体失败
        if (onFileUploaded && result.fileId) {
          await onFileUploaded(file, result.fileId)
        }
        results.push({ file, success: true, fileId: result.fileId })
      } catch (error) {
        console.error(`上传文件 ${file.name} 失败:`, error)
        results.push({ file, success: false })
      }
    }
  }

  // 启动 maxConcurrent 个 worker
  const workers: Promise<void>[] = []
  for (let i = 0; i < Math.min(maxConcurrent, list.length); i++) {
    workers.push(worker())
  }

  await Promise.all(workers)
  return results
}
