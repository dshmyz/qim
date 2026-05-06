import { ref, watch } from 'vue'
import { useRealtimeCommunication } from './useRealtimeCommunication'
import type { InitiateOptions, ReceiveOptions } from './useRealtimeCommunication'
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

const sessionInstances = new Map<SessionType, ReturnType<typeof createSession>>()

export function useSession(type: SessionType) {
  if (!sessionInstances.has(type)) {
    sessionInstances.set(type, createSession(type))
  }
  return sessionInstances.get(type)!
}

function createSession(type: SessionType) {
  const mediaType = getMediaTypeFromSessionType(type)
  const rtc = useRealtimeCommunication(mediaType)
  
  const sessionState = ref<SessionState>('idle')
  const participants = ref<Participant[]>([])
  
  const start = async (targetUserId: number, options?: InitiateOptions) => {
    console.log(`[Session] Starting ${type} session with user ${targetUserId}`)
    
    if (sessionState.value !== 'idle') {
      console.warn(`[Session] Cannot start session in state: ${sessionState.value}`)
      throw new Error(`Cannot start session in state: ${sessionState.value}`)
    }
    
    sessionState.value = 'connecting'
    
    try {
      await rtc.initiate(targetUserId, options)
      
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
  
  const join = async (signal: RTCSessionDescriptionInit, fromUserId: number, options?: ReceiveOptions) => {
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
      const answer = await rtc.receive(signal, fromUserId, options)
      console.log(`[Session] rtc.receive completed`)
      
      participants.value.push({
        userId: fromUserId,
        role: 'initiator',
        state: 'active'
      })
      
      console.log(`[Session] ${type} session joined successfully`)
      return answer
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
    console.log(`[Session] rtc.state is ref:`, typeof rtc.state === 'object' && 'value' in rtc.state)
    console.log(`[Session] rtc.state.value:`, rtc.state.value)
    
    if (newState === 'disconnected' && sessionState.value !== 'idle' && sessionState.value !== 'ended') {
      console.log(`[Session] Connection lost, resetting session state`)
      end()
    }
    
    if (newState === 'connected' && sessionState.value === 'connecting') {
      console.log(`[Session] Connection established, setting session to active`)
      sessionState.value = 'active'
    }
  }, { immediate: true })
  
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
