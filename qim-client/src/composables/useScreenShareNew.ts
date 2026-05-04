import { ref } from 'vue'
import { useSession } from './useSession'
import { useSignaling } from './useSignaling'

export function useScreenShareNew() {
  const session = useSession('screen-share')
  const signaling = useSignaling()
  
  const selectedSource = ref<DisplayMediaStreamOptions | null>(null)
  const isPaused = ref(false)
  
  const selectSource = async () => {
    console.log('[ScreenShare] Selecting source')
    
    try {
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
  
  const startSharing = async (targetUserId: number, conversationId: number) => {
    console.log('[ScreenShare] Starting share with user:', targetUserId)
    
    try {
      signaling.sendScreenShareRequest(targetUserId, conversationId)
      
      await session.start(targetUserId)
      
      console.log('[ScreenShare] Share started successfully')
    } catch (error) {
      console.error('[ScreenShare] Failed to start share:', error)
      throw error
    }
  }
  
  const acceptShare = async (signal: RTCSessionDescriptionInit, fromUserId: number) => {
    console.log('[ScreenShare] Accepting share from user:', fromUserId)
    
    try {
      signaling.sendScreenShareResponse(fromUserId, true)
      
      await session.join(signal, fromUserId)
      
      console.log('[ScreenShare] Share accepted successfully')
    } catch (error) {
      console.error('[ScreenShare] Failed to accept share:', error)
      signaling.sendScreenShareResponse(fromUserId, false)
      throw error
    }
  }
  
  const rejectShare = (fromUserId: number) => {
    console.log('[ScreenShare] Rejecting share from user:', fromUserId)
    signaling.sendScreenShareResponse(fromUserId, false)
  }
  
  const stopSharing = () => {
    console.log('[ScreenShare] Stopping share')
    
    signaling.sendScreenShareStop()
    session.end()
    selectedSource.value = null
    isPaused.value = false
    
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
    selectSource,
    startSharing,
    acceptShare,
    rejectShare,
    stopSharing,
    pause,
    resume,
    togglePause
  }
}
