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
