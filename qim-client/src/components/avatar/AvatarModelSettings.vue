<template>
  <div class="avatar-model-settings">
    <div class="setting-item">
      <label>模型来源</label>
      <div class="radio-group">
        <label class="radio-label">
          <input 
            type="radio" 
            :checked="modelValue.useSystemConfig" 
            @change="update('useSystemConfig', true)"
          />
          <span>使用系统默认模型（推荐）</span>
        </label>
        <label class="radio-label">
          <input 
            type="radio" 
            :checked="!modelValue.useSystemConfig" 
            @change="update('useSystemConfig', false)"
          />
          <span>使用我的自定义配置</span>
        </label>
      </div>
    </div>

    <div v-if="!modelValue.useSystemConfig" class="setting-item">
      <label>选择配置</label>
      <select 
        :value="modelValue.modelConfigId || ''" 
        @change="update('modelConfigId', Number(($event.target as HTMLSelectElement).value) || null)" 
        class="form-select"
      >
        <option value="">请选择...</option>
        <option v-for="cfg in modelConfigs" :key="cfg.id" :value="cfg.id">
          {{ cfg.config_name }} ({{ cfg.model_name }})
        </option>
      </select>
      <span v-if="modelConfigs.length === 0" class="setting-hint error">
        暂无配置，请先在"我的模型配置"中添加
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AvatarConfig } from '../../types/avatar'
import type { UserAIConfig as AIConfig } from '../../types/ai'

const props = defineProps<{
  modelValue: AvatarConfig
  modelConfigs: AIConfig[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

function update<K extends keyof AvatarConfig>(key: K, value: AvatarConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}
</script>

<style scoped>
.avatar-model-settings {
  padding: 16px;
}

.setting-item {
  margin-bottom: 16px;
}

.setting-item > label {
  display: block;
  margin-bottom: 6px;
  font-size: 14px;
  font-weight: 500;
}

.radio-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 14px;
}

.radio-label input[type="radio"] {
  cursor: pointer;
}

.form-select {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-color);
  font-size: 14px;
  box-sizing: border-box;
}

.form-select:focus {
  outline: none;
  border-color: var(--primary-color);
}

.setting-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.setting-hint.error {
  color: #F44336;
}
</style>
