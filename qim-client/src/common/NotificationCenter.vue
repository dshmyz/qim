<template>
  <div 
    v-if="show" 
    class="notification-center" 
    :style="{ left: position.x + 'px', top: position.y + 'px' }" 
    @click.stop
  >
    <div class="notification-center-header">
      <h3>通知中心</h3>
      <div class="notification-center-actions">
        <button class="clear-all-btn" @click="clearAll">清空</button>
      </div>
    </div>
    
    <div class="notification-center-tabs">
      <div
        v-for="type in notificationTypes"
        :key="type.value"
        class="notification-tab"
        :class="{ active: currentType === type.value }"
        @click="switchType(type.value)"
      >
        {{ type.label }}
        <span v-if="type.value !== 'all'" class="tab-badge">{{ getCountByType(type.value) }}</span>
      </div>
    </div>
    
    <div class="notification-center-content">
      <div v-if="filteredNotifications.length === 0" class="empty-notifications">
        <i class="fas fa-bell-slash"></i>
        <p>暂无通知</p>
      </div>
      <div v-else class="notification-list">
        <div
          v-for="notification in filteredNotifications"
          :key="notification.id"
          class="notification-item"
          :class="{ unread: !notification.read }"
          @click="handleClick(notification)"
        >
          <div class="notification-icon">
            <i v-if="notification.type === 'message'" class="fas fa-comment"></i>
            <i v-else-if="notification.type === 'system'" class="fas fa-bell"></i>
            <i v-else-if="notification.type === 'group'" class="fas fa-user-friends"></i>
          </div>
          <div class="notification-content">
            <div class="notification-title">{{ notification.title }}</div>
            <div class="notification-text">{{ notification.content }}</div>
            <div class="notification-time">{{ formatTime(notification.timestamp) }}</div>
          </div>
          <div v-if="!notification.read" class="notification-badge"></div>
        </div>
        
        <div v-if="hasMore" class="load-more-container">
          <button 
            class="load-more-btn" 
            @click="loadMore"
            :disabled="isLoading"
          >
            <span v-if="isLoading">加载中...</span>
            <span v-else>加载更多</span>
          </button>
        </div>
        <div v-else-if="filteredNotifications.length > 0" class="no-more-container">
          <p>没有更多通知了</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import axios from 'axios'
import { API_BASE_URL } from '../config'

interface Notification {
  id: string
  title: string
  content: string
  timestamp: number
  read: boolean
  type: 'message' | 'system' | 'group'
  data?: {
    conversationId?: string
    groupId?: string
  }
}

interface Props {
  show: boolean
  position: { x: number; y: number }
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'notificationClick', notification: Notification): void
}>()

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)
const getToken = () => localStorage.getItem('token')

const notifications = ref<Notification[]>([])
const filteredNotifications = ref<Notification[]>([])
const currentType = ref('all')
const isLoading = ref(false)
const hasMore = ref(true)
const page = ref(1)
const pageSize = 20

const notificationTypes = [
  { value: 'all', label: '全部' },
  { value: 'message', label: '消息' },
  { value: 'system', label: '系统' },
  { value: 'group', label: '群聊' }
]

const unreadCount = computed(() => {
  return notifications.value.filter(n => !n.read).length
})

const getCountByType = (type: string) => {
  if (type === 'all') return notifications.value.length
  return notifications.value.filter(n => n.type === type).length
}

const filterNotifications = () => {
  if (currentType.value === 'all') {
    filteredNotifications.value = notifications.value
  } else {
    filteredNotifications.value = notifications.value.filter(n => n.type === currentType.value)
  }
}

const loadNotifications = async (isLoadMore = false) => {
  if (isLoading.value || (!hasMore.value && isLoadMore)) return
  
  isLoading.value = true
  try {
    const token = getToken()
    const currentPage = isLoadMore ? page.value + 1 : 1
    const response = await axios.get(`${serverUrl.value}/api/v1/notifications`, {
      headers: {
        'Authorization': `Bearer ${token}`
      },
      params: {
        page: currentPage,
        page_size: pageSize
      }
    })
    if (response.data.code === 0) {
      const newNotifications = response.data.data
      if (isLoadMore) {
        notifications.value = [...notifications.value, ...newNotifications]
      } else {
        notifications.value = newNotifications
      }
      page.value = currentPage
      hasMore.value = newNotifications.length === pageSize
      filterNotifications()
    }
  } catch (error) {
    console.error('加载通知失败:', error)
    if (!isLoadMore) {
      notifications.value = [
        {
          id: '1',
          title: '新消息',
          content: '张三发送了一条新消息',
          timestamp: Date.now() - 1000 * 60 * 5,
          read: false,
          type: 'message',
          data: { conversationId: '1' }
        },
        {
          id: '2',
          title: '系统通知',
          content: '您的账号已成功登录',
          timestamp: Date.now() - 1000 * 60 * 30,
          read: false,
          type: 'system'
        },
        {
          id: '3',
          title: '群聊邀请',
          content: '李四邀请您加入群聊 "技术讨论"',
          timestamp: Date.now() - 1000 * 60 * 60,
          read: true,
          type: 'group',
          data: { groupId: '1' }
        },
        {
          id: '4',
          title: '新消息',
          content: '王五发送了一条新消息',
          timestamp: Date.now() - 1000 * 60 * 120,
          read: false,
          type: 'message',
          data: { conversationId: '2' }
        },
        {
          id: '5',
          title: '系统通知',
          content: '您的账号权限已更新',
          timestamp: Date.now() - 1000 * 60 * 180,
          read: true,
          type: 'system'
        },
        {
          id: '6',
          title: '群聊邀请',
          content: '赵六邀请您加入群聊 "产品讨论"',
          timestamp: Date.now() - 1000 * 60 * 240,
          read: false,
          type: 'group',
          data: { groupId: '2' }
        }
      ]
      page.value = 1
      hasMore.value = false
      filterNotifications()
    }
  } finally {
    isLoading.value = false
  }
}

const markAsRead = async (notificationId: string) => {
  try {
    const token = getToken()
    await axios.put(`${serverUrl.value}/api/v1/notifications/${notificationId}/read`, {}, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    const notification = notifications.value.find(n => n.id === notificationId)
    if (notification) {
      notification.read = true
      filterNotifications()
    }
  } catch (error) {
    console.error('标记通知已读失败:', error)
    const notification = notifications.value.find(n => n.id === notificationId)
    if (notification) {
      notification.read = true
      filterNotifications()
    }
  }
}

const markAllAsRead = async () => {
  try {
    const token = getToken()
    await axios.put(`${serverUrl.value}/api/v1/notifications/read-all`, {}, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    notifications.value.forEach(n => { n.read = true })
    filterNotifications()
  } catch (error) {
    console.error('标记所有通知已读失败:', error)
    notifications.value.forEach(n => { n.read = true })
    filterNotifications()
  }
}

const switchType = (type: string) => {
  currentType.value = type
  filterNotifications()
}

const loadMore = () => {
  loadNotifications(true)
}

const handleClick = (notification: Notification) => {
  markAsRead(notification.id)
  emit('notificationClick', notification)
  emit('close')
}

const clearAll = async () => {
  try {
    const token = getToken()
    await axios.delete(`${serverUrl.value}/api/v1/notifications`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    notifications.value = []
    filteredNotifications.value = []
  } catch (error) {
    console.error('清空通知失败:', error)
    notifications.value = []
    filteredNotifications.value = []
  }
}

const formatTime = (timestamp: number): string => {
  const now = Date.now()
  const diff = now - timestamp
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)
  
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  
  const date = new Date(timestamp)
  return `${date.getMonth() + 1}-${date.getDate()}`
}

const handleClickOutside = (event: MouseEvent) => {
  emit('close')
}

onMounted(() => {
  setTimeout(() => {
    document.addEventListener('click', handleClickOutside)
  }, 0)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})

defineExpose({
  loadNotifications,
  markAllAsRead,
  unreadCount,
  notifications,
  filterNotifications
})
</script>

<style scoped>
.notification-center {
  position: fixed;
  width: 380px;
  max-height: 480px;
  background: var(--sidebar-bg);
  border-radius: 8px;
  box-shadow: var(--shadow-md);
  z-index: 1000;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.notification-center-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--header-panel-bg);
}

.notification-center-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
}

.notification-center-actions {
  display: flex;
  gap: 8px;
}

.clear-all-btn {
  padding: 4px 12px;
  font-size: 12px;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.3s ease;
}

.clear-all-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.notification-center-tabs {
  display: flex;
  border-bottom: 1px solid var(--border-color);
  background: var(--sidebar-bg);
}

.notification-tab {
  flex: 1;
  padding: 12px 16px;
  text-align: center;
  font-size: 13px;
  color: var(--text-secondary);
  cursor: pointer;
  position: relative;
  transition: all 0.3s ease;
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 6px;
}

.notification-tab:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.notification-tab.active {
  color: var(--primary-color);
  font-weight: 500;
  border-bottom: 2px solid var(--primary-color);
}

.tab-badge {
  background: var(--primary-color);
  color: white;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 10px;
  min-width: 18px;
  text-align: center;
}

.notification-center-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.notification-list {
  padding: 0 8px;
}

.notification-item {
  padding: 12px 16px;
  margin: 4px 0;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  background: var(--list-bg);
  border: 1px solid var(--border-color);
}

.notification-item:hover {
  background: var(--hover-color);
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.notification-item.unread {
  background: var(--primary-light);
  border-left: 3px solid var(--primary-color);
}

.notification-icon {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--primary-light);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--primary-color);
  font-size: 14px;
}

.notification-content {
  flex: 1;
  min-width: 0;
}

.notification-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 4px;
  line-height: 1.4;
}

.notification-text {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 6px;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.notification-time {
  font-size: 11px;
  color: var(--text-secondary);
  opacity: 0.8;
}

.notification-badge {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--primary-color);
  margin-top: 6px;
  flex-shrink: 0;
}

.empty-notifications {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: var(--text-secondary);
  text-align: center;
}

.empty-notifications i {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty-notifications p {
  margin: 0;
  font-size: 14px;
}

.load-more-container {
  padding: 16px;
  text-align: center;
}

.load-more-btn {
  padding: 8px 20px;
  font-size: 13px;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.3s ease;
}

.load-more-btn:hover:not(:disabled) {
  background: var(--hover-color);
  color: var(--text-color);
}

.load-more-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.no-more-container {
  padding: 16px;
  text-align: center;
  font-size: 12px;
  color: var(--text-secondary);
}

.no-more-container p {
  margin: 0;
}

[data-theme="dark"] .notification-center {
  background: var(--sidebar-bg) !important;
  box-shadow: var(--shadow-md) !important;
}

[data-theme="dark"] .notification-center-header {
  background: var(--header-panel-bg) !important;
  border-bottom-color: var(--border-color) !important;
}

[data-theme="dark"] .notification-center-header h3 {
  color: var(--text-color) !important;
}

[data-theme="dark"] .clear-all-btn {
  border-color: var(--border-color) !important;
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .clear-all-btn:hover {
  background: var(--hover-color) !important;
  color: var(--text-color) !important;
}

[data-theme="dark"] .notification-center-tabs {
  background: var(--sidebar-bg) !important;
  border-bottom-color: var(--border-color) !important;
}

[data-theme="dark"] .notification-tab {
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .notification-tab:hover {
  background: var(--hover-color) !important;
  color: var(--text-color) !important;
}

[data-theme="dark"] .notification-tab.active {
  color: var(--primary-color) !important;
  border-bottom-color: var(--primary-color) !important;
}

[data-theme="dark"] .tab-badge {
  background: var(--primary-color) !important;
}

[data-theme="dark"] .notification-item {
  background: var(--list-bg) !important;
  border-color: var(--border-color) !important;
}

[data-theme="dark"] .notification-item:hover {
  background: var(--hover-color) !important;
  box-shadow: var(--shadow-sm) !important;
}

[data-theme="dark"] .notification-item.unread {
  background: var(--primary-light) !important;
  border-left-color: var(--primary-color) !important;
}

[data-theme="dark"] .notification-icon {
  background: var(--primary-light) !important;
  color: var(--primary-color) !important;
}

[data-theme="dark"] .notification-title {
  color: var(--text-color) !important;
}

[data-theme="dark"] .notification-text {
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .notification-time {
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .empty-notifications {
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .load-more-btn {
  border-color: var(--border-color) !important;
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .load-more-btn:hover:not(:disabled) {
  background: var(--hover-color) !important;
  color: var(--text-color) !important;
}

[data-theme="dark"] .no-more-container {
  color: var(--text-secondary) !important;
}
</style>
