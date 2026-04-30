import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { API_BASE_URL } from '../config'
import type { RealtimeSession, JoinRequest } from '../types/realtime'

function getToken() {
  return localStorage.getItem('token')
}

export const useRealtimeStore = defineStore('realtime', () => {
  const activeSessions = ref<RealtimeSession[]>([])
  const mySession = ref<RealtimeSession | null>(null)
  const pendingRequests = ref<JoinRequest[]>([])
  const currentViewingSession = ref<RealtimeSession | null>(null)

  const isSharing = computed(() => mySession.value !== null)
  const isViewing = computed(() => currentViewingSession.value !== null)

  function getActiveSessionByUser(userId: number) {
    return activeSessions.value.find(session => session.initiator_id === userId)
  }

  function getActiveSessionByConversation(conversationId: number) {
    return activeSessions.value.find(session => session.conversation_id === conversationId)
  }

  async function createSession(data: {
    type: 'screen_share' | 'voice_call' | 'video_call'
    conversation_id: number
  }): Promise<RealtimeSession> {
    const response = await fetch(`${API_BASE_URL}/api/v1/realtime/sessions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${getToken()}`
      },
      body: JSON.stringify(data)
    })
    if (!response.ok) {
      throw new Error('Failed to create session')
    }
    const result = await response.json()
    mySession.value = result.data
    return result.data
  }

  async function fetchActiveSessions(): Promise<RealtimeSession[]> {
    const response = await fetch(`${API_BASE_URL}/api/v1/realtime/sessions/active`, {
      headers: {
        'Authorization': `Bearer ${getToken()}`
      }
    })
    if (!response.ok) {
      throw new Error('Failed to fetch active sessions')
    }
    const result = await response.json()
    activeSessions.value = result.data
    return result.data
  }

  async function fetchSession(sessionId: string): Promise<RealtimeSession> {
    const response = await fetch(`${API_BASE_URL}/api/v1/realtime/sessions/${sessionId}`, {
      headers: {
        'Authorization': `Bearer ${getToken()}`
      }
    })
    if (!response.ok) {
      throw new Error('Failed to fetch session')
    }
    const result = await response.json()
    return result.data
  }

  async function requestJoin(sessionId: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/api/v1/realtime/sessions/${sessionId}/participants`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${getToken()}`
      }
    })
    if (!response.ok) {
      throw new Error('Failed to request join')
    }
  }

  async function approveJoin(sessionId: string, userId: number): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/api/v1/realtime/sessions/${sessionId}/participants/${userId}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${getToken()}`
      },
      body: JSON.stringify({ user_id: userId })
    })
    if (!response.ok) {
      throw new Error('Failed to approve join')
    }
  }

  async function rejectJoin(sessionId: string, userId: number): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/api/v1/realtime/sessions/${sessionId}/participants/${userId}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${getToken()}`
      },
      body: JSON.stringify({ user_id: userId })
    })
    if (!response.ok) {
      throw new Error('Failed to reject join')
    }
  }

  async function leaveSession(sessionId: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/api/v1/realtime/sessions/${sessionId}/participants`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${getToken()}`
      }
    })
    if (!response.ok) {
      throw new Error('Failed to leave session')
    }
    currentViewingSession.value = null
  }

  async function endSession(sessionId: string): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/api/v1/realtime/sessions/${sessionId}/end`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${getToken()}`
      }
    })
    if (!response.ok) {
      throw new Error('Failed to end session')
    }
    mySession.value = null
  }

  function addPendingRequest(request: JoinRequest) {
    pendingRequests.value.push(request)
  }

  function updateSession(session: RealtimeSession) {
    const index = activeSessions.value.findIndex(s => s.id === session.id)
    if (index !== -1) {
      activeSessions.value[index] = session
    } else {
      activeSessions.value.push(session)
    }
    if (mySession.value && mySession.value.id === session.id) {
      mySession.value = session
    }
    if (currentViewingSession.value && currentViewingSession.value.id === session.id) {
      currentViewingSession.value = session
    }
  }

  function removeSession(sessionId: string) {
    activeSessions.value = activeSessions.value.filter(s => s.id !== sessionId)
    if (mySession.value && mySession.value.id === sessionId) {
      mySession.value = null
    }
    if (currentViewingSession.value && currentViewingSession.value.id === sessionId) {
      currentViewingSession.value = null
    }
  }

  function setCurrentViewingSession(session: RealtimeSession | null) {
    currentViewingSession.value = session
  }

  return {
    activeSessions,
    mySession,
    pendingRequests,
    currentViewingSession,
    isSharing,
    isViewing,
    getActiveSessionByUser,
    getActiveSessionByConversation,
    createSession,
    fetchActiveSessions,
    fetchSession,
    requestJoin,
    approveJoin,
    rejectJoin,
    leaveSession,
    endSession,
    addPendingRequest,
    updateSession,
    removeSession,
    setCurrentViewingSession
  }
})
