// src/composables/useAIStream.ts

import { ref } from 'vue'

// 待确认发送载荷：与后端 ai.PendingSend 对齐（SSE pending 帧）
export interface PendingSend {
  id: number
  target_conversation_id: number
  target_name: string
  preview: string
}

interface StreamOptions {
  url: string
  body: Record<string, any>
  onChunk: (content: string) => void
  onComplete: () => void
  onError: (error: Error) => void
  // 收到待确认发送帧时回调（send_message 确认制）：由调用方挂到消息上渲染确认条
  onPending?: (pending: PendingSend) => void
  // 收到工具调用进度帧时回调：由调用方按 tool_call_id upsert 工具轨迹（渲染复用 ToolCallTrace）
  onTool?: (ev: AIToolEvent) => void
}

// 工具调用进度事件：与后端 ai.ToolEvent 对齐（SSE tool_event 帧），
// 按 tool_call_id 跨帧 upsert 成工具轨迹（start=running → end=ok|error）
export interface AIToolEvent {
  step: number
  tool_call_id?: string
  tool_name: string
  label: string
  status: 'running' | 'ok' | 'error'
  args?: Record<string, unknown>
  error?: string
}

/** SSE 帧内 ai.StreamChunk 的客户端形态（仅消费本文件关心的字段） */
interface StreamChunk {
  error?: string
  pending?: PendingSend
  tool_event?: AIToolEvent
  content?: string
  finish?: 'stop'
}

function handleChunk(chunk: StreamChunk, options: StreamOptions): 'stop' | null {
  if (chunk.error) {
    options.onError(new Error(chunk.error))
    return 'stop'
  }
  if (chunk.pending) {
    options.onPending?.(chunk.pending)
  }
  if (chunk.tool_event) {
    options.onTool?.(chunk.tool_event)
  }
  if (chunk.content) {
    options.onChunk(chunk.content)
  }
  if (chunk.finish === 'stop') {
    options.onComplete()
    return 'stop'
  }
  return null
}

export function useAIStream() {
  const abortController = ref<AbortController | null>(null)

  async function stream(options: StreamOptions): Promise<void> {
    const token = localStorage.getItem('token')
    abortController.value = new AbortController()

    try {
      const response = await fetch(options.url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(options.body),
        signal: abortController.value.signal
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }

      const reader = response.body?.getReader()
      if (!reader) throw new Error('No reader available')

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.slice(6)
            if (data.trim() === '') continue

            try {
              if (handleChunk(JSON.parse(data), options) === 'stop') return
            } catch {
              // 忽略解析错误
            }
          }
        }
      }

      // 流结束后处理 buffer 中残留的最后一行
      if (buffer.startsWith('data: ')) {
        try {
          if (handleChunk(JSON.parse(buffer.slice(6)), options) === 'stop') return
        } catch {
          // 忽略解析错误
        }
      }

      options.onComplete()
    } catch (e: any) {
      if (e.name === 'AbortError') {
        return
      }
      options.onError(e)
    }
  }

  function abort() {
    abortController.value?.abort()
  }

  return { stream, abort }
}
