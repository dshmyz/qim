import { describe, it, expect } from 'vitest'
import { resolveMessageDisplay } from '../../src/utils/messageDisplay'

describe('resolveMessageDisplay fixes', () => {
  it('decodes mention token in markdown preview', () => {
    const r = resolveMessageDisplay({ type: 'markdown', content: '@{mention:3|%E6%B5%8B%E8%AF%95%E7%94%A8%E6%88%B7} 基于通用知识，本周五例会' })
    expect(r.kind).toBe('text')
    expect(r.summary).toContain('@测试用户')
    expect(r.summary).not.toContain('@{mention')
  })

  it('shows plain text instead of 未知消息 for explicit text-type JSON', () => {
    const r = resolveMessageDisplay({ type: 'text', content: '{"foo":"bar"}' })
    expect(r.kind).toBe('text')
    expect(r.summary).toContain('foo')
  })

  it('shows raw JSON content even without a type (never 未知消息)', () => {
    // 用户发的内容就是普通文本，即使恰好是 JSON 也应原样展示，不能被吞成「未知消息」
    const r = resolveMessageDisplay({ content: '{"foo":"bar"}' })
    expect(r.kind).toBe('text')
    expect(r.summary).toContain('foo')
  })

  it('file type still resolves as file, unaffected by the text-JSON fix', () => {
    const r = resolveMessageDisplay({ type: 'file', content: '{"name":"a.pdf","size":100}' })
    expect(r.kind).toBe('file')
    expect(r.summary).toContain('[文件]')
  })

  it('card type still resolves as card, unaffected by the text-JSON fix', () => {
    const r = resolveMessageDisplay({ type: 'card', content: '{"title":"确认","text":"提交"}' })
    expect(r.kind).toBe('card')
    expect(r.summary).toContain('确认')
  })
})
