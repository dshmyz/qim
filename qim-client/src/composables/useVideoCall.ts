import { ref, onUnmounted, readonly } from 'vue'
import { videoCallManager, type CallStatus, type CallType } from '../utils/videoCall'
import type { User } from '../types'

export interface RemoteUser {
  id: number
  name: string
  avatar: string
}

export interface IncomingCall {
  userId: string
  callType: 'voice' | 'video'
  signal: any
}

// 扩展 Window 类型以支持 ws
declare global {
  interface Window {
    ws?: WebSocket
  }
}

/**
 * 视频通话 Composable
 * 封装 VideoCallManager 为 Vue 3 Composable，提供响应式的通话状态管理
 */
export function useVideoCall() {
  // 响应式状态
  const callStatus = ref<CallStatus>('idle')
  const callType = ref<CallType>('voice')
  const localStream = ref<MediaStream | null>(null)
  const remoteStream = ref<MediaStream | null>(null)
  const isMuted = ref(false)
  const isVideoEnabled = ref(true)
  const remoteUser = ref<RemoteUser | null>(null)
  const incomingCall = ref<IncomingCall | null>(null)

  // 用户信息映射（用于根据 userId 获取用户信息）
  const userInfoCache = ref<Map<string, RemoteUser>>(new Map())

  /**
   * 设置用户信息缓存
   */
  const setUserInfo = (userId: string, userInfo: RemoteUser) => {
    userInfoCache.value.set(userId, userInfo)
  }

  /**
   * 根据 userId 获取用户信息
   */
  const getUserInfo = (userId: string): RemoteUser | undefined => {
    return userInfoCache.value.get(userId)
  }

  // 事件回调函数（需要保存引用以便解绑）
  const handleLocalStream = (stream: MediaStream) => {
    localStream.value = stream
  }

  const handleRemoteStream = (stream: MediaStream) => {
    remoteStream.value = stream
  }

  const handleCallStatusChange = (status: CallStatus) => {
    callStatus.value = status

    // 通话结束时重置相关状态
    if (status === 'ended') {
      // 延迟清理，确保 UI 有时间展示结束状态
      setTimeout(() => {
        localStream.value = null
        remoteStream.value = null
        incomingCall.value = null
        remoteUser.value = null
        videoCallManager.reset()
      }, 100)
    }
  }

  const handleError = (error: Error) => {
    console.error('视频通话错误:', error)
  }

  const handleIceCandidate = (candidate: RTCIceCandidate, targetUserId: string) => {
    // 发送 ICE 候选者到对端
    try {
      const ws = window.ws
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          type: 'webrtc_ice_candidate',
          data: {
            target_user_id: targetUserId,
            candidate: candidate.toJSON()
          }
        }))
      } else {
        // 通过 electron IPC 发送
        const electron = (window as any).electron
        if (electron?.websocket) {
          electron.websocket.send({
            type: 'webrtc_ice_candidate',
            data: {
              target_user_id: targetUserId,
              candidate: candidate.toJSON()
            }
          })
        }
      }
    } catch (error) {
      console.error('发送 ICE 候选者失败:', error)
    }
  }

  // 绑定事件回调
  const bindVideoCallEvents = () => {
    videoCallManager.onLocalStream = handleLocalStream
    videoCallManager.onRemoteStream = handleRemoteStream
    videoCallManager.onCallStatusChange = handleCallStatusChange
    videoCallManager.onError = handleError
    videoCallManager.onIceCandidate = handleIceCandidate
  }

  // 解绑事件回调
  const unbindVideoCallEvents = () => {
    videoCallManager.onLocalStream = null
    videoCallManager.onRemoteStream = null
    videoCallManager.onCallStatusChange = null
    videoCallManager.onError = null
    videoCallManager.onIceCandidate = null
  }

  // 处理收到的 WebSocket 信令消息
  const handleSignalingMessage = (message: { type: string; data: any }) => {
    const { type, data } = message

    switch (type) {
      case 'call_invite':
        // 收到呼叫邀请
        handleIncomingCallInvite(data)
        break

      case 'call_accept':
        // 对方接听
        handleCallAccept(data)
        break

      case 'call_reject':
        // 对方拒绝
        handleCallReject(data)
        break

      case 'call_end':
        // 对方结束通话
        handleCallEnd(data)
        break

      case 'webrtc_offer':
        // 收到 WebRTC offer
        handleWebRtcOffer(data)
        break

      case 'webrtc_answer':
        // 收到 WebRTC answer
        handleWebRtcAnswer(data)
        break

      case 'webrtc_ice_candidate':
        // 收到 ICE 候选者
        handleIceCandidateMessage(data)
        break
    }
  }

  // 处理呼叫邀请
  const handleIncomingCallInvite = async (data: {
    target_user_id: string
    call_type: CallType
    signal: RTCSessionDescriptionInit
    user_info?: RemoteUser
  }) => {
    try {
      console.log('收到呼叫邀请:', data)

      // 从缓存或数据中获取用户信息
      let userInfo = getUserInfo(data.target_user_id)
      if (!userInfo && data.user_info) {
        userInfo = data.user_info
        setUserInfo(data.target_user_id, userInfo)
      }

      // 设置来电信息
      incomingCall.value = {
        userId: data.target_user_id,
        callType: data.call_type,
        signal: data.signal
      }

      // 设置远程用户信息
      if (userInfo) {
        remoteUser.value = userInfo
      }

      // 更新通话类型
      callType.value = data.call_type
      isVideoEnabled.value = data.call_type === 'video'

      // 处理收到的 offer（作为被叫方）
      await videoCallManager.handleIncomingCall({
        target_user_id: data.target_user_id,
        call_type: data.call_type,
        signal: data.signal
      })
    } catch (error) {
      console.error('处理呼叫邀请失败:', error)
    }
  }

  // 处理对方接听
  const handleCallAccept = async (data: { target_user_id: string; signal: RTCSessionDescriptionInit }) => {
    try {
      console.log('对方接听:', data)

      // 如果是主叫方，收到 answer
      if (videoCallManager.getCallStatus() === 'calling') {
        await videoCallManager.handleAnswer(data.signal)
      }
    } catch (error) {
      console.error('处理接听响应失败:', error)
    }
  }

  // 处理对方拒绝
  const handleCallReject = (data: { target_user_id: string }) => {
    console.log('对方拒绝通话:', data)
    videoCallManager.cleanup()
    callStatus.value = 'ended'
    localStream.value = null
    incomingCall.value = null
  }

  // 处理对方结束通话
  const handleCallEnd = (data: { target_user_id: string }) => {
    console.log('对方结束通话:', data)
    videoCallManager.handleRemoteEndCall()
  }

  // 处理 WebRTC offer
  const handleWebRtcOffer = async (data: {
    sender_id: string
    signal: RTCSessionDescriptionInit
    user_info?: RemoteUser
  }) => {
    try {
      console.log('收到 WebRTC offer:', data)

      // 从缓存或数据中获取用户信息
      let userInfo = getUserInfo(data.sender_id)
      if (!userInfo && data.user_info) {
        userInfo = data.user_info
        setUserInfo(data.sender_id, userInfo)
        remoteUser.value = userInfo
      }

      await videoCallManager.handleOffer(data.signal, data.sender_id)
    } catch (error) {
      console.error('处理 WebRTC offer 失败:', error)
    }
  }

  // 处理 WebRTC answer
  const handleWebRtcAnswer = async (data: { signal: RTCSessionDescriptionInit }) => {
    try {
      console.log('收到 WebRTC answer:', data)
      await videoCallManager.handleAnswer(data.signal)
    } catch (error) {
      console.error('处理 WebRTC answer 失败:', error)
    }
  }

  // 处理 ICE 候选者
  const handleIceCandidateMessage = (data: { candidate: RTCIceCandidateInit }) => {
    try {
      console.log('收到 ICE 候选者:', data)
      videoCallManager.addIceCandidate(data.candidate)
    } catch (error) {
      console.error('处理 ICE 候选者失败:', error)
    }
  }

  /**
   * 发起通话
   */
  const startCall = async (user: User, type: 'voice' | 'video'): Promise<void> => {
    try {
      // 将 user.id 转换为字符串
      const userIdStr = String(user.id)

      // 保存远程用户信息
      remoteUser.value = {
        id: typeof user.id === 'string' ? parseInt(user.id) : user.id as number,
        name: user.nickname || user.name,
        avatar: user.avatar
      }

      // 缓存用户信息
      setUserInfo(userIdStr, remoteUser.value)

      // 更新通话类型
      callType.value = type
      isVideoEnabled.value = type === 'video'

      // 调用 videoCallManager 发起通话
      await videoCallManager.startCall(userIdStr, type)
    } catch (error) {
      console.error('发起通话失败:', error)
      throw error
    }
  }

  /**
   * 接听通话
   */
  const answerCall = async (): Promise<void> => {
    try {
      await videoCallManager.answerCall()
    } catch (error) {
      console.error('接听通话失败:', error)
      throw error
    }
  }

  /**
   * 结束通话
   */
  const endCall = async (): Promise<void> => {
    try {
      await videoCallManager.endCall()
    } catch (error) {
      console.error('结束通话失败:', error)
      throw error
    }
  }

  /**
   * 拒绝通话
   */
  const rejectCall = async (): Promise<void> => {
    try {
      await videoCallManager.rejectCall()
    } catch (error) {
      console.error('拒绝通话失败:', error)
      throw error
    }
  }

  /**
   * 切换静音
   */
  const toggleMute = (): void => {
    videoCallManager.toggleMute()
    isMuted.value = videoCallManager.getIsMuted()
  }

  /**
   * 切换视频
   */
  const toggleVideo = (): void => {
    videoCallManager.toggleVideo()
    isVideoEnabled.value = videoCallManager.getIsVideoEnabled()
  }

  // 绑定事件（在 setup 时自动调用）
  bindVideoCallEvents()

  // 组件卸载时自动解绑
  onUnmounted(() => {
    unbindVideoCallEvents()
  })

  return {
    // 响应式状态（只读）
    callStatus: readonly(callStatus),
    callType: readonly(callType),
    localStream: readonly(localStream),
    remoteStream: readonly(remoteStream),
    isMuted: readonly(isMuted),
    isVideoEnabled: readonly(isVideoEnabled),
    remoteUser: readonly(remoteUser),
    incomingCall: readonly(incomingCall),

    // 可修改的状态（用于内部更新）
    callStatusWritable: callStatus,
    callTypeWritable: callType,
    localStreamWritable: localStream,
    remoteStreamWritable: remoteStream,
    isMutedWritable: isMuted,
    isVideoEnabledWritable: isVideoEnabled,
    remoteUserWritable: remoteUser,
    incomingCallWritable: incomingCall,

    // 用户信息管理
    setUserInfo,
    getUserInfo,

    // 信令处理
    handleSignalingMessage,

    // 通话操作方法
    startCall,
    answerCall,
    endCall,
    rejectCall,
    toggleMute,
    toggleVideo
  }
}

export default useVideoCall
