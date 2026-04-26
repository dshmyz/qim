import { logger } from './logger';
// WebRTC 视频通话核心模块

export type CallStatus = 'idle' | 'calling' | 'ringing' | 'answered' | 'ended';
export type CallType = 'voice' | 'video';

export interface VideoCallEvents {
  onLocalStream: (stream: MediaStream) => void;
  onRemoteStream: (stream: MediaStream) => void;
  onCallStatusChange: (status: CallStatus) => void;
  onError: (error: Error) => void;
  onIceCandidate: (candidate: RTCIceCandidate, targetUserId: string) => void;
}

class VideoCallManager {
  private peerConnection: RTCPeerConnection | null = null;
  private localStream: MediaStream | null = null;
  private remoteStream: MediaStream | null = null;
  private callStatus: CallStatus = 'idle';
  private callType: CallType = 'voice';
  private isMuted: boolean = false;
  private isVideoEnabled: boolean = true;
  private targetUserId: string | null = null;
  private iceCandidateCache: RTCIceCandidate[] = [];

  // 事件回调
  public onLocalStream: ((stream: MediaStream) => void) | null = null;
  public onRemoteStream: ((stream: MediaStream) => void) | null = null;
  public onCallStatusChange: ((status: CallStatus) => void) | null = null;
  public onError: ((error: Error) => void) | null = null;
  public onIceCandidate: ((candidate: RTCIceCandidate, targetUserId: string) => void) | null = null;

  // ICE 服务器配置
  private readonly iceServers: RTCIceServer[] = [
    {
      urls: 'stun:stun.l.google.com:19302'
    },
    {
      urls: 'stun:stun1.l.google.com:19302'
    }
  ];

  // 获取呼叫状态
  getCallStatus(): CallStatus {
    return this.callStatus;
  }

  // 获取通话类型
  getCallType(): CallType {
    return this.callType;
  }

  // 获取目标用户ID
  getTargetUserId(): string | null {
    return this.targetUserId;
  }

  // 获取本地流
  getLocalStream(): MediaStream | null {
    return this.localStream;
  }

  // 获取远程流
  getRemoteStream(): MediaStream | null {
    return this.remoteStream;
  }

  // 获取静音状态
  getIsMuted(): boolean {
    return this.isMuted;
  }

  // 获取视频启用状态
  getIsVideoEnabled(): boolean {
    return this.isVideoEnabled;
  }

  // 设置通话状态
  private setCallStatus(status: CallStatus): void {
    this.callStatus = status;
    if (this.onCallStatusChange) {
      this.onCallStatusChange(status);
    }
  }

  // 检查浏览器支持
  private checkBrowserSupport(): void {
    if (!navigator.mediaDevices) {
      throw new Error('浏览器不支持媒体设备 API');
    }

    if (!navigator.mediaDevices.getUserMedia) {
      throw new Error('浏览器不支持 getUserMedia API');
    }

    if (!window.RTCPeerConnection) {
      throw new Error('浏览器不支持 WebRTC API');
    }
  }

  // 获取媒体流
  private async getMediaStream(callType: CallType): Promise<MediaStream> {
    const constraints: MediaStreamConstraints = {
      audio: true,
      video: callType === 'video'
    };

    try {
      return await navigator.mediaDevices.getUserMedia(constraints);
    } catch (error: any) {
      if (error.name === 'NotAllowedError' || error.name === 'PermissionDeniedError') {
        throw new Error('用户拒绝授权摄像头/麦克风');
      } else if (error.name === 'NotFoundError') {
        throw new Error('未找到摄像头或麦克风设备');
      } else if (error.name === 'NotReadableError') {
        throw new Error('摄像头或麦克风被其他应用占用');
      }
      throw new Error(`获取媒体流失败: ${error.message}`);
    }
  }

  // 创建 RTCPeerConnection
  private createPeerConnection(): RTCPeerConnection {
    const config: RTCConfiguration = {
      iceServers: this.iceServers,
      iceTransportPolicy: 'all',
      iceCandidatePoolSize: 10
    };

    const pc = new RTCPeerConnection(config);

    // 处理 ICE 候选者
    pc.onicecandidate = (event) => {
      if (event.candidate && this.onIceCandidate && this.targetUserId) {
        this.onIceCandidate(event.candidate, this.targetUserId);
      }
    };

    // 处理 ICE 连接状态变化
    pc.oniceconnectionstatechange = () => {
      logger.log('ICE 连接状态变化:', pc.iceConnectionState);

      if (pc.iceConnectionState === 'failed') {
        this.onError?.(new Error('ICE 连接失败'));
      } else if (pc.iceConnectionState === 'disconnected') {
        logger.log('ICE 连接已断开');
      }
    };

    // 处理连接状态变化
    pc.onconnectionstatechange = () => {
      logger.log('连接状态变化:', pc.connectionState);

      if (pc.connectionState === 'connected') {
        logger.log('WebRTC 连接已建立');
      } else if (pc.connectionState === 'disconnected') {
        logger.log('WebRTC 连接已断开');
      } else if (pc.connectionState === 'failed') {
        this.onError?.(new Error('WebRTC 连接失败'));
      }
    };

    // 处理远程流
    pc.ontrack = (event) => {
      logger.log('收到远程流事件:', event);

      if (event.streams && event.streams.length > 0) {
        this.remoteStream = event.streams[0];
        logger.log('远程流:', this.remoteStream);

        if (this.onRemoteStream) {
          this.onRemoteStream(this.remoteStream);
        }
      }
    };

    return pc;
  }

  // 发送信令消息
  private sendSignalMessage(type: string, data: any): void {
    try {
      if (window.ws && window.ws.readyState === WebSocket.OPEN) {
        window.ws.send(JSON.stringify({
          type,
          data
        }));
        logger.log(`信令消息发送成功: ${type}`);
      } else if ((window as any).electron?.websocket) {
        (window as any).electron.websocket.send({
          type,
          data
        });
        logger.log(`信令消息发送成功（通过 IPC）: ${type}`);
      } else {
        console.error('WebSocket 连接不可用');
        throw new Error('WebSocket 连接不可用');
      }
    } catch (error) {
      console.error('发送信令消息失败:', error);
      throw error;
    }
  }

  // 发起通话
  async startCall(targetUserId: string, callType: CallType): Promise<void> {
    try {
      logger.log('开始通话:', targetUserId, callType);

      // 检查浏览器支持
      this.checkBrowserSupport();

      this.targetUserId = targetUserId;
      this.callType = callType;
      this.isMuted = false;
      this.isVideoEnabled = callType === 'video';

      // 获取本地媒体流
      this.localStream = await this.getMediaStream(callType);
      logger.log('本地流获取成功:', this.localStream);

      if (this.onLocalStream) {
        this.onLocalStream(this.localStream);
      }

      // 创建 RTCPeerConnection
      this.peerConnection = this.createPeerConnection();

      // 添加本地流到连接
      if (this.localStream) {
        const tracks = this.localStream.getTracks();
        tracks.forEach(track => {
          this.peerConnection?.addTrack(track, this.localStream!);
        });
        logger.log('本地流轨道已添加');
      }

      // 生成 offer
      const offer = await this.peerConnection.createOffer();
      await this.peerConnection.setLocalDescription(offer);
      logger.log('offer 生成成功');

      // 发送呼叫邀请
      this.sendSignalMessage('call_invite', {
        target_user_id: targetUserId,
        call_type: callType,
        signal: offer
      });

      this.setCallStatus('calling');
      logger.log('通话已发起');
    } catch (error: any) {
      console.error('发起通话失败:', error);
      this.doCleanup();
      this.onError?.(error);
      throw error;
    }
  }

  // 接听通话
  async answerCall(): Promise<void> {
    try {
      logger.log('接听通话');

      if (!this.peerConnection) {
        throw new Error('PeerConnection 未初始化');
      }

      // 获取本地媒体流
      this.localStream = await this.getMediaStream(this.callType);
      logger.log('本地流获取成功:', this.localStream);

      if (this.onLocalStream) {
        this.onLocalStream(this.localStream);
      }

      // 添加本地流到连接
      if (this.localStream) {
        const tracks = this.localStream.getTracks();
        tracks.forEach(track => {
          this.peerConnection?.addTrack(track, this.localStream!);
        });
        logger.log('本地流轨道已添加');
      }

      // 发送接听响应
      if (this.targetUserId) {
        const answer = await this.peerConnection.createAnswer();
        await this.peerConnection.setLocalDescription(answer);
        logger.log('answer 生成成功');

        this.sendSignalMessage('call_accept', {
          target_user_id: this.targetUserId,
          signal: answer
        });
      }

      this.setCallStatus('answered');
      logger.log('通话已接听');
    } catch (error: any) {
      console.error('接听通话失败:', error);
      this.doCleanup();
      this.onError?.(error);
      throw error;
    }
  }

  // 结束通话
  async endCall(): Promise<void> {
    try {
      logger.log('结束通话');

      if (this.targetUserId) {
        this.sendSignalMessage('call_end', {
          target_user_id: this.targetUserId
        });
      }

      this.doCleanup();
      this.setCallStatus('ended');
      logger.log('通话已结束');
    } catch (error: any) {
      console.error('结束通话失败:', error);
      this.doCleanup();
      this.onError?.(error);
      throw error;
    }
  }

  // 拒绝通话
  async rejectCall(): Promise<void> {
    try {
      logger.log('拒绝通话');

      if (this.targetUserId) {
        this.sendSignalMessage('call_reject', {
          target_user_id: this.targetUserId
        });
      }

      this.doCleanup();
      this.setCallStatus('ended');
      logger.log('通话已拒绝');
    } catch (error: any) {
      console.error('拒绝通话失败:', error);
      this.doCleanup();
      this.onError?.(error);
      throw error;
    }
  }

  // 切换静音
  toggleMute(): void {
    if (this.localStream) {
      const audioTracks = this.localStream.getAudioTracks();
      audioTracks.forEach(track => {
        track.enabled = !track.enabled;
      });
      this.isMuted = !this.isMuted;
      logger.log('静音状态:', this.isMuted);
    }
  }

  // 切换摄像头
  toggleVideo(): void {
    if (this.localStream) {
      const videoTracks = this.localStream.getVideoTracks();
      videoTracks.forEach(track => {
        track.enabled = !track.enabled;
      });
      this.isVideoEnabled = !this.isVideoEnabled;
      logger.log('视频启用状态:', this.isVideoEnabled);
    }
  }

  // 处理收到的 offer
  async handleOffer(offer: RTCSessionDescriptionInit, senderId: string): Promise<void> {
    try {
      logger.log('处理 offer，发送者 ID:', senderId);

      this.targetUserId = senderId;
      this.peerConnection = this.createPeerConnection();

      // 设置远程描述
      await this.peerConnection.setRemoteDescription(new RTCSessionDescription(offer));
      logger.log('远程描述设置成功');

      // 刷新缓存的 ICE 候选者
      this.flushIceCandidates();

      // 生成 answer
      const answer = await this.peerConnection.createAnswer();
      await this.peerConnection.setLocalDescription(answer);
      logger.log('answer 生成成功');

      // 发送 answer
      this.sendSignalMessage('call_accept', {
        target_user_id: senderId,
        signal: answer
      });

      this.setCallStatus('ringing');
    } catch (error: any) {
      console.error('处理 offer 失败:', error);
      this.onError?.(error);
      throw error;
    }
  }

  // 处理收到的 answer
  async handleAnswer(answer: RTCSessionDescriptionInit): Promise<void> {
    try {
      logger.log('处理 answer');

      if (!this.peerConnection) {
        throw new Error('PeerConnection 未初始化');
      }

      await this.peerConnection.setRemoteDescription(new RTCSessionDescription(answer));
      logger.log('远程描述设置成功');

      // 刷新缓存的 ICE 候选者
      this.flushIceCandidates();

      this.setCallStatus('answered');
    } catch (error: any) {
      console.error('处理 answer 失败:', error);
      this.onError?.(error);
      throw error;
    }
  }

  // 处理收到的 ICE 候选者
  addIceCandidate(candidate: RTCIceCandidateInit): void {
    try {
      if (!this.peerConnection) {
        logger.log('PeerConnection 未初始化，缓存 ICE 候选者');
        this.iceCandidateCache.push(new RTCIceCandidate(candidate));
        return;
      }

      if (this.peerConnection.remoteDescription) {
        const iceCandidate = new RTCIceCandidate(candidate);
        this.peerConnection.addIceCandidate(iceCandidate);
        logger.log('ICE 候选者添加成功:', iceCandidate);
      } else {
        logger.log('远程描述未设置，缓存 ICE 候选者');
        this.iceCandidateCache.push(new RTCIceCandidate(candidate));
      }
    } catch (error) {
      console.error('添加 ICE 候选者失败:', error);
    }
  }

  // 刷新缓存的 ICE 候选者
  private flushIceCandidates(): void {
    if (this.peerConnection && this.peerConnection.remoteDescription) {
      logger.log('刷新缓存的 ICE 候选者，数量:', this.iceCandidateCache.length);

      while (this.iceCandidateCache.length > 0) {
        const candidate = this.iceCandidateCache.shift();
        if (candidate) {
          try {
            this.peerConnection.addIceCandidate(candidate);
            logger.log('缓存的 ICE 候选者添加成功:', candidate);
          } catch (error) {
            console.error('添加缓存的 ICE 候选者失败:', error);
          }
        }
      }
    }
  }

  // 处理收到的呼叫邀请（作为被叫方）
  async handleIncomingCall(data: {
    target_user_id: string;
    call_type: CallType;
    signal: RTCSessionDescriptionInit;
  }): Promise<void> {
    try {
      logger.log('收到呼叫邀请:', data);

      this.targetUserId = data.target_user_id;
      this.callType = data.call_type;
      this.isMuted = false;
      this.isVideoEnabled = data.call_type === 'video';

      this.peerConnection = this.createPeerConnection();

      // 设置远程描述
      await this.peerConnection.setRemoteDescription(new RTCSessionDescription(data.signal));
      logger.log('远程描述设置成功');

      // 刷新缓存的 ICE 候选者
      this.flushIceCandidates();

      this.setCallStatus('ringing');
    } catch (error: any) {
      console.error('处理呼叫邀请失败:', error);
      this.onError?.(error);
      throw error;
    }
  }

  // 处理对方结束通话
  handleRemoteEndCall(): void {
    logger.log('对方结束通话');
    this.doCleanup();
    this.setCallStatus('ended');
  }

  // 清理资源（对外暴露）
  public cleanup(): void {
    this.doCleanup()
  }

  // 内部清理资源
  private doCleanup(): void {
    logger.log('清理资源');

    // 停止本地流
    if (this.localStream) {
      this.localStream.getTracks().forEach(track => track.stop());
      this.localStream = null;
    }

    // 关闭 PeerConnection
    if (this.peerConnection) {
      this.peerConnection.close();
      this.peerConnection = null;
    }

    this.remoteStream = null;
    this.iceCandidateCache = [];
    this.isMuted = false;
    this.isVideoEnabled = true;
  }

  // 重置状态
  reset(): void {
    this.doCleanup();
    this.callStatus = 'idle';
    this.callType = 'voice';
    this.targetUserId = null;
  }
}

// 导出单例实例
export const videoCallManager = new VideoCallManager();
export default videoCallManager;
