<template>
  <div class="group-panel-container">
    <!-- 头部操作区域 -->
    <div class="group-header-actions">
      <span v-if="isGroupOrDiscussion" class="header-icon" title="邀请成员" @click.stop="handleInviteMembers">
        <i class="fas fa-user-plus"></i>
      </span>
      <span class="header-icon" @click.stop="handleToggleHeaderMenu" ref="moreButtonRef">
        <i class="fas fa-ellipsis-v"></i>
      </span>
    </div>

    <!-- 头部下拉菜单 -->
    <Teleport to="body">
      <Transition name="dropdown">
        <div v-if="showHeaderMenu" class="header-menu-teleport" :style="headerMenuPosition" @click.stop>
          <div class="menu-item" @click="handleEditGroupInfo">
            <i class="fas fa-edit"></i> 修改群名称
          </div>
          <div class="menu-item" @click="handleEditGroupAnnouncement">
            <i class="fas fa-bullhorn"></i> 编辑群公告
          </div>
          <div v-if="isOwner" class="menu-item" @click="handleConfirmDeleteGroup">
            <i class="fas fa-trash"></i> 解散群聊
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 本地确认对话框 -->
    <Teleport to="body">
      <div v-if="localShowConfirmDialog" class="confirm-dialog-modal" @click="closeLocalConfirmDialog">
        <div class="confirm-dialog-content" @click.stop>
          <div class="confirm-dialog-header">
            <h3>{{ localConfirmDialogTitle }}</h3>
            <button class="close-btn" @click="closeLocalConfirmDialog">&times;</button>
          </div>
          <div class="confirm-dialog-body">
            <p>{{ localConfirmDialogMessage }}</p>
          </div>
          <div class="confirm-dialog-footer">
            <button class="cancel" @click="closeLocalConfirmDialog">取消</button>
            <button class="confirm" @click="executeConfirmCallback">确定</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 编辑群信息模态框 -->
    <Teleport to="body">
      <div v-if="showEditGroupInfoModal" class="modal-overlay" @click="handleCloseEditGroupInfoModal">
        <div class="modal-content" @click.stop>
          <div class="modal-header">
            <h3>修改群名称</h3>
            <button class="close-btn" @click="handleCloseEditGroupInfoModal">&times;</button>
          </div>
          <div class="modal-body">
            <div class="form-group">
              <label>群名称</label>
              <input
                type="text"
                :value="editGroupName"
                @input="handleEditGroupNameInput"
                class="form-input"
                placeholder="请输入新的群名称"
              />
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" @click="handleCloseEditGroupInfoModal">取消</button>
            <button class="btn btn-primary" @click="handleSaveGroupInfo">保存</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- 编辑群公告模态框 -->
    <Teleport to="body">
      <div v-if="showEditAnnouncementModal" class="modal-overlay" @click="handleCloseEditAnnouncementModal">
        <div class="modal-content" @click.stop>
          <div class="modal-header">
            <h3>编辑群公告</h3>
            <button class="close-btn" @click="handleCloseEditAnnouncementModal">&times;</button>
          </div>
          <div class="modal-body">
            <div class="form-group">
              <label>群公告内容</label>
              <textarea
                :value="editAnnouncement"
                @input="handleEditAnnouncementInput"
                class="form-textarea"
                placeholder="输入群公告内容..."
                rows="5"
              ></textarea>
              <p class="form-tip">群公告将对所有群成员可见</p>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-secondary" @click="handleCloseEditAnnouncementModal">取消</button>
            <button class="btn btn-primary" @click="handleSaveAnnouncement">保存</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import QMessage from '../../utils/qmessage'
import type { Conversation } from '../../types'
import { getCurrentUser } from '../../utils/user'
import MemberSidebar from './MemberSidebar.vue'
import MemberContextMenu from './MemberContextMenu.vue'

// 类型定义
interface GroupMember {
  id: string
  name: string
  role?: 'owner' | 'admin' | 'member' | 'user' | 'guest'
  avatar?: string
  [key: string]: unknown
}

interface GroupConversation extends Omit<Conversation, 'members'> {
  announcement?: string
  members?: GroupMember[]
}

// Props 定义
interface Props {
  conversation: Conversation | null
  currentUser: any
  showHeaderMenu: boolean
  showEditGroupInfoModal: boolean
  showEditAnnouncementModal: boolean
  editGroupName: string
  editAnnouncement: string
}

const props = defineProps<Props>()

// Emits 定义
const emit = defineEmits<{
  'update:showHeaderMenu': [value: boolean]
  'update:showEditGroupInfoModal': [value: boolean]
  'update:showEditAnnouncementModal': [value: boolean]
  'update:editGroupName': [value: string]
  'update:editAnnouncement': [value: string]
  'invite-members': []
  'delete-group': []
  'save-group-info': [groupName: string]
  'save-group-announcement': [announcement: string]
  'switch-conversation': [conversationId: string]
  'show-user-profile': [user: any]
  'remove-member': [memberId: string, memberName: string]
  'set-admin': [memberId: string, memberName: string, isAdmin: boolean]
  'transfer-owner': [memberId: string, memberName: string]
  'start-private-chat': [memberId: string]
}>()

// Refs
const moreButtonRef = ref<HTMLElement | null>(null)
const headerMenuPosition = ref<Record<string, string>>({})

// 本地确认对话框状态
const localShowConfirmDialog = ref(false)
const localConfirmDialogTitle = ref('确认操作')
const localConfirmDialogMessage = ref('')
const localConfirmDialogCallback = ref<(() => void) | null>(null)

// 成员管理状态
const showMemberContextMenuFlag = ref(false)
const memberContextMenuPosition = ref({ x: 0, y: 0 })
const selectedMember = ref<GroupMember | null>(null)
const isMembersSidebarExpanded = ref(true)
const showMemberSearch = ref(false)
const memberSearchQuery = ref('')

// Computed
const isGroupOrDiscussion = computed(() => {
  return props.conversation?.type === 'group' || props.conversation?.type === 'discussion'
})

const members = computed(() => {
  return props.conversation?.members || []
})

const isOwner = computed(() => {
  return isGroupOwner(props.conversation)
})

const currentUserId = computed((): string | number => {
  const user = props.currentUser || getCurrentUser()
  return user?.id ?? ''
})

const currentUserRole = computed((): string => {
  let currentUser = props.currentUser
  if (!currentUser) {
    currentUser = getCurrentUser()
  }
  if (!props.conversation?.members || !currentUser) return 'member'
  const member = props.conversation.members.find((m) => String(m.id) === String(currentUser.id))
  return (member?.role as string) || 'member'
})

// 方法
function isGroupOwner(conversation: Conversation | null): boolean {
  if (!conversation || !conversation.members) return false
  const currentUser = getCurrentUser()
  if (!currentUser) return false
  const currentUserId = currentUser.id?.toString() || ''
  const owner = conversation.members.find((member: any) => String(member.id) === currentUserId)
  return owner ? owner.role === 'owner' : false
}

// 切换头部下拉菜单
function handleToggleHeaderMenu() {
  const newValue = !props.showHeaderMenu
  emit('update:showHeaderMenu', newValue)
  if (newValue) {
    nextTick(() => {
      if (moreButtonRef.value) {
        const rect = moreButtonRef.value.getBoundingClientRect()
        const menuWidth = 180
        const menuHeight = 150
        const viewportWidth = window.innerWidth
        const viewportHeight = window.innerHeight

        let right = viewportWidth - rect.right
        if (right + menuWidth > viewportWidth) {
          right = 16
        }

        let top = rect.bottom + 8
        if (top + menuHeight > viewportHeight) {
          top = rect.top - menuHeight - 8
        }

        headerMenuPosition.value = {
          position: 'fixed',
          top: `${top}px`,
          right: `${right}px`
        }
      }
      document.addEventListener('click', closeHeaderMenu)
    })
  } else {
    document.removeEventListener('click', closeHeaderMenu)
  }
}

function closeHeaderMenu() {
  emit('update:showHeaderMenu', false)
  document.removeEventListener('click', closeHeaderMenu)
}

// 邀请成员
function handleInviteMembers() {
  emit('invite-members')
  emit('update:showHeaderMenu', false)
  document.removeEventListener('click', closeHeaderMenu)
}

// 编辑群信息
function handleEditGroupInfo() {
  if (props.conversation) {
    emit('update:editGroupName', props.conversation.name || '')
    emit('update:showEditGroupInfoModal', true)
  }
  emit('update:showHeaderMenu', false)
  document.removeEventListener('click', closeHeaderMenu)
}

function handleCloseEditGroupInfoModal() {
  emit('update:showEditGroupInfoModal', false)
}

function handleSaveGroupInfo() {
  emit('save-group-info', props.editGroupName)
  emit('update:showEditGroupInfoModal', false)
}

function handleEditGroupNameInput(event: Event) {
  const target = event.target as HTMLInputElement
  emit('update:editGroupName', target.value)
}

// 编辑群公告
function handleEditGroupAnnouncement() {
  if (props.conversation) {
    emit('update:editAnnouncement', (props.conversation as GroupConversation).announcement || '')
    emit('update:showEditAnnouncementModal', true)
  }
  emit('update:showHeaderMenu', false)
  document.removeEventListener('click', closeHeaderMenu)
}

function handleCloseEditAnnouncementModal() {
  emit('update:showEditAnnouncementModal', false)
}

function handleSaveAnnouncement() {
  emit('save-group-announcement', props.editAnnouncement)
  emit('update:showEditAnnouncementModal', false)
}

function handleEditAnnouncementInput(event: Event) {
  const target = event.target as HTMLTextAreaElement
  emit('update:editAnnouncement', target.value)
}

// 确认解散群聊
function handleConfirmDeleteGroup() {
  if (!props.conversation) return

  closeHeaderMenu()

  openLocalConfirmDialog(
    '确认解散群聊',
    '确定要解散此群聊吗？解散后所有消息和成员数据将被删除。',
    () => {
      emit('delete-group')
    }
  )
}

// 本地确认对话框方法
function openLocalConfirmDialog(title: string, message: string, callback: () => void) {
  localConfirmDialogTitle.value = title
  localConfirmDialogMessage.value = message
  localConfirmDialogCallback.value = callback
  localShowConfirmDialog.value = true
}

function closeLocalConfirmDialog() {
  localShowConfirmDialog.value = false
  localConfirmDialogCallback.value = null
}

function executeConfirmCallback() {
  if (localConfirmDialogCallback.value) {
    localConfirmDialogCallback.value()
  }
  closeLocalConfirmDialog()
}

// 成员侧边栏操作
function toggleMembersSidebar() {
  isMembersSidebarExpanded.value = !isMembersSidebarExpanded.value
}

function toggleMemberSearch() {
  showMemberSearch.value = !showMemberSearch.value
  if (showMemberSearch.value) {
    memberSearchQuery.value = ''
  }
}

// 成员右键菜单
function handleShowMemberContextMenu(event: MouseEvent, member: GroupMember) {
  event.stopPropagation()

  const menuWidth = 180
  const menuHeight = 80
  const windowWidth = window.innerWidth
  const windowHeight = window.innerHeight

  let x = event.clientX
  let y = event.clientY

  if (x + menuWidth > windowWidth) {
    x = windowWidth - menuWidth - 10
  }

  if (y + menuHeight > windowHeight) {
    y = windowHeight - menuHeight - 10
  }

  memberContextMenuPosition.value = { x, y }
  selectedMember.value = member
  showMemberContextMenuFlag.value = true

  setTimeout(() => {
    document.addEventListener('click', closeMemberContextMenu)
  }, 0)
}

function closeMemberContextMenu() {
  showMemberContextMenuFlag.value = false
  selectedMember.value = null
  document.removeEventListener('click', closeMemberContextMenu)
}

function handleStartPrivateChat(member: GroupMember) {
  emit('start-private-chat', member.id)
  closeMemberContextMenu()
}

// 成员操作 - 转发给父组件处理
function handleViewMemberInfo() {
  if (selectedMember.value) {
    emit('show-user-profile', selectedMember.value)
  }
  closeMemberContextMenu()
}

function handleRemoveMember() {
  if (selectedMember.value) {
    emit('remove-member', selectedMember.value.id, selectedMember.value.name)
  }
  closeMemberContextMenu()
}

function handleSetAdmin() {
  if (selectedMember.value) {
    const isAdmin = selectedMember.value.role === 'admin'
    emit('set-admin', selectedMember.value.id, selectedMember.value.name, !isAdmin)
  }
  closeMemberContextMenu()
}

function handleTransferOwner() {
  if (selectedMember.value) {
    emit('transfer-owner', selectedMember.value.id, selectedMember.value.name)
  }
  closeMemberContextMenu()
}

function handleSendPrivateMessage() {
  if (selectedMember.value) {
    emit('start-private-chat', selectedMember.value.id)
  }
  closeMemberContextMenu()
}

// 清理
onUnmounted(() => {
  document.removeEventListener('click', closeHeaderMenu)
  document.removeEventListener('click', closeMemberContextMenu)
})
</script>

<style scoped>
.group-panel-container {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  justify-content: flex-end;
}

.group-header-actions {
  display: flex;
  gap: 8px;
  position: relative;
}

.header-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--text-color);
  opacity: 0.7;
  font-size: 14px;
  border-radius: 6px;
  transition: background 0.2s;
  position: relative;
}

.header-icon:hover {
  background: var(--hover-color);
  opacity: 1;
}

/* 头部下拉菜单 */
.header-menu-teleport {
  position: fixed;
  background: var(--sidebar-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 9999;
  min-width: 180px;
  overflow: hidden;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  transition: background-color 0.2s;
  font-size: 14px;
}

.menu-item:hover {
  background-color: var(--hover-bg);
}

.menu-item i {
  margin-right: 8px;
  color: var(--text-secondary);
}

/* 模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.modal-content {
  background: var(--sidebar-bg);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  width: 90%;
  max-width: 500px;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-color);
}

.modal-header .close-btn {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary);
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.modal-header .close-btn:hover {
  background: var(--hover-bg);
}

.modal-body {
  padding: 20px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  background: var(--background-light);
}

/* 表单样式 */
.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.form-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  background: var(--background-light);
  color: var(--text-color);
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.form-textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  background: var(--background-light);
  color: var(--text-color);
  resize: vertical;
  min-height: 100px;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.form-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
}

.form-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 0;
}

/* 按钮样式 */
.btn {
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-secondary {
  background: var(--background-light);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover {
  background: var(--hover-bg);
}

/* 确认对话框样式 */
.confirm-dialog-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  backdrop-filter: blur(5px);
}

.confirm-dialog-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
  width: 90%;
  max-width: 400px;
  overflow: hidden;
}

.confirm-dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: var(--sidebar-bg);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  border-bottom: 1px solid var(--border-color);
}

.confirm-dialog-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.confirm-dialog-header .close-btn {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary);
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.confirm-dialog-header .close-btn:hover {
  background: var(--hover-bg);
}

.confirm-dialog-body {
  padding: 24px;
  background: var(--sidebar-bg);
}

.confirm-dialog-body p {
  margin: 0;
  font-size: 14px;
  color: var(--text-color);
  line-height: 1.5;
  text-align: center;
}

.confirm-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  background: var(--sidebar-bg);
  border-top: 1px solid var(--border-color);
}

.confirm-dialog-footer button {
  padding: 8px 24px;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.confirm-dialog-footer button.cancel {
  background: var(--border-color);
  color: var(--text-color);
}

.confirm-dialog-footer button.cancel:hover {
  background: var(--hover-color);
}

.confirm-dialog-footer button.confirm {
  background: var(--danger-color);
  color: #fff;
}

.confirm-dialog-footer button.confirm:hover {
  opacity: 0.9;
}

/* 暗黑主题 */
[data-theme="dark"] .modal-content,
[data-theme="dark"] .confirm-dialog-content {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .form-input,
[data-theme="dark"] .form-textarea {
  background: var(--secondary-color) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .form-input:focus,
[data-theme="dark"] .form-textarea:focus {
  border-color: var(--primary-color) !important;
}

[data-theme="dark"] .btn-secondary {
  background: var(--secondary-color) !important;
  color: var(--text-color) !important;
  border-color: var(--border-color) !important;
}

[data-theme="dark"] .btn-secondary:hover {
  background: var(--hover-bg) !important;
}

[data-theme="dark"] .confirm-dialog-body {
  background: var(--secondary-color) !important;
}

/* 下拉动画 */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
