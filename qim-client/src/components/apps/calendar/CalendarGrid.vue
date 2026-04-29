<template>
  <div class="calendar-grid-panel">
    <div class="calendar-nav">
      <span class="current-month">{{ currentMonthLabel }}</span>
      <div class="nav-btns">
        <button class="nav-btn" @click="$emit('prevMonth')">‹</button>
        <button class="nav-btn" @click="$emit('nextMonth')">›</button>
      </div>
    </div>
    <div class="calendar-week-header">
      <div v-for="d in weekDays" :key="d" class="week-day">{{ d }}</div>
    </div>
    <div class="calendar-body">
      <div
        v-for="day in calendarDays"
        :key="day.key"
        class="calendar-cell"
        :class="cellClasses(day)"
        @click="$emit('selectDate', day.date)"
      >
        <div class="cell-date" :class="{ 'today-circle': day.isToday }">
          {{ day.date.getDate() }}
        </div>
        <div class="cell-info-row">
          <span
            class="lunar-text"
            :class="lunarClasses(day.lunar)"
          >{{ day.lunar.lunarDayName }}</span>
        </div>
        <div v-if="day.events.length > 0" class="cell-event-row">
          <span
            class="event-tag"
            :style="{ backgroundColor: eventColor(day.events[0]), color: eventTextColor(day.events[0]) }"
          >{{ day.events[0].title }}</span>
          <span v-if="day.events.length > 1" class="more-tag">+{{ day.events.length - 1 }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getLunarDayInfo, LunarDayInfo } from '../../../utils/lunar'

interface CalendarEvent {
  id: string
  title: string
  description: string
  start: string
  end: string
  allDay: boolean
  reminder: number
}

interface CalendarDay {
  key: string
  date: Date
  isCurrentMonth: boolean
  isToday: boolean
  isSelected: boolean
  events: CalendarEvent[]
  lunar: LunarDayInfo
}

const props = defineProps<{
  selectedDate: Date
  events: CalendarEvent[]
}>()

defineEmits<{
  prevMonth: []
  nextMonth: []
  selectDate: [date: Date]
}>()

const weekDays = ['日', '一', '二', '三', '四', '五', '六']

const currentMonthLabel = computed(() => {
  return props.selectedDate.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long' })
})

const calendarDays = computed<CalendarDay[]>(() => {
  const year = props.selectedDate.getFullYear()
  const month = props.selectedDate.getMonth()

  const firstDay = new Date(year, month, 1)
  const startDate = new Date(firstDay)
  startDate.setDate(startDate.getDate() - firstDay.getDay())

  const days: CalendarDay[] = []
  for (let i = 0; i < 42; i++) {
    const date = new Date(startDate)
    date.setDate(startDate.getDate() + i)

    const dayEvents = props.events.filter(event => {
      const eventDate = new Date(event.start)
      return eventDate.getFullYear() === date.getFullYear() &&
             eventDate.getMonth() === date.getMonth() &&
             eventDate.getDate() === date.getDate()
    })

    const lunar = getLunarDayInfo(date.getFullYear(), date.getMonth() + 1, date.getDate())

    days.push({
      key: date.toISOString(),
      date,
      isCurrentMonth: date.getMonth() === month,
      isToday: date.toDateString() === new Date().toDateString(),
      isSelected: date.toDateString() === props.selectedDate.toDateString(),
      events: dayEvents,
      lunar,
    })
  }

  return days
})

function cellClasses(day: CalendarDay) {
  return {
    'other-month': !day.isCurrentMonth,
    'is-today': day.isToday,
    'is-selected': day.isSelected,
    'is-festival': day.lunar.isFestival,
    'has-events': day.events.length > 0,
  }
}

function lunarClasses(lunar: LunarDayInfo) {
  return {
    'festival-text': lunar.isFestival,
    'solar-term-text': lunar.isSolarTerm,
  }
}

function eventColor(event: CalendarEvent) {
  if (event.allDay) return 'rgba(38,179,97,0.12)'
  return 'var(--primary-light)'
}

function eventTextColor(event: CalendarEvent) {
  if (event.allDay) return '#26b361'
  return 'var(--primary-color)'
}
</script>

<style scoped>
.calendar-grid-panel {
  flex: 5;
  padding: 24px;
  background: var(--card-bg);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  border-right: 1px solid var(--border-color);
}

.calendar-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.current-month {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-color);
}

.nav-btns {
  display: flex;
  gap: 6px;
}

.nav-btn {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: var(--hover-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  border: 1px solid var(--border-color);
  transition: background 0.2s;
}

.nav-btn:hover {
  background: var(--primary-light);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.calendar-week-header {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
  margin-bottom: 4px;
}

.week-day {
  text-align: center;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  padding: 5px 0;
}

.calendar-body {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
  flex: 1;
}

.calendar-cell {
  min-height: 46px;
  padding: 4px 3px;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  cursor: pointer;
  transition: background 0.15s;
  border: 1px solid transparent;
}

.calendar-cell:hover {
  background: var(--hover-color);
}

.calendar-cell.other-month {
  opacity: 0.4;
}

.calendar-cell.other-month .lunar-text,
.calendar-cell.other-month .cell-date {
  color: var(--text-secondary);
}

.calendar-cell.is-today .cell-date {
  background: var(--primary-color);
  color: #fff;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
}

.calendar-cell.is-today .lunar-text,
.calendar-cell.is-today .event-tag,
.calendar-cell.is-today .more-tag {
  color: var(--primary-color);
}

.calendar-cell.is-selected {
  background: var(--primary-light);
  border-color: rgba(59, 130, 246, 0.2);
}

.calendar-cell.is-selected .cell-date {
  color: var(--primary-color);
  font-weight: 500;
}

.calendar-cell.is-selected .lunar-text,
.calendar-cell.is-selected .event-tag {
  color: var(--primary-color);
}

.calendar-cell.is-festival {
  background: #fef2f2;
}

.calendar-cell.is-festival .cell-date {
  color: #f34040;
  font-weight: 500;
}

.calendar-cell.is-festival .lunar-text {
  color: #f34040;
  font-weight: 500;
}

.cell-date {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-color);
  line-height: 1;
}

.cell-info-row {
  margin-top: 2px;
  line-height: 1;
}

.lunar-text {
  font-size: 9px;
  color: var(--text-secondary);
}

.lunar-text.festival-text {
  color: #f34040;
  font-weight: 500;
}

.lunar-text.solar-term-text {
  color: #26b361;
  font-weight: 500;
}

.cell-event-row {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-top: 2px;
  line-height: 1;
  flex-wrap: wrap;
}

.event-tag {
  font-size: 9px;
  padding: 0 2px;
  border-radius: 1px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 80%;
}

.more-tag {
  font-size: 9px;
  color: var(--text-secondary);
}
</style>