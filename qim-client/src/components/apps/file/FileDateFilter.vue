<template>
  <div class="file-date-filter" ref="dropdownRef">
    <button
      :class="['date-filter-btn', { active: isActive }]"
      @click="toggleDropdown"
    >
      <i class="fas fa-calendar-alt date-icon"></i>
      <span>{{ displayLabel }}</span>
      <i class="fas fa-chevron-down arrow-icon"></i>
    </button>

    <div v-if="showDropdown" class="date-dropdown">
      <div class="date-preset-list">
        <button
          v-for="preset in presets"
          :key="preset.value"
          :class="['preset-item', { active: selectedPreset === preset.value }]"
          @click="handlePresetSelect(preset)"
        >
          <i :class="preset.icon"></i>
          <span>{{ preset.label }}</span>
        </button>
      </div>

      <div class="date-custom">
        <div class="date-custom-title">自定义日期范围</div>
        <div class="date-range-inputs">
          <div class="date-field">
            <label>开始日期</label>
            <input
              type="date"
              :value="customFrom"
              @input="handleCustomFromChange"
              class="date-input"
            />
          </div>
          <div class="date-field">
            <label>结束日期</label>
            <input
              type="date"
              :value="customTo"
              @input="handleCustomToChange"
              class="date-input"
            />
          </div>
        </div>
        <button
          class="apply-custom-btn"
          :disabled="!customFrom || !customTo"
          @click="handleApplyCustom"
        >
          应用
        </button>
      </div>

      <div v-if="isActive" class="date-clear">
        <button class="clear-range-btn" @click="handleClear">
          <i class="fas fa-times"></i>
          <span>清除日期筛选</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'

defineOptions({
  name: 'FileDateFilter'
})

interface Props {
  dateFrom: string
  dateTo: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'change', from: string, to: string): void
  (e: 'clear'): void
}>()

const showDropdown = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const selectedPreset = ref<string>('')
const customFrom = ref('')
const customTo = ref('')

const presets = [
  { value: 'today', label: '今天', icon: 'fas fa-clock' },
  { value: '7days', label: '最近 7 天', icon: 'fas fa-calendar-day' },
  { value: '30days', label: '最近 30 天', icon: 'fas fa-calendar-week' },
  { value: '90days', label: '最近 90 天', icon: 'fas fa-calendar' }
]

const isActive = computed(() => props.dateFrom !== '' || props.dateTo !== '')

const displayLabel = computed(() => {
  if (!isActive.value) return '日期范围'
  if (selectedPreset.value) {
    const preset = presets.find(p => p.value === selectedPreset.value)
    return preset?.label ?? '日期范围'
  }
  return `${props.dateFrom} ~ ${props.dateTo}`
})

const formatDate = (date: Date): string => {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

const getDateRange = (preset: string): { from: string; to: string } => {
  const now = new Date()
  const to = formatDate(now)
  let from = to

  switch (preset) {
    case 'today':
      from = to
      break
    case '7days': {
      const d = new Date(now)
      d.setDate(d.getDate() - 6)
      from = formatDate(d)
      break
    }
    case '30days': {
      const d = new Date(now)
      d.setDate(d.getDate() - 29)
      from = formatDate(d)
      break
    }
    case '90days': {
      const d = new Date(now)
      d.setDate(d.getDate() - 89)
      from = formatDate(d)
      break
    }
  }

  return { from, to }
}

const toggleDropdown = () => {
  showDropdown.value = !showDropdown.value
}

const handlePresetSelect = (preset: { value: string }) => {
  selectedPreset.value = preset.value
  const { from, to } = getDateRange(preset.value)
  customFrom.value = from
  customTo.value = to
  emit('change', from, to)
  showDropdown.value = false
}

const handleCustomFromChange = (event: Event) => {
  customFrom.value = (event.target as HTMLInputElement).value
  selectedPreset.value = ''
}

const handleCustomToChange = (event: Event) => {
  customTo.value = (event.target as HTMLInputElement).value
  selectedPreset.value = ''
}

const handleApplyCustom = () => {
  if (customFrom.value && customTo.value) {
    emit('change', customFrom.value, customTo.value)
    showDropdown.value = false
  }
}

const handleClear = () => {
  selectedPreset.value = ''
  customFrom.value = ''
  customTo.value = ''
  emit('clear')
  showDropdown.value = false
}

const handleClickOutside = (event: MouseEvent) => {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    showDropdown.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.file-date-filter {
  position: relative;
}

.date-filter-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 8px;
  height: 28px;
  border: 1px solid var(--border-color, #e8ecf0);
  background: var(--card-bg, #fff);
  border-radius: 16px;
  cursor: pointer;
  color: var(--text-secondary, #8c95a6);
  font-size: 12px;
  font-weight: 500;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.date-filter-btn:hover {
  border-color: var(--primary-color, #4f6ef7);
  color: var(--primary-color, #4f6ef7);
  background: rgba(79, 110, 247, 0.04);
}

.date-filter-btn.active {
  border-color: var(--primary-color, #4f6ef7);
  color: var(--primary-color, #4f6ef7);
  background: rgba(79, 110, 247, 0.06);
}

.date-icon {
  font-size: 11px;
}

.arrow-icon {
  font-size: 9px;
  transition: transform 0.2s ease;
}

.date-dropdown {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  background: var(--card-bg, #fff);
  border: 1px solid var(--border-color, #e8ecf0);
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.06);
  padding: 6px;
  min-width: 220px;
  z-index: 1000;
  animation: dropdownIn 0.15s ease;
}

@keyframes dropdownIn {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.date-preset-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--border-color, #e8ecf0);
  margin-bottom: 6px;
}

.preset-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  color: var(--text-color, #4a5568);
  font-size: 13px;
  transition: all 0.15s ease;
  text-align: left;
  width: 100%;
}

.preset-item:hover {
  background: var(--hover-color, #f0f2f5);
  color: var(--primary-color, #4f6ef7);
}

.preset-item.active {
  background: rgba(79, 110, 247, 0.08);
  color: var(--primary-color, #4f6ef7);
  font-weight: 500;
}

.preset-item i {
  width: 16px;
  text-align: center;
  font-size: 13px;
}

.date-custom {
  padding: 6px 0;
}

.date-custom-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary, #8c95a6);
  padding: 0 12px 8px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.date-range-inputs {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 0 12px;
}

.date-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.date-field label {
  font-size: 11px;
  color: var(--text-secondary, #8c95a6);
}

.date-input {
  padding: 6px 8px;
  border: 1px solid var(--border-color, #e8ecf0);
  border-radius: 6px;
  font-size: 13px;
  color: var(--text-color, #4a5568);
  background: var(--card-bg, #fff);
  outline: none;
  transition: border-color 0.2s ease;
}

.date-input:focus {
  border-color: var(--primary-color, #4f6ef7);
}

.apply-custom-btn {
  width: calc(100% - 24px);
  margin: 8px 12px 0;
  padding: 6px 0;
  background: var(--primary-color, #4f6ef7);
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.apply-custom-btn:hover:not(:disabled) {
  background: var(--primary-hover, #3d5ce0);
}

.apply-custom-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.date-clear {
  padding-top: 6px;
  border-top: 1px solid var(--border-color, #e8ecf0);
  margin-top: 6px;
}

.clear-range-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  color: var(--error-color, #e53e3e);
  font-size: 13px;
  transition: all 0.15s ease;
  text-align: left;
}

.clear-range-btn:hover {
  background: rgba(229, 62, 62, 0.06);
}
</style>
