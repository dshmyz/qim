<template>
  <div class="avatar-basic-settings-simple">
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
        <!-- 未申请（一般不会出现，创建时已自动申请） -->
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
      <div class="setting-row">
        <span class="setting-label">仅在被 @ 时回复</span>
        <Switch
          :model-value="!!modelValue.triggerRules?.requireMention"
          @update:model-value="updateTrigger('requireMention', $event)"
        />
      </div>
      <span class="setting-hint">仅群聊生效：群里被 @ 才回复；私聊没有 @ 语义，会自动改由智能判断是否回复</span>
    </div>

    <div class="setting-item">
      <div class="setting-row">
        <span class="setting-label">智能判断是否需要回复</span>
        <Switch
          :model-value="!!modelValue.triggerRules?.smartDecide"
          @update:model-value="updateTrigger('smartDecide', $event)"
        />
      </div>
      <span class="setting-hint">让 AI 判断这条消息是否值得你回复，避免无关闲聊触发分身</span>
    </div>

    <div class="setting-item">
      <div class="setting-row">
        <span class="setting-label">关键词命中才回复</span>
        <Switch
          :model-value="!!modelValue.triggerRules?.keywordOnly"
          @update:model-value="updateTrigger('keywordOnly', $event)"
        />
      </div>
      <span class="setting-hint">仅当消息包含下方关键词时才回复</span>
    </div>

    <div class="setting-item">
      <div class="setting-row">
        <span class="setting-label">仅离线时自动回复</span>
        <Switch
          :model-value="!!modelValue.triggerRules?.offlineOnly"
          @update:model-value="updateTrigger('offlineOnly', $event)"
        />
      </div>
      <span class="setting-hint">触发时机：仅在你离线时才自动回复，作为上面意图条件的叠加门槛；未勾选则无论在线与否都可回</span>
    </div>

    <div class="setting-item">
      <span class="setting-hint setting-note">上面的意图开关（被 @ / 智能判断 / 关键词）均未勾选时，分身会回复该范围内的所有消息。</span>
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
      <span class="setting-hint">点击「接管分身」后，分身暂停回复的时间</span>
    </div>

    <div class="setting-item">
      <label>你发消息后，分身暂停回复</label>
      <select
        :value="modelValue.selfMessagePause ?? 0"
        @change="update('selfMessagePause', Number(($event.target as HTMLSelectElement).value))"
        class="form-select"
      >
        <option :value="0">不暂停</option>
        <option :value="5">5 分钟</option>
        <option :value="10">10 分钟</option>
        <option :value="30">30 分钟</option>
        <option :value="60">1 小时</option>
      </select>
      <span class="setting-hint">你在会话发言后，分身在这段时间内不自动回复，避免插话；选「不暂停」则发言后照常可回</span>
    </div>

    <div class="setting-item">
      <div class="setting-row">
        <span class="setting-label">默认在所有会话激活</span>
        <Switch
          :model-value="modelValue.activateByDefault"
          @update:model-value="update('activateByDefault', $event)"
        />
      </div>
      <span class="setting-hint">开启后，未单独设置的会话（含新建）自动激活分身，可逐个关闭；关闭则需逐会话手动开启</span>
    </div>

    <div class="setting-item">
      <button class="btn-reset-sessions" :disabled="resettingSessions" @click="handleResetSessions">
        {{ resettingSessions ? '重置中...' : '重置所有会话设置为跟随全局默认' }}
      </button>
      <span class="setting-hint">清除所有会话单独设置，未单独设置的会话将重新按上方「默认在所有会话激活」开关生效</span>
    </div>

    <template v-if="modelValue.triggerRules?.keywordOnly">
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
import type { AvatarConfigWithApproval, AvatarApprovalStatus, AvatarTriggerRules } from '../../types/avatar'
import ApprovalStatusSection from './ApprovalStatusSection.vue'
import Switch from '../common/Switch.vue'
import { avatarAPI } from '../../api/avatar'
import { useAvatar } from '../../composables/useAvatar'
import { validateAliasName } from '../../utils/validation'

const props = defineProps<{
  modelValue: AvatarConfigWithApproval
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfigWithApproval]
}>()

const { clearSessions: clearAllSessions } = useAvatar()

const applying = ref(false)
const resettingSessions = ref(false)
const keywordInput = ref('')

// Switch 变更处理：开启走审批，关闭直接生效
async function handleSwitchChange(value: boolean) {
  if (value) {
    await handleApply()
  } else {
    update('enabled', false)
  }
}

const approvalStatus = computed<AvatarApprovalStatus>(() => {
  return props.modelValue.approvalStatus || 'none'
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

async function handleEnable() {
  applying.value = true
  try {
    const result = await avatarAPI.applyForApproval()
    emit('update:modelValue', result)
  } catch (error) {
    console.error('启用分身失败', error)
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

async function handleResetSessions() {
  const result = await window.$QMessageBox?.confirm(
    '将清除所有会话的单独设置，之后未单独设置的会话按上方「默认在所有会话激活」开关生效。确定重置吗？',
    '重置会话设置',
    { confirmButtonText: '确认重置', type: 'warning' }
  )
  if (result?.action !== 'confirm') return
  resettingSessions.value = true
  try {
    await clearAllSessions()
    window.$QMessage?.success?.('已重置所有会话设置为跟随全局默认')
  } catch {
    window.$QMessage?.error?.('重置会话设置失败')
  } finally {
    resettingSessions.value = false
  }
}

function updateTrigger<K extends keyof AvatarTriggerRules>(key: K, value: AvatarTriggerRules[K]) {
  emit('update:modelValue', {
    ...props.modelValue,
    triggerRules: {
      ...props.modelValue.triggerRules,
      [key]: value
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

.status-tag {
  display: inline-flex;
  align-items: center;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: var(--font-size-xs);
  font-weight: 500;
}

.status-tag.pending {
  background: rgba(59, 130, 246, 0.1);
  color: #3B82F6;
}

.btn-apply {
  padding: 4px 16px;
  border-radius: 6px;
  font-size: var(--font-size-xs);
  font-weight: 500;
  cursor: pointer;
  border: none;
  background: var(--primary-color, #3B82F6);
  color: #fff;
  transition: opacity 0.2s;
}

.btn-apply:hover:not(:disabled) {
  opacity: 0.85;
}

.btn-apply:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.setting-divider {
  height: 1px;
  background: var(--border-color);
  margin: 16px 0;
}

.btn-reset-sessions {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: color 0.2s, border-color 0.2s, background-color 0.2s;
}

.btn-reset-sessions:hover:not(:disabled) {
  color: #d33;
  border-color: #d33;
  background: rgba(221, 51, 51, 0.06);
}

.btn-reset-sessions:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.setting-item {
  margin-bottom: 16px;
}

.setting-item > label {
  display: block;
  margin-bottom: 6px;
  font-size: var(--font-size-sm);
  font-weight: 500;
}

.setting-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.setting-label {
  font-size: var(--font-size-sm);
  font-weight: 500;
}

.setting-hint {
  display: block;
  margin-top: 4px;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}

.setting-note {
  padding: 6px 10px;
  background: var(--primary-color-alpha, rgba(99, 102, 241, 0.06));
  border: 1px solid var(--border-color);
  border-radius: 6px;
}

.form-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-color);
  font-size: var(--font-size-sm);
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
  font-size: var(--font-size-sm);
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
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.trigger-value {
  font-size: var(--font-size-sm);
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
  font-size: var(--font-size-xs);
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
  font-size: var(--font-size-xxs);
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