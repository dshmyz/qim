import { describe, expect, it } from 'vitest'
import { getMessagePreview } from '@/utils/messagePreview'

describe('getMessagePreview', () => {
  it('formats structured message content without exposing JSON', () => {
    expect(getMessagePreview({ type: 'file', content: JSON.stringify({ name: '方案.pdf', size: 1024 }) }))
      .toEqual({ kind: 'file', label: '方案.pdf · 1 KB' })
    expect(getMessagePreview({ type: 'share', content: JSON.stringify({ name: '设计说明' }) }))
      .toEqual({ kind: 'share', label: '分享：设计说明' })
    expect(getMessagePreview({ type: 'unknown', content: JSON.stringify({ raw: true }) }))
      .toEqual({ kind: 'unknown', label: '未知消息' })
  })

  it('formats file payloads and finite byte sizes', () => {
    expect(getMessagePreview({ type: 'file', content: JSON.stringify({ fileName: '预算.xlsx', size: 500 }) }))
      .toEqual({ kind: 'file', label: '预算.xlsx · 500 B' })
    expect(getMessagePreview({ type: 'text', content: JSON.stringify({ name: '演示.pptx', size: 1024 * 1024 }) }))
      .toEqual({ kind: 'file', label: '演示.pptx · 1 MB' })
    expect(getMessagePreview({ type: 'file', content: JSON.stringify({ name: '附件', size: Number.POSITIVE_INFINITY }) }))
      .toEqual({ kind: 'file', label: '附件' })
  })

  it('uses content-aware labels for supported rich messages', () => {
    expect(getMessagePreview({ type: 'image', content: 'image-url' })).toEqual({ kind: 'image', label: '图片' })
    expect(getMessagePreview({ type: 'mini-app', content: JSON.stringify({ name: '审批助手' }) }))
      .toEqual({ kind: 'miniApp', label: '小程序：审批助手' })
    expect(getMessagePreview({ type: 'news', content: JSON.stringify({ title: '今日资讯' }) }))
      .toEqual({ kind: 'news', label: '今日资讯' })
  })

  it('strips markdown formatting and safely falls back for plain or malformed content', () => {
    expect(getMessagePreview({ type: 'markdown', content: '# 标题\n- [文档](https://example.com)' }))
      .toEqual({ kind: 'text', label: '标题 文档' })
    expect(getMessagePreview({ type: 'markdown', content: '```ts\nconst answer = 42\n```' }))
      .toEqual({ kind: 'text', label: 'const answer = 42' })
    expect(getMessagePreview({ type: 'text', content: '普通消息' })).toEqual({ kind: 'text', label: '普通消息' })
    expect(getMessagePreview({ type: 'text', content: '' })).toEqual({ kind: 'text', label: '无内容' })
    expect(getMessagePreview({ type: 'text', content: '{not-json' })).toEqual({ kind: 'text', label: '{not-json' })
  })
})
