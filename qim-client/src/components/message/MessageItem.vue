<template>
  <div
    class="message-item"
    :class="{ self: isSelf, recalled: isRecalled, system: message.type === 'system' }"
    :data-message-id="message.id"
    @contextmenu.prevent="$emit('contextmenu', $event, message)"
  >
    <!-- 系统消息 -->
    <SystemMessage v-if="message.type === 'system'" :content="message.content" />

    <!-- 普通消息 -->
    <template v-else>
      <img
        :src="getAvatarUrl(message.sender)"
        :alt="message.sender.name || '未知用户'"
        class="message-avatar"
        @click="$emit('showUserProfile', message.sender)"
      />
      <div class="message-content">
        <div v-if="conversationType === 'group' && !isSelf" class="message-sender">{{ message.sender.name || '未知用户' }}</div>

        <!-- 撤回消息 -->
        <div v-if="isRecalled" class="message-bubble recalled-message">
          <span>{{ isSelf ? '你' : (message.sender.name || '未知用户') }} 撤回了一条消息</span>
        </div>

        <template v-else>
          <!-- 引用消息 -->
          <div v-if="message.quotedMessage" class="quoted-message-preview" @click="$emit('scrollToQuotedMessage', message.quotedMessage.id)">
            <div class="quoted-message-preview-header">
              <span>{{ message.quotedMessage.sender?.name || message.quotedMessage.name || '未知用户' }}</span>
            </div>
            <div class="quoted-message-preview-content">
              <template v-if="message.quotedMessage.type === 'text'">
                {{ message.quotedMessage.content || '无内容' }}
              </template>
              <template v-else-if="message.quotedMessage.type === 'image'">
                [图片] {{ getFileName(message.quotedMessage.content) }}
              </template>
              <template v-else-if="message.quotedMessage.type === 'file'">
                [文件] {{ getFileName(message.quotedMessage.content) }}
              </template>
              <template v-else-if="message.quotedMessage.type === 'mini-app' || message.quotedMessage.type === 'miniApp'">
                [小程序]
              </template>
              <template v-else-if="message.quotedMessage.type === 'share'">
                [分享]
              </template>
              <template v-else>
                <!-- 尝试检测内容是否为JSON格式的文件数据 -->
                <template v-if="isFileContent(message.quotedMessage.content)">
                  [文件]
                </template>
                <template v-else>
                  {{ message.quotedMessage.content || '无内容' }}
                </template>
              </template>
            </div>
          </div>

          <!-- 文本消息 -->
          <TextMessage v-if="message.type === 'text'" :content="message.content" :is-self="isSelf" />

          <!-- 图片消息 -->
          <ImageMessage
            v-else-if="message.type === 'image'"
            :src="message.content"
            :is-self="isSelf"
            :server-url="serverUrl"
            @preview="$emit('previewImage', message.content)"
          />

          <!-- 文件消息 -->
          <FileMessage
            v-else-if="message.type === 'file'"
            :content="message.content"
            :is-self="isSelf"
            :server-url="serverUrl"
            @download="$emit('downloadFile', message.content)"
            @saveAs="$emit('saveFileAs', message.content)"
          />

          <!-- 分享消息 -->
          <ShareMessage
            v-else-if="message.type === 'share'"
            :content="message.content"
            :share-data="message.shareData"
            :is-self="isSelf"
            @view="$emit('viewSharedContent', message.content)"
          />

          <!-- 小程序消息 -->
          <MiniAppMessage
            v-else-if="message.type === 'miniApp'"
            :mini-app-data="message.miniAppData"
            :is-self="isSelf"
            @open="$emit('openMiniApp', message.miniAppData)"
          />

          <!-- 资讯消息 -->
          <NewsMessage
            v-else-if="message.type === 'news'"
            :news-data="message.newsData"
            :is-self="isSelf"
            @open="$emit('openNewsLink', message.newsData?.url)"
          />
        </template>

        <div class="message-meta">
          <div class="message-time">{{ formatTime(message.timestamp) }}</div>
          <div v-if="isSelf && message.isFailed" class="message-read-status failed" title="发送失败">
            <i class="fas fa-exclamation-circle"></i> 发送失败
            <span class="retry-btn" @click.stop="$emit('retrySendMessage', message)"><i class="fas fa-redo"></i></span>
          </div>
          <div v-else-if="isSelf && conversationType === 'group' && !isRecalled" class="message-read-status clickable" :class="{ 'read': message.isRead }" @click="$emit('showReadUsers', message)">
            {{ message.isRead ? `${readUsersMap[message.id]?.read_users?.length || 0}人已读` : '未读' }}
          </div>
          <div v-else-if="isSelf && !isRecalled" class="message-read-status" :class="{ 'read': message.isRead }">
            {{ message.isRead ? '已读' : '未读' }}
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import TextMessage from './TextMessage.vue'
import ImageMessage from './ImageMessage.vue'
import FileMessage from './FileMessage.vue'
import ShareMessage from './ShareMessage.vue'
import MiniAppMessage from './MiniAppMessage.vue'
import NewsMessage from './NewsMessage.vue'
import SystemMessage from './SystemMessage.vue'

const props = defineProps<{
  message: any
  isSelf: boolean
  isRecalled: boolean
  conversationType: string
  readUsersMap: Record<string, { read_users: any[], total_members: number }>
  serverUrl: string
}>()

const emit = defineEmits<{
  contextmenu: [event: MouseEvent, message: any]
  showUserProfile: [user: any]
  scrollToQuotedMessage: [messageId: string]
  previewImage: [url: string]
  downloadFile: [url: string, fileName?: string]
  saveFileAs: [url: string, fileName?: string]
  viewSharedContent: [content: string]
  openMiniApp: [data: any]
  openNewsLink: [url: string]
  retrySendMessage: [message: any]
  showReadUsers: [message: any]
}>()

const getAvatarUrl = (sender: any): string => {
  if (sender.avatar && sender.avatar.startsWith('http')) {
    return sender.avatar
  } else if (sender.avatar) {
    return props.serverUrl + sender.avatar
  } else {
    return `https://api.dicebear.com/7.x/avataaars/svg?seed=${sender.name || 'user'}`
  }
}

// 格式化时间函数
function formatTime(timestamp: number | string | null | undefined): string {
  // 检查 timestamp 是否有效
  if (!timestamp || (typeof timestamp !== 'number' && typeof timestamp !== 'string')) {
    return '未知时间'
  }
  
  const date = new Date(timestamp)
  
  // 检查日期是否有效
  if (isNaN(date.getTime())) {
    return '未知时间'
  }
  
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const messageDate = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  const diffDays = Math.floor((today.getTime() - messageDate.getTime()) / (24 * 60 * 60 * 1000))
  
  if (diffDays === 0) {
    // 今天的消息，显示具体时间
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } else if (diffDays === 1) {
    // 昨天的消息，显示"昨天 时间"
    return `昨天 ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
  } else if (diffDays < 7) {
    // 本周的消息，显示星期几和时间
    const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
    const weekday = weekdays[date.getDay()]
    return `${weekday} ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
  } else {
    // 更早的消息，显示具体日期和时间
    return date.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  }
}

// 转换URL为链接的函数
const convertUrlsToLinks = (text: string): string => {
  // 正则表达式匹配URL
  const urlRegex = /(https?:\/\/[\w\-._~:/?#[\]@!$&'()*+,;=.]+)/g
  // 正则表达式匹配@用户
  const atRegex = /@([\u4e00-\u9fa5\w]+)/g
  
  let result = text
  
  // 先处理URL
  result = result.replace(urlRegex, (url) => {
    return `<a href="${url}" target="_blank" rel="noopener noreferrer" class="message-link">${url}</a>`
  })
  
  // 再处理@用户
  result = result.replace(atRegex, (match, username) => {
    return `<span class="at-user">@${username}</span>`
  })
  
  return result
}

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

// 检测内容是否为JSON格式的文件数据
const isFileContent = (content: string): boolean => {
  try {
    const contentObj = JSON.parse(content)
    return contentObj.url && (contentObj.name || contentObj.fileName) && (contentObj.size || contentObj.fileSize)
  } catch (e) {
    return false
  }
}
</script>

<style scoped>
.message-item {
  display: flex;
  align-items: flex-start;
  margin-bottom: 16px;
  animation: messageFadeIn 0.3s ease-out;
}

.message-item.self {
  flex-direction: row-reverse;
}

.message-item.system {
  justify-content: center;
  margin: 12px 0;
}

@keyframes messageFadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.message-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  cursor: pointer;
  transition: transform 0.2s;
  border: 1px solid var(--border-color);
}

.message-avatar:hover {
  transform: scale(1.1);
}

.message-content {
  max-width: 60%;
  min-width: 0;
  margin: 0 12px;
}

.message-sender {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 4px;
  margin-left: 4px;
}

.message-item.self .message-sender {
  display: none;
}

.quoted-message-preview {
  background: var(--hover-color);
  border-left: 4px solid var(--primary-color);
  padding: 10px;
  margin-bottom: 10px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 13px;
  line-height: 1.4;
}

.quoted-message-preview:hover {
  background: var(--hover-color);
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
  transform: translateY(-1px);
}

.quoted-message-preview-header {
  font-weight: 600;
  color: var(--text-color);
  margin-bottom: 5px;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.quoted-message-preview-content {
  color: var(--text-color);
  opacity: 0.9;
  line-height: 1.4;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  padding-left: 12px;
  position: relative;
}

.quoted-message-preview-content::before {
  content: '';
  position: absolute;
  left: 0;
  top: 2px;
  bottom: 2px;
  width: 2px;
  background: var(--primary-color);
  opacity: 0.3;
  border-radius: 1px;
}

.recalled-message {
  background: rgba(0, 0, 0, 0.06);
  color: var(--text-secondary);
  font-size: 12px;
  padding: 8px 12px;
  border-radius: 16px;
}

.message-meta {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-secondary);
}

.message-item.self .message-meta {
  justify-content: flex-end;
}

.message-time {
  font-size: 11px;
  color: var(--text-color);
  opacity: 0;
  transition: opacity 0.3s ease;
  flex: 0 0 auto;
}

.message-item:hover .message-time {
  opacity: 0.6;
}

.message-read-status {
  font-size: 10px;
  color: #999;
  opacity: 0.8;
}

.message-read-status.failed {
  color: #f56c6c;
  opacity: 1;
  display: flex;
  align-items: center;
  gap: 4px;
}

.message-read-status.failed .retry-btn {
  cursor: pointer;
  transition: all 0.2s;
}

.message-read-status.failed .retry-btn:hover {
  color: #f78989;
  transform: scale(1.1);
}

.message-read-status.clickable {
  cursor: pointer;
  transition: all 0.2s;
}

.message-read-status.clickable:hover {
  opacity: 0.8;
  transform: scale(1.05);
}

.message-read-status.read {
  color: #4caf50;
  opacity: 1;
}

/* 消息气泡样式 */
.message-bubble {
  padding: 10px 14px;
  border-radius: 12px;
  background: var(--sidebar-bg);
  color: var(--text-color);
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
  white-space: pre-wrap;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.message-item.self .message-bubble {
  background: var(--primary-color);
  color: white;
  border: none;
}

/* 为其他主题保留原来的样式 */
[data-theme="elegant-dark"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="elegant-dark"] .message-item.self .file-message {
  background: var(--primary-color);
  color: var(--secondary-color);
}

[data-theme="elegant-dark"] .message-item.self .recalled-message {
  background: rgba(255, 255, 255, 0.1) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

[data-theme="ocean-blue"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="ocean-blue"] .message-item.self .file-message {
  background: var(--primary-color);
  color: #fff;
}

[data-theme="ocean-blue"] .message-item.self .recalled-message {
  background: rgba(66, 153, 225, 0.8) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

[data-theme="elegant-purple"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="elegant-purple"] .message-item.self .file-message {
  background: var(--primary-color);
  color: #fff;
}

[data-theme="elegant-purple"] .message-item.self .recalled-message {
  background: rgba(139, 92, 246, 0.8) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

[data-theme="warm-amber"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="warm-amber"] .message-item.self .file-message {
  background: var(--primary-color);
  color: #fff;
}

[data-theme="warm-amber"] .message-item.self .recalled-message {
  background: rgba(217, 119, 6, 0.8) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

[data-theme="crimson-red"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="crimson-red"] .message-item.self .file-message {
  background: var(--primary-color);
  color: #fff;
}

[data-theme="crimson-red"] .message-item.self .recalled-message {
  background: rgba(220, 38, 38, 0.8) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

[data-theme="emerald-green"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="emerald-green"] .message-item.self .file-message {
  background: var(--primary-color);
  color: #fff;
}

[data-theme="emerald-green"] .message-item.self .recalled-message {
  background: rgba(16, 185, 129, 0.8) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

/* 失败消息样式 */
.message-item.self.failed .message-bubble {
  background-color: #f56c6c;
}

/* 撤回消息样式 */
.message-item.recalled .message-bubble {
  background: var(--sidebar-bg);
  color: #999;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  padding: 8px 16px;
}

.message-item.self.recalled .message-bubble {
  background: var(--sidebar-bg);
  color: #999;
}

.message-item.self .message-link {
  color: #e3f2fd;
}

.message-item.self .message-link:hover {
  color: white;
  text-decoration: underline;
}

/* 引用消息主题样式 */
.message-item.self .quoted-message-preview {
  background: rgba(59, 130, 246, 0.15);
  border-left-color: var(--primary-color);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.message-item.self .quoted-message-preview:hover {
  background: rgba(59, 130, 246, 0.2);
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.15);
  transform: translateY(-1px);
}

.message-item.self .quoted-message-preview-header,
.message-item.self .quoted-message-preview-content {
  color: var(--text-color);
}

.message-item.self .quoted-message-preview-content::before {
  background: var(--primary-color);
  opacity: 0.5;
}





/* 暗黑主题下的引用消息样式 */
[data-theme="elegant-dark"] .quoted-message-preview {
  background: rgba(255, 255, 255, 0.05) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3) !important;
}

[data-theme="elegant-dark"] .quoted-message-preview:hover {
  background: rgba(255, 255, 255, 0.1) !important;
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.4) !important;
}

[data-theme="elegant-dark"] .message-item.self .quoted-message-preview {
  background: rgba(59, 130, 246, 0.2) !important;
}

[data-theme="elegant-dark"] .message-item.self .quoted-message-preview:hover {
  background: rgba(59, 130, 246, 0.3) !important;
}

/* 消息链接样式 */
.message-link {
  color: #3b82f6;
  text-decoration: none;
  font-weight: 500;
  transition: all 0.3s ease;
}

.message-link:hover {
  color: #2563eb;
  text-decoration: underline;
  transform: translateY(-1px);
}

/* @用户样式 */
.at-user {
  color: #3b82f6;
  font-weight: 600;
  background-color: rgba(59, 130, 246, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
  transition: all 0.3s ease;
}

.at-user:hover {
  background-color: rgba(59, 130, 246, 0.2);
  transform: translateY(-1px);
}

/* 自己的消息中的@用户样式 */
.message-item.self .at-user {
  color: #e3f2fd;
  background-color: rgba(255, 255, 255, 0.1);
}

.message-item.self .at-user:hover {
  background-color: rgba(255, 255, 255, 0.2);
}
</style>