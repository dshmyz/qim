/**
 * /quick 斜杠命令：触发后弹出快速回复短语列表，选中插入到输入框（可编辑后发送）。
 *
 * 数据源：/api/v1/user-settings/quick_replies（用户自定义，未自定义时后端返回 has_value=false，前端兜底默认短语）。
 * 每次触发都拉最新数据（/quick 是低频操作，无需缓存）。
 * 后端请求失败时用前端兜底默认值，保证断网也能用核心短语。
 * 与 /note 的区别：/note 直接发分享消息，/quick 只插入文本，用户可改可发。
 *
 * 整个斜杠命令面板的开关由 useSlashCommandPanelEnabled 控制，在 ChatWindow 输入检测入口统一拦截，
 * 不在单个命令上判断 available，避免分散。
 */

import { markRaw } from 'vue'
import QuickCommandItem from '../../components/chat/QuickCommandItem.vue'
import { fetchQuickReplies, DEFAULT_QUICK_REPLIES } from '../../api/quickReplies'
import type { SlashCommand, SlashCommandItem } from '../slashCommand'

/** 快速回复候选项：text 是短语本身，id 用短语作 id（去重） */
export interface QuickReplyItem extends SlashCommandItem {
  id: string
  text: string
}

/** 把短字符串数组转成候选项 */
function toItems(replies: string[]): QuickReplyItem[] {
  return replies.map((text, idx) => ({ id: `qr-${idx}-${text}`, text }))
}

/** 每次都拉最新数据；失败时用前端兜底默认值 */
async function fetchItems(): Promise<QuickReplyItem[]> {
  try {
    const replies = await fetchQuickReplies()
    // fetchQuickReplies 已处理"未自定义→默认值"，理论上总返回非空数组；空数组也走兜底
    return toItems(replies.length > 0 ? replies : DEFAULT_QUICK_REPLIES)
  } catch {
    return toItems(DEFAULT_QUICK_REPLIES)
  }
}

function filter(items: QuickReplyItem[], query: string): QuickReplyItem[] {
  const q = query.trim().toLowerCase()
  if (!q) return items
  return items.filter(it => it.text.toLowerCase().includes(q))
}

/** 选中后把短语插入输入框（末尾带空格，方便用户继续输入） */
function getInsertText(item: QuickReplyItem): string {
  return `${item.text} `
}

/** /quick 命令：所有会话类型可用（面板开关在 ChatWindow 入口统一拦截） */
export const quickCommand: SlashCommand<QuickReplyItem> = {
  trigger: '/quick',
  title: '快速回复',
  icon: 'fas fa-bolt',
  description: '插入常用短语',
  fetchItems,
  filter,
  getInsertText,
  itemComponent: markRaw(QuickCommandItem),
}
