<!--
  MessageCard.vue - 卡片模式消息组件

  功能：
  - 显示消息卡片（头像、发送者、时间、内容）
  - 显示操作按钮（点赞、评论、复制链接）
  - 支持创建者标识

  使用示例：
  <MessageCard
    :message="message"
    :is-creator="isCreator"
  />
-->
<template>
  <div class="message-card" role="article" :aria-label="`来自 ${senderName} 的消息`">
    <div class="card-main">
      <img
        :src="getAvatarUrl(message.sender?.avatar, senderName, serverUrl)"
        :alt="`${senderName}的头像`"
        class="card-avatar"
      />
      <div class="card-content">
        <div class="card-header">
          <span class="card-sender">
            {{ senderName }}
            <span v-if="isCreator" class="creator-badge">创建者</span>
          </span>
          <span class="card-time">{{ formatTime(message.created_at) }}</span>
        </div>
        <div class="card-body">
          <p class="card-text">{{ message.content }}</p>
        </div>
      </div>
    </div>
    <div class="card-actions">
      <button
        class="action-btn"
        :class="{ active: isLiked }"
        @click="handleLike"
        :aria-label="isLiked ? '取消点赞' : '点赞'"
        :aria-pressed="isLiked"
      >
        <i :class="isLiked ? 'fas fa-heart' : 'far fa-heart'"></i>
        <span>{{ likeCount > 0 ? likeCount : '点赞' }}</span>
      </button>
      <button
        class="action-btn"
        @click="handleComment"
        aria-label="评论"
      >
        <i class="far fa-comment"></i>
        <span>评论</span>
      </button>
      <button
        class="action-btn"
        @click="handleCopyLink"
        aria-label="复制链接"
      >
        <i class="far fa-copy"></i>
        <span>复制</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { getAvatarUrl } from '../../utils/avatar'
import { useChatUtils } from '../../composables/useChatUtils'
import { API_BASE_URL } from '../../config'
import type { ChannelMessage } from '../../types'

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

interface Props {
  message: ChannelMessage
  isCreator?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  isCreator: false
})

const emit = defineEmits<{
  like: [message: ChannelMessage]
  unlike: [message: ChannelMessage]
  comment: [message: ChannelMessage]
  copyLink: [message: ChannelMessage]
}>()

const { formatTime } = useChatUtils()

// 点赞状态（本地状态）
// 注意：当前 ChannelMessage 类型定义中不包含点赞相关字段
// 这是临时方案，使用本地状态管理点赞功能
// TODO: 等待后端支持后，应从 message 数据中获取点赞状态和点赞数
// 届时需要在 ChannelMessage 类型中添加：
// - like_count?: number
// - is_liked?: boolean
const isLiked = ref(false)
const likeCount = ref(0)

const senderName = computed(() => props.message.sender?.name || '未知用户')

const handleLike = () => {
  if (isLiked.value) {
    isLiked.value = false
    likeCount.value--
    emit('unlike', props.message)
  } else {
    isLiked.value = true
    likeCount.value++
    emit('like', props.message)
  }
}

const handleComment = () => {
  emit('comment', props.message)
}

const handleCopyLink = async () => {
  emit('copyLink', props.message)
  // 复制链接到剪贴板
  try {
    const url = `${window.location.origin}/channels/${props.message.channel_id}/messages/${props.message.id}`
    await navigator.clipboard.writeText(url)
  } catch (error) {
    console.error('复制链接失败:', error)
  }
}
</script>

<style scoped>
.message-card {
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  padding: var(--spacing-4);
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-color);
  transition: all var(--transition-fast);
}

.card-main {
  display: flex;
  gap: var(--spacing-3);
}

.card-avatar {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  object-fit: cover;
  flex-shrink: 0;
}

.card-content {
  flex: 1;
  min-width: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--spacing-2);
}

.card-sender {
  font-weight: var(--font-weight-medium);
  color: var(--text-color);
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}

.creator-badge {
  font-size: var(--font-size-xs);
  padding: 2px var(--spacing-2);
  background: var(--primary-color);
  color: white;
  border-radius: var(--radius-sm);
  font-weight: var(--font-weight-medium);
}

.card-time {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.card-body {
  margin: 0;
}

.card-text {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--text-color);
  line-height: 1.6;
  word-break: break-word;
  white-space: pre-wrap;
}

.card-actions {
  display: flex;
  gap: var(--spacing-2);
  margin-top: var(--spacing-3);
  padding-top: var(--spacing-3);
  border-top: 1px solid var(--border-color);
}

.action-btn {
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
  padding: var(--spacing-1) var(--spacing-3);
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: var(--font-size-xs);
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}

.action-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.action-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.action-btn.active {
  color: var(--danger-color);
}

.action-btn.active:hover {
  color: var(--danger-color);
}

.action-btn i {
  font-size: 14px;
}
</style>
