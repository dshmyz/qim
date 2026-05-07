<template>
  <div class="avatar-knowledge-settings">
    <div class="setting-section">
      <div class="section-header">
        <i class="fas fa-book-open"></i>
        <h4>知识来源</h4>
      </div>
      <div class="setting-item">
        <label class="toggle-label">
          <div class="label-content">
            <span class="label-title">当前会话历史</span>
            <span class="label-hint">分身可以参考当前会话中的历史消息来回复</span>
          </div>
          <label class="switch">
            <input type="checkbox" :checked="knowledgeScope?.conversationHistory ?? false" @change="updateScope('conversationHistory', ($event.target as HTMLInputElement).checked)" />
            <span class="slider round"></span>
          </label>
        </label>
      </div>

      <div class="setting-item">
        <label class="toggle-label">
          <div class="label-content">
            <span class="label-title">知识库文档</span>
            <span class="label-hint">分身可以访问你上传的知识库文档</span>
          </div>
          <label class="switch">
            <input type="checkbox" :checked="knowledgeScope?.knowledgeDocs ?? false" @change="updateScope('knowledgeDocs', ($event.target as HTMLInputElement).checked)" />
            <span class="slider round"></span>
          </label>
        </label>
      </div>

      <div class="setting-item">
        <label class="toggle-label">
          <div class="label-content">
            <span class="label-title">用户笔记</span>
            <span class="label-hint">分身可以读取你的笔记内容</span>
          </div>
          <label class="switch">
            <input type="checkbox" :checked="knowledgeScope?.notes ?? false" @change="updateScope('notes', ($event.target as HTMLInputElement).checked)" />
            <span class="slider round"></span>
          </label>
        </label>
      </div>

      <div class="setting-item">
        <label class="toggle-label">
          <div class="label-content">
            <span class="label-title">用户任务</span>
            <span class="label-hint">分身可以读取你的任务列表</span>
          </div>
          <label class="switch">
            <input type="checkbox" :checked="knowledgeScope?.tasks ?? false" @change="updateScope('tasks', ($event.target as HTMLInputElement).checked)" />
            <span class="slider round"></span>
          </label>
        </label>
      </div>
    </div>

    <div class="privacy-notice">
      <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
        <path d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z"/>
      </svg>
      <span>分身仅在你允许的范围内读取信息，不会访问未授权的数据</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AvatarConfig, AvatarKnowledgeScope } from '../../types/avatar'

const props = defineProps<{
  modelValue: AvatarConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

const knowledgeScope = computed(() => props.modelValue?.knowledgeScope)

function updateScope(key: keyof AvatarKnowledgeScope, value: boolean) {
  const currentScope = props.modelValue.knowledgeScope || {
    conversationHistory: false,
    knowledgeDocs: false,
    notes: false,
    tasks: false
  }
  emit('update:modelValue', {
    ...props.modelValue,
    knowledgeScope: { ...currentScope, [key]: value }
  })
}
</script>

<style scoped>
.avatar-knowledge-settings { 
  padding: 16px; 
  min-height: 100%;
}

.setting-section {
  background: var(--card-bg);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-color);
}

.section-header i {
  color: var(--primary-color);
  font-size: 16px;
}

.section-header h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
}

.setting-item { 
  margin-bottom: 16px; 
  padding: 12px;
  background: var(--bg-color);
  border-radius: 6px;
  transition: background 0.2s;
}

.setting-item:hover {
  background: var(--hover-color);
}

.setting-item:last-child {
  margin-bottom: 0;
}

.toggle-label { 
  display: flex; 
  align-items: center; 
  justify-content: space-between; 
  cursor: pointer; 
}

.label-content {
  flex: 1;
  margin-right: 12px;
}

.label-title {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.label-hint {
  display: block;
  font-size: 12px;
  color: var(--text-secondary);
}

.switch { 
  position: relative; 
  display: inline-block; 
  width: 48px; 
  height: 24px; 
  min-width: 48px; 
  flex-shrink: 0;
}

.switch input { 
  opacity: 0; 
  width: 0; 
  height: 0; 
}

.slider { 
  position: absolute; 
  cursor: pointer; 
  top: 0; 
  left: 0; 
  right: 0; 
  bottom: 0; 
  background-color: var(--border-color); 
  transition: background-color 0.3s; 
  border-radius: 12px; 
}

.slider:before { 
  position: absolute; 
  content: ''; 
  height: 18px; 
  width: 18px; 
  left: 3px; 
  bottom: 3px; 
  background-color: white; 
  transition: transform 0.3s; 
  border-radius: 50%; 
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

input:checked + .slider { 
  background-color: var(--primary-color); 
}

input:checked + .slider:before { 
  transform: translateX(24px); 
}

.privacy-notice { 
  display: flex; 
  align-items: flex-start; 
  gap: 8px; 
  padding: 12px; 
  background: rgba(59, 130, 246, 0.06); 
  border-radius: 8px; 
  font-size: 12px; 
  color: var(--text-secondary); 
  border-left: 3px solid var(--primary-color);
}

.privacy-notice svg {
  flex-shrink: 0;
  color: var(--primary-color);
}
</style>
