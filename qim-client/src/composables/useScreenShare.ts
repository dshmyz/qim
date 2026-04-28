import { ref, nextTick } from 'vue'
import QMessage from '../utils/qmessage'
// @ts-ignore - WebRTC module has no type declarations
import { screenShareSender, screenShareReceiver } from '../utils/webrtc'
import { getCurrentUser } from '../utils/user'

/**
 * 屏幕共享 WebRTC 相关逻辑
 * 包含：屏幕共享请求处理、WebRTC 信令、开始/停止共享等
 */
export function useScreenShare(conversation: { value: any }) {
  const screenShareComponent = ref<any>(null)
  const remoteScreenUserId = ref<number | null>(null)
  const showScreenShareViewer = ref(false)

  const registeredScreenShareHandlers: { handler: Function; type: string }[] = []

  /**
   * 处理屏幕共享消息
   */
  const handleScreenShareMessage = (type: string, data: any) => {
    switch (type) {
      case 'webrtc_offer':
        handleWebRTCOffer(data)
        break
      case 'webrtc_answer':
        handleWebRTCAnswer(data)
        break
      case 'webrtc_ice_candidate':
        handleWebRTCIceCandidate(data)
        break
      case 'screen-share-request':
        handleScreenShareRequest(data)
        break
      case 'screen-share-accepted':
        handleScreenShareAccepted(data)
        break
      case 'screen-share-rejected':
        handleScreenShareRejected(data)
        break
      case 'screen-share-start':
        handleScreenShareStart(data)
        break
      case 'screen-share-stop':
        handleScreenShareStop(data)
        break
    }
  }

  /**
   * 处理 WebRTC offer
   */
  const handleWebRTCOffer = async (data: any) => {
    try {
      remoteScreenUserId.value = data.from_user_id
      await nextTick()
      QMessage.info('收到屏幕共享请求', 5000)
      if (screenShareComponent.value) {
        await screenShareComponent.value.handleOffer(data.signal, data.from_user_id)
      }
    } catch (error) {
      console.error('处理 WebRTC offer 失败:', error)
    }
  }

  /**
   * 处理 WebRTC answer
   */
  const handleWebRTCAnswer = (data: any) => {
    try {
      screenShareSender.handleAnswer(data.signal)
    } catch (error) {
      console.error('处理 WebRTC answer 失败:', error)
    }
  }

  /**
   * 处理 WebRTC ICE candidate
   */
  const handleWebRTCIceCandidate = async (data: any) => {
    try {
      if (screenShareSender.getIsSharing()) {
        screenShareSender.addIceCandidate(data.signal)
      } else if (screenShareComponent.value) {
        screenShareComponent.value.handleIceCandidate(data.signal)
      } else {
        screenShareReceiver.handleIceCandidate(data.signal)
      }
    } catch (error) {
      console.error('处理 WebRTC ICE candidate 失败:', error)
    }
  }

  /**
   * 处理屏幕共享请求
   */
  const handleScreenShareRequest = (data: any) => {
    QMessage.info(`${data.from_user_name} 请求屏幕共享`, 5000)
    screenShareReceiver.handleShareRequest(data)
  }

  /**
   * 处理屏幕共享已接受
   */
  const handleScreenShareAccepted = async (data: any) => {
    try {
      QMessage.success('屏幕共享请求已接受')
      if (screenShareComponent.value) {
        screenShareComponent.value.establishConnection()
      }
    } catch (error) {
      console.error('处理屏幕共享接受失败:', error)
    }
  }

  /**
   * 处理屏幕共享被拒绝
   */
  const handleScreenShareRejected = (data: any) => {
    QMessage.warning('屏幕共享请求被拒绝')
  }

  /**
   * 处理屏幕共享开始
   */
  const handleScreenShareStart = (data: any) => {
    showScreenShareViewer.value = true
    QMessage.success('屏幕共享已开始')
  }

  /**
   * 处理屏幕共享停止
   */
  const handleScreenShareStop = (data: any) => {
    showScreenShareViewer.value = false
    if (screenShareComponent.value) {
      screenShareComponent.value.stopReceiving()
    }
    QMessage.info('屏幕共享已停止')
  }

  /**
   * 发送屏幕共享信令
   */
  const sendScreenShareSignal = async (sendSignal: Function, conversationId: string | number, signal: any) => {
    try {
      await sendSignal(conversationId, signal)
    } catch (error) {
      console.error('发送屏幕共享信令失败:', error)
    }
  }

  /**
   * 开始屏幕共享
   */
  const startScreenShare = () => {
    if (!conversation.value) {
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
    
    if (screenShareComponent.value) {
      screenShareComponent.value.startScreenShare()
    } else {
      QMessage.error('屏幕共享组件未初始化，请稍后重试')
    }
  }

  /**
   * 接收屏幕共享流
   */
  const receiveScreenShareStream = (data: any) => {
    if (screenShareComponent.value) {
      screenShareComponent.value.receiveScreenShareStream(data)
    }
  }

  /**
   * 停止屏幕共享
   */
  const stopScreenShare = () => {
    screenShareSender.stopScreenShare()
  }

  /**
   * 清理屏幕共享资源
   */
  const cleanupScreenShare = () => {
    registeredScreenShareHandlers.length = 0
    if (screenShareComponent.value) {
      screenShareComponent.value.stopReceiving()
    }
    showScreenShareViewer.value = false
    remoteScreenUserId.value = null
  }

  return {
    // 状态
    screenShareComponent,
    remoteScreenUserId,
    showScreenShareViewer,
    registeredScreenShareHandlers,
    // 方法
    handleScreenShareMessage,
    handleWebRTCOffer,
    handleWebRTCAnswer,
    handleWebRTCIceCandidate,
    handleScreenShareRequest,
    handleScreenShareAccepted,
    handleScreenShareRejected,
    handleScreenShareStart,
    handleScreenShareStop,
    sendScreenShareSignal,
    startScreenShare,
    receiveScreenShareStream,
    stopScreenShare,
    cleanupScreenShare
  }
}
