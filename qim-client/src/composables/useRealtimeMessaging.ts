import { ref } from 'vue'
import { useScreenShareNew } from './useScreenShareNew'
import { useVideoCallNew } from './useVideoCallNew'
import type { MediaType } from '@/types/realtime'

let realtimeMessagingInstance: ReturnType<typeof createRealtimeMessaging> | null = null

function createRealtimeMessaging() {
  const screenShare = useScreenShareNew()
  const videoCall = useVideoCallNew()
  
  const getMediaType = (data: any): MediaType | null => {
    if (data.media_type) {
      return data.media_type
    }
    if (data.share_type) {
      return data.share_type
    }
    if (data.call_type) {
      return data.call_type
    }
    return null
  }
  
  const handleWebRTCOffer = async (data: any) => {
    console.log('[RealtimeMessaging] Handling WebRTC offer:', data)

    const mediaType = getMediaType(data)
    if (!mediaType) {
      console.warn('[RealtimeMessaging] No media_type in offer, skipping')
      return
    }

    console.log('[RealtimeMessaging] Media type:', mediaType)

    try {
      switch (mediaType) {
        case 'screen':
          console.log('[RealtimeMessaging] Screen share offer - NOT auto-accepting, waiting for UI acceptance')
          break
        case 'video':
        case 'audio':
          console.log('[RealtimeMessaging] Call offer - routing to UI for acceptance')
          if (onCallOffer.value) {
            onCallOffer.value(data)
          }
          break
        default:
          console.warn('[RealtimeMessaging] Unknown media type:', mediaType)
      }
    } catch (error) {
      console.error('[RealtimeMessaging] Failed to handle offer:', error)
    }
  }
  
  const handleWebRTCAnswer = async (data: any) => {
    console.log('[RealtimeMessaging] Handling WebRTC answer:', data)
    
    const mediaType = getMediaType(data)
    if (!mediaType) {
      console.warn('[RealtimeMessaging] No media_type in answer, skipping')
      return
    }
    
    console.log('[RealtimeMessaging] Media type:', mediaType)
    
    try {
      switch (mediaType) {
        case 'screen':
          await screenShare.handleAnswer(data.signal)
          break
        case 'video':
        case 'audio':
          await videoCall.handleAnswer(data.signal)
          break
        default:
          console.warn('[RealtimeMessaging] Unknown media type:', mediaType)
      }
    } catch (error) {
      console.error('[RealtimeMessaging] Failed to handle answer:', error)
    }
  }
  
  const handleWebRTCIceCandidate = async (data: any) => {
    console.log('[RealtimeMessaging] Handling WebRTC ICE candidate:', data)
    
    const mediaType = getMediaType(data)
    if (!mediaType) {
      console.warn('[RealtimeMessaging] No media_type in ICE candidate, skipping')
      return
    }
    
    console.log('[RealtimeMessaging] Media type:', mediaType)
    
    const candidate = data.candidate || data.signal
    
    if (!candidate) {
      console.error('[RealtimeMessaging] No ICE candidate data found in:', data)
      return
    }
    
    console.log('[RealtimeMessaging] ICE candidate object:', candidate)
    
    try {
      switch (mediaType) {
        case 'screen':
          await screenShare.handleIceCandidate(candidate)
          break
        case 'video':
        case 'audio':
          await videoCall.handleIceCandidate(candidate)
          break
        default:
          console.warn('[RealtimeMessaging] Unknown media type:', mediaType)
      }
    } catch (error) {
      console.error('[RealtimeMessaging] Failed to handle ICE candidate:', error)
    }
  }
  
  const onScreenShareRequest = ref<((data: any) => void) | null>(null)
  const onScreenShareAccepted = ref<((data: any) => void) | null>(null)
  const onScreenShareRejected = ref<((data: any) => void) | null>(null)
  const onWebRTCOffer = ref<((data: any) => void) | null>(null)
  const onCallOffer = ref<((data: any) => void) | null>(null)

  const handleScreenShareRequest = (data: any) => {
    console.log('[RealtimeMessaging] Handling screen share request:', data)
    if (onScreenShareRequest.value) {
      onScreenShareRequest.value(data)
    }
  }

  const handleScreenShareResponse = (data: any) => {
    console.log('[RealtimeMessaging] Handling screen share response:', data)
  }

  const handleScreenShareAccepted = (data: any) => {
    console.log('[RealtimeMessaging] Screen share accepted:', data)
    if (onScreenShareAccepted.value) {
      onScreenShareAccepted.value(data)
    }
  }

  const handleScreenShareRejected = (data: any) => {
    console.log('[RealtimeMessaging] Screen share rejected:', data)
    if (onScreenShareRejected.value) {
      onScreenShareRejected.value(data)
    }
  }

  const handleWebRTCOfferForScreen = (data: any) => {
    console.log('[RealtimeMessaging] Routing WebRTC offer for screen to UI')
    if (onWebRTCOffer.value) {
      onWebRTCOffer.value(data)
    }
  }
  
  const handleScreenShareStop = (data: any) => {
    console.log('[RealtimeMessaging] Handling screen share stop:', data)
    screenShare.stopSharing()
  }
  
  const handleCallStart = (data: any) => {
    console.log('[RealtimeMessaging] Handling call start:', data)
    
  }
  
  const handleCallAnswer = (data: any) => {
    console.log('[RealtimeMessaging] Handling call answer:', data)
    
  }
  
  const handleCallEnd = (data: any) => {
    console.log('[RealtimeMessaging] Handling call end:', data)
    videoCall.endCall()
  }
  
  return {
    screenShare,
    videoCall,
    onScreenShareRequest,
    onScreenShareAccepted,
    onScreenShareRejected,
    onWebRTCOffer,
    onCallOffer,
    handleWebRTCOffer,
    handleWebRTCAnswer,
    handleWebRTCIceCandidate,
    handleScreenShareRequest,
    handleScreenShareResponse,
    handleScreenShareAccepted,
    handleScreenShareRejected,
    handleWebRTCOfferForScreen,
    handleScreenShareStop,
    handleCallStart,
    handleCallAnswer,
    handleCallEnd
  }
}

export function useRealtimeMessaging() {
  if (!realtimeMessagingInstance) {
    realtimeMessagingInstance = createRealtimeMessaging()
  }
  
  return realtimeMessagingInstance
}
