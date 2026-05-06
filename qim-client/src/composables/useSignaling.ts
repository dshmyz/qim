import type { MediaType, WebRTCSignalingMessage } from '@/types/realtime'

export function useSignaling() {
  const sendMessage = (type: string, data: any) => {
    if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
      const message = JSON.stringify({ type, data })
      console.log('Sending signaling message:', type, data)
      window.ws.send(message)
    } else if (typeof window !== 'undefined' && window.electron && window.electron.websocket) {
      console.log('Sending signaling message via IPC:', type, data)
      window.electron.websocket.send({ type, data })
    } else {
      console.error('WebSocket connection not available')
      throw new Error('WebSocket connection not available')
    }
  }
  
  const createSignalingMessage = (
    type: string,
    targetUserId: number,
    mediaType: MediaType,
    payload: any
  ): WebRTCSignalingMessage => {
    return {
      type: type as any,
      data: {
        target_user_id: targetUserId,
        media_type: mediaType,
        ...payload
      }
    }
  }
  
  const sendOffer = (
    targetUserId: number,
    mediaType: MediaType,
    offer: RTCSessionDescriptionInit
  ) => {
    const message = createSignalingMessage('webrtc.offer', targetUserId, mediaType, {
      signal: offer
    })
    sendMessage(message.type, message.data)
  }
  
  const sendAnswer = (
    targetUserId: number,
    mediaType: MediaType,
    answer: RTCSessionDescriptionInit
  ) => {
    const message = createSignalingMessage('webrtc.answer', targetUserId, mediaType, {
      signal: answer
    })
    sendMessage(message.type, message.data)
  }
  
  const sendIceCandidate = (
    targetUserId: number,
    mediaType: MediaType,
    candidate: RTCIceCandidate
  ) => {
    const message = createSignalingMessage('webrtc.ice-candidate', targetUserId, mediaType, {
      candidate: candidate.toJSON()
    })
    sendMessage(message.type, message.data)
  }
  
  const sendBusinessMessage = (
    type: string,
    targetUserId: number,
    mediaType: MediaType,
    payload: any
  ) => {
    const message = {
      type,
      data: {
        target_user_id: targetUserId,
        media_type: mediaType,
        ...payload
      }
    }
    sendMessage(type, message.data)
  }
  
  const sendScreenShareRequest = (targetUserId: number, conversationId: number) => {
    sendMessage('screen-share.request', {
      target_user_id: targetUserId,
      conversation_id: conversationId
    })
  }
  
  const sendScreenShareResponse = (conversationId: number, requesterId: number, accepted: boolean) => {
    sendMessage('screen-share.response', {
      conversation_id: conversationId,
      requester_id: requesterId,
      status: accepted ? 'accepted' : 'rejected'
    })
  }
  
  const sendScreenShareStop = () => {
    sendMessage('screen-share.stop', {})
  }
  
  const sendCallStart = (targetUserId: number, callType: 'voice' | 'video') => {
    sendMessage('call_invite', {
      target_user_id: targetUserId,
      call_type: callType
    })
  }

  const sendCallAnswer = (targetUserId: number, accepted: boolean) => {
    sendMessage('call_accept', {
      target_user_id: targetUserId,
      accepted
    })
  }

  const sendCallEnd = () => {
    sendMessage('call_end', {})
  }
  
  return {
    sendMessage,
    createSignalingMessage,
    sendOffer,
    sendAnswer,
    sendIceCandidate,
    sendBusinessMessage,
    sendScreenShareRequest,
    sendScreenShareResponse,
    sendScreenShareStop,
    sendCallStart,
    sendCallAnswer,
    sendCallEnd
  }
}
