<template>
  <div class="chat-header">
    <div class="header-info">
      <div class="avatar-wrapper">
        <img :src="avatarUrl" :alt="displayName" class="header-avatar" />
        <span
          v-if="isSingleChat"
          :class="['status-dot', conversation?.status === 'online' ? 'online' : 'offline']"
          :title="conversation?.status === 'online' ? '在线' : '离线'"
        ></span>
      </div>
      <div class="header-text">
        <div class="header-name">{{ displayName }}</div>
        <div class="header-status">
          <template v-if="isGroupOrDiscussion">
            {{ conversation?.type === 'group' ? '群聊' : '讨论组' }}
            <span v-if="memberCount" class="member-count">
              ({{ memberCount }}人)
            </span>
          </template>
          <template v-else-if="isSingleChat">
            <span v-if="conversation?.ip" class="ip-info">
              {{ conversation.ip }}
            </span>
            <span v-if="conversation?.signature" class="signature-info">
              {{ conversation.signature }}
            </span>
          </template>
          <template v-else>
            在线
          </template>
          <span v-if="isGroupOrDiscussion && conversation?.announcement" class="header-announcement-inline">
            <i class="fas fa-bullhorn"></i>
            {{ conversation.announcement }}
          </span>
        </div>
      </div>
    </div>

    <GroupPanel
      :conversation="conversation"
      :current-user="currentUser"
      :server-url="serverUrl"
      v-model:showHeaderMenu="showHeaderMenu"
      v-model:showEditGroupInfoModal="showEditGroupInfoModal"
      v-model:showEditAnnouncementModal="showEditAnnouncementModal"
      v-model:editGroupName="editGroupName"
      v-model:editAnnouncement="editAnnouncement"
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
      @invite-members="emit('invite-members')"
      @delete-group="emit('delete-group')"
      @save-group-info="(name: string) => emit('save-group-info', name)"
      @save-group-announcement="(announcement: string) => emit('save-group-announcement', announcement)"
      @switch-conversation="(id: string) => emit('switch-conversation', id)"
      @show-user-profile="(user: any) => emit('show-user-profile', user)"
      @remove-member="(id: string, name: string) => emit('remove-member', id, name)"
      @set-admin="(id: string, name: string, isAdmin: boolean) => emit('set-admin', id, name, isAdmin)"
      @transfer-owner="(id: string, name: string) => emit('transfer-owner', id, name)"
      @start-private-chat="(id: string) => emit('start-private-chat', id)"
      @update-ai-settings="(settings) => emit('update-ai-settings', settings)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Conversation } from '../../types'
import GroupPanel from './GroupPanel.vue'
import { getAvatarUrl } from '../../utils/avatar'
import { ref } from 'vue'

interface Props {
  conversation: Conversation
  currentUser: any
  serverUrl: string
}

interface Emits {
  (e: 'invite-members'): void
  (e: 'delete-group'): void
  (e: 'save-group-info', name: string): void
  (e: 'save-group-announcement', announcement: string): void
  (e: 'switch-conversation', id: string): void
  (e: 'show-user-profile', user: any): void
  (e: 'remove-member', id: string, name: string): void
  (e: 'set-admin', id: string, name: string, isAdmin: boolean): void
  (e: 'transfer-owner', id: string, name: string): void
  (e: 'start-private-chat', id: string): void
  (e: 'update:showHeaderMenu', value: boolean): void
  (e: 'update:showEditGroupInfoModal', value: boolean): void
  (e: 'update:showEditAnnouncementModal', value: boolean): void
  (e: 'update:editGroupName', value: string): void
  (e: 'update:editAnnouncement', value: string): void
  (e: 'update-ai-settings', settings: { aiEnabled: boolean; aiAssistantName: string; aiReplyMode: string; aiPersonality: string; aiCustomPrompt: string; aiLanguage: string; aiMaxLength: string; aiMentionReplyMode: string; aiAntiSpamInterval: number; aiTriggerKeywords: string[]; aiLearnEnabled: boolean }): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const showHeaderMenu = ref(false)
const showEditGroupInfoModal = ref(false)
const showEditAnnouncementModal = ref(false)
const editGroupName = ref('')
const editAnnouncement = ref('')

const aiEnabled = computed(() => props.conversation?.ai_enabled ?? false)
const aiAssistantName = computed(() => props.conversation?.ai_assistant_name ?? 'AI助手')
const aiReplyMode = computed(() => props.conversation?.ai_reply_mode ?? 'mention_only')
const aiPersonality = computed(() => props.conversation?.ai_personality ?? 'professional')
const aiCustomPrompt = computed(() => props.conversation?.ai_custom_prompt ?? '')
const aiLanguage = computed(() => props.conversation?.ai_language ?? 'auto')
const aiMaxLength = computed(() => props.conversation?.ai_max_length ?? 'medium')
const aiMentionReplyMode = computed(() => props.conversation?.ai_mention_reply_mode ?? 'mention')
const aiAntiSpamInterval = computed(() => props.conversation?.ai_anti_spam_interval ?? 5)
const aiTriggerKeywords = computed(() => {
  const kw = props.conversation?.ai_trigger_keywords ?? ''
  return kw ? kw.split(',').filter(Boolean) : []
})
const aiLearnEnabled = computed(() => props.conversation?.ai_learn_enabled ?? false)
const contextMessages = computed(() => props.conversation?.context_messages ?? 10)

const isGroupOrDiscussion = computed(() =>
  props.conversation?.type === 'group' || props.conversation?.type === 'discussion'
)

const isSingleChat = computed(() => props.conversation?.type === 'single')

const memberCount = computed(() =>
  props.conversation?.members?.length ?? 0
)

const displayName = computed(() => props.conversation?.name || '未知会话')

const avatarUrl = computed(() =>
  getAvatarUrl(
    props.conversation?.avatar,
    props.conversation?.name || '用户',
    props.serverUrl
  )
)

defineExpose({
  showHeaderMenu,
  showEditGroupInfoModal,
  showEditAnnouncementModal,
  editGroupName,
  editAnnouncement,
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
  contextMessages
})
</script>

<style scoped>
.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--sidebar-bg);
  height: 72px;
  box-sizing: border-box;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  margin: 0;
  margin-bottom: 1px;
  border-radius: 0;
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

.status-dot {
  position: absolute;
  bottom: 4px;
  right: 1px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 2px solid var(--sidebar-bg);
  flex-shrink: 0;
}

.status-dot.online {
  background: var(--color-success-500);
}

.status-dot.offline {
  background: var(--color-gray-500);
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
  margin-left: 8px;
  padding: 2px 6px;
  background: var(--hover-color);
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
  margin-left: 8px;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-announcement-inline {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: 8px;
  padding: 2px 8px;
  background: var(--input-bg);
  border-radius: 4px;
  font-size: 12px;
  color: var(--text-secondary);
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-announcement-inline i {
  font-size: 11px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.member-count {
  color: var(--primary-color);
  cursor: pointer;
  font-size: 12px;
  margin-left: 4px;
}

.member-count:hover {
  text-decoration: underline;
}
</style>
