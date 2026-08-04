/**
 * 频道富文本渲染（声明式 Markdown，方案 A）。
 *
 * 频道定位为订阅内容/公告流，正文一律按 Markdown 解析。
 * 渲染链路与单聊 MarkdownMessage 一致：
 *   marked 渲染 Markdown → DOMPurify 消毒防 XSS → emoji/经典表情转图片
 * 纯函数便于单测；在组件中调用后配合 useCodeHighlight 做代码高亮。
 */
import { marked } from 'marked'
import { sanitizeMarkdown } from './sanitize'
import { emojiToHtml, classicToHtml } from './emoji'

export function renderChannelMarkdown(content: string): string {
  if (!content) return ''
  const html = marked(content)
  const htmlString = typeof html === 'string' ? html : String(html)
  return classicToHtml(emojiToHtml(sanitizeMarkdown(htmlString)))
}
