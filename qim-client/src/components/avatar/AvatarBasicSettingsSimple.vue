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

    <div class="setting-item">
      <button class="link-btn" @click="$emit('goToAdvanced')">
        <i class="fas fa-arrow-right"></i> 查看触发规则详细设置
      </button>
    </div>
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
  'goToAdvanced': []
}>()

const applying = ref(false)

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
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
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

.link-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 0;
  background: none;
  border: none;
  color: var(--primary-color);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  width: 100%;
}

.link-btn:hover {
  opacity: 0.8;
}
</style>