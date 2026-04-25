<template>
  <div class="header-actions">
    <span v-if="isGroupOrDiscussion" class="header-icon" title="邀请成员" @click="handleInviteMembers"><i class="fas fa-user-plus"></i></span>
    <span class="header-icon" @click="handleToggleHeaderMenu">
      <i class="fas fa-ellipsis-v"></i>
      <!-- 头部下拉菜单 -->
      <div v-if="showHeaderMenu" class="header-menu" @click.stop>
        <div v-if="isGroupOrDiscussion" class="menu-item" @click="handleEditGroupInfo">
          <i class="fas fa-edit"></i> 修改群名称
        </div>
        <div v-if="isGroupOrDiscussion" class="menu-item" @click="handleEditGroupAnnouncement">
          <i class="fas fa-bullhorn"></i> 编辑群公告
        </div>
        <div v-if="isGroupOrDiscussion && isOwner" class="menu-item" @click="handleConfirmDeleteGroup">
          <i class="fas fa-trash"></i> 解散群聊
        </div>
      </div>
    </span>
  </div>

  <!-- 确认对话框 -->
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

  <!-- 编辑群信息模态框 -->
  <div v-if="showEditGroupInfoModal" class="modal-overlay" @click="handleCloseEditGroupInfoModal">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <h3>修改群名称</h3>
        <button class="close-btn" @click="handleCloseEditGroupInfoModal">&times;</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>群名称</label>
          <input type="text" :value="editGroupName" @input="handleEditGroupNameInput" class="form-input" placeholder="请输入新的群名称" />
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="handleCloseEditGroupInfoModal">取消</button>
        <button class="btn btn-primary" @click="handleSaveGroupInfo">保存</button>
      </div>
    </div>
  </div>

  <!-- 编辑群公告模态框 -->
  <div v-if="showEditAnnouncementModal" class="modal-overlay" @click="handleCloseEditAnnouncementModal">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <h3>编辑群公告</h3>
        <button class="close-btn" @click="handleCloseEditAnnouncementModal">&times;</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>群公告内容</label>
          <textarea :value="editAnnouncement" @input="handleEditAnnouncementInput" class="form-textarea" placeholder="输入群公告内容..." rows="5"></textarea>
          <p class="form-tip">群公告将对所有群成员可见</p>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="handleCloseEditAnnouncementModal">取消</button>
        <button class="btn btn-primary" @click="handleSaveAnnouncement">保存</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import type { Conversation } from '../../types'
import { getCurrentUser } from '../../utils/user'

// 扩展 Conversation 类型以支持 announcement 字段和 members 中的 role 字段
interface GroupMember {
  id: string
  name?: string
  role?: string
  avatar?: string
  [key: string]: unknown
}

interface GroupConversation extends Conversation {
  announcement?: string
  members?: GroupMember[]
}

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
}>()

// 本地确认对话框状态
const localShowConfirmDialog = ref(false)
const localConfirmDialogTitle = ref('确认操作')
const localConfirmDialogMessage = ref('')
const localConfirmDialogCallback = ref<(() => void) | null>(null)

// Computed properties
const isGroupOrDiscussion = computed(() => {
  return props.conversation?.type === 'group' || props.conversation?.type === 'discussion'
})

const isOwner = computed(() => {
  return isGroupOwner(props.conversation)
})

// 计算当前用户在群中的角色
const currentUserRole = computed(() => {
  let currentUser = props.currentUser
  if (!currentUser) {
    currentUser = getCurrentUser()
  }
  if (!props.conversation?.members || !currentUser) return 'member'
  const member = props.conversation.members.find((m) => String(m.id) === String(currentUser.id))
  return member?.role || 'member'
})

// 检查是否有权限修改群名称
const canEditGroupName = computed(() => {
  if (!props.conversation) return false

  // 讨论组全员可修改
  if (props.conversation.type === 'discussion') {
    return true
  }

  // 群只有管理员和群主能修改
  if (props.conversation.type === 'group') {
    const userRole = currentUserRole.value
    return userRole === 'owner' || userRole === 'admin'
  }

  return false
})

// 检查当前用户是否是群主
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
  // 点击其他地方关闭菜单
  if (newValue) {
    setTimeout(() => {
      document.addEventListener('click', closeHeaderMenu)
    }, 0)
  }
}

// 关闭头部下拉菜单
function closeHeaderMenu() {
  emit('update:showHeaderMenu', false)
  document.removeEventListener('click', closeHeaderMenu)
}

// 邀请成员
function handleInviteMembers() {
  emit('invite-members')
  closeHeaderMenu()
}

// 编辑群信息
function handleEditGroupInfo() {
  if (props.conversation && canEditGroupName.value) {
    emit('update:editGroupName', props.conversation.name || '')
    emit('update:showEditGroupInfoModal', true)
  } else if (props.conversation && !canEditGroupName.value) {
    ElMessage.warning('只有管理员和群主可以修改群名称')
  }
  closeHeaderMenu()
}

function handleCloseEditGroupInfoModal() {
  emit('update:showEditGroupInfoModal', false)
}

// 保存群信息
function handleSaveGroupInfo() {
  emit('save-group-info', props.editGroupName)
  emit('update:showEditGroupInfoModal', false)
}

// 编辑群公告
function handleEditGroupAnnouncement() {
  if (props.conversation) {
    emit('update:editAnnouncement', (props.conversation as GroupConversation).announcement || '')
    emit('update:showEditAnnouncementModal', true)
  }
  closeHeaderMenu()
}

function handleCloseEditAnnouncementModal() {
  emit('update:showEditAnnouncementModal', false)
}

// 保存群公告
function handleSaveAnnouncement() {
  emit('save-group-announcement', props.editAnnouncement)
  emit('update:showEditAnnouncementModal', false)
}

// 输入处理
function handleEditGroupNameInput(event: Event) {
  const target = event.target as HTMLInputElement
  emit('update:editGroupName', target.value)
}

function handleEditAnnouncementInput(event: Event) {
  const target = event.target as HTMLTextAreaElement
  emit('update:editAnnouncement', target.value)
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

// 清理全局事件监听
onUnmounted(() => {
  document.removeEventListener('click', closeHeaderMenu)
})
</script>

<style scoped>
/* 头部操作按钮 */
.header-actions {
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
.header-menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 8px;
  background: var(--sidebar-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 1000;
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

.modal-body {
  padding: 20px;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  gap: 12px;
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
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  background: var(--content-bg);
  color: var(--text-color);
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.form-textarea {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  background: var(--content-bg);
  color: var(--text-color);
  resize: vertical;
  box-sizing: border-box;
}

.form-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
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
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
}

.btn-primary {
  background-color: var(--primary-color);
  color: white;
}

.btn-primary:hover {
  background-color: var(--primary-dark);
}

.btn-secondary {
  background-color: var(--content-bg);
  color: var(--text-color);
  border-color: var(--border-color);
}

.btn-secondary:hover {
  background-color: var(--hover-bg);
}

/* 暗黑主题下的模态框样式 */
[data-theme="dark"] .modal-content {
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
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2) !important;
}

[data-theme="dark"] .btn-secondary {
  background-color: var(--secondary-color) !important;
  color: var(--text-color) !important;
  border-color: var(--border-color) !important;
}

[data-theme="dark"] .btn-secondary:hover {
  background-color: var(--hover-bg) !important;
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
  z-index: 1000;
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
}

.confirm-dialog-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
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

.confirm-dialog-footer button.confirm {
  background: var(--primary-color);
  color: #fff;
}

.confirm-dialog-footer button:hover {
  opacity: 0.9;
}

.confirm-dialog-footer button.cancel:hover {
  background: var(--hover-color);
}

.confirm-dialog-footer button.confirm:hover {
  background: var(--primary-color);
  opacity: 0.9;
}

/* 暗黑主题下的确认对话框样式 */
[data-theme="dark"] .confirm-dialog-content {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .confirm-dialog-header {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .confirm-dialog-header h3 {
  color: var(--text-color) !important;
}

[data-theme="dark"] .confirm-dialog-body {
  background: var(--secondary-color) !important;
}

[data-theme="dark"] .confirm-dialog-body p {
  color: var(--text-color) !important;
}

[data-theme="dark"] .confirm-dialog-footer {
  background: var(--sidebar-bg) !important;
  border-top: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .confirm-dialog-footer button.cancel {
  background: var(--border-color) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .confirm-dialog-footer button.confirm {
  background: var(--primary-color) !important;
  color: white !important;
  border: 1px solid var(--primary-color) !important;
}
</style>
