import { beforeEach, describe, expect, it, vi } from 'vitest'

class MockRTCPeerConnection {
  localDescription: RTCSessionDescriptionInit | null = null
  remoteDescription: RTCSessionDescriptionInit | null = null
  onicecandidate: ((event: RTCPeerConnectionIceEvent) => void) | null = null
  ontrack: ((event: RTCTrackEvent) => void) | null = null

  addTrack = vi.fn()
  createOffer = vi.fn(async () => ({ type: 'offer' as RTCSdpType, sdp: 'mock-offer' }))
  createAnswer = vi.fn(async () => ({ type: 'answer' as RTCSdpType, sdp: 'mock-answer' }))
  setLocalDescription = vi.fn(async (description: RTCSessionDescriptionInit) => {
    this.localDescription = description
  })
  setRemoteDescription = vi.fn(async (description: RTCSessionDescriptionInit) => {
    this.remoteDescription = description
  })
  addIceCandidate = vi.fn(async () => undefined)
  close = vi.fn()
}

const createAudioTrack = (readyState: MediaStreamTrackState = 'live') => ({
  kind: 'audio',
  readyState,
  enabled: true,
  stop: vi.fn(),
})

const createStream = (tracks: any[]) => ({
  getTracks: vi.fn(() => tracks),
  getAudioTracks: vi.fn(() => tracks.filter(track => track.kind === 'audio')),
  getVideoTracks: vi.fn(() => tracks.filter(track => track.kind === 'video')),
})

const importUseVideoCallNew = async () => {
  vi.resetModules()
  return import('@/composables/useVideoCallNew')
}

describe('useVideoCallNew media validation', () => {
  let send: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.restoreAllMocks()
    send = vi.fn()

    Object.defineProperty(window, 'ws', {
      value: {
        readyState: WebSocket.OPEN,
        send,
      },
      configurable: true,
    })

    Object.defineProperty(globalThis, 'RTCPeerConnection', {
      value: MockRTCPeerConnection,
      configurable: true,
    })
  })

  it('does not send voice call signaling when no audio input device is available', async () => {
    const getUserMedia = vi.fn(async () => createStream([createAudioTrack()]))
    const enumerateDevices = vi.fn(async () => [
      { kind: 'videoinput', deviceId: 'camera-1', label: 'Camera' },
    ])

    Object.defineProperty(navigator, 'mediaDevices', {
      value: { enumerateDevices, getUserMedia },
      configurable: true,
    })

    const { useVideoCallNew } = await importUseVideoCallNew()

    await expect(useVideoCallNew().startCall(42, 'voice')).rejects.toThrow('未检测到麦克风')

    expect(getUserMedia).not.toHaveBeenCalled()
    expect(send).not.toHaveBeenCalled()
  })

  it('does not send voice call signaling when getUserMedia returns no live audio track', async () => {
    const endedTrack = createAudioTrack('ended')
    const getUserMedia = vi.fn(async () => createStream([endedTrack]))
    const enumerateDevices = vi.fn(async () => [
      { kind: 'audioinput', deviceId: 'mic-1', label: 'Microphone' },
    ])

    Object.defineProperty(navigator, 'mediaDevices', {
      value: { enumerateDevices, getUserMedia },
      configurable: true,
    })

    const { useVideoCallNew } = await importUseVideoCallNew()

    await expect(useVideoCallNew().startCall(42, 'voice')).rejects.toThrow('未检测到麦克风')

    expect(endedTrack.stop).toHaveBeenCalled()
    expect(send).not.toHaveBeenCalled()
  })

  it('sends voice call signaling when an audio input provides a live audio track', async () => {
    const getUserMedia = vi.fn(async () => createStream([createAudioTrack()]))
    const enumerateDevices = vi.fn(async () => [
      { kind: 'audioinput', deviceId: 'mic-1', label: 'Microphone' },
    ])

    Object.defineProperty(navigator, 'mediaDevices', {
      value: { enumerateDevices, getUserMedia },
      configurable: true,
    })

    const { useVideoCallNew } = await importUseVideoCallNew()

    await useVideoCallNew().startCall(42, 'voice')

    expect(getUserMedia).toHaveBeenCalledWith({ video: false, audio: true })
    expect(send).toHaveBeenCalledTimes(2)
    expect(JSON.parse(send.mock.calls[0][0]).type).toBe('webrtc.offer')
    expect(JSON.parse(send.mock.calls[1][0])).toMatchObject({
      type: 'call.start',
      data: {
        target_user_id: 42,
        call_type: 'voice',
      },
    })
  })
})
