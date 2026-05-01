<template>
  <ModalContainer
    :visible="visible"
    :title="task ? '编辑任务' : '创建任务'"
    @close="$emit('close')"
  >
    <div class="task-form-group">
      <label>任务标题</label>
      <input type="text" class="task-form-input" v-model="form.title" placeholder="请输入任务标题">
    </div>
    <div class="task-form-group">
      <label>任务描述</label>
      <textarea class="task-form-textarea" v-model="form.description" placeholder="请输入任务描述"></textarea>
    </div>
    <div class="task-form-row">
      <div class="task-form-group">
        <label>截止日期</label>
        <input type="date" class="task-form-input" v-model="form.due_date">
      </div>
      <div class="task-form-group">
        <label>优先级</label>
        <select class="task-form-select" v-model="form.priority">
          <option value="low">低</option>
          <option value="medium">中</option>
          <option value="high">高</option>
        </select>
      </div>
    </div>
    <div class="task-form-group">
      <label>状态</label>
      <select class="task-form-select" v-model="form.status">
        <option value="todo">待办</option>
        <option value="in_progress">进行中</option>
        <option value="completed">已完成</option>
      </select>
    </div>
    <template #footer>
      <button class="task-modal-btn task-cancel-btn" @click="$emit('close')">取消</button>
      <button class="task-modal-btn task-confirm-btn" @click="onSubmit">
        {{ task ? '更新' : '创建' }}
      </button>
    </template>
  </ModalContainer>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import type { Task, TaskPriority, TaskStatus } from '../../../../types/task'
import ModalContainer from '../../../shared/ModalContainer.vue'

const props = defineProps<{
  visible: boolean
  task?: Task | null
}>()

const emit = defineEmits<{
  close: []
  submit: [data: {
    title: string
    description: string
    due_date: string | null
    priority: TaskPriority
    status: TaskStatus
  }]
}>()

const form = reactive({
  title: '',
  description: '',
  due_date: '' as string | null,
  priority: 'medium' as TaskPriority,
  status: 'todo' as TaskStatus
})

watch(() => props.visible, (val) => {
  if (val) {
    if (props.task) {
      form.title = props.task.title
      form.description = props.task.description
      form.due_date = props.task.due_date
      form.priority = props.task.priority
      form.status = props.task.status
    } else {
      form.title = ''
      form.description = ''
      form.due_date = null
      form.priority = 'medium'
      form.status = 'todo'
    }
  }
})

function onSubmit() {
  if (!form.title.trim()) return
  emit('submit', {
    title: form.title.trim(),
    description: form.description.trim(),
    due_date: form.due_date || null,
    priority: form.priority,
    status: form.status
  })
}
</script>

<style scoped>
.task-form-group { margin-bottom: var(--spacing-3); }
.task-form-group label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: var(--spacing-1);
}
.task-form-input,
.task-form-textarea,
.task-form-select {
  width: 100%;
  padding: var(--spacing-2) var(--spacing-3);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  font-size: 13px;
  color: var(--text-primary);
  background: var(--input-bg);
  transition: border-color var(--animation-fast) ease;
  box-sizing: border-box;
}
.task-form-input:focus,
.task-form-textarea:focus,
.task-form-select:focus { outline: none; border-color: #8b5cf6; }
.task-form-textarea { resize: vertical; min-height: 80px; }
.task-form-row { display: flex; gap: var(--spacing-3); }
.task-form-row .task-form-group { flex: 1; }
.task-modal-btn {
  padding: var(--spacing-2) var(--spacing-4);
  border: none;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--animation-fast) ease;
}
.task-cancel-btn {
  background: var(--hover-bg);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}
.task-cancel-btn:hover { background: var(--border-color); }
.task-confirm-btn { background: #8b5cf6; color: #fff; }
.task-confirm-btn:hover { background: #7c3aed; }
</style>
