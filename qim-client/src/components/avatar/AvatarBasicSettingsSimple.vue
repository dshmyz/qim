<template>
  <div class="avatar-basic-settings-simple">
    <ApprovalStatusSection
      :approval-status="approvalStatus"
      :reject-reason="modelValue.approvalRejectedReason"
      :applied-at="modelValue.approvalAppliedAt"
      :approved-at="modelValue.approvalReviewedAt"
      :applying="applying"
      @apply="handleApply"
      @cancel="handleCancel"
    />

    <div class="setting-divider"></div>

    <div class="setting-item">
      <div class="setting-row">
        <span class="setting-label">启用分身</span>
        <Switch 
          v-model="localEnabled" 
          :disabled="!canEnable"
        />
      </div>
      <span class="setting-hint" v-if="!canEnable">
        需要先通过审批才能启用分身
      </span>
      <span class="setting-hint" v-else>
        开启后，分身将在你设定的规则下代替你回复消息
      </span>
    </div>

    <div class="setting-item">
      <label>分身名称</label>
      <input 
        :value="modelValue.name" 
        @input="handleNameInput" 
        class="form-input" 
        placeholder="我的分身" 
        maxlength="20"
      />
      <span class="setting-hint">其他人在私聊中看到的分身名称</span>
    </div>

    <div class="setting-item">
      <label>触发模式</label>
      <select 
        :value="modelValue.triggerRules?.mode ?? 'mention'" 
        @change="handleTriggerModeChange" 
        class="form-select"
      >
        <option value="mention">被 @ 时回复</option>
        <option value="offline">离线时自动回复</option>
        <option value="smart">智能模式</option>
        <option value="keyword">关键词触发</option>
        <option value="all">所有消息</option>
      </select>
      <span class="setting-hint">设置分身何时自动回复消息</span>
    </div>

    <div class="setting-item">
      <label>接管冷却期</label>
      <select
        :value="modelValue.takeoverCooldown ?? 10"
        @change="update('takeoverCooldown', Number(($event.target as HTMLSelectElement).value))"
        class="form-select"
      >
        <option :value="5">5 分钟</option>
        <option :value="10">10 分钟</option>
        <option :value="30">30 分钟</option>
        <option :value="60">1 小时</option>
      </select>
      <span class="setting-hint">你发消息后，分身暂停回复的时间</span>
    </div>

    <template v-if="modelValue.triggerRules?.mode === 'keyword' || modelValue.triggerRules?.mode === 'smart'">
      <div class="setting-divider"></div>
      <div class="setting-item">
        <label>触发关键词</label>
        <div class="keyword-input-wrapper">
          <input
            :value="keywordInput"
            @input="keywordInput = ($event.target as HTMLInputElement).value"
            @keydown.enter.prevent="addKeyword"
            class="form-input keyword-field"
            placeholder="输入关键词后按回车"
          />
          <div class="keyword-tags">
            <span v-for="(kw, i) in modelValue.triggerRules?.keywords ?? []" :key="i" class="keyword-tag">
              {{ kw }}
              <button class="remove-tag" @click="removeKeyword(i)">
                <i class="fas fa-times"></i>
              </button>
            </span>
          </div>
        </div>
        <span class="setting-hint">添加关键词后，分身只在消息包含这些词时才回复</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { AvatarConfigWithApproval, AvatarApprovalStatus } from '../../types/avatar'
import ApprovalStatusSection from './ApprovalStatusSection.vue'
import Switch from '../common/Switch.vue'
import { avatarAPI } from '../../api/avatar'
import { validateAliasName } from '../../utils/validation'

const props = defineProps<{
  modelValue: AvatarConfigWithApproval
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfigWithApproval]
}>()

const applying = ref(false)
const keywordInput = ref('')

const localEnabled = computed({
  get: () => props.modelValue?.enabled ?? false,
  set: (value: boolean) => {
    if (canEnable.value) {
      update('enabled', value)
    }
  }
})

const approvalStatus = computed<AvatarApprovalStatus>(() => {
  return props.modelValue.approvalStatus || 'none'
})

const canEnable = computed(() => {
  return approvalStatus.value === 'approved'
})

const modeLabel = computed(() => {
  const labels: Record<string, string> = {
    mention: '被 @ 时回复',
    offline: '离线时自动回复',
    smart: '智能模式',
    keyword: '关键词触发',
    all: '所有消息'
  }
  return labels[props.modelValue.triggerRules?.mode ?? 'mention'] || '未设置'
})

const cooldownLabel = computed(() => {
  const minutes = props.modelValue.takeoverCooldown ?? 10
  if (minutes >= 60) return `${minutes / 60} 小时`
  return `${minutes} 分钟`
})

function update<K extends keyof AvatarConfigWithApproval>(key: K, value: AvatarConfigWithApproval[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

async function handleApply() {
  applying.value = true
  try {
    const result = await avatarAPI.applyForApproval()
    emit('update:modelValue', result)
  } catch (error) {
    console.error('申请审批失败', error)
  } finally {
    applying.value = false
  }
}

async function handleCancel() {
  applying.value = true
  try {
    const result = await avatarAPI.cancelApplication()
    emit('update:modelValue', result)
  } catch (error) {
    console.error('取消申请失败', error)
  } finally {
    applying.value = false
  }
}

function handleNameInput(event: Event) {
  const value = (event.target as HTMLInputElement).value
  const result = validateAliasName(value)
  if (!result.valid) {
    window.$QMessage.warning(result.message)
    return
  }
  update('name', value)
}

function handleTriggerModeChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value as 'mention' | 'offline' | 'keyword' | 'all' | 'smart'
  emit('update:modelValue', {
    ...props.modelValue,
    triggerRules: {
      ...props.modelValue.triggerRules,
      mode: value
    }
  })
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
.avatar-basic-settings-simple {
  padding: 16px;
}

.setting-divider {
  height: 1px;
  background: var(--border-color);
  margin: 16px 0;
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

.setting-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.setting-label {
  font-size: 14px;
  font-weight: 500;
}

.setting-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.form-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-color);
  font-size: 14px;
  box-sizing: border-box;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px var(--primary-color-alpha, rgba(99, 102, 241, 0.15));
}

.form-select {
  appearance: none;
  -webkit-appearance: none;
  width: 100%;
  padding: 8px 36px 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color) url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23666' d='M6 8.825L1.175 4 2.238 2.938 6 6.7l3.763-3.762L10.825 4z'/%3E%3C/svg%3E") no-repeat right 12px center;
  color: var(--text-color);
  font-size: 14px;
  box-sizing: border-box;
  cursor: pointer;
  transition: border-color 0.2s, box-shadow 0.2s, background-color 0.2s;
}

.form-select:hover {
  border-color: var(--text-secondary);
}

.form-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px var(--primary-color-alpha, rgba(99, 102, 241, 0.15));
}

.trigger-info {
  background: var(--hover-color, rgba(0, 0, 0, 0.03));
  border-radius: 6px;
  padding: 10px 12px;
  margin-bottom: 8px;
}

.trigger-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
}

.trigger-label {
  font-size: 13px;
  color: var(--text-secondary);
}

.trigger-value {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.keyword-input-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.keyword-field {
  margin-bottom: 0;
}

.keyword-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-height: 24px;
}

.keyword-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: var(--primary-color-alpha, rgba(99, 102, 241, 0.1));
  color: var(--primary-color);
  border-radius: 12px;
  font-size: 13px;
  animation: tag-fade-in 0.15s ease;
}

@keyframes tag-fade-in {
  from { opacity: 0; transform: scale(0.9); }
  to { opacity: 1; transform: scale(1); }
}

.remove-tag {
  background: none;
  border: none;
  color: var(--primary-color);
  cursor: pointer;
  font-size: 12px;
  padding: 0;
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: background 0.15s;
}

.remove-tag:hover {
  background: rgba(99, 102, 241, 0.2);
}
</style>