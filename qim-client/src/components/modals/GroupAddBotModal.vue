<template>
  <div v-if="visible" class="add-bot-modal" @click.self="handleClose">
    <div class="add-bot-content">
      <div class="add-bot-header">
        <h3>添加外部机器人</h3>
        <button class="close-btn" @click="handleClose">×</button>
      </div>
      <div class="add-bot-body">
        <p class="add-bot-tip">
          把外部 agent 机器人拉进群后，群成员可直接 <b>@机器人</b> 触发其回复，机器人也能在群里主动发言。
        </p>

        <div v-if="loading" class="add-bot-empty">加载中...</div>

        <div v-else-if="bots.length === 0" class="add-bot-empty">
          暂无可添加的外部机器人。
          <span class="add-bot-empty-sub">仅显示已配置 webhook 且可 @ 触发的外部 agent 机器人。</span>
        </div>

        <div v-else class="bot-list">
          <button
            v-for="bot in bots"
            :key="bot.id"
            class="bot-item"
            :disabled="bot.inGroup"
            @click="selectBot(bot)"
          >
            <div class="bot-avatar">
              <i class="fas fa-robot"></i>
            </div>
            <div class="bot-info">
              <span class="bot-name">{{ bot.name }}</span>
              <span class="bot-desc">{{ bot.description || '外部 agent 机器人' }}</span>
            </div>
            <span v-if="bot.inGroup" class="bot-state">已在群</span>
            <i v-else class="fas fa-chevron-right bot-arrow"></i>
          </button>
        </div>
      </div>
      <div class="add-bot-footer">
        <button class="btn-cancel" @click="handleClose">取消</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

export interface ExternalBotCandidate {
  id: number
  name: string
  description?: string
  /** 该 bot 是否已在当前群里（初次进入时由父组件预标记） */
  inGroup?: boolean
}

const props = defineProps<{
  visible: boolean
  loading?: boolean
  bots: ExternalBotCandidate[]
}>()

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  select: [bot: ExternalBotCandidate]
}>()

const loading = ref(props.loading ?? false)

watch(
  () => props.loading,
  (val) => { loading.value = val ?? false }
)

const handleClose = () => {
  emit('update:visible', false)
}

const selectBot = (bot: ExternalBotCandidate) => {
  if (bot.inGroup) return
  emit('select', bot)
}
</script>

<style scoped>
.add-bot-modal {
  position: fixed;
  inset: 0;
  z-index: 1100;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 12vh;
}

.add-bot-content {
  width: 380px;
  max-width: calc(100vw - 32px);
  background: var(--panel-bg, #fff);
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.25);
}

.add-bot-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border-color, #eee);
}

.add-bot-header h3 {
  margin: 0;
  font-size: 15px;
}

.close-btn {
  border: none;
  background: transparent;
  font-size: 20px;
  line-height: 1;
  cursor: pointer;
  color: var(--text-secondary, #888);
}

.add-bot-body {
  padding: 14px 16px;
  max-height: 48vh;
  overflow-y: auto;
}

.add-bot-tip {
  margin: 0 0 12px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-secondary, #666);
}

.add-bot-tip b {
  color: var(--primary-color, #2f6fed);
}

.bot-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.bot-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color, #eee);
  border-radius: 8px;
  background: var(--card-bg, #fafafa);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s, background 0.15s;
}

.bot-item:hover:not(:disabled) {
  border-color: var(--primary-color, #2f6fed);
  background: var(--primary-bg, rgba(47, 111, 237, 0.06));
}

.bot-item:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.bot-avatar {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--primary-bg, rgba(47, 111, 237, 0.12));
  color: var(--primary-color, #2f6fed);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.bot-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.bot-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary, #333);
}

.bot-desc {
  font-size: 12px;
  color: var(--text-secondary, #999);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.bot-state {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-secondary, #999);
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: 10px;
  padding: 1px 8px;
}

.bot-arrow {
  color: var(--text-secondary, #bbb);
  font-size: 13px;
}

.add-bot-empty {
  padding: 28px 12px;
  text-align: center;
  font-size: 13px;
  color: var(--text-secondary, #888);
}

.add-bot-empty-sub {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-tertiary, #aaa);
}

.add-bot-footer {
  padding: 12px 16px;
  border-top: 1px solid var(--border-color, #eee);
  display: flex;
  justify-content: flex-end;
}

.btn-cancel {
  border: 1px solid var(--border-color, #ddd);
  background: transparent;
  color: var(--text-secondary, #666);
  padding: 6px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}

.btn-cancel:hover {
  background: var(--border-color, #f0f0f0);
}
</style>
