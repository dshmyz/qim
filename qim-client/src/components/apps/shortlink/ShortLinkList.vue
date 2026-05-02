<template>
  <div class="short-link-list-container">
    <div class="list-header">
      <h3>我的短链接</h3>
      <div class="list-actions">
        <button class="list-action-btn" @click="$emit('export')">
          <i class="fas fa-download"></i> 导出
        </button>
        <button
          :class="['list-action-btn', { active: selectMode }]"
          @click="toggleSelectMode"
        >
          <i class="fas fa-check-square"></i>
          {{ selectMode ? '取消选择' : '批量操作' }}
        </button>
      </div>
    </div>

    <!-- 批量操作工具栏 -->
    <div v-if="selectMode" class="batch-toolbar">
      <div class="batch-left">
        <label class="select-all">
          <input
            type="checkbox"
            :checked="isAllSelected"
            :indeterminate.prop="isIndeterminate"
            @change="toggleSelectAll"
          />
          <span>全选</span>
        </label>
        <span class="selected-count">已选择 {{ selectedIds.length }} 项</span>
      </div>
      <div class="batch-right">
        <button
          class="batch-btn batch-btn--export"
          :disabled="selectedIds.length === 0"
          @click="handleExportSelected"
        >
          <i class="fas fa-download"></i> 导出选中
        </button>
        <button
          class="batch-btn batch-btn--delete"
          :disabled="selectedIds.length === 0"
          @click="handleBatchDelete"
        >
          <i class="fas fa-trash"></i> 批量删除
        </button>
      </div>
    </div>

    <div class="list-controls">
      <SearchBar v-model="searchQuery" @search="handleSearch" />
      <FilterDropdown
        v-model="selectedFilter"
        :options="filterOptions"
        @change="handleFilterChange"
      />
      <FilterDropdown
        v-model="selectedSort"
        :options="sortOptions"
        @change="handleSortChange"
      />
    </div>

    <div class="short-link-list">
      <ShortLinkItem
        v-for="link in filteredLinks"
        :key="link.id"
        :link="link"
        :select-mode="selectMode"
        :is-selected="selectedIds.includes(link.id)"
        @copy="handleCopy"
        @delete="handleDelete"
        @select-change="handleSelectChange"
      />

      <div v-if="filteredLinks.length === 0" class="empty-state">
        <div class="empty-icon"><i class="fas fa-link"></i></div>
        <p>暂无短链接</p>
        <p class="empty-hint">生成你的第一个短链接吧</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import SearchBar from './SearchBar.vue'
import FilterDropdown from './FilterDropdown.vue'
import ShortLinkItem, { type ShortLink } from './ShortLinkItem.vue'

const props = defineProps<{
  links: ShortLink[]
}>()

const emit = defineEmits<{
  copy: [url: string]
  delete: [id: number]
  export: []
  'export-selected': [ids: number[]]
  'batch-delete': [ids: number[]]
}>()

const searchQuery = ref('')
const selectedFilter = ref('all')
const selectedSort = ref('created')
const selectMode = ref(false)
const selectedIds = ref<number[]>([])

const filterOptions = [
  { label: '全部状态', value: 'all' },
  { label: '活跃', value: 'active' },
  { label: '未访问', value: 'inactive' }
]

const sortOptions = [
  { label: '按创建时间', value: 'created' },
  { label: '按访问量', value: 'visits' },
  { label: '按最近访问', value: 'recent' }
]

const filteredLinks = computed(() => {
  let result = [...props.links]

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(link =>
      link.original_url.toLowerCase().includes(query) ||
      link.short_url.toLowerCase().includes(query)
    )
  }

  if (selectedFilter.value === 'active') {
    result = result.filter(link => link.visit_count > 0)
  } else if (selectedFilter.value === 'inactive') {
    result = result.filter(link => link.visit_count === 0)
  }

  if (selectedSort.value === 'visits') {
    result.sort((a, b) => b.visit_count - a.visit_count)
  } else if (selectedSort.value === 'created') {
    result.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
  }

  return result
})

// 是否全选
const isAllSelected = computed(() => {
  return filteredLinks.value.length > 0 && selectedIds.value.length === filteredLinks.value.length
})

// 是否部分选中
const isIndeterminate = computed(() => {
  return selectedIds.value.length > 0 && selectedIds.value.length < filteredLinks.value.length
})

// 切换选择模式
const toggleSelectMode = () => {
  selectMode.value = !selectMode.value
  if (!selectMode.value) {
    selectedIds.value = []
  }
}

// 全选/取消全选
const toggleSelectAll = () => {
  if (isAllSelected.value) {
    selectedIds.value = []
  } else {
    selectedIds.value = filteredLinks.value.map(link => link.id)
  }
}

// 处理单项选择变化
const handleSelectChange = (id: number, selected: boolean) => {
  if (selected) {
    if (!selectedIds.value.includes(id)) {
      selectedIds.value.push(id)
    }
  } else {
    selectedIds.value = selectedIds.value.filter(i => i !== id)
  }
}

// 导出选中项
const handleExportSelected = () => {
  emit('export-selected', selectedIds.value)
}

// 批量删除
const handleBatchDelete = () => {
  emit('batch-delete', selectedIds.value)
}

const handleSearch = (query: string) => {
  console.log('Search:', query)
}

const handleFilterChange = (filter: string) => {
  console.log('Filter:', filter)
}

const handleSortChange = (sort: string) => {
  console.log('Sort:', sort)
}

const handleCopy = (url: string) => {
  emit('copy', url)
}

const handleDelete = (id: number) => {
  emit('delete', id)
}

// 暴露方法供父组件调用
defineExpose({
  getSelectedIds: () => selectedIds.value,
  clearSelection: () => {
    selectedIds.value = []
    selectMode.value = false
  }
})
</script>

<style scoped>
.short-link-list-container {
  background: var(--card-bg, white);
  border-radius: 16px;
  box-shadow: 0 1px 3px var(--shadow-color, rgba(0, 0, 0, 0.1));
  overflow: hidden;
  border: 1px solid var(--border-color, transparent);
}

.list-header {
  padding: 20px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.list-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary, #1f2937);
}

.list-actions {
  display: flex;
  gap: 8px;
}

.list-action-btn {
  padding: 6px 12px;
  background: var(--button-bg, #f3f4f6);
  color: var(--text-secondary, #374151);
  border: none;
  border-radius: 8px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 4px;
}

.list-action-btn:hover {
  background: var(--button-bg-hover, #e5e7eb);
}

.list-action-btn.active {
  background: var(--primary-color, #667eea);
  color: white;
}

.batch-toolbar {
  padding: 12px 20px;
  background: var(--color-gray-50, #f9fafb);
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.batch-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.select-all {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-color);
}

.select-all input {
  cursor: pointer;
}

.selected-count {
  font-size: 13px;
  color: var(--text-secondary);
}

.batch-right {
  display: flex;
  gap: 8px;
}

.batch-btn {
  padding: 6px 12px;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 4px;
}

.batch-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.batch-btn--export {
  background: var(--accent-bg, rgba(102, 126, 234, 0.1));
  color: var(--accent-color, #667eea);
}

.batch-btn--export:hover:not(:disabled) {
  background: var(--accent-bg-hover, rgba(102, 126, 234, 0.2));
}

.batch-btn--delete {
  background: var(--danger-bg, rgba(239, 68, 68, 0.1));
  color: var(--danger-color, #ef4444);
}

.batch-btn--delete:hover:not(:disabled) {
  background: var(--danger-bg-hover, rgba(239, 68, 68, 0.2));
}

.list-controls {
  padding: 16px 20px;
  display: flex;
  gap: 12px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
}

.short-link-list {
  max-height: 400px;
  overflow-y: auto;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
  color: var(--text-secondary, #6b7280);
}

.empty-state p {
  margin: 8px 0;
  color: var(--text-secondary, #6b7280);
  font-size: 14px;
}

.empty-hint {
  font-size: 12px !important;
  opacity: 0.8;
}
</style>
