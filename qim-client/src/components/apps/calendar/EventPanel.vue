<template>
  <div class="event-panel">
    <div class="event-panel-header">
      <div>
        <div class="event-panel-date">{{ dateLabel }}</div>
        <div class="event-panel-count">{{ events.length > 0 ? `${events.length}个事件` : '无事件' }}</div>
      </div>
    </div>

    <div v-if="events.length === 0" class="event-empty">
      <div class="empty-icon">📅</div>
      <div class="empty-text">没有事件</div>
    </div>

    <div v-else class="event-list">
      <div
        v-for="event in events"
        :key="event.id"
        class="event-card"
        @click="$emit('editEvent', event)"
      >
        <div class="event-card-inner">
          <div class="event-color-bar" :style="{ backgroundColor: event.allDay ? '#26b361' : 'var(--primary-color)' }"></div>
          <div class="event-card-content">
            <div class="event-card-title">{{ event.title }}</div>
            <div class="event-card-time">{{ formatEventTime(event) }}</div>
            <div v-if="event.description" class="event-card-desc">{{ event.description }}</div>
          </div>
          <button class="event-delete-btn" @click.stop="$emit('deleteEvent', event.id)">
            ×
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getLunarDayInfo } from '../../../utils/lunar'

interface CalendarEvent {
  id: string
  title: string
  description: string
  start: string
  end: string
  allDay: boolean
  reminder: number
}

const props = defineProps<{
  selectedDate: Date
  events: CalendarEvent[]
}>()

defineEmits<{
  editEvent: [event: CalendarEvent]
  deleteEvent: [eventId: string]
}>()

const dateLabel = computed(() => {
  const date = props.selectedDate
  const lunar = getLunarDayInfo(date.getFullYear(), date.getMonth() + 1, date.getDate())
  const dateStr = date.toLocaleDateString('zh-CN', { month: 'long', day: 'numeric', weekday: 'short' })
  return `${dateStr} · ${lunar.lunarDayName}`
})

function formatEventTime(event: CalendarEvent) {
  if (event.allDay) return '全天'
  const start = new Date(event.start)
  const end = new Date(event.end)
  const startStr = `${start.getHours().toString().padStart(2, '0')}:${start.getMinutes().toString().padStart(2, '0')}`
  const endStr = `${end.getHours().toString().padStart(2, '0')}:${end.getMinutes().toString().padStart(2, '0')}`
  return `${startStr} - ${endStr}`
}
</script>

<style scoped>
.event-panel {
  flex: 2;
  padding: 20px 16px;
  background: var(--card-bg);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.event-panel-header {
  margin-bottom: 16px;
}

.event-panel-date {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-color);
}

.event-panel-count {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 3px;
}

.event-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  font-size: 12px;
}

.empty-icon {
  font-size: 24px;
  opacity: 0.4;
  margin-bottom: 6px;
  text-align: center;
}

.empty-text {
  text-align: center;
}

.event-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.event-card {
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 12px;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.event-card:hover {
  border-color: var(--primary-color);
  background: var(--hover-color);
}

.event-card-inner {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.event-color-bar {
  width: 4px;
  height: 40px;
  border-radius: 2px;
  flex-shrink: 0;
}

.event-card-content {
  flex: 1;
  min-width: 0;
}

.event-card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-color);
}

.event-card-time {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 3px;
}

.event-card-desc {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.event-delete-btn {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 12px;
  border-radius: 4px;
  border: none;
  background: transparent;
  transition: background 0.15s, color 0.15s;
  flex-shrink: 0;
}

.event-delete-btn:hover {
  background: var(--hover-color);
  color: #f34040;
}
</style>