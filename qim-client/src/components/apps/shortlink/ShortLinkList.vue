<template>
  <div class="short-link-list-container">
    <div class="list-header">
      <h3>我的短链接</h3>
      <div class="list-actions">
        <button class="list-action-btn" @click="$emit('export')">导出</button>
        <button class="list-action-btn" @click="$emit('batch-delete')">批量操作</button>
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
        @copy="handleCopy"
        @delete="handleDelete"
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
  'batch-delete': []
}>()

const searchQuery = ref('')
const selectedFilter = ref('all')
const selectedSort = ref('created')

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
}

.list-action-btn:hover {
  background: var(--button-bg-hover, #e5e7eb);
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
