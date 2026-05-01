<template>
  <div class="assignee-selector">
    <label class="assignee-label">指派人</label>
    <div class="assignee-dropdown" v-if="showDropdown">
      <input
        class="assignee-search"
        v-model="searchQuery"
        placeholder="搜索联系人..."
      />
      <div class="assignee-options">
        <button class="assignee-option" @click="selectAssignee(null)">
          <span class="no-assignee">未指派</span>
        </button>
        <button
          v-for="user in filteredUsers"
          :key="user.id"
          class="assignee-option"
          :class="{ selected: user.id === modelValue }"
          @click="selectAssignee(user.id)"
        >
          <div class="user-avatar" :style="{ background: getAvatarColor(user.name) }">
            {{ user.name.charAt(0) }}
          </div>
          <span>{{ user.name }}</span>
        </button>
      </div>
    </div>
    <button class="assignee-trigger" @click="showDropdown = !showDropdown">
      <template v-if="currentAssignee">
        <div class="user-avatar" :style="{ background: getAvatarColor(currentAssignee.name) }">
          {{ currentAssignee.name.charAt(0) }}
        </div>
        <span>{{ currentAssignee.name }}</span>
      </template>
      <template v-else>
        <i class="fas fa-user-plus"></i>
        <span>指派</span>
      </template>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { TaskUser } from '../../../../types/task'

const props = defineProps<{
  modelValue: string | null
  contacts: TaskUser[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string | null]
}>()

const showDropdown = ref(false)
const searchQuery = ref('')

const currentAssignee = computed(() =>
  props.contacts.find(u => u.id === props.modelValue) || null
)

const filteredUsers = computed(() => {
  if (!searchQuery.value) return props.contacts
  const q = searchQuery.value.toLowerCase()
  return props.contacts.filter(u => u.name.toLowerCase().includes(q))
})

function selectAssignee(id: string | null) {
  emit('update:modelValue', id)
  showDropdown.value = false
  searchQuery.value = ''
}

function getAvatarColor(name: string) {
  const colors = ['#8b5cf6', '#6366f1', '#ec4899', '#f59e0b', '#10b981', '#3b82f6']
  const index = name.charCodeAt(0) % colors.length
  return colors[index]
}
</script>

<style scoped>
.assignee-selector { margin-bottom: var(--spacing-3); position: relative; }
.assignee-label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: var(--spacing-1);
}
.assignee-trigger {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2) var(--spacing-3);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  cursor: pointer;
  font-size: 12px;
  color: var(--text-primary);
}
.assignee-trigger:hover { border-color: #8b5cf6; }
.assignee-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: var(--z-dropdown, 1000);
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: var(--spacing-2);
  min-width: 200px;
  margin-top: var(--spacing-1);
}
.assignee-search {
  width: 100%;
  padding: var(--spacing-1) var(--spacing-2);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  font-size: 12px;
  margin-bottom: var(--spacing-2);
  color: var(--text-primary);
  background: var(--input-bg);
}
.assignee-search:focus { outline: none; border-color: #8b5cf6; }
.assignee-options {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 200px;
  overflow-y: auto;
}
.assignee-option {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-1) var(--spacing-2);
  border: none;
  background: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12px;
  color: var(--text-primary);
  text-align: left;
}
.assignee-option:hover { background: var(--hover-bg); }
.assignee-option.selected { background: #f5f3ff; }
.no-assignee { color: var(--text-secondary); font-style: italic; }
.user-avatar {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9px;
  color: #fff;
  font-weight: 500;
  flex-shrink: 0;
}
</style>
