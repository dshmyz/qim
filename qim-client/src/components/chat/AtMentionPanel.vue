<template>
  <div v-if="visible" class="at-mention-panel-wrapper" :style="panelWrapperStyle">
    <div class="at-mention-panel-backdrop" @click="handleClose"></div>
    <div
      ref="panelRef"
      class="at-mention-panel"
      role="listbox"
      aria-label="选择提及成员"
      @keydown="handleKeyDown"
    >
      <div class="at-mention-panel-header">
        <h4>选择成员</h4>
      </div>

      <div class="at-mention-panel-search">
        <input
          ref="searchInputRef"
          v-model="localSearchQuery"
          type="text"
          placeholder="搜索成员..."
          class="at-mention-search-input"
          @input="handleSearchInput"
        />
      </div>

      <div class="at-mention-panel-list" role="list">
        <div
          class="at-mention-item"
          :class="{ 'at-mention-item--active': activeIndex === -1 }"
          role="option"
          aria-selected="false"
          @click="handleSelectAll"
        >
          <img
            :src="generateAvatar('所有人')"
            alt="所有人"
            class="at-mention-item-avatar"
          />
          <span class="at-mention-item-name">所有人</span>
        </div>

        <div
          v-for="(member, index) in filteredMembers"
          :key="member.id"
          class="at-mention-item"
          :class="{ 'at-mention-item--active': activeIndex === index }"
          role="option"
          aria-selected="false"
          @click="handleSelectMember(member)"
        >
          <img
            :src="member.avatar"
            :alt="member.name || '未知用户'"
            class="at-mention-item-avatar"
          />
          <span class="at-mention-item-name">{{ member.name || '未知用户' }}</span>
        </div>

        <div v-if="filteredMembers.length === 0" class="at-mention-empty">
          <p>没有找到匹配的成员</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { generateAvatar } from '../../utils/avatar'

export interface MemberItem {
  id: string
  name: string
  avatar: string
}

export interface Position {
  x?: number
  y?: number
}

interface Props {
  members: MemberItem[]
  visible: boolean
  position?: Position
  searchQuery?: string
}

const props = withDefaults(defineProps<Props>(), {
  position: () => ({}),
  searchQuery: ''
})

const emit = defineEmits<{
  (e: 'select', member: MemberItem): void
  (e: 'selectAll'): void
  (e: 'close'): void
  (e: 'update:searchQuery', value: string): void
}>()

const panelRef = ref<HTMLDivElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const activeIndex = ref<number>(-1) // -1 represents "All" option

// 本地搜索 query
const localSearchQuery = ref(props.searchQuery)

// 监听 props 传入的 searchQuery 变化
watch(
  () => props.searchQuery,
  (newVal) => {
    localSearchQuery.value = newVal
  }
)

// 过滤后的成员列表
const filteredMembers = computed(() => {
  if (!props.members) return []
  if (!localSearchQuery.value) return props.members
  const query = localSearchQuery.value.toLowerCase()
  return props.members.filter(member =>
    member.name.toLowerCase().includes(query)
  )
})

// 面板位置样式
const panelWrapperStyle = computed(() => {
  const { x, y } = props.position
  if (x !== undefined && y !== undefined) {
    return {
      position: 'fixed' as const,
      left: `${x}px`,
      top: `${y}px`
    }
  }
  return {}
})

// 处理搜索输入
const handleSearchInput = () => {
  activeIndex.value = -1
  emit('update:searchQuery', localSearchQuery.value)
}

// 处理关闭
const handleClose = () => {
  emit('close')
}

// 处理选择成员
const handleSelectMember = (member: MemberItem) => {
  emit('select', member)
}

// 处理选择所有人
const handleSelectAll = () => {
  emit('selectAll')
}

// 键盘导航
const handleKeyDown = (event: KeyboardEvent) => {
  const totalOptions = filteredMembers.value.length + 1 // +1 for "All" option

  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      activeIndex.value = (activeIndex.value + 1) % totalOptions
      break
    case 'ArrowUp':
      event.preventDefault()
      activeIndex.value = (activeIndex.value - 1 + totalOptions) % totalOptions
      break
    case 'Enter':
      event.preventDefault()
      if (activeIndex.value === -1) {
        handleSelectAll()
      } else if (activeIndex.value >= 0 && activeIndex.value < filteredMembers.value.length) {
        handleSelectMember(filteredMembers.value[activeIndex.value])
      }
      break
    case 'Escape':
      event.preventDefault()
      handleClose()
      break
  }
}

// 面板可见时自动聚焦搜索框
watch(
  () => props.visible,
  async (newVal) => {
    if (newVal) {
      await nextTick()
      searchInputRef.value?.focus()
    }
  }
)
</script>

<style scoped>
.at-mention-panel-wrapper {
  position: absolute;
  z-index: 1000;
  width: 100%;
}

.at-mention-panel-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: transparent;
  z-index: -1;
}

.at-mention-panel {
  background: var(--list-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  max-height: 320px;
  overflow-y: auto;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  min-width: 220px;
  width: 100%;
}

.at-mention-panel-header {
  margin-bottom: 12px;
}

.at-mention-panel-header h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
}

.at-mention-panel-search {
  margin-bottom: 12px;
}

.at-mention-search-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 14px;
  box-sizing: border-box;
  transition: border-color 0.2s ease;
}

.at-mention-search-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.at-mention-search-input::placeholder {
  color: var(--text-color);
  opacity: 0.5;
}

.at-mention-panel-list {
  max-height: 220px;
  overflow-y: auto;
}

.at-mention-item {
  display: flex;
  align-items: center;
  padding: 8px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.15s ease;
  margin-bottom: 4px;
  user-select: none;
}

.at-mention-item:hover,
.at-mention-item--active {
  background: var(--hover-color);
}

.at-mention-item-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  margin-right: 10px;
  object-fit: cover;
  flex-shrink: 0;
}

.at-mention-item-name {
  font-size: 14px;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.at-mention-empty {
  padding: 24px 12px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 13px;
}

.at-mention-empty p {
  margin: 0;
}

/* 滚动条样式 */
.at-mention-panel::-webkit-scrollbar,
.at-mention-panel-list::-webkit-scrollbar {
  width: 6px;
}

.at-mention-panel::-webkit-scrollbar-track,
.at-mention-panel-list::-webkit-scrollbar-track {
  background: transparent;
}

.at-mention-panel::-webkit-scrollbar-thumb,
.at-mention-panel-list::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.at-mention-panel::-webkit-scrollbar-thumb:hover,
.at-mention-panel-list::-webkit-scrollbar-thumb:hover {
  background: var(--text-color);
  opacity: 0.3;
}

/* 暗色主题适配 */
[data-theme='dark'] .at-mention-panel {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
}
</style>
