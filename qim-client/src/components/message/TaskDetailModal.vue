<template>
  <ModalContainer
    :visible="visible"
    title="任务详情"
    width="560px"
    max-height="80vh"
    :show-footer="false"
    @close="$emit('close')"
  >
    <div v-if="loading" class="task-detail-loading">
      <i class="fas fa-spinner fa-spin"></i>
      <span>加载中...</span>
    </div>

    <div v-else-if="error" class="task-detail-error">
      <i class="fas fa-exclamation-circle"></i>
      <span>{{ error }}</span>
    </div>

    <div v-else-if="task" class="task-detail">
      <!-- 标题区 -->
      <div class="task-detail-header">
        <span class="task-detail-icon" :class="`status-${task.status}`">
          <i :class="statusIcon"></i>
        </span>
        <h3 class="task-detail-title">{{ task.title }}</h3>
      </div>

      <!-- 状态条 -->
      <div class="task-detail-meta">
        <div class="task-detail-meta-item">
          <span class="task-detail-meta-label">状态</span>
          <span class="task-detail-meta-value">
            <span class="task-detail-badge" :class="`status-${task.status}`">
              {{ statusLabel }}
            </span>
          </span>
        </div>
        <div class="task-detail-meta-item">
          <span class="task-detail-meta-label">优先级</span>
          <span class="task-detail-meta-value">
            <span class="task-detail-badge" :class="`priority-${task.priority}`">
              {{ priorityLabel }}
            </span>
          </span>
        </div>
        <div v-if="dueLabel" class="task-detail-meta-item">
          <span class="task-detail-meta-label">到期</span>
          <span class="task-detail-meta-value" :class="{ overdue: isOverdue }">
            <i class="far fa-calendar"></i>
            {{ dueLabel }}
          </span>
        </div>
      </div>

      <!-- 描述 -->
      <div v-if="task.description" class="task-detail-section">
        <div class="task-detail-section-title">描述</div>
        <div class="task-detail-description">{{ task.description }}</div>
      </div>

      <!-- 指派人 / 创建者 -->
      <div class="task-detail-section task-detail-people">
        <div v-if="task.assignee_id" class="task-detail-person">
          <span class="task-detail-section-label">指派给</span>
          <span class="task-detail-person-name">{{ task.assignee_id }}</span>
        </div>
        <div class="task-detail-person">
          <span class="task-detail-section-label">创建者</span>
          <span class="task-detail-person-name">{{ task.user_id }}</span>
        </div>
      </div>

      <!-- 子任务 -->
      <div v-if="subTasks.length > 0" class="task-detail-section">
        <div class="task-detail-section-title">
          子任务
          <span class="task-detail-count">{{ subTasks.length }}</span>
        </div>
        <ul class="task-detail-subtasks">
          <li
            v-for="(sub, i) in subTasks"
            :key="i"
            class="task-detail-subtask"
            :class="{ done: sub.done }"
          >
            <i :class="sub.done ? 'fas fa-check-square' : 'far fa-square'"></i>
            <span>{{ sub.title }}</span>
          </li>
        </ul>
      </div>

      <!-- 标签 -->
      <div v-if="tags.length > 0" class="task-detail-section">
        <div class="task-detail-section-title">标签</div>
        <div class="task-detail-tags">
          <span v-for="tag in tags" :key="tag" class="task-detail-tag">{{ tag }}</span>
        </div>
      </div>

      <!-- 时间信息 -->
      <div class="task-detail-section task-detail-times">
        <span>创建于 {{ formatTime(task.created_at) }}</span>
        <span v-if="task.updated_at">更新于 {{ formatTime(task.updated_at) }}</span>
      </div>
    </div>
  </ModalContainer>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import ModalContainer from '../shared/ModalContainer.vue'
import { request } from '../../composables/useRequest'

interface Props {
  visible: boolean
  taskId: number
  conversationId: number
}

const props = defineProps<Props>()
const emit = defineEmits<{ close: [] }>()

interface SubTask { title: string; done: boolean }
interface Tag { name: string; color?: string }

interface TaskDetail {
  id: number
  title: string
  description: string
  status: string
  priority: string
  due_date: string | null
  assignee_id: string
  user_id: number
  sub_tasks: string
  tags: string
  created_at: string
  updated_at: string
}

const loading = ref(false)
const error = ref('')
const task = ref<TaskDetail | null>(null)

// 详情用独立请求（不复用卡片缓存，因为详情需要完整字段）
async function fetchTaskDetail() {
  if (!props.taskId || !props.conversationId) return
  loading.value = true
  error.value = ''
  task.value = null

  try {
    const res = await request<{ code: number; data: TaskDetail; message?: string }>(
      `/api/v1/tasks/${props.taskId}`,
      { params: { conversation_id: props.conversationId } }
    )
    if (res && res.code === 0 && res.data) {
      task.value = res.data
    } else {
      error.value = res?.message || '加载任务失败'
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载任务失败'
  } finally {
    loading.value = false
  }
}

watch(() => props.visible, (v) => {
  if (v) fetchTaskDetail()
}, { immediate: true })

const statusIcon = computed(() => {
  switch (task.value?.status) {
    case 'completed': return 'fas fa-check-circle'
    case 'in_progress': return 'fas fa-circle-half-stroke'
    default: return 'far fa-circle'
  }
})

const statusLabel = computed(() => {
  switch (task.value?.status) {
    case 'todo': return '待办'
    case 'in_progress': return '进行中'
    case 'completed': return '已完成'
    default: return task.value?.status || ''
  }
})

const priorityLabel = computed(() => {
  switch (task.value?.priority) {
    case 'high': return '高'
    case 'medium': return '中'
    case 'low': return '低'
    case 'urgent': return '紧急'
    default: return task.value?.priority || ''
  }
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

const isOverdue = computed(() => {
  if (!task.value?.due_date) return false
  return new Date(task.value.due_date).getTime() < Date.now() && task.value.status !== 'completed'
})

const subTasks = computed<SubTask[]>(() => {
  if (!task.value?.sub_tasks) return []
  try {
    const parsed = JSON.parse(task.value.sub_tasks)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
})

const tags = computed<string[]>(() => {
  if (!task.value?.tags) return []
  try {
    const parsed = JSON.parse(task.value.tags)
    if (Array.isArray(parsed)) {
      // tags 可能是 string[] 或 {name, color}[]
      return parsed.map((t: string | Tag) => typeof t === 'string' ? t : t.name)
    }
    return []
  } catch {
    return []
  }
})

function formatTime(timeStr: string): string {
  if (!timeStr) return ''
  const d = new Date(timeStr)
  return d.toLocaleString('zh-CN', {
    year: 'numeric', month: 'numeric', day: 'numeric',
    hour: '2-digit', minute: '2-digit'
  })
}
</script>

<style scoped>
.task-detail-loading,
.task-detail-error {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px 0;
  color: #909399;
  font-size: 14px;
}
.task-detail-error {
  color: #f56c6c;
}

.task-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 标题区 */
.task-detail-header {
  display: flex;
  align-items: center;
  gap: 10px;
}
.task-detail-icon {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  flex-shrink: 0;
}
.task-detail-icon.status-completed { background: #f0f9eb; color: #67c23a; }
.task-detail-icon.status-in_progress { background: #fdf6ec; color: #e6a23c; }
.task-detail-icon.status-todo { background: #f4f4f5; color: #909399; }
.task-detail-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  line-height: 1.4;
  word-break: break-word;
}

/* 状态条 */
.task-detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 8px;
}
.task-detail-meta-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.task-detail-meta-label {
  font-size: 12px;
  color: #909399;
}
.task-detail-meta-value {
  font-size: 13px;
  color: #303133;
  display: flex;
  align-items: center;
  gap: 4px;
}
.task-detail-meta-value.overdue { color: #f56c6c; }
.task-detail-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 500;
}
.task-detail-badge.status-completed { background: #f0f9eb; color: #67c23a; }
.task-detail-badge.status-in_progress { background: #fdf6ec; color: #e6a23c; }
.task-detail-badge.status-todo { background: #f4f4f5; color: #909399; }
.task-detail-badge.priority-high { background: #fef0f0; color: #f56c6c; }
.task-detail-badge.priority-urgent { background: #fef0f0; color: #f56c6c; }
.task-detail-badge.priority-medium { background: #fdf6ec; color: #e6a23c; }
.task-detail-badge.priority-low { background: #f0f9eb; color: #67c23a; }

/* 区块 */
.task-detail-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.task-detail-section-title {
  font-size: 13px;
  font-weight: 600;
  color: #606266;
  display: flex;
  align-items: center;
  gap: 6px;
}
.task-detail-count {
  background: #e4e7ed;
  color: #606266;
  padding: 0 6px;
  border-radius: 8px;
  font-size: 11px;
  font-weight: 500;
}
.task-detail-description {
  font-size: 13px;
  color: #303133;
  line-height: 1.6;
  padding: 10px 12px;
  background: #fafbfc;
  border-radius: 6px;
  border: 1px solid #ebeef5;
  white-space: pre-wrap;
  word-break: break-word;
}

/* 人员 */
.task-detail-people {
  flex-direction: row;
  gap: 24px;
  padding: 10px 12px;
  background: #fafbfc;
  border-radius: 6px;
  border: 1px solid #ebeef5;
}
.task-detail-person {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.task-detail-section-label {
  font-size: 12px;
  color: #909399;
}
.task-detail-person-name {
  font-size: 13px;
  color: #303133;
}

/* 子任务 */
.task-detail-subtasks {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.task-detail-subtask {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 6px;
  font-size: 13px;
  color: #303133;
  background: #fafbfc;
}
.task-detail-subtask.done {
  color: #909399;
  text-decoration: line-through;
}
.task-detail-subtask i {
  font-size: 14px;
  color: #c0c4cc;
}
.task-detail-subtask.done i {
  color: #67c23a;
}

/* 标签 */
.task-detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.task-detail-tag {
  background: #ecf5ff;
  color: #409eff;
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 12px;
}

/* 时间 */
.task-detail-times {
  flex-direction: row;
  gap: 16px;
  padding-top: 8px;
  border-top: 1px solid #ebeef5;
  font-size: 12px;
  color: #909399;
}
</style>
