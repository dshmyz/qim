<template>
  <div
    v-if="visible"
    class="ai-context-menu"
    :style="{ left: position.x + 'px', top: position.y + 'px' }"
    @click.stop
  >
    <!-- AI 相关操作 -->
    <div
      v-for="item in menuItems"
      :key="item.id"
      class="context-menu-item"
      @click="handleSelect(item.id)"
    >
      <span class="context-menu-icon">
        <i :class="item.iconClass"></i>
      </span>
      <span>{{ item.label }}</span>
    </div>

    <!-- 分隔线 -->
    <div class="context-menu-divider"></div>

    <!-- 基础操作 -->
    <div class="context-menu-item" @click="handleCopyMessage">
      <span class="context-menu-icon"><i class="fas fa-copy"></i></span>
      <span>复制</span>
    </div>
    <div class="context-menu-item" @click="handleQuoteMessage">
      <span class="context-menu-icon"><i class="fas fa-quote-right"></i></span>
      <span>引用</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Message } from '../../types'

interface Props {
  visible: boolean
  position: { x: number; y: number }
  message: Message | null
}

interface MenuItem {
  id: string
  label: string
  iconClass: string
}

interface Emits {
  (e: 'select', actionId: string, message: Message | null): void
  (e: 'close'): void
  (e: 'copy-message'): void
  (e: 'quote-message'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const menuItems = computed<MenuItem[]>(() => [
  {
    id: 'ai_summary',
    label: 'AI 总结此消息',
    iconClass: 'fas fa-robot'
  },
  {
    id: 'translate',
    label: '翻译为中文',
    iconClass: 'fas fa-language'
  },
  {
    id: 'rewrite',
    label: '改写文本',
    iconClass: 'fas fa-pen-fancy'
  },
  {
    id: 'polish',
    label: '润色表达',
    iconClass: 'fas fa-sparkles'
  }
])

const handleSelect = (actionId: string) => {
  emit('select', actionId, props.message)
  emit('close')
}

const handleCopyMessage = () => {
  emit('copy-message')
  emit('close')
}

const handleQuoteMessage = () => {
  emit('quote-message')
  emit('close')
}
</script>

<style scoped>
.ai-context-menu {
  position: fixed;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 6px 0;
  z-index: 3000;
  min-width: 180px;
}

.context-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 0.15s;
  font-size: 14px;
  color: var(--text-color);
}

.context-menu-item:hover {
  background: var(--hover-color);
}

.context-menu-icon {
  width: 16px;
  text-align: center;
  font-size: 14px;
  color: var(--primary-color);
}

.context-menu-divider {
  height: 1px;
  background: var(--border-color);
  margin: 4px 0;
  padding: 0;
}
</style>
