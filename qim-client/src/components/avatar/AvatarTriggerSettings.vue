<template>
  <div class="avatar-trigger-settings">
    <div class="setting-item">
      <label>触发模式</label>
      <select :value="modelValue.triggerRules.mode" @change="updateTrigger('mode', ($event.target as HTMLSelectElement).value)" class="form-select">
        <option value="mention">被 @ 时回复</option>
        <option value="offline">离线时自动回复</option>
        <option value="keyword">关键词触发</option>
        <option value="all">所有消息</option>
        <option value="custom">自定义规则</option>
      </select>
      <span class="setting-hint">
        {{ triggerModeHint }}
      </span>
    </div>

    <div v-if="modelValue.triggerRules.mode === 'keyword' || modelValue.triggerRules.mode === 'custom'" class="setting-item">
      <label>触发关键词</label>
      <div class="keyword-input-wrapper">
        <input
          :value="keywordInput"
          @input="keywordInput = ($event.target as HTMLInputElement).value"
          @keydown.enter.prevent="addKeyword"
          class="form-input"
          placeholder="输入关键词后按回车"
        />
        <div class="keyword-tags">
          <span v-for="(kw, i) in modelValue.triggerRules.keywords" :key="i" class="keyword-tag">
            {{ kw }}
            <button class="remove-tag" @click="removeKeyword(i)">x</button>
          </span>
        </div>
      </div>
    </div>

    <div class="setting-item">
      <label>接管冷却期</label>
      <select :value="modelValue.takeoverCooldown" @change="update('takeoverCooldown', Number(($event.target as HTMLSelectElement).value))" class="form-select">
        <option :value="5">5 分钟</option>
        <option :value="10">10 分钟</option>
        <option :value="30">30 分钟</option>
        <option :value="60">1 小时</option>
      </select>
      <span class="setting-hint">你发消息后，分身暂停回复的时间</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { AvatarConfig } from '../../types/avatar'

const props = defineProps<{
  modelValue: AvatarConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

const keywordInput = ref('')

const triggerModeHint = computed(() => {
  const hints: Record<string, string> = {
    mention: '当有人在私聊中发消息，或在群聊中 @你时，分身会回复',
    offline: '当你离线时，分身自动回复私聊消息',
    keyword: '仅当消息包含指定关键词时，分身才回复',
    all: '分身会回复所有消息（请谨慎使用）',
    custom: '自定义触发规则，结合关键词和时间段'
  }
  return hints[props.modelValue.triggerRules.mode] || ''
})

function update<K extends keyof AvatarConfig>(key: K, value: AvatarConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function updateTrigger(key: string, value: any) {
  emit('update:modelValue', {
    ...props.modelValue,
    triggerRules: { ...props.modelValue.triggerRules, [key]: value }
  })
}

function addKeyword() {
  const kw = keywordInput.value.trim()
  if (kw && !props.modelValue.triggerRules.keywords.includes(kw)) {
    emit('update:modelValue', {
      ...props.modelValue,
      triggerRules: {
        ...props.modelValue.triggerRules,
        keywords: [...props.modelValue.triggerRules.keywords, kw]
      }
    })
  }
  keywordInput.value = ''
}

function removeKeyword(index: number) {
  const keywords = [...props.modelValue.triggerRules.keywords]
  keywords.splice(index, 1)
  emit('update:modelValue', {
    ...props.modelValue,
    triggerRules: { ...props.modelValue.triggerRules, keywords }
  })
}
</script>

<style scoped>
.avatar-trigger-settings { padding: 16px; }
.setting-item { margin-bottom: 16px; }
.setting-item > label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.form-select, .form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-select:focus, .form-input:focus { outline: none; border-color: var(--primary-color); }
.keyword-input-wrapper { display: flex; flex-direction: column; gap: 8px; }
.keyword-tags { display: flex; flex-wrap: wrap; gap: 6px; }
.keyword-tag { display: inline-flex; align-items: center; gap: 4px; padding: 4px 10px; background: var(--primary-color-alpha, rgba(99, 102, 241, 0.1)); color: var(--primary-color); border-radius: 12px; font-size: 13px; }
.remove-tag { background: none; border: none; color: var(--primary-color); cursor: pointer; font-size: 14px; padding: 0; width: 16px; height: 16px; display: flex; align-items: center; justify-content: center; }
</style>
