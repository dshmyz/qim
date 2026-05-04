import { logger } from './logger';
// WebRTC 屏幕共享工具类

class ScreenShareSender {
  constructor() {
    this.peerConnection = null;
    this.screenStream = null;
    this.isSharing = false;
    this.receiverId = null;
    this.enableDirectConnect = true; // 启用直连尝试
    this.iceCandidateCache = []; // ICE 候选者缓存
  }

  // 开始屏幕共享
  async startScreenShare(receiverId) {
    try {
      this.receiverId = receiverId;
      
      // 检查浏览器支持
      logger.log('检查浏览器支持...');
      logger.log('navigator.mediaDevices:', navigator.mediaDevices);
      logger.log('navigator.mediaDevices.getDisplayMedia:', navigator.mediaDevices?.getDisplayMedia);
      logger.log('window.RTCPeerConnection:', window.RTCPeerConnection);
      logger.log('window.electron:', window.electron);
      
      if (!navigator.mediaDevices) {
        throw new Error('浏览器不支持媒体设备 API');
      }
      
      if (!navigator.mediaDevices.getDisplayMedia) {
        throw new Error('浏览器不支持屏幕共享 API');
      }
      
      if (!window.RTCPeerConnection) {
        throw new Error('浏览器不支持 WebRTC API');
      }
      
      if (!window.electron || !window.electron.ipcRenderer) {
        throw new Error('Electron IPC 不可用');
      }
      
      // 获取屏幕共享流 - 使用 Electron 的 desktopCapturer API
      logger.log('请求屏幕共享流...');
      try {
        // 发送请求到主进程获取屏幕源
        window.electron.ipcRenderer.send('start-screen-share');
        
        // 等待屏幕源信息
        const sources = await new Promise((resolve, reject) => {
          window.electron.ipcRenderer.once('screen-sources', (event, sources) => {
            resolve(sources);
          });
        });
        
        logger.log('收到屏幕源:', sources);
        
        if (sources.length === 0) {
          throw new Error('没有可用的屏幕源');
        }
        
        // 使用第一个屏幕源
        const source = sources[0];
        logger.log('选择屏幕源:', source.name, source.id);
        
        // 使用 getUserMedia 获取屏幕共享流
        this.screenStream = await navigator.mediaDevices.getUserMedia({
          audio: false,
          video: {
            mandatory: {
              chromeMediaSource: 'desktop',
              chromeMediaSourceId: source.id
            }
          }
        });
        
        logger.log('屏幕共享流获取成功:', this.screenStream);
        logger.log('屏幕流轨道:', this.screenStream.getTracks());
      } catch (error) {
        console.error('获取屏幕共享流失败:', error);
        console.error('错误类型:', error.name);
        console.error('错误消息:', error.message);
        throw error;
      }

      // 创建 RTCPeerConnection
      logger.log('创建 RTCPeerConnection...');
      try {
        let peerConfig = {};
        
        // 使用 60443 端口
        const portRangeBegin = 60443;
        const portRangeEnd = 60443;
        
        // 先尝试直连
        if (this.enableDirectConnect) {
          logger.log('尝试直连模式...');
          // 不使用 ICE 服务器，尝试直接连接
          // 但仍然配置端口范围
          peerConfig = {
            // 尝试配置端口范围
            iceCandidatePortRange: {
              min: portRangeBegin,
              max: portRangeEnd
            },
            iceTransportPolicy: 'all',
            iceCandidatePoolSize: 10
          };
          logger.log('直连模式端口范围:', portRangeBegin, '-', portRangeEnd);
        } else {
          // 添加 ICE 服务器配置
          const iceServers = [
            {
              urls: 'stun:stun.l.google.com:19302'
            },
            {
              urls: 'stun:stun1.l.google.com:19302'
            },
            {
              urls: 'stun:stun2.l.google.com:19302'
            }
          ];
          
          peerConfig = { 
            iceServers,
            // 添加端口范围配置
            // 注意：不是所有浏览器都支持这个选项
            iceTransportPolicy: 'all',
            iceCandidatePoolSize: 10,
            // 尝试配置端口范围
            iceCandidatePortRange: {
              min: portRangeBegin,
              max: portRangeEnd
            }
          };
          
          logger.log('使用 ICE 服务器模式...');
          logger.log('ICE 服务器配置:', iceServers);
          logger.log('ICE 端口范围:', portRangeBegin, '-', portRangeEnd);
        }
        
        this.peerConnection = new RTCPeerConnection(peerConfig);
        logger.log('RTCPeerConnection 创建成功:', this.peerConnection);
      } catch (error) {
        console.error('创建 RTCPeerConnection 失败:', error);
        console.error('错误类型:', error.name);
        console.error('错误消息:', error.message);
        throw error;
      }

      // 添加屏幕流到连接
      logger.log('添加屏幕流到连接...');
      try {
        const tracks = this.screenStream.getTracks();
        logger.log('屏幕流轨道数量:', tracks.length);
        tracks.forEach(track => {
          logger.log('添加轨道:', track.kind, track.id);
          try {
            this.peerConnection.addTrack(track, this.screenStream);
            logger.log('轨道添加成功');
          } catch (error) {
            console.error('添加轨道失败:', error);
          }
        });
      } catch (error) {
        console.error('添加屏幕流到连接失败:', error);
        console.error('错误类型:', error.name);
        console.error('错误消息:', error.message);
        throw error;
      }

      // 处理 ICE 候选者
      logger.log('设置 ICE 候选者处理...');
      this.peerConnection.onicecandidate = (event) => {
        if (event.candidate) {
          logger.log('生成 ICE 候选者:', event.candidate);
          // 发送 ICE 候选者到接收方
          try {
            // 尝试使用全局 WebSocket 连接
            if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
              window.ws.send(JSON.stringify({
                type: 'webrtc_ice_candidate',
                data: {
                  target_user_id: receiverId,
                  candidate: event.candidate
                }
              }));
              logger.log('ICE 候选者发送成功');
            } else if (window.electron && window.electron.websocket) {
              // 回退到 IPC 方式
              window.electron.websocket.send({
                type: 'webrtc_ice_candidate',
                data: {
                  target_user_id: receiverId,
                  candidate: event.candidate
                }
              });
              logger.log('ICE 候选者发送成功（通过 IPC）');
            } else {
              console.error('WebSocket 连接不可用');
            }
          } catch (error) {
            console.error('发送 ICE 候选者失败:', error);
          }
        } else {
          logger.log('ICE 候选者收集完成（event.candidate 为 null）');
        }
      };

      // 处理 ICE 连接状态变化
      this.peerConnection.oniceconnectionstatechange = () => {
        logger.log('ICE 连接状态变化:', this.peerConnection.iceConnectionState);
        
        // 如果直连失败，尝试使用 ICE 服务器
        if (this.enableDirectConnect && 
            (this.peerConnection.iceConnectionState === 'failed' || 
             this.peerConnection.iceConnectionState === 'disconnected')) {
          logger.log('直连失败，尝试使用 ICE 服务器...');
          this.enableDirectConnect = false;
          
          // 重新创建 RTCPeerConnection，使用 ICE 服务器
          // 这里需要重新开始连接流程
        }
      };

      // 处理连接状态变化
      this.peerConnection.onconnectionstatechange = () => {
        logger.log('连接状态变化:', this.peerConnection.connectionState);
        
        if (this.peerConnection.connectionState === 'connected') {
          logger.log('WebRTC 连接已建立');
        } else if (this.peerConnection.connectionState === 'disconnected') {
          logger.log('WebRTC 连接已断开');
        } else if (this.peerConnection.connectionState === 'failed') {
          logger.log('WebRTC 连接失败');
        }
      };

      // 生成 offer
      logger.log('生成 offer...');
      let offer;
      try {
        offer = await this.peerConnection.createOffer();
        logger.log('offer 生成成功:', offer);
        
        logger.log('设置本地描述...');
        await this.peerConnection.setLocalDescription(offer);
        logger.log('本地描述设置成功');
      } catch (error) {
        console.error('生成 offer 或设置本地描述失败:', error);
        console.error('错误类型:', error.name);
        console.error('错误消息:', error.message);
        throw error;
      }

      // 发送 offer 到接收方
      logger.log('发送 offer 到接收方...');
      try {
        // 尝试使用全局 WebSocket 连接
        if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
          window.ws.send(JSON.stringify({
            type: 'webrtc_offer',
            data: {
              target_user_id: receiverId,
              signal: offer,
              share_type: 'screen'  // 标识为屏幕共享
            }
          }));
          logger.log('offer 发送成功');
        } else if (window.electron && window.electron.websocket) {
          // 回退到 IPC 方式
          window.electron.websocket.send({
            type: 'webrtc_offer',
            data: {
              target_user_id: receiverId,
              signal: offer,
              share_type: 'screen'  // 标识为屏幕共享
            }
          });
          logger.log('offer 发送成功（通过 IPC）');
        } else {
          console.error('WebSocket 连接不可用');
          throw new Error('WebSocket 连接不可用');
        }
      } catch (error) {
        console.error('发送 offer 失败:', error);
        console.error('错误类型:', error.name);
        console.error('错误消息:', error.message);
        throw error;
      }

      this.isSharing = true;
      logger.log('屏幕共享已开始');
      return true;
    } catch (error) {
      console.error('开始屏幕共享失败:', error);
      console.error('错误类型:', error.name);
      console.error('错误消息:', error.message);
      console.error('错误堆栈:', error.stack);
      throw error;
    }
  }

  // 处理 answer
  async handleAnswer(answer) {
    try {
      console.log('=== ScreenShareSender.handleAnswer 被调用 ===')
      console.log('answer:', answer)
      console.log('peerConnection 是否存在:', !!this.peerConnection)
      if (this.peerConnection) {
        console.log('当前连接状态:', this.peerConnection.connectionState)
        console.log('信令状态:', this.peerConnection.signalingState)
        console.log('远程描述是否已设置:', !!this.peerConnection.remoteDescription)
        
        // 检查信令状态，只有在 have-local-offer 状态下才能设置 answer
        if (this.peerConnection.signalingState !== 'have-local-offer') {
          console.warn('当前信令状态不是 have-local-offer，跳过设置 answer');
          console.warn('当前状态:', this.peerConnection.signalingState);
          return;
        }
        
        await this.peerConnection.setRemoteDescription(new RTCSessionDescription(answer));
        logger.log('远程描述设置成功');
        console.log('设置后的连接状态:', this.peerConnection.connectionState)
        console.log('设置后的信令状态:', this.peerConnection.signalingState)
        
        // 处理缓存的 ICE candidates
        if (this.iceCandidateCache.length > 0) {
          console.log('处理缓存的 ICE candidates:', this.iceCandidateCache.length, '个')
          for (const candidate of this.iceCandidateCache) {
            try {
              const iceCandidate = new RTCIceCandidate({
                candidate: candidate.candidate,
                sdpMid: candidate.sdpMid || '',
                sdpMLineIndex: candidate.sdpMLineIndex || 0
              });
              await this.peerConnection.addIceCandidate(iceCandidate);
            } catch (e) {
              console.error('添加缓存的 ICE candidate 失败:', e);
            }
          }
          this.iceCandidateCache = [];
          console.log('所有缓存的 ICE candidates 处理完成')
        }
      } else {
        console.error('peerConnection 不存在，无法设置 answer')
      }
    } catch (error) {
      console.error('处理 answer 失败:', error);
    }
  }

  // 处理 ICE 候选者
  addIceCandidate(candidate) {
    try {
      if (!candidate || !candidate.candidate) {
        logger.log('无效的 ICE 候选者，跳过:', candidate);
        return;
      }
      
      if (!this.peerConnection) {
        logger.log('peerConnection 不存在，缓存 ICE 候选者');
        this.iceCandidateCache.push(candidate);
        return;
      }
      
      // 检查远程描述是否已设置
      if (this.peerConnection.remoteDescription) {
        // 构造 RTCIceCandidate 对象
        const iceCandidate = new RTCIceCandidate({
          candidate: candidate.candidate,
          sdpMid: candidate.sdpMid || '',
          sdpMLineIndex: candidate.sdpMLineIndex || 0
        });
        this.peerConnection.addIceCandidate(iceCandidate);
        logger.log('ICE 候选者添加成功:', iceCandidate);
      } else {
        // 远程描述未设置，缓存候选者
        logger.log('远程描述未设置，缓存 ICE 候选者');
        this.iceCandidateCache.push(candidate);
      }
    } catch (error) {
      console.error('添加 ICE 候选者失败:', error);
      console.error('候选者数据:', candidate);
    }
  }

  // 停止屏幕共享
  stopScreenShare() {
    if (this.screenStream) {
      this.screenStream.getTracks().forEach(track => track.stop());
      this.screenStream = null;
    }

    if (this.peerConnection) {
      this.peerConnection.close();
      this.peerConnection = null;
    }

    this.isSharing = false;
    this.receiverId = null;
    logger.log('屏幕共享已停止');
  }

  // 检查是否正在共享
  getIsSharing() {
    return this.isSharing;
  }

  // 获取屏幕共享流
  getScreenStream() {
    return this.screenStream;
  }

  // 发送屏幕共享请求
  sendShareRequest(receiverId, conversationId) {
    logger.log('发送屏幕共享请求，接收者ID:', receiverId, '会话ID:', conversationId);
    try {
      // 尝试使用全局 WebSocket 连接
      if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
        window.ws.send(JSON.stringify({
        type: 'screen-share-request',
        data: {
          target_user_id: receiverId,
          requester_id: window.currentUser?.id || 0,
          from_user_id: window.currentUser?.id || 0,
          from_user_name: window.currentUser?.name || window.currentUser?.nickname || '未知用户',
          conversation_id: conversationId || receiverId
        }
      }));
        logger.log('屏幕共享请求发送成功');
      } else if (window.electron && window.electron.websocket) {
        // 回退到 IPC 方式
        window.electron.websocket.send({
          type: 'screen-share-request',
          data: {
            target_user_id: receiverId,
            requester_id: window.currentUser?.id || 0,
            from_user_id: window.currentUser?.id || 0,
            from_user_name: window.currentUser?.name || window.currentUser?.nickname || '未知用户',
            conversation_id: conversationId || receiverId
          }
        });
        logger.log('屏幕共享请求发送成功（通过 IPC）');
      } else {
        console.error('WebSocket 连接不可用');
      }
    } catch (error) {
      console.error('发送屏幕共享请求失败:', error);
    }
  }

  // 使用已有的屏幕流开始屏幕共享
  async startScreenShareWithStream(receiverId, stream) {
    try {
      this.receiverId = receiverId;
      this.screenStream = stream;
      
      logger.log('使用已有的屏幕流开始屏幕共享');
      logger.log('屏幕流:', stream);
      logger.log('屏幕流轨道:', stream.getTracks());
      
      // 检查浏览器支持
      logger.log('检查浏览器支持...');
      logger.log('window.RTCPeerConnection:', window.RTCPeerConnection);
      logger.log('window.electron:', window.electron);
      
      if (!window.RTCPeerConnection) {
        throw new Error('浏览器不支持 WebRTC API');
      }
      
      if (!window.electron || !window.electron.ipcRenderer) {
        throw new Error('Electron IPC 不可用');
      }

      // 创建 RTCPeerConnection
      logger.log('创建 RTCPeerConnection...');
      try {
        let peerConfig = {};
        
        // 使用 60443 端口
        const portRangeBegin = 60443;
        const portRangeEnd = 60443;
        
        // 先尝试直连
        if (this.enableDirectConnect) {
          logger.log('尝试直连模式...');
          // 不使用 ICE 服务器，尝试直接连接
          // 但仍然配置端口范围
          peerConfig = {
            // 尝试配置端口范围
            iceCandidatePortRange: {
              min: portRangeBegin,
              max: portRangeEnd
            },
            iceTransportPolicy: 'all',
            iceCandidatePoolSize: 10
          };
          logger.log('直连模式端口范围:', portRangeBegin, '-', portRangeEnd);
        } else {
          // 添加 ICE 服务器配置
          const iceServers = [
            {
              urls: 'stun:stun.l.google.com:19302'
            },
            {
              urls: 'stun:stun1.l.google.com:19302'
            },
            {
              urls: 'stun:stun2.l.google.com:19302'
            }
          ];
          
          peerConfig = { 
            iceServers,
            // 添加端口范围配置
            // 注意：不是所有浏览器都支持这个选项
            iceTransportPolicy: 'all',
            iceCandidatePoolSize: 10,
            // 尝试配置端口范围
            iceCandidatePortRange: {
              min: portRangeBegin,
              max: portRangeEnd
            }
          };
          
          logger.log('使用 ICE 服务器模式...');
          logger.log('ICE 服务器配置:', iceServers);
          logger.log('ICE 端口范围:', portRangeBegin, '-', portRangeEnd);
        }
        
        this.peerConnection = new RTCPeerConnection(peerConfig);
        logger.log('RTCPeerConnection 创建成功:', this.peerConnection);
      } catch (error) {
        console.error('创建 RTCPeerConnection 失败:', error);
        console.error('错误类型:', error.name);
        console.error('错误消息:', error.message);
        throw error;
      }

      // 添加屏幕流到连接
      logger.log('添加屏幕流到连接...');
      try {
        const tracks = this.screenStream.getTracks();
        logger.log('屏幕流轨道数量:', tracks.length);
        tracks.forEach(track => {
          logger.log('添加轨道:', track.kind, track.id);
          try {
            this.peerConnection.addTrack(track, this.screenStream);
            logger.log('轨道添加成功');
          } catch (error) {
            console.error('添加轨道失败:', error);
          }
        });
      } catch (error) {
        console.error('添加屏幕流到连接失败:', error);
        console.error('错误类型:', error.name);
        console.error('错误消息:', error.message);
        throw error;
      }

      // 处理 ICE 候选者
      logger.log('设置 ICE 候选者处理...');
      this.peerConnection.onicecandidate = (event) => {
        if (event.candidate) {
          logger.log('生成 ICE 候选者:', event.candidate);
          // 发送 ICE 候选者到接收方
          try {
            // 尝试使用全局 WebSocket 连接
            if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
              window.ws.send(JSON.stringify({
                type: 'webrtc_ice_candidate',
                data: {
                  target_user_id: receiverId,
                  candidate: event.candidate
                }
              }));
              logger.log('ICE 候选者发送成功');
            } else if (window.electron && window.electron.websocket) {
              // 回退到 IPC 方式
              window.electron.websocket.send({
                type: 'webrtc_ice_candidate',
                data: {
                  target_user_id: receiverId,
                  candidate: event.candidate
                }
              });
              logger.log('ICE 候选者发送成功（通过 IPC）');
            } else {
              console.error('WebSocket 连接不可用');
            }
          } catch (error) {
            console.error('发送 ICE 候选者失败:', error);
          }
        }
      };

      // 处理 ICE 连接状态变化
      this.peerConnection.oniceconnectionstatechange = () => {
        logger.log('ICE 连接状态变化:', this.peerConnection.iceConnectionState);
        
        // 如果直连失败，尝试使用 ICE 服务器
        if (this.enableDirectConnect && 
            (this.peerConnection.iceConnectionState === 'failed' || 
             this.peerConnection.iceConnectionState === 'disconnected')) {
          logger.log('直连失败，尝试使用 ICE 服务器...');
          this.enableDirectConnect = false;
          
          // 重新创建 RTCPeerConnection，使用 ICE 服务器
          // 这里需要重新开始连接流程
        }
      };

      // 处理连接状态变化
      this.peerConnection.onconnectionstatechange = () => {
        logger.log('连接状态变化:', this.peerConnection.connectionState);
        
        if (this.peerConnection.connectionState === 'connected') {
          logger.log('WebRTC 连接已建立');
        } else if (this.peerConnection.connectionState === 'disconnected') {
          logger.log('WebRTC 连接已断开');
        } else if (this.peerConnection.connectionState === 'failed') {
          logger.log('WebRTC 连接失败');
        }
      };

      // 生成 offer
      logger.log('生成 offer...');
      let offer;
      try {
        offer = await this.peerConnection.createOffer();
        logger.log('offer 生成成功:', offer);
        
        logger.log('设置本地描述...');
        await this.peerConnection.setLocalDescription(offer);
        logger.log('本地描述设置成功');
      } catch (error) {
        console.error('生成 offer 或设置本地描述失败:', error);
        console.error('错误类型:', error.name);
        console.error('错误消息:', error.message);
        throw error;
      }

      // 发送 offer 到接收方
      logger.log('发送 offer 到接收方...');
      try {
        // 尝试使用全局 WebSocket 连接
        if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
          window.ws.send(JSON.stringify({
            type: 'webrtc_offer',
            data: {
              target_user_id: receiverId,
              signal: offer
            }
          }));
          logger.log('offer 发送成功');
        } else if (window.electron && window.electron.websocket) {
          // 回退到 IPC 方式
          window.electron.websocket.send({
            type: 'webrtc_offer',
            data: {
              target_user_id: receiverId,
              signal: offer
            }
          });
          logger.log('offer 发送成功（通过 IPC）');
        } else {
          console.error('WebSocket 连接不可用');
          throw new Error('WebSocket 连接不可用');
        }
      } catch (error) {
        console.error('发送 offer 失败:', error);
        console.error('错误类型:', error.name);
        console.error('错误消息:', error.message);
        throw error;
      }

      this.isSharing = true;
      logger.log('屏幕共享已开始');
      return true;
    } catch (error) {
      console.error('开始屏幕共享失败:', error);
      console.error('错误类型:', error.name);
      console.error('错误消息:', error.message);
      console.error('错误堆栈:', error.stack);
      throw error;
    }
  }
}

class ScreenShareReceiver {
  constructor() {
    this.peerConnection = null;
    this.remoteStream = null;
    this.videoElement = null;
    this.senderId = null;
    this.enableDirectConnect = true; // 启用直连尝试
    this.onStreamReceived = null; // 远程流接收回调
    this.iceCandidateCache = []; // ICE 候选者缓存
  }

  // 初始化
  init(videoElement, onStreamReceived) {
    this.videoElement = videoElement;
    this.onStreamReceived = onStreamReceived;
    logger.log('ScreenShareReceiver 初始化成功，视频元素:', videoElement);
  }

  // 处理 offer
  async handleOffer(offer, senderId) {
    try {
      this.senderId = senderId;
      logger.log('处理 offer，发送者 ID:', senderId);
      logger.log('offer:', offer);
      
      // 创建 RTCPeerConnection
      logger.log('创建 RTCPeerConnection...');
      try {
        let peerConfig = {};
        
        // 先尝试直连
        if (this.enableDirectConnect) {
          logger.log('尝试直连模式...');
          // 不使用 ICE 服务器，尝试直接连接
        } else {
          // 添加 ICE 服务器配置
          const iceServers = [
            {
              urls: 'stun:stun.l.google.com:19302'
            },
            {
              urls: 'stun:stun1.l.google.com:19302'
            },
            {
              urls: 'stun:stun2.l.google.com:19302'
            }
          ];
          
          // 使用 60443 端口
          const portRangeBegin = 60443;
          const portRangeEnd = 60443;
          
          peerConfig = { 
            iceServers,
            // 添加端口范围配置
            // 注意：不是所有浏览器都支持这个选项
            iceTransportPolicy: 'all',
            iceCandidatePoolSize: 10,
            // 尝试配置端口范围
            iceCandidatePortRange: {
              min: portRangeBegin,
              max: portRangeEnd
            }
          };
          
          logger.log('使用 ICE 服务器模式...');
          logger.log('ICE 服务器配置:', iceServers);
          logger.log('ICE 端口范围:', portRangeBegin, '-', portRangeEnd);
        }
        
        this.peerConnection = new RTCPeerConnection(peerConfig);
        logger.log('RTCPeerConnection 创建成功:', this.peerConnection);
      } catch (error) {
        console.error('创建 RTCPeerConnection 失败:', error);
        throw error;
      }

      // 处理远程流
      this.peerConnection.ontrack = (event) => {
        logger.log('收到远程流事件:', event);
        logger.log('远程流数量:', event.streams.length);
        
        if (event.streams && event.streams.length > 0) {
          this.remoteStream = event.streams[0];
          logger.log('远程流:', this.remoteStream);
          logger.log('流ID:', this.remoteStream.id);
          logger.log('轨道数量:', this.remoteStream.getTracks().length);
          
          // 调用远程流接收回调（让调用方决定如何处理流）
          if (this.onStreamReceived) {
            logger.log('调用远程流接收回调');
            this.onStreamReceived(this.remoteStream);
          } else if (this.videoElement) {
            // 如果没有回调，直接设置视频元素
            logger.log('设置视频元素的 srcObject');
            this.videoElement.srcObject = this.remoteStream;
            logger.log('视频元素 srcObject 已设置');
            
            // 尝试播放视频
            try {
              this.videoElement.play().catch(err => {
                console.error('尝试播放视频失败:', err);
              });
            } catch (error) {
              console.error('播放视频出错:', error);
            }
          } else {
            console.error('视频元素未初始化，无法设置 srcObject');
          }
        } else {
          console.error('没有收到远程流');
        }
      };

      // 处理 ICE 候选者
      this.peerConnection.onicecandidate = (event) => {
        if (event.candidate) {
          logger.log('生成 ICE 候选者:', event.candidate);
          // 发送 ICE 候选者到发送方
          try {
            // 尝试使用全局 WebSocket 连接
            if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
              window.ws.send(JSON.stringify({
                type: 'webrtc_ice_candidate',
                data: {
                  target_user_id: senderId,
                  candidate: event.candidate
                }
              }));
              logger.log('ICE 候选者发送成功');
            } else if (window.electron && window.electron.websocket) {
              // 回退到 IPC 方式
              window.electron.websocket.send({
                type: 'webrtc_ice_candidate',
                data: {
                  target_user_id: senderId,
                  candidate: event.candidate
                }
              });
              logger.log('ICE 候选者发送成功（通过 IPC）');
            } else {
              console.error('WebSocket 连接不可用');
            }
          } catch (error) {
            console.error('发送 ICE 候选者失败:', error);
          }
        }
      };

      // 处理 ICE 连接状态变化
      this.peerConnection.oniceconnectionstatechange = () => {
        logger.log('ICE 连接状态变化:', this.peerConnection.iceConnectionState);
        
        // 如果直连失败，尝试使用 ICE 服务器
        if (this.enableDirectConnect && 
            (this.peerConnection.iceConnectionState === 'failed' || 
             this.peerConnection.iceConnectionState === 'disconnected')) {
          logger.log('直连失败，尝试使用 ICE 服务器...');
          this.enableDirectConnect = false;
          
          // 重新创建 RTCPeerConnection，使用 ICE 服务器
          // 这里需要重新开始连接流程
        }
      };

      // 处理连接状态变化
      this.peerConnection.onconnectionstatechange = () => {
        logger.log('连接状态变化:', this.peerConnection.connectionState);
        
        if (this.peerConnection.connectionState === 'connected') {
          logger.log('WebRTC 连接已建立');
        } else if (this.peerConnection.connectionState === 'disconnected') {
          logger.log('WebRTC 连接已断开');
        } else if (this.peerConnection.connectionState === 'failed') {
          logger.log('WebRTC 连接失败');
        }
      };

      // 设置远程描述
      logger.log('设置远程描述...');
      try {
        await this.peerConnection.setRemoteDescription(new RTCSessionDescription(offer));
        logger.log('远程描述设置成功');
        // 刷新缓存的 ICE 候选者
        this.flushIceCandidates();
      } catch (error) {
        console.error('设置远程描述失败:', error);
        throw error;
      }

      // 生成 answer
      logger.log('生成 answer...');
      let answer;
      try {
        answer = await this.peerConnection.createAnswer();
        logger.log('answer 生成成功:', answer);
        
        logger.log('设置本地描述...');
        await this.peerConnection.setLocalDescription(answer);
        logger.log('本地描述设置成功');
      } catch (error) {
        console.error('生成 answer 或设置本地描述失败:', error);
        throw error;
      }

      // 发送 answer 到发送方
      logger.log('发送 answer 到发送方...');
      try {
        // 尝试使用全局 WebSocket 连接
        if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
          window.ws.send(JSON.stringify({
            type: 'webrtc_answer',
            data: {
              target_user_id: senderId,
              signal: answer
            }
          }));
          logger.log('answer 发送成功');
        } else if (window.electron && window.electron.websocket) {
          // 回退到 IPC 方式
          window.electron.websocket.send({
            type: 'webrtc_answer',
            data: {
              target_user_id: senderId,
              signal: answer
            }
          });
          logger.log('answer 发送成功（通过 IPC）');
        } else {
          console.error('WebSocket 连接不可用');
          throw new Error('WebSocket 连接不可用');
        }
      } catch (error) {
        console.error('发送 answer 失败:', error);
        throw error;
      }

      logger.log('屏幕共享连接已建立');
      return true;
    } catch (error) {
      console.error('处理屏幕共享 offer 失败:', error);
      console.error('错误类型:', error.name);
      console.error('错误消息:', error.message);
      throw error;
    }
  }

  // 处理 ICE 候选者
  addIceCandidate(candidate) {
    try {
      if (this.peerConnection && candidate) {
        // 验证 candidate 对象是否有效
        if (candidate.candidate) {
          // 检查远程描述是否已设置
          if (this.peerConnection.remoteDescription) {
            // 构造 RTCIceCandidate 对象
            const iceCandidate = new RTCIceCandidate({
              candidate: candidate.candidate,
              sdpMid: candidate.sdpMid || '',
              sdpMLineIndex: candidate.sdpMLineIndex || 0
            });
            this.peerConnection.addIceCandidate(iceCandidate);
            logger.log('ICE 候选者添加成功:', iceCandidate);
          } else {
            // 远程描述未设置，缓存 ICE 候选者
            logger.log('远程描述未设置，缓存 ICE 候选者:', candidate);
            this.iceCandidateCache.push(candidate);
          }
        } else {
          logger.log('无效的 ICE 候选者，跳过:', candidate);
        }
      }
    } catch (error) {
      console.error('添加 ICE 候选者失败:', error);
      console.error('候选者数据:', candidate);
    }
  }

  // 刷新缓存的 ICE 候选者
  flushIceCandidates() {
    if (this.peerConnection && this.peerConnection.remoteDescription) {
      logger.log('刷新缓存的 ICE 候选者，数量:', this.iceCandidateCache.length);
      while (this.iceCandidateCache.length > 0) {
        const candidate = this.iceCandidateCache.shift();
        try {
          const iceCandidate = new RTCIceCandidate({
            candidate: candidate.candidate,
            sdpMid: candidate.sdpMid || '',
            sdpMLineIndex: candidate.sdpMLineIndex || 0
          });
          this.peerConnection.addIceCandidate(iceCandidate);
          logger.log('缓存的 ICE 候选者添加成功:', iceCandidate);
        } catch (error) {
          console.error('添加缓存的 ICE 候选者失败:', error);
          console.error('候选者数据:', candidate);
        }
      }
    }
  }

  // 停止接收屏幕共享
  stop() {
    if (this.peerConnection) {
      this.peerConnection.close();
      this.peerConnection = null;
    }

    if (this.videoElement) {
      this.videoElement.srcObject = null;
    }

    this.remoteStream = null;
    this.senderId = null;
    logger.log('屏幕共享已停止接收');
  }

  // 处理屏幕共享请求
  handleShareRequest(data) {
    logger.log('处理屏幕共享请求:', data);
    // 这里可以添加处理逻辑
  }

  // 接受屏幕共享请求
  async acceptShareRequest(conversationId) {
    logger.log('接受屏幕共享请求，会话ID:', conversationId);
    // 发送接受响应
    try {
      // 尝试使用全局 WebSocket 连接
      if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
        window.ws.send(JSON.stringify({
          type: 'screen-share-response',
          data: {
            conversation_id: conversationId,
            status: 'accepted'
          }
        }));
        logger.log('屏幕共享接受响应发送成功');
      } else if (window.electron && window.electron.websocket) {
        // 回退到 IPC 方式
        window.electron.websocket.send({
          type: 'screen-share-response',
          data: {
            conversation_id: conversationId,
            status: 'accepted'
          }
        });
        logger.log('屏幕共享接受响应发送成功（通过 IPC）');
      } else {
        console.error('WebSocket 连接不可用');
      }
    } catch (error) {
      console.error('发送屏幕共享接受响应失败:', error);
    }
  }

  // 拒绝屏幕共享请求
  rejectShareRequest(conversationId) {
    logger.log('拒绝屏幕共享请求，会话ID:', conversationId);
    // 发送拒绝响应
    try {
      // 尝试使用全局 WebSocket 连接
      if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
        window.ws.send(JSON.stringify({
          type: 'screen-share-response',
          data: {
            conversation_id: conversationId,
            status: 'rejected'
          }
        }));
        logger.log('屏幕共享拒绝响应发送成功');
      } else if (window.electron && window.electron.websocket) {
        // 回退到 IPC 方式
        window.electron.websocket.send({
          type: 'screen-share-response',
          data: {
            conversation_id: conversationId,
            status: 'rejected'
          }
        });
        logger.log('屏幕共享拒绝响应发送成功（通过 IPC）');
      } else {
        console.error('WebSocket 连接不可用');
      }
    } catch (error) {
      console.error('发送屏幕共享拒绝响应失败:', error);
    }
  }
}

// WebRTC 视频通话工具类

class VideoCallSender {
  constructor() {
    this.peerConnection = null;
    this.localStream = null;
    this.isCalling = false;
    this.receiverId = null;
    this.enableDirectConnect = true;
    this.onAnswered = null;
    this.onHangup = null;
    this.onError = null;
  }

  async startVideoCall(receiverId, { video = true, audio = true } = {}) {
    try {
      this.receiverId = receiverId;

      logger.log('检查浏览器支持...');
      logger.log('navigator.mediaDevices:', navigator.mediaDevices);
      logger.log('navigator.mediaDevices.getUserMedia:', navigator.mediaDevices?.getUserMedia);
      logger.log('window.RTCPeerConnection:', window.RTCPeerConnection);
      
      if (!navigator.mediaDevices) {
        throw new Error('浏览器不支持媒体设备 API');
      }
      
      if (!navigator.mediaDevices.getUserMedia) {
        throw new Error('浏览器不支持 getUserMedia API');
      }
      
      if (!window.RTCPeerConnection) {
        throw new Error('浏览器不支持 WebRTC API');
      }

      logger.log('获取本地媒体流...');
      try {
        this.localStream = await navigator.mediaDevices.getUserMedia({
          video,
          audio
        });
        logger.log('本地媒体流获取成功');
      } catch (error) {
        console.error('获取本地媒体流失败:', error);
        if (error.name === 'NotAllowedError') {
          throw new Error('用户拒绝了摄像头或麦克风权限');
        } else if (error.name === 'NotFoundError') {
          throw new Error('未找到摄像头或麦克风设备');
        } else {
          throw error;
        }
      }

      logger.log('创建 RTCPeerConnection...');
      const peerConfig = this.getPeerConfig();
      this.peerConnection = new RTCPeerConnection(peerConfig);

      logger.log('添加本地轨道到连接...');
      const tracks = this.localStream.getTracks();
      tracks.forEach(track => {
        this.peerConnection.addTrack(track, this.localStream);
      });

      this.setupPeerConnectionHandlers(receiverId);

      logger.log('生成 offer...');
      const offer = await this.peerConnection.createOffer();
      await this.peerConnection.setLocalDescription(offer);

      logger.log('发送 offer 到接收方...');
      await this.sendSignalingMessage('webrtc_offer', {
        target_user_id: receiverId,
        signal: offer,
        call_type: video ? 'video' : 'audio'
      });

      this.isCalling = true;
      logger.log('视频通话已开始');
      return true;
    } catch (error) {
      console.error('开始视频通话失败:', error);
      this.cleanup();
      if (this.onError) {
        this.onError(error.message || '通话失败');
      }
      throw error;
    }
  }

  async handleAnswer(answer) {
    try {
      if (this.peerConnection) {
        // 检查信令状态，只有在 have-local-offer 状态下才能设置 answer
        if (this.peerConnection.signalingState !== 'have-local-offer') {
          console.warn('当前信令状态不是 have-local-offer，跳过设置 answer');
          console.warn('当前状态:', this.peerConnection.signalingState);
          return;
        }
        
        await this.peerConnection.setRemoteDescription(new RTCSessionDescription(answer));
        logger.log('远程描述设置成功');
      }
    } catch (error) {
      console.error('处理 answer 失败:', error);
      throw error;
    }
  }

  addIceCandidate(candidate) {
    try {
      if (this.peerConnection && candidate?.candidate && this.peerConnection.remoteDescription) {
        const iceCandidate = new RTCIceCandidate({
          candidate: candidate.candidate,
          sdpMid: candidate.sdpMid || '',
          sdpMLineIndex: candidate.sdpMLineIndex || 0
        });
        this.peerConnection.addIceCandidate(iceCandidate);
        logger.log('ICE 候选者添加成功');
      }
    } catch (error) {
      console.error('添加 ICE 候选者失败:', error);
    }
  }

  hangup() {
    this.sendSignalingMessage('webrtc_hangup', {
      target_user_id: this.receiverId
    }).catch(() => {});
    this.cleanup();
  }

  cleanup() {
    if (this.localStream) {
      this.localStream.getTracks().forEach(track => track.stop());
      this.localStream = null;
    }

    if (this.peerConnection) {
      this.peerConnection.close();
      this.peerConnection = null;
    }

    this.isCalling = false;
    this.receiverId = null;
    logger.log('通话已清理');
  }

  getPeerConfig() {
    const portRangeBegin = 60443;
    const portRangeEnd = 60443;
    
    if (this.enableDirectConnect) {
      return {
        iceCandidatePortRange: { min: portRangeBegin, max: portRangeEnd },
        iceTransportPolicy: 'all',
        iceCandidatePoolSize: 10
      };
    } else {
      return {
        iceServers: [
          { urls: 'stun:stun.l.google.com:19302' },
          { urls: 'stun:stun1.l.google.com:19302' },
          { urls: 'stun:stun2.l.google.com:19302' }
        ],
        iceTransportPolicy: 'all',
        iceCandidatePoolSize: 10,
        iceCandidatePortRange: { min: portRangeBegin, max: portRangeEnd }
      };
    }
  }

  setupPeerConnectionHandlers(receiverId) {
    this.peerConnection.onicecandidate = async (event) => {
      if (event.candidate) {
        await this.sendSignalingMessage('webrtc_ice_candidate', {
          target_user_id: receiverId,
          candidate: event.candidate
        });
      }
    };

    this.peerConnection.oniceconnectionstatechange = () => {
      const state = this.peerConnection.iceConnectionState;
      logger.log('ICE 连接状态变化:', state);
      
      if (this.enableDirectConnect && (state === 'failed' || state === 'disconnected')) {
        logger.log('直连失败，尝试使用 ICE 服务器...');
        this.enableDirectConnect = false;
      }
    };

    this.peerConnection.onconnectionstatechange = () => {
      const state = this.peerConnection.connectionState;
      logger.log('连接状态变化:', state);
      
      if (state === 'connected') {
        logger.log('WebRTC 连接已建立');
        if (this.onAnswered) {
          this.onAnswered();
        }
      } else if (state === 'disconnected' || state === 'failed') {
        logger.log('WebRTC 连接已断开');
        this.cleanup();
        if (this.onHangup) {
          this.onHangup();
        }
      }
    };
  }

  async sendSignalingMessage(type, data) {
    try {
      if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
        window.ws.send(JSON.stringify({ type, data }));
        logger.log(`${type} 发送成功`);
      } else if (window.electron && window.electron.websocket) {
        window.electron.websocket.send({ type, data });
        logger.log(`${type} 发送成功（通过 IPC）`);
      } else {
        throw new Error('WebSocket 连接不可用');
      }
    } catch (error) {
      console.error(`发送 ${type} 失败:`, error);
      throw error;
    }
  }

  getIsCalling() {
    return this.isCalling;
  }

  getLocalStream() {
    return this.localStream;
  }

  setCallbacks(callbacks) {
    if (callbacks.onAnswered) this.onAnswered = callbacks.onAnswered;
    if (callbacks.onHangup) this.onHangup = callbacks.onHangup;
    if (callbacks.onError) this.onError = callbacks.onError;
  }
}

class VideoCallReceiver {
  constructor() {
    this.peerConnection = null;
    this.localStream = null;
    this.remoteStream = null;
    this.senderId = null;
    this.enableDirectConnect = true;
    this.iceCandidateCache = [];
    this.onStreamReceived = null;
    this.onCallReceived = null;
    this.onHangup = null;
    this.onError = null;
  }

  async handleOffer(offer, senderId, callType = 'video') {
    try {
      this.senderId = senderId;
      logger.log('处理 offer，发送者 ID:', senderId);

      logger.log('获取本地媒体流...');
      try {
        this.localStream = await navigator.mediaDevices.getUserMedia({
          video: callType === 'video',
          audio: true
        });
        logger.log('本地媒体流获取成功');
      } catch (error) {
        console.error('获取本地媒体流失败:', error);
        if (error.name === 'NotAllowedError') {
          throw new Error('用户拒绝了摄像头或麦克风权限');
        } else if (error.name === 'NotFoundError') {
          throw new Error('未找到摄像头或麦克风设备');
        } else {
          throw error;
        }
      }

      logger.log('创建 RTCPeerConnection...');
      const peerConfig = this.getPeerConfig();
      this.peerConnection = new RTCPeerConnection(peerConfig);

      logger.log('添加本地轨道到连接...');
      const tracks = this.localStream.getTracks();
      tracks.forEach(track => {
        this.peerConnection.addTrack(track, this.localStream);
      });

      this.setupPeerConnectionHandlers(senderId);

      this.peerConnection.ontrack = (event) => {
        logger.log('收到远程流事件');
        if (event.streams && event.streams.length > 0) {
          this.remoteStream = event.streams[0];
          if (this.onStreamReceived) {
            this.onStreamReceived(this.remoteStream);
          }
        }
      };

      logger.log('设置远程描述...');
      await this.peerConnection.setRemoteDescription(new RTCSessionDescription(offer));
      this.flushIceCandidates();

      logger.log('生成 answer...');
      const answer = await this.peerConnection.createAnswer();
      await this.peerConnection.setLocalDescription(answer);

      logger.log('发送 answer 到发送方...');
      await this.sendSignalingMessage('webrtc_answer', {
        target_user_id: senderId,
        signal: answer
      });

      logger.log('视频通话连接已建立');
      return true;
    } catch (error) {
      console.error('处理视频通话 offer 失败:', error);
      if (this.onError) {
        this.onError(error.message || '接听失败');
      }
      throw error;
    }
  }

  addIceCandidate(candidate) {
    try {
      if (this.peerConnection && candidate?.candidate) {
        if (this.peerConnection.remoteDescription) {
          const iceCandidate = new RTCIceCandidate({
            candidate: candidate.candidate,
            sdpMid: candidate.sdpMid || '',
            sdpMLineIndex: candidate.sdpMLineIndex || 0
          });
          this.peerConnection.addIceCandidate(iceCandidate);
          logger.log('ICE 候选者添加成功');
        } else {
          logger.log('远程描述未设置，缓存 ICE 候选者');
          this.iceCandidateCache.push(candidate);
        }
      }
    } catch (error) {
      console.error('添加 ICE 候选者失败:', error);
    }
  }

  flushIceCandidates() {
    if (this.peerConnection && this.peerConnection.remoteDescription) {
      logger.log('刷新缓存的 ICE 候选者，数量:', this.iceCandidateCache.length);
      while (this.iceCandidateCache.length > 0) {
        const candidate = this.iceCandidateCache.shift();
        try {
          const iceCandidate = new RTCIceCandidate({
            candidate: candidate.candidate,
            sdpMid: candidate.sdpMid || '',
            sdpMLineIndex: candidate.sdpMLineIndex || 0
          });
          this.peerConnection.addIceCandidate(iceCandidate);
          logger.log('缓存的 ICE 候选者添加成功');
        } catch (error) {
          console.error('添加缓存的 ICE 候选者失败:', error);
        }
      }
    }
  }

  async acceptCall() {
    logger.log('接听通话');
    // 实际的接听逻辑在 handleOffer 中已完成
    // 这里发送接听确认
    await this.sendSignalingMessage('webrtc_call_accepted', {
      target_user_id: this.senderId
    }).catch(() => {});
  }

  rejectCall() {
    logger.log('拒绝通话');
    this.sendSignalingMessage('webrtc_call_rejected', {
      target_user_id: this.senderId
    }).catch(() => {});
    this.cleanup();
  }

  hangup() {
    this.sendSignalingMessage('webrtc_hangup', {
      target_user_id: this.senderId
    }).catch(() => {});
    this.cleanup();
  }

  cleanup() {
    if (this.localStream) {
      this.localStream.getTracks().forEach(track => track.stop());
      this.localStream = null;
    }

    if (this.peerConnection) {
      this.peerConnection.close();
      this.peerConnection = null;
    }

    this.remoteStream = null;
    this.senderId = null;
    this.iceCandidateCache = [];
    logger.log('通话已清理');
  }

  getPeerConfig() {
    const portRangeBegin = 60443;
    const portRangeEnd = 60443;
    
    if (this.enableDirectConnect) {
      return {
        iceCandidatePortRange: { min: portRangeBegin, max: portRangeEnd },
        iceTransportPolicy: 'all',
        iceCandidatePoolSize: 10
      };
    } else {
      return {
        iceServers: [
          { urls: 'stun:stun.l.google.com:19302' },
          { urls: 'stun:stun1.l.google.com:19302' },
          { urls: 'stun:stun2.l.google.com:19302' }
        ],
        iceTransportPolicy: 'all',
        iceCandidatePoolSize: 10,
        iceCandidatePortRange: { min: portRangeBegin, max: portRangeEnd }
      };
    }
  }

  setupPeerConnectionHandlers(senderId) {
    this.peerConnection.onicecandidate = async (event) => {
      if (event.candidate) {
        await this.sendSignalingMessage('webrtc_ice_candidate', {
          target_user_id: senderId,
          candidate: event.candidate
        });
      }
    };

    this.peerConnection.oniceconnectionstatechange = () => {
      const state = this.peerConnection.iceConnectionState;
      logger.log('ICE 连接状态变化:', state);
      
      if (this.enableDirectConnect && (state === 'failed' || state === 'disconnected')) {
        logger.log('直连失败，尝试使用 ICE 服务器...');
        this.enableDirectConnect = false;
      }
    };

    this.peerConnection.onconnectionstatechange = () => {
      const state = this.peerConnection.connectionState;
      logger.log('连接状态变化:', state);
      
      if (state === 'disconnected' || state === 'failed') {
        logger.log('WebRTC 连接已断开');
        this.cleanup();
        if (this.onHangup) {
          this.onHangup();
        }
      }
    };
  }

  async sendSignalingMessage(type, data) {
    try {
      if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
        window.ws.send(JSON.stringify({ type, data }));
        logger.log(`${type} 发送成功`);
      } else if (window.electron && window.electron.websocket) {
        window.electron.websocket.send({ type, data });
        logger.log(`${type} 发送成功（通过 IPC）`);
      } else {
        throw new Error('WebSocket 连接不可用');
      }
    } catch (error) {
      console.error(`发送 ${type} 失败:`, error);
      throw error;
    }
  }

  setCallbacks(callbacks) {
    if (callbacks.onStreamReceived) this.onStreamReceived = callbacks.onStreamReceived;
    if (callbacks.onCallReceived) this.onCallReceived = callbacks.onCallReceived;
    if (callbacks.onHangup) this.onHangup = callbacks.onHangup;
    if (callbacks.onError) this.onError = callbacks.onError;
  }

  getLocalStream() {
    return this.localStream;
  }

  getRemoteStream() {
    return this.remoteStream;
  }
}

// 导出单例实例
const screenShareSender = new ScreenShareSender();
const screenShareReceiver = new ScreenShareReceiver();
const videoCallSender = new VideoCallSender();
const videoCallReceiver = new VideoCallReceiver();

export { screenShareSender, screenShareReceiver, videoCallSender, videoCallReceiver };
export default {
  screenShareSender,
  screenShareReceiver,
  videoCallSender,
  videoCallReceiver
};
