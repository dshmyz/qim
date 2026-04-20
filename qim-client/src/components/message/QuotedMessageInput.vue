<template>
  <div class="quoted-message">
    <div class="quoted-message-header">
      <span class="quoted-message-sender">{{ quotedMessage.sender?.name || quotedMessage.name || '未知用户' }}</span>
      <button class="quoted-message-remove" @click="$emit('remove')">×</button>
    </div>
    <div class="quoted-message-content">
      <template v-if="quotedMessage.type === 'text'">
        {{ quotedMessage.content || '无内容' }}
      </template>
      <template v-else-if="quotedMessage.type === 'image'">
        [图片] {{ getFileName(quotedMessage.content) }}
      </template>
      <template v-else-if="quotedMessage.type === 'file'">
        [文件] {{ getFileName(quotedMessage.content) }}
      </template>
      <template v-else-if="quotedMessage.type === 'mini-app' || quotedMessage.type === 'miniApp'">
        [小程序]
      </template>
      <template v-else-if="quotedMessage.type === 'share'">
        [分享]
      </template>
      <template v-else>
        {{ quotedMessage.content || '无内容' }}
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  quotedMessage: any
}>()

const emit = defineEmits<{
  remove: []
}>()

// 获取文件名
const getFileName = (content: string): string => {
  try {
    // 尝试解析content为JSON
    const contentObj = JSON.parse(content)
    if (contentObj.name) {
      return contentObj.name
    } else if (contentObj.fileName) {
      return contentObj.fileName
    }
  } catch (e) {
    // 解析失败，从content字符串中提取文件名
  }
  return content.split('/').pop() || ''
}
</script>

<style scoped>
/* 引用消息样式 */
.quoted-message {
  background: var(--hover-color);
  border-left: 4px solid var(--primary-color);
  padding: 10px;
  margin-bottom: 10px;
  border-radius: 4px;
}

.quoted-message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 5px;
}

.quoted-message-sender {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-color);
}

.quoted-message-remove {
  background: none;
  border: none;
  font-size: 16px;
  cursor: pointer;
  color: var(--text-secondary);
  padding: 0;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: background 0.2s;
}

.quoted-message-remove:hover {
  background: rgba(0, 0, 0, 0.1);
  color: var(--text-color);
}

.quoted-message-content {
  font-size: 14px;
  color: var(--text-color);
  line-height: 1.4;
}

/* 暗黑主题下的引用消息样式 */
[data-theme="dark"] .quoted-message-remove {
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .quoted-message-remove:hover {
  background: rgba(255, 255, 255, 0.1) !important;
  color: var(--text-color) !important;
}
</style>