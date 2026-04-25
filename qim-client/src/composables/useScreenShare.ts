import { ref, readonly } from 'vue'

/**
 * 屏幕共享 composable
 * 提供屏幕共享的发起、接受、拒绝、数据传输等功能
 */
export function useScreenShare() {
  // 模态框状态
  const showScreenShareModal = ref(false)

  // 可用的屏幕源列表
  const screenSources = ref<any[]>([])

  // 远程屏幕共享状态
  const remoteScreenSharing = ref(false)
  const remoteScreenUserId = ref<number | null>(null)
  const remoteScreenData = ref<string | null>(null)

  // 屏幕共享组件引用
  const screenShareComponent = ref<any>(null)

  // 当前用户 ID
  const getCurrentUserId = (): number | null => {
    const userStr = localStorage.getItem('currentUser')
    if (userStr) {
      try {
        const user = JSON.parse(userStr)
        return user.id
      } catch {
        return null
      }
    }
    return null
  }

  /**
   * 获取可用的屏幕源
   */
  const getScreenSources = async () => {
    try {
      const sources = await (window as any).electron?.ipcRenderer.invoke('get-screen-sources')
      screenSources.value = sources || []
      return sources
    } catch (error) {
      console.error('获取屏幕源失败:', error)
      return []
    }
  }

  /**
   * 打开屏幕共享模态框
   */
  const openScreenShareModal = async () => {
    await getScreenSources()
    showScreenShareModal.value = true
  }

  /**
   * 关闭屏幕共享模态框
   */
  const closeScreenShareModal = () => {
    showScreenShareModal.value = false
    screenSources.value = []
  }

  /**
   * 开始屏幕共享（选择屏幕源后）
   */
  const startScreenShare = async (conversationId: number, sourceId: string, emit: any) => {
    try {
      // 通知对方开始屏幕共享
      emit('send-screen-share-start', {
        conversationId,
        requester_id: getCurrentUserId(),
        sourceId
      })
      closeScreenShareModal()
    } catch (error) {
      console.error('开始屏幕共享失败:', error)
    }
  }

  /**
   * 发送屏幕共享开始请求（请求对方接受）
   */
  const sendScreenShareStart = (conversationId: number, emit: any) => {
    emit('send-screen-share-start', {
      conversationId,
      requester_id: getCurrentUserId()
    })
  }

  /**
   * 发送屏幕共享停止
   */
  const sendScreenShareStop = (conversationId: number, emit: any) => {
    emit('send-screen-share-stop', {
      conversationId
    })
    remoteScreenSharing.value = false
    remoteScreenUserId.value = null
  }

  /**
   * 发送屏幕共享数据
   */
  const sendScreenShareData = (conversationId: number, data: string, emit: any) => {
    emit('send-screen-share-data', {
      conversationId,
      data
    })
  }

  /**
   * 停止接收屏幕共享
   */
  const stopReceiving = () => {
    if (screenShareComponent.value) {
      screenShareComponent.value.stopReceiving()
    }
    remoteScreenSharing.value = false
    remoteScreenUserId.value = null
    remoteScreenData.value = null
  }

  /**
   * 设置屏幕共享组件引用
   */
  const setScreenShareComponent = (component: any) => {
    screenShareComponent.value = component
  }

  /**
   * 处理远程屏幕共享开始
   */
  const handleRemoteScreenShareStart = (data: { userId: number }) => {
    remoteScreenSharing.value = true
    remoteScreenUserId.value = data.userId
  }

  /**
   * 处理远程屏幕共享停止
   */
  const handleRemoteScreenShareStop = () => {
    remoteScreenSharing.value = false
    remoteScreenUserId.value = null
    remoteScreenData.value = null
  }

  /**
   * 处理远程屏幕共享数据
   */
  const handleRemoteScreenShareData = (data: { data: string }) => {
    remoteScreenData.value = data.data
  }

  return {
    // 状态
    showScreenShareModal: readonly(showScreenShareModal),
    screenSources: readonly(screenSources),
    remoteScreenSharing: readonly(remoteScreenSharing),
    remoteScreenUserId: readonly(remoteScreenUserId),
    remoteScreenData: readonly(remoteScreenData),
    screenShareComponent: readonly(screenShareComponent),

    // 方法
    getScreenSources,
    openScreenShareModal,
    closeScreenShareModal,
    startScreenShare,
    sendScreenShareStart,
    sendScreenShareStop,
    sendScreenShareData,
    stopReceiving,
    setScreenShareComponent,
    handleRemoteScreenShareStart,
    handleRemoteScreenShareStop,
    handleRemoteScreenShareData,
    getCurrentUserId
  }
}
