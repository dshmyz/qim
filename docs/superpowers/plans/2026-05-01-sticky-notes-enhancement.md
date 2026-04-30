# 便签模块优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development 逐任务实现此计划。

**目标：** 优化便签模块界面，增加标签筛选和全屏编辑功能

**架构：** 前端组件拆分，复用笔记模块的设计模式

**技术栈：** Vue 3 + TypeScript

---

## 任务 1：创建 StickyTagFilter 组件

**文件：**
- 创建：`qim-client/src/components/apps/sticky/StickyTagFilter.vue`

**代码：**
```vue
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
```

---

## 任务 2：优化便签卡片样式

**文件：**
- 修改：`qim-client/src/components/apps/StickyNotesApp.vue`

**改动点：**
- 使用设计系统令牌
- 优化卡片悬停效果
- 调整间距和字体大小

---

## 任务 3：添加全屏编辑模式

**文件：**
- 修改：`qim-client/src/components/apps/StickyNotesApp.vue`

**改动点：**
- 添加全屏状态变量
- 添加全屏切换按钮
- 全屏时隐藏头部和网格，只显示编辑弹窗

---

## 任务 4：添加标签筛选功能

**文件：**
- 修改：`qim-client/src/components/apps/StickyNotesApp.vue`

**改动点：**
- 导入 StickyTagFilter 组件
- 添加筛选逻辑
- 计算所有标签

---

## 任务 5：添加 AI 分析按钮

**文件：**
- 修改：`qim-client/src/components/apps/StickyNotesApp.vue`

**改动点：**
- 在编辑弹窗中添加 AI 分析按钮
- 复用笔记模块的 AI 分析 API
- 显示分析结果
