<template>
  <div class="chat-header">
    <div class="header-info">
      <div class="avatar-wrapper">
        <Avatar
          :src="conversation?.avatar"
          :name="displayName"
          :server-url="serverUrl"
          :alt="displayName"
          size="lg"
          :badge="headerBadge"
        />
      </div>
      <div class="header-text">
        <div class="header-name">{{ displayName }}</div>
        <div class="header-status">
          <template v-if="isSingleChat">
            <span v-if="conversation?.ip" class="ip-info">
              {{ conversation.ip }}
            </span>
            <span v-if="conversation?.signature" class="signature-info">
              {{ conversation.signature }}
            </span>
          </template>
          <span v-if="isGroupOrDiscussion && conversation?.announcement" class="header-announcement-inline">
            <i class="fas fa-volume-high"></i>
            {{ conversation.announcement }}
          </span>
        </div>
      </div>
    </div>

    <ChatHeaderActions
      :conversation="conversation"
      :current-user="currentUser"
      :server-url="serverUrl"
      :ai-enabled="aiEnabled"
      :ai-assistant-name="aiAssistantName"
      :ai-reply-mode="aiReplyMode"
      :ai-personality="aiPersonality"
      :ai-custom-prompt="aiCustomPrompt"
      :ai-language="aiLanguage"
      :ai-max-length="aiMaxLength"
      :ai-mention-reply-mode="aiMentionReplyMode"
      :ai-anti-spam-interval="aiAntiSpamInterval"
      :ai-trigger-keywords="aiTriggerKeywords"
      :ai-learn-enabled="aiLearnEnabled"
      :avatar-enabled="avatarEnabled"
      :avatar-approval-status="avatarApprovalStatus"
      @invite-members="emit('invite-members')"
      @delete-group="emit('delete-group')"
      @switch-conversation="(id: string) => emit('switch-conversation', id)"
      @show-user-profile="(user: any) => emit('show-user-profile', user)"
      @remove-member="(id: string, name: string) => emit('remove-member', id, name)"
      @set-admin="(id: string, name: string, isAdmin: boolean) => emit('set-admin', id, name, isAdmin)"
      @transfer-owner="(id: string, name: string) => emit('transfer-owner', id, name)"
      @start-private-chat="(id: string) => emit('start-private-chat', id)"
      @update-ai-settings="(settings) => emit('update-ai-settings', settings)"
      @update-avatar-enabled="(value) => emit('update-avatar-enabled', value)"
      @update-avatar-takeover="() => emit('update-avatar-takeover')"
      @open-group-files="emit('open-group-files')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Conversation } from '../../types'
import Avatar from '../shared/Avatar.vue'
import ChatHeaderActions from './ChatHeaderActions.vue'
import { buildConversationBadge } from '../../utils/user'
import { ref } from 'vue'

interface Props {
  conversation: Conversation
  currentUser: any
  serverUrl: string
  avatarEnabled?: boolean
  avatarApprovalStatus?: string
}

interface Emits {
  (e: 'invite-members'): void
  (e: 'delete-group'): void
  (e: 'switch-conversation', id: string): void
  (e: 'show-user-profile', user: any): void
  (e: 'remove-member', id: string, name: string): void
  (e: 'set-admin', id: string, name: string, isAdmin: boolean): void
  (e: 'transfer-owner', id: string, name: string): void
  (e: 'start-private-chat', id: string): void
  (e: 'update-ai-settings', settings: { aiEnabled: boolean; aiAssistantName: string; aiReplyMode: string; aiPersonality: string; aiCustomPrompt: string; aiLanguage: string; aiMaxLength: string; aiMentionReplyMode: string; aiAntiSpamInterval: number; aiTriggerKeywords: string[]; aiLearnEnabled: boolean }): void
  (e: 'update-avatar-enabled', value: boolean): void
  (e: 'update-avatar-takeover'): void
  (e: 'open-group-files'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const aiEnabled = computed(() => props.conversation?.ai_config?.ai_enabled ?? false)
const aiAssistantName = computed(() => props.conversation?.ai_config?.ai_assistant_name ?? 'AI助手')
const aiReplyMode = computed(() => props.conversation?.ai_config?.ai_reply_mode ?? 'mention_only')
const aiPersonality = computed(() => props.conversation?.ai_config?.ai_personality ?? 'professional')
const aiCustomPrompt = computed(() => props.conversation?.ai_config?.ai_custom_prompt ?? '')
const aiLanguage = computed(() => props.conversation?.ai_config?.ai_language ?? 'auto')
const aiMaxLength = computed(() => props.conversation?.ai_config?.ai_max_length ?? 'medium')
const aiMentionReplyMode = computed(() => props.conversation?.ai_config?.ai_mention_reply_mode ?? 'mention')
const aiAntiSpamInterval = computed(() => props.conversation?.ai_config?.ai_anti_spam_interval ?? 0)
const aiTriggerKeywords = computed(() => {
  const kw = props.conversation?.ai_config?.ai_trigger_keywords ?? ''
  return kw ? kw.split(',').filter(Boolean) : []
})
const aiLearnEnabled = computed(() => props.conversation?.ai_config?.ai_learn_enabled ?? false)
const approvalStatus = computed(() => props.conversation?.approval_status ?? 'approved')
const rejectReason = computed(() => props.conversation?.reject_reason ?? '')
const contextMessages = computed(() => props.conversation?.context_messages ?? 10)

const isGroupOrDiscussion = computed(() =>
  props.conversation?.type === 'group' || props.conversation?.type === 'discussion'
)

const isSingleChat = computed(() => props.conversation?.type === 'single')

// 聊天头部头像角标：统一由 buildConversationBadge 构造，Avatar 只负责渲染。
// ChatHeader 持有 currentUser，传入以支持 single 会话 partner 推导时排除自己。
const headerBadge = computed(() =>
  buildConversationBadge(props.conversation, props.currentUser?.id)
)

const displayName = computed(() => props.conversation?.name || '未知会话')

defineExpose({
  aiEnabled,
  aiAssistantName,
  aiReplyMode,
  aiPersonality,
  aiCustomPrompt,
  aiLanguage,
  aiMaxLength,
  aiMentionReplyMode,
  aiAntiSpamInterval,
  aiTriggerKeywords,
  aiLearnEnabled,
  contextMessages,
  approvalStatus,
  rejectReason
})
</script>

<style scoped>
.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  height: 56px;
  background: var(--sidebar-bg);
  box-sizing: border-box;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  object-fit: cover;
}

.avatar-wrapper {
  position: relative;
  flex-shrink: 0;
}

.header-text {
  display: flex;
  flex-direction: column;
}

.header-name {
  font-weight: 500;
  font-size: 16px;
  color: var(--text-color);
}

.header-status {
  font-size: 12px;
  color: var(--color-success-500);
  display: flex;
  align-items: center;
  gap: 8px;
}

.ip-info {
  color: var(--text-color);
  opacity: 0.7;
  font-size: 11px;
  /* padding: 2px 0px; */
  /* background: var(--hover-color); */
  border-radius: 3px;
}

.online-status {
  font-size: 12px;
  padding: 1px 6px;
  border-radius: 3px;
  margin-right: 8px;
}

.online-status.online {
  color: var(--color-success-500);
  background: rgba(82, 196, 26, 0.1);
}

.online-status.offline {
  color: var(--color-gray-500);
  background: rgba(153, 153, 153, 0.1);
}

.signature-info {
  color: var(--text-color);
  opacity: 0.6;
  font-size: 12px;
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-announcement-inline {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--input-bg);
  border-radius: 4px;
  font-size: 12px;
  color: var(--text-secondary);
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-announcement-inline i {
  font-size: 11px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.avatar-toggle-wrapper {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: var(--hover-color);
  border-radius: 20px;
  transition: all 0.2s ease;
}

.avatar-toggle-wrapper:hover {
  background: var(--input-bg);
}

.avatar-toggle-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  white-space: nowrap;
}
</style>
