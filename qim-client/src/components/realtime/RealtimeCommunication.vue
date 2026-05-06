<template>
  <Teleport to="body">
    <ScreenShareSimple
      v-if="showScreenShare"
      ref="screenShareRef"
      :receiver-id="screenShareReceiverId"
      :conversation-id="screenShareConversationId"
      :sender-name="remoteScreenUserName"
      @screen-share-start="handleScreenShareStart"
      @screen-share-stop="handleScreenShareStop"
    />

    <CallOverlay
      v-if="showCallOverlay"
      ref="callOverlayRef"
      :receiver-id="callReceiverId"
      :conversation-id="callConversationId"
      :sender-name="remoteCallUserName"
      @call-start="handleCallStart"
      @call-stop="handleCallStop"
    />
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, provide, nextTick } from 'vue'
import ScreenShareSimple from '../shared/ScreenShareSimple.vue'
import CallOverlay from '../shared/CallOverlay.vue'
import { useRealtimeMessaging } from '../../composables/useRealtimeMessaging'
import { getCurrentUser } from '../../utils/user'
import QMessage from '../../utils/qmessage'

interface Props {
  currentConversation?: {
    id: string | number
    type: string
    members?: Array<{ id: string | number; name: string; avatar: string }>
    name?: string
    avatar?: string
  } | null
  currentUserId?: string | number
  conversations?: Array<any>
  onConversationSwitch?: (conversation: any) => void
  onSendMessage?: (message: any) => void
}

const props = withDefaults(defineProps<Props>(), {
  currentConversation: null,
  currentUserId: '',
  conversations: () => [],
  onConversationSwitch: () => {},
  onSendMessage: () => {}
})

const emit = defineEmits<{
  'screen-share-start': [data: { conversationId: number; requester_id: number }]
  'screen-share-stop': [data: { conversationId: number }]
  'screen-share-data': [data: { conversationId: number; data: string }]
  'screen-share-request': [data: any]
  'call-state-change': [status: string]
}>()

const currentUser = computed(() => getCurrentUser())

const realtimeMessaging = useRealtimeMessaging()
const { screenShare } = realtimeMessaging

provide('screenShare', screenShare)

const showScreenShare = ref(false)
const screenShareReceiverId = ref<number | undefined>(undefined)
const screenShareConversationId = ref<number | undefined>(undefined)
const remoteScreenUserName = ref('')
const incomingRequestData = ref<any>(null)

const screenShareRef = ref<InstanceType<typeof ScreenShareSimple>>()

const showCallOverlay = ref(false)
const callReceiverId = ref<number | undefined>(undefined)
const callConversationId = ref<number | undefined>(undefined)
const remoteCallUserName = ref('')
const callOverlayRef = ref<InstanceType<typeof CallOverlay>>()

const startScreenShare = async () => {
  if (!props.currentConversation) {
    QMessage.warning('请先选择一个会话')
    return
  }

  const user = getCurrentUser()
  if (!user || !user.id) {
    QMessage.warning('用户信息未加载，无法使用屏幕共享')
    return
  }

  const conv = props.currentConversation
  if (conv?.type === 'single' && conv.members && conv.members.length === 2) {
    const otherMember = conv.members.find(m => String(m.id) !== String(user.id))
    if (otherMember) {
      screenShareReceiverId.value = Number(otherMember.id)
      screenShareConversationId.value = Number(conv.id)
    }
  }

  showScreenShare.value = true

  await nextTick()

  if (screenShareRef.value?.initiateShare) {
    try {
      await screenShareRef.value.initiateShare()
    } catch (error) {
      console.error('[RealtimeCommunication] 屏幕共享失败:', error)
      showScreenShare.value = false
    }
  }
}

const handleScreenShareStart = (data: any) => {
  console.log('[RealtimeCommunication] 收到屏幕共享开始', data)
  emit('screen-share-start', {
    conversationId: Number(props.currentConversation?.id) || 0,
    requester_id: Number(currentUser.value?.id) || 0
  })
}

const handleScreenShareStop = () => {
  emit('screen-share-stop', { conversationId: Number(props.currentConversation?.id) || 0 })
  showScreenShare.value = false
  incomingRequestData.value = null
}

const startCall = async (type: 'voice' | 'video') => {
  if (!props.currentConversation) {
    QMessage.warning('请先选择一个会话')
    return
  }

  const user = getCurrentUser()
  if (!user || !user.id) {
    QMessage.warning('用户信息未加载，无法使用通话功能')
    return
  }

  const conv = props.currentConversation
  if (conv?.type === 'single' && conv.members && conv.members.length === 2) {
    const otherMember = conv.members.find(m => String(m.id) !== String(user.id))
    if (otherMember) {
      callReceiverId.value = Number(otherMember.id)
      callConversationId.value = Number(conv.id)
    }
  }

  showCallOverlay.value = true

  await nextTick()

  if (callOverlayRef.value?.initiateCall) {
    try {
      await callOverlayRef.value.initiateCall(type)
    } catch (error) {
      console.error('[RealtimeCommunication] 发起通话失败:', error)
      showCallOverlay.value = false
    }
  }
}

const handleCallStart = (data: any) => {
  console.log('[RealtimeCommunication] 通话开始', data)
  emit('call-state-change', 'connected')
}

const handleCallStop = () => {
  console.log('[RealtimeCommunication] 通话结束')
  showCallOverlay.value = false
  emit('call-state-change', 'idle')
}

const getMemberInfo = (fromUserId: number) => {
  const conv = props.conversations.find(c => {
    const members = c.members as any[]
    return members?.some(m => m.id == fromUserId) && c.type !== 'group'
  })
  if (conv) {
    const member = (conv.members as any[])?.find(m => m.id == fromUserId)
    if (member) {
      return {
        name: member.name || member.nickname || '未知用户',
        avatar: member.avatar || ''
      }
    }
  }
  return { name: '未知用户', avatar: '' }
}

const handleWebRTCOffer = async (data: any) => {
  console.log('[RealtimeCommunication] 收到 webrtc_offer', data)

  const fromUserId = data.from_user_id
  
  console.log('[RealtimeCommunication] webrtc_offer from:', fromUserId, 'currentUserId:', props.currentUserId)
  
  if (fromUserId == props.currentUserId) {
    console.log('[RealtimeCommunication] 忽略自己发送的 webrtc_offer')
    return
  }

  const mediaType = data.media_type || data.share_type || data.call_type
  console.log('[RealtimeCommunication] 媒体类型:', mediaType)

  if (mediaType === 'screen') {
    console.log('[RealtimeCommunication] 屏幕共享 offer - 转发给 ScreenShareSimple 处理')

    const memberInfo = getMemberInfo(fromUserId)
    remoteScreenUserName.value = memberInfo.name

    showScreenShare.value = true

    await nextTick()

    if (screenShareRef.value?.handleIncomingOffer) {
      screenShareRef.value.handleIncomingOffer(data.signal, fromUserId)
    }
  } else if (mediaType === 'video' || mediaType === 'audio') {
    console.log('[RealtimeCommunication] 处理视频/语音通话 offer')

    const memberInfo = getMemberInfo(fromUserId)
    remoteCallUserName.value = memberInfo.name

    showCallOverlay.value = true

    await nextTick()

    if (callOverlayRef.value?.handleIncomingOffer) {
      callOverlayRef.value.handleIncomingOffer(data.signal, fromUserId, mediaType === 'audio' ? 'voice' : 'video')
    }
  } else {
    console.warn('[RealtimeCommunication] 未知的媒体类型:', mediaType)
  }
}

const handleWebRTCAnswer = async (data: any) => {
  console.log('[RealtimeCommunication] 收到 webrtc_answer', data)
  await realtimeMessaging.handleWebRTCAnswer(data)
}

const handleWebRTCIceCandidate = async (data: any) => {
  console.log('[RealtimeCommunication] 收到 webrtc_ice_candidate', data)
  await realtimeMessaging.handleWebRTCIceCandidate(data)
}

const handleScreenShareRequest = (data: any) => {
  console.log('[RealtimeCommunication] 收到屏幕共享请求', data)

  const fromUserId = data.from_user_id || data.user_id
  const memberInfo = getMemberInfo(fromUserId)
  remoteScreenUserName.value = memberInfo.name

  incomingRequestData.value = data

  showScreenShare.value = true

  nextTick(() => {
    if (screenShareRef.value?.handleIncomingRequest) {
      screenShareRef.value.handleIncomingRequest(data)
    }
  })
}

const handleScreenShareStopGlobal = (data: any) => {
  console.log('[RealtimeCommunication] 收到屏幕共享停止', data)
  realtimeMessaging.handleScreenShareStop(data)
  showScreenShare.value = false
  incomingRequestData.value = null
}

const handleScreenShareMessage = (type: string, data: any) => {
  console.log('[RealtimeCommunication] 收到屏幕共享消息', type, data)
}

const handleScreenShareAccepted = async (data: any) => {
  console.log('[RealtimeCommunication] 收到屏幕共享接受', data)
  console.log('[RealtimeCommunication] screenShareRef.value:', !!screenShareRef.value)
  console.log('[RealtimeCommunication] screenShareRef.value.stopWaitingAccept:', !!screenShareRef.value?.stopWaitingAccept)

  QMessage.success('对方已接受屏幕共享')

  if (screenShareRef.value) {
    console.log('[RealtimeCommunication] Calling stopWaitingAccept')
    await screenShareRef.value.stopWaitingAccept()
    console.log('[RealtimeCommunication] stopWaitingAccept completed')
  }
}

const handleScreenShareRejected = (data: any) => {
  console.log('[RealtimeCommunication] 收到屏幕共享拒绝', data)

  QMessage.warning('对方拒绝了屏幕共享')

  screenShare.cancelPendingRequest()
  showScreenShare.value = false
}

const handleRealtimeSessionCreated = (data: any) => {
  console.log('[RealtimeCommunication] 收到实时会话创建', data)
}

const handleVideoCallSignaling = (message: { type: string; data: any }) => {
  console.log('[RealtimeCommunication] 收到视频通话信令', message)

  switch (message.type) {
    case 'call_invite':
    case 'call.start': {
      const fromUserId = message.data.from_user_id || message.data.user_info?.id
      
      console.log('[RealtimeCommunication] call_invite from:', fromUserId, 'currentUserId:', props.currentUserId)
      console.log('[RealtimeCommunication] call_invite data:', message.data)
      console.log('[RealtimeCommunication] call_type:', message.data.call_type)
      
      if (fromUserId == props.currentUserId) {
        console.log('[RealtimeCommunication] 忽略自己发送的 call_invite')
        return
      }
      
      showCallOverlay.value = true
      const memberInfo = getMemberInfo(fromUserId)
      remoteCallUserName.value = memberInfo.name
      const callType = message.data.call_type === 'voice' || message.data.call_type === 'audio' ? 'voice' : 'video'

      console.log('[RealtimeCommunication] call_invite callType:', callType, 'hasSignal:', !!message.data.signal)

      nextTick(() => {
        if (message.data.signal) {
          if (callOverlayRef.value?.handleIncomingOffer) {
            callOverlayRef.value.handleIncomingOffer(message.data.signal, fromUserId, callType)
          }
        } else {
          if (callOverlayRef.value?.showIncomingCallUI) {
            callOverlayRef.value.showIncomingCallUI(fromUserId, callType)
          }
        }
      })
      break
    }
    case 'call_accept':
    case 'call.answer':
      break
    case 'call_end':
    case 'call_reject':
    case 'call.end':
      console.log('[RealtimeCommunication] 对方挂断通话，清理本地资源')
      console.log('[RealtimeCommunication] callOverlayRef.value:', !!callOverlayRef.value)
      console.log('[RealtimeCommunication] showCallOverlay:', showCallOverlay.value)
      
      if (callOverlayRef.value && typeof callOverlayRef.value.handleRemoteEndCall === 'function') {
        console.log('[RealtimeCommunication] 调用 handleRemoteEndCall')
        callOverlayRef.value.handleRemoteEndCall()
      } else {
        console.log('[RealtimeCommunication] callOverlayRef 不可用，直接关闭界面')
        showCallOverlay.value = false
      }
      break
  }
}

onMounted(() => {
  console.log('[RealtimeCommunication] 组件已挂载')
})

defineExpose({
  startScreenShare,
  startCall,
  handleWebRTCOffer,
  handleWebRTCAnswer,
  handleWebRTCIceCandidate,
  handleScreenShareStart,
  handleScreenShareMessage,
  handleScreenShareRequest,
  handleScreenShareAccepted,
  handleScreenShareRejected,
  handleScreenShareStop: handleScreenShareStopGlobal,
  handleRealtimeSessionCreated,
  handleVideoCallSignaling,
  screenShareRef,
  callOverlayRef
})
</script>

<style scoped>
</style>
