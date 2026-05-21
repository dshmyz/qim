<template>
  <div class="avatar-basic-settings">
    <!-- 审批状态区域 -->
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

    <!-- 启用开关 -->
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
      <input :value="modelValue.name" @input="handleNameInput" class="form-input" placeholder="我的分身" maxlength="20" />
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
import { computed, ref } from 'vue'
import type { AvatarConfigWithApproval, AvatarApprovalStatus } from '../../types/avatar'
import type { UserAIConfig as AIConfig } from '../../types/ai'
import { avatarAPI } from '../../api/avatar'
import ApprovalStatusSection from './ApprovalStatusSection.vue'
import Switch from '../common/Switch.vue'
import { validateAliasName } from '../../utils/validation'

const props = defineProps<{
  modelValue: AvatarConfigWithApproval
  modelConfigs: AIConfig[]
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

// 审批状态
const approvalStatus = computed<AvatarApprovalStatus>(() => {
  return props.modelValue.approvalStatus || 'none'
})

// 是否可以启用分身
const canEnable = computed(() => {
  return approvalStatus.value === 'approved'
})

function update<K extends keyof AvatarConfigWithApproval>(key: K, value: AvatarConfigWithApproval[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function handleModelSourceChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  update('useSystemConfig', value === 'system')
  if (value === 'system') {
    update('modelConfigId', null)
  }
}

function handleEnabledChange(event: Event) {
  if (!canEnable.value) return
  update('enabled', (event.target as HTMLInputElement).checked)
}

async function handleApply() {
  applying.value = true
  try {
    const result = await avatarAPI.applyForApproval()
    emit('update:modelValue', result)
    window.$QMessage.success('申请已提交')
  } catch (e: any) {
    window.$QMessage.error(e.response?.data?.message || '申请失败')
  } finally {
    applying.value = false
  }
}

async function handleCancel() {
  try {
    await window.$QMessageBox.confirm('确定要取消申请吗？', '取消申请')
    applying.value = true
    try {
      const result = await avatarAPI.cancelApplication()
      emit('update:modelValue', result)
      window.$QMessage.success('已取消申请')
    } catch (e: any) {
      window.$QMessage.error(e.response?.data?.message || '取消失败')
    } finally {
      applying.value = false
    }
  } catch {
    // 用户取消
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
.avatar-basic-settings { padding: 16px; }

.setting-divider {
  height: 1px;
  background: var(--border-color);
  margin: 16px 0;
}

/* 设置项样式 */
.setting-item { margin-bottom: 16px; }
.setting-item > label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.setting-row { display: flex; align-items: center; justify-content: space-between; }
.setting-label { font-size: 14px; font-weight: 500; color: var(--text-color); }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.setting-hint.error { color: #F44336; }
.form-select, .form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; transition: border-color 0.2s, box-shadow 0.2s; }
.form-select:focus, .form-input:focus { outline: none; border-color: var(--primary-color); box-shadow: 0 0 0 2px var(--primary-color-alpha, rgba(99, 102, 241, 0.15)); }
</style>
