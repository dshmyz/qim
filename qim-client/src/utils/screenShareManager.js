// 屏幕共享管理器

// 屏幕共享管理器

class ScreenShareManager {
  static createPeerConnection(enableDirectConnect = true) {
    const portRangeBegin = 60443
    const portRangeEnd = 60443

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
    ]

    let peerConfig = {
      iceServers,
      iceTransportPolicy: enableDirectConnect ? 'all' : 'all',
      iceCandidatePoolSize: 10
    }

    if (!enableDirectConnect) {
      console.log('使用 ICE 服务器模式...')
      console.log('ICE 服务器配置:', iceServers)
    } else {
      console.log('尝试直连模式，配置 ICE 服务器作为后备...')
      console.log('ICE 服务器配置:', iceServers)
    }

    return new RTCPeerConnection(peerConfig)
  }

  static sendWebSocketMessage(type, data) {
    // 包装数据到 data 字段，与 webrtc.js 保持一致
    const message = { type, data };
    // 使用 WebSocket 管理器发送消息
    if (window.sendWebSocketMessage) {
      window.sendWebSocketMessage(message);
    } else if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
      // 回退到直接使用 window.ws
      window.ws.send(JSON.stringify(message))
    } else if (window.electron && window.electron.websocket) {
      // 回退到 IPC 方式
      window.electron.websocket.send(message)
    } else {
      throw new Error('WebSocket 连接不可用')
    }
  }
}

class ScreenShareInitiator {
  constructor() {
    this.state = 'idle' // idle, selecting, sharing, connecting, error
    this.peerConnection = null
    this.screenStream = null
    this.receiverId = null
    this.selectedSource = null
    this.sourceId = ''
    this.enableDirectConnect = true // 启用直连尝试
  }

  async start(receiverId) {
    try {
      this.receiverId = receiverId
      this.setState('selecting')
      
      // 获取屏幕源
      const sources = await this.getScreenSources()
      if (sources.length === 0) {
        throw new Error('没有可用的屏幕源')
      }
      
      // 默认选择第一个屏幕源
      this.selectedSource = sources[0]
      this.sourceId = sources[0].id
      
      // 开始共享（先获取屏幕流，准备好等待对方加入）
      await this.startSharing()
      
      // 发送屏幕共享请求
      this.sendShareRequest()
      
      return true
    } catch (error) {
      this.setState('error', error.message)
      throw error
    }
  }

  async getScreenSources() {
    const electron = window.electron
    if (electron?.ipcRenderer) {
      try {
        electron.ipcRenderer.send('start-screen-share')
        
        const sources = await new Promise((resolve) => {
          electron.ipcRenderer.once('screen-sources', (event, sources) => {
            resolve(sources)
          })
        })
        
        if (Array.isArray(sources) && sources.length > 0) {
          return sources.map(source => ({
            id: source.id,
            name: source.name,
            thumbnail: source.thumbnail
          }))
        }
      } catch (error) {
        console.error('获取屏幕源失败:', error)
      }
    }
    return []
  }

  async startSharing() {
    try {
      const constraints = {
        audio: false,
        video: {
          mandatory: {
            chromeMediaSource: 'desktop',
            chromeMediaSourceId: this.sourceId
          },
          // 添加额外的视频约束，避免simulcast问题
          optional: [
            { maxWidth: 1920 },
            { maxHeight: 1080 },
            { maxFrameRate: 30 },
            { googLeakyBucket: true },
            { googTemporalLayeredScreencast: true }
          ]
        }
      }

      this.screenStream = await navigator.mediaDevices.getUserMedia(constraints)
      this.setState('sharing')
      console.log('屏幕共享流已准备就绪，等待对方加入...')
    } catch (error) {
      console.error('获取屏幕流失败:', error)
      throw error
    }
  }

  sendShareRequest() {
    if (this.receiverId) {
      ScreenShareManager.sendWebSocketMessage('screen-share-request', {
        target_user_id: this.receiverId,
        conversation_id: this.receiverId
      })
      console.log('已发送屏幕共享请求，等待对方接受...')
    }
  }

  async establishConnection() {
    try {
      this.setState('connecting')

      this.peerConnection = ScreenShareManager.createPeerConnection(this.enableDirectConnect)
      this.setupEventHandlers()
      this.addStreamToConnection()
      await this.createAndSendOffer()

      console.log('WebRTC 连接正在建立...')
    } catch (error) {
      this.setState('error', error.message)
      throw error
    }
  }

  addStreamToConnection() {
    console.log("addStreamToConnection--------:", this.screenStream)
    if (this.peerConnection && this.screenStream) {
      const tracks = this.screenStream.getTracks()
      tracks.forEach(track => {
        this.peerConnection.addTrack(track, this.screenStream)
      })
    }
  }

  async createAndSendOffer() {
    if (this.peerConnection && this.receiverId) {
      // 添加视频约束，避免simulcast问题
      // const offerOptions = {
      //   offerToReceiveVideo: true,
      //   offerToReceiveAudio: false,
      //   // 禁用simulcast以避免编码器错误
      //   iceRestart: false
      // }
      
      const offer = await this.peerConnection.createOffer()
      await this.peerConnection.setLocalDescription(offer)
      
      ScreenShareManager.sendWebSocketMessage('webrtc_offer', {
        target_user_id: this.receiverId,
        signal: offer
      })
    }
  }

  setupEventHandlers() {
    if (this.peerConnection) {
      // 处理 ICE 候选者
      this.peerConnection.onicecandidate = (event) => {
        console.log("处理 ICE 候选者", event)
        if (event.candidate && this.receiverId) {
          ScreenShareManager.sendWebSocketMessage('webrtc_ice_candidate', {
            target_user_id: this.receiverId,
            candidate: event.candidate
          })
        }
      }

      // 处理 ICE 连接状态变化
      this.peerConnection.oniceconnectionstatechange = () => {
        console.log('ICE 连接状态变化:', this.peerConnection.iceConnectionState)
        
        // 如果直连失败，尝试使用 ICE 服务器
        if (this.enableDirectConnect && 
            (this.peerConnection.iceConnectionState === 'failed' || 
             this.peerConnection.iceConnectionState === 'disconnected')) {
          console.log('直连失败，尝试使用 ICE 服务器...')
          this.enableDirectConnect = false
          
          // 重新创建 RTCPeerConnection，使用 ICE 服务器
          this.stop()
          this.establishConnection()
        }
      }

      // 处理连接状态变化
      this.peerConnection.onconnectionstatechange = () => {
        console.log('连接状态变化:', this.peerConnection.connectionState)
        
        if (this.peerConnection.connectionState === 'connected') {
          this.setState('sharing')
          console.log('WebRTC 连接已建立')
        } else if (this.peerConnection.connectionState === 'failed') {
          this.setState('error', '连接失败')
          console.error('WebRTC 连接失败')
        }
      }
    }
  }

  handleAnswer(answer) {
    if (this.peerConnection) {
      this.peerConnection.setRemoteDescription(new RTCSessionDescription(answer))
        .then(() => {
          console.log('远程描述设置成功')
        })
        .catch(error => {
          console.error('设置远程描述失败:', error)
        })
    }
  }

  handleIceCandidate(candidate) {
    // 验证 candidate 数据是否有效
    if (!candidate || (candidate.sdpMid === null && candidate.sdpMLineIndex === null)) {
      console.warn('无效的 ICE 候选者，跳过:', candidate)
      return
    }

    if (this.peerConnection) {
      try {
        this.peerConnection.addIceCandidate(new RTCIceCandidate(candidate))
          .catch(error => {
            console.error('添加 ICE 候选者失败:', error)
          })
      } catch (error) {
        console.error('创建 RTCIceCandidate 对象失败:', error)
      }
    }
  }

  setState(state, error = null) {
    this.state = state
    console.log(`屏幕共享状态更新: ${state}`, error ? `错误: ${error}` : '')
  }

  stop() {
    if (this.screenStream) {
      this.screenStream.getTracks().forEach(track => track.stop())
      this.screenStream = null
    }

    if (this.peerConnection) {
      this.peerConnection.close()
      this.peerConnection = null
    }

    this.setState('idle')
    console.log('屏幕共享已停止')
  }

  getState() {
    return this.state
  }

  getScreenStream() {
    return this.screenStream
  }
}

class ScreenShareReceiver {
  constructor() {
    this.state = 'idle' // idle, receiving, connecting, viewing, error
    this.peerConnection = null
    this.remoteStream = null
    this.senderId = null
    this.videoElement = null
    this.onStreamReceived = null
    this.iceCandidateCache = []
    this.enableDirectConnect = true // 启用直连尝试
    this.lastOffer = null // 保存最后收到的 offer，用于重试
  }

  init(videoElement, onStreamReceived) {
    this.videoElement = videoElement
    this.onStreamReceived = onStreamReceived
    console.log('ScreenShareReceiver 初始化成功')
  }

  async handleShareRequest(data) {
    this.senderId = data.requester_id || data.userId || data.user_id
    this.setState('receiving')
    console.log('收到屏幕共享请求，来自用户:', this.senderId)
  }

  async acceptShareRequest(conversationId) {
    try {
      // 发送接受响应
      ScreenShareManager.sendWebSocketMessage('screen-share-response', {
        conversation_id: conversationId,
        requester_id: this.senderId,
        status: 'accepted'
      })
      
      this.setState('connecting')
      console.log('已接受屏幕共享请求，准备建立连接...')
    } catch (error) {
      this.setState('error', error.message)
      throw error
    }
  }

  rejectShareRequest(conversationId) {
    ScreenShareManager.sendWebSocketMessage('screen-share-response', {
      conversation_id: conversationId,
      requester_id: this.senderId,
      status: 'rejected'
    })
    this.setState('idle')
    console.log('已拒绝屏幕共享请求')
  }

  async handleOffer(offer, senderId) {
    try {
      this.senderId = senderId
      this.lastOffer = offer // 保存 offer，用于直连失败时重试
      this.setState('connecting')
      
      // 创建 WebRTC 连接
      this.peerConnection = ScreenShareManager.createPeerConnection(this.enableDirectConnect)
      console.log("// 创建 WebRTC 连接", this.peerConnection)
      // 设置事件处理
      this.setupEventHandlers()

      // 设置远程描述
      await this.peerConnection.setRemoteDescription(new RTCSessionDescription(offer))
       // 处理缓存的 ICE 候选者
      this.processIceCandidateCache()

      // 生成并发送 answer
      const answer = await this.peerConnection.createAnswer()
      await this.peerConnection.setLocalDescription(answer)
     

      ScreenShareManager.sendWebSocketMessage('webrtc_answer', {
        target_user_id: senderId,
        signal: answer
      })
      
      console.log('正在建立 WebRTC 连接...')
    } catch (error) {
      this.setState('error', error.message)
      throw error
    }
  }

  setupEventHandlers() {
    console.log('peerConnection:', this.peerConnection);

    if (this.peerConnection) {
      // 处理远程流
      this.peerConnection.ontrack = (event) => {
        console.log('收到远程流事件:', event);
        
        try {
          // 优先从 event.streams 获取流
          let stream = null;
          if (event.streams && event.streams.length > 0) {
            stream = event.streams[0];
            console.log('从 event.streams 获取远程流:', stream);
          } 
          // 或者从 event.track 创建新流
          else if (event.track) {
            stream = new MediaStream();
            stream.addTrack(event.track);
            console.log('从 event.track 创建远程流:', stream);
          }
          
          if (stream) {
            this.remoteStream = stream;
            console.log('收到远程流:', this.remoteStream);
            console.log('视频元素:', this.videoElement);
            
            if (this.videoElement) {
              // 检查视频元素是否已经有相同的流
              if (this.videoElement.srcObject !== this.remoteStream) {
                this.videoElement.srcObject = this.remoteStream;
                // 监听canplay事件后再播放，确保视频元素就绪
                const playWhenReady = () => {
                  this.videoElement.play().catch(err => {
                    console.error('播放视频失败:', err);
                  });
                  this.videoElement.removeEventListener('canplay', playWhenReady);
                };
                this.videoElement.addEventListener('canplay', playWhenReady);
              }
            }
            
            if (this.onStreamReceived) {
              this.onStreamReceived(this.remoteStream);
            }
            
            this.setState('viewing');
            console.log('开始观看屏幕共享');
          } else {
            console.warn('收到ontrack事件但未找到媒体流:', event);
          }
        } catch (error) {
          console.error('处理远程流事件失败:', error);
          this.setState('error', '处理远程流失败');
        }
      }

      // 处理 ICE 候选者
      this.peerConnection.onicecandidate = (event) => {
        if (event.candidate && this.senderId) {
          ScreenShareManager.sendWebSocketMessage('webrtc_ice_candidate', {
            target_user_id: this.senderId,
            candidate: event.candidate
          })
        }
      }

      // 处理 ICE 连接状态变化
      this.peerConnection.oniceconnectionstatechange = () => {
        console.log('ICE 连接状态变化:', this.peerConnection.iceConnectionState)
        
        // 如果直连失败，尝试使用 ICE 服务器
        if (this.enableDirectConnect && 
            (this.peerConnection.iceConnectionState === 'failed' || 
             this.peerConnection.iceConnectionState === 'disconnected')) {
          console.log('直连失败，尝试使用 ICE 服务器...')
          this.enableDirectConnect = false
          
          // 重新创建 RTCPeerConnection，使用 ICE 服务器
          this.stop()
          // 重新处理 offer，使用 ICE 服务器
          if (this.lastOffer && this.senderId) {
            this.handleOffer(this.lastOffer, this.senderId)
          }
        }
      }

      // 处理连接状态变化
      this.peerConnection.onconnectionstatechange = () => {
        console.log('连接状态变化:', this.peerConnection.connectionState)
        
        if (this.peerConnection.connectionState === 'connected') {
          console.log('WebRTC 连接已建立')
        } else if (this.peerConnection.connectionState === 'failed') {
          this.setState('error', '连接失败')
          console.error('WebRTC 连接失败')
        }
      }
    }
  }

  handleIceCandidate(candidate) {
    // 验证 candidate 数据是否有效
    // if (!candidate || (candidate.sdpMid === null && candidate.sdpMLineIndex === null)) {
    if (!candidate) {
      console.warn('无效的 ICE 候选者，跳过:', candidate)
      return
    }

    if (this.peerConnection) {
      try {
         const iceCandidate = new RTCIceCandidate({
              candidate: candidate.candidate,
              sdpMid: candidate.sdpMid || '',
              sdpMLineIndex: candidate.sdpMLineIndex || 0
            });
        this.peerConnection.addIceCandidate(iceCandidate)
          .catch(error => {
            console.error('添加 ICE 候选者失败:', error)
          })
                      console.log('ICE 候选者添加成功:', iceCandidate);

      } catch (error) {
        console.error('创建 RTCIceCandidate 对象失败:', error)
      }
    } else {
      // 缓存 ICE 候选者，等待连接建立
      this.iceCandidateCache.push(candidate)
      console.log('缓存 ICE 候选者，等待连接建立')
    }
  }

  processIceCandidateCache() {
    while (this.iceCandidateCache.length > 0) {
      const candidate = this.iceCandidateCache.shift()
      this.handleIceCandidate(candidate)
    }
  }

  setState(state, error = null) {
    this.state = state
    console.log(`屏幕共享接收状态更新: ${state}`, error ? `错误: ${error}` : '')
  }

  stop() {
    if (this.peerConnection) {
      this.peerConnection.close()
      this.peerConnection = null
    }

    if (this.videoElement && this.videoElement.srcObject) {
      const stream = this.videoElement.srcObject
      if (stream.getTracks) {
        stream.getTracks().forEach(track => track.stop())
      }
      this.videoElement.srcObject = null
    }

    this.remoteStream = null
    this.setState('idle')
    console.log('屏幕共享接收已停止')
  }

  getState() {
    return this.state
  }

  getRemoteStream() {
    return this.remoteStream
  }
}

// 导出单例实例
const screenShareInitiator = new ScreenShareInitiator()
const screenShareReceiver = new ScreenShareReceiver()

export { ScreenShareManager, screenShareInitiator, screenShareReceiver }
