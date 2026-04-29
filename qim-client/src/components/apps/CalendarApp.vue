<template>
  <div class="calendar-app">
    <AppHeader title="日历" @back="$emit('back')">
      <template #extra-buttons>
        <button class="icon-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
      </template>
      <template #subtitle>
        <div class="current-date">{{ headerDateLabel }}</div>
      </template>
      <template #actions>
        <button class="action-btn primary" @click="showCreateEventModal">
          <span class="btn-icon">+</span> 新建事件
        </button>
      </template>
    </AppHeader>

    <div class="calendar-content">
      <CalendarGrid
        :selectedDate="selectedDate"
        :events="events"
        @prevMonth="prevMonth"
        @nextMonth="nextMonth"
        @selectDate="selectDate"
      />
      <EventPanel
        :selectedDate="selectedDate"
        :events="selectedDateEvents"
        @editEvent="showEditEventModal"
        @deleteEvent="deleteEvent"
      />
    </div>

    <ModalContainer
      :visible="showCalendarModal"
      :title="selectedEvent ? '编辑事件' : '创建事件'"
      @close="selectedEvent ? closeEditEventModal() : closeCreateEventModal()"
    >
      <div class="calendar-form-group">
        <label>标题</label>
        <input type="text" class="calendar-form-input" v-model="formData.title" placeholder="事件标题">
      </div>
      <div class="calendar-form-group">
        <label>描述</label>
        <textarea class="calendar-form-textarea" v-model="formData.description" placeholder="事件描述"></textarea>
      </div>
      <div class="calendar-form-group">
        <label>开始时间</label>
        <input type="datetime-local" class="calendar-form-input" v-model="formData.start">
      </div>
      <div class="calendar-form-group">
        <label>结束时间</label>
        <input type="datetime-local" class="calendar-form-input" v-model="formData.end">
      </div>
      <div class="calendar-form-group calendar-checkbox-group">
        <input type="checkbox" id="calendar-allDay" v-model="formData.allDay">
        <label for="calendar-allDay">全天</label>
      </div>
      <div class="calendar-form-group">
        <label>提醒</label>
        <select v-model="formData.reminder" class="calendar-form-select">
          <option value="0">无提醒</option>
          <option value="5">5分钟前</option>
          <option value="15">15分钟前</option>
          <option value="30">30分钟前</option>
          <option value="60">1小时前</option>
          <option value="1440">1天前</option>
        </select>
      </div>

      <template #footer>
        <button class="calendar-modal-btn calendar-cancel-btn" @click="selectedEvent ? closeEditEventModal() : closeCreateEventModal()">取消</button>
        <button class="calendar-modal-btn calendar-confirm-btn" @click="selectedEvent ? updateEvent() : createEvent()">{{ selectedEvent ? '更新' : '创建' }}</button>
      </template>
    </ModalContainer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import axios from 'axios'
import QMessage from '../../utils/qmessage'
import { API_BASE_URL } from '../../config'
import { logger } from '../../utils/logger'
import ModalContainer from '../../components/shared/ModalContainer.vue'
import AppHeader from './AppHeader.vue'
import CalendarGrid from './calendar/CalendarGrid.vue'
import EventPanel from './calendar/EventPanel.vue'
import { generateAvatar } from '../../utils/avatar'
import { getLunarDayInfo } from '../../utils/lunar'

defineEmits<{
  back: []
  toggleSidebar: []
}>()

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

const getToken = () => {
  return localStorage.getItem('token')
}

const formatDateForInput = (date: Date): string => {
  const d = new Date(date)
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const hours = String(d.getHours()).padStart(2, '0')
  const minutes = String(d.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hours}:${minutes}`
}

const showCalendarModal = ref(false)
const selectedDate = ref(new Date())
const events = ref<any[]>([])
const selectedEvent = ref<any>(null)
const formData = ref({
  title: '',
  description: '',
  start: formatDateForInput(new Date()),
  end: formatDateForInput(new Date()),
  allDay: false,
  reminder: 0
})

const reminderTimers = ref<Map<string, NodeJS.Timeout>>(new Map())

const headerDateLabel = computed(() => {
  const date = selectedDate.value
  const lunar = getLunarDayInfo(date.getFullYear(), date.getMonth() + 1, date.getDate())
  const dateStr = date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'short' })
  return `${dateStr} · ${lunar.lunarDayName}`
})

const selectedDateEvents = computed(() => {
  return events.value.filter(event => {
    const eventDate = new Date(event.start)
    return eventDate.getFullYear() === selectedDate.value.getFullYear() &&
           eventDate.getMonth() === selectedDate.value.getMonth() &&
           eventDate.getDate() === selectedDate.value.getDate()
  })
})

const loadEvents = async () => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/events`, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    events.value = response.data.data.map((e: any) => ({
      ...e,
      allDay: e.all_day,
      reminder: e.reminder,
    }))
  } catch (error) {
    console.error('加载事件失败:', error)
    QMessage.error('加载事件失败，请稍后重试')
  }
  setupAllReminders()
}

const validateForm = (): string | null => {
  if (!formData.value.title.trim()) return '请输入事件标题'
  const start = new Date(formData.value.start)
  const end = new Date(formData.value.end)
  if (isNaN(start.getTime())) return '请选择有效的开始时间'
  if (isNaN(end.getTime())) return '请选择有效的结束时间'
  if (start >= end) return '结束时间必须晚于开始时间'
  return null
}

const createEvent = async () => {
  const error = validateForm()
  if (error) {
    QMessage.error(error)
    return
  }
  try {
    const token = getToken()
    const eventData = {
      title: formData.value.title,
      description: formData.value.description,
      start: new Date(formData.value.start).toISOString(),
      end: new Date(formData.value.end).toISOString(),
      all_day: Boolean(formData.value.allDay),
      reminder: Number(formData.value.reminder)
    }
    logger.log('发送事件数据:', eventData)
    logger.log('Token:', token)
    const response = await axios.post(`${serverUrl.value}/api/v1/events`, eventData, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    events.value.push(response.data.data)
    setEventReminder(response.data.data)
    closeCreateEventModal()
  } catch (error) {
    console.error('创建事件失败:', error)
    QMessage.error('创建事件失败，请稍后重试')
  }
}

const updateEvent = async () => {
  const error = validateForm()
  if (error) {
    QMessage.error(error)
    return
  }
  try {
    const updatedEvent = {
      ...selectedEvent.value,
      title: formData.value.title,
      description: formData.value.description,
      start: new Date(formData.value.start).toISOString(),
      end: new Date(formData.value.end).toISOString(),
      all_day: Boolean(formData.value.allDay),
      reminder: Number(formData.value.reminder)
    }
    const response = await axios.put(`${serverUrl.value}/api/v1/events/${selectedEvent.value.id}`, updatedEvent, {
      headers: {
        'Content-Type': 'application/json'
      }
    })
    const index = events.value.findIndex(e => e.id === selectedEvent.value.id)
    if (index !== -1) {
      events.value[index] = response.data.data
      clearEventReminder(selectedEvent.value.id)
      setEventReminder(response.data.data)
    }
    closeEditEventModal()
  } catch (error) {
    console.error('更新事件失败:', error)
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
      clearEventReminder(selectedEvent.value.id)
      setEventReminder(updatedEvent)
    }
    closeEditEventModal()
  }
}

const deleteEvent = async (eventId: string) => {
  try {
    await axios.delete(`${serverUrl.value}/api/v1/events/${eventId}`, {
      headers: {
        'Content-Type': 'application/json'
      }
    })
    clearEventReminder(eventId)
    events.value = events.value.filter(e => e.id !== eventId)
  } catch (error) {
    console.error('删除事件失败:', error)
    clearEventReminder(eventId)
    events.value = events.value.filter(e => e.id !== eventId)
  }
}

const showCreateEventModal = () => {
  formData.value = {
    title: '',
    description: '',
    start: formatDateForInput(new Date()),
    end: formatDateForInput(new Date()),
    allDay: false,
    reminder: 0
  }
  selectedEvent.value = null
  showCalendarModal.value = true
}

const showEditEventModal = (event: any) => {
  selectedEvent.value = { ...event }
  formData.value = {
    title: event.title,
    description: event.description,
    start: formatDateForInput(new Date(event.start)),
    end: formatDateForInput(new Date(event.end)),
    allDay: event.allDay,
    reminder: event.reminder || 0
  }
  showCalendarModal.value = true
}

const closeCreateEventModal = () => {
  showCalendarModal.value = false
  formData.value = {
    title: '',
    description: '',
    start: formatDateForInput(new Date()),
    end: formatDateForInput(new Date()),
    allDay: false,
    reminder: 0
  }
  selectedEvent.value = null
}

const closeEditEventModal = () => {
  showCalendarModal.value = false
  selectedEvent.value = null
  formData.value = {
    title: '',
    description: '',
    start: formatDateForInput(new Date()),
    end: formatDateForInput(new Date()),
    allDay: false,
    reminder: 0
  }
}

const prevMonth = () => {
  selectedDate.value = new Date(selectedDate.value.getFullYear(), selectedDate.value.getMonth() - 1, 1)
}

const nextMonth = () => {
  selectedDate.value = new Date(selectedDate.value.getFullYear(), selectedDate.value.getMonth() + 1, 1)
}

const selectDate = (date: Date) => {
  selectedDate.value = date
}

const showReminderNotification = (event: any) => {
  if ('Notification' in window) {
    const start = new Date(event.start)
    const end = new Date(event.end)
    const timeStr = event.allDay ? '全天' : `${start.getHours().toString().padStart(2, '0')}:${start.getMinutes().toString().padStart(2, '0')} - ${end.getHours().toString().padStart(2, '0')}:${end.getMinutes().toString().padStart(2, '0')}`
    if (Notification.permission === 'granted') {
      new Notification('日历提醒', {
        body: `事件: ${event.title}\n时间: ${timeStr}\n描述: ${event.description || '无'}`,
        icon: generateAvatar('日历')
      })
    } else if (Notification.permission !== 'denied') {
      Notification.requestPermission().then(permission => {
        if (permission === 'granted') {
          new Notification('日历提醒', {
            body: `事件: ${event.title}\n时间: ${timeStr}\n描述: ${event.description || '无'}`,
            icon: generateAvatar('日历')
          })
        }
      })
    }
  }
  logger.log('提醒:', event.title)
}

const setEventReminder = (event: any) => {
  const reminder = Number(event.reminder)
  if (!reminder) return
  const eventDate = new Date(event.start)
  const reminderTime = eventDate.getTime() - (reminder * 60 * 1000)
  const now = Date.now()
  if (reminderTime > now) {
    const timerId = setTimeout(() => {
      showReminderNotification(event)
      reminderTimers.value.delete(event.id)
    }, reminderTime - now)
    reminderTimers.value.set(event.id, timerId)
  }
}

const clearEventReminder = (eventId: string) => {
  const timerId = reminderTimers.value.get(eventId)
  if (timerId) {
    clearTimeout(timerId)
    reminderTimers.value.delete(eventId)
  }
}

const setupAllReminders = () => {
  reminderTimers.value.forEach(timerId => clearTimeout(timerId))
  reminderTimers.value.clear()
  events.value.forEach(event => {
    setEventReminder(event)
  })
}

onMounted(async () => {
  await loadEvents()
  setupAllReminders()
  if ('Notification' in window && Notification.permission !== 'granted' && Notification.permission !== 'denied') {
    Notification.requestPermission()
  }
})

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

.current-date {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 1px;
}

.calendar-content {
  flex: 1;
  display: flex;
  overflow: hidden;
  background-color: var(--right-content-bg);
}

@media (max-width: 768px) {
  .calendar-content {
    flex-direction: column;
  }
}
</style>

<style>
.calendar-form-group {
  margin-bottom: 16px;
}

.calendar-form-group label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 6px;
}

.calendar-form-input,
.calendar-form-textarea,
.calendar-form-select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  color: var(--text-color);
  background-color: var(--input-bg);
  transition: all 0.15s;
  box-sizing: border-box;
}

.calendar-form-input:focus,
.calendar-form-textarea:focus,
.calendar-form-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.calendar-form-textarea {
  resize: vertical;
  min-height: 80px;
}

.calendar-checkbox-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.calendar-checkbox-group input[type="checkbox"] {
  cursor: pointer;
  width: 16px;
  height: 16px;
}

.calendar-checkbox-group label {
  margin-bottom: 0;
  cursor: pointer;
}

.calendar-modal-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.calendar-cancel-btn {
  background-color: var(--right-content-bg);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.calendar-cancel-btn:hover {
  background-color: var(--hover-color);
  color: var(--text-color);
  border-color: var(--primary-color);
}

.calendar-confirm-btn {
  background-color: var(--primary-color);
  color: white;
}

.calendar-confirm-btn:hover {
  background-color: var(--active-color);
  color: white;
}
</style>