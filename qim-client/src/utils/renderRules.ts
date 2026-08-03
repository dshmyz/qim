import type { CompiledRule } from '../stores/renderRules'
import { escapeHTML } from './sanitize'

interface MatchSegment {
  type: 'text' | 'rule_match'
  text?: string
  rule?: CompiledRule
  matched?: string
  context?: Record<string, string>
}

/**
 * HTML 切分片段
 * - html: HTML 标签（如 <a href="...">、</a>、<br>），原样保留
 * - text: 文本节点，可应用规则
 * - protected_text: 受保护文本（<a> 标签内部），跳过规则
 */
interface HtmlSegment {
  type: 'html' | 'text' | 'protected_text'
  text: string
}

/**
 * 把已转义的 HTML 字符串按标签边界切分
 * 进入 <a> 标签后，所有内容（包括文本）标记为 protected_text，直到 </a>
 */
function splitByHtmlTags(html: string): HtmlSegment[] {
  const segments: HtmlSegment[] = []
  const tagRegex = /<\/?[a-zA-Z][^>]*>/g
  let lastIndex = 0
  let match: RegExpExecArray | null
  let inAnchor = false

  while ((match = tagRegex.exec(html)) !== null) {
    // 标签前的文本
    if (match.index > lastIndex) {
      const text = html.slice(lastIndex, match.index)
      segments.push({
        type: inAnchor ? 'protected_text' : 'text',
        text,
      })
    }
    // 标签本身
    const tag = match[0]
    if (/^<a[\s>]/i.test(tag)) {
      inAnchor = true
    } else if (/^<\/a\s*>/i.test(tag)) {
      inAnchor = false
    }
    segments.push({ type: 'html', text: tag })
    lastIndex = match.index + tag.length
  }
  // 尾部文本
  if (lastIndex < html.length) {
    segments.push({
      type: inAnchor ? 'protected_text' : 'text',
      text: html.slice(lastIndex),
    })
  }
  return segments
}

/**
 * 模板填充：用捕获组上下文替换 {{name}} 占位符
 */
function fillTemplate(tmpl: string, ctx: Record<string, string>): string {
  let result = tmpl
  for (const [k, v] of Object.entries(ctx)) {
    result = result.split(`{{${k}}}`).join(v)
  }
  return result
}

/**
 * 用单条规则切分文本段
 * 已匹配部分标记为 rule_match，未匹配部分保持 text
 */
function splitByRule(text: string, rule: CompiledRule): MatchSegment[] {
  const result: MatchSegment[] = []
  let lastIndex = 0
  let match: RegExpExecArray | null

  rule.compiledRegex.lastIndex = 0
  while ((match = rule.compiledRegex.exec(text)) !== null) {
    // 匹配前的前缀文本
    if (match.index > lastIndex) {
      result.push({ type: 'text', text: text.slice(lastIndex, match.index) })
    }
    // 提取捕获组上下文
    const ctx: Record<string, string> = {}
    for (const [name, idx] of Object.entries(rule.match.capture_groups)) {
      ctx[name] = match[idx] || ''
    }
    result.push({
      type: 'rule_match',
      rule,
      matched: match[0],
      context: ctx
    })
    lastIndex = match.index + match[0].length
    // 防止零宽匹配死循环
    if (match[0] === '') rule.compiledRegex.lastIndex++
  }
  // 尾部文本
  if (lastIndex < text.length) {
    result.push({ type: 'text', text: text.slice(lastIndex) })
  }
  return result
}

/**
 * 渲染单个匹配段为 HTML
 */
function renderMatchedSegment(
  rule: CompiledRule,
  ctx: Record<string, string>,
  matched: string
): string {
  const url = fillTemplate(rule.render.url_template, ctx)
  const label = fillTemplate(rule.render.label_template, ctx) || matched
  const title = rule.render.title_template
    ? fillTemplate(rule.render.title_template, ctx)
    : ''
  const target = rule.render.target || '_blank'
  const cls = rule.render.class || ''
  const icon = rule.render.icon
    ? `<i class="${escapeHTML(rule.render.icon)}"></i> `
    : ''

  switch (rule.render.type) {
    case 'link':
      return `<a href="${escapeHTML(url)}" target="${target}" ` +
             `rel="noopener noreferrer" class="message-link ${cls}" ` +
             `title="${escapeHTML(title)}">${icon}${escapeHTML(label)}</a>`

    case 'link_card':
      return `<a href="${escapeHTML(url)}" target="${target}" ` +
             `rel="noopener noreferrer" class="render-card ${cls}" ` +
             `title="${escapeHTML(title)}">` +
             `<span class="render-card__icon">${icon}</span>` +
             `<span class="render-card__label">${escapeHTML(label)}</span>` +
             `</a>`

    case 'text_chip':
      return `<span class="render-chip ${cls}" title="${escapeHTML(title)}">` +
             `${icon}${escapeHTML(label)}</span>`

    default:
      return escapeHTML(matched)
  }
}

/**
 * 应用渲染规则到已转义的文本，返回 HTML 字符串
 * 流程：HTML 切分 → 对文本段应用规则 → 拼接
 *
 * HTML-aware：
 * - <a>...</a> 标签内部（包括 href 属性和 label 文本）跳过规则，避免破坏已有链接
 * - 其他标签（<br>、<span> 等）原样保留，仅对标签之间的文本节点应用规则
 *
 * 注意：本函数不对输出做 sanitize，调用方需对最终结果进行 sanitizeHTML。
 * 安全性保证：
 * - 输入必须是已经过 escapeHTML 的文本
 * - renderMatchedSegment 中所有用户可控数据（url/label/title/icon/class）均用 escapeHTML 转义
 * - 生成的标签仅 <a>/<span>/<i>，均在外层 sanitizeHTML 的 ALLOWED_TAGS 内
 *
 * @param escapedText 已经过 escapeHTML 的文本（可能含已有 HTML 标签）
 * @param rules 预编译的规则数组（已按优先级排序、按作用域过滤）
 * @returns 未消毒的 HTML 字符串，调用方需负责 sanitize
 */
export function applyRenderRules(
  escapedText: string,
  rules: CompiledRule[]
): string {
  if (!rules.length) return escapedText

  // 1. 按 HTML 标签切分，识别受保护的 <a> 内部内容
  const htmlSegments = splitByHtmlTags(escapedText)

  // 2. 对每个 text 段（非 protected_text、非 html）应用规则
  //    protected_text 和 html 段原样保留
  let result = ''
  for (const seg of htmlSegments) {
    if (seg.type === 'text') {
      result += applyRulesToText(seg.text, rules)
    } else {
      // protected_text 和 html 原样保留
      result += seg.text
    }
  }

  return result
}

/**
 * 对纯文本段应用所有规则（原有逻辑）
 */
function applyRulesToText(text: string, rules: CompiledRule[]): string {
  if (!text) return text

  let segments: MatchSegment[] = [{ type: 'text', text }]

  for (const rule of rules) {
    const next: MatchSegment[] = []
    for (const seg of segments) {
      if (seg.type !== 'text' || !seg.text) {
        next.push(seg)
        continue
      }
      // 重置 regex lastIndex（全局正则复用安全）
      rule.compiledRegex.lastIndex = 0
      const subSegments = splitByRule(seg.text, rule)
      next.push(...subSegments)
    }
    segments = next
  }

  // 渲染每段并拼接
  let html = ''
  for (const seg of segments) {
    if (seg.type === 'text') {
      html += seg.text || ''
    } else if (seg.type === 'rule_match' && seg.rule && seg.context) {
      html += renderMatchedSegment(seg.rule, seg.context, seg.matched || '')
    }
  }
  return html
}
