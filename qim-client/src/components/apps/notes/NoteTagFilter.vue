<template>
  <div class="tag-filter" v-if="allTags.length > 0">
    <div class="tag-filter-header">
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
.tag-filter {
  padding: 12px;
  border-bottom: 1px solid var(--border-color);
}

.tag-filter-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.filter-label {
  font-size: 12px;
  color: var(--text-secondary);
}

.clear-btn {
  font-size: 12px;
  color: var(--primary-color);
  background: none;
  border: none;
  cursor: pointer;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tag-item {
  font-size: 12px;
  padding: 4px 10px;
  background: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.tag-item:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.tag-item.active {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}
</style>
