<template>
  <div v-if="visible" class="message-manager-modal" @click="$emit('close')">
    <div class="message-manager-content" @click.stop>
      <div class="message-manager-header">
        <h3>
          <i class="fas fa-history"></i> 消息管理器
        </h3>
        <button class="close-btn" @click="$emit('close')">×</button>
      </div>
      <div class="message-manager-body">
        <!-- 搜索框 -->
        <div class="message-manager-search">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索消息..."
            class="search-input"
            @keyup.enter="applyFilters"
          />
          <button class="search-btn" @click="applyFilters">
            <i class="fas fa-search"></i>
            搜索
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
                {{ message.type === 'text' ? '文本' : message.type === 'image' ? '图片' : message.type === 'file' ? '文件' : message.type === 'miniApp' ? '小程序' : message.type === 'share' ? '分享' : message.type === 'news' ? '资讯' : '其他' }}
              </span>
            </div>
            <div class="message-manager-item-content" :class="[`message-content-${message.type}`, { 'is-recalled': message.isRecalled }]">
              <template v-if="message.isRecalled">
                <span class="recalled-text">此消息已被撤回</span>
              </template>
              <template v-else-if="message.type === 'text'">
                {{ message.content }}
              </template>
              <template v-else-if="message.type === 'image'">
                <div class="message-file-link" @click.stop="previewImage(message)">
                  <i class="fas fa-image"></i>
                  <span>{{ message.file_name || message.content.split('/').pop() }}</span>
                </div>
              </template>
              <template v-else-if="message.type === 'file'">
                <div class="message-file-link" @click.stop="downloadFile(message)">
                  <i class="fas fa-file"></i>
                  <span>{{ message.file_name || message.content.split('/').pop() }}</span>
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
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

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
  console.log('loadMessages 被调用, conversationId:', props.conversationId)
  if (!props.conversationId) {
    console.log('loadMessages 提前返回: conversationId 为空')
    return
  }

  isLoadingMessages.value = true
  currentPage.value = page

  try {
    const token = localStorage.getItem('token')
    const serverUrl = localStorage.getItem('serverUrl') || ''
    const params = new URLSearchParams()
    params.append('conversation_id', props.conversationId)
    params.append('page', page.toString())
    params.append('page_size', pageSize.toString())
    if (selectedMessageType.value !== 'all') {
      params.append('type', selectedMessageType.value)
    }
    if (searchQuery.value) {
      params.append('search', searchQuery.value)
    }
    
    // 处理日期范围
    console.log('日期过滤检查: selectedDateRange =', selectedDateRange.value)
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
        const weekYear = weekAgo.getFullYear()
        const weekMonth = String(weekAgo.getMonth() + 1).padStart(2, '0')
        const weekDay = String(weekAgo.getDate()).padStart(2, '0')
        startDate = `${weekYear}-${weekMonth}-${weekDay}`
        endDate = todayStr
      } else if (selectedDateRange.value === 'month') {
        const monthAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000)
        const monthYear = monthAgo.getFullYear()
        const monthMonth = String(monthAgo.getMonth() + 1).padStart(2, '0')
        const monthDay = String(monthAgo.getDate()).padStart(2, '0')
        startDate = `${monthYear}-${monthMonth}-${monthDay}`
        endDate = todayStr
      } else if (selectedDateRange.value === 'custom' && customDateStart.value && customDateEnd.value) {
        startDate = customDateStart.value
        endDate = customDateEnd.value
      }

      console.log('日期过滤 - startDate:', startDate, 'endDate:', endDate, 'selectedDateRange:', selectedDateRange.value)

      if (startDate) {
        params.append('start_date', startDate)
      }
      if (endDate) {
        params.append('end_date', endDate)
      }
    }

    const response = await fetch(`${serverUrl}/api/v1/messages?${params.toString()}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      }
    })

    if (response.ok) {
      const data = await response.json()
      if (data.code === 0) {
        // 处理消息数据，添加 timestamp 字段
        messages.value = data.data.messages.map((message: any) => ({
          ...message,
          timestamp: message.created_at ? new Date(message.created_at).getTime() : Date.now(),
          isRecalled: message.is_recalled || false,
          sender: message.sender ? {
            ...message.sender,
            name: message.sender.name || message.sender.nickname || message.sender.username || message.sender.user?.nickname || message.sender.user?.username || message.username || message.name || '未知用户'
          } : null
        }))
        // 按时间倒序排列
        messages.value.sort((a, b) => b.timestamp - a.timestamp)
        total.value = data.data.total
      }
    } else if (response.status === 401) {
      ElMessage.error('登录已过期，请重新登录')
      localStorage.removeItem('token')
      setTimeout(() => {
        window.location.href = '/login'
      }, 1500)
    } else {
      ElMessage.error('加载消息失败，请稍后重试')
    }
  } catch (error) {
    console.error('加载消息失败:', error)
  } finally {
    isLoadingMessages.value = false
  }
}

// 应用过滤器
const applyFilters = () => {
  console.log('applyFilters 被调用', 'selectedDateRange:', selectedDateRange.value)
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
  const link = document.createElement('a')
  link.href = message.content
  link.download = message.file_name || message.content.split('/').pop() || 'file'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

// 预览图片
const previewImage = (message: any) => {
  previewImageUrl.value = message.content
  showImagePreview.value = true
}

// 关闭图片预览
const closeImagePreview = () => {
  showImagePreview.value = false
  previewImageUrl.value = ''
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

<style scoped>
.message-manager-modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(2px);
}

.message-manager-content {
  background: var(--card-bg);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  width: 800px;
  max-width: 90%;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.message-manager-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--secondary-color);
  border-radius: 8px 8px 0 0;
}

.message-manager-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
  display: flex;
  align-items: center;
  gap: 8px;
}

.close-btn {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary);
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: all 0.2s;
}

.close-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.message-manager-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.message-manager-search {
  margin-bottom: 8px;
  display: flex;
  gap: 10px;
  align-items: center;
}

.search-input {
  flex: 1;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 14px;
  transition: border-color 0.3s;
}

.search-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

.search-btn {
  padding: 8px 16px;
  border: 1px solid var(--primary-color);
  border-radius: 6px;
  background: var(--primary-color);
  color: white;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.search-btn:hover {
  background: var(--active-color);
  border-color: var(--active-color);
}

.search-btn i {
  font-size: 12px;
}

.message-manager-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-color);
  align-items: flex-end;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 120px;
}

.filter-group label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  white-space: nowrap;
}

.filter-select {
  padding: 6px 10px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 14px;
  cursor: pointer;
  transition: border-color 0.3s;
  min-width: 120px;
}

.filter-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

.date-range-group {
  flex: 1;
  min-width: 300px;
}

.date-range-inputs {
  display: flex;
  align-items: center;
  gap: 8px;
}

.date-input {
  padding: 6px 8px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 14px;
  transition: border-color 0.3s;
  flex: 1;
}

.date-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

.date-range-separator {
  font-size: 14px;
  color: var(--text-secondary);
  white-space: nowrap;
}

.message-manager-list {
  max-height: 450px;
  overflow-y: auto;
  background: var(--card-bg);
  border-radius: 8px;
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-color);
}

.message-manager-list::-webkit-scrollbar {
  width: 6px;
}

.message-manager-list::-webkit-scrollbar-track {
  background: var(--secondary-color);
  border-radius: 3px;
}

.message-manager-list::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.message-manager-list::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}

.loading-message,
.empty-message {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 14px;
}

.message-manager-item {
  padding: 10px 16px;
  transition: all 0.2s ease;
  cursor: pointer;
  border-bottom: 1px solid var(--border-color);
}

.message-manager-item.is-recalled {
  cursor: not-allowed;
  opacity: 0.6;
}

.message-manager-item.is-recalled:hover {
  background: transparent;
  transform: none;
  box-shadow: none;
}

.message-manager-item:hover {
  background: var(--hover-color);
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.message-manager-item:last-child {
  border-bottom: none;
}

.message-manager-item-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
  flex-wrap: wrap;
}

.message-sender {
  font-weight: 600;
  color: var(--text-color);
  font-size: 13px;
  flex: 1;
  min-width: 80px;
}

.message-time {
  font-size: 11px;
  color: var(--text-color);
  opacity: 0.6;
  transition: opacity 0.3s ease;
  flex: 0 0 auto;
}

/* 消息管理器中的消息时间始终显示 */
.message-manager-item .message-time {
  opacity: 0.7;
}

.message-manager-item:hover .message-time {
  opacity: 1;
}

.message-type {
  font-size: 11px;
  font-weight: 500;
  color: var(--primary-color);
  background: var(--hover-color);
  padding: 2px 8px;
  border-radius: 10px;
  flex: 0 0 auto;
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 4px;
}

.message-type i {
  font-size: 10px;
}

.message-type-recalled {
  color: var(--text-secondary);
  background: var(--hover-color);
}

.recalled-text {
  color: var(--text-secondary);
  font-style: italic;
}

.message-manager-item-content.is-recalled {
  color: var(--text-secondary);
  font-style: italic;
}

.message-manager-item-content {
  font-size: 13px;
  color: var(--text-color);
  line-height: 1.4;
  padding-left: 0;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 60px;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
}

.message-file-link {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-color);
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  cursor: pointer;
}

.message-file-link i {
  font-size: 14px;
  flex-shrink: 0;
  color: var(--primary-color);
}

.message-file-link span {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mini-app-info,
.share-info,
.news-info {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  max-width: 100%;
}

.mini-app-icon,
.share-icon,
.news-icon {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border-radius: 6px;
  overflow: hidden;
  background: var(--hover-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: var(--primary-color);
}

.mini-app-icon img,
.share-icon img,
.news-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.mini-app-icon i,
.share-icon i,
.news-icon i {
  font-size: inherit;
  color: inherit;
}

.mini-app-details,
.share-details,
.news-details {
  flex: 1;
  min-width: 0;
}

.mini-app-name,
.share-title,
.news-title {
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 13px;
}

.mini-app-description,
.share-description,
.news-description {
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.message-manager-pagination {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 16px;
  padding: 16px 20px;
  background: var(--secondary-color);
  border-radius: 0 0 8px 8px;
  border-top: 1px solid var(--border-color);
}

/* 悬浮分页效果 */
.sticky-pagination {
  position: sticky;
  bottom: 0;
  z-index: 10;
  margin-top: 0;
  border-radius: 0;
}

.pagination-info {
  font-size: 14px;
  color: var(--text-color);
  opacity: 0.7;
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-jump {
  display: flex;
  align-items: center;
  gap: 6px;
}

.page-input {
  width: 50px;
  padding: 6px 8px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 14px;
  text-align: center;
}

.page-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.jump-btn {
  padding: 6px 12px;
  border: 1px solid var(--primary-color);
  border-radius: 6px;
  background: var(--primary-color);
  color: white;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.jump-btn:hover {
  background: var(--active-color);
  border-color: var(--active-color);
}

.pagination-btn {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  background: var(--card-bg);
  color: var(--text-color);
  cursor: pointer;
  transition: all 0.2s ease;
}

.pagination-btn:hover:not(:disabled) {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.pagination-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* 图片预览样式 */
.image-preview-modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  cursor: pointer;
}

.image-preview-content {
  position: relative;
  max-width: 90%;
  max-height: 90%;
  cursor: default;
}

.image-preview-close {
  position: absolute;
  top: -40px;
  right: 0;
  background: none;
  border: none;
  color: white;
  font-size: 32px;
  cursor: pointer;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: all 0.2s;
}

.image-preview-close:hover {
  background: rgba(255, 255, 255, 0.2);
}

.image-preview-img {
  max-width: 100%;
  max-height: 90vh;
  object-fit: contain;
  border-radius: 8px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .message-manager-content {
    width: 95%;
    max-height: 90vh;
  }
  
  .message-manager-filters {
    flex-direction: column;
    gap: 12px;
  }
  
  .filter-group {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }
  
  .filter-select {
    flex: 1;
    min-width: 0;
  }
  
  .date-range-group {
    min-width: 0;
  }
  
  .date-range-inputs {
    flex: 1;
  }
  
  .message-manager-pagination {
    flex-direction: column;
    gap: 12px;
  }
  
  .pagination-controls {
    width: 100%;
    justify-content: center;
  }
}
</style>