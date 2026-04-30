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
  padding: 12px 16px;
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  margin-bottom: 16px;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  gap: 8px;
}

.mode-btn {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  background: var(--bg-color);
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.mode-btn:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.mode-btn.active {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.toolbar-btn {
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  background: var(--bg-color);
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.toolbar-btn:hover:not(:disabled) {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.toolbar-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.toolbar-btn.save:hover:not(:disabled) {
  background: var(--success-color);
  border-color: var(--success-color);
  color: white;
}

.toolbar-btn.ai:hover:not(:disabled) {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.toolbar-btn.delete:hover:not(:disabled) {
  background: var(--danger-color);
  border-color: var(--danger-color);
  color: white;
}
</style>
