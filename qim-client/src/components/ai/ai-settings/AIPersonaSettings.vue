<template>
  <div class="ai-persona-settings">
    <div class="setting-section">
      <label class="section-label">预设人设</label>
      <div class="persona-grid">
        <div
          v-for="p in personas"
          :key="p.value"
          :class="['persona-card', { active: modelValue.aiPersonality === p.value }]"
          @click="update('aiPersonality', p.value)"
        >
          <div class="persona-icon">{{ p.icon }}</div>
          <div class="persona-name">{{ p.name }}</div>
          <div class="persona-desc">{{ p.desc }}</div>
        </div>
      </div>
    </div>

    <div class="setting-section">
      <label class="section-label">自定义系统提示词（可选）</label>
      <textarea
        :value="modelValue.aiCustomPrompt"
        @input="update('aiCustomPrompt', ($event.target as HTMLTextAreaElement).value)"
        class="form-textarea"
        placeholder="输入自定义提示词，将覆盖预设人设。留空则使用预设人设。"
        rows="5"
      ></textarea>
      <span class="setting-hint">自定义提示词优先级高于预设人设</span>
    </div>

    <div class="setting-row">
      <div class="setting-item">
        <label>回复语言</label>
        <select :value="modelValue.aiLanguage" @change="update('aiLanguage', ($event.target as HTMLSelectElement).value)" class="form-select">
          <option value="auto">自动（跟随提问语言）</option>
          <option value="zh">中文</option>
          <option value="en">English</option>
          <option value="ja">日本語</option>
        </select>
      </div>

      <div class="setting-item">
        <label>回复长度</label>
        <select :value="modelValue.aiMaxLength" @change="update('aiMaxLength', ($event.target as HTMLSelectElement).value)" class="form-select">
          <option value="short">简短（1-2句）</option>
          <option value="medium">适中（3-5句）</option>
          <option value="long">详细（不限）</option>
        </select>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { GroupAISettings } from '../../../types/ai'

interface Props { modelValue: GroupAISettings }
interface Emits { (e: 'update:modelValue', value: GroupAISettings): void }

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

function update<K extends keyof GroupAISettings>(key: K, value: GroupAISettings[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

const personas = [
  { value: 'professional', icon: '🎓', name: '专业严谨', desc: '回答专业、严谨、客观' },
  { value: 'casual', icon: '😊', name: '轻松幽默', desc: '语气活泼、善用表情' },
  { value: 'concise', icon: '⚡', name: '简洁高效', desc: '直奔主题、不废话' },
  { value: 'friendly', icon: '🤗', name: '贴心助手', desc: '温暖亲切、有耐心' },
  { value: 'technical', icon: '💻', name: '技术专家', desc: '偏重技术深度和细节' }
]
</script>

<style scoped>
.ai-persona-settings { padding: 16px; }
.setting-section { margin-bottom: 20px; }
.section-label { display: block; margin-bottom: 10px; font-size: 14px; font-weight: 500; }
.persona-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 10px; }
.persona-card { padding: 14px; border: 2px solid var(--border-color); border-radius: 10px; cursor: pointer; text-align: center; transition: all 0.2s; }
.persona-card:hover { border-color: var(--primary-color); }
.persona-card.active { border-color: var(--primary-color); background: var(--primary-color-alpha, rgba(99, 102, 241, 0.1)); }
.persona-icon { font-size: 28px; margin-bottom: 6px; }
.persona-name { font-size: 14px; font-weight: 600; margin-bottom: 4px; }
.persona-desc { font-size: 12px; color: var(--text-secondary); }
.form-textarea { width: 100%; padding: 10px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; resize: vertical; box-sizing: border-box; font-family: inherit; }
.form-textarea:focus { outline: none; border-color: var(--primary-color); }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.setting-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.setting-item label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.form-select { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-select:focus { outline: none; border-color: var(--primary-color); }
</style>
