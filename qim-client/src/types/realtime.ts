// 实时通信类型定义

// 媒体类型
export type MediaType = 'screen' | 'video' | 'audio'

// 会话类型
export type SessionType = 'screen-share' | 'video-call' | 'audio-call'

// 连接状态
export type ConnectionState = 'disconnected' | 'connecting' | 'connected'

// 流状态
export type StreamState = 'stopped' | 'starting' | 'active'

// 会话状态
export type SessionState = 'idle' | 'connecting' | 'active' | 'ended'

// 参与者角色
export type ParticipantRole = 'initiator' | 'receiver'

// 参与者状态
export type ParticipantState = 'joining' | 'active' | 'leaving'

// 参与者
export interface Participant {
  userId: number
  role: ParticipantRole
  state: ParticipantState
}

// WebRTC 信令消息
export interface WebRTCSignalingMessage {
  type: 'webrtc.offer' | 'webrtc.answer' | 'webrtc.ice-candidate'
  data: {
    target_user_id: number
    from_user_id?: number
    media_type: MediaType
    signal?: RTCSessionDescriptionInit
    candidate?: RTCIceCandidateInit
  }
}

// 业务层消息
export interface BusinessMessage {
  type: string
  data: {
    target_user_id?: number
    from_user_id?: number
    media_type?: MediaType
    [key: string]: any
  }
}

// RTC 配置
export interface RTCConfig {
  iceServers: RTCIceServer[]
}

// 媒体流约束
export interface MediaStreamConstraints {
  video: boolean | MediaTrackConstraints
  audio: boolean | MediaTrackConstraints
}

// 媒体流源类型
export type MediaStreamSourceType = 'camera' | 'screen' | 'microphone'

// 连接协议
export type ConnectionProtocol = 'webrtc' | 'websocket'

// 流状态变化回调
export type StreamStateChangeCallback = (state: StreamState) => void

// 连接状态变化回调
export type ConnectionStateChangeCallback = (state: ConnectionState) => void

// 会话状态变化回调
export type SessionStateChangeCallback = (state: SessionState) => void

// 后端 API 返回的会话数据结构
export interface RealtimeSession {
  id: string
  type: 'screen_share' | 'voice_call' | 'video_call'
  conversation_id: number
  initiator_id: number
  initiator?: {
    id: number
    name: string
    avatar: string
  }
  status: 'active' | 'ended' | 'pending' | 'paused'
  participants: RealtimeParticipant[]
  created_at: string
}

export interface RealtimeParticipant {
  id: string
  user_id: number
  session_id: string
  role: 'initiator' | 'receiver' | 'viewer'
  state: 'joining' | 'active' | 'leaving'
  status?: 'active' | 'inactive'
  joined_at: string
  user?: {
    id: number
    name: string
    avatar: string
  }
}

export interface JoinRequest {
  id: string
  session_id: string
  user_id: number
  status: 'pending' | 'approved' | 'rejected'
  created_at: string
}

// WebRTC 信令数据
export interface WebRTCOfferData {
  type: 'webrtc.offer'
  data: {
    sdp: RTCSessionDescriptionInit
    target_user_id: number
    from_user_id: number
    media_type: MediaType
  }
}

export interface WebRTCAnswerData {
  type: 'webrtc.answer'
  data: {
    sdp: RTCSessionDescriptionInit
    target_user_id: number
    from_user_id: number
    media_type: MediaType
  }
}

export interface WebRTCIceData {
  type: 'webrtc.ice-candidate'
  data: {
    candidate: RTCIceCandidateInit
    target_user_id: number
    from_user_id: number
    media_type: MediaType
  }
}
