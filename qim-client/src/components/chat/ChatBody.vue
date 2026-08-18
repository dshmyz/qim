<template>
  <div class="chat-main">
    <!-- 消息列表 -->
    <MessageListView
      ref="messageListViewRef"
      :messages="messages"
      :has-more-messages="hasMoreMessages"
      :conversation-type="conversation?.type || 'single'"
      :read-users-map="readUsersMap"
      :show-read-receipt="showReadReceipt"
      :server-url="serverUrl"
      :current-user-id="currentUserId"
      :selection-mode="selectionMode"
      :selected-message-ids="selectedMessageIds"
      @message-contextmenu="(e, m) => emit('message-contextmenu', e, m)"
      @show-user-profile="(u) => emit('show-user-profile', u)"
      @quick-mention="(u, e) => emit('quick-mention', u, e)"
      @mention-user="(u) => emit('mention-user', u)"
      @scroll-to-quoted-message="(id) => emit('scroll-to-quoted-message', id)"
      @download-file="(d, id) => emit('download-file', d, id)"
      @save-as="(d, id) => emit('save-as', d, id)"
      @open-mini-app="(a) => emit('open-mini-app', a)"
      @open-news-link="(u) => emit('open-news-link', u)"
      @retry-send-message="(m) => emit('retry-send-message', m)"
      @show-read-users="(m) => emit('show-read-users', m)"
      @mark-read="emit('mark-read')"
      @toggle-message-selection="emit('toggle-message-selection', $event)"
      @load-more="emit('load-more')"
      @scroll="handleMessageListScroll"
      @recall-edit="(originalContent: string) => emit('recall-edit', originalContent)"
    >
      <!-- AI 思考中占位：ai_reply_started 置位后、首个回复消息到达前显示。
           渲染成带头像的正常消息行（头像+名字行+气泡），与 AI 消息同构：
           名字行在所有会话类型都显示（单聊/bot 的 assistant 消息同样有名字行，见
           MessageItem 35-36 行），头像带机器人徽章（buildSenderBadge，与正常 AI 消息一致） -->
      <template #thinking-indicator>
        <div v-if="aiThinking" class="thinking-message-row">
          <Avatar
            :src="aiThinkingSender?.avatar"
            :name="aiThinkingSender?.name || 'AI 助手'"
            :server-url="serverUrl"
            :badge="thinkingSenderBadge"
            size="md"
            alt="AI 助手"
            class="thinking-avatar"
          />
          <div class="thinking-message-content">
            <div class="thinking-message-sender">
              <span><i class="fas fa-robot"></i> {{ aiThinkingSender?.name || 'AI 助手' }}</span>
            </div>
            <div class="thinking-message-bubble">
              <ThinkingIndicator />
            </div>
          </div>
        </div>
      </template>
    </MessageListView>

    <!-- 群成员侧边栏 -->
    <MemberSidebar
      v-if="showMemberSidebar"
      :members="sidebarMembers"
      :is-expanded="isMembersSidebarExpanded"
      :show-search="showMemberSearch"
      v-model:search-query="memberSearchQueryLocal"
      @toggle-expanded="emit('toggle-members-sidebar')"
      @toggle-member-search="emit('toggle-member-search')"
      @search-focus="emit('member-search-focus')"
      @show-member-context-menu="(e, m) => emit('show-member-context-menu', e, m)"
      @start-private-chat="(member) => emit('start-private-chat', member)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Conversation, Message, User } from '../../types'
import MessageListView from './MessageListView.vue'
import MemberSidebar from './MemberSidebar.vue'
import ThinkingIndicator from '../shared/ThinkingIndicator.vue'
import Avatar from '../shared/Avatar.vue'
import { useChatStore } from '../../stores/chat'
import { getAvatarUrl } from '../../utils/avatar'
import { buildSenderBadge } from '../../utils/user'

/** MemberSidebar 组件内部使用的 Member 类型 */
interface Member {
  id: string
  name: string
  avatar: string
  role?: 'owner' | 'admin' | 'member'
  type?: string
  position?: string
  department?: string
  status?: string
  signature?: string
  ip?: string
  username?: string
  nickname?: string
  user?: Record<string, any>
}

/** 消息已读信息 */
interface MessageReadInfo {
  read_users: User[]
  total_members: number
}

interface Props {
  conversation: Conversation | null
  messages: Message[]
  hasMoreMessages: boolean
  readUsersMap: Record<string, MessageReadInfo>
  showReadReceipt: boolean
  serverUrl: string
  currentUserId?: string | number
  selectionMode: boolean
  selectedMessageIds: Set<string>
  isMembersSidebarExpanded: boolean
  showMemberSearch: boolean
  memberSearchQuery: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'message-contextmenu': [event: MouseEvent, message: Message]
  'show-user-profile': [user: User]
  'quick-mention': [user: any, event: MouseEvent]
  'mention-user': [user: any]
  'scroll': []
  'scroll-to-quoted-message': [id: string]
  'download-file': [data: string, id?: string]
  'save-as': [data: string, id?: string]
  'open-mini-app': [app: Message['miniAppData']]
  'open-news-link': [url: string]
  'retry-send-message': [msg: Message]
  'show-read-users': [msg: Message]
  'mark-read': []
  'toggle-message-selection': [messageId: string]
  'load-more': []
  'toggle-members-sidebar': []
  'toggle-member-search': []
  'member-search-focus': []
  'show-member-context-menu': [event: MouseEvent, member: Member]
  'start-private-chat': [member: Member]
  'update:member-search-query': [value: string]
  'recall-edit': [originalContent: string]
}>()

const messageListViewRef = ref<InstanceType<typeof MessageListView>>()

const memberSearchQueryLocal = computed({
  get: () => props.memberSearchQuery,
  set: (val) => emit('update:member-search-query', val)
})

const showMemberSidebar = computed(() => {
  return props.conversation?.type === 'group' || props.conversation?.type === 'discussion'
})

const chatStore = useChatStore()

/** 当前会话是否处于 AI 回复「思考中」（由 ai_reply_started 事件置位） */
const aiThinking = computed(() => {
  const id = props.conversation?.id
  return id != null && chatStore.isAiThinking(String(id))
})

/** 当前会话 AI 回复者身份（事件携带），供占位消息行渲染头像/昵称；事件缺 sender 时兜底 AI 助手 */
const aiThinkingSender = computed(() => {
  const id = props.conversation?.id
  return id != null ? chatStore.getAiThinkingSender(String(id)) : null
})

/** 占位头像徽章：思考中一定是 AI/机器人回复，与正常 AI 消息同款机器人角标
    （buildSenderBadge(sender, isAI=true)：无 type 时兜底 bot） */
const thinkingSenderBadge = computed(() => buildSenderBadge(aiThinkingSender.value, true))

/** 将 User[] 映射为 Member[]，适配 MemberSidebar 的类型要求（保留名片展示所需字段） */
const sidebarMembers = computed<Member[]>(() => {
  return (props.conversation?.members || []).map(user => ({
    id: user.id,
    name: user.name,
    avatar: getAvatarUrl(user.avatar, user.name || '用户', props.serverUrl),
    role: user.role as Member['role'] ?? 'member',
    type: (user as any).type,
    position: (user as any).position || '',
    department: (user as any).department || '',
    status: user.status,
    signature: user.signature,
    ip: user.ip,
    username: user.username,
    nickname: user.nickname,
    user: user as Record<string, any>
  }))
})

defineExpose({
  scrollToBottom: (instant: boolean = false) => messageListViewRef.value?.scrollToBottom(instant),
  scrollToBottomWithDelay: (delay: number = 100) => messageListViewRef.value?.scrollToBottomWithDelay(delay)
})
const handleMessageListScroll = () => emit('scroll')
</script>

<style scoped>
.chat-main {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* 「思考中」占位消息行：与 MessageItem 同构（40px 头像 + 12px 间距 + 气泡），
   让占位看起来像一条正常的 AI 消息而非孤立的圆点 */
.thinking-message-row {
  display: flex;
  align-items: flex-start;
  padding: 6px 20px;
}
.thinking-avatar {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
}
.thinking-message-content {
  max-width: 80%;
  min-width: 0;
  margin-left: 12px;
}
.thinking-message-sender {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  margin-bottom: 2px;
}
.thinking-message-bubble {
  display: inline-flex;
  align-items: center;
  padding: 10px 14px;
  border-radius: 12px;
  background: var(--message-bubble-bg);
}
.thinking-message-bubble :deep(.thinking-indicator) {
  padding: 0;
}
</style>
