// src/composables/useAIStream.ts

import { ref } from 'vue'

interface StreamOptions {
  url: string
  body: Record<string, any>
  onChunk: (content: string) => void
  onComplete: () => void
  onError: (error: Error) => void
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
              const chunk = JSON.parse(data)
              if (chunk.content) {
                options.onChunk(chunk.content)
              }
              if (chunk.finish === 'stop') {
                options.onComplete()
                return
              }
            } catch {
              // 忽略解析错误
            }
          }
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
