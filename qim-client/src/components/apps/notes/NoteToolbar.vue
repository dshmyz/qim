<template>
  <div class="note-toolbar">
    <div class="toolbar-left">
      <button
        :class="['mode-btn', { active: mode === 'edit' }]"
        @click="$emit('update:mode', 'edit')"
      >
        <i class="fas fa-edit"></i>
        编辑
      </button>
      <button
        :class="['mode-btn', { active: mode === 'preview' }]"
        @click="$emit('update:mode', 'preview')"
      >
        <i class="fas fa-eye"></i>
        预览
      </button>
    </div>
    <div class="toolbar-right">
      <button class="toolbar-btn save" @click="$emit('save')" :disabled="saving">
        <i class="fas fa-save"></i>
        {{ saving ? '保存中...' : '保存' }}
      </button>
      <button class="toolbar-btn ai" @click="$emit('analyze')" :disabled="analyzing">
        <i class="fas fa-magic"></i>
        {{ analyzing ? '分析中...' : 'AI 分析' }}
      </button>
      <button class="toolbar-btn export" @click="$emit('export')">
        <i class="fas fa-download"></i>
        导出
      </button>
      <button class="toolbar-btn share" @click="$emit('share')">
        <i class="fas fa-share-alt"></i>
        分享
      </button>
      <button class="toolbar-btn delete" @click="$emit('delete')">
        <i class="fas fa-trash"></i>
        删除
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  mode: 'edit' | 'preview'
  saving?: boolean
  analyzing?: boolean
}>()

defineEmits<{
  'update:mode': [mode: 'edit' | 'preview']
  save: []
  analyze: []
  export: []
  share: []
  delete: []
}>()
</script>

<style scoped>
.note-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-3) var(--spacing-4);
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  margin-bottom: var(--spacing-4);
  box-shadow: var(--shadow-xs);
}

.toolbar-left,
.toolbar-right {
  display: flex;
  gap: var(--spacing-2);
}

.mode-btn {
  padding: var(--spacing-2) var(--spacing-4);
  border: 1px solid var(--border-color);
  background: var(--btn-bg);
  color: var(--text-secondary);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
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
  padding: var(--spacing-2) var(--spacing-3);
  border: 1px solid var(--border-color);
  background: var(--btn-bg);
  color: var(--text-secondary);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
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
