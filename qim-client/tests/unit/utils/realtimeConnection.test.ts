import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { RealtimeConnectionManager } from '../../../src/utils/realtimeConnection'

class FakeRTCPeerConnection {
  onicecandidate: ((event: { candidate: unknown }) => void) | null = null
  onconnectionstatechange: (() => void) | null = null
  ontrack: (() => void) | null = null
  connectionState = 'new'

  addTrack = vi.fn()
  createOffer = vi.fn(async () => ({ type: 'offer', sdp: 'fake-offer' }))
  setLocalDescription = vi.fn(async () => {})
  close = vi.fn()
}

describe('RealtimeConnectionManager', () => {
  let originalRTCPeerConnection: typeof globalThis.RTCPeerConnection
  let originalWs: unknown

  beforeEach(() => {
    originalRTCPeerConnection = globalThis.RTCPeerConnection
    originalWs = (window as any).ws
    vi.stubGlobal('RTCPeerConnection', FakeRTCPeerConnection)
  })

  afterEach(() => {
    vi.stubGlobal('RTCPeerConnection', originalRTCPeerConnection)
    ;(window as any).ws = originalWs
  })

  it('does not send realtime signaling through a stale window WebSocket', async () => {
    const staleWebSocket = {
      readyState: WebSocket.CLOSED,
      send: vi.fn(),
    }
    ;(window as any).ws = staleWebSocket

    const manager = new RealtimeConnectionManager()
    manager.setSessionId('session-1')
    manager.setLocalStream({
      getTracks: () => [{ id: 'track-1' }],
    } as unknown as MediaStream)

    await expect(manager.createConnectionForViewer(42)).rejects.toThrow('WebSocket connection not available')
    expect(staleWebSocket.send).not.toHaveBeenCalled()
  })
})
