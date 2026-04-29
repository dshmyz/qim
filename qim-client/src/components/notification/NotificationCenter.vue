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
        <button class="icon-btn" :class="{ active: showSearch }" @click="toggleSearch" title="搜索">
          <i class="fas fa-search"></i>
        </button>
        <button class="icon-btn" @click="markAllAsRead" title="全部已读">
          <i class="fas fa-check-double"></i>
        </button>
        <button class="clear-all-btn" @click="clearAll">清空</button>
      </div>
    </div>

    <div v-if="showSearch" class="notification-search">
      <input
        v-model="searchKeyword"
        class="search-input"
        placeholder="搜索通知..."
      />
    </div>
    
    <div class="notification-center-tabs">
      <div
        v-for="tab in filterTabs"
        :key="tab.value"
        class="notification-tab"
        :class="{ active: currentFilter === tab.value }"
        @click="switchFilter(tab.value)"
      >
        {{ tab.label }}
        <span v-if="tab.value !== 'all'" class="tab-badge">{{ tab.count }}</span>
      </div>
    </div>

    <div v-if="hasAnyFilters" class="notification-sort-bar">
      <span class="sort-label">排序：</span>
      <span
        v-for="opt in sortOptions"
        :key="opt.value"
        class="sort-option"
        :class="{ active: sortBy === opt.value }"
        @click="sortBy = opt.value"
      >
        {{ opt.label }}
      </span>
    </div>
    
    <div class="notification-center-content">
      <div v-if="displayNotifications.length === 0" class="empty-notifications">
        <i class="fas fa-bell-slash"></i>
        <p>{{ emptyText }}</p>
      </div>
      <div v-else class="notification-list">
        <div v-if="pinnedNotifications.length > 0" class="notification-section">
          <div class="section-label"><i class="fas fa-thumbtack"></i> 置顶</div>
          <div
            v-for="notification in pinnedNotifications"
            :key="notification.id"
            class="notification-item"
            :class="getItemClasses(notification)"
          >
            <div class="notification-body" @click="handleClick(notification)">
              <div class="notification-icon">
                <i :class="getIconClass(notification)"></i>
              </div>
              <div class="notification-content">
                <div class="notification-title-row">
                  <span class="notification-title">{{ notification.title }}</span>
                  <div class="notification-badges">
                    <span v-if="notification.important" class="badge-important" title="重要">
                      <i class="fas fa-star"></i>
                    </span>
                    <span v-if="notification.handled" class="badge-handled">已处理</span>
                  </div>
                </div>
                <div class="notification-text">{{ notification.content }}</div>
                <div class="notification-time">{{ formatTime(notification.timestamp) }}</div>
              </div>
            </div>
            <div class="notification-footer">
              <div v-if="!notification.handled" class="notification-actions">
                <button
                  v-for="btn in getActionButtons(notification)"
                  :key="btn.value"
                  class="action-btn"
                  :class="btn.style"
                  @click.stop="handleAction(notification, btn.value)"
                >
                  {{ btn.label }}
                </button>
              </div>
              <div class="notification-tools">
                <button class="tool-btn" :class="{ active: notification.pinned }" @click.stop="togglePin(notification)" title="置顶">
                  <i class="fas fa-thumbtack"></i>
                </button>
                <button class="tool-btn" :class="{ active: notification.important }" @click.stop="toggleImportant(notification)" title="标记重要">
                  <i class="fas fa-star"></i>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div v-if="pinnedNotifications.length > 0 && normalNotifications.length > 0" class="section-divider"></div>

        <div
          v-for="notification in normalNotifications"
          :key="notification.id"
          class="notification-item"
          :class="getItemClasses(notification)"
        >
          <div class="notification-body" @click="handleClick(notification)">
            <div class="notification-icon">
              <i :class="getIconClass(notification)"></i>
            </div>
            <div class="notification-content">
              <div class="notification-title-row">
                <span class="notification-title">{{ notification.title }}</span>
                <div class="notification-badges">
                  <span v-if="notification.important" class="badge-important" title="重要">
                    <i class="fas fa-star"></i>
                  </span>
                  <span v-if="notification.handled" class="badge-handled">已处理</span>
                </div>
              </div>
              <div class="notification-text">{{ notification.content }}</div>
              <div class="notification-time">{{ formatTime(notification.timestamp) }}</div>
            </div>
          </div>
          <div class="notification-footer">
            <div v-if="!notification.handled" class="notification-actions">
              <button
                v-for="btn in getActionButtons(notification)"
                :key="btn.value"
                class="action-btn"
                :class="btn.style"
                @click.stop="handleAction(notification, btn.value)"
              >
                {{ btn.label }}
              </button>
            </div>
            <div class="notification-tools">
              <button class="tool-btn" :class="{ active: notification.pinned }" @click.stop="togglePin(notification)" title="置顶">
                <i class="fas fa-thumbtack"></i>
              </button>
              <button class="tool-btn" :class="{ active: notification.important }" @click.stop="toggleImportant(notification)" title="标记重要">
                <i class="fas fa-star"></i>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import axios from 'axios'
import { API_BASE_URL } from '../../config'
import { type Notification, mapNotifications } from '../../utils/notificationMapper'

interface Props {
  show: boolean
  position: { x: number; y: number }
}

defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'notificationClick', notification: Notification): void
}>()

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)
const getToken = () => localStorage.getItem('token')

const notifications = ref<Notification[]>([])
const currentFilter = ref('all')
const isLoading = ref(false)
const hasMore = ref(true)
const page = ref(1)
const pageSize = 50
const showSearch = ref(false)
const searchKeyword = ref('')
const sortBy = ref<'time' | 'importance'>('time')

const filterTabs = computed(() => {
  const unread = notifications.value.filter(n => !n.read).length
  const important = notifications.value.filter(n => n.important).length
  return [
    { value: 'all', label: '全部', count: notifications.value.length },
    { value: 'unread', label: '未读', count: unread },
    { value: 'important', label: '重要', count: important },
  ]
})

const sortOptions = [
  { value: 'time' as const, label: '时间' },
  { value: 'importance' as const, label: '重要性' },
]

const hasAnyFilters = computed(() => currentFilter.value !== 'all' || searchKeyword.value !== '')

const emptyText = computed(() => {
  if (searchKeyword.value) return '未找到匹配的通知'
  if (currentFilter.value === 'unread') return '暂无未读通知'
  if (currentFilter.value === 'important') return '暂无重要通知'
  return '暂无通知'
})

const applyFilters = (list: Notification[]) => {
  let result = list

  if (currentFilter.value === 'unread') {
    result = result.filter(n => !n.read)
  } else if (currentFilter.value === 'important') {
    result = result.filter(n => n.important)
  }

  if (searchKeyword.value.trim()) {
    const kw = searchKeyword.value.trim().toLowerCase()
    result = result.filter(n =>
      n.title.toLowerCase().includes(kw) || n.content.toLowerCase().includes(kw)
    )
  }

  if (sortBy.value === 'importance') {
    result = [...result].sort((a, b) => {
      if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
      if (a.important !== b.important) return a.important ? -1 : 1
      return b.timestamp - a.timestamp
    })
  }

  return result
}

const pinnedNotifications = computed(() => {
  return applyFilters(notifications.value).filter(n => n.pinned)
})

const normalNotifications = computed(() => {
  return applyFilters(notifications.value).filter(n => !n.pinned)
})

const displayNotifications = computed(() => {
  return [...pinnedNotifications.value, ...normalNotifications.value]
})

const loadNotifications = async (isLoadMore = false) => {
  if (isLoading.value || (!hasMore.value && isLoadMore)) return
  
  isLoading.value = true
  try {
    const token = getToken()
    const currentPage = isLoadMore ? page.value + 1 : 1
    const response = await axios.get(`${serverUrl.value}/api/v1/notifications`, {
      headers: { 'Authorization': `Bearer ${token}` },
      params: { page: currentPage, page_size: pageSize }
    })
    if (response.data.code === 0) {
      const mapped = mapNotifications(response.data.data)
      if (isLoadMore) {
        notifications.value = [...notifications.value, ...mapped]
      } else {
        notifications.value = mapped
      }
      page.value = currentPage
      hasMore.value = mapped.length === pageSize
    }
  } catch (error) {
    console.error('加载通知失败:', error)
    if (!isLoadMore) {
      notifications.value = []
    }
  } finally {
    isLoading.value = false
  }
}

const markAsRead = async (notificationId: string) => {
  try {
    const token = getToken()
    await axios.put(`${serverUrl.value}/api/v1/notifications/${notificationId}/read`, {}, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    const notification = notifications.value.find(n => n.id === notificationId)
    if (notification) notification.read = true
  } catch (error) {
    const notification = notifications.value.find(n => n.id === notificationId)
    if (notification) notification.read = true
  }
}

const markAllAsRead = async () => {
  try {
    const token = getToken()
    await axios.put(`${serverUrl.value}/api/v1/notifications/read-all`, {}, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    notifications.value.forEach(n => { n.read = true })
  } catch (error) {
    notifications.value.forEach(n => { n.read = true })
  }
}

const clearAll = async () => {
  try {
    const token = getToken()
    await axios.delete(`${serverUrl.value}/api/v1/notifications`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    notifications.value = []
  } catch (error) {
    console.error('清空通知失败:', error)
    notifications.value = []
  }
}

const switchFilter = (filter: string) => {
  currentFilter.value = filter
}

const toggleSearch = () => {
  showSearch.value = !showSearch.value
  if (!showSearch.value) searchKeyword.value = ''
}

const ACTION_BUTTONS: Record<string, Array<{ value: string; label: string; style: string }>> = {
  accept_ignore: [
    { value: 'accept', label: '接受', style: 'primary' },
    { value: 'ignore', label: '忽略', style: 'secondary' },
  ],
  confirm_reschedule: [
    { value: 'confirm', label: '确认', style: 'primary' },
    { value: 'reschedule', label: '延期', style: 'secondary' },
  ],
}

const getActionButtons = (notification: Notification) => {
  if (notification.handled) return []
  return ACTION_BUTTONS[notification.actionType] || []
}

const handleAction = async (notification: Notification, action: string) => {
  try {
    const token = getToken()
    await axios.patch(
      `${serverUrl.value}/api/v1/notifications/${notification.id}/action`,
      { action },
      { headers: { 'Authorization': `Bearer ${token}` } }
    )
    notification.handled = true
    notification.read = true
    emit('notificationClick', notification)
  } catch (error) {
    console.error('执行通知操作失败:', error)
  }
}

const togglePin = async (notification: Notification) => {
  try {
    const token = getToken()
    const res = await axios.patch(
      `${serverUrl.value}/api/v1/notifications/${notification.id}/pin`,
      {},
      { headers: { 'Authorization': `Bearer ${token}` } }
    )
    notification.pinned = res.data.pinned
  } catch (error) {
    notification.pinned = !notification.pinned
    console.error('切换置顶失败:', error)
  }
}

const toggleImportant = async (notification: Notification) => {
  try {
    const token = getToken()
    const res = await axios.patch(
      `${serverUrl.value}/api/v1/notifications/${notification.id}/important`,
      {},
      { headers: { 'Authorization': `Bearer ${token}` } }
    )
    notification.important = res.data.important
  } catch (error) {
    notification.important = !notification.important
    console.error('切换重要状态失败:', error)
  }
}

const handleClick = (notification: Notification) => {
  markAsRead(notification.id)
  emit('notificationClick', notification)
  emit('close')
}

const getIconClass = (notification: Notification): string => {
  switch (notification.category) {
    case 'message': return 'fas fa-comment'
    case 'group': return 'fas fa-user-friends'
    default: return 'fas fa-bell'
  }
}

const getItemClasses = (notification: Notification) => {
  return {
    unread: !notification.read,
    important: notification.important,
    pinned: notification.pinned,
    handled: notification.handled,
  }
}

const formatTime = (timestamp: number | string): string => {
  const timestampNum = typeof timestamp === 'string' ? parseInt(timestamp) : timestamp
  const now = Date.now()
  const diff = now - timestampNum
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)
  
  if (isNaN(diff)) return ''
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  
  const date = new Date(timestampNum)
  if (isNaN(date.getTime())) return ''
  return `${date.getMonth() + 1}-${date.getDate()}`
}

const handleClickOutside = () => {
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
  notifications,
})
</script>

<style scoped>
.notification-center {
  position: fixed;
  width: 400px;
  max-height: 520px;
  background: var(--sidebar-bg);
  border-radius: 8px;
  box-shadow: var(--shadow-md);
  z-index: 1000;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.notification-center-header {
  padding: 14px 18px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--header-panel-bg);
}

.notification-center-header h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-color);
}

.notification-center-actions {
  display: flex;
  gap: 6px;
  align-items: center;
}

.icon-btn {
  padding: 5px 8px;
  font-size: 13px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.icon-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.icon-btn.active {
  color: var(--primary-color);
  background: var(--primary-light);
}

.clear-all-btn {
  padding: 4px 10px;
  font-size: 12px;
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.clear-all-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.notification-search {
  padding: 10px 18px;
  border-bottom: 1px solid var(--border-color);
  background: var(--sidebar-bg);
}

.search-input {
  width: 100%;
  padding: 7px 12px;
  font-size: 13px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--list-bg);
  color: var(--text-color);
  outline: none;
  transition: border-color 0.2s;
}

.search-input:focus {
  border-color: var(--primary-color);
}

.notification-center-tabs {
  display: flex;
  border-bottom: 1px solid var(--border-color);
  background: var(--sidebar-bg);
}

.notification-tab {
  flex: 1;
  padding: 10px 14px;
  text-align: center;
  font-size: 13px;
  color: var(--text-secondary);
  cursor: pointer;
  position: relative;
  transition: all 0.2s ease;
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
  padding: 1px 5px;
  border-radius: 8px;
  min-width: 16px;
  text-align: center;
}

.notification-sort-bar {
  display: flex;
  align-items: center;
  padding: 6px 18px;
  background: var(--sidebar-bg);
  border-bottom: 1px solid var(--border-color);
  gap: 12px;
}

.sort-label {
  font-size: 12px;
  color: var(--text-secondary);
}

.sort-option {
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 2px 8px;
  border-radius: 3px;
  transition: all 0.2s;
}

.sort-option:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.sort-option.active {
  color: var(--primary-color);
  font-weight: 500;
  background: var(--primary-light);
}

.notification-center-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.notification-list {
  padding: 0 8px;
}

.notification-section {
  margin-bottom: 4px;
}

.section-label {
  font-size: 11px;
  color: var(--primary-color);
  font-weight: 600;
  padding: 4px 12px;
  display: flex;
  align-items: center;
  gap: 5px;
}

.section-divider {
  height: 1px;
  background: var(--border-color);
  margin: 8px 12px;
}

.notification-item {
  padding: 10px 14px;
  margin: 4px 0;
  border-radius: 6px;
  transition: all 0.2s ease;
  background: var(--list-bg);
  border: 1px solid var(--border-color);
}

.notification-item:hover {
  background: var(--hover-color);
  box-shadow: var(--shadow-sm);
}

.notification-item.unread {
  background: var(--primary-light);
  border-left: 3px solid var(--primary-color);
}

.notification-item.important {
  border-left-color: #e67e22;
}

.notification-item.pinned {
  background: var(--hover-color);
}

.notification-item.handled {
  opacity: 0.7;
}

.notification-body {
  cursor: pointer;
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.notification-icon {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--primary-light);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--primary-color);
  font-size: 13px;
}

.notification-content {
  flex: 1;
  min-width: 0;
}

.notification-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 3px;
}

.notification-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-color);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-badges {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
  align-items: center;
}

.badge-important {
  color: #e67e22;
  font-size: 10px;
}

.badge-handled {
  font-size: 10px;
  color: var(--text-secondary);
  background: var(--hover-color);
  padding: 1px 6px;
  border-radius: 3px;
}

.notification-text {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 4px;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.notification-time {
  font-size: 10px;
  color: var(--text-secondary);
  opacity: 0.7;
}

.notification-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
  padding-top: 6px;
  border-top: 1px solid var(--border-color);
}

.notification-actions {
  display: flex;
  gap: 6px;
}

.action-btn {
  padding: 3px 10px;
  font-size: 11px;
  border-radius: 4px;
  border: none;
  cursor: pointer;
  transition: all 0.2s;
}

.action-btn.primary {
  background: var(--primary-color);
  color: white;
}

.action-btn.primary:hover {
  opacity: 0.85;
}

.action-btn.secondary {
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
}

.action-btn.secondary:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.notification-tools {
  display: flex;
  gap: 4px;
}

.tool-btn {
  padding: 3px 6px;
  font-size: 11px;
  background: transparent;
  border: none;
  border-radius: 3px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
}

.tool-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.tool-btn.active {
  color: var(--primary-color);
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
  font-size: 40px;
  margin-bottom: 14px;
  opacity: 0.4;
}

.empty-notifications p {
  margin: 0;
  font-size: 13px;
}

[data-theme="dark"] .notification-center {
  background: var(--sidebar-bg) !important;
}

[data-theme="dark"] .notification-center-header {
  background: var(--header-panel-bg) !important;
}

[data-theme="dark"] .notification-search {
  background: var(--sidebar-bg) !important;
}

[data-theme="dark"] .search-input {
  background: var(--list-bg) !important;
  border-color: var(--border-color) !important;
  color: var(--text-color) !important;
}

[data-theme="dark"] .notification-center-tabs {
  background: var(--sidebar-bg) !important;
}

[data-theme="dark"] .notification-sort-bar {
  background: var(--sidebar-bg) !important;
}

[data-theme="dark"] .notification-item {
  background: var(--list-bg) !important;
  border-color: var(--border-color) !important;
}

[data-theme="dark"] .notification-item:hover {
  background: var(--hover-color) !important;
}

[data-theme="dark"] .notification-item.unread {
  background: var(--primary-light) !important;
}

[data-theme="dark"] .notification-icon {
  background: var(--primary-light) !important;
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

[data-theme="dark"] .notification-footer {
  border-top-color: var(--border-color) !important;
}

[data-theme="dark"] .action-btn.secondary {
  border-color: var(--border-color) !important;
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .action-btn.secondary:hover {
  background: var(--hover-color) !important;
}

[data-theme="dark"] .tool-btn.active {
  color: var(--primary-color) !important;
}
</style>