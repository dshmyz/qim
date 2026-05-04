import { ref, computed, onUnmounted } from 'vue'
import type { StreamState, MediaStreamSourceType } from '@/types/realtime'

export function useMediaStream(source: MediaStreamSourceType) {
  const stream = ref<MediaStream | null>(null)
  const state = ref<StreamState>('stopped')
  
  const constraints = computed(() => {
    switch (source) {
      case 'camera':
        return { video: true, audio: false }
      case 'microphone':
        return { video: false, audio: true }
      case 'screen':
        return { video: true, audio: false }
      default:
        return { video: false, audio: false }
    }
  })
  
  const start = async () => {
    if (stream.value) {
      console.warn('Stream already exists, stopping old stream')
      stop()
    }
    
    state.value = 'starting'
    console.log(`Starting ${source} stream with constraints:`, constraints.value)
    
    try {
      if (source === 'screen') {
        stream.value = await navigator.mediaDevices.getDisplayMedia(constraints.value)
      } else {
        stream.value = await navigator.mediaDevices.getUserMedia(constraints.value)
      }
      
      state.value = 'active'
      console.log(`${source} stream started successfully`)
      console.log('Stream tracks:', stream.value.getTracks().map(t => ({ kind: t.kind, id: t.id, label: t.label })))
      
      stream.value.getTracks().forEach(track => {
        track.onended = () => {
          console.log(`Track ${track.kind} ended`)
          if (stream.value) {
            const remainingTracks = stream.value.getTracks().filter(t => t.readyState === 'live')
            if (remainingTracks.length === 0) {
              console.log('All tracks ended, stopping stream')
              stop()
            }
          }
        }
      })
    } catch (error) {
      console.error(`Failed to start ${source} stream:`, error)
      state.value = 'stopped'
      throw error
    }
  }
  
  const stop = () => {
    if (stream.value) {
      console.log(`Stopping ${source} stream`)
      stream.value.getTracks().forEach(track => {
        console.log(`Stopping track: ${track.kind} (${track.id})`)
        track.stop()
      })
      stream.value = null
      state.value = 'stopped'
    }
  }
  
  const toggleMute = (kind: 'audio' | 'video') => {
    if (!stream.value) {
      console.warn('No stream to toggle mute')
      return
    }
    
    const tracks = stream.value.getTracks().filter(t => t.kind === kind)
    tracks.forEach(track => {
      track.enabled = !track.enabled
      console.log(`Track ${track.kind} (${track.id}) enabled: ${track.enabled}`)
    })
  }
  
  const mute = (kind: 'audio' | 'video') => {
    if (!stream.value) {
      console.warn('No stream to mute')
      return
    }
    
    const tracks = stream.value.getTracks().filter(t => t.kind === kind)
    tracks.forEach(track => {
      track.enabled = false
      console.log(`Track ${track.kind} (${track.id}) muted`)
    })
  }
  
  const unmute = (kind: 'audio' | 'video') => {
    if (!stream.value) {
      console.warn('No stream to unmute')
      return
    }
    
    const tracks = stream.value.getTracks().filter(t => t.kind === kind)
    tracks.forEach(track => {
      track.enabled = true
      console.log(`Track ${track.kind} (${track.id}) unmuted`)
    })
  }
  
  const isMuted = (kind: 'audio' | 'video'): boolean => {
    if (!stream.value) {
      return false
    }
    
    const tracks = stream.value.getTracks().filter(t => t.kind === kind)
    return tracks.length > 0 && tracks.every(t => !t.enabled)
  }
  
  const hasTrack = (kind: 'audio' | 'video'): boolean => {
    if (!stream.value) {
      return false
    }
    
    return stream.value.getTracks().some(t => t.kind === kind)
  }
  
  const getTrack = (kind: 'audio' | 'video'): MediaStreamTrack | null => {
    if (!stream.value) {
      return null
    }
    
    return stream.value.getTracks().find(t => t.kind === kind) || null
  }
  
  const replaceTrack = async (newTrack: MediaStreamTrack, kind: 'audio' | 'video') => {
    if (!stream.value) {
      console.warn('No stream to replace track')
      return
    }
    
    const oldTrack = getTrack(kind)
    if (oldTrack) {
      oldTrack.stop()
      stream.value.removeTrack(oldTrack)
    }
    
    stream.value.addTrack(newTrack)
    console.log(`Replaced ${kind} track`)
  }
  
  onUnmounted(() => {
    stop()
  })
  
  return {
    stream,
    state,
    constraints,
    start,
    stop,
    toggleMute,
    mute,
    unmute,
    isMuted,
    hasTrack,
    getTrack,
    replaceTrack
  }
}
