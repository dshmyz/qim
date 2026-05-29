<template>
  <div class="user-app-container">
    <AppHeader :title="app.name" @back="$emit('back')">
      <template #extra-buttons>
        <ToggleSidebarBtn
          icon="fas fa-compress"
          title="收起侧边栏"
          @click="$emit('toggleSidebar')"
        />
      </template>
    </AppHeader>
    <div class="user-app-content">
      <div v-if="app.url" class="user-app-iframe-container">
        <iframe
          :src="app.url"
          class="user-app-iframe"
          frameborder="0"
          allowfullscreen
        ></iframe>
      </div>
      <div v-else class="empty-user-app">
        <div class="empty-icon"><i class="fas fa-link"></i></div>
        <p>该应用没有配置URL</p>
        <p class="empty-hint">请在应用管理中编辑应用，添加URL地址</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import AppHeader from './AppHeader.vue'
import ToggleSidebarBtn from '../shared/ToggleSidebarBtn.vue'

defineProps<{
  app: {
    name: string
    url: string
  }
}>()

defineEmits(['back', 'toggleSidebar'])
</script>

<style scoped>
.user-app-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.user-app-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.user-app-iframe-container {
  flex: 1;
  overflow: hidden;
}

.user-app-iframe {
  width: 100%;
  height: 100%;
  border: none;
}

.empty-user-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-secondary);
}

.empty-user-app .empty-icon {
  font-size: 48px;
  color: var(--text-tertiary);
}

.empty-user-app p {
  margin: 0;
  font-size: 14px;
}

.empty-user-app .empty-hint {
  font-size: 12px;
  color: var(--text-tertiary);
}
</style>
