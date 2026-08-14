<template>
  <span
    class="task-ref-card"
    :class="[
      priorityClass,
      statusClass,
      { 'is-loading': loading, 'is-error': error }
    ]"
    @click.stop="handleClick"
  >
    <!-- 加载态 -->
    <template v-if="loading">
      <i class="fas fa-spinner fa-spin task-ref-card__icon"></i>
      <span class="task-ref-card__label">{{ refText }}</span>
    </template>

    <!-- 失败态：降级为纯文本引用 -->
    <template v-else-if="error || !task">
      <span class="task-ref-card__label task-ref-card__label--fallback">{{ refText }}</span>
    </template>

    <!-- 成功态：轻量卡片 -->
    <template v-else>
      <span class="task-ref-card__bar"></span>
      <span class="task-ref-card__icon" :class="`status-${task.status}`">
        <i :class="statusIcon"></i>
      </span>
      <span class="task-ref-card__body">
        <span class="task-ref-card__title">{{ task.title }}</span>
        <span class="task-ref-card__meta">
          <span v-if="task.due_date" class="task-ref-card__due" :class="{ overdue: isOverdue }">
            <i class="far fa-calendar"></i>
            {{ dueLabel }}
          </span>
        </span>
      </span>
    </template>

    <!-- 任务详情浮层 -->
    <TaskDetailModal
      :visible="showDetail"
      :task-id="taskId"
      :conversation-id="conversationId"
      @close="showDetail = false"
    />
  </span>
</template>

<script lang="ts">
// 模块级缓存，所有 TaskRefCard 实例共享
// 避免同一任务在多条消息里重复拉取
// TTL 5 分钟，超过后自动失效；maxSize 100，超限淘汰最旧
import { createTTLCache } from '../../utils/ttlCache'

export interface TaskRef {
  id: number
  title: string
  status: string
  priority: string
  due_date: string | null
  assignee_id: string
}

const TaskRefCardCache = createTTLCache<TaskRef>({ ttl: 5 * 60 * 1000, maxSize: 100 })
</script>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { request } from '../../composables/useRequest'
import TaskDetailModal from './TaskDetailModal.vue'

const props = defineProps<{
  taskId: number
  conversationId: number
}>()

const emit = defineEmits<{
  click: [taskId: number]
}>()

// 详情浮层状态
const showDetail = ref(false)

const loading = ref(true)
const error = ref(false)
const task = ref<TaskRef | null>(null)

const refText = computed(() => `#T-${props.taskId}`)

// TTL 缓存：同一会话内同一任务在 5 分钟内只拉一次
const cache = TaskRefCardCache

const priorityClass = computed(() => {
  switch (task.value?.priority) {
    case 'high': return 'priority-high'
    case 'medium': return 'priority-medium'
    case 'low': return 'priority-low'
    default: return 'priority-default'
  }
})

const statusClass = computed(() => task.value ? `status-${task.value.status}` : '')

const statusIcon = computed(() => {
  switch (task.value?.status) {
    case 'completed': return 'fas fa-check-circle'
    case 'in_progress': return 'fas fa-circle-half-stroke'
    default: return 'far fa-circle'
  }
})

const isOverdue = computed(() => {
  if (!task.value?.due_date) return false
  return new Date(task.value.due_date).getTime() < Date.now() && task.value.status !== 'completed'
})

const dueLabel = computed(() => {
  if (!task.value?.due_date) return ''
  const d = new Date(task.value.due_date)
  const now = new Date()
  const sameYear = d.getFullYear() === now.getFullYear()
  const opts: Intl.DateTimeFormatOptions = sameYear
    ? { month: 'numeric', day: 'numeric' }
    : { year: 'numeric', month: 'numeric', day: 'numeric' }
  return d.toLocaleDateString('zh-CN', opts)
})

async function fetchTask() {
  const cacheKey = `${props.taskId}_${props.conversationId}`
  const cached = cache.get(cacheKey)
  if (cached) {
    task.value = cached
    loading.value = false
    return
  }

  try {
    const res = await request<{ code: number; data: TaskRef }>(
      `/api/v1/tasks/${props.taskId}`,
      { method: 'GET', params: { conversation_id: props.conversationId } }
    )
    if (res && res.code === 0 && res.data) {
      task.value = res.data
      cache.set(cacheKey, res.data)
    } else {
      error.value = true
    }
  } catch (e) {
    error.value = true
  } finally {
    loading.value = false
  }
}

function handleClick() {
  emit('click', props.taskId)
  // 加载失败时不弹浮层（避免展示空详情）
  if (!error.value) {
    showDetail.value = true
  }
}

onMounted(() => {
  fetchTask()
})
</script>

<style scoped>
.task-ref-card {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  margin: 2px 2px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--card-bg, #fff), var(--hover-color, #f5f7fa) 50%);
  border: 1px solid color-mix(in srgb, var(--border-color, #e4e7ed) 60%, transparent);
  font-size: var(--font-size-xs);
  line-height: 1.4;
  cursor: pointer;
  transition: all 0.2s ease;
  vertical-align: baseline;
  max-width: 320px;
  position: relative;
}
.task-ref-card:hover {
  background: color-mix(in srgb, var(--card-bg, #fff), var(--hover-color, #f5f7fa) 80%);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  transform: translateY(-1px);
}
.task-ref-card.is-loading {
  background: var(--hover-color, #f5f7fa);
  cursor: progress;
}
.task-ref-card.is-error {
  background: transparent;
  padding: 0;
  cursor: default;
  border: none;
}

/* ── 优先级左侧色条 ── */
.task-ref-card__bar {
  width: 3px;
  align-self: stretch;
  border-radius: 2px;
  flex-shrink: 0;
}
.task-ref-card.priority-high .task-ref-card__bar { background: #ef4444; }
.task-ref-card.priority-medium .task-ref-card__bar { background: #f59e0b; }
.task-ref-card.priority-low .task-ref-card__bar { background: #60a5fa; }
.task-ref-card.priority-default .task-ref-card__bar { background: var(--border-color, #dcdfe6); }

/* ── 状态图标 ── */
.task-ref-card__icon {
  font-size: var(--font-size-sm);
  flex-shrink: 0;
}
.task-ref-card__icon.status-completed { color: var(--color-success-500, #26b361); }
.task-ref-card__icon.status-in_progress { color: var(--color-warning-500, #f7a826); }
.task-ref-card__icon.status-todo { color: var(--text-secondary, #a0a0a0); }

/* ── 内容区 ── */
.task-ref-card__body {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  flex: 1;
}
.task-ref-card__title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-color, #404040);
  font-weight: 500;
  font-size: var(--font-size-xs);
}
/* 已完成：标题加删除线 + 降透明度，和看板一致 */
.task-ref-card.status-completed .task-ref-card__title {
  text-decoration: line-through;
  color: var(--text-secondary, #a0a0a0);
}
.task-ref-card.status-completed {
  opacity: 0.7;
}

.task-ref-card__meta {
  display: flex;
  align-items: center;
  gap: 6px;
}
.task-ref-card__due {
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary, #a0a0a0);
}
.task-ref-card__due.overdue {
  color: var(--color-error-500, #f56c6c);
  font-weight: 500;
}
.task-ref-card__label {
  color: var(--text-secondary, #a0a0a0);
}
.task-ref-card__label--fallback {
  color: var(--text-secondary, #a0a0a0);
  text-decoration: none;
}

/* 移动端适配：窄屏下收窄卡片宽度，避免溢出消息气泡 */
@media (max-width: 768px) {
  .task-ref-card {
    max-width: 240px;
    font-size: var(--font-size-xxs);
  }
}
@media (max-width: 480px) {
  .task-ref-card {
    max-width: 180px;
    gap: 4px;
    padding: 4px 8px;
  }
  .task-ref-card__due {
    display: none; /* 小屏隐藏到期时间，优先展示标题 */
  }
}
</style>
