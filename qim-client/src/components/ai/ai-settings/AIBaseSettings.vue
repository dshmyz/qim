<template>
  <div class="ai-base-settings">
    <div class="setting-item">
      <label class="toggle-label">
        <span>启用 AI 助手</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.aiEnabled" @change="update('aiEnabled', ($event.target as HTMLInputElement).checked)" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">开启后，AI 将在群聊中辅助回复消息</span>
    </div>

    <div v-if="modelValue.aiEnabled" class="advanced-settings">
      <div class="setting-item">
        <label>AI 助手名称</label>
        <input
          type="text"
          :value="modelValue.aiAssistantName"
          @input="handleNameInput"
          class="form-input"
          placeholder="请输入 AI 助手名称"
          maxlength="20"
        />
        <span class="setting-hint">群成员 @ 此名称可触发 AI 回复</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { GroupAISettings } from '../../../types/ai'
import { validateAliasName } from '../../../utils/validation'

interface Props {
  modelValue: GroupAISettings
}

interface Emits {
  (e: 'update:modelValue', value: GroupAISettings): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

function update<K extends keyof GroupAISettings>(key: K, value: GroupAISettings[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function handleNameInput(event: Event) {
  const value = (event.target as HTMLInputElement).value
  const result = validateAliasName(value)
  if (!result.valid) {
    window.$QMessage.warning(result.message)
    return
  }
  update('aiAssistantName', value)
}
</script>

<style scoped>
.ai-base-settings { padding: 16px; }
.setting-item { margin-bottom: 20px; }
.setting-item:last-child { margin-bottom: 0; }
.setting-item label { display: block; margin-bottom: 6px; font-size: var(--font-size-sm); font-weight: 500; color: var(--text-color); }
.toggle-label { display: flex; align-items: center; justify-content: space-between; cursor: pointer; }
.setting-hint { display: block; margin-top: 6px; font-size: var(--font-size-xs); color: var(--text-secondary); line-height: 1.5; }
.form-input { width: 100%; padding: 9px 12px; border: 1px solid var(--border-color); border-radius: 8px; background: var(--bg-color); color: var(--text-color); font-size: var(--font-size-sm); box-sizing: border-box; transition: border-color 0.2s, box-shadow 0.2s; }
.form-input:focus { outline: none; border-color: var(--primary-color); box-shadow: 0 0 0 3px rgba(64, 158, 255, 0.1); }
.form-input::placeholder { color: var(--text-secondary); opacity: 0.6; }
.advanced-settings { margin-top: 16px; padding-top: 16px; border-top: 1px solid var(--border-color); }
.switch { position: relative; display: inline-block; width: 50px; height: 24px; min-width: 50px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: #ccc; transition: 0.4s; border-radius: 24px; }
.slider:before { position: absolute; content: ''; height: 18px; width: 18px; left: 3px; bottom: 3px; background-color: white; transition: 0.4s; border-radius: 50%; }
input:checked + .slider { background-color: var(--primary-color); }
input:checked + .slider:before { transform: translateX(26px); }
.slider.round { border-radius: 24px; }
</style>
