import { ref } from 'vue'
import { useRealtimeCommunication } from './useRealtimeCommunication'
import type { SessionType, MediaType, SessionState, Participant } from '@/types/realtime'

function getMediaTypeFromSessionType(type: SessionType): MediaType {
  switch (type) {
    case 'screen-share':
      return 'screen'
    case 'video-call':
      return 'video'
    case 'audio-call':
      return 'audio'
  }
}

export function useSession(type: SessionType) {
  const mediaType = getMediaTypeFromSessionType(type)
  const rtc = useRealtimeCommunication(mediaType)
  
  const sessionState = ref<SessionState>('idle')
  const participants = ref<Participant[]>([])
  
  const start = async (targetUserId: number) => {
    console.log(`[Session] Starting ${type} session with user ${targetUserId}`)
    
    if (sessionState.value !== 'idle') {
      console.warn(`[Session] Cannot start session in state: ${sessionState.value}`)
      return
    }
    
    sessionState.value = 'connecting'
    
    try {
      await rtc.initiate(targetUserId)
      
      participants.value.push({
        userId: targetUserId,
        role: 'receiver',
        state: 'active'
      })
      
      console.log(`[Session] ${type} session started successfully`)
    } catch (error) {
      console.error(`[Session] Failed to start ${type} session:`, error)
      sessionState.value = 'idle'
      throw error
    }
  }
  
  const join = async (signal: RTCSessionDescriptionInit, fromUserId: number) => {
    console.log(`[Session] Joining ${type} session from user ${fromUserId}`)
    
    if (sessionState.value !== 'idle') {
      console.warn(`[Session] Cannot join session in state: ${sessionState.value}`)
      return
    }
    
    sessionState.value = 'connecting'
    
    try {
      await rtc.receive(signal, fromUserId)
      
      participants.value.push({
        userId: fromUserId,
        role: 'initiator',
        state: 'active'
      })
      
      console.log(`[Session] ${type} session joined successfully`)
    } catch (error) {
      console.error(`[Session] Failed to join ${type} session:`, error)
      sessionState.value = 'idle'
      throw error
    }
  }
  
  const end = () => {
    console.log(`[Session] Ending ${type} session`)
    
    rtc.close()
    sessionState.value = 'ended'
    participants.value = []
    
    setTimeout(() => {
      sessionState.value = 'idle'
    }, 100)
  }
  
  const addParticipant = (userId: number, role: 'initiator' | 'receiver') => {
    if (!participants.value.find(p => p.userId === userId)) {
      participants.value.push({
        userId,
        role,
        state: 'joining'
      })
    }
  }
  
  const removeParticipant = (userId: number) => {
    const participant = participants.value.find(p => p.userId === userId)
    if (participant) {
      participant.state = 'leaving'
      setTimeout(() => {
        participants.value = participants.value.filter(p => p.userId !== userId)
      }, 100)
    }
  }
  
  const updateParticipantState = (userId: number, state: Participant['state']) => {
    const participant = participants.value.find(p => p.userId === userId)
    if (participant) {
      participant.state = state
    }
  }
  
  const isConnected = () => {
    return sessionState.value === 'active' && rtc.state.value === 'connected'
  }
  
  return {
    sessionState,
    participants,
    localStream: rtc.localStream,
    remoteStream: rtc.remoteStream,
    targetUserId: rtc.targetUserId,
    start,
    join,
    end,
    handleAnswer: rtc.handleAnswer,
    handleIceCandidate: rtc.handleIceCandidate,
    addParticipant,
    removeParticipant,
    updateParticipantState,
    isConnected,
    rtc
  }
}
