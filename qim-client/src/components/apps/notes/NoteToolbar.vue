<template>
  <div class="note-toolbar">
    <div class="toolbar-left">
      <div class="btn-wrapper">
        <button
          :class="['mode-btn', { active: mode === 'edit' }]"
          @click="$emit('update:mode', 'edit')"
        >
          <i class="fas fa-edit"></i>
        </button>
        <span class="tooltip">编辑</span>
      </div>
      <div class="btn-wrapper">
        <button
          :class="['mode-btn', { active: mode === 'preview' }]"
          @click="$emit('update:mode', 'preview')"
        >
          <i class="fas fa-eye"></i>
        </button>
        <span class="tooltip">预览</span>
      </div>
      <div class="btn-wrapper">
        <button
          :class="['mode-btn', { active: fullscreen }]"
          @click="$emit('toggle-fullscreen')"
        >
          <i :class="fullscreen ? 'fas fa-compress' : 'fas fa-expand'"></i>
        </button>
        <span class="tooltip">{{ fullscreen ? '退出全屏' : '全屏' }}</span>
      </div>
    </div>
    <div class="toolbar-right">
      <div class="btn-wrapper">
        <button class="toolbar-btn save" @click="$emit('save')" :disabled="saving">
          <i class="fas fa-save"></i>
        </button>
        <span class="tooltip">保存</span>
      </div>
      <div class="btn-wrapper">
        <button class="toolbar-btn ai" @click="$emit('analyze')" :disabled="analyzing">
          <i class="fas fa-magic"></i>
        </button>
        <span class="tooltip">AI 分析</span>
      </div>
      <div class="btn-wrapper">
        <button class="toolbar-btn import" @click="$emit('import')">
          <i class="fas fa-file-import"></i>
        </button>
        <span class="tooltip">导入 Markdown</span>
      </div>
      <div class="btn-wrapper">
        <button class="toolbar-btn export" @click="$emit('export')">
          <i class="fas fa-download"></i>
        </button>
        <span class="tooltip">导出</span>
      </div>
      <div class="btn-wrapper">
        <button class="toolbar-btn share" @click="$emit('share')">
          <i class="fas fa-share-alt"></i>
        </button>
        <span class="tooltip">分享</span>
      </div>
      <div class="btn-wrapper">
        <button class="toolbar-btn delete" @click="$emit('delete')">
          <i class="fas fa-trash"></i>
        </button>
        <span class="tooltip">删除</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  mode: 'edit' | 'preview'
  saving?: boolean
  analyzing?: boolean
  fullscreen?: boolean
}>()

defineEmits<{
  'update:mode': [mode: 'edit' | 'preview']
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
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-2) var(--spacing-3);
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  margin-bottom: var(--spacing-4);
  box-shadow: var(--shadow-xs);
  flex-wrap: wrap;
  gap: var(--spacing-2);
}

.toolbar-left,
.toolbar-right {
  display: flex;
  gap: var(--spacing-2);
}

.btn-wrapper {
  position: relative;
  display: inline-block;
}

.tooltip {
  position: absolute;
  bottom: -28px;
  left: 50%;
  transform: translateX(-50%);
  background: var(--text-color);
  color: var(--card-bg);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  font-size: 11px;
  white-space: nowrap;
  opacity: 0;
  visibility: hidden;
  transition: all 0.15s ease;
  z-index: 100;
  pointer-events: none;
}

.tooltip::before {
  content: '';
  position: absolute;
  top: -4px;
  left: 50%;
  transform: translateX(-50%);
  border: 4px solid transparent;
  border-bottom-color: var(--text-color);
}

.btn-wrapper:hover .tooltip {
  opacity: 1;
  visibility: visible;
}

.mode-btn {
  width: 36px;
  height: 36px;
  border: 1px solid var(--border-color);
  background: var(--btn-bg);
  color: var(--text-secondary);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--font-size-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-base);
}

.mode-btn:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  background: var(--primary-light);
}

.mode-btn.active {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
  box-shadow: var(--shadow-sm);
}

.toolbar-btn {
  width: 36px;
  height: 36px;
  border: 1px solid var(--border-color);
  background: var(--btn-bg);
  color: var(--text-secondary);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--font-size-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-base);
}

.toolbar-btn:hover:not(:disabled) {
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.toolbar-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}

.toolbar-btn.save:hover:not(:disabled) {
  background: var(--success-color);
  border-color: var(--success-color);
  color: white;
  box-shadow: 0 4px 12px rgba(38, 179, 97, 0.3);
}

.toolbar-btn.ai {
  background: linear-gradient(135deg, var(--primary-color), var(--color-primary-600));
  border-color: transparent;
  color: white;
}

.toolbar-btn.ai:hover:not(:disabled) {
  background: linear-gradient(135deg, var(--color-primary-600), var(--color-primary-700));
  border-color: transparent;
  color: white;
  box-shadow: 0 4px 12px rgba(51, 133, 255, 0.4);
}

.toolbar-btn.export:hover:not(:disabled) {
  background: var(--color-info-500);
  border-color: var(--color-info-500);
  color: white;
}

.toolbar-btn.import:hover:not(:disabled) {
  background: var(--color-success-500);
  border-color: var(--color-success-500);
  color: white;
}

.toolbar-btn.share:hover:not(:disabled) {
  background: var(--color-warning-500);
  border-color: var(--color-warning-500);
  color: white;
}

.toolbar-btn.delete:hover:not(:disabled) {
  background: var(--danger-color);
  border-color: var(--danger-color);
  color: white;
  box-shadow: 0 4px 12px rgba(243, 64, 64, 0.3);
}
</style>
