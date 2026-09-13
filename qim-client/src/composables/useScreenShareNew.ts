import { ref } from 'vue'
import { useSession } from './useSession'
import { useSignaling } from './useSignaling'
import type { ScreenShareMode } from '@/types/realtime'

function getScreenShareErrorMessage(error: any): string {
  if (error.name === 'NotAllowedError' || error.name === 'PermissionDeniedError') {
    return '屏幕共享权限被拒绝，请在浏览器权限设置中允许访问'
  }
  if (error.name === 'NotFoundError' || error.name === 'DevicesNotFoundError') {
    return '未检测到可用的屏幕共享源'
  }
  if (error.name === 'NotSupportedError') {
    return '浏览器不支持屏幕共享功能，请更换浏览器'
  }
  if (error.message && error.message.includes('超时')) {
    return '连接超时，对方可能不在线或网络不稳定，请稍后重试'
  }
  if (error.message && error.message.includes('对方拒绝') || error.message && error.message.includes('拒绝')) {
    return '对方拒绝了屏幕共享请求'
  }
  if (error.message && error.message.includes('网络')) {
    return '网络连接失败，请检查网络设置后重试'
  }
  return '建立屏幕共享连接失败，请检查网络连接后重试'
}

let instance: ReturnType<typeof createScreenShare> | null = null

function createScreenShare() {
  const session = useSession('screen-share')
  const signaling = useSignaling()
  
  const selectedSource = ref<DisplayMediaStreamOptions | null>(null)
  const isPaused = ref(false)

  // 本次共享/查看的方向：broadcast=我共享给目标；view=我申请查看目标。
  // 决定"谁采集 + 谁发 WebRTC offer"：broadcast 由请求者采集/发 offer，
  // view 由被查看方（同意方）采集/发 offer，请求者只收流。
  const shareMode = ref<ScreenShareMode>('broadcast')

  const setShareMode = (mode: ScreenShareMode) => {
    shareMode.value = mode
  }
  
  const selectSource = async () => {
    console.log('[ScreenShare] Selecting source')

    try {
      if (window.electron && window.electron.ipcRenderer) {
        console.log('[ScreenShare] Using Electron desktopCapturer')

        window.electron.ipcRenderer.send('start-screen-share')

        const sources = await new Promise<any[]>((resolve, reject) => {
          window.electron.ipcRenderer.once('screen-sources', (_event: any, sources: any[]) => {
            resolve(sources)
          })

          setTimeout(() => reject(new Error('获取屏幕源超时')), 10000)
        })

        console.log('[ScreenShare] Received sources:', sources.length)

        if (sources.length === 0) {
          throw new Error('没有可用的屏幕源')
        }

        const source = sources[0]
        console.log('[ScreenShare] Selected source:', source.name, source.id)

        const stream = await navigator.mediaDevices.getUserMedia({
          audio: false,
          video: {
            mandatory: {
              chromeMediaSource: 'desktop',
              chromeMediaSourceId: source.id
            }
          } as any
        })

        console.log('[ScreenShare] Stream obtained:', stream.id)
        selectedSource.value = { video: true }
        return stream
      }

      console.log('[ScreenShare] Using getDisplayMedia (non-Electron)')
      const stream = await navigator.mediaDevices.getDisplayMedia({
        video: true,
        audio: false
      })

      const videoTrack = stream.getVideoTracks()[0]
      if (videoTrack) {
        selectedSource.value = {
          video: {
            displaySurface: (videoTrack.getSettings() as any).displaySurface
          }
        }
      }

      console.log('[ScreenShare] Source selected:', selectedSource.value)
      return stream
    } catch (error) {
      console.error('[ScreenShare] Failed to select source:', error)
      throw error
    }
  }
  
  const pendingStream = ref<MediaStream | null>(null)
  const pendingTargetUserId = ref<number | null>(null)
  const pendingConversationId = ref<number | null>(null)

  const sendRequest = (targetUserId: number, conversationId: number, stream: MediaStream, mode: ScreenShareMode = 'broadcast') => {
    console.log('[ScreenShare] Sending request to user:', targetUserId, 'conversation:', conversationId, 'mode:', mode)

    shareMode.value = mode
    pendingStream.value = stream
    pendingTargetUserId.value = targetUserId
    pendingConversationId.value = conversationId

    signaling.sendScreenShareRequest(targetUserId, conversationId, mode)
  }

  // 申请查看对方屏幕：view 模式下我不采集（不提供 stream），只发申请，
  // 由被查看方采集并向我推送。pendingStream/pendingTarget 仅记录上下文，供停止清理用。
  const sendViewRequest = (targetUserId: number, conversationId: number) => {
    console.log('[ScreenShare] Sending view request to user:', targetUserId, 'conversation:', conversationId)

    shareMode.value = 'view'
    pendingStream.value = null
    pendingTargetUserId.value = targetUserId
    pendingConversationId.value = conversationId

    signaling.sendScreenShareRequest(targetUserId, conversationId, 'view')
  }

  const startConnection = async () => {
    if (!pendingStream.value || !pendingTargetUserId.value || !pendingConversationId.value) {
      console.error('[ScreenShare] No pending share data to start connection')
      throw new Error('没有待建立的共享连接')
    }

    const stream = pendingStream.value
    const targetUserId = pendingTargetUserId.value

    console.log('[ScreenShare] Starting connection after acceptance, user:', targetUserId)

    try {
      await session.start(targetUserId, { stream })

      pendingStream.value = null
      pendingTargetUserId.value = null
      pendingConversationId.value = null

      console.log('[ScreenShare] Connection started successfully')
    } catch (error: any) {
      console.error('[ScreenShare] Failed to start connection:', error)
      pendingStream.value = null
      pendingTargetUserId.value = null
      pendingConversationId.value = null
      
      const errorMessage = getScreenShareErrorMessage(error)
      const friendlyError = new Error(errorMessage)
      ;(friendlyError as any).code = error.name || error.code || 'UnknownError'
      throw friendlyError
    }
  }

  const startConnectionWithStream = async (targetUserId: number, stream: MediaStream) => {
    console.log('[ScreenShare] Starting connection with stream, user:', targetUserId)

    try {
      await session.start(targetUserId, { stream })

      console.log('[ScreenShare] Connection started successfully')
    } catch (error) {
      console.error('[ScreenShare] Failed to start connection with stream:', error)
      throw error
    }
  }

  const cancelPendingRequest = () => {
    console.log('[ScreenShare] Cancelling pending request')
    if (pendingStream.value) {
      pendingStream.value.getTracks().forEach(t => t.stop())
    }
    pendingStream.value = null
    pendingTargetUserId.value = null
    pendingConversationId.value = null
    shareMode.value = 'broadcast'
  }

  const startSharing = async (targetUserId: number, conversationId: number) => {
    console.log('[ScreenShare] Starting share with user:', targetUserId)

    try {
      signaling.sendScreenShareRequest(targetUserId, conversationId)

      await session.start(targetUserId)

      console.log('[ScreenShare] Share started successfully')
    } catch (error: any) {
      console.error('[ScreenShare] Failed to start share:', error)
      const errorMessage = getScreenShareErrorMessage(error)
      throw new Error(errorMessage)
    }
  }

  const startSharingWithStream = async (targetUserId: number, conversationId: number, stream: MediaStream) => {
    console.log('[ScreenShare] Starting share with external stream, user:', targetUserId)

    try {
      signaling.sendScreenShareRequest(targetUserId, conversationId)

      await session.start(targetUserId, { stream })

      console.log('[ScreenShare] Share started successfully with external stream')
    } catch (error: any) {
      console.error('[ScreenShare] Failed to start share with stream:', error)
      const errorMessage = getScreenShareErrorMessage(error)
      throw new Error(errorMessage)
    }
  }
  
  const acceptRequest = (fromUserId: number, conversationId: number, mode: ScreenShareMode = 'broadcast') => {
    console.log('[ScreenShare] Accepting request (sending response only) from user:', fromUserId, 'mode:', mode)
    shareMode.value = mode
    signaling.sendScreenShareResponse(conversationId, fromUserId, true, mode)
  }

  const rejectRequest = (fromUserId: number, conversationId: number, mode: ScreenShareMode = 'broadcast') => {
    console.log('[ScreenShare] Rejecting request from user:', fromUserId, 'mode:', mode)
    shareMode.value = mode
    signaling.sendScreenShareResponse(conversationId, fromUserId, false, mode)
  }

  const acceptShare = async (signal: RTCSessionDescriptionInit, fromUserId: number, conversationId?: number) => {
    console.log('[ScreenShare] Accepting share from user:', fromUserId)

    try {
      if (conversationId) {
        signaling.sendScreenShareResponse(conversationId, fromUserId, true)
      }

      console.log('[ScreenShare] Calling session.join...')
      await session.join(signal, fromUserId)
      console.log('[ScreenShare] session.join completed')

      console.log('[ScreenShare] Share accepted successfully')
    } catch (error) {
      console.error('[ScreenShare] Failed to accept share:', error)
      if (conversationId) {
        signaling.sendScreenShareResponse(conversationId, fromUserId, false)
      }
      throw error
    }
  }

  const rejectShare = (fromUserId: number, conversationId?: number) => {
    console.log('[ScreenShare] Rejecting share from user:', fromUserId)
    if (conversationId) {
      signaling.sendScreenShareResponse(conversationId, fromUserId, false)
    }
  }
  
  const stopSharing = () => {
    console.log('[ScreenShare] Stopping share')

    signaling.sendScreenShareStop()
    session.end()
    selectedSource.value = null
    isPaused.value = false
    shareMode.value = 'broadcast'

    console.log('[ScreenShare] Share stopped successfully')
  }
  
  const pause = () => {
    console.log('[ScreenShare] Pausing share')
    
    if (session.localStream.value) {
      const videoTrack = session.localStream.value.getVideoTracks()[0]
      if (videoTrack) {
        videoTrack.enabled = false
        isPaused.value = true
        console.log('[ScreenShare] Share paused successfully')
      }
    }
  }
  
  const resume = () => {
    console.log('[ScreenShare] Resuming share')
    
    if (session.localStream.value) {
      const videoTrack = session.localStream.value.getVideoTracks()[0]
      if (videoTrack) {
        videoTrack.enabled = true
        isPaused.value = false
        console.log('[ScreenShare] Share resumed successfully')
      }
    }
  }
  
  const togglePause = () => {
    if (isPaused.value) {
      resume()
    } else {
      pause()
    }
  }
  
  return {
    ...session,
    selectedSource,
    isPaused,
    shareMode,
    setShareMode,
    pendingStream,
    pendingTargetUserId,
    pendingConversationId,
    selectSource,
    sendRequest,
    sendViewRequest,
    startConnection,
    startConnectionWithStream,
    cancelPendingRequest,
    startSharing,
    startSharingWithStream,
    acceptRequest,
    rejectRequest,
    acceptShare,
    rejectShare,
    stopSharing,
    pause,
    resume,
    togglePause
  }
}

export function useScreenShareNew() {
  if (!instance) {
    instance = createScreenShare()
  }
  return instance
}
