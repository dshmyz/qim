<template>
  <div class="group-ai-panel">
    <h4>AI 助手设置</h4>

    <div class="setting-item">
      <label class="toggle-label">
        <span>启用 AI 助手</span>
        <label class="switch">
          <input type="checkbox" v-model="localSettings.enabled" @change="saveSettings" />
          <span class="slider round"></span>
        </label>
      </label>
    </div>

    <div v-if="localSettings.enabled" class="advanced-settings">
      <div class="setting-item">
        <label>AI 助手名称</label>
        <input
          type="text"
          v-model="localSettings.assistantName"
          @blur="saveSettings"
          class="form-input"
          placeholder="请输入 AI 助手名称"
        />
      </div>

      <div class="setting-item">
        <label>回复模式</label>
        <select v-model="localSettings.replyMode" @change="saveSettings" class="form-select">
          <option value="mention_only">仅被 @ 时回复</option>
          <option value="smart">智能判断回复</option>
          <option value="always">始终回复</option>
          <option value="off">关闭 AI 回复</option>
        </select>
      </div>

      <div class="setting-item">
        <label>上下文消息数</label>
        <div class="number-input-wrapper">
          <input
            type="number"
            v-model.number="localSettings.contextMessages"
            @blur="saveSettings"
            class="form-input"
            min="5"
            max="50"
          />
          <span class="setting-hint">AI 回复时参考的最近消息数</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface GroupAISettings {
  enabled: boolean
  assistantName: string
  replyMode: string
  contextMessages: number
}

interface Props {
  groupId: number
  aiEnabled: boolean
  aiAssistantName?: string
  aiReplyMode?: string
  contextMessages?: number
}

interface Emits {
  (e: 'update', settings: GroupAISettings): void
}

const props = withDefaults(defineProps<Props>(), {
  aiAssistantName: 'AI助手',
  aiReplyMode: 'mention_only',
  contextMessages: 10
})

const emit = defineEmits<Emits>()

const localSettings = ref<GroupAISettings>({
  enabled: props.aiEnabled,
  assistantName: props.aiAssistantName,
  replyMode: props.aiReplyMode,
  contextMessages: props.contextMessages
})

// 监听 props 变化，同步更新本地状态
watch(
  () => [props.aiEnabled, props.aiAssistantName, props.aiReplyMode, props.contextMessages],
  ([enabled, name, mode, count]) => {
    localSettings.value = {
      enabled: enabled as boolean,
      assistantName: name as string,
      replyMode: mode as string,
      contextMessages: count as number
    }
  }
)

const saveSettings = () => {
  emit('update', { ...localSettings.value })
}
</script>

<style scoped>
.group-ai-panel {
  padding: 16px;
  background: var(--card-bg);
  border-radius: 8px;
}

.group-ai-panel h4 {
  margin: 0 0 16px 0;
  font-size: 16px;
  color: var(--text-color);
  font-weight: 600;
}

.setting-item {
  margin-bottom: 16px;
}

.setting-item label {
  display: block;
  margin-bottom: 6px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.toggle-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
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
  transition: border-color 0.2s;
}

.form-input:focus,
.form-select:focus {
  outline: none;
  border-color: var(--primary-color);
}

.form-select {
  cursor: pointer;
}

.setting-hint {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-secondary);
}

.number-input-wrapper {
  width: 100%;
}

/* 开关切换按钮样式 */
.switch {
  position: relative;
  display: inline-block;
  width: 50px;
  height: 24px;
  min-width: 50px;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #ccc;
  transition: 0.4s;
  border-radius: 24px;
}

.slider:before {
  position: absolute;
  content: '';
  height: 18px;
  width: 18px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: 0.4s;
  border-radius: 50%;
}

input:checked + .slider {
  background-color: var(--primary-color);
}

input:checked + .slider:before {
  transform: translateX(26px);
}

.slider.round {
  border-radius: 24px;
}

.advanced-settings {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
}
</style>
