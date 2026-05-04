import { ref, computed } from 'vue'
import { useSession } from './useSession'
import { useSignaling } from './useSignaling'

export function useVideoCallNew() {
  const session = useSession('video-call')
  const signaling = useSignaling()
  
  const isCameraEnabled = ref(true)
  const isMicrophoneEnabled = ref(true)
  
  const isVideoEnabled = computed(() => {
    return session.localStream.value?.getVideoTracks().some(t => t.enabled) ?? false
  })
  
  const isAudioEnabled = computed(() => {
    return session.localStream.value?.getAudioTracks().some(t => t.enabled) ?? false
  })
  
  const startCall = async (targetUserId: number) => {
    console.log('[VideoCall] Starting call with user:', targetUserId)
    
    try {
      signaling.sendCallStart(targetUserId, 'video')
      
      await session.start(targetUserId)
      
      console.log('[VideoCall] Call started successfully')
    } catch (error) {
      console.error('[VideoCall] Failed to start call:', error)
      throw error
    }
  }
  
  const acceptCall = async (signal: RTCSessionDescriptionInit, fromUserId: number) => {
    console.log('[VideoCall] Accepting call from user:', fromUserId)
    
    try {
      signaling.sendCallAnswer(fromUserId, true)
      
      await session.join(signal, fromUserId)
      
      console.log('[VideoCall] Call accepted successfully')
    } catch (error) {
      console.error('[VideoCall] Failed to accept call:', error)
      signaling.sendCallAnswer(fromUserId, false)
      throw error
    }
  }
  
  const rejectCall = (fromUserId: number) => {
    console.log('[VideoCall] Rejecting call from user:', fromUserId)
    signaling.sendCallAnswer(fromUserId, false)
  }
  
  const endCall = () => {
    console.log('[VideoCall] Ending call')
    
    signaling.sendCallEnd()
    session.end()
    
    console.log('[VideoCall] Call ended successfully')
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
  
  return {
    ...session,
    isCameraEnabled,
    isMicrophoneEnabled,
    isVideoEnabled,
    isAudioEnabled,
    startCall,
    acceptCall,
    rejectCall,
    endCall,
    toggleCamera,
    toggleMicrophone,
    enableCamera,
    disableCamera,
    enableMicrophone,
    disableMicrophone
  }
}
