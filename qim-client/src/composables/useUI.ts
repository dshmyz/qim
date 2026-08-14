import { computed, ref } from 'vue'

export interface UpdateInfo {
  version: string
  releaseDate?: string
  releaseNotes?: string
}

// 全局菜单状态：同一时间只有一个菜单可见
export const activeMenu = ref<string>('')
export const activeMenuPosition = ref({ x: 0, y: 0 })

/**
 * 纯函数：计算上下文菜单位置，防止菜单超出视口边界。
 * 不依赖 DOM，便于单元测试断言。
 *
 * @param clientX       鼠标或触发点的视口 X 坐标
 * @param clientY       鼠标或触发点的视口 Y 坐标
 * @param menuWidth     菜单实际宽度（由调用方通过 getBoundingClientRect 获取）
 * @param menuHeight    菜单实际高度
 * @param viewportWidth 视口宽度（默认 window.innerWidth）
 * @param viewportHeight视口高度（默认 window.innerHeight）
 * @param margin        距离视口边缘的安全间距（默认 4px）
 */
export function computeContextMenuPosition(
  clientX: number,
  clientY: number,
  menuWidth: number,
  menuHeight: number,
  viewportWidth: number = typeof window !== 'undefined' ? window.innerWidth : 1024,
  viewportHeight: number = typeof window !== 'undefined' ? window.innerHeight : 768,
  margin: number = 4,
): { x: number; y: number } {
  let x = clientX
  let y = clientY
  if (x + menuWidth > viewportWidth) x = Math.max(0, viewportWidth - menuWidth - margin)
  if (y + menuHeight > viewportHeight) y = Math.max(0, viewportHeight - menuHeight - margin)
  return { x, y }
}

export function openMenu(id: string, x: number, y: number) {
  activeMenu.value = id
  activeMenuPosition.value = { x, y }
}

export function closeMenu() {
  activeMenu.value = ''
}

const formatDownloadProgress = (percent: unknown): number => {
  const value = typeof percent === 'number' ? percent : Number(percent)
  if (!Number.isFinite(value)) return 0

  const clamped = Math.min(Math.max(value, 0), 100)
  return Math.round(clamped * 10) / 10
}

const formatDownloadBytes = (bytes: number): string => {
  if (bytes >= 1024 * 1024 * 1024) {
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
  }
  if (bytes >= 1024 * 1024) {
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }
  if (bytes >= 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`
  }
  return `${bytes} B`
}

// 渲染层猜测当前平台，用于「安装中」提示给平台感知的文案（deb/AppImage = linux，nsis = win32）
const detectUpdatePlatform = (): string => {
  const ua = navigator.userAgent.toLowerCase()
  if (ua.includes('linux')) return 'linux'
  if (ua.includes('mac')) return 'macos'
  if (ua.includes('win')) return 'win32'
  return ''
}

/**
 * UI 状态管理 composable
 * 管理所有 UI 相关的状态：上下文菜单、模态框、对话框、操作菜单等
 */
export function useUI() {

  /**
   * 计算上下文菜单的位置，防止菜单超出视口边界
   */
  const computeMenuPosition = (clientX: number, clientY: number, menuWidth: number = 160, menuHeight: number = 160) => {
    const windowWidth = window.innerWidth
    const windowHeight = window.innerHeight

    let x = clientX
    let y = clientY

    if (x + menuWidth > windowWidth) {
      x = windowWidth - menuWidth - 10
    }
    if (x < 0) {
      x = 10
    }

    if (y + menuHeight > windowHeight) {
      y = windowHeight - menuHeight - 10
    }
    if (y < 0) {
      y = 10
    }

    return { x, y }
  }

  // 会话右键菜单
  const selectedConversation = ref<any>(null)

  // 用户右键菜单
  const selectedEmployee = ref<any>(null)

  // 群聊右键菜单
  const selectedGroupForContextMenu = ref<any>(null)

  // 成员右键菜单
  const selectedMember = ref<any>(null)

  // 更多菜单
  const showMoreMenuFlag = computed(() => activeMenu.value === 'more')
  const moreMenuPosition = computed(() => activeMenu.value === 'more' ? activeMenuPosition.value : { x: 0, y: 0 })

  // 分享模态框
  const showShareModal = ref(false)
  const shareType = ref('')
  const shareData = ref<any>(null)
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
    targetIds: [] as (string | number)[]
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

  // 编辑群名称模态框
  const showEditGroupNameModal = ref(false)
  const editGroupName = ref('')

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
  const isUpdateReadyToInstall = ref(false)
  const isInstalling = ref(false)
  const updatePlatform = ref('')
  const downloadProgress = ref(0)
  const downloadTransferred = ref(0)
  const downloadTotal = ref(0)
  const downloadSizeText = computed(() => {
    if (downloadTransferred.value <= 0 || downloadTotal.value <= 0) return ''
    return `${formatDownloadBytes(downloadTransferred.value)} / ${formatDownloadBytes(downloadTotal.value)}`
  })
  const updateResult = ref('')
  const hasNewVersion = ref(false)
  const forceUpdate = ref(false)
  // 静默强制更新：自动检查发现的强制版本，自动下载并立即自动安装，界面上无手动按钮
  const silentForce = ref(false)
  const updateInfo = ref<UpdateInfo | null>(null)

  // 设置模态框
  const showSettingsModal = ref(false)
  const activeSettingsTab = ref('basic')

  // ========== 会话右键菜单操作 ==========

  const showContextMenu = (event: MouseEvent, conversation: any) => {
    event.preventDefault()
    const pos = computeMenuPosition(event.clientX, event.clientY, 160, 150)
    selectedConversation.value = conversation
    openMenu('context', pos.x, pos.y)
  }

  const hideContextMenu = () => {
    closeMenu()
    selectedConversation.value = null
  }

  // ========== 操作菜单操作 ==========

  const showActionMenu = (event: MouseEvent) => {
    event.stopPropagation()
    if (activeMenu.value === 'action') {
      closeMenu()
      return
    }
    const pos = computeMenuPosition(event.clientX, event.clientY, 180, 180)
    openMenu('action', pos.x, pos.y)
  }

  const hideActionMenu = () => closeMenu()

  // ========== 用户右键菜单操作 ==========

  const showUserContextMenu = (event: MouseEvent, user: any) => {
    event.preventDefault()
    const pos = computeMenuPosition(event.clientX, event.clientY, 140, 80)
    selectedEmployee.value = user
    openMenu('user', pos.x, pos.y)
  }

  const hideUserContextMenu = () => {
    closeMenu()
    selectedEmployee.value = null
  }

  // ========== 群聊右键菜单操作 ==========

  const showGroupContextMenu = (event: MouseEvent, group: any) => {
    event.preventDefault()
    const pos = computeMenuPosition(event.clientX, event.clientY, 160, 200)
    selectedGroupForContextMenu.value = group
    openMenu('group', pos.x, pos.y)
  }

  const closeGroupContextMenu = () => {
    closeMenu()
    selectedGroupForContextMenu.value = null
  }

  // ========== 成员右键菜单操作 ==========

  // 聊天区成员右键菜单
  const showMemberContextMenu = (event: MouseEvent, member: any) => {
    event.preventDefault()
    const pos = computeMenuPosition(event.clientX, event.clientY, 160, 110)
    selectedMember.value = member
    openMenu('member', pos.x, pos.y)
  }

  const hideMemberContextMenu = () => {
    closeMenu()
    selectedMember.value = null
  }

  // 侧边栏成员右键菜单
  const showSidebarMemberContextMenu = (event: MouseEvent, member: any) => {
    event.preventDefault()
    const pos = computeMenuPosition(event.clientX, event.clientY, 160, 110)
    selectedMember.value = member
    openMenu('sidebar-member', pos.x, pos.y)
  }

  // ========== 设置菜单操作 ==========

  // 显示设置菜单
  const showSettingsMenu = (event: MouseEvent) => {
    event.stopPropagation()
    if (activeMenu.value === 'settings') {
      closeMenu()
      return
    }
    const settingsButton = event.currentTarget as HTMLElement
    if (settingsButton) {
      const rect = settingsButton.getBoundingClientRect()
      let x = rect.right + 2
      let y = event.clientY - 200
      if (x + 180 > window.innerWidth) x = rect.left - 190
      if (y + 200 > window.innerHeight) y = window.innerHeight - 210
      if (y < 0) y = 10
      openMenu('settings', x, y)
    }
  }

  // 隐藏设置菜单
  const hideSettingsMenu = () => closeMenu()

  // ========== 主题菜单操作 ==========

  // 显示主题菜单
  const showThemeMenu = (event: MouseEvent) => {
    event.stopPropagation()
    if (activeMenu.value === 'theme') {
      closeMenu()
      return
    }
    const themeButton = event.currentTarget as HTMLElement
    if (themeButton) {
      const rect = themeButton.getBoundingClientRect()
      let x = rect.right + 2
      let y = rect.top
      if (x + 180 > window.innerWidth) x = rect.left - 190
      if (y + 400 > window.innerHeight - 10) y = window.innerHeight - 410
      if (y < 10) y = 10
      openMenu('theme', x, y)
    }
  }

  const hideThemeMenu = () => closeMenu()

  // ========== 更多菜单操作 ==========

  // 显示更多菜单
  const showMoreMenu = (event: MouseEvent) => {
    event.stopPropagation()
    const pos = computeMenuPosition(event.clientX, event.clientY, 160, 50)
    openMenu('more', pos.x, pos.y)
  }

  // 隐藏更多菜单
  const closeMoreMenu = () => closeMenu()

  // ========== 分享模态框操作 ==========

  // 打开分享模态框
  const openShareModal = (type: string, data: any, options?: { users?: any[]; groups?: any[] }) => {
    showShareModal.value = true
    shareType.value = type
    shareData.value = data
    if (options?.users !== undefined) {
      shareUsers.value = options.users
    }
    if (options?.groups !== undefined) {
      shareGroups.value = options.groups
    }
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

  // 关闭用户资料（不清理 selectedUser，避免组织架构面板连带消失）
  const closeUserProfile = () => {
    showUserProfile.value = false
  }

  // ========== 创建会话模态框操作 ==========

  // 打开创建群聊模态框
  const openCreateGroupModal = (type: string = 'group') => {
    createConversationType.value = type
    createConversationTitle.value = type === 'discussion' ? '创建讨论组' : '创建群聊'
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
      targetIds: []
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

  // ========== 编辑群名称模态框操作 ==========

  // 打开编辑群名称模态框
  const openEditGroupNameModal = (groupName: string = '') => {
    editGroupName.value = groupName
    showEditGroupNameModal.value = true
  }

  // 关闭编辑群名称模态框
  const closeEditGroupNameModal = () => {
    showEditGroupNameModal.value = false
    editGroupName.value = ''
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

  // 更新事件监听器注册（Electron IPC）
  const updateEventListeners: Array<{ channel: string; handler: (...args: any[]) => void }> = []

  const registerUpdateEventListeners = () => {
    if (!window.electron) return

    unregisterUpdateEventListeners()

    const normalizeReleaseNotes = (releaseNotes: any): string => {
      if (Array.isArray(releaseNotes)) {
        return releaseNotes
          .map((item) => typeof item === 'string' ? item : item?.note || item?.value || '')
          .filter(Boolean)
          .join('\n')
      }
      return typeof releaseNotes === 'string' ? releaseNotes : ''
    }

    const handlers = [
      {
        channel: 'update-checking',
        handler: () => {
          isCheckingUpdate.value = true
          isDownloading.value = false
          isUpdateReadyToInstall.value = false
          isInstalling.value = false
          downloadProgress.value = 0
          downloadTransferred.value = 0
          downloadTotal.value = 0
          updateInfo.value = null
          updateResult.value = '正在检查更新...'
        }
      },
      {
        channel: 'update-available',
        handler: (_event: any, info: any) => {
          isCheckingUpdate.value = false
          isDownloading.value = false
          isUpdateReadyToInstall.value = false
          isInstalling.value = false
          downloadProgress.value = 0
          downloadTransferred.value = 0
          downloadTotal.value = 0
          hasNewVersion.value = true
          forceUpdate.value = !!info.forceUpdate
          silentForce.value = !!info.silent
          updateInfo.value = {
            version: info.version || '',
            releaseDate: info.releaseDate || info.release_date || '',
            releaseNotes: normalizeReleaseNotes(info.releaseNotes || info.release_notes || info.changelog)
          }
          updateResult.value = info.forceUpdate
            ? (info.silent
                ? `发现新版本 v${info.version}，系统将自动升级，请在弹窗提示后尽快保存工作`
                : `发现新版本 v${info.version}（需要强制更新）`)
            : `发现新版本 v${info.version}`
          // 自动检查发现新版本时，弹出更新提示对话框
          showUpdateDialog.value = true
        }
      },
      {
        channel: 'update-not-available',
        handler: () => {
          isCheckingUpdate.value = false
          isDownloading.value = false
          isUpdateReadyToInstall.value = false
          isInstalling.value = false
          downloadProgress.value = 0
          downloadTransferred.value = 0
          downloadTotal.value = 0
          hasNewVersion.value = false
          forceUpdate.value = false
          silentForce.value = false
          updateInfo.value = null
          updateResult.value = '当前已是最新版本'
          // 自动检查无新版本时，不弹窗（仅在用户手动检查时对话框已打开）
        }
      },
      {
        channel: 'update-error',
        handler: (_event: any, error: any) => {
          isCheckingUpdate.value = false
          isDownloading.value = false
          isUpdateReadyToInstall.value = false
          isInstalling.value = false
          downloadProgress.value = 0
          downloadTransferred.value = 0
          downloadTotal.value = 0

          // 解析错误信息，显示友好提示
          let friendlyMessage = '检查更新失败'

          if (typeof error === 'string') {
            friendlyMessage = error
            // 处理 electron-updater 的错误信息
            if (error.includes('404') || error.includes('Cannot find channel')) {
              friendlyMessage = '暂无可用更新'
            } else if (error.includes('timeout') || error.includes('ETIMEDOUT')) {
              friendlyMessage = '网络连接超时，请稍后重试'
            } else if (error.includes('ENOTFOUND') || error.includes('ECONNREFUSED')) {
              friendlyMessage = '无法连接到更新服务器'
            } else if (error.includes('net::ERR')) {
              friendlyMessage = '网络错误，请检查网络连接'
            }
          } else if (error?.message) {
            // 处理 Error 对象
            if (error.message.includes('404')) {
              friendlyMessage = '暂无可用更新'
            } else {
              friendlyMessage = error.message
            }
          }

          updateResult.value = friendlyMessage
          console.error('更新错误:', error)
        }
      },
      {
        channel: 'update-progress',
        handler: (_event: any, progressObj: any) => {
          isDownloading.value = true
          isUpdateReadyToInstall.value = false
          isInstalling.value = false
          downloadProgress.value = formatDownloadProgress(progressObj?.percent)
          downloadTransferred.value = Math.max(Number(progressObj?.transferred) || 0, 0)
          downloadTotal.value = Math.max(Number(progressObj?.total) || 0, 0)
        }
      },
      {
        channel: 'update-downloaded',
        handler: (_event: any, info: any) => {
          isDownloading.value = false
          isUpdateReadyToInstall.value = false
          isInstalling.value = false
          downloadProgress.value = 100
          downloadTransferred.value = downloadTotal.value
          if (info?.silent) {
            // 静默强制更新：下载完成后不等待用户点击，排队到夜间自动重启安装
            updateResult.value = '强制更新已下载完成，正在重新启动应用完成升级，请保存工作。'
            isUpdateReadyToInstall.value = false
          } else {
            updateResult.value = '更新已下载完成，是否立即重启安装？'
            isUpdateReadyToInstall.value = true
          }
        }
      },
      {
        channel: 'update-installing',
        handler: () => {
          isCheckingUpdate.value = false
          isDownloading.value = false
          isUpdateReadyToInstall.value = false
          isInstalling.value = true
          downloadProgress.value = 100
          downloadTransferred.value = downloadTotal.value
          updateResult.value = '正在重启并安装更新...'
          // 主进程安装更早退出，这里记下平台，供渲染层给平台感知的安装提示文案
          updatePlatform.value = detectUpdatePlatform()
        }
      }
    ]

    handlers.forEach(({ channel, handler }) => {
      window.electron.ipcRenderer.on(channel, handler)
      updateEventListeners.push({ channel, handler })
    })
  }

  const unregisterUpdateEventListeners = () => {
    if (!window.electron) return

    updateEventListeners.forEach(({ channel, handler }) => {
      window.electron.ipcRenderer.removeListener(channel, handler)
    })
    updateEventListeners.length = 0
  }

  // ========== 语音通话操作 ==========

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
    closeMenu()
    selectedConversation.value = null
    selectedEmployee.value = null
    selectedGroupForContextMenu.value = null
    selectedMember.value = null
  }

  return {
    // 右键菜单数据
    selectedConversation,
    selectedEmployee,
    selectedGroupForContextMenu,
    selectedMember,
    // 右键菜单操作
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
    showSidebarMemberContextMenu,
    showSettingsMenu,
    hideSettingsMenu,
    showThemeMenu,
    hideThemeMenu,
    showMoreMenu,
    closeMoreMenu,
    showShareModal,
    shareType,
    shareData,
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
    showEditGroupNameModal,
    editGroupName,
    showGroupInfoModal,
    showAboutDialog,
    showLogoutDialog,
    showUpdateDialog,
    isCheckingUpdate,
    isDownloading,
    isUpdateReadyToInstall,
    isInstalling,
    updatePlatform,
    downloadProgress,
    downloadTransferred,
    downloadTotal,
    downloadSizeText,
    updateResult,
    hasNewVersion,
    forceUpdate,
    silentForce,
    updateInfo,
    showSettingsModal,
    activeSettingsTab,

    // 操作方法
    computeMenuPosition,
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
    openEditGroupNameModal,
    closeEditGroupNameModal,
    openGroupInfoModal,
    closeGroupInfoModal,
    openAboutDialog,
    closeAboutDialog,
    openLogoutDialog,
    cancelLogout,
    confirmLogout,
    openUpdateDialog,
    closeUpdateDialog,
    registerUpdateEventListeners,
    unregisterUpdateEventListeners,
    openSettings,
    closeSettingsModal,
    switchSettingsTab,
    handleClickOutside
  }
}
