import { ref } from 'vue'
import type { ConnectionState } from '@/types/realtime'

const DIRECT_CONNECT_CONFIG = {
  iceTransportPolicy: 'all' as RTCIceTransportPolicy,
  iceCandidatePoolSize: 10
}

const ICE_SERVER_CONFIG = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' },
    { urls: 'stun:stun2.l.google.com:19302' }
  ],
  iceTransportPolicy: 'all' as RTCIceTransportPolicy,
  iceCandidatePoolSize: 10
}

export function useConnection() {
  const peerConnection = ref<RTCPeerConnection | null>(null)
  const state = ref<ConnectionState>('disconnected')
  const enableDirectConnect = ref(true)

  const createConnection = async (useDirectConnect = true) => {
    if (peerConnection.value) {
      console.warn('Connection already exists, closing old connection')
      close()
    }

    enableDirectConnect.value = useDirectConnect
    const config = useDirectConnect ? DIRECT_CONNECT_CONFIG : ICE_SERVER_CONFIG

    console.log('[Connection] Creating RTCPeerConnection, directConnect:', useDirectConnect)
    peerConnection.value = new RTCPeerConnection(config)
    state.value = 'connecting'

    peerConnection.value.onconnectionstatechange = () => {
      const connectionState = peerConnection.value?.connectionState
      console.log('[Connection] State changed:', connectionState)

      if (connectionState === 'connected') {
        state.value = 'connected'
        console.log('[Connection] state.value set to:', state.value)
      } else if (connectionState === 'disconnected' || connectionState === 'failed' || connectionState === 'closed') {
        state.value = 'disconnected'
        console.log('[Connection] state.value set to:', state.value)
      }
    }

    peerConnection.value.oniceconnectionstatechange = () => {
      const iceState = peerConnection.value?.iceConnectionState
      console.log('[Connection] ICE state changed:', iceState)

      if (iceState === 'connected' || iceState === 'completed') {
        state.value = 'connected'
        console.log('[Connection] state.value set to:', state.value)
      } else if (iceState === 'failed' || iceState === 'disconnected' || iceState === 'closed') {
        state.value = 'disconnected'
        console.log('[Connection] state.value set to:', state.value)

        if (enableDirectConnect.value && iceState === 'failed') {
          console.log('[Connection] Direct connect failed, will retry with ICE servers')
        }
      }
    }
  }

  const recreateWithIceServers = async () => {
    console.log('[Connection] Recreating connection with ICE servers')
    const oldConnection = peerConnection.value
    close()

    enableDirectConnect.value = false
    peerConnection.value = new RTCPeerConnection(ICE_SERVER_CONFIG)
    state.value = 'connecting'

    peerConnection.value.onconnectionstatechange = () => {
      const connectionState = peerConnection.value?.connectionState
      console.log('[Connection] State changed:', connectionState)

      if (connectionState === 'connected') {
        state.value = 'connected'
      } else if (connectionState === 'disconnected' || connectionState === 'failed' || connectionState === 'closed') {
        state.value = 'disconnected'
      }
    }

    peerConnection.value.oniceconnectionstatechange = () => {
      const iceState = peerConnection.value?.iceConnectionState
      console.log('[Connection] ICE state changed:', iceState)

      if (iceState === 'connected' || iceState === 'completed') {
        state.value = 'connected'
      } else if (iceState === 'disconnected' || iceState === 'failed' || iceState === 'closed') {
        state.value = 'disconnected'
      }
    }

    return oldConnection
  }

  const addTrack = (track: MediaStreamTrack, stream: MediaStream) => {
    if (!peerConnection.value) {
      console.error('PeerConnection not initialized')
      return
    }

    peerConnection.value.addTrack(track, stream)
  }

  const removeTrack = (track: MediaStreamTrack) => {
    if (!peerConnection.value) {
      console.error('PeerConnection not initialized')
      return
    }

    const sender = peerConnection.value.getSenders().find(s => s.track === track)
    if (sender) {
      peerConnection.value.removeTrack(sender)
    }
  }

  const createOffer = async () => {
    if (!peerConnection.value) {
      throw new Error('PeerConnection not initialized')
    }

    const offer = await peerConnection.value.createOffer()
    return offer
  }

  const createAnswer = async () => {
    if (!peerConnection.value) {
      throw new Error('PeerConnection not initialized')
    }

    const answer = await peerConnection.value.createAnswer()
    return answer
  }

  const setLocalDescription = async (description: RTCSessionDescriptionInit) => {
    if (!peerConnection.value) {
      throw new Error('PeerConnection not initialized')
    }

    await peerConnection.value.setLocalDescription(description)
  }

  const setRemoteDescription = async (description: RTCSessionDescriptionInit) => {
    if (!peerConnection.value) {
      throw new Error('PeerConnection not initialized')
    }

    console.log('[Connection] Setting remote description:', description.type, 'signaling state:', peerConnection.value.signalingState)

    await peerConnection.value.setRemoteDescription(new RTCSessionDescription(description))
  }

  const addIceCandidate = async (candidate: RTCIceCandidateInit) => {
    if (!peerConnection.value) {
      throw new Error('PeerConnection not initialized')
    }

    if (!candidate || (!candidate.candidate && !candidate.sdpMid && !candidate.sdpMLineIndex)) {
      console.warn('[Connection] Invalid ICE candidate, skipping:', candidate)
      return
    }

    let iceCandidate: RTCIceCandidate
    try {
      if (candidate.sdpMid === null && candidate.sdpMLineIndex === null) {
        iceCandidate = new RTCIceCandidate({
          candidate: candidate.candidate || '',
          sdpMid: '0',
          sdpMLineIndex: 0
        })
      } else {
        iceCandidate = new RTCIceCandidate(candidate)
      }

      await peerConnection.value.addIceCandidate(iceCandidate)
      console.log('[Connection] ICE candidate added successfully')
    } catch (error) {
      console.error('[Connection] Failed to add ICE candidate:', error, candidate)
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

  return {
    peerConnection,
    state,
    enableDirectConnect,
    createConnection,
    recreateWithIceServers,
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
