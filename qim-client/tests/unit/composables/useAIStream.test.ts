import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useAIStream } from '@/composables/useAIStream'

// 构造 SSE 响应：lines 为逐条已序列化的 data 帧，读取器按顺序吐出再结束
const sseResponse = (frames: object[]) => {
  const lines = frames.map(f => `data: ${JSON.stringify(f)}\n\n`)
  const chunks = [...lines]
  return {
    ok: true,
    body: {
      getReader: () => ({
        read: () =>
          chunks.length
            ? Promise.resolve({ done: false, value: new TextEncoder().encode(chunks.shift()) })
            : Promise.resolve({ done: true, value: undefined }),
      }),
    },
  }
}

describe('useAIStream', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.getItem = vi.fn(() => 'test-token')
  })

  it('content 帧按序透传给 onChunk，finish 帧触发 onComplete', async () => {
    const contents: string[] = []
    let completed = false
    global.fetch = vi.fn().mockResolvedValueOnce(sseResponse([
      { content: '你' },
      { content: '好' },
      { finish: 'stop' },
    ]))

    const { stream } = useAIStream()
    await stream({
      url: 'http://localhost/api',
      body: {},
      onChunk: c => contents.push(c),
      onComplete: () => { completed = true },
      onError: () => {},
    })

    expect(contents.join('')).toBe('你好')
    expect(completed).toBe(true)
  })

  it('move_to_reasoning 帧回调 onReasoning 携带草稿原文（思考草稿回撤契约）', async () => {
    const contents: string[] = []
    const drafts: string[] = []
    global.fetch = vi.fn().mockResolvedValueOnce(sseResponse([
      { content: '我先查一下待办…' },
      { tool_event: { step: 1, tool_name: 'list_tasks', label: '查询待办', status: 'running' } },
      { move_to_reasoning: '我先查一下待办…' },
      { content: '答案' },
      { finish: 'stop' },
    ]))

    const { stream } = useAIStream()
    await stream({
      url: 'http://localhost/api',
      body: {},
      onChunk: c => contents.push(c),
      onReasoning: d => drafts.push(d),
      onComplete: () => {},
      onError: () => {},
    })

    // 草稿先按正文流出（打字机），随后回撤——消费方负责从答案文本切除
    expect(contents.join('')).toBe('我先查一下待办…答案')
    expect(drafts).toEqual(['我先查一下待办…'])
  })

  it('pending / tool_event 帧分别回调 onPending / onTool', async () => {
    const pendings: unknown[] = []
    const tools: unknown[] = []
    global.fetch = vi.fn().mockResolvedValueOnce(sseResponse([
      { tool_event: { step: 1, tool_call_id: 'c1', tool_name: 'search_knowledge', label: '知识检索', status: 'ok' } },
      { pending: { id: 9, target_conversation_id: 3, target_name: '项目群', preview: 'hi' } },
      { finish: 'stop' },
    ]))

    const { stream } = useAIStream()
    await stream({
      url: 'http://localhost/api',
      body: {},
      onChunk: () => {},
      onComplete: () => {},
      onError: () => {},
      onPending: p => pendings.push(p),
      onTool: ev => tools.push(ev),
    })

    expect(pendings).toHaveLength(1)
    expect(tools).toHaveLength(1)
  })

  it('error 帧触发 onError 并中止流处理', async () => {
    const onError = vi.fn()
    const onChunk = vi.fn()
    global.fetch = vi.fn().mockResolvedValueOnce(sseResponse([
      { error: 'AI 服务不可用' },
      { content: '不应再收到' },
    ]))

    const { stream } = useAIStream()
    await stream({
      url: 'http://localhost/api',
      body: {},
      onChunk,
      onComplete: () => {},
      onError,
    })

    expect(onError).toHaveBeenCalledWith(expect.objectContaining({ message: 'AI 服务不可用' }))
    expect(onChunk).not.toHaveBeenCalled()
  })
})
