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
        @change="updateTriggerMode(($event.target as HTMLSelectElement).value)" 
        class="form-select"
      >
        <option value="mention">被 @ 时回复</option>
        <option value="offline">离线时自动回复</option>
        <option value="smart">智能模式（推荐）</option>
        <option value="keyword">关键词触发</option>
      </select>
      <span class="setting-hint">{{ triggerModeHint }}</span>
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
      <span class="setting-hint">你发消息后，分身暂停回复的时间</span>
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

const triggerModeHint = computed(() => {
  const hints: Record<string, string> = {
    mention: '当有人在私聊中发消息，或在群聊中 @你时，分身会回复',
    offline: '当你离线时，分身自动回复私聊消息',
    smart: '分身会智能判断是否需要回复（推荐）',
    keyword: '仅当消息包含指定关键词时，分身才回复'
  }
  return hints[props.modelValue.triggerRules?.mode ?? ''] || ''
})

function update<K extends keyof AvatarConfigWithApproval>(key: K, value: AvatarConfigWithApproval[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function updateTriggerMode(mode: string) {
  emit('update:modelValue', {
    ...props.modelValue,
    triggerRules: { ...props.modelValue.triggerRules ?? {}, mode: mode as any }
  })
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

.form-input,
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

.form-input:focus,
.form-select:focus {
  outline: none;
  border-color: var(--primary-color);
}
</style>
