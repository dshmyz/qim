import SparkMD5 from 'spark-md5'

/**
 * MD5 计算 Web Worker
 * 在后台线程计算文件的 MD5 哈希值，避免阻塞主线程
 */

// 默认分片大小：2MB
const DEFAULT_CHUNK_SIZE = 2 * 1024 * 1024

// Worker 接收的消息类型
interface WorkerMessage {
  file: File
  chunkSize?: number
}

// Worker 发送的消息类型
interface ProgressMessage {
  progress: number
}

interface CompleteMessage {
  hash: string
  progress: 100
}

interface ErrorMessage {
  error: string
}

type WorkerResponse = ProgressMessage | CompleteMessage | ErrorMessage

/**
 * 计算文件的 MD5 哈希值
 */
async function calculateMD5(file: File, chunkSize: number = DEFAULT_CHUNK_SIZE): Promise<string> {
  return new Promise((resolve, reject) => {
    const spark = new SparkMD5.ArrayBuffer()
    const reader = new FileReader()
    const chunks = Math.ceil(file.size / chunkSize)
    let currentChunk = 0

    reader.onload = (e) => {
      if (e.target?.result instanceof ArrayBuffer) {
        spark.append(e.target.result)
        currentChunk++

        // 计算并发送进度
        const progress = Math.round((currentChunk / chunks) * 100)
        self.postMessage({ progress } as ProgressMessage)

        // 继续读取下一块
        loadNextChunk()
      }
    }

    reader.onerror = () => {
      reject(new Error('文件读取失败'))
    }

    const loadNextChunk = () => {
      if (currentChunk < chunks) {
        const start = currentChunk * chunkSize
        const end = Math.min(start + chunkSize, file.size)
        const blob = file.slice(start, end)
        reader.readAsArrayBuffer(blob)
      } else {
        // 所有块读取完成，计算最终哈希值
        const hash = spark.end()
        self.postMessage({ hash, progress: 100 } as CompleteMessage)
        resolve(hash)
      }
    }

    // 开始读取第一块
    loadNextChunk()
  })
}

// 监听主线程消息
self.onmessage = async (e: MessageEvent<WorkerMessage>) => {
  try {
    const { file, chunkSize } = e.data

    if (!file) {
      throw new Error('未提供文件')
    }

    if (!(file instanceof File)) {
      throw new Error('参数必须是 File 对象')
    }

    await calculateMD5(file, chunkSize)
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : '未知错误'
    self.postMessage({ error: errorMessage } as ErrorMessage)
  }
}

// 导出类型定义，供主线程使用
export type { WorkerMessage, WorkerResponse, ProgressMessage, CompleteMessage, ErrorMessage }
