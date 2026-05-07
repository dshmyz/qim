<template>
  <div class="app-header">
    <div class="header-left">
      <button class="back-btn" @click="$emit('back')">
        <i class="fas fa-chevron-left"></i>
      </button>
      <slot name="extra-buttons"></slot>
      <div class="header-info">
        <h2>
          <i v-if="icon" :class="icon" class="header-icon"></i>
          {{ title }}
        </h2>
        <slot name="subtitle"></slot>
      </div>
    </div>
    <div class="header-right">
      <slot name="actions"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  icon?: string
}>()

defineEmits<{
  back: []
}>()
</script>

<style scoped>
.app-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background-color: var(--card-bg);
  height: 72px;
  box-sizing: border-box;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.back-btn {
  width: 28px;
  height: 28px;
  border: 1px solid var(--border-color);
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: background 0.2s;
  color: var(--primary-color);
}

.back-btn:hover {
  background: var(--primary-light);
}

.header-left :deep(.icon-btn) {
  width: 28px;
  height: 28px;
  border: 1px solid var(--border-color);
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: background 0.2s;
  color: var(--primary-color);
}

.header-left :deep(.icon-btn:hover) {
  background: var(--primary-light);
}

.header-left :deep(.toggle-sidebar-btn) {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: all 0.3s ease;
  color: var(--primary-color);
}

.header-left :deep(.toggle-sidebar-btn:hover) {
  background: var(--primary-light);
}

.header-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.header-info h2 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-info :deep(.header-description) {
  margin: 0;
  font-size: 12px;
  color: var(--text-secondary);
}

.header-icon {
  color: var(--primary-color);
  font-size: 18px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-right :deep(.search-box) {
  position: relative;
  display: flex;
  align-items: center;
}

.header-right :deep(.search-input) {
  padding: 8px 12px 8px 36px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  background-color: var(--bg-color);
  color: var(--text-primary);
  transition: all 0.3s ease;
  width: 200px;
}

.header-right :deep(.search-input:focus) {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  width: 250px;
}

.header-right :deep(.search-icon) {
  position: absolute;
  left: 12px;
  color: var(--text-secondary);
  font-size: 14px;
  transition: color 0.3s ease;
}

.header-right :deep(.search-input:focus + .search-icon) {
  color: var(--primary-color);
}

.header-right :deep(.create-btn),
.header-right :deep(.create-note-btn),
.header-right :deep(.create-app-btn),
.header-right :deep(.create-task-btn),
.header-right :deep(.create-event-btn) {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 14px;
  height: 32px;
  background-color: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  white-space: nowrap;
}

.header-right :deep(.create-btn:hover),
.header-right :deep(.create-note-btn:hover),
.header-right :deep(.create-app-btn:hover),
.header-right :deep(.create-task-btn:hover),
.header-right :deep(.create-event-btn:hover) {
  background-color: var(--active-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.header-right :deep(.action-btn),
.header-right :deep(.create-task-btn),
.header-right :deep(.create-event-btn) {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 14px;
  height: 32px;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-color);
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s ease;
}

.header-right :deep(.action-btn:hover),
.header-right :deep(.create-task-btn:hover),
.header-right :deep(.create-event-btn:hover) {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.header-right :deep(.action-btn.primary),
.header-right :deep(.create-task-btn),
.header-right :deep(.create-event-btn) {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: #fff;
}

.header-right :deep(.action-btn.primary:hover),
.header-right :deep(.create-task-btn:hover),
.header-right :deep(.create-event-btn:hover) {
  background: var(--active-color);
  border-color: var(--active-color);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

@media (max-width: 768px) {
  .app-header {
    padding: 12px;
    height: auto;
    min-height: 72px;
  }

  .header-right :deep(.search-box) {
    width: 100%;
  }

  .header-right :deep(.search-input) {
    width: 100%;
  }

  .header-right :deep(.search-input:focus) {
    width: 100%;
  }

  .header-right :deep(.create-btn),
  .header-right :deep(.create-note-btn),
  .header-right :deep(.create-app-btn),
  .header-right :deep(.create-task-btn),
  .header-right :deep(.create-event-btn) {
    align-self: stretch;
  }
}
</style>
