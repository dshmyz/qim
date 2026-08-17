<template>
  <div class="chat-header">
    <div class="header-info">
      <!-- 单聊/机器人头像可点：弹出随行资料小卡（跟随头像位置，移开即消失，与群成员名片一致） -->
      <div
        class="avatar-wrapper"
        :class="{ 'clickable': isAvatarClickable }"
        @click="popoverOpen"
        @mouseleave="popoverScheduleHide"
      >
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
        <div class="header-name">
          <span class="header-name-text">{{ displayName }}</span>
          <span v-if="isBotChat" class="bot-badge"><i class="fas fa-robot"></i>AI 机器人</span>
        </div>
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
      :avatar-enabled-raw="avatarEnabledRaw"
      :avatar-approval-status="avatarApprovalStatus"
      :ai-extract-todos="aiExtractTodos"
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
      @update-extract-todos="(value) => emit('update-extract-todos', value)"
      @open-group-files="emit('open-group-files')"
    />

    <!-- 单聊/机器人随行资料小卡：点击头部头像弹出，跟随头像位置，移开即消失 -->
    <Teleport to="body">
      <UserProfileCard
        v-if="isAvatarClickable && popoverState"
        :member="popoverState.member"
        :server-url="serverUrl"
        :fallback-avatar="conversation?.avatar"
        :style="popoverStyle"
        @mouseenter="popoverCancelHide"
        @mouseleave="popoverScheduleHide"
      />
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Conversation } from '../../types'
import Avatar from '../shared/Avatar.vue'
import ChatHeaderActions from './ChatHeaderActions.vue'
import UserProfileCard from '../shared/UserProfileCard.vue'
import { buildConversationBadge } from '../../utils/user'
import { useProfilePopover, type ProfileMember } from '../../composables/useProfilePopover'

interface Props {
  conversation: Conversation
  currentUser: any
  serverUrl: string
  avatarEnabled?: boolean
  avatarEnabledRaw?: boolean
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
  (e: 'update-extract-todos', enabled: boolean): void
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
const aiLearnEnabled = computed(() => props.conversation?.ai_config?.ai_learn_enabled ?? true)
const aiExtractTodos = computed(() => props.conversation?.ai_config?.ai_extract_todos ?? false)
const approvalStatus = computed(() => props.conversation?.approval_status ?? 'approved')
const rejectReason = computed(() => props.conversation?.reject_reason ?? '')
const contextMessages = computed(() => props.conversation?.context_messages ?? 10)

const isGroupOrDiscussion = computed(() =>
  props.conversation?.type === 'group' || props.conversation?.type === 'discussion'
)

const isSingleChat = computed(() => props.conversation?.type === 'single')

// bot 1:1 会话：头部显示「AI 机器人」徽标，与普通单聊/群聊区分（bot 虚拟用户昵称 = bot 名）。
const isBotChat = computed(() => props.conversation?.type === 'bot')

// 聊天头部头像角标：统一由 buildConversationBadge 构造，Avatar 只负责渲染。
// ChatHeader 持有 currentUser，传入以支持 single 会话 partner 推导时排除自己。
const headerBadge = computed(() =>
  buildConversationBadge(props.conversation, props.currentUser?.id)
)

const displayName = computed(() => props.conversation?.name || '未知会话')

// ---------------------------------------------------------------------------
// 单聊/机器人随行资料小卡：点击头部头像弹出，跟随头像位置，移开即消失。
// 行为共享 useProfilePopover（与群成员名片对齐）：重复点击切换、悬停保持、
// 离开头像/卡片 200ms 后收起；列表数据不完整时异步拉取用户详情富化名片。
// ---------------------------------------------------------------------------

const isAvatarClickable = computed(() => isSingleChat.value || isBotChat.value)

// 对端身份解析：
// - 单聊：members 中非自己的成员；单聊会话的 members 常为空，回退会话级
//   other_member_id / other_member_name（取不到对端 id 时无法打开资料卡）。
// - 机器人：对端即 bot 的虚拟用户（成员里非自己的那个），名字/头像取会话级
//   （bot 1:1 会话直接以 bot 身份命名），卡片只体现「AI 助手」身份、不显示个人资料。
const buildPeerProfile = (): ProfileMember | null => {
  const conv = props.conversation
  if (!conv) return null
  const cid = String(props.currentUser?.id ?? '')
  const others = conv.members || []
  const other = others.find(m => String((m as any).id) !== cid) || others[0] || null
  const o = other as any

  if (isSingleChat.value) {
    const userId = other ? (o.user?.id ?? o.id) : (conv.other_member_id ?? null)
    if (!userId) return null
    return {
      id: String(userId),
      name: (other ? (o.name || o.nickname) : null) || conv.other_member_name || displayName.value,
      avatar: (other ? o.avatar : null) || conv.avatar || '',
      username: other ? o.username : (conv as any).username,
      email: other ? o.email : undefined,
      mobile: other ? (o.mobile ?? o.phone) : undefined,
      department: other ? o.department : undefined,
      position: other ? o.position : undefined,
      signature: (other ? o.signature : null) || conv.signature,
      ip: (other ? o.ip : null) || conv.ip,
      status: other ? o.status : conv.status,
      user: other ? o : undefined
    }
  }

  if (isBotChat.value) {
    const userId = other ? (o.user?.id ?? o.id) : (conv.other_member_id ?? null)
    // 兜底用会话 id 保持「重复点击切换」稳定（取不到虚拟用户 id 的极端情形）
    return {
      id: String(userId ?? conv.id),
      name: (other ? (o.name || o.nickname) : null) || conv.other_member_name || displayName.value,
      avatar: (other ? o.avatar : null) || conv.avatar || '',
      type: 'bot',
      status: other ? o.status : conv.status,
      user: other ? o : undefined
    }
  }

  return null
}

// 共享随行资料小卡。解构到组件顶层：popoverState/popoverStyle 是 ref，模板里
// 顶层 ref 才能自动解包（嵌套在返回对象里访问 popover.popoverState 不会解包，
// RefImpl 恒真值 + 无 .member 属性，会让 UserProfileCard 收到 undefined member）。
const {
  popoverState,
  popoverStyle,
  open: popoverOpen,
  scheduleHide: popoverScheduleHide,
  cancelHide: popoverCancelHide
} = useProfilePopover(buildPeerProfile)

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
  min-width: 0;
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

.avatar-wrapper.clickable {
  cursor: pointer;
}

.header-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

/* 名字 + 「AI 机器人」徽标同行：名字可截断（flex），徽标 flex-shrink:0 永远可见 */
.header-name {
  font-weight: 500;
  font-size: var(--font-size-sm);
  color: var(--text-color);
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.header-name-text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
  flex: 0 1 auto;
}

/* 「AI 机器人」徽标：bot 身份紫色胶囊 + 机器人图标（与群内 bot 角色标签同款 accent） */
.bot-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: var(--font-size-xxxs);
  font-weight: 500;
  line-height: 1.3;
  color: #7c3aed;
  background: linear-gradient(135deg, rgba(124, 58, 237, 0.10), rgba(79, 172, 254, 0.06));
  border: 1px solid rgba(124, 58, 237, 0.20);
  white-space: nowrap;
}

.bot-badge i {
  font-size: var(--font-size-tiny);
  line-height: 1;
}

.header-status {
  font-size: var(--font-size-xxs);
  color: var(--color-success-500);
  display: flex;
  align-items: center;
  gap: 8px;
}

.ip-info {
  color: var(--text-color);
  opacity: 0.7;
  font-size: var(--font-size-xxxs);
  /* padding: 2px 0px; */
  /* background: var(--hover-color); */
  border-radius: 3px;
}

.online-status {
  font-size: var(--font-size-xxs);
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
  font-size: var(--font-size-xxs);
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
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.header-announcement-inline i {
  font-size: var(--font-size-xxxs);
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
  font-size: var(--font-size-xxs);
  font-weight: 500;
  color: var(--text-secondary);
  white-space: nowrap;
}
</style>
