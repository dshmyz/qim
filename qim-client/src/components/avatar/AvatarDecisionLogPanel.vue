<template>
  <div class="avatar-decision-log">
    <div class="log-header">
      <div class="header-text">
        <h4 class="log-title">运行记录</h4>
        <span class="log-subtitle">最近 {{ events.length }} 条分身决策及原因</span>
      </div>
      <button class="refresh-btn" :disabled="loading" title="刷新" @click="fetchEvents">
        <i class="fas fa-sync-alt" :class="{ spinning: loading }"></i>
      </button>
    </div>

    <div class="search-section">
      <input
        v-model="searchQuery"
        class="search-input"
        placeholder="筛选消息内容、原因或发送者..."
      />
      <i class="fas fa-search search-icon"></i>
    </div>

    <div v-if="loading" class="log-state">
      <LoadingSpinner />
    </div>

    <div v-else-if="filteredEvents.length === 0" class="log-state">
      <i class="fas fa-clipboard-list empty-icon"></i>
      <p class="empty-text">{{ events.length === 0 ? '暂无分身决策记录' : '没有匹配的记录' }}</p>
    </div>

    <div v-else class="event-list">
      <div
        v-for="event in filteredEvents"
        :key="event.id"
        class="event-item"
      >
        <div class="event-icon-col">
          <i :class="iconClass(event)"></i>
        </div>
        <div class="event-body">
          <div class="event-head">
            <span class="event-decision" :class="event.decision">{{ stageLabel(event.stage) }}</span>
            <span v-if="event.reason" class="event-reason">{{ event.reason }}</span>
          </div>
          <p v-if="event.message_snippet" class="event-snippet">{{ event.message_snippet }}</p>
          <div class="event-meta">
            <span v-if="event.trigger_name" class="meta-item">
              <i class="fas fa-user"></i>{{ event.trigger_name }}
            </span>
            <span class="meta-item">
              <i class="fas fa-comment"></i>#{{ event.conversation_id }}
            </span>
            <span class="meta-item">
              <i class="far fa-clock"></i>{{ formatDate(event.created_at) }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { avatarAPI } from '../../api/avatar'
import type { AvatarDecisionEvent } from '../../types/avatar'
import { formatDate } from '../../utils/date'
import LoadingSpinner from '../shared/LoadingSpinner.vue'

const events = ref<AvatarDecisionEvent[]>([])
const loading = ref(false)
const searchQuery = ref('')

const stageLabels: Record<AvatarDecisionEvent['stage'], string> = {
  trigger: '未触发',
  stand_down: '分身让位',
  empty: '主动不回复',
  quality: '质量核验未过',
  queue_busy: '队列繁忙',
  generate_error: '生成失败',
  sent: '已回复'
}

function stageLabel(stage: AvatarDecisionEvent['stage']): string {
  return stageLabels[stage] ?? stage
}

function iconClass(event: AvatarDecisionEvent): string {
  if (event.decision === 'replied') return 'fas fa-check-circle replied-icon'
  const byStage: Record<AvatarDecisionEvent['stage'], string> = {
    trigger: 'fas fa-minus-circle',
    stand_down: 'fas fa-hand-paper',
    empty: 'fas fa-ban',
    quality: 'fas fa-exclamation-circle',
    queue_busy: 'fas fa-hourglass-half',
    generate_error: 'fas fa-times-circle',
    sent: 'fas fa-check-circle'
  }
  return `${byStage[event.stage] ?? 'fas fa-question-circle'} skipped-icon`
}

const filteredEvents = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase()
  if (!keyword) return events.value
  return events.value.filter((event) =>
    event.message_snippet?.toLowerCase().includes(keyword) ||
    event.reason?.toLowerCase().includes(keyword) ||
    event.trigger_name?.toLowerCase().includes(keyword) ||
    String(event.conversation_id).includes(keyword)
  )
})

async function fetchEvents() {
  loading.value = true
  try {
    events.value = await avatarAPI.listDecisionEvents({ limit: 100 })
  } catch {
    events.value = []
  } finally {
    loading.value = false
  }
}

defineExpose({ fetchEvents })

onMounted(fetchEvents)
</script>

<style scoped>
.avatar-decision-log {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
}

/* ---- 头部 ---- */
.log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.log-title {
  margin: 0;
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-primary);
}

.log-subtitle {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}

.refresh-btn {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  background: var(--card-bg, #fff);
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.2s;
}

.refresh-btn:hover {
  color: var(--primary-color);
  border-color: var(--primary-color);
}

.refresh-btn:disabled {
  cursor: default;
}

.spinning {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ---- 搜索 ---- */
.search-section {
  position: relative;
  margin: 16px 0;
}

.search-input {
  width: 100%;
  padding: 9px 34px 9px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--input-bg, var(--card-bg, #fff));
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  outline: none;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.search-input:focus {
  border-color: var(--primary-color);
}

.search-icon {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  pointer-events: none;
}

/* ---- 状态 ---- */
.log-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 40px 0;
}

.empty-icon {
  font-size: var(--font-size-2xl);
  color: var(--text-secondary);
  opacity: 0.5;
}

.empty-text {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

/* ---- 事件列表 ---- */
.event-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.event-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: 10px;
  background: var(--card-bg, #fff);
}

.event-icon-col {
  width: 20px;
  display: flex;
  justify-content: center;
  flex-shrink: 0;
  padding-top: 1px;
}

.event-icon-col i {
  font-size: var(--font-size-base);
}

.replied-icon {
  color: #26b361;
}

.skipped-icon {
  color: var(--text-secondary);
}

.event-body {
  flex: 1;
  min-width: 0;
}

.event-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  flex-wrap: wrap;
}

.event-decision {
  font-size: var(--font-size-sm);
  font-weight: 600;
  flex-shrink: 0;
}

.event-decision.replied {
  color: #26b361;
}

.event-decision.skipped {
  color: var(--text-secondary);
}

.event-reason {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.event-snippet {
  margin: 6px 0 0;
  font-size: var(--font-size-sm);
  color: var(--text-primary);
  line-height: 1.5;
  word-break: break-word;
}

.event-meta {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 8px;
  flex-wrap: wrap;
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}
</style>
