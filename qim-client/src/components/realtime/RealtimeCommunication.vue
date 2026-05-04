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

    <CallModal
      :visible="showCallModal"
      :call-type="callType"
      :call-status="callStatus"
      :avatar="callAvatar || ''"
      :name="callName || '未知用户'"
      @reject-call="handleRejectCall"
      @answer-call="handleAnswerCall"
      @end-call="handleEndCall"
      @close-call-modal="handleCallModalClose"
    />
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, provide } from 'vue'
import ScreenShareSimple from '../shared/ScreenShareSimple.vue'
import CallModal from '../chat/CallModal.vue'
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
const { screenShare, videoCall } = realtimeMessaging

provide('screenShare', screenShare)

const showScreenShare = ref(false)
const screenShareReceiverId = ref<number | undefined>(undefined)
const screenShareConversationId = ref<number | undefined>(undefined)
const remoteScreenUserName = ref('')

const screenShareRef = ref<InstanceType<typeof ScreenShareSimple>>()

const showCallModal = ref(false)
const callType = ref<'voice' | 'video'>('video')
const callStatus = ref<'idle' | 'calling' | 'connecting' | 'connected' | 'ended'>('idle')
const callAvatar = ref('')
const callName = ref('')

const startScreenShare = () => {
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
      console.log('[RealtimeCommunication] 设置接收者 ID:', otherMember.id)
    }
  }
  
  showScreenShare.value = true
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
}

const handleRejectCall = async () => {
  try {
    // TODO: 需要获取 fromUserId
    // await videoCall.rejectCall(fromUserId)
    showCallModal.value = false
    console.log('[RealtimeCommunication] 拒绝通话')
  } catch (error) {
    console.error('[RealtimeCommunication] 拒绝通话失败:', error)
  }
}

const handleAnswerCall = async () => {
  try {
    // TODO: 需要获取 signal 和 fromUserId
    // await videoCall.acceptCall(signal, fromUserId)
    console.log('[RealtimeCommunication] 接听通话')
  } catch (error) {
    console.error('[RealtimeCommunication] 接听通话失败:', error)
  }
}

const handleEndCall = async () => {
  try {
    await videoCall.endCall()
    showCallModal.value = false
  } catch (error) {
    console.error('[RealtimeCommunication] 结束通话失败:', error)
  }
}

const handleCallModalClose = async () => {
  try {
    await videoCall.endCall()
    showCallModal.value = false
  } catch (error) {
    console.error('[RealtimeCommunication] 关闭通话模态框失败:', error)
    showCallModal.value = false
  }
}

const handleWebRTCOffer = async (data: any) => {
  console.log('[RealtimeCommunication] 收到 webrtc_offer', data)
  
  const mediaType = data.media_type || data.share_type || data.call_type
  console.log('[RealtimeCommunication] 媒体类型:', mediaType)
  
  if (mediaType === 'screen') {
    console.log('[RealtimeCommunication] 处理屏幕共享 offer')
    
    const fromUserId = data.from_user_id
    const conv = props.conversations.find(c => {
      const members = c.members as any[]
      return members?.some(m => m.id == fromUserId) && c.type !== 'group'
    })
    
    if (conv) {
      const member = (conv.members as any[])?.find(m => m.id == fromUserId)
      if (member && (member.name || member.nickname)) {
        remoteScreenUserName.value = member.name || member.nickname
      }
    }
    
    console.log('[RealtimeCommunication] 设置 showScreenShare = true')
    showScreenShare.value = true
    console.log('[RealtimeCommunication] showScreenShare:', showScreenShare.value)
    
    await realtimeMessaging.handleWebRTCOffer(data)
  } else if (mediaType === 'video' || mediaType === 'audio') {
    console.log('[RealtimeCommunication] 处理视频/语音通话 offer')
    
    // 显示通话模态框
    showCallModal.value = true
    callStatus.value = 'calling'
    callType.value = mediaType === 'audio' ? 'voice' : 'video'
    
    // 获取用户信息
    const fromUserId = data.from_user_id
    const conv = props.conversations.find(c => {
      const members = c.members as any[]
      return members?.some(m => m.id == fromUserId) && c.type !== 'group'
    })
    
    if (conv) {
      const member = (conv.members as any[])?.find(m => m.id == fromUserId)
      if (member) {
        callAvatar.value = member.avatar || ''
        callName.value = member.name || member.nickname || '未知用户'
      }
    }
    
    await realtimeMessaging.handleWebRTCOffer(data)
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
  realtimeMessaging.handleScreenShareRequest(data)
}

const handleScreenShareStopGlobal = (data: any) => {
  console.log('[RealtimeCommunication] 收到屏幕共享停止', data)
  realtimeMessaging.handleScreenShareStop(data)
  showScreenShare.value = false
}

const handleScreenShareMessage = (type: string, data: any) => {
  console.log('[RealtimeCommunication] 收到屏幕共享消息', type, data)
  // TODO: 处理屏幕共享数据
}

const handleScreenShareAccepted = (data: any) => {
  console.log('[RealtimeCommunication] 收到屏幕共享接受', data)
  // TODO: 处理屏幕共享接受
}

const handleScreenShareRejected = (data: any) => {
  console.log('[RealtimeCommunication] 收到屏幕共享拒绝', data)
  // TODO: 处理屏幕共享拒绝
}

const handleRealtimeSessionCreated = (data: any) => {
  console.log('[RealtimeCommunication] 收到实时会话创建', data)
  // TODO: 处理实时会话创建
}

const handleVideoCallSignaling = (message: { type: string; data: any }) => {
  console.log('[RealtimeCommunication] 收到视频通话信令', message)
  
  switch (message.type) {
    case 'call_invite':
      showCallModal.value = true
      callStatus.value = 'calling'
      callType.value = message.data.call_type === 'voice' ? 'voice' : 'video'
      if (message.data.user_info) {
        callAvatar.value = message.data.user_info.avatar || ''
        callName.value = message.data.user_info.name || '未知用户'
      }
      break
    case 'call_accept':
      callStatus.value = 'connecting'
      break
    case 'call_end':
    case 'call_reject':
      showCallModal.value = false
      callStatus.value = 'idle'
      break
  }
}

onMounted(() => {
  console.log('[RealtimeCommunication] 组件已挂载')
})

defineExpose({
  startScreenShare,
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
  screenShareRef
})
</script>

<style scoped>
</style>
