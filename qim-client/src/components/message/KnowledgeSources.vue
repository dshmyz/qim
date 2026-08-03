<template>
  <!-- Bot 回复命中创建者笔记时的知识来源折叠标签 -->
  <details v-if="sources && sources.length > 0" class="knowledge-sources">
    <summary>
      <i class="fas fa-book"></i>
      <span>知识来源</span>
      <span class="count">{{ sources.length }}</span>
    </summary>
    <ul>
      <li v-for="(src, i) in sources" :key="i">
        <span class="title" :title="src.title">{{ src.title || '未命名' }}</span>
        <span class="score">{{ formatScore(src.score) }}</span>
      </li>
    </ul>
  </details>
</template>

<script setup lang="ts">
import type { KnowledgeSource } from '../../types'

defineProps<{
  sources?: KnowledgeSource[]
}>()

// 分数格式化：0.92 → 92%
const formatScore = (score: number) => {
  if (typeof score !== 'number' || isNaN(score)) return ''
  return Math.round(score * 100) + '%'
}
</script>

<style scoped>
.knowledge-sources {
  margin-top: 6px;
  padding: 4px 10px;
  border: 1px solid #e8ecf3;
  border-radius: 6px;
  background: #f9fafc;
  font-size: 12px;
  color: #666;
}
.knowledge-sources summary {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  list-style: none;
  user-select: none;
  color: #888;
}
.knowledge-sources summary::-webkit-details-marker { display: none; }
.knowledge-sources summary i { color: #4f7cff; font-size: 11px; }
.knowledge-sources .count {
  display: inline-block;
  min-width: 16px;
  padding: 0 5px;
  height: 16px;
  line-height: 16px;
  text-align: center;
  background: #e8ecf3;
  color: #666;
  border-radius: 8px;
  font-size: 10px;
}
.knowledge-sources ul {
  margin: 6px 0 2px;
  padding-left: 18px;
  list-style: none;
}
.knowledge-sources li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding: 3px 0;
  border-top: 1px dashed #eef1f6;
}
.knowledge-sources li:first-child { border-top: none; }
.knowledge-sources .title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #333;
}
.knowledge-sources .score {
  flex-shrink: 0;
  color: #4f7cff;
  font-size: 11px;
  font-weight: 500;
}
</style>
