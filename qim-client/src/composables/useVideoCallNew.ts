import { ref, computed, watch } from 'vue'
import { useSession } from './useSession'
import { useSignaling } from './useSignaling'

let videoCallInstance: ReturnType<typeof createVideoCall> | null = null

function createVideoCall() {
  const session = useSession('video-call')
  const signaling = useSignaling()

  const isCameraEnabled = ref(true)
  const isMicrophoneEnabled = ref(true)

  const callType = ref<'voice' | 'video'>('video')
  const callStatus = ref<'idle' | 'calling' | 'connecting' | 'connected'>('idle')

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
      signaling.sendCallStart(targetUserId, type)
      console.log('[VideoCall] Sent call_invite with type:', type)
      
      await session.start(targetUserId, { needVideo: type !== 'voice' })

      console.log('[VideoCall] Call started successfully')
    } catch (error) {
      console.error('[VideoCall] Failed to start call:', error)
      callStatus.value = 'idle'
      throw error
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

    try {
      signaling.sendCallEnd()
      session.end()
    } finally {
      callStatus.value = 'idle'
      pendingOffer.value = null
    }

    console.log('[VideoCall] Call ended successfully')
  }

  const handleRemoteEndCall = () => {
    console.log('[VideoCall] Handling remote end call')
    console.log('[VideoCall] 当前 callStatus:', callStatus.value)
    console.log('[VideoCall] 当前 pendingOffer:', !!pendingOffer.value)

    try {
      session.end()
      console.log('[VideoCall] session.end() 调用完成')
    } catch (error) {
      console.error('[VideoCall] session.end() 出错:', error)
    } finally {
      callStatus.value = 'idle'
      pendingOffer.value = null
      console.log('[VideoCall] callStatus 设置为 idle')
    }

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
