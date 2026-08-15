package ws

import (
	"encoding/json"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/utils"
	"github.com/golang-jwt/jwt/v5"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func (c *Client) sendJSON(m WSMessage) {
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	safeSend(c, data)
}

func (c *Client) handleAuth(data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		c.sendJSON(WSMessage{Type: "auth_error", Data: map[string]string{"message": "认证数据格式错误"}})
		return
	}

	tokenStr, _ := dataMap["token"].(string)
	if tokenStr == "" {
		c.sendJSON(WSMessage{Type: "auth_error", Data: map[string]string{"message": "缺少认证令牌"}})
		return
	}

	claims := &wsClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(c.hub.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		logger.WithModule("WS").Warn("WebSocket认证失败", "error", err)
		c.sendJSON(WSMessage{Type: "auth_error", Data: map[string]string{"message": "认证令牌无效"}})
		return
	}

	// 只接受 access token
	if claims.TokenType != "" && claims.TokenType != "access" {
		c.sendJSON(WSMessage{Type: "auth_error", Data: map[string]string{"message": "请使用访问令牌"}})
		return
	}

	c.userID = claims.UserID
	c.username = claims.Username
	c.authed = true
	// 认证成功后恢复为正常的 60s 读超时（未认证阶段用的是 10s 认证超时窗口）
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.hub.register <- c

	c.sendJSON(WSMessage{Type: "auth_success", Data: map[string]interface{}{"user_id": c.userID}})

	// 更新用户在线状态，只写连接状态，避免全量 Save 触碰账号管理等无关字段。
	c.hub.db.Model(&model.User{}).Where("id = ?", c.userID).Update("status", StatusOnline)

	logger.WithModule("WS").Info("WebSocket认证成功", "userID", c.userID)
}

func (c *Client) readPump() {

	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// 设置读取消息大小限制（1MB）和读取超时
	c.conn.SetReadLimit(1 * 1024 * 1024)
	// 未认证连接用较短的认证超时窗口：连上后须在 authTimeout 内发出合法 auth 消息，
	// 否则 ReadJSON 超时触发断开，避免未认证连接长期占用连接/被反复连而刷日志。
	// 认证成功后（handleAuth 里）会恢复为正常的 60s 读超时。
	timeout := time.Duration(60 * time.Second)
	if !c.authed {
		timeout = time.Duration(10 * time.Second)
	}
	c.conn.SetReadDeadline(time.Now().Add(timeout))
	// 收到协议级 ping 控制帧时续期读超时。但未认证连接不能通过无脑发 ping 来绕过
	// 上面的 10s 认证超时窗口无限续命，故仅在已认证后才会把截止时间重置回 60s；
	// 未认证时保持 10s 的硬上限，到点即断。
	c.conn.SetPongHandler(func(string) error {
		if c.authed {
			c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		}
		return nil
	})

	for {
		var msg WSMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.WithModule("WS").Error("读取错误", "error", err)
			}
			break
		}

		// 未认证时只处理 auth 消息
		if !c.authed {
			if msg.Type != "auth" {
				logger.WithModule("WS").Warn("未认证连接收到非auth消息", "type", msg.Type)
				c.sendJSON(WSMessage{Type: "auth_error", Data: map[string]string{"message": "请先发送认证消息"}})
				continue
			}
			c.handleAuth(msg.Data)
			continue
		}

		logger.WithModule("WS").Debug("收到客户端消息", "type", msg.Type)

		switch msg.Type {
		case "heartbeat":
			// 心跳，无需处理
		case "acknowledge_sync":
			// 客户端确认已收到 sync_hint 并完成增量拉取，清除溢出标记。
			// 同时归零 syncHintStartedAt：若不复位，本次溢出已开始的重试窗口会残留，
			// 下次溢出时会按旧起点立刻判超时，静默丢失补偿直接失效。
			c.needsSync.Store(false)
			c.syncHintStartedAt.Store(0)
		case "subscribe_user_status":
			handleSubscribeUserStatus(c, msg.Data)
		case "unsubscribe_user_status":
			handleUnsubscribeUserStatus(c, msg.Data)
		case "send_message":
			handleSendMessage(c, msg.Data)
		case "read_message":
			handleReadMessage(c, msg.Data)
		case "webrtc.offer":
			handleWebRTCSignal(c, msg.Data, "webrtc.offer")
		case "webrtc.answer":
			handleWebRTCSignal(c, msg.Data, "webrtc.answer")
		case "webrtc.ice-candidate":
			handleWebRTCSignal(c, msg.Data, "webrtc.ice-candidate")
		case "call.start":
			handleCallInvite(c, msg.Data)
		case "call.answer":
			handleCallAccept(c, msg.Data)
		case "call.reject":
			handleCallReject(c, msg.Data)
		case "call.end":
			handleCallEnd(c, msg.Data)
		case "screen-share.start":
			handleScreenShareStart(c, msg.Data)
		case "screen-share.stop":
			handleScreenShareStop(c, msg.Data)
		case "screen-share.data":
			handleScreenShareData(c, msg.Data)
		case "screen-share.request":
			handleScreenShareRequest(c, msg.Data)
		case "screen-share.response":
			handleScreenShareResponse(c, msg.Data)
		// 实时通信事件
		case "realtime:session:create":
			HandleRealtimeSessionCreate(c, msg.Data)
		case "realtime:session:end":
			HandleRealtimeSessionEnd(c, msg.Data)
		case "realtime:join:request":
			HandleRealtimeJoinRequest(c, msg.Data)
		case "realtime:join:approve":
			HandleRealtimeJoinApprove(c, msg.Data)
		case "realtime:join:reject":
			HandleRealtimeJoinReject(c, msg.Data)
		case "realtime:leave":
			HandleRealtimeLeave(c, msg.Data)
		case "realtime:webrtc:offer":
			HandleRealtimeWebRTCOffer(c, msg.Data)
		case "realtime:webrtc:answer":
			HandleRealtimeWebRTCAnswer(c, msg.Data)
		case "realtime:webrtc:ice":
			HandleRealtimeWebRTCIce(c, msg.Data)
		default:
			logger.WithModule("WS").Warn("未知消息类型", "type", msg.Type)
		}
	}
}

func (c *Client) writePump() {
	pingTicker := time.NewTicker(30 * time.Second)
	defer func() {
		pingTicker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-pingTicker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWs(hub *Hub, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.WithModule("WS").Error("WebSocket升级失败", "error", err)
		return
	}

	// 尝试从 context 获取 user_id（兼容旧的 header 认证方式）
	userID, exists := c.Get("user_id")
	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 1024),
		version:  c.Query("version"),  // 客户端版本号（用于版本分布统计）
		platform: c.Query("platform"), // 客户端平台
	}
	if exists {
		client.userID = userID.(uint)
		client.authed = true
		if uname, ok := c.Get("username"); ok {
			client.username, _ = uname.(string)
		}
		client.hub.register <- client
	}

	utils.SafeGoWithLabel("ws-write", func() { client.writePump() })
	utils.SafeGoWithLabel("ws-read", func() { client.readPump() })
}

// ServeScreenShare 处理屏幕共享的 WebSocket 连接

func ServeScreenShare(hub *Hub, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.WithModule("WS").Error("屏幕共享WebSocket升级失败", "error", err)
		return
	}

	// 尝试从 context 获取 user_id（兼容旧的 header 认证方式）
	userID, exists := c.Get("user_id")
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 1024)}
	if exists {
		client.userID = userID.(uint)
		client.authed = true
		client.hub.register <- client
	}

	utils.SafeGoWithLabel("ws-write", func() { client.writePump() })
	utils.SafeGoWithLabel("ws-read", func() { client.readPump() })
}

// 处理屏幕共享开始
