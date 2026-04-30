export type SessionType = 'screen_share' | 'voice_call' | 'video_call';

export type SessionStatus = 'pending' | 'active' | 'paused' | 'ended';

export type ParticipantRole = 'initiator' | 'viewer';

export type ParticipantStatus = 'pending' | 'approved' | 'rejected' | 'joined' | 'left';

export interface RealtimeSession {
  id: string;
  type: SessionType;
  initiator_id: number;
  conversation_id: number;
  status: SessionStatus;
  started_at: string | null;
  ended_at: string | null;
  metadata: string | null;
  created_at: string;
  updated_at: string;
  initiator?: {
    id: number;
    nickname: string;
    avatar: string;
  };
  participants?: RealtimeParticipant[];
}

export interface RealtimeParticipant {
  id: string;
  session_id: string;
  user_id: number;
  role: ParticipantRole;
  status: ParticipantStatus;
  requested_at: string;
  approved_at: string | null;
  joined_at: string | null;
  left_at: string | null;
  user?: {
    id: number;
    nickname: string;
    avatar: string;
  };
}

export interface JoinRequest {
  session_id: string;
  user_id: number;
  user?: {
    id: number;
    nickname: string;
    avatar: string;
  };
  timestamp: number;
}

export interface WebRTCOfferData {
  session_id: string;
  from_user_id: number;
  signal: RTCSessionDescriptionInit;
  timestamp: number;
}

export interface WebRTCAnswerData {
  session_id: string;
  from_user_id: number;
  signal: RTCSessionDescriptionInit;
  timestamp: number;
}

export interface WebRTCIceData {
  session_id: string;
  from_user_id: number;
  signal: RTCIceCandidateInit;
  timestamp: number;
}
