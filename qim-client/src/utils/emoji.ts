import twemoji from 'twemoji'
import { CLASSIC_EMOJIS } from './classic-emoji'

/**
 * 表情图片化工具（Twemoji 14.0.2，CC-BY 4.0 资产）
 *
 * 资产已自托管在 public/emoji/72x72/，不依赖系统表情字体，
 * 保证 macOS / Windows / Linux 下渲染一致。
 */

const EMOJI_OPTIONS = {
  base: '/emoji/',
  size: '72x72',
  ext: '.png',
  className: 'emoji-img',
} as const

/**
 * 将文本/已消毒 HTML 中的表情字符替换为 <img>。
 * 仅替换表情码点，标签与其它文本原样保留，可安全接在 sanitizeHTML/sanitizeMarkdown 之后。
 */
export function emojiToHtml(html: string): string {
  if (!html) return ''
  return twemoji.parse(html, EMOJI_OPTIONS)
}

/**
 * 单个表情字符 → 图片 URL，供面板 :src 绑定。
 * 复刻 twemoji.parse 内部 grabTheRightIcon：无 ZWJ(U+200D) 序列时去掉 VS16(U+FE0F)，
 * 以匹配资产命名（如 ❤️ → 2764.png，而非 2764-fe0f.png）。
 */
export function emojiUrl(emoji: string): string {
  if (!emoji) return ''
  const hasZwj = emoji.includes('‍')
  const cleaned = hasZwj ? emoji : emoji.replace(/️/g, '')
  return `/emoji/72x72/${twemoji.convert.toCodePoint(cleaned)}.png`
}

// 经典表情：名称 → id，用于把 [名称] 标记替换为 <img>
const CLASSIC_NAME_TO_ID = new Map<string, number>()
for (const e of CLASSIC_EMOJIS) {
  if (!CLASSIC_NAME_TO_ID.has(e.name)) CLASSIC_NAME_TO_ID.set(e.name, e.id)
}

const escapeRegExp = (s: string): string => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

// 按名称长度降序，避免短名称先匹配（如 [示爱] 优先于 [爱]）
const CLASSIC_MARKER_REGEX = new RegExp(
  '\\[(' +
    [...CLASSIC_NAME_TO_ID.keys()]
      .sort((a, b) => b.length - a.length)
      .map(escapeRegExp)
      .join('|') +
    ')\\]',
  'g',
)

/**
 * 把已消毒 HTML 中的经典表情标记 [名称] 替换为 <img>。
 * 仅替换白名单内的已知名称，用户输入的普通 [文字] 不受影响。
 * 与 emojiToHtml 一样应接在 sanitizeHTML/sanitizeMarkdown 之后。
 */
export function classicToHtml(html: string): string {
  if (!html) return ''
  return html.replace(CLASSIC_MARKER_REGEX, (_, name: string) => {
    const id = CLASSIC_NAME_TO_ID.get(name)
    if (id === undefined) return _
    return `<img class="emoji-img classic-emoji-img" src="/emoji/classic/${id}.gif" alt="[${name}]" title="[${name}]" draggable="false"/>`
  })
}

