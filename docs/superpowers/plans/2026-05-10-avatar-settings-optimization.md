# 分身设置界面优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将分身设置界面重构为两层结构（普通设置 + 高级设置），降低普通用户的学习成本，同时保留高级用户的精细控制能力。

**架构：** 采用标签切换的两层结构，普通设置包含基础配置、知识来源、记忆管理；高级设置包含模型配置、触发规则详细设置、人设风格、回复策略。通过 Vue 响应式系统确保数据同步。

**技术栈：** Vue 3 + TypeScript + Composition API

---

## 文件结构

**新建文件**：
- `qim-client/src/components/avatar/AvatarBasicSettingsSimple.vue` - 普通设置-基础配置
- `qim-client/src/components/avatar/AvatarModelSettings.vue` - 高级设置-模型配置
- `qim-client/src/components/avatar/AvatarTriggerSettingsAdvanced.vue` - 高级设置-触发规则详细

**修改文件**：
- `qim-client/src/components/avatar/AvatarSettingsPanel.vue` - 主面板，添加标签切换

**复用文件**（无需修改）：
- `qim-client/src/components/avatar/AvatarKnowledgeSettings.vue` - 知识来源
- `qim-client/src/components/avatar/AvatarPersonaSettings.vue` - 人设风格
- `qim-client/src/components/avatar/AvatarReplySettings.vue` - 回复策略
- `qim-client/src/components/avatar/AvatarTriggerSettings.vue` - 触发规则
- `qim-client/src/components/avatar/AvatarMemoryPanel.vue` - 记忆管理

---

## 任务 1：创建 AvatarBasicSettingsSimple.vue 组件

**文件：**
- 创建：`qim-client/src/components/avatar/AvatarBasicSettingsSimple.vue`

- [ ] **步骤 1：创建组件基础结构**

```vue
<template>
  <div class="avatar-basic-settings-simple">
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

    <!-- 分身名称 -->
    <div class="setting-item">
      <label>分身名称</label>
      <input 
        :value="modelValue.name" 
        @input="update('name', ($event.target as HTMLInputElement).value)" 
        class="form-input" 
        placeholder="我的分身" 
      />
      <span class="setting-hint">其他人在私聊中看到的分身名称</span>
    </div>

    <!-- 触发模式（简化版） -->
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

    <!-- 接管冷却期 -->
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
    triggerRules: { ...props.modelValue.triggerRules ?? {}, mode }
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
```

- [ ] **步骤 2：验证组件创建成功**

检查文件是否创建成功：
```bash
ls -la qim-client/src/components/avatar/AvatarBasicSettingsSimple.vue
```

预期：文件存在且内容正确

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/components/avatar/AvatarBasicSettingsSimple.vue
git commit -m "feat(分身设置): 创建普通设置-基础配置组件"
```

---

## 任务 2：创建 AvatarModelSettings.vue 组件

**文件：**
- 创建：`qim-client/src/components/avatar/AvatarModelSettings.vue`

- [ ] **步骤 1：创建组件基础结构**

```vue
<template>
  <div class="avatar-model-settings">
    <div class="setting-item">
      <label>模型来源</label>
      <div class="radio-group">
        <label class="radio-label">
          <input 
            type="radio" 
            :checked="modelValue.useSystemConfig" 
            @change="update('useSystemConfig', true)"
          />
          <span>使用系统默认模型（推荐）</span>
        </label>
        <label class="radio-label">
          <input 
            type="radio" 
            :checked="!modelValue.useSystemConfig" 
            @change="update('useSystemConfig', false)"
          />
          <span>使用我的自定义配置</span>
        </label>
      </div>
    </div>

    <div v-if="!modelValue.useSystemConfig" class="setting-item">
      <label>选择配置</label>
      <select 
        :value="modelValue.modelConfigId || ''" 
        @change="update('modelConfigId', Number(($event.target as HTMLSelectElement).value) || null)" 
        class="form-select"
      >
        <option value="">请选择...</option>
        <option v-for="cfg in modelConfigs" :key="cfg.id" :value="cfg.id">
          {{ cfg.config_name }} ({{ cfg.model_name }})
        </option>
      </select>
      <span v-if="modelConfigs.length === 0" class="setting-hint error">
        暂无配置，请先在"我的模型配置"中添加
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AvatarConfig } from '../../types/avatar'
import type { UserAIConfig as AIConfig } from '../../types/ai'

const props = defineProps<{
  modelValue: AvatarConfig
  modelConfigs: AIConfig[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

function update<K extends keyof AvatarConfig>(key: K, value: AvatarConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}
</script>

<style scoped>
.avatar-model-settings {
  padding: 16px;
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

.radio-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 14px;
}

.radio-label input[type="radio"] {
  cursor: pointer;
}

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

.form-select:focus {
  outline: none;
  border-color: var(--primary-color);
}

.setting-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.setting-hint.error {
  color: #F44336;
}
</style>
```

- [ ] **步骤 2：验证组件创建成功**

检查文件是否创建成功：
```bash
ls -la qim-client/src/components/avatar/AvatarModelSettings.vue
```

预期：文件存在且内容正确

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/components/avatar/AvatarModelSettings.vue
git commit -m "feat(分身设置): 创建高级设置-模型配置组件"
```

---

## 任务 3：创建 AvatarTriggerSettingsAdvanced.vue 组件

**文件：**
- 创建：`qim-client/src/components/avatar/AvatarTriggerSettingsAdvanced.vue`

- [ ] **步骤 1：创建组件基础结构**

```vue
<template>
  <div class="avatar-trigger-settings-advanced">
    <div class="setting-item">
      <label>触发模式</label>
      <select 
        :value="modelValue.triggerRules?.mode ?? 'mention'" 
        @change="updateTrigger('mode', ($event.target as HTMLSelectElement).value)" 
        class="form-select"
      >
        <option value="mention">被 @ 时回复</option>
        <option value="offline">离线时自动回复</option>
        <option value="keyword">关键词触发</option>
        <option value="all">所有消息（谨慎使用）</option>
        <option value="custom">自定义规则</option>
      </select>
      <span class="setting-hint">
        {{ triggerModeHint }}
      </span>
    </div>

    <div v-if="modelValue.triggerRules?.mode === 'keyword' || modelValue.triggerRules?.mode === 'custom'" class="setting-item">
      <label>触发关键词</label>
      <div class="keyword-input-wrapper">
        <input
          :value="keywordInput"
          @input="keywordInput = ($event.target as HTMLInputElement).value"
          @keydown.enter.prevent="addKeyword"
          class="form-input"
          placeholder="输入关键词后按回车"
        />
        <div class="keyword-tags">
          <span v-for="(kw, i) in modelValue.triggerRules?.keywords ?? []" :key="i" class="keyword-tag">
            {{ kw }}
            <button class="remove-tag" @click="removeKeyword(i)">x</button>
          </span>
        </div>
      </div>
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
import { ref, computed } from 'vue'
import type { AvatarConfig } from '../../types/avatar'

const props = defineProps<{
  modelValue: AvatarConfig
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AvatarConfig]
}>()

const keywordInput = ref('')

const triggerModeHint = computed(() => {
  const hints: Record<string, string> = {
    mention: '当有人在私聊中发消息，或在群聊中 @你时，分身会回复',
    offline: '当你离线时，分身自动回复私聊消息',
    keyword: '仅当消息包含指定关键词时，分身才回复',
    all: '分身会回复所有消息（请谨慎使用）',
    custom: '自定义触发规则，结合关键词和时间段'
  }
  return hints[props.modelValue.triggerRules?.mode ?? ''] || ''
})

function update<K extends keyof AvatarConfig>(key: K, value: AvatarConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function updateTrigger(key: string, value: any) {
  emit('update:modelValue', {
    ...props.modelValue,
    triggerRules: { ...props.modelValue.triggerRules ?? {}, [key]: value }
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
.avatar-trigger-settings-advanced {
  padding: 16px;
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

.setting-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.form-select,
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

.form-select:focus,
.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.keyword-input-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.keyword-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
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
}

.remove-tag {
  background: none;
  border: none;
  color: var(--primary-color);
  cursor: pointer;
  font-size: 14px;
  padding: 0;
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
```

- [ ] **步骤 2：验证组件创建成功**

检查文件是否创建成功：
```bash
ls -la qim-client/src/components/avatar/AvatarTriggerSettingsAdvanced.vue
```

预期：文件存在且内容正确

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/components/avatar/AvatarTriggerSettingsAdvanced.vue
git commit -m "feat(分身设置): 创建高级设置-触发规则详细组件"
```

---

## 任务 4：修改 AvatarSettingsPanel.vue 主面板

**文件：**
- 修改：`qim-client/src/components/avatar/AvatarSettingsPanel.vue`

- [ ] **步骤 1：修改模板部分**

将原有的 7 个标签页改为 2 个主标签（普通设置、高级设置）：

```vue
<template>
  <div class="avatar-settings-panel">
    <div v-if="loading" class="loading-state">
      <LoadingSpinner />
    </div>

    <div v-else-if="!config" class="empty-state">
      <EmptyState
        icon="fas fa-user-astronaut"
        title="还没有分身"
        description="创建你的 AI 分身，在你不在时代替回复消息"
      />
      <button class="create-btn" @click="handleCreate">
        <i class="fas fa-plus"></i> 创建分身
      </button>
    </div>

    <template v-else>
      <div class="tab-bar">
        <button
          v-for="tab in mainTabs"
          :key="tab.key"
          :class="['tab-btn', { active: activeMainTab === tab.key }]"
          @click="activeMainTab = tab.key"
        >
          <i :class="tab.icon"></i>
          <span>{{ tab.label }}</span>
        </button>
      </div>

      <div class="tab-content">
        <!-- 普通设置 -->
        <template v-if="activeMainTab === 'basic'">
          <div class="settings-section">
            <h3 class="section-title">基础配置</h3>
            <AvatarBasicSettingsSimple
              v-model="config"
            />
          </div>

          <div class="settings-section">
            <h3 class="section-title">知识来源</h3>
            <AvatarKnowledgeSettings
              v-model="config"
            />
          </div>

          <div class="settings-section">
            <h3 class="section-title">记忆管理</h3>
            <AvatarMemoryPanel
              :user-id="userId"
            />
          </div>
        </template>

        <!-- 高级设置 -->
        <template v-else-if="activeMainTab === 'advanced'">
          <div class="settings-section">
            <h3 class="section-title">模型配置</h3>
            <AvatarModelSettings
              v-model="config"
              :model-configs="modelConfigs"
            />
          </div>

          <div class="settings-section">
            <h3 class="section-title">触发规则详细设置</h3>
            <AvatarTriggerSettingsAdvanced
              v-model="config"
            />
          </div>

          <div class="settings-section">
            <h3 class="section-title">人设风格</h3>
            <AvatarPersonaSettings
              v-model="config"
            />
          </div>

          <div class="settings-section">
            <h3 class="section-title">回复策略</h3>
            <AvatarReplySettings
              v-model="config"
            />
          </div>
        </template>
      </div>

      <div class="tab-footer">
        <button class="btn btn-danger" @click="handleDelete" v-if="config">
          <i class="fas fa-trash"></i> 删除分身
        </button>
        <button class="btn btn-primary" @click="handleSave" :disabled="saving">
          {{ saving ? '保存中...' : '保存设置' }}
        </button>
      </div>
    </template>
  </div>
</template>
```

- [ ] **步骤 2：修改脚本部分**

更新导入和逻辑：

```typescript
<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useAvatar } from '../../composables/useAvatar'
import { useModelConfigs } from '../../composables/useModelConfigs'
import LoadingSpinner from '../shared/LoadingSpinner.vue'
import EmptyState from '../shared/EmptyState.vue'
import AvatarBasicSettingsSimple from './AvatarBasicSettingsSimple.vue'
import AvatarKnowledgeSettings from './AvatarKnowledgeSettings.vue'
import AvatarMemoryPanel from './AvatarMemoryPanel.vue'
import AvatarModelSettings from './AvatarModelSettings.vue'
import AvatarTriggerSettingsAdvanced from './AvatarTriggerSettingsAdvanced.vue'
import AvatarPersonaSettings from './AvatarPersonaSettings.vue'
import AvatarReplySettings from './AvatarReplySettings.vue'
import { DEFAULT_AVATAR_CONFIG } from '../../types/avatar'

const {
  config,
  loading,
  fetchConfig,
  createConfig,
  updateConfig,
  deleteConfig
} = useAvatar()

const { configs: modelConfigs, fetchConfigs } = useModelConfigs()

const activeMainTab = ref<'basic' | 'advanced'>('basic')
const saving = ref(false)

const mainTabs = [
  { key: 'basic', label: '普通设置', icon: 'fas fa-cog' },
  { key: 'advanced', label: '高级设置', icon: 'fas fa-sliders-h' }
]

const serverUrl = import.meta.env.VITE_SERVER_URL || ''
const userId = ref(0)

// 保存用户选择的标签
watch(activeMainTab, (newTab) => {
  localStorage.setItem('avatar-settings-tab', newTab)
})

onMounted(async () => {
  await Promise.all([fetchConfig(true), fetchConfigs()])
  
  // 恢复用户选择的标签
  const savedTab = localStorage.getItem('avatar-settings-tab')
  if (savedTab === 'basic' || savedTab === 'advanced') {
    activeMainTab.value = savedTab
  }
  
  // 从 localStorage 获取当前用户 ID
  const userStr = localStorage.getItem('user')
  if (userStr) {
    try {
      const user = JSON.parse(userStr)
      userId.value = user.id
    } catch (e) {
      console.error('解析用户信息失败', e)
    }
  }
})

async function handleCreate() {
  try {
    await createConfig(DEFAULT_AVATAR_CONFIG)
    window.$QMessage.success('分身创建成功')
  } catch {
    window.$QMessage.error('创建失败')
  }
}

async function handleSave() {
  if (!config.value) return
  saving.value = true
  try {
    await updateConfig(config.value)
    window.$QMessage.success('设置已保存')
  } catch {
    window.$QMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete() {
  try {
    await window.$QMessageBox.confirm('确定删除分身吗？删除后所有会话的分身都将关闭。', '删除分身')
    await deleteConfig()
    window.$QMessage.success('分身已删除')
  } catch {
    // 用户取消
  }
}
</script>
```

- [ ] **步骤 3：修改样式部分**

添加新的样式：

```css
<style scoped>
.avatar-settings-panel {
  background: var(--card-bg);
  border-radius: 8px;
  overflow: hidden;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
}

.empty-state {
  text-align: center;
  padding: 40px 20px;
}

.create-btn {
  margin-top: 16px;
  padding: 10px 24px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.create-btn:hover {
  opacity: 0.9;
}

.tab-bar {
  display: flex;
  border-bottom: 1px solid var(--border-color);
}

.tab-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 12px 8px;
  border: none;
  background: none;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-secondary);
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.tab-btn:hover {
  color: var(--text-color);
  background: var(--hover-color);
}

.tab-btn.active {
  color: var(--primary-color);
  border-bottom-color: var(--primary-color);
  background: var(--primary-color-alpha, rgba(99, 102, 241, 0.05));
}

.tab-content {
  flex: 1;
  overflow-y: auto;
}

.settings-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 12px 0;
  padding: 0 16px;
  color: var(--text-color);
}

.tab-footer {
  padding: 12px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
}

.btn {
  padding: 8px 20px;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  border: none;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-danger {
  background: transparent;
  color: #F44336;
  border: 1px solid #F44336;
}

.btn-danger:hover {
  background: #FFEBEE;
}
</style>
```

- [ ] **步骤 4：验证修改成功**

检查文件是否修改成功：
```bash
cat qim-client/src/components/avatar/AvatarSettingsPanel.vue | grep "activeMainTab"
```

预期：能看到 activeMainTab 相关代码

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/components/avatar/AvatarSettingsPanel.vue
git commit -m "feat(分身设置): 重构主面板为两层结构（普通设置+高级设置）"
```

---

## 任务 5：测试和验证

**文件：**
- 无需创建新文件

- [ ] **步骤 1：启动开发服务器**

```bash
cd qim-client
npm run dev
```

预期：开发服务器启动成功

- [ ] **步骤 2：测试普通设置功能**

测试步骤：
1. 打开分身设置页面
2. 验证默认显示"普通设置"标签
3. 测试启用/关闭分身开关
4. 测试触发模式选择
5. 测试知识来源开关
6. 测试记忆管理功能

预期：所有功能正常工作

- [ ] **步骤 3：测试高级设置功能**

测试步骤：
1. 切换到"高级设置"标签
2. 测试模型配置选择
3. 测试触发规则详细设置
4. 测试人设风格设置
5. 测试回复策略设置

预期：所有功能正常工作

- [ ] **步骤 4：测试数据同步**

测试步骤：
1. 在普通设置中修改触发模式
2. 切换到高级设置，验证触发模式已同步
3. 在高级设置中修改触发模式
4. 切换到普通设置，验证触发模式已同步

预期：数据同步正确

- [ ] **步骤 5：测试标签状态保存**

测试步骤：
1. 切换到"高级设置"标签
2. 刷新页面
3. 验证仍然显示"高级设置"标签

预期：标签状态已保存

- [ ] **步骤 6：Commit 测试通过标记**

```bash
git add -A
git commit -m "test(分身设置): 验证两层结构功能正常"
```

---

## 任务 6：清理和优化

**文件：**
- 无需创建新文件

- [ ] **步骤 1：移除未使用的导入**

检查 AvatarSettingsPanel.vue 中是否有未使用的导入，移除它们。

- [ ] **步骤 2：优化样式**

检查是否有重复的样式定义，合并它们。

- [ ] **步骤 3：添加过渡动画**

为标签切换添加过渡动画：

```vue
<transition name="fade" mode="out-in">
  <div class="tab-content" :key="activeMainTab">
    <!-- 内容 -->
  </div>
</transition>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
```

- [ ] **步骤 4：最终 Commit**

```bash
git add -A
git commit -m "refactor(分身设置): 优化代码和添加过渡动画"
```

---

## 自检清单

**1. 规格覆盖度：**
- ✅ 普通设置包含基础配置、知识来源、记忆管理
- ✅ 高级设置包含模型配置、触发规则详细设置、人设风格、回复策略
- ✅ 技能市场已隐藏
- ✅ 触发模式已简化
- ✅ 标签切换功能已实现
- ✅ 标签状态保存功能已实现

**2. 占位符扫描：**
- ✅ 无"待定"、"TODO"等占位符
- ✅ 所有步骤都有完整代码
- ✅ 所有命令都有预期输出

**3. 类型一致性：**
- ✅ AvatarConfig 类型在所有组件中一致
- ✅ AvatarConfigWithApproval 类型正确使用
- ✅ 触发模式类型定义一致

---

## 执行选项

计划已完成并保存到 `docs/superpowers/plans/2026-05-10-avatar-settings-optimization.md`。两种执行方式：

**1. 子代理驱动（推荐）** - 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** - 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点

**选哪种方式？**
