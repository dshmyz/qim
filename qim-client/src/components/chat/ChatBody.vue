<template>
  <div class="chat-main">
    <!-- 消息列表 -->
    <MessageListView
      ref="messageListViewRef"
      :messages="messages"
      :has-more-messages="hasMoreMessages"
      :conversation-type="conversation?.type || 'single'"
      :read-users-map="readUsersMap"
      :server-url="serverUrl"
      @message-contextmenu="(e, m) => emit('message-contextmenu', e, m)"
      @show-user-profile="(u) => emit('show-user-profile', u)"
      @scroll-to-quoted-message="(id) => emit('scroll-to-quoted-message', id)"
      @preview-image="(d) => emit('preview-image', d)"
      @download-file="(d) => emit('download-file', d)"
      @save-as="(d) => emit('save-as', d)"
      @open-mini-app="(a) => emit('open-mini-app', a)"
      @open-news-link="(u) => emit('open-news-link', u)"
      @retry-send-message="(m) => emit('retry-send-message', m)"
      @show-read-users="(m) => emit('show-read-users', m)"
      @mark-read="emit('mark-read')"
      @load-more="emit('load-more')"
    />

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
import { getAvatarUrl } from '../../utils/avatar'

/** MemberSidebar 组件内部使用的 Member 类型 */
interface Member {
  id: string
  name: string
  avatar: string
  role?: 'owner' | 'admin' | 'member'
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
  serverUrl: string
  isMembersSidebarExpanded: boolean
  showMemberSearch: boolean
  memberSearchQuery: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'message-contextmenu': [event: MouseEvent, message: Message]
  'show-user-profile': [user: User]
  'scroll-to-quoted-message': [id: string]
  'preview-image': [data: string]
  'download-file': [data: string]
  'save-as': [data: string]
  'open-mini-app': [app: Message['miniAppData']]
  'open-news-link': [url: string]
  'retry-send-message': [msg: Message]
  'show-read-users': [msg: Message]
  'mark-read': []
  'load-more': []
  'toggle-members-sidebar': []
  'toggle-member-search': []
  'member-search-focus': []
  'show-member-context-menu': [event: MouseEvent, member: Member]
  'start-private-chat': [member: Member]
  'update:member-search-query': [value: string]
}>()

const messageListViewRef = ref<InstanceType<typeof MessageListView>>()

const memberSearchQueryLocal = computed({
  get: () => props.memberSearchQuery,
  set: (val) => emit('update:member-search-query', val)
})

const showMemberSidebar = computed(() => {
  return props.conversation?.type === 'group' || props.conversation?.type === 'discussion'
})

/** 将 User[] 映射为 Member[]，适配 MemberSidebar 的类型要求 */
const sidebarMembers = computed<Member[]>(() => {
  return (props.conversation?.members || []).map(user => ({
    id: user.id,
    name: user.name,
    avatar: getAvatarUrl(user.avatar, user.name || '用户', props.serverUrl),
    role: user.role as Member['role'] ?? 'member'
  }))
})

defineExpose({
  scrollToBottom: () => messageListViewRef.value?.scrollToBottom()
})
</script>

<style scoped>
.chat-main {
  flex: 1;
  display: flex;
  overflow: hidden;
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.03);
}
</style>
