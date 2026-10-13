<template>
  <div class="message-hits-panel">
    <div class="hits-header">
      <span>聊天记录</span>
      <span class="hits-count">{{ total }} 条相关消息</span>
    </div>
    <div class="hits-list">
      <div v-for="group in groups" :key="group.conversationId" class="hits-group">
        <div class="hits-group-header">
          <Avatar :src="group.avatar" :name="group.name" :server-url="serverUrl" size="xs" />
          <span class="hits-group-name">{{ group.name }}</span>
          <span class="hits-group-count">{{ group.hits.length }} 条</span>
        </div>
        <div
          v-for="hit in group.hits"
          :key="hit.id"
          class="hits-item"
          @click="$emit('select', hit)"
        >
          <span class="hits-item-sender">{{ hit.senderName }}</span>
          <span class="hits-item-time">{{ hit.time }}</span>
          <span class="hits-item-content" v-html="renderPreview(hit)"></span>
        </div>
        <button
          v-if="canLoadMore(group)"
          class="hits-more-btn"
          @click="$emit('loadMore', group)"
        >
          查看该会话更多结果
        </button>
      </div>
      <div v-if="isLoadingMore" class="hits-loading">加载中...</div>
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
  /** 已按类型折算的纯文本预览：文本消息为正文纯文本，其余为类型标签（[图片] 等） */
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
  /** 会话名/头像解析：单聊 members 常为空，按 other_member 兜底 */
  resolveConversation: (conversationId: number | string) => { name: string, avatar: string } | null
  /** 已确认无更多结果的会话 ID（点击"查看更多"后没有新命中时由父级标记） */
  exhaustedIds?: string[]
  isLoadingMore?: boolean
}>()

defineEmits<{
  (e: 'select', hit: MessageHit): void
  (e: 'loadMore', group: MessageHitGroup): void
}>()

// 组内超过该条数才出现"查看更多"——更少说明该会话命中大概率已在首屏拿全
const MAX_GROUP_HITS = 5

/** 转义后再包 <mark>：消息正文按纯文本渲染，尖括号内容不会变成元素 */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function renderPreview(hit: MessageHit): string {
  const text = hit.previewText
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
  return group.hits.length >= MAX_GROUP_HITS && !exhausted.value.has(String(group.conversationId))
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
  width: 100%;
  max-height: 420px;
  background: var(--sidebar-bg);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  z-index: 100;
  overflow: hidden;
  margin-bottom: 8px;
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
}

.hits-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
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
  max-height: 380px;
  overflow-y: auto;
}

.hits-group-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px 4px;
  background: var(--panel-bg);
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

.hits-group-count {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  margin-left: auto;
  flex-shrink: 0;
}

.hits-item {
  display: flex;
  align-items: baseline;
  gap: 6px;
  padding: 6px 16px 6px 40px;
  cursor: pointer;
  transition: background-color 0.15s ease;
  min-width: 0;
}

.hits-item:hover {
  background-color: var(--hover-color);
}

.hits-item-sender {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  flex-shrink: 0;
  max-width: 72px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.hits-item-time {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  flex-shrink: 0;
}

.hits-item-content {
  font-size: var(--font-size-xs);
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.hits-item-content :deep(mark) {
  background: var(--primary-light, #ffe58f);
  color: inherit;
  padding: 0 1px;
  border-radius: 2px;
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
}

.hits-more-btn:hover {
  background: var(--hover-color);
}

.hits-loading {
  padding: 8px 16px;
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  text-align: center;
}
</style>
