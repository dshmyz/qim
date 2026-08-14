<template>
  <div v-if="visible" class="slash-cmd-panel" role="listbox" :aria-label="title">
    <div class="slash-cmd-panel__header">
      <span>{{ title }}</span>
    </div>
    <div ref="listRef" class="slash-cmd-panel__list" role="list">
      <div
        v-for="(item, index) in filteredItems"
        :key="item.id"
        class="slash-cmd-panel__item"
        :class="{ 'slash-cmd-panel__item--active': activeIndex === index }"
        role="option"
        :aria-selected="activeIndex === index"
        @click="handleSelect(item)"
        @mouseenter="activeIndex = index"
      >
        <component :is="itemComponent" :item="item" :active="activeIndex === index" />
      </div>
      <div v-if="filteredItems.length === 0" class="slash-cmd-panel__empty">
        <span v-if="items.length === 0">暂无可用项</span>
        <span v-else>没有匹配项</span>
      </div>
    </div>
    <div v-if="footerLabel" class="slash-cmd-panel__footer" @click="$emit('footer-action')">
      <i v-if="footerIcon" :class="footerIcon"></i>
      <span>{{ footerLabel }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, type Component } from 'vue'
import type { SlashCommandItem } from '../../utils/slashCommand'

const props = defineProps<{
  /** 面板标题 */
  title: string
  /** 全量候选项（已拉取） */
  items: SlashCommandItem[]
  /** 是否可见 */
  visible: boolean
  /** 搜索词 */
  searchQuery: string
  /** 渲染单条候选项的组件，props: { item, active } */
  itemComponent: Component
  /** 底部操作按钮文本（如"管理快速回复"）。可选 */
  footerLabel?: string
  /** 底部操作按钮图标 class。可选 */
  footerIcon?: string
}>()

const emit = defineEmits<{
  select: [item: SlashCommandItem]
  close: []
  'footer-action': []
}>()

const listRef = ref<HTMLDivElement | null>(null)
const activeIndex = ref(0)

/**
 * 按 searchQuery 过滤。
 * 简单的 includes 过滤（大小写不敏感）由各命令自行实现更复杂逻辑；
 * 这里作为通用兜底，调用方通常在传入 items 前已用 command.filter 过滤，
 * 本组件不再重复过滤，直接展示传入的 items。
 */
const filteredItems = computed(() => props.items as typeof props.items)

// 搜索词变化时重置高亮项
watch(() => props.searchQuery, () => {
  activeIndex.value = 0
})

// 面板打开/列表变化时重置
watch(() => props.visible, (v) => {
  if (v) activeIndex.value = 0
})

const scrollToActive = () => {
  nextTick(() => {
    const items = listRef.value?.querySelectorAll('.slash-cmd-panel__item')
    if (!items || !items[activeIndex.value]) return
    const el = items[activeIndex.value] as HTMLElement
    const container = listRef.value!
    const cRect = container.getBoundingClientRect()
    const eRect = el.getBoundingClientRect()
    if (eRect.top < cRect.top) {
      container.scrollTop += eRect.top - cRect.top
    } else if (eRect.bottom > cRect.bottom) {
      container.scrollTop += eRect.bottom - cRect.bottom
    }
  })
}

const handleSelect = (item: SlashCommandItem) => {
  emit('select', item)
}

// 键盘导航：由父组件在 textarea 的 keydown 里转发调用
// 返回 true 表示已处理（父组件应 preventDefault）
const handleKeyDown = (event: KeyboardEvent): boolean => {
  const len = filteredItems.value.length
  if (len === 0) {
    if (event.key === 'Escape') {
      emit('close')
      return true
    }
    return false
  }
  switch (event.key) {
    case 'ArrowDown':
      activeIndex.value = (activeIndex.value + 1) % len
      scrollToActive()
      return true
    case 'ArrowUp':
      activeIndex.value = (activeIndex.value - 1 + len) % len
      scrollToActive()
      return true
    case 'Enter':
      handleSelect(filteredItems.value[activeIndex.value])
      return true
    case 'Escape':
      emit('close')
      return true
  }
  return false
}

defineExpose({ handleKeyDown })
</script>

<style scoped>
.slash-cmd-panel {
  background: var(--list-bg, var(--card-bg, #fff));
  border: 1px solid var(--border-color, #e4e7ed);
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  overflow: hidden;
  max-width: 420px;
}

.slash-cmd-panel__header {
  padding: 8px 12px;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, var(--color-gray-500, #909399));
  border-bottom: 1px solid var(--border-color, #e4e7ed);
}

.slash-cmd-panel__list {
  max-height: 240px;
  overflow-y: auto;
  padding: 4px;
}

.slash-cmd-panel__item {
  display: flex;
  align-items: center;
  padding: 8px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.15s ease;
  user-select: none;
}

.slash-cmd-panel__item:hover,
.slash-cmd-panel__item--active {
  background: var(--hover-color, var(--color-gray-100, #f5f7fa));
}

.slash-cmd-panel__empty {
  padding: 20px 12px;
  text-align: center;
  color: var(--text-secondary, #909399);
  font-size: var(--font-size-xs);
}

.slash-cmd-panel__footer {
  padding: 8px 12px;
  border-top: 1px solid var(--border-color, #e4e7ed);
  color: var(--primary-color, #3385ff);
  font-size: var(--font-size-xxs);
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: background-color 0.15s ease;
}

.slash-cmd-panel__footer:hover {
  background: var(--hover-color, var(--color-gray-100, #f5f7fa));
}

.slash-cmd-panel__list::-webkit-scrollbar {
  width: 6px;
}
.slash-cmd-panel__list::-webkit-scrollbar-thumb {
  background: var(--border-color, #dcdfe6);
  border-radius: 3px;
}
</style>
