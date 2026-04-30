<template>
  <div class="sticky-tag-filter" v-if="allTags.length > 0">
    <div class="filter-header">
      <span class="filter-label">标签筛选</span>
      <button v-if="selectedTag" class="clear-btn" @click="$emit('clear')">
        清除
      </button>
    </div>
    <div class="tag-list">
      <span
        v-for="tag in allTags"
        :key="tag"
        :class="['tag-item', { active: selectedTag === tag }]"
        @click="$emit('select', tag)"
      >
        {{ tag }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  allTags: string[]
  selectedTag: string | null
}>()

defineEmits<{
  select: [tag: string]
  clear: []
}>()
</script>

<style scoped>
.sticky-tag-filter {
  padding: var(--spacing-2) var(--spacing-3);
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
}

.filter-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--spacing-1);
}

.filter-label {
  font-size: 10px;
  font-weight: var(--font-weight-semibold);
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.clear-btn {
  font-size: 10px;
  font-weight: var(--font-weight-medium);
  color: var(--primary-color);
  background: var(--primary-light);
  border: none;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}

.clear-btn:hover {
  background: var(--primary-color);
  color: white;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.tag-item {
  font-size: 10px;
  padding: 1px 6px;
  background: var(--btn-bg);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: var(--font-weight-medium);
}

.tag-item:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  background: var(--primary-light);
}

.tag-item.active {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}
</style>
