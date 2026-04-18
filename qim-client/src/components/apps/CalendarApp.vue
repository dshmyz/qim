<template>
  <div class="calendar-app">
    <div class="calendar-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-arrow-left"></i>
        </button>
        <div class="calendar-header-info">
          <h2>日历</h2>
          <div class="current-date">{{ selectedDate.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' }) }}</div>
        </div>
      </div>
      <button class="create-event-btn" @click="showCreateEventModal">+ 新建事件</button>
    </div>
    <div class="calendar-content">
      <div class="calendar-nav">
        <button class="calendar-nav-btn" @click="prevMonth">&lt;</button>
        <div class="current-month">{{ selectedDate.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long' }) }}</div>
        <button class="calendar-nav-btn" @click="nextMonth">&gt;</button>
      </div>
      <div class="calendar-grid">
        <div class="calendar-header-row">
          <div class="calendar-day-header">日</div>
          <div class="calendar-day-header">一</div>
          <div class="calendar-day-header">二</div>
          <div class="calendar-day-header">三</div>
          <div class="calendar-day-header">四</div>
          <div class="calendar-day-header">五</div>
          <div class="calendar-day-header">六</div>
        </div>
        <div class="calendar-body">
          <div 
            v-for="day in calendarDays" 
            :key="day.date.toString()"
            class="calendar-day"
            :class="{ 'other-month': !day.isCurrentMonth, 'today': day.isToday, 'has-events': day.events.length > 0 }"
            @click="selectDate(day.date)"
          >
            <div class="day-number">{{ day.date.getDate() }}</div>
            <div class="day-events">
              <div 
                v-for="event in day.events.slice(0, 2)" 
                :key="event.id"
                class="day-event"
                :class="{ 'all-day': event.allDay }"
                @click.stop="showEditEventModal(event)"
              >
                {{ event.title }}
              </div>
              <div v-if="day.events.length > 2" class="more-events">+{{ day.events.length - 2 }} 更多</div>
            </div>
          </div>
        </div>
      </div>
      <div class="events-list">
        <h3>今日事件</h3>
        <div class="events-list-content">
          <div 
            v-for="event in todayEvents" 
            :key="event.id"
            class="event-item"
            @click="showEditEventModal(event)"
          >
            <div class="event-time">{{ formatEventTime(event) }}</div>
            <div class="event-info">
              <div class="event-title">{{ event.title }}</div>
              <div class="event-description">{{ event.description }}</div>
            </div>
            <button class="event-delete-btn" @click.stop="deleteEvent(event.id)">
              <i class="fas fa-trash"></i>
            </button>
          </div>
          <div v-if="todayEvents.length === 0" class="empty-events">
            <div class="empty-icon"><i class="far fa-calendar-alt"></i></div>
            <p>今日无事件</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 事件模态框 -->
    <div v-if="showCalendarModal" class="modal-overlay" @click="selectedEvent ? closeEditEventModal() : closeCreateEventModal()">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>{{ selectedEvent ? '编辑事件' : '创建事件' }}</h3>
          <button class="modal-close" @click="selectedEvent ? closeEditEventModal() : closeCreateEventModal()">×</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>标题</label>
            <input type="text" class="form-input" v-model="formData.title" placeholder="事件标题">
          </div>
          <div class="form-group">
            <label>描述</label>
            <textarea class="form-textarea" v-model="formData.description" placeholder="事件描述"></textarea>
          </div>
          <div class="form-group">
            <label>开始时间</label>
            <input type="datetime-local" class="form-input" v-model="formData.start">
          </div>
          <div class="form-group">
            <label>结束时间</label>
            <input type="datetime-local" class="form-input" v-model="formData.end">
          </div>
          <div class="form-group checkbox-group">
            <input type="checkbox" id="allDay" v-model="formData.allDay">
            <label for="allDay">全天</label>
          </div>
          <div class="form-group">
            <label>提醒</label>
            <select v-model="formData.reminder" class="form-select">
              <option value="0">无提醒</option>
              <option value="5">5分钟前</option>
              <option value="15">15分钟前</option>
              <option value="30">30分钟前</option>
              <option value="60">1小时前</option>
              <option value="1440">1天前</option>
            </select>
          </div>
        </div>
        <div class="modal-footer">
          <button class="modal-btn cancel-btn" @click="selectedEvent ? closeEditEventModal() : closeCreateEventModal()">取消</button>
          <button class="modal-btn confirm-btn" @click="selectedEvent ? updateEvent() : createEvent()">{{ selectedEvent ? '更新' : '创建' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { API_BASE_URL } from '../../config'

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 获取token
const getToken = () => {
  return localStorage.getItem('token')
}

// 日历应用相关状态
const showCalendarModal = ref(false)
const selectedDate = ref(new Date())
const events = ref<any[]>([])
const newEvent = ref({
  title: '',
  description: '',
  start: new Date(),
  end: new Date(),
  allDay: false,
  reminder: 0
})
const selectedEvent = ref<any>(null)
const formData = ref({
  title: '',
  description: '',
  start: new Date(),
  end: new Date(),
  allDay: false,
  reminder: 0
})

// 存储定时器ID
const reminderTimers = ref<Map<string, NodeJS.Timeout>>(new Map())

// 加载事件数据
const loadEvents = async () => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/events`, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    events.value = response.data.data
  } catch (error) {
    console.error('加载事件失败:', error)
    ElMessage.error('加载事件失败，请稍后重试')
  }
  
  // 设置所有事件的提醒
  setupAllReminders()
}

// 创建事件
const createEvent = async () => {
  try {
    const token = getToken()
    // 确保日期是Date对象并格式化为ISO字符串，allDay转换为布尔值
    // 不发送reminder字段，与Main.vue保持一致
    const eventData = {
      title: formData.value.title,
      description: formData.value.description,
      start: new Date(formData.value.start).toISOString(),
      end: new Date(formData.value.end).toISOString(),
      allDay: Boolean(formData.value.allDay)
    }
    console.log('发送事件数据:', eventData)
    console.log('Token:', token)
    const response = await axios.post(`${serverUrl.value}/api/v1/events`, eventData, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    events.value.push(response.data.data)
    // 设置新事件的提醒
    setEventReminder(response.data.data)
    closeCreateEventModal()
  } catch (error) {
    console.error('创建事件失败:', error)
    ElMessage.error('创建事件失败，请稍后重试')
  }
}

// 更新事件
const updateEvent = async () => {
  try {
    const updatedEvent = {
      ...selectedEvent.value,
      title: formData.value.title,
      description: formData.value.description,
      start: new Date(formData.value.start).toISOString(),
      end: new Date(formData.value.end).toISOString(),
      allDay: Boolean(formData.value.allDay)
    }
    const response = await axios.put(`${serverUrl.value}/api/v1/events/${selectedEvent.value.id}`, updatedEvent, {
      headers: {
        'Content-Type': 'application/json'
      }
    })
    const index = events.value.findIndex(e => e.id === selectedEvent.value.id)
    if (index !== -1) {
      events.value[index] = response.data.data
      // 清除旧的提醒
      clearEventReminder(selectedEvent.value.id)
      // 设置新的提醒
      setEventReminder(response.data.data)
    }
    closeEditEventModal()
  } catch (error) {
    console.error('更新事件失败:', error)
    // 直接更新本地数据
    const updatedEvent = {
      ...selectedEvent.value,
      title: formData.value.title,
      description: formData.value.description,
      start: new Date(formData.value.start),
      end: new Date(formData.value.end),
      allDay: formData.value.allDay,
      reminder: formData.value.reminder
    }
    const index = events.value.findIndex(e => e.id === selectedEvent.value.id)
    if (index !== -1) {
      events.value[index] = updatedEvent
      // 清除旧的提醒
      clearEventReminder(selectedEvent.value.id)
      // 设置新的提醒
      setEventReminder(updatedEvent)
    }
    closeEditEventModal()
  }
}

// 删除事件
const deleteEvent = async (eventId: string) => {
  try {
    await axios.delete(`${serverUrl.value}/api/v1/events/${eventId}`, {
      headers: {
        'Content-Type': 'application/json'
      }
    })
    // 清除事件的提醒
    clearEventReminder(eventId)
    events.value = events.value.filter(e => e.id !== eventId)
  } catch (error) {
    console.error('删除事件失败:', error)
    // 清除事件的提醒
    clearEventReminder(eventId)
    // 直接更新本地数据
    events.value = events.value.filter(e => e.id !== eventId)
  }
}

// 显示创建事件模态框
const showCreateEventModal = () => {
  formData.value = {
    title: '',
    description: '',
    start: new Date(),
    end: new Date(),
    allDay: false,
    reminder: 0
  }
  selectedEvent.value = null
  showCalendarModal.value = true
}

// 显示编辑事件模态框
const showEditEventModal = (event: any) => {
  selectedEvent.value = { ...event }
  formData.value = {
    title: event.title,
    description: event.description,
    start: event.start,
    end: event.end,
    allDay: event.allDay,
    reminder: event.reminder || 0
  }
  showCalendarModal.value = true
}

// 关闭创建事件模态框
const closeCreateEventModal = () => {
  showCalendarModal.value = false
  formData.value = {
    title: '',
    description: '',
    start: new Date(),
    end: new Date(),
    allDay: false,
    reminder: 0
  }
  selectedEvent.value = null
}

// 关闭编辑事件模态框
const closeEditEventModal = () => {
  showCalendarModal.value = false
  selectedEvent.value = null
  formData.value = {
    title: '',
    description: '',
    start: new Date(),
    end: new Date(),
    allDay: false,
    reminder: 0
  }
}

// 计算日历天数
const calendarDays = computed(() => {
  const year = selectedDate.value.getFullYear()
  const month = selectedDate.value.getMonth()
  
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)
  const startDate = new Date(firstDay)
  startDate.setDate(startDate.getDate() - firstDay.getDay())
  
  const days = []
  for (let i = 0; i < 42; i++) {
    const date = new Date(startDate)
    date.setDate(startDate.getDate() + i)
    
    const dayEvents = events.value.filter(event => {
      const eventDate = new Date(event.start)
      return eventDate.getFullYear() === date.getFullYear() &&
             eventDate.getMonth() === date.getMonth() &&
             eventDate.getDate() === date.getDate()
    })
    
    days.push({
      date,
      isCurrentMonth: date.getMonth() === month,
      isToday: date.toDateString() === new Date().toDateString(),
      events: dayEvents
    })
  }
  
  return days
})

// 今日事件
const todayEvents = computed(() => {
  return events.value.filter(event => {
    const eventDate = new Date(event.start)
    const today = new Date()
    return eventDate.getFullYear() === today.getFullYear() &&
           eventDate.getMonth() === today.getMonth() &&
           eventDate.getDate() === today.getDate()
  })
})

// 上个月
const prevMonth = () => {
  selectedDate.value = new Date(selectedDate.value.getFullYear(), selectedDate.value.getMonth() - 1, 1)
}

// 下个月
const nextMonth = () => {
  selectedDate.value = new Date(selectedDate.value.getFullYear(), selectedDate.value.getMonth() + 1, 1)
}

// 选择日期
const selectDate = (date: Date) => {
  selectedDate.value = date
}

// 格式化事件时间
const formatEventTime = (event: any) => {
  if (event.allDay) {
    return '全天'
  }
  const start = new Date(event.start)
  const end = new Date(event.end)
  return `${start.getHours().toString().padStart(2, '0')}:${start.getMinutes().toString().padStart(2, '0')} - ${end.getHours().toString().padStart(2, '0')}:${end.getMinutes().toString().padStart(2, '0')}`
}

// 显示提醒通知
const showReminderNotification = (event: any) => {
  if ('Notification' in window) {
    if (Notification.permission === 'granted') {
      new Notification('日历提醒', {
        body: `事件: ${event.title}\n时间: ${formatEventTime(event)}\n描述: ${event.description || '无'}`,
        icon: 'https://api.dicebear.com/7.x/avataaars/svg?seed=calendar'
      })
    } else if (Notification.permission !== 'denied') {
      Notification.requestPermission().then(permission => {
        if (permission === 'granted') {
          new Notification('日历提醒', {
            body: `事件: ${event.title}\n时间: ${formatEventTime(event)}\n描述: ${event.description || '无'}`,
            icon: 'https://api.dicebear.com/7.x/avataaars/svg?seed=calendar'
          })
        }
      })
    }
  }
  
  // 显示浏览器通知
  console.log('提醒:', event.title)
}

// 设置事件提醒
const setEventReminder = (event: any) => {
  if (!event.reminder || event.reminder === 0) return
  
  const eventDate = new Date(event.start)
  const reminderTime = eventDate.getTime() - (event.reminder * 60 * 1000)
  const now = Date.now()
  
  if (reminderTime > now) {
    const timerId = setTimeout(() => {
      showReminderNotification(event)
      reminderTimers.value.delete(event.id)
    }, reminderTime - now)
    
    reminderTimers.value.set(event.id, timerId)
  }
}

// 清除事件提醒
const clearEventReminder = (eventId: string) => {
  const timerId = reminderTimers.value.get(eventId)
  if (timerId) {
    clearTimeout(timerId)
    reminderTimers.value.delete(eventId)
  }
}

// 为所有事件设置提醒
const setupAllReminders = () => {
  // 清除所有现有定时器
  reminderTimers.value.forEach(timerId => clearTimeout(timerId))
  reminderTimers.value.clear()
  
  // 为每个事件设置提醒
  events.value.forEach(event => {
    setEventReminder(event)
  })
}

// 组件挂载时加载事件数据
onMounted(async () => {
  await loadEvents()
  // 设置所有事件的提醒
  setupAllReminders()
  
  // 请求通知权限
  if ('Notification' in window && Notification.permission !== 'granted' && Notification.permission !== 'denied') {
    Notification.requestPermission()
  }
})

// 组件卸载时清除所有定时器
onUnmounted(() => {
  reminderTimers.value.forEach(timerId => clearTimeout(timerId))
  reminderTimers.value.clear()
})
</script>

<style scoped>
.calendar-app {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.calendar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background-color: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
  height: 72px;
  box-sizing: border-box;
}

.calendar-header:hover {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: background 0.2s;
  color: var(--primary-color);
}

.back-btn:hover {
  background: var(--primary-light);
}

.calendar-header-info h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px 0;
  transition: color 0.3s ease;
}

.current-date {
  font-size: 14px;
  color: var(--text-secondary);
  transition: color 0.3s ease;
}

.create-event-btn {
  padding: 8px 16px;
  background-color: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 6px;
}

.create-event-btn:hover {
  background-color: var(--active-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

.calendar-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 24px;
  gap: 24px;
  overflow-y: auto;
  background-color: var(--bg-color);
}

.calendar-nav {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 20px;
  margin-bottom: 16px;
}

.calendar-nav-btn {
  width: 36px;
  height: 36px;
  border: 1px solid var(--border-color);
  background-color: var(--card-bg);
  color: var(--text-primary);
  border-radius: 50%;
  font-size: 16px;
  font-weight: bold;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.calendar-nav-btn:hover {
  background-color: var(--hover-color);
  border-color: var(--primary-color);
  transform: scale(1.05);
}

.current-month {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  transition: color 0.3s ease;
}

.calendar-grid {
  background-color: var(--card-bg);
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.calendar-grid:hover {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

.calendar-header-row {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 4px;
  margin-bottom: 12px;
}

.calendar-day-header {
  text-align: center;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
  padding: 8px;
  border-radius: 4px;
  transition: all 0.3s ease;
}

.calendar-body {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 4px;
}

.calendar-day {
  min-height: 100px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.calendar-day:hover {
  border-color: var(--primary-color);
  background-color: var(--hover-color);
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

.calendar-day.other-month {
  color: var(--text-tertiary);
  background-color: var(--bg-color);
}

.calendar-day.today {
  background-color: var(--primary-light);
  border-color: var(--primary-color);
  font-weight: 600;
}

.calendar-day.has-events {
  border-color: var(--primary-color);
}

.day-number {
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 4px;
  transition: color 0.3s ease;
}

.day-events {
  font-size: 12px;
  line-height: 1.2;
  max-height: 60px;
  overflow: hidden;
}

.day-event {
  padding: 2px 4px;
  margin-bottom: 2px;
  border-radius: 2px;
  background-color: var(--primary-light);
  color: var(--primary-color);
  font-size: 11px;
  cursor: pointer;
  transition: all 0.3s ease;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.day-event:hover {
  background-color: var(--primary-color);
  color: white;
  transform: translateX(2px);
}

.day-event.all-day {
  background-color: var(--success-light);
  color: var(--success-color);
}

.day-event.all-day:hover {
  background-color: var(--success-color);
  color: white;
}

.more-events {
  font-size: 10px;
  color: var(--text-tertiary);
  margin-top: 2px;
  font-style: italic;
}

.events-list {
  background-color: var(--card-bg);
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.events-list:hover {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

.events-list h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 16px 0;
  transition: color 0.3s ease;
}

.event-item {
  display: flex;
  align-items: flex-start;
  padding: 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  margin-bottom: 8px;
  transition: all 0.3s ease;
  cursor: pointer;
  background-color: var(--bg-color);
}

.event-item:hover {
  border-color: var(--primary-color);
  background-color: var(--hover-color);
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

.event-time {
  font-size: 12px;
  font-weight: 600;
  color: var(--primary-color);
  min-width: 80px;
  margin-right: 12px;
  padding-top: 2px;
  transition: color 0.3s ease;
}

.event-info {
  flex: 1;
}

.event-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
  transition: color 0.3s ease;
}

.event-description {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.4;
  transition: color 0.3s ease;
}

.event-delete-btn {
  width: 24px;
  height: 24px;
  border: none;
  background-color: transparent;
  color: var(--text-tertiary);
  border-radius: 50%;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-left: 8px;
}

.event-delete-btn:hover {
  background-color: var(--error-light);
  color: var(--error-color);
  transform: scale(1.1);
}

.empty-events {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  text-align: center;
  color: var(--text-tertiary);
  transition: color 0.3s ease;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
  transition: opacity 0.3s ease;
}

.empty-events:hover .empty-icon {
  opacity: 0.8;
}

.empty-events p {
  margin: 0;
  font-size: 14px;
  transition: color 0.3s ease;
}

/* 模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.3s ease;
}

.modal-content {
  background-color: var(--card-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  animation: slideIn 0.3s ease;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background-color: var(--bg-color);
  border-radius: 8px 8px 0 0;
}

.modal-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  transition: color 0.3s ease;
}

.modal-close {
  width: 24px;
  height: 24px;
  border: none;
  background-color: transparent;
  color: var(--text-secondary);
  font-size: 20px;
  font-weight: bold;
  cursor: pointer;
  border-radius: 50%;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-close:hover {
  background-color: var(--hover-color);
  color: var(--text-primary);
  transform: rotate(90deg);
}

.modal-body {
  padding: 20px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 6px;
  transition: color 0.3s ease;
}

.form-input,
.form-textarea,
.form-select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  color: var(--text-primary);
  background-color: var(--bg-color);
  transition: all 0.3s ease;
  box-sizing: border-box;
}

.form-input:focus,
.form-textarea:focus,
.form-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-textarea {
  resize: vertical;
  min-height: 100px;
}

.checkbox-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.checkbox-group input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
  accent-color: var(--primary-color);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 0 20px 20px;
}

.modal-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.modal-btn.cancel-btn {
  background-color: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.modal-btn.cancel-btn:hover {
  background-color: var(--hover-color);
  color: var(--text-primary);
  border-color: var(--primary-color);
  transform: translateY(-1px);
}

.modal-btn.confirm-btn {
  background-color: var(--primary-color);
  color: white;
}

.modal-btn.confirm-btn:hover {
  background-color: var(--active-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

/* 动画效果 */
@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 响应式设计 */
@media (max-width: 768px) {
  .calendar-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    padding: 16px 20px;
  }
  
  .create-event-btn {
    align-self: stretch;
  }
  
  .calendar-content {
    padding: 16px 20px;
  }
  
  .calendar-grid {
    font-size: 14px;
  }
  
  .calendar-day {
    min-height: 80px;
  }
  
  .day-events {
    font-size: 12px;
  }
  
  .events-list h3 {
    font-size: 14px;
  }
  
  .event-item {
    padding: 12px;
  }
  
  .event-time {
    font-size: 12px;
  }
  
  .event-title {
    font-size: 13px;
  }
  
  .event-description {
    font-size: 12px;
  }
  
  .modal-content {
    width: 95%;
    margin: 20px;
  }
  
  .modal-header h3 {
    font-size: 14px;
  }
  
  .modal-body {
    padding: 16px;
  }
  
  .form-group label {
    font-size: 13px;
  }
  
  .form-input,
  .form-textarea,
  .form-select {
    font-size: 14px;
    padding: 8px 12px;
  }
  
  .modal-footer {
    padding: 0 16px 16px;
  }
  
  .modal-btn {
    padding: 8px 20px;
    font-size: 13px;
  }
}

@media (max-width: 480px) {
  .calendar-day {
    min-height: 60px;
  }
  
  .day-number {
    font-size: 12px;
  }
  
  .day-event {
    font-size: 10px;
    padding: 2px 4px;
  }
  
  .event-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  
  .event-time {
    min-width: unset;
    margin-right: 0;
  }
  
  .event-delete-btn {
    align-self: flex-end;
    margin-left: 0;
  }
}
</style>