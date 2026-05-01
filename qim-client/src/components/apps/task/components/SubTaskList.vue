<template>
  <div class="subtask-list">
    <label class="subtask-label">子任务 ({{ completedCount }}/{{ subTasks.length }})</label>
    <div class="subtask-items">
      <div v-for="st in subTasks" :key="st.id" class="subtask-item">
        <button class="subtask-check" @click="$emit('toggle', st.id)">
          <i v-if="st.completed" class="fas fa-check-square" style="color:#34d399;"></i>
          <i v-else class="far fa-square" style="color:var(--border-color);"></i>
        </button>
        <span class="subtask-title" :class="{ completed: st.completed }">{{ st.title }}</span>
        <button class="subtask-delete" @click="$emit('remove', st.id)">
          <i class="fas fa-times"></i>
        </button>
      </div>
    </div>
    <div class="subtask-add">
      <input
        v-model="newSubTask"
        class="subtask-input"
        placeholder="添加子任务..."
        @keydown.enter="addSubTask"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { SubTask } from '../../../../types/task'

const props = defineProps<{
  subTasks: SubTask[]
}>()

const emit = defineEmits<{
  toggle: [id: string]
  remove: [id: string]
  add: [title: string]
}>()

const newSubTask = ref('')

const completedCount = computed(() =>
  props.subTasks.filter(st => st.completed).length
)

function addSubTask() {
  const title = newSubTask.value.trim()
  if (!title) return
  emit('add', title)
  newSubTask.value = ''
}
</script>

<style scoped>
.subtask-list { margin-bottom: var(--spacing-3); }
.subtask-label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: var(--spacing-1);
}
.subtask-items { display: flex; flex-direction: column; gap: var(--spacing-1); }
.subtask-item { display: flex; align-items: center; gap: var(--spacing-2); padding: var(--spacing-1) 0; }
.subtask-check { background: none; border: none; cursor: pointer; padding: 0; font-size: 14px; }
.subtask-title { flex: 1; font-size: 12px; color: var(--text-primary); }
.subtask-title.completed { text-decoration: line-through; color: var(--text-secondary); }
.subtask-delete {
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
  font-size: 10px;
  color: var(--text-secondary);
  opacity: 0;
  transition: opacity var(--animation-fast) ease;
}
.subtask-item:hover .subtask-delete { opacity: 1; }
.subtask-delete:hover { color: #ef4444; }
.subtask-add { margin-top: var(--spacing-2); }
.subtask-input {
  width: 100%;
  padding: var(--spacing-1) var(--spacing-2);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--text-primary);
  background: var(--input-bg);
}
.subtask-input:focus { outline: none; border-color: #8b5cf6; }
</style>
