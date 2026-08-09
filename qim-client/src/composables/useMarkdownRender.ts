import { computed, ref, isRef, type ComputedRef, type Ref } from 'vue'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../utils/sanitize'
import { decodeToPlainText } from '../utils/mentions'
import { emojiToHtml, classicToHtml } from '../utils/emoji'
import { useCodeHighlight } from './useCodeHighlight'

/**
 * AI 渲染能力统一 —— 单一 markdown 渲染管道。
 *
 * 曾经这份「markdown → 安全 HTML」链散落在 MarkdownRenderer / MarkdownMessage /
 * StreamingMessage 三处，各自重复 marked→sanitize→emoji→classic、handleLinkClick、
 * useCodeHighlight。这里收敛为一处，三处渲染器（及新增的 AIAnswerBubble）全部委托，
 * 保证所有 AI 回答（IM 气泡 / BotChat / 通用 markdown）渲染结果与排版一致。
 */
export interface MarkdownRenderOpts {
  /** 流式中：normalizePartialMarkdown 自动闭合未闭合的代码块/行内反引号，避免解析跳变 */
  streaming?: boolean
  /** 解码 mention token：@{mention:3|张三} → @张三（IM 气泡需要） */
  decodeMention?: boolean
  /** 渲染后把表情字符/经典标记替换为 <img>（IM 气泡需要，BotChat/NoteEditor 不需要） */
  withEmoji?: boolean
}

/**
 * 流式过程中 markdown 不完整（代码块/行内代码未闭合），marked 解析结果
 * 与最终全文不一致。预处理：自动闭合未关闭的代码块和行内反引号。
 * 段落分隔不在此处理——GFM 模式下 marked 自身会正确处理换行。
 * @see 原 StreamingMessage.vue 的 normalizePartialMarkdown
 */
export function normalizePartialMarkdown(md: string): string {
  // 闭合未关闭的 ``` 代码块
  const fenceCount = (md.match(/^```/gm) || []).length
  if (fenceCount % 2 !== 0) {
    md += '\n```'
  }

  // 闭合未关闭的行内 ` 反引号
  const backtickSegments = md.split('`')
  if (backtickSegments.length % 2 === 0) {
    md += '`'
  }

  return md
}

/**
 * 剔除块级元素之间的纯空白文本节点（marked 序列化产物里的 "\n"）。
 *
 * marked() 把 markdown 源里的空行/换行折叠成 "<p>…</p>\n<p>…</p>" 的形式——即
 * 块级标签之间残留一个 "\n" 文本节点。若容器设了 white-space: pre-wrap（AI 气泡为
 * 保留流式中尚未成块的纯文本换行需要它），这个 "\n" 会被渲染成一个整行高的空行，
 * 导致段落间留白明显偏大（实测 4px margin 之上再叠一整行 22px）。
 *
 * 旧版终态 AI 消息走 MarkdownMessage（无 pre-wrap，换行按 normal 折叠）是紧凑的；
 * 统一到 AIAnswerBubble 后 pre-wrap 使其膨胀。这里在渲染管道统一剔除「位于块级标签
 * 之间」的纯空白节点：解析后的 markdown 块间距交给 CSS margin，流式纯文本（单段落
 * 内含 \n、尚无块结构）不受影响。对全部分发方（AIAnswerBubble / MarkdownMessage /
 * NoteEditor）一致生效，非流式下与 white-space: normal 的折叠结果一致。
 */
/** 块级标签，用于界定哪些 `\n` 属于"标签之间"的排版残留 */
const BETWEEN_BLOCK_RE = /(<\/?(?:p|div|h[1-6]|pre|blockquote|ul|ol|li|table|tr|section)[^>]*>)\s*(?=<\/?(?:p|div|h[1-6]|pre|blockquote|ul|ol|li|table|tr|section)[^>]*>|$)/gi

function stripBetweenBlockWhitespace(html: string): string {
  // 剔除"块级结束标签 + 纯空白"后紧跟另一个块级标签（或到结尾）的空白。
  // 只删块级标签之间的换行/空白，段落内部、<li> 内部、单段裸文本里的 \n 原样保留。
  return html.replace(BETWEEN_BLOCK_RE, '$1')
}

/** 纯函数：markdown → 消毒后的安全 HTML 字符串。参数化全局的渲染选项。 */
export function renderMarkdown(md: string, opts: MarkdownRenderOpts = {}): string {
  if (!md) return ''
  let text = opts.decodeMention ? decodeToPlainText(md) : md
  text = opts.streaming ? normalizePartialMarkdown(text) : text
  const result = marked(text)
  let html = typeof result === 'string' ? result : String(result)
  html = stripBetweenBlockWhitespace(html)
  // 使用 DOMPurify 进行消毒，防止 XSS 攻击；需要时再把表情字符/经典标记替换为 <img>
  const sanitized = sanitizeMarkdown(html)
  return opts.withEmoji ? classicToHtml(emojiToHtml(sanitized)) : sanitized
}

/** 外链点击：Electron 环境用外部浏览器打开，其余环境放行默认行为。 */
export function handleLinkClick(event: MouseEvent): void {
  const target = event.target as HTMLElement
  const link = target.closest('a')
  if (link && window.electron?.shell?.openExternal) {
    event.preventDefault()
    const href = link.getAttribute('href')
    if (href) {
      window.electron.shell.openExternal(href)
    }
  }
}

export interface MarkdownRenderResult {
  /** 渲染后的安全 HTML（响应式，随 content/opts 变化） */
  html: ComputedRef<string>
  /** 容器元素引用，已接好代码高亮与外链点击，直接绑定到 v-html 根节点 */
  containerRef: Ref<HTMLElement | null>
}

/**
 * 渲染容器组合式：给定内容与选项（可为 ref），返回安全 HTML 与容器 ref。
 * 容器 ref 已自动接上 useCodeHighlight（代码高亮）与 handleLinkClick（外链点击），
 * 调用方只需 <div :ref="containerRef" v-html="html" @click="handleLinkClick" />。
 */
export function useMarkdownRender(
  content: Ref<string>,
  opts: ComputedRef<MarkdownRenderOpts> | MarkdownRenderOpts
): MarkdownRenderResult {
  const optsRef = isRef(opts) ? opts : ref(opts)
  const html = computed(() => renderMarkdown(content.value, optsRef.value))
  const containerRef = ref<HTMLElement | null>(null)
  useCodeHighlight(containerRef, html)
  return { html, containerRef }
}
