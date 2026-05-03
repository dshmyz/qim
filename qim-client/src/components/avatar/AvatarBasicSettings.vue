<template>
  <div class="avatar-basic-settings">
    <div class="setting-item">
      <label class="toggle-label">
        <span>启用分身</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.enabled" @change="update('enabled', ($event.target as HTMLInputElement).checked)" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">开启后，分身将在你设定的规则下代替你回复消息</span>
    </div>

    <div class="setting-item">
      <label>分身名称</label>
      <input :value="modelValue.name" @input="update('name', ($event.target as HTMLInputElement).value)" class="form-input" placeholder="我的分身" />
      <span class="setting-hint">其他人在私聊中看到的分身名称</span>
    </div>

    <div class="setting-item">
      <label>模型来源</label>
      <select :value="modelValue.useSystemConfig ? 'system' : 'custom'" @change="handleModelSourceChange" class="form-select">
        <option value="system">使用系统默认模型</option>
        <option value="custom">使用我的自定义配置</option>
      </select>
    </div>

    <div v-if="!modelValue.useSystemConfig" class="setting-item">
      <label>选择配置</label>
      <select :value="modelValue.modelConfigId || ''" @change="update('modelConfigId', Number(($event.target as HTMLSelectElement).value) || null)" class="form-select">
        <option value="">请选择...</option>
        <option v-for="cfg in modelConfigs" :key="cfg.id" :value="cfg.id">
          {{ cfg.config_name }} ({{ cfg.model_name }})
        </option>
      </select>
      <span v-if="modelConfigs.length === 0" class="setting-hint error">暂无配置，请先在"我的模型配置"中添加</span>
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

function handleModelSourceChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  update('useSystemConfig', value === 'system')
  if (value === 'system') {
    update('modelConfigId', null)
  }
}
</script>

<style scoped>
.avatar-basic-settings { padding: 16px; }
.setting-item { margin-bottom: 16px; }
.setting-item > label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.toggle-label { display: flex; align-items: center; justify-content: space-between; cursor: pointer; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.setting-hint.error { color: #F44336; }
.form-select, .form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-select:focus, .form-input:focus { outline: none; border-color: var(--primary-color); }
.switch { position: relative; display: inline-block; width: 50px; height: 24px; min-width: 50px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: #ccc; transition: 0.4s; border-radius: 24px; }
.slider:before { position: absolute; content: ''; height: 18px; width: 18px; left: 3px; bottom: 3px; background-color: white; transition: 0.4s; border-radius: 50%; }
input:checked + .slider { background-color: var(--primary-color); }
input:checked + .slider:before { transform: translateX(26px); }
.slider.round { border-radius: 24px; }
</style>
