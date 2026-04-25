import { ref } from 'vue'

/**
 * UI 状态管理 composable
 * 管理所有 UI 相关的状态：上下文菜单、模态框、对话框、操作菜单等
 */
export function useUI() {
  // 会话右键菜单
  const showMenu = ref(false)
  const menuPosition = ref({ x: 0, y: 0 })
  const selectedConversation = ref<any>(null)

  // 操作菜单
  const showActionMenuFlag = ref(false)
  const actionMenuPosition = ref({ x: 0, y: 0 })

  // 用户右键菜单
  const showUserContextMenuFlag = ref(false)
  const userContextMenuPosition = ref({ x: 0, y: 0 })
  const selectedEmployee = ref<any>(null)

  // 群聊右键菜单
  const showGroupContextMenuFlag = ref(false)
  const groupContextMenuPosition = ref({ x: 0, y: 0 })

  // 成员右键菜单
  const showMemberContextMenuFlag = ref(false)
  const memberContextMenuPosition = ref({ x: 0, y: 0 })
  const selectedMember = ref<any>(null)

  // 设置菜单
  const showSettingsMenuFlag = ref(false)
  const settingsMenuPosition = ref({ x: 0, y: 0 })

  // 主题菜单
  const showThemeMenuFlag = ref(false)
  const themeMenuPosition = ref({ x: 0, y: 0 })

  // 更多菜单
  const showMoreMenuFlag = ref(false)
  const moreMenuPosition = ref({ x: 0, y: 0 })

  // 分享模态框
  const showShareModal = ref(false)
  const shareType = ref('')
  const shareUsers = ref<any[]>([])
  const shareGroups = ref<any[]>([])

  // 用户资料弹窗
  const showUserProfile = ref(false)
  const selectedUser = ref<any>(null)

  // 创建会话模态框
  const showCreateConversationModal = ref(false)
  const createConversationType = ref('group')
  const createConversationTitle = ref('')

  // 系统消息模态框
  const showSystemMessageModal = ref(false)
  const systemMessage = ref({
    title: '',
    content: '',
    target: 'all',
    groupId: '',
    userId: ''
  })

  // 群成员模态框
  const showGroupMembersModal = ref(false)
  const groupMembers = ref<any[]>([])

  // 邀请成员模态框
  const showInviteMembersModal = ref(false)

  // 添加成员模态框
  const showAddMembersModal = ref(false)
  const addMembersSearchQuery = ref('')
  const selectedAddMembers = ref<any[]>([])

  // 编辑群公告模态框
  const showEditAnnouncementModal = ref(false)
  const editAnnouncementContent = ref('')

  // 群资料模态框
  const showGroupInfoModal = ref(false)

  // 关于对话框
  const showAboutDialog = ref(false)

  // 退出登录对话框
  const showLogoutDialog = ref(false)

  // 检查更新对话框
  const showUpdateDialog = ref(false)
  const isCheckingUpdate = ref(false)
  const isDownloading = ref(false)
  const downloadProgress = ref(0)
  const updateResult = ref('')
  const hasNewVersion = ref(false)

  // 语音通话模态框
  const showVoiceCallModal = ref(false)
  const voiceCallStatus = ref('calling')
  const voiceCallDuration = ref(0)

  // 设置模态框
  const showSettingsModal = ref(false)
  const activeSettingsTab = ref('basic')

  // ========== 会话右键菜单操作 ==========

  // 显示会话右键菜单
  const showContextMenu = (event: MouseEvent, conversation: any) => {
    event.preventDefault()
    showMenu.value = true
    menuPosition.value = { x: event.clientX, y: event.clientY }
    selectedConversation.value = conversation
  }

  // 隐藏会话右键菜单
  const hideContextMenu = () => {
    showMenu.value = false
    selectedConversation.value = null
  }

  // ========== 操作菜单操作 ==========

  // 显示操作菜单
  const showActionMenu = (event: MouseEvent) => {
    event.stopPropagation()
    showActionMenuFlag.value = true
    actionMenuPosition.value = { x: event.clientX, y: event.clientY }
  }

  // 隐藏操作菜单
  const hideActionMenu = () => {
    showActionMenuFlag.value = false
  }

  // ========== 用户右键菜单操作 ==========

  // 显示用户右键菜单
  const showUserContextMenu = (event: MouseEvent, user: any) => {
    event.preventDefault()
    showUserContextMenuFlag.value = true
    userContextMenuPosition.value = { x: event.clientX, y: event.clientY }
    selectedEmployee.value = user
  }

  // 隐藏用户右键菜单
  const hideUserContextMenu = () => {
    showUserContextMenuFlag.value = false
    selectedEmployee.value = null
  }

  // ========== 群聊右键菜单操作 ==========

  // 显示群聊右键菜单
  const showGroupContextMenu = (event: MouseEvent, group: any) => {
    event.preventDefault()
    showGroupContextMenuFlag.value = true
    groupContextMenuPosition.value = { x: event.clientX, y: event.clientY }
    // 保持 selectedGroup 不变，由外部设置
  }

  // 隐藏群聊右键菜单
  const closeGroupContextMenu = () => {
    showGroupContextMenuFlag.value = false
  }

  // ========== 成员右键菜单操作 ==========

  // 显示成员右键菜单
  const showMemberContextMenu = (event: MouseEvent, member: any) => {
    event.preventDefault()
    showMemberContextMenuFlag.value = true
    memberContextMenuPosition.value = { x: event.clientX, y: event.clientY }
    selectedMember.value = member
  }

  // 隐藏成员右键菜单
  const hideMemberContextMenu = () => {
    showMemberContextMenuFlag.value = false
    selectedMember.value = null
  }

  // ========== 设置菜单操作 ==========

  // 显示设置菜单
  const showSettingsMenu = (event: MouseEvent) => {
    event.stopPropagation()
    showSettingsMenuFlag.value = true
    settingsMenuPosition.value = { x: event.clientX, y: event.clientY }
  }

  // 隐藏设置菜单
  const hideSettingsMenu = () => {
    showSettingsMenuFlag.value = false
  }

  // ========== 主题菜单操作 ==========

  // 显示主题菜单
  const showThemeMenu = (event: MouseEvent) => {
    event.stopPropagation()
    showThemeMenuFlag.value = true
    themeMenuPosition.value = { x: event.clientX, y: event.clientY }
  }

  // 隐藏主题菜单
  const hideThemeMenu = () => {
    showThemeMenuFlag.value = false
  }

  // ========== 更多菜单操作 ==========

  // 显示更多菜单
  const showMoreMenu = (event: MouseEvent) => {
    event.stopPropagation()
    showMoreMenuFlag.value = true
    moreMenuPosition.value = { x: event.clientX, y: event.clientY }
  }

  // 隐藏更多菜单
  const closeMoreMenu = () => {
    showMoreMenuFlag.value = false
  }

  // ========== 分享模态框操作 ==========

  // 打开分享模态框
  const openShareModal = (type: string, users: any[], groups: any[]) => {
    showShareModal.value = true
    shareType.value = type
    shareUsers.value = users
    shareGroups.value = groups
  }

  // 关闭分享模态框
  const closeShareModal = () => {
    showShareModal.value = false
  }

  // ========== 用户资料弹窗操作 ==========

  // 打开用户资料
  const openUserProfile = (user: any) => {
    showUserProfile.value = true
    selectedUser.value = user
  }

  // 关闭用户资料
  const closeUserProfile = () => {
    showUserProfile.value = false
    selectedUser.value = null
  }

  // ========== 创建会话模态框操作 ==========

  // 打开创建群聊模态框
  const openCreateGroupModal = () => {
    createConversationType.value = 'group'
    createConversationTitle.value = '创建群聊'
    showCreateConversationModal.value = true
    hideActionMenu()
  }

  // 关闭创建会话模态框
  const closeCreateConversationModal = () => {
    showCreateConversationModal.value = false
  }

  // ========== 系统消息模态框操作 ==========

  // 打开系统消息模态框
  const openSystemMessageModal = () => {
    showSystemMessageModal.value = true
    hideActionMenu()
  }

  // 关闭系统消息模态框
  const closeSystemMessageModal = () => {
    showSystemMessageModal.value = false
    systemMessage.value = {
      title: '',
      content: '',
      target: 'all',
      groupId: '',
      userId: ''
    }
  }

  // ========== 群成员模态框操作 ==========

  // 打开群成员模态框
  const openGroupMembersModal = () => {
    showGroupMembersModal.value = true
  }

  // 关闭群成员模态框
  const closeGroupMembersModal = () => {
    showGroupMembersModal.value = false
  }

  // ========== 邀请成员模态框操作 ==========

  // 打开邀请成员模态框
  const openInviteMembersModal = () => {
    showInviteMembersModal.value = true
  }

  // 关闭邀请成员模态框
  const closeInviteMembersModal = () => {
    showInviteMembersModal.value = false
  }

  // ========== 添加成员模态框操作 ==========

  // 打开添加成员模态框
  const openAddMembersModal = () => {
    showAddMembersModal.value = true
    addMembersSearchQuery.value = ''
    selectedAddMembers.value = []
  }

  // 关闭添加成员模态框
  const closeAddMembersModal = () => {
    showAddMembersModal.value = false
  }

  // ========== 编辑群公告模态框操作 ==========

  // 打开编辑群公告模态框
  const openEditAnnouncementModal = () => {
    showEditAnnouncementModal.value = true
  }

  // 关闭编辑群公告模态框
  const closeEditAnnouncementModal = () => {
    showEditAnnouncementModal.value = false
    editAnnouncementContent.value = ''
  }

  // ========== 群资料模态框操作 ==========

  // 打开群资料模态框
  const openGroupInfoModal = () => {
    showGroupInfoModal.value = true
    closeGroupContextMenu()
  }

  // 关闭群资料模态框
  const closeGroupInfoModal = () => {
    showGroupInfoModal.value = false
  }

  // ========== 关于对话框操作 ==========

  // 打开关于对话框
  const openAboutDialog = () => {
    showAboutDialog.value = true
    hideSettingsMenu()
  }

  // 关闭关于对话框
  const closeAboutDialog = () => {
    showAboutDialog.value = false
  }

  // ========== 退出登录对话框操作 ==========

  // 打开退出登录对话框
  const openLogoutDialog = () => {
    showLogoutDialog.value = true
    hideSettingsMenu()
  }

  // 关闭退出登录对话框
  const cancelLogout = () => {
    showLogoutDialog.value = false
  }

  // 确认退出登录
  const confirmLogout = () => {
    showLogoutDialog.value = false
    // 由外部实现具体的退出逻辑
  }

  // ========== 检查更新对话框操作 ==========

  // 打开检查更新对话框
  const openUpdateDialog = () => {
    showUpdateDialog.value = true
    hideSettingsMenu()
  }

  // 关闭检查更新对话框
  const closeUpdateDialog = () => {
    showUpdateDialog.value = false
  }

  // ========== 语音通话操作 ==========

  // 打开语音通话
  const openVoiceCall = () => {
    showVoiceCallModal.value = true
    voiceCallStatus.value = 'calling'
    voiceCallDuration.value = 0
  }

  // 关闭语音通话
  const closeVoiceCall = () => {
    showVoiceCallModal.value = false
  }

  // ========== 设置模态框操作 ==========

  // 打开设置模态框
  const openSettings = () => {
    showSettingsModal.value = true
    activeSettingsTab.value = 'basic'
    hideSettingsMenu()
  }

  // 关闭设置模态框
  const closeSettingsModal = () => {
    showSettingsModal.value = false
  }

  // 切换设置标签页
  const switchSettingsTab = (tab: string) => {
    activeSettingsTab.value = tab
  }

  // 点击外部区域关闭所有菜单
  const handleClickOutside = () => {
    hideContextMenu()
    hideActionMenu()
    hideUserContextMenu()
    closeGroupContextMenu()
    hideMemberContextMenu()
    hideSettingsMenu()
    hideThemeMenu()
    closeMoreMenu()
  }

  return {
    // 状态
    showMenu,
    menuPosition,
    selectedConversation,
    showActionMenuFlag,
    actionMenuPosition,
    showUserContextMenuFlag,
    userContextMenuPosition,
    selectedEmployee,
    showGroupContextMenuFlag,
    groupContextMenuPosition,
    showMemberContextMenuFlag,
    memberContextMenuPosition,
    selectedMember,
    showSettingsMenuFlag,
    settingsMenuPosition,
    showThemeMenuFlag,
    themeMenuPosition,
    showMoreMenuFlag,
    moreMenuPosition,
    showShareModal,
    shareType,
    shareUsers,
    shareGroups,
    showUserProfile,
    selectedUser,
    showCreateConversationModal,
    createConversationType,
    createConversationTitle,
    showSystemMessageModal,
    systemMessage,
    showGroupMembersModal,
    groupMembers,
    showInviteMembersModal,
    showAddMembersModal,
    addMembersSearchQuery,
    selectedAddMembers,
    showEditAnnouncementModal,
    editAnnouncementContent,
    showGroupInfoModal,
    showAboutDialog,
    showLogoutDialog,
    showUpdateDialog,
    isCheckingUpdate,
    isDownloading,
    downloadProgress,
    updateResult,
    hasNewVersion,
    showVoiceCallModal,
    voiceCallStatus,
    voiceCallDuration,
    showSettingsModal,
    activeSettingsTab,

    // 操作方法
    showContextMenu,
    hideContextMenu,
    showActionMenu,
    hideActionMenu,
    showUserContextMenu,
    hideUserContextMenu,
    showGroupContextMenu,
    closeGroupContextMenu,
    showMemberContextMenu,
    hideMemberContextMenu,
    showSettingsMenu,
    hideSettingsMenu,
    showThemeMenu,
    hideThemeMenu,
    showMoreMenu,
    closeMoreMenu,
    openShareModal,
    closeShareModal,
    openUserProfile,
    closeUserProfile,
    openCreateGroupModal,
    closeCreateConversationModal,
    openSystemMessageModal,
    closeSystemMessageModal,
    openGroupMembersModal,
    closeGroupMembersModal,
    openInviteMembersModal,
    closeInviteMembersModal,
    openAddMembersModal,
    closeAddMembersModal,
    openEditAnnouncementModal,
    closeEditAnnouncementModal,
    openGroupInfoModal,
    closeGroupInfoModal,
    openAboutDialog,
    closeAboutDialog,
    openLogoutDialog,
    cancelLogout,
    confirmLogout,
    openUpdateDialog,
    closeUpdateDialog,
    openVoiceCall,
    closeVoiceCall,
    openSettings,
    closeSettingsModal,
    switchSettingsTab,
    handleClickOutside
  }
}
