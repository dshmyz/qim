<template>
  <div class="message-card" :class="{ 'is-creator': isCreator }" role="article" :aria-label="`来自 ${senderName} 的消息`">
    <div class="card-accent-bar"></div>
    <div class="card-body">
      <div class="card-top">
        <div class="card-author">
          <Avatar
            :src="message.sender?.avatar"
            :name="getDisplayName(message.sender)"
            :server-url="serverUrl"
            :alt="`${senderName}的头像`"
            size="sm"
            class="author-avatar"
          />
          <div class="author-info">
            <span class="author-name">
              {{ senderName }}
              <span v-if="isCreator" class="creator-badge">
                <i class="fas fa-crown"></i> 创建者
              </span>
            </span>
            <span class="author-time">{{ formatTime(message.created_at) }}</span>
          </div>
        </div>
      </div>

      <div class="card-content">
        <p class="content-text">{{ message.content }}</p>
      </div>

      <div v-if="interactive" class="card-actions">
        <button
          class="action-btn like-btn"
          :class="{ active: isLiked }"
          @click="handleLike"
          :aria-label="isLiked ? '取消点赞' : '点赞'"
          :aria-pressed="isLiked"
        >
          <i :class="isLiked ? 'fas fa-heart' : 'far fa-heart'"></i>
          <span>{{ likeCount > 0 ? likeCount : '点赞' }}</span>
        </button>
        <button class="action-btn" @click="handleComment" aria-label="评论">
          <i class="far fa-comment"></i>
          <span>评论</span>
        </button>
        <button class="action-btn" @click="handleCopyLink" aria-label="复制链接">
          <i class="far fa-copy"></i>
          <span>复制</span>
        </button>
      </div>
      <div v-else class="card-actions-locked">
        <i class="fas fa-lock"></i>
        <span>订阅后可互动</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import Avatar from '../shared/Avatar.vue'
import { getDisplayName } from '../../utils/avatar'
import { API_BASE_URL } from '../../config'
import { useChatUtils } from '../../composables/useChatUtils'
import { request } from '../../composables/useRequest'
import type { ChannelMessage } from '../../types'

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

interface Props {
  message: ChannelMessage
  isCreator?: boolean
  interactive?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  isCreator: false,
  interactive: true
})

const emit = defineEmits<{
  like: [message: ChannelMessage]
  unlike: [message: ChannelMessage]
  comment: [message: ChannelMessage]
  copyLink: [message: ChannelMessage]
}>()

const { formatTime } = useChatUtils()

const isLiked = ref(false)
const likeCount = ref(0)
const loadingLike = ref(false)

const senderName = computed(() => getDisplayName(props.message.sender))

onMounted(async () => {
  try {
    const res = await request(`/api/v1/channels/messages/${props.message.id}/likes`)
    if (res.code === 0 && res.data) {
      isLiked.value = res.data.is_liked
      likeCount.value = res.data.like_count
    }
  } catch {
    // 默认值即可
  }
})

const handleLike = async () => {
  if (loadingLike.value) return
  loadingLike.value = true

  if (isLiked.value) {
    try {
      const res = await request(`/api/v1/channels/messages/${props.message.id}/unlike`, { method: 'POST' })
      if (res.code === 0) {
        isLiked.value = res.data?.is_liked ?? false
        likeCount.value = res.data?.like_count ?? likeCount.value - 1
        emit('unlike', props.message)
      }
    } catch {
      // 回滚
      isLiked.value = true
      likeCount.value++
    }
  } else {
    try {
      const res = await request(`/api/v1/channels/messages/${props.message.id}/like`, { method: 'POST' })
      if (res.code === 0) {
        isLiked.value = res.data?.is_liked ?? true
        likeCount.value = res.data?.like_count ?? likeCount.value + 1
        emit('like', props.message)
      }
    } catch {
      // 回滚
      isLiked.value = false
      likeCount.value--
    }
  }
  loadingLike.value = false
}

const handleComment = () => {
  emit('comment', props.message)
}

const handleCopyLink = async () => {
  emit('copyLink', props.message)
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
  display: flex;
  background: var(--card-bg);
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid var(--border-color);
  transition: box-shadow 0.2s, transform 0.15s;
}

.message-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  transform: translateY(-1px);
}

.card-accent-bar {
  width: 4px;
  flex-shrink: 0;
  background: var(--primary-color);
  opacity: 0.6;
  transition: opacity 0.2s;
}

.message-card:hover .card-accent-bar {
  opacity: 1;
}

.message-card.is-creator .card-accent-bar {
  background: linear-gradient(180deg, var(--primary-color), var(--success-color));
  opacity: 1;
}

.card-body {
  flex: 1;
  padding: 16px 20px;
  min-width: 0;
}

.card-top {
  margin-bottom: 12px;
}

.card-author {
  display: flex;
  align-items: center;
  gap: 10px;
}

.author-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  border: 2px solid var(--border-color);
}

.message-card.is-creator .author-avatar {
  border-color: var(--primary-light, rgba(51, 133, 255, 0.3));
}

.author-info {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.author-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
  display: flex;
  align-items: center;
  gap: 6px;
}

.creator-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 10px;
  padding: 1px 6px;
  background: linear-gradient(135deg, var(--primary-color), #6366f1);
  color: white;
  border-radius: 10px;
  font-weight: 500;
}

.creator-badge i {
  font-size: 8px;
}

.author-time {
  font-size: 12px;
  color: var(--text-secondary);
}

.card-content {
  margin-bottom: 12px;
}

.content-text {
  margin: 0;
  font-size: 15px;
  color: var(--text-color);
  line-height: 1.7;
  word-break: break-word;
  white-space: pre-wrap;
}

.card-actions {
  display: flex;
  gap: 4px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 5px 12px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 12px;
  border-radius: 6px;
  transition: all 0.15s;
}

.action-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.action-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.action-btn i {
  font-size: 14px;
}

.like-btn.active {
  color: var(--danger-color);
}

.like-btn.active:hover {
  background: rgba(239, 68, 68, 0.08);
  color: var(--danger-color);
}

.card-actions-locked {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding-top: 12px;
  border-top: 1px dashed var(--border-color);
  font-size: 12px;
  color: var(--text-secondary);
  opacity: 0.5;
}

.card-actions-locked i {
  font-size: 11px;
}
</style>
