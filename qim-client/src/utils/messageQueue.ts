import { logger } from './logger'

interface QueuedMessage {
  id: string
  data: any
  timestamp: number
  retryCount: number
}

const STORAGE_KEY = 'qim_message_queue'
const MAX_QUEUE_SIZE = 100

class MessageQueue {
  private queue: QueuedMessage[] = []
  
  constructor() {
    this.loadFromStorage()
  }
  
  private generateId(): string {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
  }
  
  private loadFromStorage(): void {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) {
        this.queue = JSON.parse(stored)
        logger.log(`[MessageQueue] 从存储加载 ${this.queue.length} 条消息`)
      }
    } catch (error) {
      logger.error('[MessageQueue] 加载存储失败:', error)
      this.queue = []
    }
  }
  
  private saveToStorage(): void {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(this.queue))
    } catch (error) {
      logger.error('[MessageQueue] 保存存储失败:', error)
    }
  }
  
  enqueue(data: any): string {
    if (this.queue.length >= MAX_QUEUE_SIZE) {
      logger.warn('[MessageQueue] 队列已满，移除最旧消息')
      this.queue.shift()
    }
    
    const message: QueuedMessage = {
      id: this.generateId(),
      data,
      timestamp: Date.now(),
      retryCount: 0
    }
    
    this.queue.push(message)
    this.saveToStorage()
    
    logger.log(`[MessageQueue] 消息入队: ${message.id}`)
    return message.id
  }
  
  dequeue(): QueuedMessage | null {
    if (this.queue.length === 0) {
      return null
    }
    
    const message = this.queue.shift()!
    this.saveToStorage()
    
    logger.log(`[MessageQueue] 消息出队: ${message.id}`)
    return message
  }
  
  peek(): QueuedMessage | null {
    return this.queue.length > 0 ? this.queue[0] : null
  }
  
  flush(sendFn: (data: any) => boolean): void {
    logger.log(`[MessageQueue] 开始刷新队列，共 ${this.queue.length} 条消息`)
    
    const failed: QueuedMessage[] = []
    
    while (this.queue.length > 0) {
      const message = this.queue.shift()!
      
      try {
        const success = sendFn(message.data)
        if (!success) {
          message.retryCount++
          if (message.retryCount < 3) {
            failed.push(message)
          } else {
            logger.error(`[MessageQueue] 消息发送失败，已丢弃: ${message.id}`)
          }
        }
      } catch (error) {
        logger.error(`[MessageQueue] 消息发送异常: ${message.id}`, error)
        message.retryCount++
        if (message.retryCount < 3) {
          failed.push(message)
        }
      }
    }
    
    this.queue = failed
    this.saveToStorage()
    
    logger.log(`[MessageQueue] 队列刷新完成，失败 ${failed.length} 条`)
  }
  
  clear(): void {
    this.queue = []
    this.saveToStorage()
    logger.log('[MessageQueue] 队列已清空')
  }
  
  size(): number {
    return this.queue.length
  }
  
  isEmpty(): boolean {
    return this.queue.length === 0
  }
  
  getAll(): QueuedMessage[] {
    return [...this.queue]
  }
}

export const messageQueue = new MessageQueue()
