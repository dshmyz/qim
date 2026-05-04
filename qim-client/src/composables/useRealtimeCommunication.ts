import { ref, onUnmounted } from 'vue'
import { useConnection } from './useConnection'
import { useMediaStream } from './useMediaStream'
import { useSignaling } from './useSignaling'
import type { MediaType, MediaStreamSourceType } from '@/types/realtime'

function getSourceFromMediaType(mediaType: MediaType): MediaStreamSourceType {
  switch (mediaType) {
    case 'screen':
      return 'screen'
    case 'video':
      return 'camera'
    case 'audio':
      return 'microphone'
  }
}

export function useRealtimeCommunication(mediaType: MediaType) {
  const connection = useConnection()
  const mediaStream = useMediaStream(getSourceFromMediaType(mediaType))
  const signaling = useSignaling()
  
  const remoteStream = ref<MediaStream | null>(null)
  const targetUserId = ref<number | null>(null)
  
  const initiate = async (userId: number) => {
    console.log(`[RealtimeCommunication] Initiating ${mediaType} connection to user ${userId}`)
    
    targetUserId.value = userId
    
    try {
      await connection.createConnection()
      
      await mediaStream.start()
      
      if (mediaStream.stream.value) {
        mediaStream.stream.value.getTracks().forEach(track => {
          connection.addTrack(track, mediaStream.stream.value!)
        })
      }
      
      const offer = await connection.createOffer()
      await connection.setLocalDescription(offer!)
      
      signaling.sendOffer(userId, mediaType, offer!)
      
      console.log(`[RealtimeCommunication] Offer sent to user ${userId}`)
    } catch (error) {
      console.error(`[RealtimeCommunication] Failed to initiate connection:`, error)
      cleanup()
      throw error
    }
  }
  
  const receive = async (signal: RTCSessionDescriptionInit, fromUserId: number) => {
    console.log(`[RealtimeCommunication] Receiving ${mediaType} connection from user ${fromUserId}`)
    console.log(`[RealtimeCommunication] Signal:`, signal)
    
    targetUserId.value = fromUserId
    
    try {
      await connection.createConnection()
      console.log(`[RealtimeCommunication] Connection created for ${mediaType}`)
      
      setupRemoteStreamHandler()
      console.log(`[RealtimeCommunication] Remote stream handler set up for ${mediaType}`)
      
      await connection.setRemoteDescription(signal)
      console.log(`[RealtimeCommunication] Remote description set for ${mediaType}`)
      
      const answer = await connection.createAnswer()
      await connection.setLocalDescription(answer!)
      
      signaling.sendAnswer(fromUserId, mediaType, answer!)
      
      console.log(`[RealtimeCommunication] Answer sent to user ${fromUserId}`)
    } catch (error) {
      console.error(`[RealtimeCommunication] Failed to receive connection:`, error)
      cleanup()
      throw error
    }
  }
  
  const handleAnswer = async (signal: RTCSessionDescriptionInit) => {
    console.log(`[RealtimeCommunication] Handling answer for ${mediaType}`)
    
    try {
      const signalingState = connection.getSignalingState()
      console.log(`[RealtimeCommunication] Current signaling state:`, signalingState)
      
      if (signalingState !== 'have-local-offer') {
        console.warn(`[RealtimeCommunication] Invalid signaling state for answer: ${signalingState}`)
        return
      }
      
      await connection.setRemoteDescription(signal)
      console.log(`[RealtimeCommunication] Answer processed successfully`)
    } catch (error) {
      console.error(`[RealtimeCommunication] Failed to handle answer:`, error)
      throw error
    }
  }
  
  const handleIceCandidate = async (candidate: RTCIceCandidateInit) => {
    console.log(`[RealtimeCommunication] Handling ICE candidate for ${mediaType}`)
    
    try {
      await connection.addIceCandidate(candidate)
      console.log(`[RealtimeCommunication] ICE candidate processed successfully`)
    } catch (error) {
      console.error(`[RealtimeCommunication] Failed to handle ICE candidate:`, error)
    }
  }
  
  const setupRemoteStreamHandler = () => {
    if (connection.peerConnection.value) {
      console.log(`[RealtimeCommunication] Setting up remote stream handler for ${mediaType}`)
      connection.peerConnection.value.ontrack = (event) => {
        console.log(`[RealtimeCommunication] Received remote track:`, event.track.kind, event.track.id)
        console.log(`[RealtimeCommunication] Track state:`, event.track.readyState)
        console.log(`[RealtimeCommunication] Current remoteStream:`, remoteStream.value)
        
        if (!remoteStream.value) {
          remoteStream.value = new MediaStream()
          console.log(`[RealtimeCommunication] Created new MediaStream for remote`)
        }
        
        remoteStream.value.addTrack(event.track)
        console.log(`[RealtimeCommunication] Added track to remoteStream, total tracks:`, remoteStream.value.getTracks().length)
        
        event.track.onended = () => {
          console.log(`[RealtimeCommunication] Remote track ended:`, event.track.kind)
          if (remoteStream.value) {
            remoteStream.value.removeTrack(event.track)
          }
        }
      }
    } else {
      console.warn(`[RealtimeCommunication] No peer connection available for setting up remote stream handler`)
    }
  }
  
  const cleanup = () => {
    console.log(`[RealtimeCommunication] Cleaning up ${mediaType} connection`)
    mediaStream.stop()
    connection.close()
    remoteStream.value = null
    targetUserId.value = null
  }
  
  const close = () => {
    cleanup()
  }
  
  onUnmounted(() => {
    cleanup()
  })
  
  return {
    state: connection.state,
    localStream: mediaStream.stream,
    remoteStream,
    targetUserId,
    initiate,
    receive,
    handleAnswer,
    handleIceCandidate,
    close,
    mediaStream,
    connection
  }
}
