<template>
  <div class="avatar-reply-settings">
    <div class="setting-item">
      <label>回复长度</label>
      <select :value="modelValue.replyStrategy.maxReplyLength" @change="updateStrategy('maxReplyLength', ($event.target as HTMLSelectElement).value)" class="form-select">
        <option value="short">简短（1-2 句）</option>
        <option value="medium">适中（3-5 句）</option>
        <option value="long">详细（6 句以上）</option>
      </select>
    </div>

    <div class="setting-item">
      <label>回复延迟</label>
      <select :value="modelValue.replyStrategy.replyDelay" @change="updateStrategy('replyDelay', Number(($event.target as HTMLSelectElement).value))" class="form-select">
        <option :value="0">无延迟</option>
        <option :value="3">3 秒</option>
        <option :value="5">5 秒</option>
        <option :value="10">10 秒</option>
      </select>
      <span class="setting-hint">模拟真人思考时间，避免回复过快显得不自然</span>
    </div>

    <div class="setting-item">
      <label>置信度阈值</label>
      <div class="threshold-slider">
        <input type="range" :value="modelValue.replyStrategy.confidenceThreshold" @input="updateStrategy('confidenceThreshold', Number(($event.target as HTMLInputElement).value))" min="0" max="1" step="0.1" class="slider-input" />
        <span class="threshold-value">{{ (modelValue.replyStrategy.confidenceThreshold * 100).toFixed(0) }}%</span>
      </div>
      <span class="setting-hint">低于此阈值时分身不会回复，而是通知你亲自回复</span>
    </div>

    <div class="setting-item">
      <label>AI 标记样式</label>
      <select :value="modelValue.replyStrategy.disclaimerStyle" @change="updateStrategy('disclaimerStyle', ($event.target as HTMLSelectElement).value)" class="form-select">
        <option value="badge">徽章标记</option>
        <option value="footer">底部标注</option>
        <option value="both">两者都有</option>
      </select>
      <span class="setting-hint">分身回复消息中"AI 代回复"标记的展示方式</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AvatarConfig, AvatarReplyStrategy } from '../../types/avatar'

const props = defineProps<{
  modelValue: AvatarConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

function updateStrategy<K extends keyof AvatarReplyStrategy>(key: K, value: AvatarReplyStrategy[K]) {
  emit('update:modelValue', {
    ...props.modelValue,
    replyStrategy: { ...props.modelValue.replyStrategy, [key]: value }
  })
}
</script>

<style scoped>
.avatar-reply-settings { padding: 16px; }
.setting-item { margin-bottom: 16px; }
.setting-item > label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.form-select { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-select:focus { outline: none; border-color: var(--primary-color); }
.threshold-slider { display: flex; align-items: center; gap: 12px; }
.slider-input { flex: 1; }
.threshold-value { font-size: 14px; font-weight: 500; color: var(--primary-color); min-width: 40px; }
</style>
