import { ref, watch, onUnmounted } from 'vue'
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
      throw new Error(`Cannot start session in state: ${sessionState.value}`)
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
    console.log(`[Session] Signal type:`, signal.type)
    console.log(`[Session] Current session state:`, sessionState.value)
    
    if (sessionState.value !== 'idle') {
      console.warn(`[Session] Cannot join session in state: ${sessionState.value}`)
      throw new Error(`Cannot join session in state: ${sessionState.value}`)
    }
    
    sessionState.value = 'connecting'
    console.log(`[Session] Session state set to connecting`)
    
    try {
      console.log(`[Session] Calling rtc.receive...`)
      await rtc.receive(signal, fromUserId)
      console.log(`[Session] rtc.receive completed`)
      
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
  
  watch(rtc.state, (newState) => {
    console.log(`[Session] Connection state changed to: ${newState}, session state: ${sessionState.value}`)
    
    if (newState === 'disconnected' && sessionState.value !== 'idle' && sessionState.value !== 'ended') {
      console.log(`[Session] Connection lost, resetting session state`)
      end()
    }
    
    if (newState === 'connected' && sessionState.value === 'connecting') {
      console.log(`[Session] Connection established, setting session to active`)
      sessionState.value = 'active'
    }
  })
  
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
