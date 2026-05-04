import { ref, onUnmounted } from 'vue'
import type { ConnectionState, RTCConfig } from '@/types/realtime'

const DEFAULT_RTC_CONFIG: RTCConfig = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' }
  ]
}

export function useConnection() {
  const peerConnection = ref<RTCPeerConnection | null>(null)
  const state = ref<ConnectionState>('disconnected')
  
  const createConnection = async (config: RTCConfig = DEFAULT_RTC_CONFIG) => {
    if (peerConnection.value) {
      console.warn('Connection already exists, closing old connection')
      close()
    }
    
    console.log('Creating RTCPeerConnection with config:', config)
    peerConnection.value = new RTCPeerConnection(config)
    state.value = 'connecting'
    
    peerConnection.value.onconnectionstatechange = () => {
      const connectionState = peerConnection.value?.connectionState
      console.log('Connection state changed:', connectionState)
      
      if (connectionState === 'connected') {
        state.value = 'connected'
      } else if (connectionState === 'disconnected' || connectionState === 'failed' || connectionState === 'closed') {
        state.value = 'disconnected'
      }
    }
    
    peerConnection.value.oniceconnectionstatechange = () => {
      const iceState = peerConnection.value?.iceConnectionState
      console.log('ICE connection state changed:', iceState)
      
      if (iceState === 'connected' || iceState === 'completed') {
        state.value = 'connected'
      } else if (iceState === 'disconnected' || iceState === 'failed' || iceState === 'closed') {
        state.value = 'disconnected'
      }
    }
  }
  
  const addTrack = (track: MediaStreamTrack, stream: MediaStream) => {
    if (!peerConnection.value) {
      console.error('PeerConnection not initialized')
      return
    }
    
    console.log('Adding track:', track.kind, track.id)
    peerConnection.value.addTrack(track, stream)
  }
  
  const removeTrack = (track: MediaStreamTrack) => {
    if (!peerConnection.value) {
      console.error('PeerConnection not initialized')
      return
    }
    
    console.log('Removing track:', track.kind, track.id)
    const sender = peerConnection.value.getSenders().find(s => s.track === track)
    if (sender) {
      peerConnection.value.removeTrack(sender)
    }
  }
  
  const createOffer = async () => {
    if (!peerConnection.value) {
      throw new Error('PeerConnection not initialized')
    }
    
    console.log('Creating offer...')
    const offer = await peerConnection.value.createOffer()
    console.log('Offer created:', offer)
    return offer
  }
  
  const createAnswer = async () => {
    if (!peerConnection.value) {
      throw new Error('PeerConnection not initialized')
    }
    
    console.log('Creating answer...')
    const answer = await peerConnection.value.createAnswer()
    console.log('Answer created:', answer)
    return answer
  }
  
  const setLocalDescription = async (description: RTCSessionDescriptionInit) => {
    if (!peerConnection.value) {
      throw new Error('PeerConnection not initialized')
    }
    
    console.log('Setting local description:', description.type)
    await peerConnection.value.setLocalDescription(description)
    console.log('Local description set successfully')
  }
  
  const setRemoteDescription = async (description: RTCSessionDescriptionInit) => {
    if (!peerConnection.value) {
      throw new Error('PeerConnection not initialized')
    }
    
    console.log('Setting remote description:', description.type)
    console.log('Current signaling state:', peerConnection.value.signalingState)
    
    await peerConnection.value.setRemoteDescription(new RTCSessionDescription(description))
    console.log('Remote description set successfully')
    console.log('New signaling state:', peerConnection.value.signalingState)
  }
  
  const addIceCandidate = async (candidate: RTCIceCandidateInit) => {
    if (!peerConnection.value) {
      throw new Error('PeerConnection not initialized')
    }
    
    console.log('Adding ICE candidate:', candidate)
    
    // 验证 candidate 对象
    if (!candidate || (!candidate.candidate && !candidate.sdpMid && !candidate.sdpMLineIndex)) {
      console.warn('Invalid ICE candidate, skipping:', candidate)
      return
    }
    
    // 如果 sdpMid 和 sdpMLineIndex 都为 null，尝试从 candidate 字符串中提取
    let iceCandidate: RTCIceCandidate
    try {
      if (candidate.sdpMid === null && candidate.sdpMLineIndex === null) {
        // 创建一个简单的 candidate 对象
        iceCandidate = new RTCIceCandidate({
          candidate: candidate.candidate || '',
          sdpMid: '0',
          sdpMLineIndex: 0
        })
      } else {
        iceCandidate = new RTCIceCandidate(candidate)
      }
      
      await peerConnection.value.addIceCandidate(iceCandidate)
      console.log('ICE candidate added successfully')
    } catch (error) {
      console.error('Failed to add ICE candidate:', error)
      console.error('Candidate data:', candidate)
      // 不抛出错误，继续处理
    }
  }
  
  const getStats = async () => {
    if (!peerConnection.value) {
      throw new Error('PeerConnection not initialized')
    }
    
    return await peerConnection.value.getStats()
  }
  
  const close = () => {
    if (peerConnection.value) {
      console.log('Closing connection')
      peerConnection.value.close()
      peerConnection.value = null
      state.value = 'disconnected'
    }
  }
  
  const getSignalingState = () => {
    return peerConnection.value?.signalingState
  }
  
  const getConnectionState = () => {
    return peerConnection.value?.connectionState
  }
  
  const getIceConnectionState = () => {
    return peerConnection.value?.iceConnectionState
  }
  
  onUnmounted(() => {
    close()
  })
  
  return {
    peerConnection,
    state,
    createConnection,
    addTrack,
    removeTrack,
    createOffer,
    createAnswer,
    setLocalDescription,
    setRemoteDescription,
    addIceCandidate,
    getStats,
    close,
    getSignalingState,
    getConnectionState,
    getIceConnectionState
  }
}
