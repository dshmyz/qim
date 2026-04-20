<template>
  <div class="apps-container">
    <div class="app-tabs">
      <div
        class="app-tab-item"
        :class="{ active: activeAppTab === 'categories' }"
        @click="handleTabClick('categories')"
      >
        <div class="tab-icon"><i class="fas fa-th-large"></i></div>
        <span class="tab-name">应用分类</span>
      </div>
    </div>

    <div v-if="activeAppTab === 'categories'" class="app-tab-content">
      <div class="app-categories">
        <div
          v-for="category in appCategories"
          :key="category.id"
          class="app-category-item"
        >
          <div class="category-header" @click="toggleCategory(category.id)">
            <span class="category-icon"><i :class="category.icon || 'fas fa-folder'"></i></span>
            <span class="category-name">{{ category.name }}</span>
            <span class="category-toggle">{{ category.expanded ? '▼' : '▶' }}</span>
          </div>
          <div v-if="category.expanded" class="category-apps">
            <div
              v-for="app in category.apps"
              :key="app.id"
              class="category-app-item"
              @click="handleAppClick(app)"
            >
              <div class="category-app-icon"><i :class="app.icon"></i></div>
              <span class="category-app-name">{{ app.name }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

interface AppCategory {
  id: string
  name: string
  icon?: string
  expanded: boolean
  apps: any[]
}

interface Props {
  appCategories: AppCategory[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'openApp', appId: string): void
  (e: 'openExternalApp', url: string): void
  (e: 'resetApp'): void
}>()

const activeAppTab = ref('categories')

const toggleCategory = (id: string) => {
  const category = props.appCategories.find(c => c.id === id)
  if (category) {
    category.expanded = !category.expanded
  }
}

const handleAppClick = (app: any) => {
  if (app.url) {
    emit('openExternalApp', app.url)
  } else {
    emit('openApp', app.id)
  }
}

const handleTabClick = (tab: string) => {
  activeAppTab.value = tab
  emit('resetApp')
}
</script>

<style scoped>
.apps-container {
  background: #fafafa;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  overflow: hidden;
  margin: 8px 8px;
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.app-tabs {
  display: flex;
  margin-bottom: 16px;
}

.app-tab-item {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  border-bottom: 2px solid transparent;
  gap: 8px;
}

.app-tab-item:hover {
  background: var(--hover-color);
}

.app-tab-item.active {
  border-bottom-color: var(--primary-color);
  color: var(--primary-color);
}

.tab-icon {
  font-size: 16px;
}

.tab-name {
  font-size: 14px;
  font-weight: 500;
}

.app-tab-content {
  padding: 0;
}

.category-icon {
  margin-right: 8px;
  font-size: 14px;
  color: var(--primary-color);
}

.category-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.category-toggle {
  font-size: 12px;
  color: var(--text-color);
  opacity: 0.7;
  transition: transform 0.2s;
}

.app-categories {
  padding: 0 8px;
}

.app-category-item {
  margin-bottom: 8px;
  border-radius: 4px;
  overflow: hidden;
}

.category-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background: transparent;
  cursor: pointer;
  transition: background 0.2s;
}

.category-header:hover {
  background: var(--hover-color);
}

.category-apps {
  background: transparent;
}

.category-app-item {
  display: flex;
  align-items: center;
  padding: 8px 16px 8px 32px;
  cursor: pointer;
  transition: background 0.2s;
  border: none;
}

.category-app-item:hover {
  background: var(--hover-color);
}

.category-app-icon {
  font-size: 16px;
  margin-right: 12px;
  width: 20px;
  text-align: center;
  color: var(--primary-color);
}

.category-app-name {
  font-size: 13px;
  color: var(--text-color);
  flex: 1;
}

.all-apps {
  padding: 0 8px;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}

.app-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
  text-align: center;
}

.app-item:hover {
  background: var(--hover-color);
}

.app-icon {
  font-size: 32px;
  margin-bottom: 8px;
  color: var(--primary-color);
}

.app-name {
  font-size: 13px;
  color: var(--text-color);
}
</style>
