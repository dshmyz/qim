import { describe, expect, it } from 'vitest'
import { markdownToPlainText } from '@/utils/markdown'

describe('markdownToPlainText', () => {
  it('strips fenced code block but keeps code content', () => {
    const md = '```javascript\nconsole.log(1)\n```'
    expect(markdownToPlainText(md)).toBe('console.log(1)')
  })

  it('strips inline code backticks', () => {
    expect(markdownToPlainText('use `console.log` to debug'))
      .toBe('use console.log to debug')
  })

  it('strips heading markers', () => {
    expect(markdownToPlainText('# Title')).toBe('Title')
    expect(markdownToPlainText('### Subtitle')).toBe('Subtitle')
  })

  it('strips bold and italic markers', () => {
    expect(markdownToPlainText('**bold** text')).toBe('bold text')
    expect(markdownToPlainText('*italic* text')).toBe('italic text')
  })

  it('converts links to link text only', () => {
    expect(markdownToPlainText('[Google](https://google.com)'))
      .toBe('Google')
  })

  it('converts images to [图片] placeholder', () => {
    expect(markdownToPlainText('![logo](https://a.com/logo.png)'))
      .toBe('[图片]')
  })

  it('strips list markers', () => {
    expect(markdownToPlainText('- item1\n- item2')).toBe('item1\nitem2')
    expect(markdownToPlainText('1. first\n2. second')).toBe('first\nsecond')
  })

  it('strips blockquote markers', () => {
    expect(markdownToPlainText('> quoted text')).toBe('quoted text')
  })

  it('strips horizontal rules', () => {
    expect(markdownToPlainText('text\n---\nmore')).toBe('text\n\nmore')
  })

  it('decodes mention tokens', () => {
    const md = '@{mention:3|%E5%BC%A0%E4%B8%89} 说了一段话'
    expect(markdownToPlainText(md)).toBe('@张三 说了一段话')
  })

  it('handles complex markdown with code block and text', () => {
    const md = '## 标题\n\n这是一段说明：\n\n```python\nprint(1)\n```\n\n**重点**：[链接](https://a.com)'
    const result = markdownToPlainText(md)
    expect(result).toContain('标题')
    expect(result).toContain('这是一段说明')
    expect(result).toContain('print(1)')
    expect(result).toContain('重点')
    expect(result).toContain('链接')
    expect(result).not.toContain('```')
    expect(result).not.toContain('##')
    expect(result).not.toContain('**')
    expect(result).not.toContain('https://a.com')
  })

  it('returns empty string for empty input', () => {
    expect(markdownToPlainText('')).toBe('')
  })

  it('preserves plain text without markdown', () => {
    expect(markdownToPlainText('just plain text')).toBe('just plain text')
  })
})
