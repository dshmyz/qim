import type { WebRTCOfferData, WebRTCAnswerData, WebRTCIceData } from '../types/realtime'

interface SignalingMessage {
  type: 'webrtc_offer' | 'webrtc_answer' | 'webrtc_ice_candidate'
  data: WebRTCOfferData | WebRTCAnswerData | WebRTCIceData
}

interface ManagerCallbacks {
  onRemoteStream?: (viewerId: number, stream: MediaStream) => void
  onConnectionStateChange?: (viewerId: number, state: RTCPeerConnectionState) => void
  onError?: (viewerId: number, error: Error) => void
}

interface ViewerCallbacks {
  onRemoteStream?: (stream: MediaStream) => void
  onConnectionStateChange?: (state: RTCPeerConnectionState) => void
  onError?: (error: Error) => void
}

const ICE_SERVERS: RTCConfiguration = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' },
    { urls: 'stun:stun2.l.google.com:19302' }
  ],
  iceCandidatePoolSize: 10
}

function sendSignalingMessage(message: SignalingMessage): void {
  if (window.electron?.websocket) {
    window.electron.websocket.send(message)
  } else if ((window as any).ws) {
    ;(window as any).ws.send(JSON.stringify(message))
  }
}

export class RealtimeConnectionManager {
  private connections: Map<number, RTCPeerConnection> = new Map()
  private localStream: MediaStream | null = null
  private sessionId: string | null = null
  private callbacks: ManagerCallbacks = {}

  setLocalStream(stream: MediaStream): void {
    this.localStream = stream
  }

  setSessionId(sessionId: string): void {
    this.sessionId = sessionId
  }

  setCallbacks(callbacks: ManagerCallbacks): void {
    this.callbacks = callbacks
  }

  async createConnectionForViewer(viewerId: number): Promise<void> {
    if (!this.localStream || !this.sessionId) {
      throw new Error('Local stream or session ID not set')
    }

    const connection = new RTCPeerConnection(ICE_SERVERS)

    this.localStream.getTracks().forEach(track => {
      connection.addTrack(track, this.localStream!)
    })

    connection.onicecandidate = (event) => {
      if (event.candidate) {
        const message: SignalingMessage = {
          type: 'webrtc_ice_candidate',
          data: {
            session_id: this.sessionId!,
            from_user_id: 0,
            signal: event.candidate.toJSON()
          } as WebRTCIceData
        }
        sendSignalingMessage(message)
      }
    }

    connection.onconnectionstatechange = () => {
      this.callbacks.onConnectionStateChange?.(viewerId, connection.connectionState)
    }

    connection.ontrack = (event) => {
      if (event.streams && event.streams[0]) {
        this.callbacks.onRemoteStream?.(viewerId, event.streams[0])
      }
    }

    const offer = await connection.createOffer()
    await connection.setLocalDescription(offer)

    const message: SignalingMessage = {
      type: 'webrtc_offer',
      data: {
        session_id: this.sessionId,
        from_user_id: 0,
        signal: {
          type: offer.type,
          sdp: offer.sdp
        }
      } as WebRTCOfferData
    }
    sendSignalingMessage(message)

    this.connections.set(viewerId, connection)
  }

  async handleAnswer(viewerId: number, answer: RTCSessionDescriptionInit): Promise<void> {
    const connection = this.connections.get(viewerId)
    if (!connection) {
      throw new Error(`No connection found for viewer ${viewerId}`)
    }

    await connection.setRemoteDescription(new RTCSessionDescription(answer))
  }

  async handleIceCandidate(viewerId: number, candidate: RTCIceCandidateInit): Promise<void> {
    const connection = this.connections.get(viewerId)
    if (!connection) {
      throw new Error(`No connection found for viewer ${viewerId}`)
    }

    await connection.addIceCandidate(new RTCIceCandidate(candidate))
  }

  closeConnection(viewerId: number): void {
    const connection = this.connections.get(viewerId)
    if (connection) {
      connection.close()
      this.connections.delete(viewerId)
    }
  }

  closeAllConnections(): void {
    this.connections.forEach((connection) => {
      connection.close()
    })
    this.connections.clear()
  }

  getViewerIds(): number[] {
    return Array.from(this.connections.keys())
  }

  getConnectionCount(): number {
    return this.connections.size
  }
}

export class RealtimeViewerConnection {
  private connection: RTCPeerConnection | null = null
  private sessionId: string | null = null
  private callbacks: ViewerCallbacks = {}

  setCallbacks(callbacks: ViewerCallbacks): void {
    this.callbacks = callbacks
  }

  async handleOffer(sessionId: string, _initiatorId: number, offer: RTCSessionDescriptionInit): Promise<void> {
    this.sessionId = sessionId

    this.connection = new RTCPeerConnection(ICE_SERVERS)

    this.connection.onicecandidate = (event) => {
      if (event.candidate) {
        const message: SignalingMessage = {
          type: 'webrtc_ice_candidate',
          data: {
            session_id: this.sessionId!,
            from_user_id: 0,
            signal: event.candidate.toJSON()
          } as WebRTCIceData
        }
        sendSignalingMessage(message)
      }
    }

    this.connection.onconnectionstatechange = () => {
      this.callbacks.onConnectionStateChange?.(this.connection!.connectionState)
    }

    this.connection.ontrack = (event) => {
      if (event.streams && event.streams[0]) {
        this.callbacks.onRemoteStream?.(event.streams[0])
      }
    }

    await this.connection.setRemoteDescription(new RTCSessionDescription(offer))

    const answer = await this.connection.createAnswer()
    await this.connection.setLocalDescription(answer)

    const message: SignalingMessage = {
      type: 'webrtc_answer',
      data: {
        session_id: this.sessionId,
        from_user_id: 0,
        signal: {
          type: answer.type,
          sdp: answer.sdp
        }
      } as WebRTCAnswerData
    }
    sendSignalingMessage(message)
  }

  async handleIceCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    if (!this.connection) {
      throw new Error('No active connection')
    }

    await this.connection.addIceCandidate(new RTCIceCandidate(candidate))
  }

  close(): void {
    if (this.connection) {
      this.connection.close()
      this.connection = null
    }
    this.sessionId = null
  }
}
