<template>
  <!-- 统一的 AI 回复来源徽章。群助手/Bot（折叠列表）与分身（行内标签）共用本组件，
       由 variant 区分渲染形态；数据统一走 AISource 结构（source 编码 + 兼容分身旧 type shape）。 -->
  <details
    v-if="variant === 'list' && sources && sources.length > 0"
    class="knowledge-sources"
  >
    <summary>
      <i class="fas fa-book"></i>
      <span>知识来源</span>
      <span class="count">{{ sources.length }}</span>
    </summary>
    <ul>
      <li v-for="(src, i) in sources" :key="i">
        <span v-if="src.source || src.type" class="source-tag">{{ sourceLabel(src.source, src.type) }}</span>
        <span
          class="title"
        >{{ src.title || '未命名' }}</span>
        <span v-if="src.snippet" class="snippet">{{ src.snippet }}</span>
      </li>
    </ul>
  </details>

  <!-- 分身「依据」行内标签 -->
  <div v-else-if="variant === 'inline' && sources && sources.length > 0" class="avatar-sources">
    <span class="avatar-sources-label">
      <svg viewBox="0 0 24 24" width="11" height="11" fill="currentColor" aria-hidden="true">
        <path d="M11 2a10 10 0 100 20 10 10 0 000-20zm0 3a7 7 0 110 14 7 7 0 010-14zm-1 4v4l3 2 .8-1.2-2.3-1.5V9H10z"/>
      </svg>
      依据
    </span>
    <span
      v-for="(src, i) in sources"
      :key="i"
      class="avatar-source-tag"
      :title="src.snippet || ''"
    >
      {{ sourceLabel(src.source, src.type) }}
      <template v-if="src.title">《{{ src.title }}》</template>
    </span>
  </div>
</template>

<script setup lang="ts">
import type { AISource } from '../../types'

// 归一来源：新数据走 source（knowledge/notes/memory），兼容未统一前分身旧 shape 的 type（note/group/memory）
type NormalizedSource = 'knowledge' | 'notes' | 'memory'

defineProps<{
  sources?: AISource[]
  /** list = 群助手/Bot 折叠列表（带来源标签+标题，悬停/点击看摘要），inline = 分身行内标签（snippet tooltip） */
  variant: 'list' | 'inline'
}>()

// 来源标签映射（统一单套）：新数据走 source（knowledge/notes/memory），
// 兼容旧 shape 的 type（note→笔记 / group→群知识 / memory→记忆）。
const sourceLabel = (source?: string, type?: string): string => {
  const labels: Record<string, string> = {
    knowledge: '知识库',
    notes: '笔记',
    memory: '记忆',
    note: '笔记',
    group: '群知识',
  }
  // 优先读 source，无则回退到 type（兼容分身旧 shape），最终兜底中文
  const key = source || type
  return labels[key ?? ''] || key || '资料'
}
</script>

<style scoped>
/* ===== list 变体（原 KnowledgeSources.vue）===== */
.knowledge-sources {
  margin-top: 6px;
  padding: 4px 10px;
  border: 1px solid var(--border-color, #e8ecf3);
  border-radius: 6px;
  background: var(--hover-color, #f9fafc);
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, #666);
}
.knowledge-sources summary {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  list-style: none;
  user-select: none;
  color: var(--text-secondary, #888);
}
.knowledge-sources summary::-webkit-details-marker { display: none; }
.knowledge-sources summary i { color: #4f7cff; font-size: var(--font-size-xxxs); }
.knowledge-sources .count {
  display: inline-block;
  min-width: 16px;
  padding: 0 5px;
  height: 16px;
  line-height: 16px;
  text-align: center;
  background: var(--border-color, #e8ecf3);
  color: var(--text-secondary, #666);
  border-radius: 8px;
  font-size: var(--font-size-tiny);
}
.knowledge-sources ul {
  margin: 6px 0 2px;
  padding-left: 18px;
  list-style: none;
}
.knowledge-sources li {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 6px 8px;
  padding: 3px 0;
  border-top: 1px dashed var(--border-color, #eef1f6);
}
.knowledge-sources li:first-child { border-top: none; }
.knowledge-sources .source-tag {
  flex-shrink: 0;
  padding: 0 4px;
  height: 14px;
  line-height: 14px;
  font-size: 9px;
  border-radius: 3px;
  background: var(--border-color, #e8ecf3);
  color: var(--text-secondary, #888);
  white-space: nowrap;
}
.knowledge-sources .title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-color, #333);
}
.knowledge-sources .snippet {
  width: 100%;
  margin-top: 2px;
  font-size: var(--font-size-xxxs);
  line-height: 1.4;
  color: var(--text-secondary, #999);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* ===== inline 变体（原 AvatarSources.vue）===== */
.avatar-sources {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 8px;
  margin-top: 4px;
  font-size: var(--font-size-xxxs);
  line-height: 1.4;
  color: var(--text-secondary, #666);
  opacity: 0.75;
}
.avatar-sources-label {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: inherit;
}
.avatar-source-tag {
  display: inline-flex;
  align-items: center;
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 0 6px;
  border-radius: 8px;
  background: rgba(59, 130, 246, 0.08);
  color: #4f7cff;
}
[data-theme="elegant-dark"] .avatar-source-tag {
  background: rgba(59, 130, 246, 0.16);
  color: #7aa2ff;
}
</style>
