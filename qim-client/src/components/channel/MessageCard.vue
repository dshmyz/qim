<template>
  <div class="message-card" :class="{ 'is-creator': isCreator, 'has-comments': comments.length > 0 || showCommentInput }" role="article" :aria-label="`来自 ${senderName} 的消息`">
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
        <div ref="contentRef" class="content-text" v-html="renderedContent"></div>
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
        <button 
          v-if="canComment"
          class="action-btn" 
          @click="toggleCommentInput" 
          :class="{ active: showCommentInput }"
          aria-label="评论"
        >
          <i class="far fa-comment"></i>
          <span>{{ comments.length > 0 ? comments.length : '评论' }}</span>
        </button>
      </div>
      <div v-else-if="interactive && !canComment" class="card-actions-locked">
        <i class="fas fa-lock"></i>
        <span>评论已关闭</span>
      </div>
      <div v-else class="card-actions-locked">
        <i class="fas fa-lock"></i>
        <span>订阅后可互动</span>
      </div>

      <div v-if="showCommentInput" class="comment-section">
        <div v-if="comments.length > 0" class="comments-list">
          <div 
            v-for="comment in comments" 
            :key="comment.id" 
            class="comment-item"
          >
            <Avatar
              :src="comment.user?.avatar"
              :name="getDisplayName(comment.user)"
              :server-url="serverUrl"
              :alt="`${getDisplayName(comment.user)}的头像`"
              size="xs"
              class="comment-avatar"
            />
            <div class="comment-content">
              <span class="comment-author">{{ getDisplayName(comment.user) }}</span>
              <span class="comment-text" v-html="renderComment(comment.content)"></span>
              <span class="comment-time">{{ formatTime(comment.created_at) }}</span>
            </div>
          </div>
        </div>

        <div class="comment-input-wrapper">
          <textarea
            v-model="commentContent"
            placeholder="写下你的评论..."
            rows="2"
            class="comment-textarea"
            @keydown.enter.ctrl="submitComment"
            :aria-label="'评论输入框'"
          ></textarea>
          <div class="comment-actions">
            <span class="comment-hint">Ctrl + Enter 发送</span>
            <button
              class="comment-submit-btn"
              @click="submitComment"
              :disabled="!commentContent.trim()"
              :aria-label="'发送评论'"
            >
              <i class="fas fa-paper-plane"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import Avatar from '../shared/Avatar.vue'
import { getDisplayName } from '../../utils/avatar'
import { useServerUrl } from '../../composables/useServerUrl'
import { useChatUtils } from '../../composables/useChatUtils'
import { useCodeHighlight } from '../../composables/useCodeHighlight'
import { request } from '../../composables/useRequest'
import { renderChannelMarkdown } from '../../utils/channelMarkdown'
import type { ChannelMessage, Channel } from '../../types'

const { serverUrl } = useServerUrl()

// 频道正文：声明式 Markdown（方案 A）
const contentRef = ref<HTMLElement | null>(null)
const renderedContent = computed(() => renderChannelMarkdown(props.message.content))
useCodeHighlight(contentRef, renderedContent)

interface Comment {
  id: number
  message_id: number
  user_id: number
  content: string
  created_at: string
  user?: {
    id: number
    avatar: string
    nickname?: string
    username?: string
    name?: string
  }
}

interface Props {
  message: ChannelMessage
  channel?: Channel
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
}>()

const { formatTime } = useChatUtils()

const isLiked = ref(false)
const likeCount = ref(0)
const loadingLike = ref(false)
const showCommentInput = ref(false)
const comments = ref<Comment[]>([])
const commentContent = ref('')
const loadingComments = ref(false)
const submittingComment = ref(false)

const senderName = computed(() => getDisplayName(props.message.sender))

const canComment = computed(() => {
  if (!props.channel) return true
  return props.channel.comment_permission !== 'disabled'
})

// 评论渲染：与正文一致走声明式 Markdown（短文本同样受益于链接/Markdown 解析）
const renderComment = (content: string): string => renderChannelMarkdown(content)

onMounted(async () => {
  await Promise.all([
    loadLikeStatus(),
    loadComments()
  ])
})

const loadLikeStatus = async () => {
  try {
    const res = await request(`/api/v1/channels/messages/${props.message.id}/likes`)
    if (res.code === 0 && res.data) {
      isLiked.value = res.data.is_liked
      likeCount.value = res.data.like_count
    }
  } catch {
    // 默认值即可
  }
}

const loadComments = async () => {
  loadingComments.value = true
  try {
    const res = await request(`/api/v1/channels/messages/${props.message.id}/comments`)
    if (res.code === 0 && res.data) {
      comments.value = res.data
    }
  } catch {
    // 默认值即可
  }
  loadingComments.value = false
}

const handleLike = async () => {
  if (loadingLike.value) return
  loadingLike.value = true

  if (isLiked.value) {
    try {
      const res = await request(`/api/v1/channels/messages/${props.message.id}/like`, { method: 'DELETE' })
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

const toggleCommentInput = async () => {
  showCommentInput.value = !showCommentInput.value
  if (showCommentInput.value && comments.value.length === 0 && !loadingComments.value) {
    await loadComments()
  }
}

const submitComment = async () => {
  if (!commentContent.value.trim() || submittingComment.value) return
  
  submittingComment.value = true
  
  try {
    const res = await request(`/api/v1/channels/messages/${props.message.id}/comments`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ content: commentContent.value.trim() })
    })
    
    if (res.code === 0 && res.data) {
      comments.value.push(res.data)
      commentContent.value = ''
      emit('comment', props.message)
    }
  } catch (error) {
    console.error('提交评论失败:', error)
  }
  
  submittingComment.value = false
}
</script>

<style scoped>
.message-card {
  display: flex;
  background: var(--card-bg);
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--border-color);
  box-shadow: 0 3px 10px rgba(0, 0, 0, 0.06);
  transition: box-shadow 0.2s;
}

.message-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
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
  padding: 18px;
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

.content-text :deep(.emoji-img) {
  width: 18px;
  height: 18px;
  vertical-align: middle;
  margin: 0 1px;
}

/* Markdown 声明的排版 */
.content-text :deep(h1),
.content-text :deep(h2),
.content-text :deep(h3) {
  font-weight: 600;
  color: var(--text-color);
  margin: 1em 0 0.5em 0;
  line-height: 1.3;
}
.content-text :deep(h1) { font-size: 1.45em; }
.content-text :deep(h2) { font-size: 1.25em; }
.content-text :deep(h3) { font-size: 1.1em; }
.content-text :deep(strong) { font-weight: 600; }
.content-text :deep(em) { font-style: italic; }
.content-text :deep(a) {
  color: var(--primary-color);
  text-decoration: none;
}
.content-text :deep(a:hover) {
  text-decoration: underline;
}
.content-text :deep(p) { margin: 0.5em 0; }
.content-text :deep(ul),
.content-text :deep(ol) {
  margin: 0.5em 0;
  padding-left: 1.6em;
}
.content-text :deep(li) { margin: 0.25em 0; }
.content-text :deep(blockquote) {
  margin: 0.5em 0;
  padding: 0.25em 1em;
  border-left: 3px solid var(--primary-color);
  color: var(--text-secondary);
  background: var(--hover-color);
  border-radius: 0 4px 4px 0;
}
.content-text :deep(hr) {
  border: none;
  border-top: 1px solid var(--border-color);
  margin: 1em 0;
}
.content-text :deep(pre) {
  background: var(--hover-color);
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  font-size: 13px;
  line-height: 1.5;
  margin: 0.5em 0;
}
.content-text :deep(code) {
  background: var(--hover-color);
  padding: 2px 5px;
  border-radius: 4px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 0.92em;
}
.content-text :deep(pre code) {
  background: transparent;
  padding: 0;
  border-radius: 0;
}
.content-text :deep(.hljs) {
  background: transparent !important;
  padding: 0 !important;
}
.content-text :deep(table) {
  border-collapse: collapse;
  margin: 0.5em 0;
}
.content-text :deep(th),
.content-text :deep(td) {
  border: 1px solid var(--border-color);
  padding: 6px 12px;
  text-align: left;
}
.content-text :deep(th) {
  background: var(--hover-color);
  font-weight: 600;
}
.content-text :deep(img) {
  max-width: 100%;
  border-radius: 6px;
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

.message-card.has-comments {
  border-radius: 12px 12px 12px 12px;
}

.message-card.has-comments .card-accent-bar {
  border-radius: 12px 0 0 0;
}

.comment-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.comments-list {
  margin-bottom: 16px;
}

.comment-item {
  display: flex;
  gap: 10px;
  padding: 10px 0;
  border-bottom: 1px dashed var(--border-color);
}

.comment-item:last-child {
  border-bottom: none;
}

.comment-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.comment-content {
  flex: 1;
  min-width: 0;
}

.comment-author {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-color);
  margin-right: 8px;
}

.comment-text {
  font-size: 13px;
  color: var(--text-color);
  line-height: 1.5;
  display: block;
}

.comment-text :deep(.emoji-img) {
  width: 16px;
  height: 16px;
  vertical-align: middle;
  margin: 0 1px;
}

/* 评论 Markdown 轻量排版 */
.comment-text :deep(p) { margin: 0.3em 0; }
.comment-text :deep(a) { color: var(--primary-color); text-decoration: none; }
.comment-text :deep(a:hover) { text-decoration: underline; }
.comment-text :deep(strong) { font-weight: 600; }
.comment-text :deep(em) { font-style: italic; }
.comment-text :deep(code) {
  background: var(--hover-color);
  padding: 1px 4px;
  border-radius: 4px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 0.9em;
}

.comment-time {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 4px;
  display: block;
}

.comment-input-wrapper {
  background: var(--bg-color);
  border-radius: 8px;
  padding: 10px;
  border: 1px solid var(--border-color);
}

.comment-textarea {
  width: 100%;
  padding: 8px 12px;
  border: none;
  background: transparent;
  resize: none;
  font-size: 13px;
  color: var(--text-color);
  line-height: 1.5;
  box-sizing: border-box;
}

.comment-textarea:focus {
  outline: none;
}

.comment-textarea::placeholder {
  color: var(--text-secondary);
}

.comment-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
}

.comment-hint {
  font-size: 11px;
  color: var(--text-secondary);
}

.comment-submit-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background: var(--primary-color);
  color: white;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;
}

.comment-submit-btn:hover:not(:disabled) {
  background: var(--primary-dark);
}

.comment-submit-btn:disabled {
  background: var(--border-color);
  cursor: not-allowed;
}

.comment-submit-btn i {
  font-size: 14px;
}
</style>
