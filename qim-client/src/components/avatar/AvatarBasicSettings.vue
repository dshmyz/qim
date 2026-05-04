<template>
  <div class="avatar-basic-settings">
    <!-- 审批状态区域 -->
    <div class="approval-status-section">
      <div class="approval-status-header">
        <span class="status-label">审批状态</span>
        <span :class="['status-badge', approvalStatus]">
          <i :class="statusIcon"></i>
          {{ statusText }}
        </span>
      </div>

      <!-- 未申请状态 -->
      <div v-if="approvalStatus === 'none'" class="approval-action">
        <p class="approval-hint">分身功能需要管理员审批后才能启用</p>
        <button class="btn btn-primary" @click="handleApply" :disabled="applying">
          <i class="fas fa-paper-plane"></i>
          {{ applying ? '申请中...' : '申请启用' }}
        </button>
      </div>

      <!-- 待审批状态 -->
      <div v-else-if="approvalStatus === 'pending'" class="approval-action">
        <p class="approval-hint">您的申请已提交，请等待管理员审批</p>
        <p class="approval-time" v-if="modelValue.approvalAppliedAt">
          申请时间：{{ formatDate(modelValue.approvalAppliedAt) }}
        </p>
        <button class="btn btn-secondary" @click="handleCancel" :disabled="applying">
          <i class="fas fa-times"></i>
          取消申请
        </button>
      </div>

      <!-- 已通过状态 -->
      <div v-else-if="approvalStatus === 'approved'" class="approval-action approved">
        <p class="approval-hint success">
          <i class="fas fa-check-circle"></i>
          您的分身功能已通过审批，可以启用
        </p>
      </div>

      <!-- 已拒绝状态 -->
      <div v-else-if="approvalStatus === 'rejected'" class="approval-action rejected">
        <div class="reject-reason">
          <p class="approval-hint error">
            <i class="fas fa-exclamation-circle"></i>
            您的申请已被拒绝
          </p>
          <p class="reason-text" v-if="modelValue.approvalRejectedReason">
            拒绝原因：{{ modelValue.approvalRejectedReason }}
          </p>
          <p class="approval-time" v-if="modelValue.approvalReviewedAt">
            审批时间：{{ formatDate(modelValue.approvalReviewedAt) }}
          </p>
        </div>
        <button class="btn btn-primary" @click="handleApply" :disabled="applying">
          <i class="fas fa-redo"></i>
          {{ applying ? '申请中...' : '重新申请' }}
        </button>
      </div>
    </div>

    <div class="setting-divider"></div>

    <!-- 启用开关 -->
    <div class="setting-item">
      <label class="toggle-label">
        <span>启用分身</span>
        <label class="switch">
          <input 
            type="checkbox" 
            :checked="modelValue.enabled" 
            @change="handleEnabledChange"
            :disabled="!canEnable"
          />
          <span class="slider round"></span>
        </label>
      </label>
      <span class="setting-hint" v-if="!canEnable">
        需要先通过审批才能启用分身
      </span>
      <span class="setting-hint" v-else>
        开启后，分身将在你设定的规则下代替你回复消息
      </span>
    </div>

    <div class="setting-item">
      <label>分身名称</label>
      <input :value="modelValue.name" @input="update('name', ($event.target as HTMLInputElement).value)" class="form-input" placeholder="我的分身" />
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

const props = defineProps<{
  modelValue: AvatarConfigWithApproval
  modelConfigs: AIConfig[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfigWithApproval]
}>()

const applying = ref(false)

// 审批状态
const approvalStatus = computed<AvatarApprovalStatus>(() => {
  return props.modelValue.approvalStatus || 'none'
})

// 是否可以启用分身
const canEnable = computed(() => {
  return approvalStatus.value === 'approved'
})

// 状态文本
const statusText = computed(() => {
  const texts: Record<AvatarApprovalStatus, string> = {
    none: '未申请',
    pending: '审批中',
    approved: '已通过',
    rejected: '已拒绝'
  }
  return texts[approvalStatus.value]
})

// 状态图标
const statusIcon = computed(() => {
  const icons: Record<AvatarApprovalStatus, string> = {
    none: 'fas fa-minus-circle',
    pending: 'fas fa-clock',
    approved: 'fas fa-check-circle',
    rejected: 'fas fa-times-circle'
  }
  return icons[approvalStatus.value]
})

// 格式化日期
function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

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
</script>

<style scoped>
.avatar-basic-settings { padding: 16px; }

/* 审批状态区域 */
.approval-status-section {
  background: var(--hover-color);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
}

.approval-status-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.status-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 13px;
  font-weight: 500;
}

.status-badge.none {
  background: var(--color-gray-100);
  color: var(--text-secondary);
}

.status-badge.pending {
  background: rgba(245, 158, 11, 0.1);
  color: #F59E0B;
}

.status-badge.approved {
  background: rgba(16, 185, 129, 0.1);
  color: #10B981;
}

.status-badge.rejected {
  background: rgba(239, 68, 68, 0.1);
  color: #EF4444;
}

.approval-action {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.approval-hint {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0;
}

.approval-hint.success {
  color: #10B981;
}

.approval-hint.error {
  color: #EF4444;
}

.approval-time {
  font-size: 12px;
  color: var(--text-secondary);
  margin: 0;
}

.reject-reason {
  margin-bottom: 8px;
}

.reason-text {
  font-size: 13px;
  color: var(--text-secondary);
  background: rgba(239, 68, 68, 0.05);
  padding: 8px 12px;
  border-radius: 6px;
  margin: 8px 0 0 0;
}

.setting-divider {
  height: 1px;
  background: var(--border-color);
  margin: 16px 0;
}

/* 按钮样式 */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--bg-color);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--hover-color);
}

/* 设置项样式 */
.setting-item { margin-bottom: 16px; }
.setting-item > label { display: block; margin-bottom: 6px; font-size: 14px; font-weight: 500; }
.toggle-label { display: flex; align-items: center; justify-content: space-between; cursor: pointer; }
.setting-hint { display: block; margin-top: 4px; font-size: 12px; color: var(--text-secondary); }
.setting-hint.error { color: #F44336; }
.form-select, .form-input { width: 100%; padding: 8px 12px; border: 1px solid var(--border-color); border-radius: 6px; background: var(--bg-color); color: var(--text-color); font-size: 14px; box-sizing: border-box; }
.form-select:focus, .form-input:focus { outline: none; border-color: var(--primary-color); }
.switch { position: relative; display: inline-block; width: 50px; height: 24px; min-width: 50px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background-color: #ccc; transition: 0.4s; border-radius: 24px; }
.slider:before { position: absolute; content: ''; height: 18px; width: 18px; left: 3px; bottom: 3px; background-color: white; transition: 0.4s; border-radius: 50%; }
input:checked + .slider { background-color: var(--primary-color); }
input:checked + .slider:before { transform: translateX(26px); }
input:disabled + .slider { opacity: 0.5; cursor: not-allowed; }
.slider.round { border-radius: 24px; }
</style>
