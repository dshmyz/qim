<template>
  <div class="note-toolbar">
    <div class="toolbar-row">
      <div class="toolbar-section">
        <button class="tb-btn labeled save" @click="$emit('save')" :disabled="saving" title="保存 (Ctrl+S)">
          <i class="fas fa-save"></i><span>保存</span>
        </button>
        <button class="tb-btn labeled delete" @click="$emit('delete')" title="删除">
          <i class="fas fa-trash"></i><span>删除</span>
        </button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-section">
        <button class="tb-btn labeled import" @click="$emit('import')" title="导入 Markdown">
          <i class="fas fa-file-import"></i><span>导入</span>
        </button>
        <button class="tb-btn labeled export" @click="$emit('export')" title="导出">
          <i class="fas fa-download"></i><span>导出</span>
        </button>
        <button class="tb-btn" @click="$emit('analyze')" :disabled="analyzing" title="AI 分析">
          <i class="fas fa-magic"></i>
        </button>
        <button class="tb-btn" @click="$emit('share')" title="分享">
          <i class="fas fa-share-alt"></i>
        </button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-section">
        <button
          :class="['tb-btn', { active: mode === 'edit' }]"
          @click="$emit('update:mode', 'edit')"
          title="仅编辑"
        >
          <i class="fas fa-edit"></i>
        </button>
        <button
          :class="['tb-btn', { active: mode === 'split' }]"
          @click="$emit('update:mode', 'split')"
          title="分栏预览"
        >
          <i class="fas fa-columns"></i>
        </button>
        <button
          :class="['tb-btn', { active: mode === 'preview' }]"
          @click="$emit('update:mode', 'preview')"
          title="仅预览"
        >
          <i class="fas fa-eye"></i>
        </button>
        <div class="toolbar-divider"></div>
        <button
          :class="['tb-btn', { active: fullscreen }]"
          @click="$emit('toggle-fullscreen')"
          :title="fullscreen ? '退出全屏 (F11)' : '全屏 (F11)'"
        >
          <i :class="fullscreen ? 'fas fa-compress' : 'fas fa-expand'"></i>
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
        <button class="tb-btn" @click="$emit('format', '~~', '~~')" title="删除线">
          <s>S</s>
        </button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-section">
        <button class="tb-btn" @click="$emit('format', '# ', '')" title="一级标题">H1</button>
        <button class="tb-btn" @click="$emit('format', '## ', '')" title="二级标题">H2</button>
        <button class="tb-btn" @click="$emit('format', '### ', '')" title="三级标题">H3</button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-section">
        <button class="tb-btn" @click="$emit('format', '- ', '')" title="无序列表">
          <i class="fas fa-list-ul"></i>
        </button>
        <button class="tb-btn" @click="$emit('format', '1. ', '')" title="有序列表">
          <i class="fas fa-list-ol"></i>
        </button>
        <button class="tb-btn" @click="$emit('format', '- [ ] ', '')" title="任务列表">
          <i class="fas fa-tasks"></i>
        </button>
      </div>
      <div class="toolbar-divider"></div>
      <div class="toolbar-section">
        <button class="tb-btn" @click="$emit('format', '> ', '')" title="引用">
          <i class="fas fa-quote-left"></i>
        </button>
        <button class="tb-btn" @click="$emit('format', '`', '`')" title="行内代码">
          <i class="fas fa-code"></i>
        </button>
        <button class="tb-btn" @click="$emit('format', '```\n', '\n```')" title="代码块">
          <i class="fas fa-file-code"></i>
        </button>
        <button class="tb-btn" @click="$emit('insert-link')" title="链接 (Ctrl+K)">
          <i class="fas fa-link"></i>
        </button>
        <button class="tb-btn" @click="$emit('format', '---\n', '')" title="分割线">
          <i class="fas fa-minus"></i>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  mode: 'edit' | 'split' | 'preview'
  saving?: boolean
  analyzing?: boolean
  fullscreen?: boolean
}>()

defineEmits<{
  'update:mode': [mode: 'edit' | 'split' | 'preview']
  format: [prefix: string, suffix: string]
  'insert-link': []
  save: []
  analyze: []
  import: []
  export: []
  share: []
  delete: []
  'toggle-fullscreen': []
}>()
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
  font-size: 12px;
  font-weight: var(--font-weight-medium);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
  white-space: nowrap;
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
  font-size: 11px;
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

.tb-btn.delete:hover:not(:disabled) {
  background: var(--danger-color);
  border-color: var(--danger-color);
  color: white;
}

.tb-btn.delete:hover:not(:disabled) span {
  color: white;
}
</style>
