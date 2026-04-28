<template>
  <div class="folder-tree-item">
    <!-- 文件夹行 -->
    <div
      class="folder-row"
      :class="{
        'folder-row--selected': isSelected,
        'folder-row--hovered': isHovered
      }"
      @click="handleClick"
      @mouseenter="isHovered = true"
      @mouseleave="isHovered = false"
    >
      <!-- 展开/收起箭头 -->
      <span
        v-if="isExpandableProp || folder.children?.length"
        class="folder-toggle"
        :class="{ 'folder-toggle--expanded': isExpandedState }"
        @click.stop="handleToggle"
      >
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="9 18 15 12 9 6" />
        </svg>
      </span>
      <!-- 占位，保持对齐 -->
      <span v-else class="folder-indent" />

      <!-- 文件夹图标 -->
      <span class="folder-icon">
        <svg v-if="isExpandedState" viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
          <path d="M20 6h-8l-2-2H4C2.9 4 2 4.9 2 6v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z" />
        </svg>
        <svg v-else viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
          <path d="M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z" />
        </svg>
      </span>

      <!-- 文件夹名称 -->
      <span class="folder-name" :title="folder.name">
        {{ folder.name }}
      </span>

      <!-- 操作按钮 -->
      <div class="folder-actions" v-if="isHovered">
        <button
          class="folder-action-btn"
          title="删除文件夹"
          @click.stop="$emit('delete', folder)"
        >
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="3 6 5 6 21 6" />
            <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2" />
          </svg>
        </button>
      </div>
    </div>

    <!-- 子文件夹列表（递归） -->
    <transition name="folder-expand">
      <div v-if="isExpandedState && folder.children && folder.children.length > 0" class="folder-children">
        <FolderTreeItem
          v-for="child in folder.children"
          :key="child.id"
          :folder="child"
          :expanded-ids="expandedIds"
          :selected-id="selectedId"
          :is-expandable-fn="isExpandableFn"
          :loading-ids="loadingIds"
          @toggle="$emit('toggle', $event)"
          @select="$emit('select', $event)"
          @delete="$emit('delete', $event)"
        />
      </div>
    </transition>

    <!-- 子文件夹加载状态 -->
    <div v-if="isExpandedState && isLoadingChildren" class="folder-loading">
      <span class="loading-dots">
        <span />
        <span />
        <span />
      </span>
    </div>

    <!-- 空子文件夹提示 -->
    <div v-if="isExpandedState && folder.children && folder.children.length === 0 && !isLoadingChildren" class="folder-empty">
      空文件夹
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { FolderNode } from '../../../composables/useFolderTree'

defineOptions({
  name: 'FolderTreeItem'
})

interface Props {
  folder: FolderNode
  expandedIds: Set<string>
  selectedId: string | null
  isExpandableFn: ((folder: FolderNode) => boolean) | null
  loadingIds: Set<string>
}

const props = withDefaults(defineProps<Props>(), {
  isExpandableFn: null
})

const emit = defineEmits<{
  (e: 'toggle', folder: FolderNode): void
  (e: 'select', folder: FolderNode): void
  (e: 'delete', folder: FolderNode): void
}>()

const isHovered = ref(false)

const isExpandedState = computed(() => props.expandedIds.has(props.folder.id))

const isSelected = computed(() => props.selectedId === props.folder.id)

const isLoadingChildren = computed(() => props.loadingIds.has(props.folder.id))

const isExpandableProp = computed(() =>
  props.isExpandableFn ? props.isExpandableFn(props.folder) : false
)

const handleClick = () => {
  emit('select', props.folder)
}

const handleToggle = () => {
  emit('toggle', props.folder)
}
</script>

<style scoped>
.folder-tree-item {
  user-select: none;
}

/* 文件夹行 */
.folder-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  cursor: pointer;
  border-radius: 6px;
  transition: background-color var(--transition-base), box-shadow var(--transition-base);
  position: relative;
}

.folder-row--hovered {
  background-color: var(--hover-color);
}

.folder-row--selected {
  background-color: var(--primary-light);
  box-shadow: inset 2px 0 0 var(--primary-color);
}

/* 展开/收起箭头 */
.folder-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  color: var(--text-secondary);
  transition: transform var(--transition-base), color var(--transition-base);
  flex-shrink: 0;
  border-radius: 4px;
}

.folder-toggle:hover {
  color: var(--primary-color);
  background-color: rgba(0, 0, 0, 0.05);
}

.folder-toggle--expanded {
  transform: rotate(90deg);
}

.folder-indent {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

/* 文件夹图标 */
.folder-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-warning-500);
  flex-shrink: 0;
  transition: color var(--transition-base);
}

.folder-row--selected .folder-icon {
  color: var(--primary-color);
}

/* 文件夹名称 */
.folder-name {
  font-size: var(--font-size-sm);
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
  min-width: 0;
}

.folder-row--selected .folder-name {
  font-weight: var(--font-weight-medium);
  color: var(--primary-dark);
}

/* 操作按钮 */
.folder-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.folder-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--transition-base);
}

.folder-action-btn:hover {
  background-color: var(--color-error-100);
  color: var(--color-error-500);
}

/* 子文件夹列表 */
.folder-children {
  margin-left: 18px;
  padding-left: 8px;
  border-left: 1px solid var(--border-color);
}

/* 加载状态 */
.folder-loading {
  padding: 8px 0 8px 36px;
  display: flex;
  align-items: center;
}

.loading-dots {
  display: flex;
  gap: 4px;
}

.loading-dots span {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background-color: var(--primary-color);
  animation: loading-bounce 1.4s ease-in-out infinite both;
}

.loading-dots span:nth-child(1) { animation-delay: 0s; }
.loading-dots span:nth-child(2) { animation-delay: 0.16s; }
.loading-dots span:nth-child(3) { animation-delay: 0.32s; }

@keyframes loading-bounce {
  0%, 80%, 100% {
    opacity: 0.3;
    transform: scale(0.8);
  }
  40% {
    opacity: 1;
    transform: scale(1);
  }
}

/* 空文件夹提示 */
.folder-empty {
  padding: 6px 0 6px 36px;
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  opacity: 0.6;
}

/* 展开/收起动画 */
.folder-expand-enter-active {
  transition: all var(--transition-slow);
  overflow: hidden;
}

.folder-expand-leave-active {
  transition: all var(--transition-base);
  overflow: hidden;
}

.folder-expand-enter-from {
  opacity: 0;
  transform: translateY(-6px);
}

.folder-expand-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
