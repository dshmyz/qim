import { ref } from 'vue'

export function useScreenShare() {
  const showScreenShareModal = ref(false)
  const screenSources = ref<any[]>([])
  const remoteScreenSharing = ref(false)
  const remoteScreenUserId = ref<number | null>(null)
  const remoteScreenData = ref<string | null>(null)
  const screenShareComponent = ref<any>(null)

  const startScreenShare = async (conversationId: number, emit: any) => {
    try {
      const sources = await (window as any).electron?.ipcRenderer.invoke('get-screen-sources')
      screenSources.value = sources || []
      showScreenShareModal.value = true
    } catch (error) {
      console.error('获取屏幕源失败:', error)
    }
  }

  const sendScreenShareStart = (conversationId: number, emit: any) => {
    emit('send-screen-share-start', {
      conversationId,
      requester_id: getCurrentUserId()
    })
  }

  const sendScreenShareStop = (conversationId: number, emit: any) => {
    emit('send-screen-share-stop', {
      conversationId
    })
    remoteScreenSharing.value = false
    remoteScreenUserId.value = null
  }

  const sendScreenShareData = (conversationId: number, data: string, emit: any) => {
    emit('send-screen-share-data', {
      conversationId,
      data
    })
  }

  const stopReceiving = () => {
    if (screenShareComponent.value) {
      screenShareComponent.value.stopReceiving()
    }
    remoteScreenSharing.value = false
    remoteScreenUserId.value = null
    remoteScreenData.value = null
  }

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

  return {
    showScreenShareModal,
    screenSources,
    remoteScreenSharing,
    remoteScreenUserId,
    remoteScreenData,
    screenShareComponent,
    startScreenShare,
    sendScreenShareStart,
    sendScreenShareStop,
    sendScreenShareData,
    stopReceiving
  }
}
