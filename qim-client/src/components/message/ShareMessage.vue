<template>
  <div class="message-bubble share-message" :class="{ self: isSelf }">
    <div class="share-info">
      <div class="share-icon-container">
        <div class="share-icon" :class="shareData?.type">
          <i v-if="shareData?.type === 'file'" class="fas fa-file"></i>
          <i v-else-if="shareData?.type === 'note'" class="fas fa-file-alt"></i>
          <i v-else-if="shareData?.type === 'sticky'" class="fas fa-sticky-note"></i>
          <i v-else class="fas fa-share-alt"></i>
        </div>
        <div class="share-type">{{ getShareTypeText(shareData?.type) }}</div>
      </div>
      <div class="share-details">
        <div class="share-name">{{ shareData?.name || content }}</div>
        <div class="share-actions">
          <button class="share-action-btn" @click="viewSharedContent">查看</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  content: string
  shareData?: {
    type: string
    name: string
  }
  isSelf?: boolean
}>()

const emit = defineEmits<{
  view: [content: string]
}>()

const getShareTypeText = (type?: string): string => {
  switch (type) {
    case 'file':
      return '文件'
    case 'note':
      return '笔记'
    case 'message':
      return '消息'
    case 'sticky':
      return '便签'
    default:
      return '分享'
  }
}

const viewSharedContent = () => {
  emit('view', props.content)
}
</script>

<style scoped>
.share-message {
  background: var(--sidebar-bg);
  border-radius: 12px;
  padding: 14px;
  width: fit-content;
  max-width: 100%;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.share-message:hover {
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12);
}

.share-info {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.share-icon-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.share-icon {
  font-size: 24px;
  margin-top: 2px;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--list-bg);
  border-radius: 6px;
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.share-type {
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.2;
  white-space: nowrap;
  text-align: center;
  margin-bottom: 4px;
}

.share-details {
  flex: 1;
  min-width: 0;
}

.share-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 8px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
  word-break: break-all;
}

.share-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.share-action-btn {
  padding: 6px 16px;
  font-size: 12px;
  border-radius: 8px;
  border: none;
  background-color: var(--primary-light);
  color: var(--primary-color);
  cursor: pointer;
  transition: all 0.3s ease;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}

.share-action-btn:hover {
  background-color: var(--primary-color);
  color: #fff;
  border-color: var(--primary-color);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.3);
  transform: translateY(-1px);
}

.share-action-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.3);
}

/* 自己的分享消息样式 */
.share-message.self {
  background: var(--primary-color);
}

.share-message.self .share-name {
  color: #fff;
}

.share-message.self .share-type {
  color: rgba(255, 255, 255, 0.8);
}

.share-message.self .share-icon {
  background-color: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.3);
  color: #fff;
}

.share-message.self .share-action-btn {
  background-color: rgba(255, 255, 255, 0.2);
  border: none;
  border-color: rgba(255, 255, 255, 0.3);
  color: #fff;
}

.share-message.self .share-action-btn:hover {
  background-color: rgba(255, 255, 255, 0.3);
  border-color: rgba(255, 255, 255, 0.4);
  box-shadow: 0 4px 12px rgba(255, 255, 255, 0.3);
}
</style>