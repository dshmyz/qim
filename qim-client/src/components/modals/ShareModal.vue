<template>
  <ModalContainer
    :visible="visible"
    :title="modalTitle"
    width="480px"
    @close="canClose ? close() : undefined"
    @cancel="canClose ? close() : undefined"
  >
    <!-- 转发内容预览 -->
    <div v-if="shareType === 'message' || shareType === 'file' || shareType === 'note' || shareType === 'sticky'" class="share-preview-card">
      <div class="share-preview-icon" :class="shareType">
        <i v-if="shareType === 'file'" class="fas fa-file"></i>
        <i v-else-if="shareType === 'note'" class="fas fa-file-alt"></i>
        <i v-else-if="shareType === 'sticky'" class="fas fa-sticky-note"></i>
        <i v-else class="fas fa-share-alt"></i>
      </div>
      <div class="share-preview-info">
        <div class="share-preview-title">{{ previewTitle }}</div>
        <div class="share-preview-desc">{{ previewDesc }}</div>
      </div>
    </div>

    <!-- 搜索框 -->
    <div class="share-search-box">
      <input
        v-model="searchQuery"
        type="text"
        class="share-search-input"
        placeholder="搜索用户或群聊..."
        :disabled="sending"
      />
      <i class="fas fa-search share-search-icon"></i>
    </div>

    <!-- 加载态骨架屏 -->
    <div v-if="shareLoading" class="share-skeleton">
      <div v-for="i in 5" :key="i" class="share-skeleton-item">
        <div class="share-skeleton-avatar"></div>
        <div class="share-skeleton-lines">
          <div class="share-skeleton-line short"></div>
          <div class="share-skeleton-line"></div>
        </div>
      </div>
    </div>

    <!-- Tab 列表（加载完成后） -->
    <template v-else>
      <!-- 常用联系人横排 -->
      <div v-if="recentContacts.length > 0 && activeTab !== 'conversations'" class="share-recent-strip">
        <div class="share-recent-label">常用联系人</div>
        <div class="share-recent-avatars">
          <div
            v-for="contact in recentContacts"
            :key="contact.id"
            class="share-recent-avatar-item"
            :class="{ selected: isContactSelected(contact) }"
            @click="toggleContact(contact)"
          >
            <Avatar :src="contact.avatar" :name="contact.name" :alt="contact.name" size="sm" />
            <span class="share-recent-name">{{ contact.name }}</span>
            <i v-if="isContactSelected(contact)" class="fas fa-check share-recent-check"></i>
          </div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="share-tabs">
        <button
          class="share-tab"
          :class="{ active: activeTab === 'conversations' }"
          @click="activeTab = 'conversations'"
        >
          最近会话
        </button>
        <button
          class="share-tab"
          :class="{ active: activeTab === 'users' }"
          @click="activeTab = 'users'"
        >
          用户
        </button>
        <button
          class="share-tab"
          :class="{ active: activeTab === 'groups' }"
          @click="activeTab = 'groups'"
        >
          群聊
        </button>
      </div>

      <!-- 最近会话列表 -->
      <div v-if="activeTab === 'conversations'" class="share-list">
        <div
          v-for="conv in filteredConversations"
          :key="conv.id"
          class="share-item"
          :class="{ selected: selectedConversations.includes(conv.id) || selectedGroups.includes(conv.id) }"
          @click="toggleConversationSelection(conv.id)"
        >
          <Avatar :src="conv.avatar" :name="conv.name" :alt="conv.name" size="md" class="share-item-avatar" />
          <div class="share-item-info">
            <div class="share-item-name">
              <i v-if="conv.is_pinned" class="fas fa-thumbtack share-item-pin"></i>
              {{ conv.name }}
            </div>
            <div class="share-item-desc">
              <span v-if="conv.type === 'single'">单聊</span>
              <span v-else-if="conv.type === 'group'">群聊 · {{ conv.members?.length || 0 }} 成员</span>
              <span v-else-if="conv.type === 'bot'">Bot</span>
              <span v-else>{{ conv.type }}</span>
              <template v-if="conv.lastMessage">
                · {{ getConvPreview(conv.lastMessage) }}
              </template>
            </div>
          </div>
          <div class="share-item-checkbox">
            <i v-if="selectedConversations.includes(conv.id) || selectedGroups.includes(conv.id)" class="fas fa-check"></i>
          </div>
        </div>
        <div v-if="filteredConversations.length === 0" class="empty-share">
          没有匹配的会话
        </div>
      </div>

      <!-- 用户列表（组织架构树） -->
      <div v-else-if="activeTab === 'users'" class="share-list">
        <OrgTreePicker
          v-if="props.departments && props.departments.length > 0"
          :org-structure="props.departments"
          :selected-members="selectedOrgMembers"
          :search-query="searchQuery"
          @update:selected-members="selectedOrgMembers = $event"
        />
        <div v-else class="empty-share">
          暂无组织架构
        </div>
      </div>

      <!-- 群聊列表 -->
      <div v-else-if="activeTab === 'groups'" class="share-list">
        <div
          v-for="group in filteredGroups"
          :key="group.id"
          class="share-item"
          :class="{ selected: selectedGroups.includes(group.id) }"
          @click="toggleGroupSelection(group.id)"
        >
          <Avatar :src="group.avatar" :name="group.name" :alt="group.name" size="md" class="share-item-avatar" />
          <div class="share-item-info">
            <div class="share-item-name">{{ group.name }}</div>
            <div class="share-item-desc">{{ group.members.length }} 成员</div>
          </div>
          <div class="share-item-checkbox">
            <i v-if="selectedGroups.includes(group.id)" class="fas fa-check"></i>
          </div>
        </div>
        <div v-if="filteredGroups.length === 0" class="empty-share">
          没有找到匹配的群聊
        </div>
      </div>
    </template>

    <template #footer>
      <!-- 发送进度 -->
      <div v-if="sending" class="share-sending">
        <div class="share-progress-bar">
          <div class="share-progress-fill" :style="{ width: progressPercent + '%' }"></div>
        </div>
        <span class="share-progress-text">正在发送 {{ sendingProgress?.sent || 0 }}/{{ sendingProgress?.total || 0 }}</span>
      </div>
      <template v-else>
        <span class="share-selected-count" v-if="totalSelected > 0">已选 {{ totalSelected }} 个接收方</span>
        <span class="share-max-hint" v-else-if="MAX_RECIPIENTS">最多选 {{ MAX_RECIPIENTS }} 个</span>
        <button class="cancel-btn" @click="close">取消</button>
        <button
          class="confirm-btn"
          :disabled="totalSelected === 0"
          @click="confirm"
        >
          {{ shareType === 'message' ? '转发' : '分享' }}
        </button>
      </template>
    </template>
  </ModalContainer>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Avatar from '../shared/Avatar.vue'
import ModalContainer from '../shared/ModalContainer.vue'
import OrgTreePicker from '../shared/OrgTreePicker.vue'
import { getMessagePreview } from '@/utils/messagePreview'
import type { Conversation } from '@/types'
import { useCurrentUser } from '@/composables/useCurrentUser'

const { currentUser } = useCurrentUser()

const props = defineProps<{
  visible: boolean
  share: { type: string; data: any }
  users: { id: string; name: string; username?: string; avatar: string; department?: string }[]
  groups: { id: string; name: string; avatar: string; members: any[] }[]
  departments?: { id: string; name: string; employees: any[]; subDepartments: any[] }[]
  shareLoading?: boolean
  sending?: boolean
  sendingProgress?: { sent: number; total: number }
  conversations?: Conversation[]
}>()

const emit = defineEmits(['close', 'confirm'])

const searchQuery = ref('')
const activeTab = ref('conversations')
const selectedGroups = ref<string[]>([])
const selectedConversations = ref<string[]>([])
const selectedOrgMembers = ref<any[]>([])

// selectedUsers 由 OrgTreePicker 选中项直接派生，消除双向 watch 冗余
const selectedUsers = computed(() =>
  selectedOrgMembers.value.map(m => String(m.id))
)

// ── 计算属性 ──
const shareType = computed(() => props.share?.type || '')
const shareData = computed(() => props.share?.data)
const canClose = computed(() => !props.sending)

const modalTitle = computed(() => {
  const prefix = shareType.value === 'file' ? '分享文件'
    : shareType.value === 'note' ? '分享笔记'
    : shareType.value === 'message' ? '转发消息'
    : '分享便签'
  return prefix
})

const totalSelected = computed(() =>
  selectedUsers.value.length + selectedGroups.value.length + selectedConversations.value.length
)

const progressPercent = computed(() => {
  const p = props.sendingProgress
  if (!p || p.total === 0) return 0
  return Math.round((p.sent / p.total) * 100)
})

// ── 转发预览 ──
const previewTitle = computed(() => {
  const data = shareData.value
  if (!data) return '消息'
  if (shareType.value === 'file') return data.name || '文件'
  if (shareType.value === 'note') return data.title || '笔记'
  if (shareType.value === 'sticky') return data.title || '便签'
  // message: 用标准消息格式化
  const messages = Array.isArray(data) ? data
    : Array.isArray(data?.messages) ? data.messages
    : [data]
  const first = messages[0]
  if (!first) return '消息'
  const { label } = getMessagePreview({ type: first.type || 'text', content: first.content || '' })
  return label.length > 20 ? label.slice(0, 20) + '...' : label
})

const previewDesc = computed(() => {
  const data = shareData.value
  if (shareType.value === 'file') return '文件'
  if (shareType.value === 'note') return '笔记'
  if (shareType.value === 'sticky') return '便签'
  const messages = Array.isArray(data) ? data
    : Array.isArray(data?.messages) ? data.messages
    : [data]
  if (messages.length > 1) return `共 ${messages.length} 条消息`
  const first = messages[0]
  if (first?.type === 'image') return '图片消息'
  if (first?.type === 'file') return '文件消息'
  if (first?.type === 'merged_forward') return '聊天记录'
  return '文本消息'
})

// ── 常用联系人（从会话列表取最近单聊） ──
// 会话排序：置顶优先 + 时间倒序（recentContacts / sortedConversations 共用）
const compareConversation = (a: any, b: any) => {
  if (a.is_pinned && !b.is_pinned) return -1
  if (!a.is_pinned && b.is_pinned) return 1
  return (b.timestamp || 0) - (a.timestamp || 0)
}

const recentContacts = computed(() => {
  if (!props.conversations || !Array.isArray(props.conversations)) return []
  return props.conversations
    .filter((c) => c.type === 'single' && !c.is_deleted)
    .sort(compareConversation)
    .slice(0, 8)
    .map((c) => ({
      id: c.id,
      name: c.name || c.other_member_name || '未知',
      avatar: c.avatar || '',
      type: 'single' as const,
      isGroup: false,
    }))
})

// ── 最近会话列表 ──
const sortedConversations = computed(() => {
  if (!props.conversations || !Array.isArray(props.conversations)) return []
  return [...props.conversations]
    .filter((c) => !c.is_deleted)
    .sort(compareConversation)
})

const filteredGroups = computed(() => {
  if (!props.groups || !Array.isArray(props.groups)) return []
  if (!searchQuery.value) return props.groups
  const query = searchQuery.value.toLowerCase()
  return props.groups.filter(group =>
    group.name.toLowerCase().includes(query)
  )
})

const filteredConversations = computed(() => {
  if (!searchQuery.value) return sortedConversations.value
  const query = searchQuery.value.toLowerCase()
  return sortedConversations.value.filter((c) =>
    (c.name && c.name.toLowerCase().includes(query)) ||
    (c.other_member_name && c.other_member_name.toLowerCase().includes(query))
  )
})

// ── 选中逻辑 ──
const isContactSelected = (contact: { id: string; type: string }) => {
  if (contact.type === 'single') return selectedConversations.value.includes(contact.id)
  return false
}

const toggleContact = (contact: { id: string; type: string }) => {
  if (contact.type === 'single') toggleConversationSelection(contact.id)
}

const MAX_RECIPIENTS = 50

const toggleGroupSelection = (groupId: string) => {
  const index = selectedGroups.value.indexOf(groupId)
  if (index > -1) {
    selectedGroups.value.splice(index, 1)
  } else if (totalSelected.value < MAX_RECIPIENTS) {
    selectedGroups.value.push(groupId)
  }
}

const toggleConversationSelection = (convId: string) => {
  const conv = props.conversations?.find((c) => String(c.id) === convId)
  if (!conv) return

  if (conv.type === 'group') {
    toggleGroupSelection(convId)
  } else {
    const index = selectedConversations.value.indexOf(convId)
    if (index > -1) {
      selectedConversations.value.splice(index, 1)
    } else if (totalSelected.value < MAX_RECIPIENTS) {
      selectedConversations.value.push(convId)
    }
  }
}

const getConvPreview = (lastMessage: any): string => {
  const { label } = getMessagePreview({ type: lastMessage.type || 'text', content: lastMessage.content || '' })
  return label.length > 20 ? label.slice(0, 20) + '...' : label
}

const close = () => {
  if (props.sending) return
  emit('close')
}

const confirm = () => {
  if (totalSelected.value === 0 || props.sending) return

  // 用 Map 做 O(1) 查找，避免对每个选中会话都线性扫描
  const convMap = new Map<string, Conversation>()
  for (const c of props.conversations ?? []) {
    convMap.set(String(c.id), c)
  }

  // selectedConversations 只含单聊（toggleConversationSelection 已分流群聊到 selectedGroups）。
  // other_member_id 缺失时跳过（数据异常），其余正常映射为 user ID。
  const singleConvUserIds = selectedConversations.value
    .map(convId => {
      const conv = convMap.get(convId)
      if (!conv?.other_member_id) return ''
      return conv.other_member_id.toString()
    })
    .filter(id => id !== '')

  // 去重：同一用户可能同时从组织树和单聊会话被选中，合并后去重避免重复发送
  const allUserIds = [...selectedUsers.value, ...singleConvUserIds]
  const uniqueUserIds = [...new Set(allUserIds)]

  emit('confirm', {
    users: uniqueUserIds,
    groups: selectedGroups.value
  })
}

watch(() => props.visible, (newVal) => {
  if (newVal) {
    selectedGroups.value = []
    selectedConversations.value = []
    selectedOrgMembers.value = [] // selectedUsers 是 computed，清 selectedOrgMembers 即可
    searchQuery.value = ''
    activeTab.value = 'conversations'
  }
})
</script>

<style scoped>
/* ── 转发预览卡片 ── */
.share-preview-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  margin-bottom: 12px;
  border-radius: 8px;
  background: var(--hover-color, #f5f7fa);
  border: 1px solid var(--border-color, #e8e8e8);
}

.share-preview-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: var(--font-size-sm);
  flex-shrink: 0;
}

.share-preview-icon.file {
  background: linear-gradient(135deg, #2563eb, #0ea5e9);
}

.share-preview-icon.note {
  background: linear-gradient(135deg, #7c3aed, #a855f7);
}

.share-preview-icon.sticky {
  background: linear-gradient(135deg, #f59e0b, #f97316);
}

.share-preview-icon.message {
  background: linear-gradient(135deg, #7c3aed, #2563eb);
}

.share-preview-info {
  min-width: 0;
}

.share-preview-title {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--text-color, #333);
}

.share-preview-desc {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, #999);
}

/* ── 搜索框 ── */
.share-search-box {
  position: relative;
  margin-bottom: 12px;
}

.share-search-input {
  width: 100%;
  padding: 8px 32px 8px 12px;
  border: 1px solid var(--border-color, #d9d9d9);
  border-radius: 6px;
  font-size: var(--font-size-sm);
  transition: all 0.2s;
  background: var(--input-bg, #ffffff);
  color: var(--text-color, #333);
  box-sizing: border-box;
}

.share-search-input:focus {
  outline: none;
  border-color: var(--primary-color, #1890ff);
  box-shadow: 0 0 0 2px var(--primary-light, rgba(24, 144, 255, 0.2));
}

.share-search-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.share-search-icon {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-secondary, #999);
  font-size: var(--font-size-sm);
}

/* ── 骨架屏 ── */
.share-skeleton {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 8px 0;
}

.share-skeleton-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.share-skeleton-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--hover-color, #f0f0f0);
  animation: skeleton-pulse 1.5s ease-in-out infinite;
}

.share-skeleton-lines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.share-skeleton-line {
  height: 12px;
  border-radius: 4px;
  background: var(--hover-color, #f0f0f0);
  animation: skeleton-pulse 1.5s ease-in-out infinite;
}

.share-skeleton-line.short {
  width: 60%;
}

@keyframes skeleton-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

/* ── 常用联系人横排 ── */
.share-recent-strip {
  margin-bottom: 12px;
}

.share-recent-label {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, #999);
  margin-bottom: 8px;
}

.share-recent-avatars {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  padding-bottom: 4px;
}

.share-recent-avatars::-webkit-scrollbar {
  height: 0;
}

.share-recent-avatar-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  position: relative;
  min-width: 56px;
  padding: 6px 4px;
  border-radius: 8px;
  transition: background 0.2s;
}

.share-recent-avatar-item:hover {
  background: var(--hover-color, #f5f5f5);
}

.share-recent-avatar-item.selected {
  background: var(--primary-light, #e6f7ff);
}

.share-recent-name {
  font-size: var(--font-size-xxxs);
  color: var(--text-color, #333);
  max-width: 56px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.share-recent-check {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--primary-color, #1890ff);
  color: #fff;
  font-size: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* ── Tabs ── */
.share-tabs {
  display: flex;
  margin-bottom: 12px;
  border-bottom: 1px solid var(--border-color, #e8e8e8);
}

.share-tab {
  flex: 1;
  padding: 8px 12px;
  background: none;
  border: none;
  font-size: var(--font-size-xs);
  color: var(--text-secondary, #666);
  cursor: pointer;
  transition: all 0.2s;
  border-bottom: 2px solid transparent;
}

.share-tab:hover {
  color: var(--primary-color, #1890ff);
}

.share-tab.active {
  color: var(--primary-color, #1890ff);
  border-bottom-color: var(--primary-color, #1890ff);
  font-weight: 500;
}

/* ── 列表 ── */
.share-list {
  max-height: 300px;
  overflow-y: auto;
}

.share-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
  margin-bottom: 4px;
}

.share-item:hover {
  background: var(--hover-color, #f5f5f5);
}

.share-item.selected {
  background: var(--primary-light, #e6f7ff);
  border: 1px solid var(--primary-color, #91d5ff);
}

.share-item-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  margin-right: 10px;
  object-fit: cover;
  flex-shrink: 0;
}

.share-item-info {
  flex: 1;
  min-width: 0;
}

.share-item-name {
  font-size: var(--font-size-xs);
  font-weight: 500;
  color: var(--text-color, #333);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: flex;
  align-items: center;
  gap: 4px;
}

.share-item-pin {
  font-size: var(--font-size-tiny);
  color: var(--primary-color, #1890ff);
}

.share-item-desc {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, #999);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 2px;
}

.share-item-checkbox {
  width: 20px;
  height: 20px;
  border: 1px solid var(--border-color, #d9d9d9);
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary-color, #1890ff);
  font-size: var(--font-size-xxs);
  transition: all 0.2s;
  flex-shrink: 0;
}

.share-item.selected .share-item-checkbox {
  background: var(--primary-color, #1890ff);
  border-color: var(--primary-color, #1890ff);
  color: white;
}

.empty-share {
  text-align: center;
  padding: 40px 0;
  color: var(--text-secondary, #999);
  font-size: var(--font-size-sm);
}

/* ── Footer ── */
.cancel-btn {
  padding: 6px 16px;
  border: 1px solid var(--border-color, #d9d9d9);
  border-radius: 6px;
  background: var(--card-bg, white);
  color: var(--text-color, #333);
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: all 0.2s;
}

.cancel-btn:hover {
  border-color: var(--primary-color, #1890ff);
  color: var(--primary-color, #1890ff);
}

.confirm-btn {
  padding: 6px 16px;
  border: 1px solid var(--primary-color, #1890ff);
  border-radius: 6px;
  background: var(--primary-color, #1890ff);
  color: white;
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: all 0.2s;
}

.confirm-btn:hover:not(:disabled) {
  background: var(--active-color, #40a9ff);
  border-color: var(--active-color, #40a9ff);
}

.confirm-btn:disabled {
  background: var(--color-gray-100, #f0f0f0);
  border-color: var(--border-color, #d9d9d9);
  color: var(--text-secondary, #999);
  cursor: not-allowed;
}

/* ── 发送进度 ── */
.share-sending {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}

.share-progress-bar {
  flex: 1;
  height: 6px;
  border-radius: 3px;
  background: var(--hover-color, #f0f0f0);
  overflow: hidden;
}

.share-progress-fill {
  height: 100%;
  border-radius: 3px;
  background: var(--primary-color, #1890ff);
  transition: width 0.3s ease;
}

.share-progress-text {
  font-size: var(--font-size-xs);
  color: var(--text-secondary, #999);
  white-space: nowrap;
  flex-shrink: 0;
}

.share-selected-count {
  font-size: var(--font-size-xs);
  color: var(--text-secondary, #999);
  margin-right: auto;
}

.share-max-hint {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, #bbb);
  margin-right: auto;
}
</style>
