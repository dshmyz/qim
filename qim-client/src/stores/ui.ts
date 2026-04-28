import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { User, Message } from '../types'

export interface MemberContextMenuPosition {
  x: number
  y: number
}

export const useUIStore = defineStore('ui', () => {
  // 用户资料弹窗
  const showUserProfile = ref(false)
  const selectedUser = ref<User | null>(null)
  
  // 已读用户列表
  const showReadUsersModal = ref(false)
  const currentReadUsers = ref<{ read_users: User[]; total_members: number }>({ read_users: [], total_members: 0 })
  
  // 消息右键菜单
  const showMessageContextMenu = ref(false)
  const messageContextMenuPosition = ref({ x: 0, y: 0 })
  const selectedMessage = ref<Message | null>(null)
  
  // 成员右键菜单
  const showMemberContextMenu = ref(false)
  const memberContextMenuPosition = ref<MemberContextMenuPosition>({ x: 0, y: 0 })
  const selectedMember = ref<User | null>(null)
  
  // 消息管理器
  const showMessageManager = ref(false)
  
  // 确认对话框
  const showConfirmDialog = ref(false)
  const confirmDialogTitle = ref('确认操作')
  const confirmDialogMessage = ref('')
  let confirmDialogCallback: (() => void) | null = null
  
  // 截图预览
  const showScreenshotPreview = ref(false)
  const screenshotImageData = ref('')
  
  // 通话
  const showCallModal = ref(false)
  
  // 图片预览
  const showImagePreview = ref(false)
  const previewImageUrl = ref('')
  
  // 分享预览
  const showSharePreview = ref(false)
  const sharePreviewData = ref<any>(null)
  
  // 方法
  function openConfirmDialog(title: string, message: string, callback: () => void) {
    confirmDialogTitle.value = title
    confirmDialogMessage.value = message
    confirmDialogCallback = callback
    showConfirmDialog.value = true
  }
  
  function closeConfirmDialog() {
    showConfirmDialog.value = false
    confirmDialogCallback = null
  }
  
  function handleConfirmAction() {
    if (confirmDialogCallback) {
      confirmDialogCallback()
    }
    closeConfirmDialog()
  }
  
  function resetUserProfile() {
    showUserProfile.value = false
    selectedUser.value = null
  }
  
  function resetMessageContextMenu() {
    showMessageContextMenu.value = false
    selectedMessage.value = null
  }
  
  function resetMemberContextMenu() {
    showMemberContextMenu.value = false
    selectedMember.value = null
  }
  
  return {
    // 状态
    showUserProfile,
    selectedUser,
    showReadUsersModal,
    currentReadUsers,
    showMessageContextMenu,
    messageContextMenuPosition,
    selectedMessage,
    showMemberContextMenu,
    memberContextMenuPosition,
    selectedMember,
    showMessageManager,
    showConfirmDialog,
    confirmDialogTitle,
    confirmDialogMessage,
    showScreenshotPreview,
    screenshotImageData,
    showCallModal,
    showImagePreview,
    previewImageUrl,
    showSharePreview,
    sharePreviewData,
    // 方法
    openConfirmDialog,
    closeConfirmDialog,
    handleConfirmAction,
    resetUserProfile,
    resetMessageContextMenu,
    resetMemberContextMenu,
  }
})
