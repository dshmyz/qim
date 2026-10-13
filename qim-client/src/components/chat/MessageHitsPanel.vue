<template>
  <div class="message-hits-panel">
    <div class="hits-header">
      <span>聊天记录</span>
      <span class="hits-count">{{ total }} 条相关消息</span>
    </div>
    <div class="hits-list">
      <div v-for="group in groups" :key="group.conversationId" class="hits-group">
        <div class="hits-group-header" @click="$emit('openConversation', group.conversationId)">
          <Avatar :src="group.avatar" :name="group.name" :server-url="serverUrl" size="xs" />
          <span class="hits-group-name">{{ group.name }}</span>
          <span class="hits-group-count">{{ group.hits.length }} 条</span>
          <i class="fas fa-chevron-right hits-group-arrow"></i>
        </div>
        <div
          v-for="hit in group.hits"
          :key="hit.id"
          class="hits-item"
          :data-hit-id="hit.id"
          @click="$emit('select', hit)"
        >
          <div class="hits-item-top">
            <span class="hits-item-sender">{{ hit.senderName }}</span>
            <span class="hits-item-time">{{ hit.time }}</span>
          </div>
          <div class="hits-item-body">
            <span v-if="!isTextType(hit.type)" class="hits-item-badge">
              <i :class="typeIcon(hit.type)"></i>{{ typeLabel(hit.type) }}
            </span>
            <span class="hits-item-content" v-html="renderPreview(hit)"></span>
          </div>
        </div>
        <button
          v-if="canLoadMore(group)"
          class="hits-more-btn"
          @click="$emit('loadMore', group)"
        >
          查看该会话更多结果
          <i class="fas fa-chevron-down"></i>
        </button>
      </div>
      <div v-if="isLoadingMore" class="hits-loading">加载中...</div>
      <div v-else-if="!hits.length" class="hits-empty">无相关聊天记录</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Avatar from '../shared/Avatar.vue'

export interface MessageHit {
  id: number | string
  conversationId: number | string
  senderName: string
  time: string
  /** 已按类型折算的纯文本预览：文本消息为正文纯文本，其余为空串（由徽标展示类型） */
  previewText: string
  type: string
}

export interface MessageHitGroup {
  conversationId: number | string
  name: string
  avatar: string
  hits: MessageHit[]
}

const props = defineProps<{
  hits: MessageHit[]
  total: number
  keyword: string
  serverUrl: string
  /** 会话名/头像解析：优先后端 conversation_meta，兜底本地会话列表 */
  resolveConversation: (conversationId: number | string) => { name: string, avatar: string } | null
  /** 已确认无更多结果的会话 ID（点击"查看更多"后没有新命中时由父级标记） */
  exhaustedIds?: string[]
  isLoadingMore?: boolean
}>()

defineEmits<{
  (e: 'select', hit: MessageHit): void
  (e: 'loadMore', group: MessageHitGroup): void
  (e: 'openConversation', conversationId: number | string): void
}>()

// 组内超过该条数才出现"查看更多"——更少说明该会话命中大概率已在首屏拿全
const MAX_GROUP_HITS = 5

const TYPE_LABELS: Record<string, string> = {
  image: '图片',
  file: '文件',
  video: '视频',
  audio: '语音',
  miniApp: '小程序',
  news: '资讯',
  // card = bot 发出的可操作卡片（title + buttons，见 validateCardContent），
  // 不是名片——与「成员名片」浮层无关
  card: '卡片',
  card_action: '按钮操作',
  merged_forward: '聊天记录',
  share: '分享',
}

const TYPE_ICONS: Record<string, string> = {
  image: 'fas fa-image',
  file: 'fas fa-file',
  video: 'fas fa-video',
  audio: 'fas fa-microphone',
  miniApp: 'fas fa-th-large',
  news: 'fas fa-newspaper',
  card: 'fas fa-rectangle-list',
  card_action: 'fas fa-hand-pointer',
  merged_forward: 'fas fa-comments',
  share: 'fas fa-share-alt',
}

const typeLabel = (type: string): string => TYPE_LABELS[type] || '消息'
const typeIcon = (type: string): string => TYPE_ICONS[type] || 'fas fa-comment'
// markdown 是文本类消息，出正文片段而非类型徽标
const isTextType = (type: string): boolean => type === 'text' || type === 'markdown'

/** 转义后再包 <mark>：消息正文按纯文本渲染，尖括号内容不会变成元素 */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

/** 预览截断：从首个关键词命中位置往前取一点上下文，命中点尽量留在窗口内 */
function clipPreview(text: string, keyword: string, window: number): string {
  if (text.length <= window) return text
  const kw = keyword.trim().toLowerCase()
  const at = kw ? text.toLowerCase().indexOf(kw) : -1
  if (at < 0) return text.slice(0, window) + '…'
  const start = Math.max(0, at - 10)
  const prefix = start > 0 ? '…' : ''
  return prefix + text.slice(start, start + window) + (start + window < text.length ? '…' : '')
}

function renderPreview(hit: MessageHit): string {
  if (!isTextType(hit.type)) return ''
  const text = clipPreview(hit.previewText, props.keyword, 80)
  const kw = props.keyword.trim().toLowerCase()
  if (!text || !kw) return escapeHtml(text)
  const lower = text.toLowerCase()
  let html = ''
  let idx = 0
  while (idx < text.length) {
    const at = lower.indexOf(kw, idx)
    if (at === -1) {
      html += escapeHtml(text.slice(idx))
      break
    }
    html += escapeHtml(text.slice(idx, at))
    html += `<mark>${escapeHtml(text.slice(at, at + kw.length))}</mark>`
    idx = at + kw.length
  }
  return html
}

const exhausted = computed(() => new Set(props.exhaustedIds || []))

const canLoadMore = (group: MessageHitGroup): boolean => {
  if (group.hits.length < MAX_GROUP_HITS) return false
  if (exhausted.value.has(String(group.conversationId))) return false
  // 全局命中已全部加载（total ≤ 已加载条数）时该会话不可能有更多，
  // 显示"查看更多"只会点击即静默无反应（实测：全会话命中 15 条全在首屏）
  return props.total > props.hits.length
}

const groups = computed<MessageHitGroup[]>(() => {
  const map = new Map<string, MessageHitGroup & { order: number }>()
  let seq = 0
  for (const hit of props.hits) {
    const key = String(hit.conversationId)
    let group = map.get(key)
    if (!group) {
      const meta = props.resolveConversation(hit.conversationId) || { name: '会话', avatar: '' }
      group = {
        conversationId: hit.conversationId,
        name: meta.name,
        avatar: meta.avatar,
        hits: [],
        order: seq++,
      }
      map.set(key, group)
    }
    group.hits.push(hit)
  }
  return [...map.values()].sort((a, b) => a.order - b.order)
})
</script>

<style scoped>
.message-hits-panel {
  position: relative;
  /* 不设 width:100%：flex 列容器内靠 align-self:stretch + 水平 margin 内缩，
     显式 width:100% 会让 margin 溢出父容器 8px */
  box-sizing: border-box;
  max-height: 460px;
  background: var(--sidebar-bg);
  border-radius: 8px;
  /* 浮层与侧栏同为 --sidebar-bg（浅色主题均纯白），靠阴影与标题分层条浮起 */
  box-shadow: 0 8px 20px rgba(0, 0, 0, 0.12);
  z-index: 100;
  overflow: hidden;
  /* 水平边距与上方搜索框（padding 8px）对齐，不贴侧栏边缘 */
  margin: 0 8px 8px;
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
}

.hits-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border-color);
  /* 标题行常驻主题浅色分层（--hover-color：浅色主题为浅蓝/浅紫，深色为亮层），
     白底白卡靠它获得第一层视觉锚点 */
  background: var(--hover-color);
  font-weight: 600;
  font-size: var(--font-size-sm);
  color: var(--text-color);
  flex-shrink: 0;
}

.hits-count {
  font-weight: normal;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}

.hits-list {
  max-height: 420px;
  overflow-y: auto;
}

.hits-group-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px 4px;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.hits-group-header:hover {
  background-color: var(--hover-color);
}

.hits-group-count {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  flex-shrink: 0;
}

.hits-group-arrow {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  margin-left: auto;
}

.hits-group-name {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.hits-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 8px 16px 8px 40px;
  cursor: pointer;
  transition: background-color 0.15s ease;
  min-width: 0;
}

.hits-item:hover {
  background-color: var(--hover-color);
}

/* 第一行：发送人（主层级）居左，时间（次要层级）钉在右端 */
.hits-item-top {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}

.hits-item-sender {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.hits-item-time {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  flex-shrink: 0;
}

/* 第二行：正文片段独占整行，降为次要色让关键词高亮成为视觉焦点 */
.hits-item-body {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.hits-item-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  background: var(--hover-color);
  padding: 1px 6px;
  border-radius: 4px;
  flex-shrink: 0;
}

.hits-item-badge i {
  font-size: var(--font-size-xxs);
}

.hits-item-content {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.hits-item-content :deep(mark) {
  background: var(--primary-light, #ffe58f);
  color: inherit;
  font-weight: 600;
  padding: 0 1px;
  border-radius: 2px;
}

.hits-item-time {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  flex-shrink: 0;
}

.hits-more-btn {
  margin: 4px 16px 8px 40px;
  padding: 4px 10px;
  border: none;
  background: none;
  color: var(--primary-color);
  font-size: var(--font-size-xs);
  cursor: pointer;
  text-align: left;
  border-radius: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.hits-more-btn:hover {
  background: var(--hover-color);
}

.hits-loading,
.hits-empty {
  padding: 12px 16px;
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  text-align: center;
}
</style>
