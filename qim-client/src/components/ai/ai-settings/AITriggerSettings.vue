<template>
  <div class="ai-trigger-settings">
    <div class="setting-item">
      <label>回复模式</label>
      <select :value="modelValue.aiReplyMode" @change="update('aiReplyMode', ($event.target as HTMLSelectElement).value)" class="form-select">
        <option value="mention_only">仅被 @ 时回复</option>
        <option value="smart">智能判断回复</option>
        <option value="always">始终回复</option>
        <option value="off">关闭 AI 回复</option>
      </select>
    </div>

    <div class="setting-item">
      <label class="toggle-label">
        <span>@ 后回复方式</span>
        <label class="switch">
          <input type="checkbox" :checked="modelValue.aiMentionReplyMode === 'mention'" @change="update('aiMentionReplyMode', ($event.target as HTMLInputElement).checked ? 'mention' : 'direct')" />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint">{{ modelValue.aiMentionReplyMode === 'mention' ? '@提问者后回复' : '直接回复' }}</span>
    </div>

    <div class="setting-item">
      <label>防刷屏间隔</label>
      <select :value="modelValue.aiAntiSpamInterval" @change="update('aiAntiSpamInterval', Number(($event.target as HTMLSelectElement).value))" class="form-select">
        <option :value="0">关闭</option>
        <option :value="3">3 分钟</option>
        <option :value="5">5 分钟</option>
        <option :value="10">10 分钟</option>
        <option :value="15">15 分钟</option>
      </select>
      <span class="setting-hint">同一话题在此间隔内只回复一次</span>
    </div>

    <div class="setting-item">
      <label>触发关键词（可选）</label>
      <div class="keyword-input-wrapper">
        <input
          :value="keywordInput"
          @input="keywordInput = ($event.target as HTMLInputElement).value"
          @keydown.enter.prevent="addKeyword"
          class="form-input"
          placeholder="输入关键词后按回车"
        />
        <div class="keyword-tags">
          <span v-for="(kw, i) in modelValue.aiTriggerKeywords" :key="i" class="keyword-tag">
            {{ kw }}
            <button class="remove-tag" @click="removeKeyword(i)">x</button>
          </span>
        </div>
      </div>
      <span class="setting-hint">设置后仅当消息包含关键词时 AI 才触发（留空则不限）</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { GroupAISettings } from '../../../types/ai'

interface Props { modelValue: GroupAISettings }
interface Emits { (e: 'update:modelValue', value: GroupAISettings): void }

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const keywordInput = ref('')

function update<K extends keyof GroupAISettings>(key: K, value: GroupAISettings[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function addKeyword() {
  const kw = keywordInput.value.trim()
  if (kw && !props.modelValue.aiTriggerKeywords.includes(kw)) {
    update('aiTriggerKeywords', [...props.modelValue.aiTriggerKeywords, kw])
  }
  keywordInput.value = ''
}

function removeKeyword(index: number) {
  const keywords = [...props.modelValue.aiTriggerKeywords]
  keywords.splice(index, 1)
  update('aiTriggerKeywords', keywords)
}
</script>

<style scoped>
.ai-trigger-settings { padding: 16px; }
.setting-item { margin-bottom: 16px; }
.setting-item label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.toggle-label { display: flex; align-items: center; justify-content: space-between; cursor: pointer; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.form-select, .form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-select:focus, .form-input:focus { outline: none; border-color: var(--primary-color); }
.switch { position: relative; display: inline-block; width: 50px; height: 24px; min-width: 50px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: #ccc; transition: 0.4s; border-radius: 24px; }
.slider:before { position: absolute; content: ''; height: 18px; width: 18px; left: 3px; bottom: 3px; background-color: white; transition: 0.4s; border-radius: 50%; }
input:checked + .slider { background-color: var(--primary-color); }
input:checked + .slider:before { transform: translateX(26px); }
.slider.round { border-radius: 24px; }
.keyword-input-wrapper { display: flex; flex-direction: column; gap: 8px; }
.keyword-tags { display: flex; flex-wrap: wrap; gap: 6px; }
.keyword-tag { display: inline-flex; align-items: center; gap: 4px; padding: 4px 10px; background: var(--primary-color-alpha, rgba(99, 102, 241, 0.1)); color: var(--primary-color); border-radius: 12px; font-size: 13px; }
.remove-tag { background: none; border: none; color: var(--primary-color); cursor: pointer; font-size: 14px; padding: 0; width: 16px; height: 16px; display: flex; align-items: center; justify-content: center; }
</style>
