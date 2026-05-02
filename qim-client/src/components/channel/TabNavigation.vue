<!--
  TabNavigation.vue - 标签页导航组件

  功能：
  - 显示打开的频道标签页
  - 支持标签页切换
  - 支持关闭标签页
  - 支持添加新标签页

  使用示例：
  <TabNavigation
    :tabs="openTabs"
    :active-tab-id="selectedChannelId"
    @select="handleSelectTab"
    @close="handleCloseTab"
    @add="handleAddTab"
  />
-->
<template>
  <div class="tab-navigation" role="tablist" aria-label="频道标签页">
    <!-- 标签页列表 -->
    <div class="tab-list">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        class="tab-item"
        :class="{ active: tab.id === activeTabId }"
        role="tab"
        :aria-selected="tab.id === activeTabId"
        :aria-label="`频道 ${tab.name}`"
        tabindex="0"
        @click="handleSelect(tab.id)"
        @keydown.enter="handleSelect(tab.id)"
        @keydown.space.prevent="handleSelect(tab.id)"
      >
        <span class="tab-name">{{ tab.name }}</span>
        <button
          class="tab-close-btn"
          :aria-label="`关闭 ${tab.name}`"
          title="关闭标签页"
          @click.stop="handleClose(tab.id)"
        >
          <i class="fas fa-times"></i>
        </button>
      </div>

      <!-- 添加按钮 -->
      <button
        class="tab-add-btn"
        aria-label="添加新标签页"
        title="添加新标签页"
        @click="handleAdd"
      >
        <i class="fas fa-plus"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Tab {
  id: string | number
  name: string
}

interface Props {
  tabs: Tab[]
  activeTabId: string | number | null
}

defineProps<Props>()

const emit = defineEmits<{
  select: [tabId: string | number]
  close: [tabId: string | number]
  add: []
}>()

const handleSelect = (tabId: string | number) => {
  emit('select', tabId)
}

const handleClose = (tabId: string | number) => {
  emit('close', tabId)
}

const handleAdd = () => {
  emit('add')
}
</script>

<style scoped>
.tab-navigation {
  display: flex;
  align-items: center;
  width: 100%;
  background: var(--card-bg);
  border-bottom: 1px solid var(--border-color);
  height: 72px;
  padding: 0 var(--spacing-4);
  box-sizing: border-box;
}

.tab-list {
  display: flex;
  align-items: center;
  gap: 0;
  overflow-x: auto;
  flex: 1;
  height: 100%;
}

/* 隐藏滚动条但保持滚动功能 */
.tab-list::-webkit-scrollbar {
  display: none;
}

.tab-list {
  -ms-overflow-style: none;
  scrollbar-width: none;
}

.tab-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: 0 var(--spacing-4);
  background: transparent;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
  flex-shrink: 0;
  max-width: 200px;
  height: 100%;
  color: var(--text-secondary);
}

.tab-item:hover {
  color: var(--text-color);
  background: var(--color-gray-50);
}

.tab-item.active {
  color: var(--primary-color);
  border-bottom-color: var(--primary-color);
  background: transparent;
}

.tab-item:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: -2px;
}

.tab-item.active:focus {
  outline-color: var(--primary-color);
}

.tab-name {
  font-size: 14px;
  font-weight: var(--font-weight-medium);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tab-close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  opacity: 0;
  transition: all var(--transition-fast);
  flex-shrink: 0;
  color: var(--text-secondary);
}

.tab-item:hover .tab-close-btn {
  opacity: 0.6;
}

.tab-close-btn:hover {
  opacity: 1 !important;
  background: var(--color-gray-200);
}

.tab-item.active .tab-close-btn {
  color: var(--primary-color);
}

.tab-close-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 1px;
  opacity: 1;
}

.tab-add-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  flex-shrink: 0;
  margin-left: var(--spacing-2);
}

.tab-add-btn:hover {
  color: var(--primary-color);
  background: var(--primary-light);
}

.tab-add-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .tab-navigation {
    height: 60px;
    padding: 0 var(--spacing-3);
  }

  .tab-item {
    padding: 0 var(--spacing-3);
    max-width: 150px;
  }

  .tab-name {
    font-size: 13px;
  }

  .tab-add-btn {
    width: 28px;
    height: 28px;
  }
}
</style>
