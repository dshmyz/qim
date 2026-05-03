<template>
  <div class="avatar-knowledge-settings">
    <div class="setting-item">
      <label class="toggle-label">
        <span>当前会话历史</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.knowledgeScope.conversationHistory" @change="updateScope('conversationHistory', ($event.target as HTMLInputElement).checked)" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">分身可以参考当前会话中的历史消息来回复</span>
    </div>

    <div class="setting-item">
      <label class="toggle-label">
        <span>知识库文档</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.knowledgeScope.knowledgeDocs" @change="updateScope('knowledgeDocs', ($event.target as HTMLInputElement).checked)" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">分身可以访问你上传的知识库文档</span>
    </div>

    <div class="setting-item">
      <label class="toggle-label">
        <span>用户笔记</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.knowledgeScope.notes" @change="updateScope('notes', ($event.target as HTMLInputElement).checked)" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">分身可以读取你的笔记内容</span>
    </div>

    <div class="setting-item">
      <label class="toggle-label">
        <span>用户任务</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.knowledgeScope.tasks" @change="updateScope('tasks', ($event.target as HTMLInputElement).checked)" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">分身可以读取你的任务列表</span>
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
import type { AvatarConfig, AvatarKnowledgeScope } from '../../types/avatar'

const props = defineProps<{
  modelValue: AvatarConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

function updateScope(key: keyof AvatarKnowledgeScope, value: boolean) {
  emit('update:modelValue', {
    ...props.modelValue,
    knowledgeScope: { ...props.modelValue.knowledgeScope, [key]: value }
  })
}
</script>

<style scoped>
.avatar-knowledge-settings { padding: 16px; }
.setting-item { margin-bottom: 16px; }
.toggle-label { display: flex; align-items: center; justify-content: space-between; cursor: pointer; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.switch { position: relative; display: inline-block; width: 50px; height: 24px; min-width: 50px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: #ccc; transition: 0.4s; border-radius: 24px; }
.slider:before { position: absolute; content: ''; height: 18px; width: 18px; left: 3px; bottom: 3px; background-color: white; transition: 0.4s; border-radius: 50%; }
input:checked + .slider { background-color: var(--primary-color); }
input:checked + .slider:before { transform: translateX(26px); }
.slider.round { border-radius: 24px; }
.privacy-notice { display: flex; align-items: center; gap: 6px; padding: 10px 12px; background: rgba(59, 130, 246, 0.06); border-radius: 6px; font-size: 12px; color: var(--text-secondary); margin-top: 8px; }
</style>
