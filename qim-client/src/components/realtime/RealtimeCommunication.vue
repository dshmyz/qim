<template>
  <Teleport to="body">
    <!-- 屏幕共享组件 -->
    <ScreenShare
      v-if="showScreenShare"
      ref="screenShareRef"
      :receiver-id="screenShareReceiverId"
      :sender-id="remoteScreenUserId"
      :sender-name="remoteScreenUserName"
      :conversation-id="screenShareConversationId"
      @vue:mounted="onScreenShareMounted"
      @screen-share-start="handleScreenShareStart"
      @screen-share-stop="handleScreenShareStop"
      @screen-share-join="handleScreenShareJoin"
      @screen-share-leave="handleScreenShareLeave"
    />

    <!-- 通话模态框 -->
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
import { ref, computed, provide, watch, shallowRef, nextTick } from 'vue'
import ScreenShare from '../shared/ScreenShare.vue'
import CallModal from '../chat/CallModal.vue'
import { useScreenShare, consumeCachedOffer } from '../../composables/useScreenShare'
import { useVideoCall } from '../../composables/useVideoCall'
import { getCurrentUser } from '../../utils/user'
import QMessage from '../../utils/qmessage'
// @ts-ignore - WebRTC module has no type declarations
import { screenShareSender } from '../../utils/webrtc'

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
  onSendMessage?: (type: string, data: any) => void
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

// ==========================================
// 屏幕共享相关状态
// ==========================================
const showScreenShare = ref(false)
const screenShareReceiverId = ref<number | undefined>(undefined)
const remoteScreenUserId = ref<number | null>(null)
const remoteScreenUserName = ref('')
const screenShareConversationId = ref<string | number | undefined>(undefined)

// 创建响应式的 conversation ref，跟随 props.currentConversation 更新
const screenShareConversation = ref(props.currentConversation)

watch(
  () => props.currentConversation,
  (newConv) => {
    screenShareConversation.value = newConv
  }
)

const screenShare = useScreenShare(screenShareConversation)
const {
  screenShareComponent,
  setOnShowScreenShareRequest,
  handleScreenShareMessage,
  handleWebRTCOffer,
  handleWebRTCAnswer,
  handleWebRTCIceCandidate,
  handleScreenShareRequest,
  handleScreenShareAccepted,
  handleScreenShareRejected,
  startScreenShare: originalStartScreenShare,
  cleanupScreenShare
} = screenShare

/**
 * 开始屏幕共享
 * 注意：先设置 showScreenShare 为 true 以确保组件挂载，再调用原始方法
 */
const startScreenShare = () => {
  if (!screenShareConversation.value) {
    QMessage.warning('请先选择一个会话')
    return
  }
  
  const user = getCurrentUser()
  if (!user || !user.id) {
    QMessage.warning('用户信息未加载，无法使用屏幕共享')
    return
  }
  
  if (screenShareSender.getIsSharing()) {
    QMessage.warning('屏幕共享已在运行')
    return
  }
  
  // 从会话中获取对方用户 ID
  const conv = screenShareConversation.value
  if (conv?.type === 'single' && conv.members && conv.members.length === 2) {
    const otherMember = conv.members.find(m => String(m.id) !== String(user.id))
    if (otherMember) {
      screenShareReceiverId.value = Number(otherMember.id)
      console.log('RealtimeCommunication: 设置接收者 ID:', otherMember.id)
    }
  }
  
  // 先显示 ScreenShare 组件，确保组件已挂载
  showScreenShare.value = true
  
  // 等待组件渲染后再调用
  nextTick(() => {
    if (screenShareComponent.value) {
      screenShareComponent.value.startScreenShare()
    } else {
      QMessage.error('屏幕共享组件未初始化，请稍后重试')
    }
  })
}

const screenShareRef = ref<InstanceType<typeof ScreenShare>>()

// 屏幕共享组件挂载完成后的回调
const onScreenShareMounted = () => {
  console.log('RealtimeCommunication: ScreenShare 组件已挂载')
  if (screenShareRef.value) {
    screenShareComponent.value = screenShareRef.value
    console.log('RealtimeCommunication: 屏幕共享组件引用设置成功')
  }
}

// 注册回调：当收到 offer 时显示组件
setOnShowScreenShareRequest(() => {
  console.log('RealtimeCommunication: 收到 offer，需要显示组件')
  showScreenShare.value = true
})

// ==========================================
// 视频通话相关状态
// ==========================================
const showCallModal = ref(false)
const callType = ref<'voice' | 'video'>('video')
const callStatus = ref<'idle' | 'calling' | 'connecting' | 'connected' | 'ended'>('idle')
const callAvatar = ref('')
const callName = ref('')

const videoCall = useVideoCall()
const {
  callStatus: videoCallStatus,
  callType: videoCallType,
  remoteUser: videoCallRemoteUser,
  localStream: videoCallLocalStream,
  remoteStream: videoCallRemoteStream,
  isMuted: videoCallIsMuted,
  isVideoEnabled: videoCallIsVideoEnabled,
  incomingCall: videoCallIncomingCall,
  startCall: videoCallStart,
  answerCall: videoCallAnswer,
  endCall: videoCallEnd,
  rejectCall: videoCallReject,
  toggleMute: videoCallToggleMute,
  toggleVideo: videoCallToggleVideo,
  handleSignalingMessage: videoCallHandleSignaling
} = videoCall

// 同步视频通话状态到组件 props

watch([videoCallStatus, videoCallType, videoCallRemoteUser], ([status, type, remoteUser]) => {
  callStatus.value = status as any
  callType.value = type
  if (remoteUser) {
    callAvatar.value = remoteUser.avatar || ''
    callName.value = remoteUser.name || '未知用户'
  }
  showCallModal.value = status !== 'idle' && status !== 'ended'
})

// ==========================================
// 屏幕共享事件处理
// ==========================================
const handleScreenShareStart = (data: { conversationId: string | number }) => {
  emit('screen-share-start', { conversationId: Number(props.currentConversation?.id) || 0, requester_id: Number(currentUser.value?.id) || 0 })
}

const handleScreenShareStop = () => {
  emit('screen-share-stop', { conversationId: Number(props.currentConversation?.id) || 0 })
}

const handleScreenShareJoin = () => {
  emit('screen-share-data', { conversationId: Number(props.currentConversation?.id) || 0, data: 'join' })
}

const handleScreenShareLeave = () => {
  emit('screen-share-data', { conversationId: Number(props.currentConversation?.id) || 0, data: 'leave' })
}

// ==========================================
// 通话事件处理
// ==========================================
const handleRejectCall = async () => {
  try {
    await videoCallReject()
    showCallModal.value = false
  } catch (error) {
    console.error('拒绝通话失败:', error)
  }
}

const handleAnswerCall = async () => {
  try {
    await videoCallAnswer()
  } catch (error) {
    console.error('接听通话失败:', error)
  }
}

const handleEndCall = async () => {
  try {
    await videoCallEnd()
    showCallModal.value = false
  } catch (error) {
    console.error('结束通话失败:', error)
  }
}

const handleCallModalClose = async () => {
  try {
    await videoCallEnd()
    showCallModal.value = false
  } catch (error) {
    console.error('关闭通话模态框失败:', error)
    showCallModal.value = false
  }
}

// ==========================================
// 全局消息处理接口（供 Main.vue 调用）
// ==========================================
const handleScreenShareMessageGlobal = (type: string, data: any) => {
  console.log('RealtimeCommunication: handleScreenShareMessageGlobal 被调用, type:', type, 'data:', data)
  handleScreenShareMessage(type, data)
}

const handleScreenShareRequestGlobal = (data: any) => {
  console.log('RealtimeCommunication: handleScreenShareRequestGlobal 被调用, data:', data)
  handleScreenShareRequest(data)
}

const handleScreenShareAcceptedGlobal = (data: any) => {
  console.log('RealtimeCommunication: handleScreenShareAcceptedGlobal 被调用, data:', data)
  handleScreenShareAccepted(data)
}

const handleScreenShareRejectedGlobal = (data: any) => {
  console.log('RealtimeCommunication: handleScreenShareRejectedGlobal 被调用, data:', data)
  handleScreenShareRejected(data)
}

const handleWebRTCOfferGlobal = (data: any) => {
  console.log('RealtimeCommunication: 收到 webrtc_offer', data)
  if (!data?.signal || !data?.from_user_id) return

  const fromUserId = data.from_user_id
  
  // 尝试从会话成员中获取用户名
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

  if (conv && props.currentConversation?.id !== conv.id) {
    console.log('RealtimeCommunication: webrtc_offer 来自非当前会话')
    if (props.onConversationSwitch) {
      props.onConversationSwitch(conv)
    }
  }

  handleWebRTCOffer(data)
}

const handleWebRTCAnswerGlobal = (data: any) => {
  handleWebRTCAnswer(data)
}

const handleWebRTCIceCandidateGlobal = (data: any) => {
  handleWebRTCIceCandidate(data)
}

const handleScreenShareStartGlobal = (data: any) => {
  showScreenShare.value = true
  if (props.currentConversation) {
    screenShareConversationId.value = props.currentConversation.id
  }
}

const handleScreenShareStopGlobal = (data: any) => {
  showScreenShare.value = false
  cleanupScreenShare()
}

const handleVideoCallSignalingGlobal = (message: { type: string; data: any }) => {
  videoCallHandleSignaling(message)
}

// ==========================================
// 组件挂载后初始化
// ==========================================
import { onMounted } from 'vue'

onMounted(() => {
  const cachedOffer = consumeCachedOffer()
  if (cachedOffer) {
    console.log('RealtimeCommunication: 发现缓存的 webrtc_offer')
    nextTick(() => {
      handleWebRTCOfferGlobal({ signal: cachedOffer.signal, from_user_id: cachedOffer.fromUserId })
    })
  }
})

// ==========================================
// 暴露方法供外部调用
// ==========================================
defineExpose({
  startScreenShare,
  handleScreenShareMessage: handleScreenShareMessageGlobal,
  handleWebRTCOffer: handleWebRTCOfferGlobal,
  handleWebRTCAnswer: handleWebRTCAnswerGlobal,
  handleWebRTCIceCandidate: handleWebRTCIceCandidateGlobal,
  handleScreenShareRequest: handleScreenShareRequestGlobal,
  handleScreenShareAccepted: handleScreenShareAcceptedGlobal,
  handleScreenShareRejected: handleScreenShareRejectedGlobal,
  handleScreenShareStart: handleScreenShareStartGlobal,
  handleScreenShareStop: handleScreenShareStopGlobal,
  handleVideoCallSignaling: handleVideoCallSignalingGlobal,
  screenShareRef,
  videoCall
})
</script>

<style scoped>
</style>
