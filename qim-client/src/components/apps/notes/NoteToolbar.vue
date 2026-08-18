<template>
  <div class="note-toolbar">
    <div class="toolbar-row">
      <div class="toolbar-section">
        <button
          :class="['tb-btn', 'labeled', 'save', { 'is-synced': saveStatus === 'saved', 'is-draft': saveStatus === 'draft', 'is-error': saveStatus === 'error' }]"
          @click="$emit('save')"
          :disabled="saving"
          :title="saveStatusTitle"
        >
          <i :class="saveStatusIcon"></i><span>{{ saveStatusLabel }}</span>
        </button>
        <button class="tb-btn labeled delete" @click="$emit('delete')" title="删除">
          <i class="fas fa-trash"></i><span>删除</span>
        </button>
        <button
          :class="['tb-btn', 'labeled', 'ai-access', { hidden: !aiAccessible }]"
          @click="$emit('update:ai-accessible', !aiAccessible)"
          :title="aiAccessible ? '分身可以读取这篇笔记，点击改为不可读' : '分身读不到这篇笔记，点击改为可读'"
        >
          <i :class="aiAccessible ? 'fas fa-eye' : 'fas fa-eye-slash'"></i>
          <span>{{ aiAccessible ? '分身可见' : '分身不可见' }}</span>
        </button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-section">
        <button class="tb-btn labeled import" @click="$emit('import')" title="导入文件 (MD / TXT / HTML / DOCX / PDF)">
          <i class="fas fa-file-import"></i><span>导入</span>
        </button>
        <button class="tb-btn labeled export" @click="$emit('export')" title="导出">
          <i class="fas fa-download"></i><span>导出</span>
        </button>
        <button class="tb-btn labeled" @click="$emit('analyze')" :disabled="analyzing" title="AI 分析 (生成摘要和标签)">
          <i :class="analyzing ? 'fas fa-spinner fa-spin' : 'fas fa-magic'"></i><span>AI 分析</span>
        </button>
        <button class="tb-btn labeled" @click="$emit('ai-format')" :disabled="formatting" title="AI 格式化 (整理为 Markdown)">
          <i :class="formatting ? 'fas fa-spinner fa-spin' : 'fas fa-wand-magic-sparkles'"></i><span>AI 格式化</span>
        </button>
        <button class="tb-btn labeled" @click="$emit('share')" title="分享笔记">
          <i class="fas fa-share-alt"></i><span>分享</span>
        </button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-section">
        <button
          :class="['tb-btn', 'labeled', { active: mode === 'edit' }]"
          @click="$emit('update:mode', 'edit')"
          title="仅编辑"
        >
          <i class="fas fa-edit"></i><span>编辑</span>
        </button>
        <button
          :class="['tb-btn', 'labeled', { active: mode === 'split' }]"
          @click="$emit('update:mode', 'split')"
          title="分栏预览"
        >
          <i class="fas fa-columns"></i><span>分栏</span>
        </button>
        <button
          :class="['tb-btn', 'labeled', { active: mode === 'preview' }]"
          @click="$emit('update:mode', 'preview')"
          title="仅预览"
        >
          <i class="fas fa-eye"></i><span>预览</span>
        </button>
      </div>
    </div>

    <div class="toolbar-row">
      <div class="toolbar-section">
        <button class="tb-btn" @click="$emit('format', '**', '**')" title="粗体 (Ctrl+B)">
          <strong>B</strong>
        </button>
        <button class="tb-btn" @click="$emit('format', '*', '*')" title="斜体 (Ctrl+I)">
          <em>I</em>
        </button>
        <button class="tb-btn" @click="$emit('format', '~~', '~~')" title="删除线 (Ctrl+Shift+X)">
          <s>S</s>
        </button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-section">
        <button class="tb-btn" @click="$emit('format', '# ', '')" title="一级标题 (Ctrl+Shift+1)">H1</button>
        <button class="tb-btn" @click="$emit('format', '## ', '')" title="二级标题 (Ctrl+Shift+2)">H2</button>
        <button class="tb-btn" @click="$emit('format', '### ', '')" title="三级标题 (Ctrl+Shift+3)">H3</button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-section">
        <button class="tb-btn" @click="$emit('format', '- ', '')" title="无序列表 (Ctrl+Shift+U)">
          <i class="fas fa-list-ul"></i>
        </button>
        <button class="tb-btn" @click="$emit('format', '1. ', '')" title="有序列表 (Ctrl+Shift+O)">
          <i class="fas fa-list-ol"></i>
        </button>
        <button class="tb-btn" @click="$emit('format', '- [ ] ', '')" title="任务列表 (Ctrl+Shift+T)">
          <i class="fas fa-tasks"></i>
        </button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-section">
        <button class="tb-btn" @click="$emit('format', '> ', '')" title="引用 (Ctrl+Shift+Q)">
          <i class="fas fa-quote-left"></i>
        </button>
        <button class="tb-btn" @click="$emit('format', '`', '`')" title="行内代码 (Ctrl+Shift+C)">
          <i class="fas fa-code"></i>
        </button>
        <button class="tb-btn" @click="$emit('insert-code-block')" title="代码块 (Ctrl+Shift+B)">
          <i class="fas fa-file-code"></i>
        </button>
        <button class="tb-btn" @click="$emit('insert-link')" title="链接 (Ctrl+K)">
          <i class="fas fa-link"></i>
        </button>
        <button class="tb-btn" @click="$emit('format', '---\n', '')" title="分割线">
          <i class="fas fa-minus"></i>
        </button>
        <button class="tb-btn" @click="$emit('insert-table')" title="插入表格">
          <i class="fas fa-table"></i>
        </button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-section">
        <button
          :class="['tb-btn', 'labeled', { active: fullscreen }]"
          @click="$emit('toggle-fullscreen')"
          :title="fullscreen ? '退出全屏 (F11)' : '全屏 (F11)'"
        >
          <i :class="fullscreen ? 'fas fa-compress' : 'fas fa-expand'"></i>
        </button>
        <button class="tb-btn" @click="$emit('show-shortcuts')" title="快捷键帮助 (?)">
          <i class="fas fa-keyboard"></i>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AutoSaveStatus } from '../../../composables/useAutoSave'

const props = defineProps<{
  mode: 'edit' | 'split' | 'preview'
  saving?: boolean
  saveStatus?: AutoSaveStatus
  analyzing?: boolean
  formatting?: boolean
  fullscreen?: boolean
  aiAccessible?: boolean
}>()

defineEmits<{
  'update:mode': [mode: 'edit' | 'split' | 'preview']
  format: [prefix: string, suffix: string]
  'insert-link': []
  'insert-code-block': []
  save: []
  analyze: []
  'ai-format': []
  import: []
  export: []
  share: []
  delete: []
  'toggle-fullscreen': []
  'update:ai-accessible': [value: boolean]
  'show-shortcuts': []
  'insert-table': []
}>()

const saveStatusLabel = computed(() => {
  switch (props.saveStatus) {
    case 'saving':
      return '同步中…'
    case 'saved':
      return '已同步'
    case 'error':
      return '同步失败'
    case 'draft':
      return '草稿已保存'
    default:
      return '保存'
  }
})

const saveStatusIcon = computed(() => {
  switch (props.saveStatus) {
    case 'saving':
      return 'fas fa-spinner fa-spin'
    case 'saved':
      return 'fas fa-cloud-upload-alt'
    case 'error':
      return 'fas fa-exclamation-circle'
    case 'draft':
      return 'fas fa-file-alt'
    default:
      return 'fas fa-save'
  }
})

const saveStatusTitle = computed(() => {
  switch (props.saveStatus) {
    case 'saving':
      return '正在同步到服务器…'
    case 'saved':
      return '已同步到服务器'
    case 'error':
      return '同步失败，点击手动重试 (Ctrl+S)'
    case 'draft':
      return '草稿已保存到本地，等待同步'
    default:
      return '保存 (Ctrl+S)'
  }
})
</script>

<style scoped>
.note-toolbar {
  display: flex;
  flex-direction: column;
  padding: var(--spacing-1) var(--spacing-2);
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  margin-bottom: var(--spacing-3);
  box-shadow: var(--shadow-xs);
  gap: var(--spacing-1);
}

.toolbar-row {
  display: flex;
  align-items: center;
  gap: 2px;
  /* 窗口过窄时按钮换行而非溢出被裁：按钮均有 nowrap + min-width，不换行会溢出
     到容器外被外层裁掉（用户反馈窄窗口下按钮「消失」） */
  flex-wrap: wrap;
}

.toolbar-section {
  display: flex;
  align-items: center;
  gap: 2px;
}

.toolbar-divider {
  width: 1px;
  height: 16px;
  background: var(--border-color);
  margin: 0 var(--spacing-1);
  flex-shrink: 0;
}

.tb-btn {
  height: 28px;
  min-width: 28px;
  padding: 0 5px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--text-secondary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: var(--font-size-xxs);
  font-weight: var(--font-weight-medium);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
  white-space: nowrap;
  position: relative;
}

/* CSS tooltip：替代原生 title，hover 即出 */
.tb-btn[title]::after {
  content: attr(title);
  position: absolute;
  top: calc(100% + 6px);
  left: 50%;
  transform: translateX(-50%);
  padding: 4px 8px;
  font-size: var(--font-size-xxxs);
  font-weight: 400;
  white-space: nowrap;
  color: #fff;
  background: #333;
  border-radius: 4px;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.15s;
  z-index: 10;
}

.tb-btn[title]:hover::after {
  opacity: 1;
}

.tb-btn:hover {
  background: var(--primary-light);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.tb-btn:active {
  transform: scale(0.95);
}

.tb-btn.active {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.tb-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.tb-btn.labeled {
  padding: 0 8px;
  gap: 4px;
}

.tb-btn.labeled span {
  font-size: var(--font-size-xxxs);
  line-height: 1;
}

.tb-btn.save:hover:not(:disabled) {
  background: var(--success-color);
  border-color: var(--success-color);
  color: white;
}

.tb-btn.save:hover:not(:disabled) span {
  color: white;
}

.tb-btn.save.is-synced {
  color: var(--success-color);
  border-color: var(--success-color);
}

.tb-btn.save.is-draft {
  color: var(--text-secondary);
  border-color: var(--border-color);
  opacity: 0.85;
}

.tb-btn.save.is-error {
  color: var(--danger-color);
  border-color: var(--danger-color);
}

.tb-btn.delete:hover:not(:disabled) {
  background: var(--danger-color);
  border-color: var(--danger-color);
  color: white;
}

.tb-btn.delete:hover:not(:disabled) span {
  color: white;
}

.tb-btn.ai-access {
  color: var(--primary-color);
  border-color: var(--primary-color);
  opacity: 0.9;
}

.tb-btn.ai-access:not(.hidden) {
  background: rgba(51, 133, 255, 0.06);
}

.tb-btn.ai-access:hover:not(:disabled) {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.tb-btn.ai-access:hover:not(:disabled) span {
  color: white;
}
</style>
