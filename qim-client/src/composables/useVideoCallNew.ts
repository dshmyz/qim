import { ref, computed, watch } from 'vue'
import { useSession } from './useSession'
import { useSignaling } from './useSignaling'

function getMediaErrorMessage(error: any, callType: 'voice' | 'video'): string {
  const deviceLabel = callType === 'video' ? '摄像头和麦克风' : '麦克风'
  
  if (error.name === 'NotAllowedError' || error.name === 'PermissionDeniedError') {
    return `无法访问${deviceLabel}，请在浏览器权限设置中允许访问`
  }
  if (error.name === 'NotFoundError' || error.name === 'DevicesNotFoundError') {
    return `未检测到${deviceLabel}，请检查设备连接`
  }
  if (error.name === 'NotReadableError' || error.name === 'TrackStartError') {
    return `${deviceLabel}正在被其他应用使用，请先关闭其他应用`
  }
  if (error.name === 'NotSupportedError') {
    return `浏览器不支持${deviceLabel}访问，请更换浏览器`
  }
  if (error.name === 'OverconstrainedError') {
    return `${deviceLabel}不支持所需的分辨率或格式`
  }
  return `无法获取${deviceLabel}权限，请检查设备连接和浏览器设置`
}

function getConnectionErrorMessage(error: any): string {
  if (error.message && error.message.includes('超时')) {
    return '连接超时，对方可能不在线或网络不稳定，请稍后重试'
  }
  if (error.message && error.message.includes('对方拒绝')) {
    return '对方拒绝了通话请求'
  }
  if (error.message && error.message.includes('网络')) {
    return '网络连接失败，请检查网络设置后重试'
  }
  return '建立通话连接失败，请检查网络连接后重试'
}

async function checkDeviceAvailability(callType: 'voice' | 'video'): Promise<void> {
  if (!navigator.mediaDevices || !navigator.mediaDevices.enumerateDevices) {
    return
  }

  try {
    const devices = await navigator.mediaDevices.enumerateDevices()
    const hasAudioInput = devices.some(d => d.kind === 'audioinput')
    const hasVideoInput = devices.some(d => d.kind === 'videoinput')

    if (!hasAudioInput && !hasVideoInput) {
      throw Object.assign(new Error(), { name: 'NotFoundError' })
    }

    if (!hasAudioInput) {
      const label = callType === 'video' ? '摄像头和麦克风' : '麦克风'
      throw Object.assign(new Error(`未检测到${label}，请检查设备连接`), { name: 'NotFoundError' })
    }

    if (callType === 'video' && !hasVideoInput) {
      throw Object.assign(new Error(), { name: 'NotFoundError' })
    }
  } catch (error: any) {
    if (error.name === 'NotFoundError') {
      throw error
    }
  }
}

let videoCallInstance: ReturnType<typeof createVideoCall> | null = null

function createVideoCall() {
  const session = useSession('video-call')
  const signaling = useSignaling()

  const isCameraEnabled = ref(true)
  const isMicrophoneEnabled = ref(true)

  const callType = ref<'voice' | 'video'>('video')
  const callStatus = ref<'idle' | 'calling' | 'connecting' | 'connected'>('idle')
  const isClosing = ref(false)

  const isVideoEnabled = computed(() => {
    return session.localStream.value?.getVideoTracks().some(t => t.enabled) ?? false
  })

  const isAudioEnabled = computed(() => {
    return session.localStream.value?.getAudioTracks().some(t => t.enabled) ?? false
  })

  const pendingOffer = ref<{ signal: RTCSessionDescriptionInit; fromUserId: number } | null>(null)

  const startCall = async (targetUserId: number, type: 'voice' | 'video' = 'video') => {
    console.log('[VideoCall] Starting call with user:', targetUserId, 'type:', type)

    if (callStatus.value !== 'idle') {
      console.warn('[VideoCall] Cannot start call in state:', callStatus.value)
      throw new Error('当前正在通话中')
    }

    callType.value = type
    callStatus.value = 'calling'

    try {
      await checkDeviceAvailability(type)

      signaling.sendCallStart(targetUserId, type)
      console.log('[VideoCall] Sent call.start with type:', type)
      
      await session.start(targetUserId, { needVideo: type !== 'voice' })

      console.log('[VideoCall] Call started successfully')
    } catch (error: any) {
      console.error('[VideoCall] Failed to start call:', error)
      callStatus.value = 'idle'
      
      let friendlyError: Error
      
      if (error.name === 'NotAllowedError' || error.name === 'NotFoundError' || 
          error.name === 'NotReadableError' || error.name === 'NotSupportedError' || 
          error.name === 'OverconstrainedError') {
        const errorMessage = getMediaErrorMessage(error, type)
        friendlyError = new Error(errorMessage)
      } else {
        const errorMessage = getConnectionErrorMessage(error)
        friendlyError = new Error(errorMessage)
      }
      
      ;(friendlyError as any).code = error.name || error.code || 'UnknownError'
      throw friendlyError
    }
  }

  const handleIncomingCall = (signal: RTCSessionDescriptionInit, fromUserId: number, type: 'voice' | 'video' = 'video') => {
    console.log('[VideoCall] Incoming call from user:', fromUserId, 'type:', type, 'current status:', callStatus.value)

    if (callStatus.value === 'idle' || (callStatus.value === 'calling' && !pendingOffer.value)) {
      callType.value = type
      callStatus.value = 'calling'
      pendingOffer.value = { signal, fromUserId }
      console.log('[VideoCall] Set pending offer')
    } else {
      console.warn('[VideoCall] Cannot accept incoming call in state:', callStatus.value, 'pendingOffer:', !!pendingOffer.value)
    }
  }

  const setIncomingCallType = (type: 'voice' | 'video') => {
    console.log('[VideoCall] Setting incoming call type:', type)
    if (callStatus.value === 'idle') {
      callType.value = type
      callStatus.value = 'calling'
    }
  }

  const acceptCall = async () => {
    console.log('[VideoCall] Accepting call')

    if (!pendingOffer.value) {
      console.error('[VideoCall] No pending offer to accept')
      throw new Error('没有来电数据')
    }

    const offer = pendingOffer.value.signal
    const userId = pendingOffer.value.fromUserId

    if (!offer) {
      console.error('[VideoCall] No WebRTC offer in pending data')
      throw new Error('没有 WebRTC offer')
    }

    callStatus.value = 'connecting'

    try {
      await session.join(offer, userId, { needVideo: callType.value !== 'voice' })
      pendingOffer.value = null

      console.log('[VideoCall] Call accepted successfully')
    } catch (error) {
      console.error('[VideoCall] Failed to accept call:', error)
      callStatus.value = 'idle'
      pendingOffer.value = null
      throw error
    }
  }

  const rejectCall = () => {
    console.log('[VideoCall] Rejecting call')

    const fromUserId = pendingOffer.value?.fromUserId
    if (fromUserId) {
      signaling.sendCallAnswer(fromUserId, false)
    }

    callStatus.value = 'idle'
    pendingOffer.value = null
  }

  const endCall = () => {
    console.log('[VideoCall] Ending call')

    isClosing.value = true
    callStatus.value = 'idle'
    pendingOffer.value = null

    setTimeout(() => {
      try {
        signaling.sendCallEnd()
        session.end()
        console.log('[VideoCall] Call ended successfully')
      } catch (error) {
        console.error('[VideoCall] Failed to end call:', error)
      } finally {
        isClosing.value = false
      }
    }, 50)
  }

  const handleRemoteEndCall = () => {
    console.log('[VideoCall] Handling remote end call')
    console.log('[VideoCall] 当前 callStatus:', callStatus.value)
    console.log('[VideoCall] 当前 pendingOffer:', !!pendingOffer.value)

    isClosing.value = true
    callStatus.value = 'idle'
    pendingOffer.value = null
    console.log('[VideoCall] callStatus 设置为 idle')

    setTimeout(() => {
      try {
        session.end()
        console.log('[VideoCall] session.end() 调用完成')
      } catch (error) {
        console.error('[VideoCall] session.end() 出错:', error)
      } finally {
        isClosing.value = false
      }
    }, 50)

    console.log('[VideoCall] Remote call ended, local resources cleaned up')
  }

  const toggleCamera = () => {
    console.log('[VideoCall] Toggling camera')

    if (session.localStream.value) {
      const videoTracks = session.localStream.value.getVideoTracks()
      videoTracks.forEach(track => {
        track.enabled = !track.enabled
      })
      isCameraEnabled.value = videoTracks.some(t => t.enabled)

      console.log('[VideoCall] Camera enabled:', isCameraEnabled.value)
    }
  }

  const toggleMicrophone = () => {
    console.log('[VideoCall] Toggling microphone')

    if (session.localStream.value) {
      const audioTracks = session.localStream.value.getAudioTracks()
      audioTracks.forEach(track => {
        track.enabled = !track.enabled
      })
      isMicrophoneEnabled.value = audioTracks.some(t => t.enabled)

      console.log('[VideoCall] Microphone enabled:', isMicrophoneEnabled.value)
    }
  }

  const toggleMute = () => {
    toggleMicrophone()
  }

  const enableCamera = () => {
    if (session.localStream.value) {
      session.localStream.value.getVideoTracks().forEach(track => {
        track.enabled = true
      })
      isCameraEnabled.value = true
    }
  }

  const disableCamera = () => {
    if (session.localStream.value) {
      session.localStream.value.getVideoTracks().forEach(track => {
        track.enabled = false
      })
      isCameraEnabled.value = false
    }
  }

  const enableMicrophone = () => {
    if (session.localStream.value) {
      session.localStream.value.getAudioTracks().forEach(track => {
        track.enabled = true
      })
      isMicrophoneEnabled.value = true
    }
  }

  const disableMicrophone = () => {
    if (session.localStream.value) {
      session.localStream.value.getAudioTracks().forEach(track => {
        track.enabled = false
      })
      isMicrophoneEnabled.value = false
    }
  }

  watch(session.sessionState, (state) => {
    console.log('[VideoCall] sessionState changed:', state, 'callStatus:', callStatus.value)
    if (state === 'active' && (callStatus.value === 'connecting' || callStatus.value === 'calling')) {
      callStatus.value = 'connected'
      console.log('[VideoCall] callStatus set to connected')
    } else if (state === 'idle' || state === 'ended') {
      if (callStatus.value !== 'idle') {
        callStatus.value = 'idle'
        pendingOffer.value = null
        console.log('[VideoCall] callStatus set to idle')
      }
    }
  }, { immediate: true })

  watch(session.rtc?.state, (newState) => {
    console.log('[VideoCall] RTC connection state changed:', newState, 'callStatus:', callStatus.value)
    if (newState === 'disconnected' && callStatus.value !== 'idle') {
      console.log('[VideoCall] WebRTC connection lost, triggering remote end call')
      handleRemoteEndCall()
    }
  }, { immediate: true })

  const cleanupOnUnload = () => {
    console.log('[VideoCall] Page unload detected, cleaning up media streams')
    if (callStatus.value !== 'idle') {
      endCall()
    }
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('beforeunload', cleanupOnUnload)
    window.addEventListener('visibilitychange', () => {
      if (document.hidden) {
        cleanupOnUnload()
      }
    })
  }

  return {
    ...session,
    // Explicitly expose remoteStream for clarity
    remoteStream: session.remoteStream,
    callType,
    callStatus,
    isCameraEnabled,
    isMicrophoneEnabled,
    isVideoEnabled,
    isAudioEnabled,
    isClosing,
    pendingOffer,
    startCall,
    handleIncomingCall,
    setIncomingCallType,
    acceptCall,
    rejectCall,
    endCall,
    handleRemoteEndCall,
    toggleMute,
    toggleCamera,
    toggleMicrophone,
    enableCamera,
    disableCamera,
    enableMicrophone,
    disableMicrophone
  }
}

export function useVideoCallNew() {
  if (!videoCallInstance) {
    videoCallInstance = createVideoCall()
  }
  return videoCallInstance
}
