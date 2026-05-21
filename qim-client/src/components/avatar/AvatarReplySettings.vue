<template>
  <div class="avatar-reply-settings">
    <div class="setting-section">
      <div class="section-header">
        <i class="fas fa-sliders-h"></i>
        <h4>回复控制</h4>
      </div>

      <div class="setting-item">
        <label class="setting-label">回复长度</label>
        <select :value="replyStrategy?.maxReplyLength ?? 'medium'" @change="updateStrategy('maxReplyLength', ($event.target as HTMLSelectElement).value as 'short' | 'medium' | 'long')" class="form-select">
          <option value="short">简短（1-2 句）</option>
          <option value="medium">适中（3-5 句）</option>
          <option value="long">详细（6 句以上）</option>
        </select>
      </div>

      <div class="setting-item">
        <label class="setting-label">回复延迟</label>
        <select :value="replyStrategy?.replyDelay ?? 3" @change="updateStrategy('replyDelay', Number(($event.target as HTMLSelectElement).value))" class="form-select">
          <option :value="0">无延迟</option>
          <option :value="3">3 秒</option>
          <option :value="5">5 秒</option>
          <option :value="10">10 秒</option>
        </select>
        <span class="setting-hint">模拟真人思考时间，避免回复过快显得不自然</span>
      </div>

      <div class="setting-item">
        <label class="setting-label">置信度阈值</label>
        <div class="threshold-slider">
          <input type="range" :value="replyStrategy?.confidenceThreshold ?? 0.6" @input="updateStrategy('confidenceThreshold', Number(($event.target as HTMLInputElement).value))" min="0" max="1" step="0.1" class="slider-input" />
          <span class="threshold-value">{{ ((replyStrategy?.confidenceThreshold ?? 0.6) * 100).toFixed(0) }}%</span>
        </div>
        <span class="setting-hint">低于此阈值时分身不会回复，而是通知你亲自回复</span>
      </div>
    </div>

    <div class="setting-section">
      <div class="section-header">
        <i class="fas fa-tag"></i>
        <h4>回复标记</h4>
      </div>

      <div class="setting-item">
        <label class="setting-label">AI 标记样式</label>
        <select :value="replyStrategy?.disclaimerStyle ?? 'badge'" @change="updateStrategy('disclaimerStyle', ($event.target as HTMLSelectElement).value as 'badge' | 'footer' | 'both')" class="form-select">
          <option value="badge">徽章标记</option>
          <option value="footer">底部标注</option>
          <option value="both">两者都有</option>
        </select>
        <span class="setting-hint">分身回复消息中"AI 代回复"标记的展示方式</span>
      </div>
    </div>

    <div class="setting-section">
      <div class="section-header">
        <i class="fas fa-filter"></i>
        <h4>知识范围控制</h4>
      </div>

      <div class="setting-item">
        <label class="toggle-label">
          <div class="label-content">
            <span class="label-title">回复知识范围外的消息</span>
            <span class="label-hint">关闭时，分身只在有相关知识内容时才回复，超出知识范围的消息会静默跳过不回复</span>
          </div>
          <label class="switch">
            <input type="checkbox" :checked="replyStrategy?.replyOutOfScope ?? false" @change="updateStrategy('replyOutOfScope', ($event.target as HTMLInputElement).checked)" />
            <span class="slider round"></span>
          </label>
        </label>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AvatarConfig, AvatarReplyStrategy } from '../../types/avatar'

const props = defineProps<{
  modelValue: AvatarConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

const replyStrategy = computed(() => props.modelValue?.replyStrategy)

function updateStrategy<K extends keyof AvatarReplyStrategy>(key: K, value: AvatarReplyStrategy[K]) {
  const currentStrategy = props.modelValue.replyStrategy || {
    maxReplyLength: 'medium',
    replyDelay: 3,
    confidenceThreshold: 0.6,
    disclaimerStyle: 'badge',
    replyOutOfScope: false
  }
  emit('update:modelValue', {
    ...props.modelValue,
    replyStrategy: { ...currentStrategy, [key]: value }
  })
}
</script>

<style scoped>
.avatar-reply-settings { 
  padding: 16px; 
  min-height: 100%;
}

.setting-section {
  background: var(--card-bg);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
}

.setting-section:last-child {
  margin-bottom: 0;
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

.setting-label { 
  display: block; 
  margin-bottom: 8px; 
  font-size: 14px; 
  font-weight: 500;
  color: var(--text-primary);
}

.setting-hint { 
  display: block; 
  margin-top: 6px; 
  font-size: 12px; 
  color: var(--text-secondary); 
}

.form-select {
  width: 100%;
  padding: 10px 36px 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color) url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' fill='none' viewBox='0 0 20 20'%3e%3cpath stroke='%236b7280' stroke-linecap='round' stroke-linejoin='round' stroke-width='1.5' d='M6 8l4 4 4-4'/%3e%3c/svg%3e") no-repeat right 10px center;
  color: var(--text-color);
  font-size: 14px;
  box-sizing: border-box;
  appearance: none;
  -webkit-appearance: none;
  cursor: pointer;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.form-select:hover {
  border-color: var(--text-secondary);
}

.form-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px var(--primary-color-alpha, rgba(99, 102, 241, 0.15));
}

.threshold-slider { 
  display: flex; 
  align-items: center; 
  gap: 12px; 
}

.slider-input { 
  flex: 1;
  height: 6px;
  -webkit-appearance: none;
  appearance: none;
  background: var(--border-color);
  border-radius: 3px;
  outline: none;
}

.slider-input::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 18px;
  height: 18px;
  background: var(--primary-color);
  border-radius: 50%;
  cursor: pointer;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
  transition: transform 0.2s;
}

.slider-input::-webkit-slider-thumb:hover {
  transform: scale(1.1);
}

.slider-input::-moz-range-thumb {
  width: 18px;
  height: 18px;
  background: var(--primary-color);
  border: none;
  border-radius: 50%;
  cursor: pointer;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
}

.threshold-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--primary-color);
  min-width: 50px;
  text-align: right;
}

.toggle-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
}

.toggle-label .label-content {
  flex: 1;
  margin-right: 12px;
}

.toggle-label .label-title {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.toggle-label .label-hint {
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

.slider.round {
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

.slider.round:before {
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

input:checked + .slider.round {
  background-color: var(--primary-color);
}

input:checked + .slider.round:before {
  transform: translateX(24px);
}
</style>
