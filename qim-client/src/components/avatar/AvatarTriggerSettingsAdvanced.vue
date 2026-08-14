<template>
  <div class="avatar-trigger-settings-advanced">
    <div class="current-mode-display">
      <span class="mode-label">当前触发模式：</span>
      <span class="mode-value">{{ modeLabel }}</span>
      <span class="mode-hint">（在普通设置中修改触发模式）</span>
    </div>

    <div v-if="modelValue.triggerRules?.keywordOnly" class="setting-item">
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
          <span v-for="(kw, i) in modelValue.triggerRules?.keywords ?? []" :key="i" class="keyword-tag">
            {{ kw }}
            <button class="remove-tag" @click="removeKeyword(i)">x</button>
          </span>
        </div>
      </div>
      <span class="setting-hint">添加关键词后，分身只在消息包含这些词时才回复</span>
    </div>

    <div class="setting-item">
      <label>接管冷却期</label>
      <select
        :value="modelValue.takeoverCooldown"
        @change="update('takeoverCooldown', Number(($event.target as HTMLSelectElement).value))"
        class="form-select"
      >
        <option :value="5">5 分钟</option>
        <option :value="10">10 分钟</option>
        <option :value="30">30 分钟</option>
        <option :value="60">1 小时</option>
      </select>
      <span class="setting-hint">点击「接管分身」后，分身暂停回复的时间</span>
    </div>

    <div class="setting-item">
      <label>你发消息后，分身暂停回复</label>
      <select
        :value="modelValue.selfMessagePause ?? 0"
        @change="update('selfMessagePause', Number(($event.target as HTMLSelectElement).value))"
        class="form-select"
      >
        <option :value="0">不暂停</option>
        <option :value="5">5 分钟</option>
        <option :value="10">10 分钟</option>
        <option :value="30">30 分钟</option>
        <option :value="60">1 小时</option>
      </select>
      <span class="setting-hint">你在会话发言后，分身在这段时间内不自动回复，避免插话</span>
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

const modeLabel = computed(() => {
  const parts: string[] = []
  const r = props.modelValue.triggerRules
  if (r?.requireMention) parts.push('被 @ 时回复')
  if (r?.smartDecide) parts.push('智能判断')
  if (r?.keywordOnly) parts.push('关键词命中')
  if (r?.offlineOnly) parts.push('仅离线')
  if (parts.length === 0) return '所有消息'
  return parts.join(' + ')
})

function update<K extends keyof AvatarConfig>(key: K, value: AvatarConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function addKeyword() {
  const kw = keywordInput.value.trim()
  const keywords = props.modelValue.triggerRules?.keywords ?? []
  if (kw && !keywords.includes(kw)) {
    emit('update:modelValue', {
      ...props.modelValue,
      triggerRules: {
        ...props.modelValue.triggerRules,
        keywords: [...keywords, kw]
      }
    })
  }
  keywordInput.value = ''
}

function removeKeyword(index: number) {
  const keywords = [...(props.modelValue.triggerRules?.keywords ?? [])]
  keywords.splice(index, 1)
  emit('update:modelValue', {
    ...props.modelValue,
    triggerRules: { ...props.modelValue.triggerRules, keywords }
  })
}
</script>

<style scoped>
.avatar-trigger-settings-advanced {
  padding: 16px;
}

.current-mode-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 6px;
  margin-bottom: 20px;
}

.mode-label {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.mode-value {
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--primary-color);
}

.mode-hint {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}

.setting-item {
  margin-bottom: 16px;
}

.setting-item > label {
  display: block;
  margin-bottom: 6px;
  font-size: var(--font-size-sm);
  font-weight: 500;
}

.setting-hint {
  display: block;
  margin-top: 4px;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}

.form-select,
.form-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-color);
  font-size: var(--font-size-sm);
  box-sizing: border-box;
}

.form-select:focus,
.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.keyword-input-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.keyword-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.keyword-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: var(--primary-color-alpha, rgba(99, 102, 241, 0.1));
  color: var(--primary-color);
  border-radius: 12px;
  font-size: var(--font-size-xs);
}

.remove-tag {
  background: none;
  border: none;
  color: var(--primary-color);
  cursor: pointer;
  font-size: var(--font-size-sm);
  padding: 0;
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>