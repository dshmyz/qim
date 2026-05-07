import { ref } from 'vue'
import { useConnection } from './useConnection'
import { useMediaStream } from './useMediaStream'
import { useSignaling } from './useSignaling'
import type { MediaType, MediaStreamSourceType } from '@/types/realtime'

export interface InitiateOptions {
  stream?: MediaStream
  needVideo?: boolean
  offerMediaType?: MediaType
}

export interface ReceiveOptions {
  needVideo?: boolean
  answerMediaType?: MediaType
}

function getSourceFromMediaType(mediaType: MediaType): MediaStreamSourceType {
  switch (mediaType) {
    case 'screen':
      return 'screen'
    case 'video':
      return 'camera'
    case 'audio':
      return 'microphone'
  }
}

const rtcInstances = new Map<MediaType, ReturnType<typeof createRealtimeCommunication>>()

export function useRealtimeCommunication(mediaType: MediaType) {
  if (!rtcInstances.has(mediaType)) {
    rtcInstances.set(mediaType, createRealtimeCommunication(mediaType))
  }
  return rtcInstances.get(mediaType)!
}

function createRealtimeCommunication(mediaType: MediaType) {
  const connection = useConnection()
  const mediaStream = useMediaStream(getSourceFromMediaType(mediaType))
  const signaling = useSignaling()

  const remoteStream = ref<MediaStream | null>(null)
  const targetUserId = ref<number | null>(null)
  const localOffer = ref<RTCSessionDescriptionInit | null>(null)
  const iceCandidateCache: RTCIceCandidateInit[] = []

  const flushIceCandidates = async () => {
    const pc = connection.peerConnection.value
    if (!pc || !pc.remoteDescription) {
      console.log(`[RTC] Cannot flush ICE candidates: peerConnection=${!!pc}, remoteDescription=${!!pc?.remoteDescription}`)
      return
    }

    if (iceCandidateCache.length === 0) {
      console.log(`[RTC] No cached ICE candidates to flush for ${mediaType}`)
      return
    }

    console.log(`[RTC] Flushing ${iceCandidateCache.length} cached ICE candidates for ${mediaType}`)

    while (iceCandidateCache.length > 0) {
      const candidate = iceCandidateCache.shift()!
      try {
        const iceCandidate = new RTCIceCandidate({
          candidate: candidate.candidate || '',
          sdpMid: candidate.sdpMid || '',
          sdpMLineIndex: candidate.sdpMLineIndex ?? 0
        })
        await pc.addIceCandidate(iceCandidate)
        console.log(`[RTC] Cached ICE candidate added successfully`)
      } catch (error) {
        console.error(`[RTC] Failed to add cached ICE candidate:`, error)
      }
    }
  }

  const addIceCandidate = async (candidate: RTCIceCandidateInit) => {
    if (!candidate || !candidate.candidate) {
      console.log(`[RTC] Invalid ICE candidate, skipping:`, candidate)
      return
    }

    const pc = connection.peerConnection.value

    if (!pc) {
      console.log(`[RTC] peerConnection not created, caching ICE candidate`)
      iceCandidateCache.push(candidate)
      return
    }

    if (pc.remoteDescription) {
      try {
        const iceCandidate = new RTCIceCandidate({
          candidate: candidate.candidate || '',
          sdpMid: candidate.sdpMid || '',
          sdpMLineIndex: candidate.sdpMLineIndex ?? 0
        })
        await pc.addIceCandidate(iceCandidate)
        console.log(`[RTC] ICE candidate added successfully`)
      } catch (error) {
        console.error(`[RTC] Failed to add ICE candidate:`, error)
      }
    } else {
      console.log(`[RTC] remoteDescription not set, caching ICE candidate`)
      iceCandidateCache.push(candidate)
    }
  }

  const resolveStream = async (options?: InitiateOptions): Promise<MediaType> => {
    const offerMediaType = options?.offerMediaType ?? mediaType

    if (options?.stream) {
      mediaStream.stream.value = options.stream
      return offerMediaType
    }

    if (options?.needVideo !== undefined) {
      const constraints = options.needVideo ? { video: true, audio: true } : { video: false, audio: true }
      const newStream = await navigator.mediaDevices.getUserMedia(constraints)
      mediaStream.stream.value = newStream
      console.log(`[RTC] Got media stream:`, newStream.getTracks().map(t => `${t.kind}:${t.readyState}`))
      return options.needVideo ? 'video' : 'audio'
    }

    await mediaStream.start()
    return offerMediaType
  }

  const addLocalTracks = () => {
    if (mediaStream.stream.value) {
      mediaStream.stream.value.getTracks().forEach(track => {
        connection.addTrack(track, mediaStream.stream.value!)
      })
    }
  }

  const initiate = async (userId: number, options?: InitiateOptions) => {
    console.log(`[RTC] Initiating ${mediaType} connection to user ${userId}`, options ? `with options: stream=${!!options.stream}, needVideo=${options.needVideo}` : '')

    targetUserId.value = userId

    try {
      await connection.createConnection()
      setupIceCandidateHandler()
      setupRemoteStreamHandler()

      const actualMediaType = await resolveStream(options)
      addLocalTracks()

      const offer = await connection.createOffer()
      await connection.setLocalDescription(offer!)
      localOffer.value = offer!

      signaling.sendOffer(userId, actualMediaType, offer!)

      console.log(`[RTC] Offer sent to user ${userId}, mediaType: ${actualMediaType}`)
    } catch (error) {
      console.error(`[RTC] Failed to initiate connection:`, error)
      cleanup()
      throw error
    }
  }

  const resolveReceiveStream = async (options?: ReceiveOptions): Promise<MediaType> => {
    const answerMediaType = options?.answerMediaType ?? mediaType

    if (options?.needVideo !== undefined) {
      const constraints = options.needVideo ? { video: true, audio: true } : { video: false, audio: true }
      const newStream = await navigator.mediaDevices.getUserMedia(constraints)
      mediaStream.stream.value = newStream
      console.log(`[RTC] receiveWithMedia got media stream:`, newStream.getTracks().map(t => `${t.kind}:${t.readyState}`))
      return options.needVideo ? 'video' : 'audio'
    }

    return answerMediaType
  }

  const receive = async (signal: RTCSessionDescriptionInit, fromUserId: number, options?: ReceiveOptions) => {
    console.log(`[RTC] Receiving ${mediaType} connection from user ${fromUserId}`, options ? `with options: needVideo=${options.needVideo}` : '')

    targetUserId.value = fromUserId

    try {
      await connection.createConnection()
      setupIceCandidateHandler()
      setupRemoteStreamHandler()

      const actualMediaType = await resolveReceiveStream(options)
      addLocalTracks()

      await connection.setRemoteDescription(signal)
      await flushIceCandidates()

      const answer = await connection.createAnswer()
      await connection.setLocalDescription(answer!)

      signaling.sendAnswer(fromUserId, actualMediaType, answer!)

      console.log(`[RTC] Answer sent to user ${fromUserId}, mediaType: ${actualMediaType}`)
      return answer
    } catch (error) {
      console.error(`[RTC] Failed to receive connection:`, error)
      cleanup()
      throw error
    }
  }

  const startMedia = async (needVideo: boolean = true) => {
    console.log(`[RTC] Starting media stream for ${mediaType}, needVideo: ${needVideo}`)

    try {
      const constraints = needVideo ? { video: true, audio: true } : { video: false, audio: true }
      const newStream = await navigator.mediaDevices.getUserMedia(constraints)
      mediaStream.stream.value = newStream

      const pc = connection.peerConnection.value
      if (pc && mediaStream.stream.value) {
        mediaStream.stream.value.getTracks().forEach(track => {
          pc.addTrack(track, mediaStream.stream.value!)
        })
      }

      console.log(`[RTC] Media stream started successfully`)
    } catch (error) {
      console.error(`[RTC] Failed to start media stream:`, error)
      throw error
    }
  }

  const handleAnswer = async (signal: RTCSessionDescriptionInit) => {
    console.log(`[RTC] Handling answer for ${mediaType}`)

    try {
      const signalingState = connection.getSignalingState()
      console.log(`[RTC] Current signaling state:`, signalingState)

      if (signalingState !== 'have-local-offer') {
        console.warn(`[RTC] Invalid signaling state for answer: ${signalingState}`)
        return
      }

      await connection.setRemoteDescription(signal)
      await flushIceCandidates()

      console.log(`[RTC] Answer processed successfully`)
    } catch (error) {
      console.error(`[RTC] Failed to handle answer:`, error)
      throw error
    }
  }

  const handleIceCandidate = async (candidate: RTCIceCandidateInit) => {
    console.log(`[RTC] Handling ICE candidate for ${mediaType}`)
    await addIceCandidate(candidate)
  }

  const setupIceCandidateHandler = () => {
    const pc = connection.peerConnection.value
    if (pc) {
      pc.onicecandidate = (event) => {
        if (event.candidate) {
          console.log(`[RTC] Generated ICE candidate for ${mediaType}`)
          if (targetUserId.value) {
            signaling.sendIceCandidate(targetUserId.value, mediaType, event.candidate)
          } else {
            console.error(`[RTC] Cannot send ICE candidate: targetUserId is null`)
          }
        } else {
          console.log(`[RTC] ICE candidate gathering complete for ${mediaType}`)
        }
      }
    } else {
      console.error(`[RTC] Cannot setup ICE candidate handler: peerConnection is null`)
    }
  }

  const setupRemoteStreamHandler = () => {
    const pc = connection.peerConnection.value
    if (pc) {
      console.log(`[RTC] Setting up remote stream handler for ${mediaType}`)
      pc.ontrack = (event) => {
        console.log(`[RTC] Received remote track:`, event.track.kind, event.track.readyState)
        console.log(`[RTC] Track event streams:`, event.streams.length)

        if (event.streams && event.streams.length > 0) {
          remoteStream.value = event.streams[0]
          console.log(`[RTC] Set remote stream from event.streams[0]`)
        } else {
          if (!remoteStream.value) {
            remoteStream.value = new MediaStream()
          }
          remoteStream.value.addTrack(event.track)
          console.log(`[RTC] Added track to new/existing remote stream`)
        }

        event.track.onended = () => {
          console.log(`[RTC] Remote track ended:`, event.track.kind)
          if (remoteStream.value) {
            remoteStream.value.removeTrack(event.track)
          }
        }
      }
    } else {
      console.error(`[RTC] PeerConnection not available for setting up remote stream handler`)
    }
  }

  const cleanup = () => {
    console.log(`[RTC] Cleaning up ${mediaType} connection`)
    iceCandidateCache.length = 0
    mediaStream.stop()
    connection.close()

    if (remoteStream.value) {
      remoteStream.value.getTracks().forEach(track => {
        track.stop()
      })
      remoteStream.value = null
    }

    targetUserId.value = null
    localOffer.value = null
  }

  const close = () => {
    cleanup()
    rtcInstances.delete(mediaType)
  }

  return {
    state: connection.state,
    localStream: mediaStream.stream,
    remoteStream,
    targetUserId,
    localOffer,
    initiate,
    receive,
    startMedia,
    handleAnswer,
    handleIceCandidate,
    close,
    mediaStream,
    connection
  }
}
