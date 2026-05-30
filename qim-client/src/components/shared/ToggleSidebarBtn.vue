<template>
  <button
    class="toggle-sidebar-btn"
    :class="{ active: isActive }"
    :title="title"
    @click="$emit('click')"
  >
    <i :class="icon"></i>
  </button>
</template>

<script setup lang="ts">
interface Props {
  icon?: string
  title?: string
  isActive?: boolean
}

withDefaults(defineProps<Props>(), {
  icon: 'fas fa-chevron-left',
  title: '切换侧边栏',
  isActive: false
})

defineEmits<{
  click: []
}>()
</script>

<style scoped>
.toggle-sidebar-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid var(--border-color);
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  color: var(--primary-color);
  font-size: 14px;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.toggle-sidebar-btn:hover {
  background: var(--primary-light);
  color: var(--primary-color);
}

.toggle-sidebar-btn:active {
  transform: scale(0.95);
}

.toggle-sidebar-btn.active {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.toggle-sidebar-btn.active:hover {
  background: var(--active-color);
  border-color: var(--active-color);
  color: white;
}
</style>
