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
              <option value="merged_forward">聊天记录</option>
              <option value="video">视频</option>
              <option value="audio">语音</option>
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
          <div
            v-else
            v-for="message in messages"
            :key="message.id"
            class="message-manager-item"
            :class="{ 'is-recalled': message.isRecalled }"
            @click="handleMessageClick(message)"
          >
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
                <i v-else-if="message.type === 'merged_forward'" class="fas fa-comments"></i>
                <i v-else-if="message.type === 'video'" class="fas fa-video"></i>
                <i v-else-if="message.type === 'audio'" class="fas fa-microphone"></i>
                {{ messageTypeLabel(message) }}
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
                  <TaskRefCard
                    v-else-if="seg.type === 'task_ref'"
                    :task-id="seg.taskId"
                    :conversation-id="seg.conversationId"
                  />
                  <template v-else>
                    <template v-for="(part, partIndex) in splitHighlightParts(seg.text)" :key="partIndex">
                      <mark v-if="part.highlight" class="search-highlight" v-html="previewTextToHtml(part.text)"></mark>
                      <span v-else v-html="previewTextToHtml(part.text)"></span>
                    </template>
                  </template>
                </template>
              </template>
              <template v-else-if="message.type === 'image'">
                <div
                  class="message-file-link"
                  @click.stop="previewImage(message)"
                  @contextmenu.prevent.stop="handleMediaClick(message, $event)"
                >
                  <i class="fas fa-image"></i>
                  <span>{{ getFileName(message) }}</span>
                </div>
              </template>
              <template v-else-if="message.type === 'file'">
                <div
                  class="message-file-link"
                  @click.stop="handleMediaClick(message, $event)"
                  @contextmenu.prevent.stop="handleMediaClick(message, $event)"
                >
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
                    <div class="mini-app-name">{{ getMessageDisplay(message).title }}</div>
                    <div class="mini-app-description">{{ getMessageDisplay(message).description }}</div>
                  </div>
                </div>
              </template>
              <template v-else-if="message.type === 'share'">
                <div class="share-info">
                  <div class="share-icon">
                    <i class="fas fa-share-alt"></i>
                  </div>
                  <div class="share-details">
                    <div class="share-title">{{ getMessageDisplay(message).title }}</div>
                    <div class="share-description">{{ getMessageDisplay(message).description }}</div>
                  </div>
                </div>
              </template>
              <template v-else-if="message.type === 'news'">
                <div class="news-info">
                  <div class="news-icon">
                    <i class="fas fa-newspaper"></i>
                  </div>
                  <div class="news-details">
                    <div class="news-title">{{ getMessageDisplay(message).title }}</div>
                    <div class="news-description">{{ getMessageDisplay(message).description }}</div>
                  </div>
                </div>
              </template>
              <template v-else-if="message.type === 'markdown'">
                <MarkdownMessage :content="message.content" />
              </template>
              <template v-else-if="message.type === 'merged_forward'">
                <MergedForwardMessage :content="message.content" />
              </template>
              <template v-else-if="message.type === 'video' || message.type === 'audio' || message.type === 'system' || message.type === 'streaming'">
                <div class="message-structured-summary">{{ getMessageDisplay(message).summary }}</div>
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
    <UniversalContextMenu menuId="media" :items="mediaMenuItems" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import QMessage from '../../utils/qmessage'
import QMessageBox from '../../utils/qmessagebox'
import { messageApi } from '../../api/message'
import { getStoredServerUrl } from '../../composables/useServerUrl'
import { getToken } from '../../composables/useRequest'
import { parseContent } from '../../utils/mentions'
import { downloadUrl } from '../../utils/download'
import { resolveMessageDisplay } from '../../utils/messageDisplay'
import { previewTextToHtml } from '../../utils/emoji'
import MarkdownMessage from '../message/MarkdownMessage.vue'
import MergedForwardMessage from '../message/MergedForwardMessage.vue'
import TaskRefCard from '../message/TaskRefCard.vue'
import UniversalContextMenu from '../shared/UniversalContextMenu.vue'
import { openMenu, closeMenu } from '../../composables/useUI'

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
const previewImageFilename = ref('image.png')

const getMessageDisplay = (message: any) => resolveMessageDisplay(message)

const messageTypeLabel = (message: any): string => ({
  text: '文本',
  image: '图片',
  file: '文件',
  miniApp: '小程序',
  share: '分享',
  news: '资讯',
  video: '视频',
  audio: '语音',
  system: '系统',
  streaming: '流式消息',
  mergedForward: '聊天记录',
  unknown: '其他',
}[getMessageDisplay(message).kind] ?? '其他')

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
  } else if (!newVal) {
    searchQuery.value = ''
    selectedMessageType.value = 'all'
    selectedDateRange.value = 'all'
    customDateStart.value = ''
    customDateEnd.value = ''
    messages.value = []
    total.value = 0
    currentPage.value = 1
    jumpToPage.value = 1
    isLoadingMessages.value = false
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
  jumpToPage.value = page

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
    total.value = result.total
  } catch (error: any) {
    const msg = error?.response?.data?.message
      || error?.message
      || '加载消息失败，请稍后重试'
    QMessage.error(msg)
    messages.value = []
    total.value = 0
  } finally {
    isLoadingMessages.value = false
  }
}

// 应用过滤器
const applyFilters = () => {
  if (selectedDateRange.value === 'custom') {
    const start = customDateStart.value
    const end = customDateEnd.value
    if (!start || !end) {
      QMessage.warning('请选择完整的日期范围')
      return
    }
    if (start > end) {
      QMessage.warning('开始日期不能晚于结束日期')
      return
    }
  }
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
const handleMessageClick = (message: any) => {
  if (message.isRecalled || window.getSelection()?.toString().trim()) {
    return
  }
  emit('scrollToMessage', message.id)
}

const resolveMediaUrl = (message: any): string => {
  const serverUrl = getStoredServerUrl()
  let mediaUrl = message.content
  try {
    const contentObj = JSON.parse(message.content)
    if (contentObj.url) {
      mediaUrl = contentObj.url
    }
  } catch (e) {
    mediaUrl = message.content
  }
  if (mediaUrl && !mediaUrl.startsWith('http')) {
    return `${serverUrl.replace(/\/$/, '')}/${mediaUrl.replace(/^\//, '')}`
  }
  return mediaUrl
}

const downloadMedia = async (url: string, filename: string, successMessage: string, errorMessage: string) => {
  if (!url) {
    QMessage.error(errorMessage)
    return
  }
  try {
    await downloadUrl({
      url,
      filename,
      token: getToken(),
    })
    QMessage.success(successMessage)
  } catch (error) {
    console.error(errorMessage, error)
    QMessage.error(errorMessage)
  }
}

// 下载文件
const downloadFile = async (message: any) => {
  await downloadMedia(resolveMediaUrl(message), getFileName(message), '文件下载已开始', '文件下载失败')
}

// 预览图片
const previewImage = (message: any) => {
  previewImageUrl.value = resolveMediaUrl(message)
  previewImageFilename.value = getFileName(message) || 'image.png'
  showImagePreview.value = true
}

// 关闭图片预览
const closeImagePreview = () => {
  showImagePreview.value = false
  previewImageUrl.value = ''
  previewImageFilename.value = 'image.png'
}

// 处理媒体文件点击
const currentMediaMessage = ref<any>(null)

const mediaMenuItems = computed(() => [
  { label: '跳转', icon: 'fas fa-chevron-right', action: () => { emit('scrollToMessage', currentMediaMessage.value?.id); closeMenu() } },
  { label: '下载', icon: 'fas fa-download', action: () => { if (currentMediaMessage.value) downloadFile(currentMediaMessage.value); closeMenu() } }
])

const handleMediaClick = (message: any, event: MouseEvent) => {
  event.stopPropagation()
  currentMediaMessage.value = message
  openMenu('media', event.clientX, event.clientY)
}

// 下载图片
const downloadImage = async () => {
  if (!previewImageUrl.value) return
  await downloadMedia(previewImageUrl.value, previewImageFilename.value, '图片下载已开始', '图片下载失败')
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

// 解析文本消息 content 为片段（文本 + mention + task_ref），用于正确渲染 @ 提及和任务卡片
type TextSegment =
  | { type: 'text'; text: string }
  | { type: 'mention'; text: string; userId: number | 'all' }
  | { type: 'task_ref'; taskId: number; conversationId: number }

// 任务引用正则：#T-123（与 TextMessage.vue 保持一致）
const taskRefRegex = /#T-(\d+)\b/g

// 数值化 conversationId（无效时返回 0，0 表示不渲染任务卡片，降级为纯文本）
const numericConvId = computed((): number => {
  const v = props.conversationId
  if (!v) return 0
  const n = Number(v)
  return Number.isFinite(n) && n > 0 ? n : 0
})

type HighlightPart = {
  text: string
  highlight: boolean
}

const parseTextSegments = (content: string): TextSegment[] => {
  const { text, mentions } = parseContent(content)
  // 先按 mention 拆分，得到 text 段和 mention 段
  const rawSegments: Array<{ type: 'text'; text: string } | { type: 'mention'; text: string; userId: number | 'all' }> = []
  if (mentions.length === 0) {
    rawSegments.push({ type: 'text', text })
  } else {
    let lastEnd = 0
    for (const m of mentions) {
      if (m.start > lastEnd) {
        rawSegments.push({ type: 'text', text: text.slice(lastEnd, m.start) })
      }
      rawSegments.push({ type: 'mention', text: m.text, userId: m.userId })
      lastEnd = m.end
    }
    if (lastEnd < text.length) {
      rawSegments.push({ type: 'text', text: text.slice(lastEnd) })
    }
  }

  // 对 text 段再拆 task_ref（conversationId 无效时降级为纯文本，后端无法校验权限）
  const convId = numericConvId.value
  const result: TextSegment[] = []
  for (const seg of rawSegments) {
    if (seg.type !== 'text') {
      result.push(seg)
      continue
    }
    if (convId === 0) {
      result.push({ type: 'text', text: seg.text })
      continue
    }
    taskRefRegex.lastIndex = 0
    let lastIdx = 0
    let m: RegExpExecArray | null
    while ((m = taskRefRegex.exec(seg.text)) !== null) {
      if (m.index > lastIdx) {
        result.push({ type: 'text', text: seg.text.slice(lastIdx, m.index) })
      }
      result.push({ type: 'task_ref', taskId: Number(m[1]), conversationId: convId })
      lastIdx = m.index + m[0].length
    }
    if (lastIdx < seg.text.length) {
      result.push({ type: 'text', text: seg.text.slice(lastIdx) })
    }
  }
  return result
}

const splitHighlightParts = (text: string): HighlightPart[] => {
  const keyword = searchQuery.value.trim()
  if (!keyword) {
    return [{ text, highlight: false }]
  }

  const lowerText = text.toLowerCase()
  const lowerKeyword = keyword.toLowerCase()
  const parts: HighlightPart[] = []
  let cursor = 0
  let index = lowerText.indexOf(lowerKeyword)

  while (index !== -1) {
    if (index > cursor) {
      parts.push({ text: text.slice(cursor, index), highlight: false })
    }
    const end = index + keyword.length
    parts.push({ text: text.slice(index, end), highlight: true })
    cursor = end
    index = lowerText.indexOf(lowerKeyword, cursor)
  }

  if (cursor < text.length) {
    parts.push({ text: text.slice(cursor), highlight: false })
  }

  return parts.length > 0 ? parts : [{ text, highlight: false }]
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
  user-select: text;
}

.message-manager-item-content :deep(.emoji-img) {
  width: 18px;
  height: 18px;
  vertical-align: middle;
  margin: 0 1px;
}

.message-content-file {
  display: block;
  overflow: visible;
  text-overflow: unset;
  -webkit-line-clamp: unset;
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

.search-highlight {
  padding: 0 2px;
  border-radius: 3px;
  background: #fff3a3;
  color: inherit;
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
