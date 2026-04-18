<template>
  <div class="message-bubble mini-app-message" :class="{ self: isSelf }">
    <div class="mini-app-info" @click="openMiniApp">
      <div class="mini-app-icon-container">
        <img :src="miniAppData?.icon" class="mini-app-icon" :alt="miniAppData?.name" />
      </div>
      <div class="mini-app-details">
        <div class="mini-app-name">{{ miniAppData?.name }}</div>
        <div class="mini-app-description">{{ miniAppData?.description }}</div>
        <div class="mini-app-tag">小程序</div>
      </div>
      <div class="mini-app-arrow">
        <i class="fas fa-chevron-right"></i>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  miniAppData?: {
    icon: string
    name: string
    description: string
  }
  isSelf?: boolean
}>()

const emit = defineEmits<{
  open: [data: any]
}>()

const openMiniApp = () => {
  emit('open', props.miniAppData)
}
</script>

<style scoped>
.mini-app-message {
  cursor: pointer;
  transition: all 0.2s ease;
  width: fit-content;
  max-width: 100%;
  box-sizing: border-box;
}

.mini-app-message:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.mini-app-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.mini-app-icon-container {
  flex-shrink: 0;
}

.mini-app-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  object-fit: cover;
}

.mini-app-details {
  flex: 1;
  min-width: 0;
}

.mini-app-name {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mini-app-description {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mini-app-tag {
  display: inline-block;
  font-size: 10px;
  padding: 2px 6px;
  background-color: var(--hover-color);
  border-radius: 4px;
  color: var(--text-secondary);
}

.mini-app-arrow {
  color: var(--text-secondary);
  font-size: 12px;
  flex-shrink: 0;
}

/* 自己的小程序消息样式 */
.mini-app-message.self .mini-app-name {
  color: white;
  font-weight: 600;
}

.mini-app-message.self .mini-app-description {
  color: rgba(255, 255, 255, 0.8);
}

.mini-app-message.self .mini-app-tag {
  color: rgba(255, 255, 255, 0.8);
  background-color: rgba(255, 255, 255, 0.1);
}

.mini-app-message.self .mini-app-arrow {
  color: rgba(255, 255, 255, 0.8);
}
</style>