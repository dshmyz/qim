<template>
  <span class="task-ref-card" :class="{ 'is-loading': loading, 'is-error': error }" @click.stop="handleClick">
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
      <span class="task-ref-card__icon" :class="`status-${task.status}`">
        <i :class="statusIcon"></i>
      </span>
      <span class="task-ref-card__title">{{ task.title }}</span>
      <span v-if="task.due_date" class="task-ref-card__due" :class="{ overdue: isOverdue }">
        <i class="far fa-calendar"></i>
        {{ dueLabel }}
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
  padding: 2px 8px;
  margin: 0 2px;
  border-radius: 6px;
  /* 中性灰底：用 --card-bg 与 --hover-color 混合，避开气泡的色相区间
     （self 气泡是主题色浅版，别人气泡是 message-bubble-bg，两者都偏色相；
      卡片走中性灰，靠"灰 vs 彩"和亮度差与气泡区分） */
  background: color-mix(in srgb, var(--card-bg, #fff), var(--hover-color, #f5f7fa) 60%);
  /* 去掉边框：小卡片上的实色边框显硬，靠背景色本身界定边界 */
  border: none;
  font-size: 13px;
  line-height: 1.4;
  cursor: pointer;
  transition: background 0.15s;
  vertical-align: baseline;
  max-width: 320px;
}
.task-ref-card:hover {
  /* hover 时混入更多 hover-color，背景明显变化 */
  background: color-mix(in srgb, var(--card-bg, #fff), var(--hover-color, #f5f7fa) 100%);
}
.task-ref-card.is-loading {
  background: var(--hover-color, #f5f7fa);
  cursor: progress;
}
.task-ref-card.is-error {
  background: transparent;
  padding: 0;
  cursor: default;
}
.task-ref-card__icon {
  font-size: 12px;
  flex-shrink: 0;
}
.task-ref-card__icon.status-completed { color: var(--color-success-500, #67c23a); }
.task-ref-card__icon.status-in_progress { color: var(--color-warning-500, #e6a23c); }
.task-ref-card__icon.status-todo { color: var(--color-gray-500, #909399); }
.task-ref-card__title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  /* 主题色文字：保留"可点击"提示，同时与中性灰底形成对比 */
  color: var(--primary-color, #3b82f6);
  font-weight: 500;
}
.task-ref-card__due {
  font-size: 11px;
  color: var(--text-secondary, #909399);
  flex-shrink: 0;
}
.task-ref-card__due.overdue {
  color: var(--color-error-500, #f56c6c);
}
.task-ref-card__label {
  color: var(--text-secondary, #909399);
}
.task-ref-card__label--fallback {
  color: var(--text-secondary, #909399);
  text-decoration: none;
}

/* 移动端适配：窄屏下收窄卡片宽度，避免溢出消息气泡 */
@media (max-width: 768px) {
  .task-ref-card {
    max-width: 240px;
    font-size: 12px;
  }
}
@media (max-width: 480px) {
  .task-ref-card {
    max-width: 180px;
    gap: 4px;
    padding: 2px 6px;
  }
  .task-ref-card__due {
    display: none; /* 小屏隐藏到期时间，优先展示标题 */
  }
}
</style>
