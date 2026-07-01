<template>
  <div class="avatar-basic-settings">
    <!-- 审批状态区域 -->
    <ApprovalStatusSection
      :approval-status="approvalStatus"
      :enabled="modelValue.enabled"
      :reject-reason="modelValue.approvalRejectedReason"
      :applied-at="modelValue.approvalAppliedAt"
      :approved-at="modelValue.approvalReviewedAt"
      :applying="applying"
      @apply="handleApply"
      @cancel="handleCancel"
      @enable="handleEnable"
    />

    <div class="setting-divider"></div>

    <!-- 启用开关 -->
    <div class="setting-item">
      <div class="setting-row">
        <span class="setting-label">启用分身</span>
        <!-- 审批通过后：显示 Switch 可自由开关 -->
        <Switch
          v-if="approvalStatus === 'approved'"
          :model-value="modelValue.enabled"
          :disabled="applying"
          @update:model-value="handleSwitchChange"
        />
        <!-- 审批中：显示状态标签 -->
        <span v-else-if="approvalStatus === 'pending'" class="status-tag pending">
          审批中
        </span>
        <!-- 被拒绝：显示重新申请按钮 -->
        <button v-else-if="approvalStatus === 'rejected'" class="btn-apply" @click="handleApply" :disabled="applying">
          {{ applying ? '申请中...' : '重新申请' }}
        </button>
        <!-- 未申请 -->
        <button v-else class="btn-apply" @click="handleApply" :disabled="applying">
          {{ applying ? '申请中...' : '申请启用' }}
        </button>
      </div>
      <span class="setting-hint" v-if="applying">
        处理中...
      </span>
      <span class="setting-hint" v-else-if="approvalStatus === 'pending'">
        分身正在审批中，请等待管理员审核
      </span>
      <span class="setting-hint" v-else-if="approvalStatus === 'rejected'">
        申请已被拒绝，请修改配置后重新申请
      </span>
      <span class="setting-hint" v-else-if="!modelValue.enabled && approvalStatus === 'approved'">
        分身已通过审批，可开启使用
      </span>
      <span class="setting-hint" v-else-if="modelValue.enabled">
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
// Switch 变更处理：开启走审批，关闭直接生效
async function handleSwitchChange(value: boolean) {
  if (value) {
    await handleApply()
  } else {
    update('enabled', false)
  }
}

// 审批状态
const approvalStatus = computed<AvatarApprovalStatus>(() => {
  return props.modelValue.approvalStatus || 'none'
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
  const confirmResult = await window.$QMessageBox.confirm('确定要取消申请吗？', '取消申请')
  if (confirmResult.action !== 'confirm') return

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
}

async function handleEnable() {
  applying.value = true
  try {
    const result = await avatarAPI.applyForApproval()
    emit('update:modelValue', result)
    window.$QMessage.success('已提交启用申请')
  } catch (e: any) {
    window.$QMessage.error(e.response?.data?.message || '启用失败')
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
@import './avatar-shared.css';

.avatar-basic-settings { padding: 16px; }
</style>
