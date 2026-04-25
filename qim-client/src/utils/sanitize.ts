import DOMPurify from 'dompurify'

/**
 * HTML 消毒工具
 * 使用 DOMPurify 对 HTML 内容进行安全过滤，防止 XSS 攻击
 */

/**
 * 对 HTML 内容进行消毒
 * @param html - 需要消毒的 HTML 字符串
 * @returns 消毒后的安全 HTML 字符串
 */
export function sanitizeHTML(html: string): string {
  if (!html) return ''
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ['b', 'i', 'em', 'strong', 'a', 'p', 'br', 'ul', 'ol', 'li', 'code', 'pre'],
    ALLOWED_ATTR: ['href', 'target', 'rel'],
    ADD_ATTR: ['target'],
    ALLOW_DATA_ATTR: false,
  })
}

/**
 * Markdown 渲染专用消毒配置
 * 允许更多 Markdown 常用的标签，但依然阻止脚本执行
 */
const MARKDOWN_CONFIG = {
  ALLOWED_TAGS: [
    'b', 'i', 'em', 'strong', 'a', 'p', 'br', 'ul', 'ol', 'li', 'code', 'pre',
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
    'blockquote', 'hr', 'img', 'table', 'thead', 'tbody', 'tr', 'th', 'td',
    'span', 'div', 'del', 'ins', 'sub', 'sup',
  ],
  ALLOWED_ATTR: ['href', 'target', 'rel', 'src', 'alt', 'title', 'class'],
  ADD_ATTR: ['target'],
  ALLOW_DATA_ATTR: false,
}

/**
 * 对 Markdown 渲染后的 HTML 进行消毒
 * @param html - Markdown 渲染后的 HTML 字符串
 * @returns 消毒后的安全 HTML 字符串
 */
export function sanitizeMarkdown(html: string): string {
  if (!html) return ''
  return DOMPurify.sanitize(html, MARKDOWN_CONFIG)
}

/**
 * HTML 实体编码函数，用于将用户输入安全地嵌入到 HTML 中
 * 在构建 HTML 模板字符串时，对用户可控的数据进行编码
 * @param str - 需要编码的字符串
 * @returns 编码后的安全字符串
 */
export function escapeHTML(str: string): string {
  if (!str) return ''
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;')
}
