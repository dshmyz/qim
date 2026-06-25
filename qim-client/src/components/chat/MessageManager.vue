<template>
  <div v-if="visible" class="message-manager-modal" @click="$emit('close')">
    <div class="message-manager-content" @click.stop>
      <div class="message-manager-header">
        <div class="header-left">
          <div class="header-icon">
            <i class="fas fa-history"></i>
          </div>
          <h3>消息管理器</h3>
        </div>
        <button class="close-btn" @click="$emit('close')">
          <i class="fas fa-times"></i>
        </button>
      </div>
      <div class="message-manager-body">
        <!-- 搜索框 -->
        <div class="message-manager-search">
          <div class="search-input-wrapper">
            <i class="fas fa-search search-input-icon"></i>
            <input
              v-model="searchQuery"
              type="text"
              placeholder="搜索消息内容..."
              class="search-input"
              @keyup.enter="applyFilters"
            />
            <button v-if="searchQuery" class="search-clear" @click="searchQuery = ''; applyFilters()">
              <i class="fas fa-times"></i>
            </button>
          </div>
          <button class="search-btn" @click="applyFilters">
            <i class="fas fa-search"></i>
          </button>
        </div>
        
        <!-- 过滤选项 -->
        <div class="message-manager-filters">
          <div class="filter-group">
            <label>消息类型</label>
            <select v-model="selectedMessageType" class="filter-select" @change="applyFilters">
              <option value="all">全部</option>
              <option value="text">文本</option>
              <option value="image">图片</option>
              <option value="file">文件</option>
              <option value="miniApp">小程序</option>
              <option value="share">分享</option>
              <option value="news">资讯</option>
              <option value="markdown">Markdown</option>
            </select>
          </div>
          <div class="filter-group">
            <label>日期范围</label>
            <select v-model="selectedDateRange" class="filter-select" @change="applyFilters">
              <option value="all">全部</option>
              <option value="today">今天</option>
              <option value="week">本周</option>
              <option value="month">本月</option>
              <option value="custom">自定义</option>
            </select>
          </div>
          <div v-if="selectedDateRange === 'custom'" class="filter-group date-range-group">
            <label>自定义范围</label>
            <div class="date-range-inputs">
              <input
                type="date"
                v-model="customDateStart"
                class="date-input"
                @change="applyFilters"
              />
              <span class="date-range-separator">至</span>
              <input
                type="date"
                v-model="customDateEnd"
                class="date-input"
                @change="applyFilters"
              />
            </div>
          </div>
        </div>
        
        <!-- 消息列表 -->
        <div class="message-manager-list">
          <div v-if="isLoadingMessages" class="loading-message">
            加载中...
          </div>
          <div v-else-if="messages.length === 0" class="empty-message">
            暂无消息
          </div>
          <div v-else v-for="message in messages" :key="message.id" class="message-manager-item" :class="{ 'is-recalled': message.isRecalled }" @click="!message.isRecalled && handleMessageClick(message.id)">
            <div class="message-manager-item-header">
              <span class="message-sender">{{ message.sender?.name || '未知用户' }}</span>
              <span class="message-time">{{ formatTime(message.timestamp) }}</span>
              <span v-if="message.isRecalled" class="message-type message-type-recalled">
                <i class="fas fa-ban"></i> 已撤回
              </span>
              <span v-else class="message-type" :class="`message-type-${message.type}`">
                <i v-if="message.type === 'text'" class="fas fa-comment"></i>
                <i v-else-if="message.type === 'image'" class="fas fa-image"></i>
                <i v-else-if="message.type === 'file'" class="fas fa-file"></i>
                <i v-else-if="message.type === 'miniApp'" class="fas fa-th-large"></i>
                <i v-else-if="message.type === 'share'" class="fas fa-share-alt"></i>
                <i v-else-if="message.type === 'news'" class="fas fa-newspaper"></i>
                <i v-else-if="message.type === 'markdown'" class="fas fa-code"></i>
                {{ message.type === 'text' ? '文本' : message.type === 'image' ? '图片' : message.type === 'file' ? '文件' : message.type === 'miniApp' ? '小程序' : message.type === 'share' ? '分享' : message.type === 'news' ? '资讯' : message.type === 'markdown' ? 'Markdown' : '其他' }}
              </span>
            </div>
            <div class="message-manager-item-content" :class="[`message-content-${message.type}`, { 'is-recalled': message.isRecalled }]">
              <template v-if="message.isRecalled">
                <span class="recalled-text">此消息已被撤回</span>
              </template>
              <template v-else-if="message.type === 'text'">
                <template v-for="(seg, i) in parseTextSegments(message.content)" :key="i">
                  <span
                    v-if="seg.type === 'mention'"
                    class="at-mention-chip"
                    :class="{ 'at-mention-chip--all': seg.userId === 'all' }"
                  >{{ seg.text }}</span>
                  <span v-else>{{ seg.text }}</span>
                </template>
              </template>
              <template v-else-if="message.type === 'image'">
                <div class="message-file-link" @click.stop="handleMediaClick(message, $event)">
                  <i class="fas fa-image"></i>
                  <span>{{ getFileName(message) }}</span>
                </div>
              </template>
              <template v-else-if="message.type === 'file'">
                <div class="message-file-link" @click.stop="handleMediaClick(message, $event)">
                  <i class="fas fa-file"></i>
                  <span>{{ getFileName(message) }}</span>
                </div>
              </template>
              <template v-else-if="message.type === 'miniApp'">
                <div class="mini-app-info">
                  <div class="mini-app-icon">
                    <i class="fas fa-th-large"></i>
                  </div>
                  <div class="mini-app-details">
                    <div class="mini-app-name">{{ message.title || '小程序' }}</div>
                    <div class="mini-app-description">{{ message.description || '点击打开小程序' }}</div>
                  </div>
                </div>
              </template>
              <template v-else-if="message.type === 'share'">
                <div class="share-info">
                  <div class="share-icon">
                    <i class="fas fa-share-alt"></i>
                  </div>
                  <div class="share-details">
                    <div class="share-title">{{ message.title || '分享内容' }}</div>
                    <div class="share-description">{{ message.description || '点击查看分享' }}</div>
                  </div>
                </div>
              </template>
              <template v-else-if="message.type === 'news'">
                <div class="news-info">
                  <div class="news-icon">
                    <i class="fas fa-newspaper"></i>
                  </div>
                  <div class="news-details">
                    <div class="news-title">{{ message.title || '资讯' }}</div>
                    <div class="news-description">{{ message.description || '点击查看资讯' }}</div>
                  </div>
                </div>
              </template>
            </div>
          </div>
        </div>
        
        <!-- 分页 -->
        <div v-if="total > 0" class="message-manager-pagination">
          <span class="pagination-info">
            第 {{ currentPage }} / {{ totalPages }} 页，共 {{ total }} 条，显示 {{ currentPageCount.start }}-{{ currentPageCount.end }} 条
          </span>
          <div class="pagination-controls">
            <button 
              class="pagination-btn" 
              :disabled="currentPage === 1" 
              @click="changePage(currentPage - 1)"
            >
              上一页
            </button>
            <div class="page-jump">
              <input
                v-model.number="jumpToPage"
                type="number"
                class="page-input"
                :min="1"
                :max="totalPages"
                @keyup.enter="handleJump"
              />
              <button class="jump-btn" @click="handleJump">
                跳转
              </button>
            </div>
            <button 
              class="pagination-btn" 
              :disabled="currentPage >= totalPages" 
              @click="changePage(currentPage + 1)"
            >
              下一页
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 图片预览 -->
    <div v-if="showImagePreview" class="image-preview-modal" @click="closeImagePreview">
      <div class="image-preview-content" @click.stop>
        <button class="image-preview-close" @click="closeImagePreview">×</button>
        <img :src="previewImageUrl" alt="预览图片" class="image-preview-img" />
        <div class="image-preview-actions">
          <button class="image-preview-download" @click="downloadImage">
            <i class="fas fa-download"></i> 下载图片
          </button>
        </div>
      </div>
    </div>

    <!-- 媒体操作菜单 -->
    <Teleport to="body">
      <div v-if="mediaMenuVisible" class="media-action-overlay" @click="closeMediaMenu">
        <div
          class="media-action-menu"
          :style="{ left: mediaMenuPosition.x + 'px', top: mediaMenuPosition.y + 'px' }"
          @click.stop
        >
          <div class="media-menu-item" @click="handleJumpFromMenu">
            <i class="fas fa-chevron-right"></i>
            <span>跳转</span>
          </div>
          <div class="media-menu-item" @click="handleDownloadFromMenu">
            <i class="fas fa-download"></i>
            <span>下载</span>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import QMessage from '../../utils/qmessage'
import QMessageBox from '../../utils/qmessagebox'
import { messageApi } from '../../api/message'
import { getStoredServerUrl } from '../../composables/useServerUrl'
import { parseContent } from '../../utils/mentions'

const props = defineProps<{
  visible: boolean
  conversationId: string
}>()

const emit = defineEmits<{
  close: []
  scrollToMessage: [messageId: string]
}>()

// 消息管理器相关
const searchQuery = ref('')
const selectedMessageType = ref('all')
const selectedDateRange = ref('all')
const customDateStart = ref('')
const customDateEnd = ref('')
const messages = ref<any[]>([])
const isLoadingMessages = ref(false)
const currentPage = ref(1)
const total = ref(0)
const pageSize = 20
const jumpToPage = ref(1)
const showImagePreview = ref(false)
const previewImageUrl = ref('')

const totalPages = computed(() => {
  return Math.ceil(total.value / pageSize)
})

const currentPageCount = computed(() => {
  const start = (currentPage.value - 1) * pageSize + 1
  const end = Math.min(currentPage.value * pageSize, total.value)
  return { start, end }
})

// 监听 visible 属性变化
watch(() => props.visible, (newVal) => {
  if (newVal && props.conversationId) {
    loadMessages()
  }
})

// 监听 conversationId 变化
watch(() => props.conversationId, (newVal) => {
  if (newVal && props.visible) {
    loadMessages()
  }
})

// 加载消息
const loadMessages = async (page: number = 1) => {
  if (!props.conversationId) return

  isLoadingMessages.value = true
  currentPage.value = page

  try {
    const params: Record<string, string> = {
      conversation_id: props.conversationId,
      page: page.toString(),
      page_size: pageSize.toString(),
    }
    if (selectedMessageType.value !== 'all') {
      params.type = selectedMessageType.value
    }
    if (searchQuery.value) {
      params.search = searchQuery.value
    }

    if (selectedDateRange.value !== 'all') {
      const now = new Date()
      const year = now.getFullYear()
      const month = String(now.getMonth() + 1).padStart(2, '0')
      const day = String(now.getDate()).padStart(2, '0')
      const todayStr = `${year}-${month}-${day}`
      let startDate = ''
      let endDate = ''

      if (selectedDateRange.value === 'today') {
        startDate = todayStr
        endDate = todayStr
      } else if (selectedDateRange.value === 'week') {
        const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
        startDate = `${weekAgo.getFullYear()}-${String(weekAgo.getMonth() + 1).padStart(2, '0')}-${String(weekAgo.getDate()).padStart(2, '0')}`
        endDate = todayStr
      } else if (selectedDateRange.value === 'month') {
        const monthAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
        startDate = `${monthAgo.getFullYear()}-${String(monthAgo.getMonth() + 1).padStart(2, '0')}-${String(monthAgo.getDate()).padStart(2, '0')}`
        endDate = todayStr
      } else if (selectedDateRange.value === 'custom' && customDateStart.value && customDateEnd.value) {
        startDate = customDateStart.value
        endDate = customDateEnd.value
      }

      if (startDate) params.start_date = startDate
      if (endDate) params.end_date = endDate
    }

    const result = await messageApi.getMessagesByFilter(params)
    const rawMessages = result.messages || []
    messages.value = rawMessages.map((message: any) => ({
      ...message,
      timestamp: message.created_at ? new Date(message.created_at).getTime() : Date.now(),
      isRecalled: message.is_recalled || false,
      sender: message.sender ? {
        ...message.sender,
        name: message.sender.name || message.sender.nickname || message.sender.username || '未知用户'
      } : null
    }))
    messages.value.sort((a, b) => b.timestamp - a.timestamp)
    total.value = result.total
  } catch (error) {
    if (error instanceof Error) {
      QMessage.error(error.message || '加载消息失败，请稍后重试')
    }
  } finally {
    isLoadingMessages.value = false
  }
}

// 应用过滤器
const applyFilters = () => {
  loadMessages(1)
}

// 改变页码
const changePage = (page: number) => {
  loadMessages(page)
}

// 跳转到指定页面
const handleJump = () => {
  const page = jumpToPage.value
  if (page >= 1 && page <= totalPages.value) {
    changePage(page)
  } else {
    jumpToPage.value = currentPage.value
  }
}

// 处理消息点击，跳转到聊天窗口中的对应消息
const handleMessageClick = (messageId: string) => {
  emit('scrollToMessage', messageId)
}

// 下载文件
const downloadFile = (message: any) => {
  const serverUrl = getStoredServerUrl()
  let fileUrl = message.content
  try {
    // 尝试解析content为JSON
    const contentObj = JSON.parse(message.content)
    if (contentObj.url) {
      fileUrl = contentObj.url
      // 确保文件URL包含服务器地址
      if (fileUrl && !fileUrl.startsWith('http')) {
        fileUrl = serverUrl + fileUrl
      }
    }
  } catch (e) {
    // 解析失败，直接使用content
    // 确保文件URL包含服务器地址
    if (fileUrl && !fileUrl.startsWith('http')) {
      fileUrl = serverUrl + fileUrl
    }
  }
  const link = document.createElement('a')
  link.href = fileUrl
  link.download = getFileName(message)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

// 预览图片
const previewImage = (message: any) => {
  const serverUrl = getStoredServerUrl()
  try {
    // 尝试解析content为JSON
    const contentObj = JSON.parse(message.content)
    if (contentObj.url) {
      let imageUrl = contentObj.url
      // 确保图片URL包含服务器地址
      if (imageUrl && !imageUrl.startsWith('http')) {
        imageUrl = serverUrl + imageUrl
      }
      previewImageUrl.value = imageUrl
    } else {
      let imageUrl = message.content
      // 确保图片URL包含服务器地址
      if (imageUrl && !imageUrl.startsWith('http')) {
        imageUrl = serverUrl + imageUrl
      }
      previewImageUrl.value = imageUrl
    }
  } catch (e) {
    // 解析失败，直接使用content
    let imageUrl = message.content
    // 确保图片URL包含服务器地址
    if (imageUrl && !imageUrl.startsWith('http')) {
      imageUrl = serverUrl + imageUrl
    }
    previewImageUrl.value = imageUrl
  }
  showImagePreview.value = true
}

// 关闭图片预览
const closeImagePreview = () => {
  showImagePreview.value = false
  previewImageUrl.value = ''
}

// 处理媒体文件点击
const mediaMenuVisible = ref(false)
const mediaMenuPosition = ref({ x: 0, y: 0 })
const currentMediaMessage = ref<any>(null)

const handleMediaClick = (message: any, event: MouseEvent) => {
  event.stopPropagation()
  currentMediaMessage.value = message
  mediaMenuPosition.value = { x: event.clientX, y: event.clientY }
  mediaMenuVisible.value = true
}

const closeMediaMenu = () => {
  mediaMenuVisible.value = false
  currentMediaMessage.value = null
}

const handleJumpFromMenu = () => {
  if (currentMediaMessage.value) {
    handleMessageClick(currentMediaMessage.value.id)
  }
  closeMediaMenu()
}

const handleDownloadFromMenu = () => {
  if (currentMediaMessage.value) {
    if (currentMediaMessage.value.type === 'image') {
      previewImage(currentMediaMessage.value)
    } else {
      downloadFile(currentMediaMessage.value)
    }
  }
  closeMediaMenu()
}

// 下载图片
const downloadImage = async () => {
  if (!previewImageUrl.value) return
  
  try {
    const response = await fetch(previewImageUrl.value)
    if (response.ok) {
      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'image.png'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(url)
      QMessage.success('图片下载成功')
    } else {
      QMessage.error('图片下载失败')
    }
  } catch (error) {
    console.error('图片下载失败:', error)
    QMessage.error('图片下载失败')
  }
}

// 获取文件名
const getFileName = (message: any): string => {
  try {
    // 尝试解析content为JSON
    const contentObj = JSON.parse(message.content)
    if (contentObj.name) {
      return contentObj.name
    } else if (contentObj.fileName) {
      return contentObj.fileName
    }
  } catch (e) {
    // 解析失败，从content字符串中提取文件名
  }
  return message.content.split('/').pop() || '文件'
}

// 解析文本消息 content 为片段（文本 + mention），用于正确渲染 @ 提及
type TextSegment =
  | { type: 'text'; text: string }
  | { type: 'mention'; text: string; userId: number | 'all' }

const parseTextSegments = (content: string): TextSegment[] => {
  const { text, mentions } = parseContent(content)
  if (mentions.length === 0) {
    return [{ type: 'text', text }]
  }
  const result: TextSegment[] = []
  let lastEnd = 0
  for (const m of mentions) {
    if (m.start > lastEnd) {
      result.push({ type: 'text', text: text.slice(lastEnd, m.start) })
    }
    result.push({ type: 'mention', text: m.text, userId: m.userId })
    lastEnd = m.end
  }
  if (lastEnd < text.length) {
    result.push({ type: 'text', text: text.slice(lastEnd) })
  }
  return result
}

// 格式化时间
const formatTime = (timestamp: number): string => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) {
    return '刚刚'
  } else if (diff < 3600000) {
    return `${Math.floor(diff / 60000)}分钟前`
  } else if (diff < 86400000) {
    return `${Math.floor(diff / 3600000)}小时前`
  } else if (diff < 604800000) {
    return `${Math.floor(diff / 86400000)}天前`
  } else {
    return date.toLocaleDateString('zh-CN')
  }
}

// 组件挂载时加载消息
onMounted(() => {
  if (props.visible && props.conversationId) {
    loadMessages()
  }
})
</script>

<style>
/* 媒体操作菜单 — 全局样式（Teleport 到 body） */
.media-action-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
}
.media-action-menu {
  position: fixed;
  z-index: 10000;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.12);
  padding: 6px;
  min-width: 140px;
}
.media-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  color: #1a1a2e;
  transition: background 0.1s;
}
.media-menu-item:hover {
  background: #f3f4f6;
}
.media-menu-item i {
  width: 16px;
  text-align: center;
  color: #9ca3af;
}
</style>
<style scoped>
.message-manager-modal {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.message-manager-content {
  background: var(--card-bg, #fff);
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(0, 0, 0, 0.05);
  width: 820px;
  max-width: 92vw;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 头部 */
.message-manager-header {
  padding: 16px 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--secondary-color, #f8f9fb);
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #3385ff, #6366f1);
  border-radius: 10px;
  color: #fff;
  font-size: 16px;
}

.header-left h3 {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  color: var(--text-color, #1a1a2e);
}

.close-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 10px;
  color: var(--text-secondary, #9ca3af);
  cursor: pointer;
  font-size: 16px;
  transition: all 0.15s;
}

.close-btn:hover {
  background: var(--hover-color, #f3f4f6);
  border-color: var(--border-color, #e5e7eb);
  color: var(--text-color, #1a1a2e);
}

/* Body */
.message-manager-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 搜索栏 */
.message-manager-search {
  display: flex;
  gap: 10px;
  align-items: center;
}

.search-input-wrapper {
  flex: 1;
  position: relative;
  display: flex;
  align-items: center;
}

.search-input-icon {
  position: absolute;
  left: 14px;
  color: var(--text-secondary, #9ca3af);
  font-size: 14px;
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 10px 36px 10px 40px;
  border: 1.5px solid var(--border-color, #e5e7eb);
  border-radius: 10px;
  background: var(--input-bg, #f9fafb);
  color: var(--text-color, #1a1a2e);
  font-size: 14px;
  transition: all 0.2s;
}

.search-input:focus {
  outline: none;
  border-color: #3385ff;
  box-shadow: 0 0 0 3px rgba(51, 133, 255, 0.08);
  background: #fff;
}

.search-clear {
  position: absolute;
  right: 8px;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 50%;
  color: var(--text-secondary, #9ca3af);
  cursor: pointer;
  font-size: 12px;
}

.search-clear:hover {
  background: var(--hover-color, #f3f4f6);
}

.search-btn {
  padding: 10px 18px;
  border: none;
  border-radius: 10px;
  background: linear-gradient(135deg, #3385ff, #4f46e5);
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}

.search-btn:hover {
  box-shadow: 0 4px 12px rgba(51, 133, 255, 0.35);
  transform: translateY(-1px);
}

/* 过滤器 */
.message-manager-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  align-items: flex-end;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 110px;
}

.filter-group label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary, #9ca3af);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.filter-select {
  padding: 7px 12px;
  border: 1.5px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  background: var(--input-bg, #f9fafb);
  color: var(--text-color, #1a1a2e);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%239ca3af' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 10px center;
  padding-right: 30px;
}

.filter-select:focus {
  outline: none;
  border-color: #3385ff;
  box-shadow: 0 0 0 3px rgba(51, 133, 255, 0.08);
}

.date-range-inputs {
  display: flex;
  align-items: center;
  gap: 8px;
}

.date-input {
  padding: 7px 10px;
  border: 1.5px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  background: var(--input-bg, #f9fafb);
  color: var(--text-color, #1a1a2e);
  font-size: 13px;
  flex: 1;
}

.date-input:focus {
  outline: none;
  border-color: #3385ff;
}

.date-range-separator {
  font-size: 13px;
  color: var(--text-secondary, #9ca3af);
}

/* 消息列表 */
.message-manager-list {
  flex: 1;
  min-height: 200px;
  max-height: 420px;
  overflow-y: auto;
  background: var(--card-bg, #fff);
  border-radius: 12px;
  border: 1px solid var(--border-color, #e5e7eb);
}

.loading-message,
.empty-message {
  padding: 48px 20px;
  text-align: center;
  color: var(--text-secondary, #9ca3af);
  font-size: 14px;
}

.message-manager-item {
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-color, #f0f0f0);
  transition: background 0.15s;
}

.message-manager-item:last-child {
  border-bottom: none;
}

.message-manager-item:hover {
  background: linear-gradient(135deg, #f8faff, #f0f5ff);
}

.message-manager-item.is-recalled {
  cursor: default;
  opacity: 0.5;
}

.message-manager-item.is-recalled:hover {
  background: transparent;
}

.message-manager-item-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}

.message-sender {
  font-weight: 600;
  color: var(--text-color, #1a1a2e);
  font-size: 13px;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.message-time {
  font-size: 11px;
  color: var(--text-secondary, #9ca3af);
  flex-shrink: 0;
}

.message-type {
  font-size: 11px;
  font-weight: 500;
  padding: 3px 8px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  background: #f0f5ff;
  color: #3385ff;
}

.message-type i { font-size: 10px; }

.message-type-recalled {
  background: #f3f4f6;
  color: #9ca3af;
}

.message-manager-item-content {
  font-size: 13px;
  color: var(--text-secondary, #6b7280);
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  padding-left: 0;
}

.message-manager-item-content.is-recalled {
  font-style: italic;
  color: #9ca3af;
}

/* @ 提及 chip 样式 */
.message-manager-item-content .at-mention-chip {
  color: #2563eb;
  font-weight: 600;
  padding: 1px 4px;
  border-radius: 4px;
}

.message-manager-item-content .at-mention-chip--all {
  color: #d97706;
  background: rgba(245, 158, 11, 0.18);
}

.message-file-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: #f3f4f6;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}

.message-file-link:hover {
  background: #e5e7eb;
}

.message-file-link i {
  color: #3385ff;
  font-size: 14px;
}

.mini-app-info, .share-info, .news-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.mini-app-icon, .share-icon, .news-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: #f0f5ff;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #3385ff;
  font-size: 16px;
  flex-shrink: 0;
}

.mini-app-name, .share-title, .news-title {
  font-weight: 500;
  color: var(--text-color, #1a1a2e);
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-app-description, .share-description, .news-description {
  font-size: 12px;
  color: var(--text-secondary, #9ca3af);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 分页 */
.message-manager-pagination {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 0 0;
  border-top: 1px solid var(--border-color, #e5e7eb);
  flex-shrink: 0;
}

.pagination-info {
  font-size: 13px;
  color: var(--text-secondary, #9ca3af);
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.page-jump {
  display: flex;
  align-items: center;
  gap: 4px;
}

.page-input {
  width: 48px;
  padding: 6px;
  border: 1.5px solid var(--border-color, #e5e7eb);
  border-radius: 6px;
  background: var(--input-bg, #f9fafb);
  color: var(--text-color, #1a1a2e);
  font-size: 13px;
  text-align: center;
}

.page-input:focus {
  outline: none;
  border-color: #3385ff;
}

.pagination-btn {
  padding: 7px 14px;
  border: 1.5px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  font-size: 13px;
  background: #fff;
  color: var(--text-color, #1a1a2e);
  cursor: pointer;
  transition: all 0.15s;
  font-weight: 500;
}

.pagination-btn:hover:not(:disabled) {
  border-color: #3385ff;
  color: #3385ff;
  background: #f8faff;
}

.pagination-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.jump-btn {
  padding: 7px 12px;
  border: none;
  border-radius: 8px;
  background: #3385ff;
  color: #fff;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.jump-btn:hover {
  background: #2563eb;
}

/* 图片预览 */
.image-preview-modal {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.92);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.image-preview-content {
  position: relative;
  max-width: 90vw;
  max-height: 90vh;
}

.image-preview-close {
  position: absolute;
  top: -44px;
  right: 0;
  background: rgba(255,255,255,0.1);
  border: none;
  color: #fff;
  font-size: 20px;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  cursor: pointer;
  transition: background 0.15s;
}

.image-preview-close:hover {
  background: rgba(255,255,255,0.2);
}

.image-preview-img {
  max-width: 100%;
  max-height: 80vh;
  object-fit: contain;
  border-radius: 12px;
}

.image-preview-actions {
  margin-top: 16px;
  display: flex;
  justify-content: center;
}

.image-preview-download {
  padding: 10px 20px;
  border: none;
  border-radius: 10px;
  background: linear-gradient(135deg, #3385ff, #4f46e5);
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: all 0.2s;
}

.image-preview-download:hover {
  box-shadow: 0 4px 16px rgba(51, 133, 255, 0.4);
  transform: translateY(-1px);
}

/* 响应式 */
@media (max-width: 768px) {
  .message-manager-content {
    width: 95vw;
    max-height: 92vh;
    border-radius: 12px;
  }

  .message-manager-filters {
    flex-direction: column;
    gap: 10px;
  }

  .filter-group {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }

  .message-manager-pagination {
    flex-direction: column;
    gap: 10px;
    align-items: stretch;
  }

  .pagination-controls {
    justify-content: center;
  }
}
</style>