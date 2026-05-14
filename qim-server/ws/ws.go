package ws

import (
	"crypto/rand"
	"encoding/json"
	"log"
	"net/http"
	"qim-server/model"
	"qim-server/pkg/mention"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

const (
	StatusOnline  = "online"
	StatusOffline = "offline"
	StatusBusy    = "busy"

	// 状态变更防抖延迟
	StatusDebounceDelay = 500 * time.Millisecond
)

var GlobalHub *Hub

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type cachedMembers struct {
	memberIDs []uint
	expiredAt time.Time
}

type Hub struct {
	clients             sync.Map
	register            chan *Client
	unregister          chan *Client
	broadcast           chan []byte
	Broadcast           chan []byte
	userClients         sync.Map
	conversationMembers map[uint]cachedMembers
	mu                  sync.RWMutex
	nodes               []string
	nodeID              string
	db                  *gorm.DB
	dbType              string

	statusDebouncer *StatusDebouncer
	userSubscribers sync.Map
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID uint
}

type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id,omitempty"`
}

func NewHub(db *gorm.DB, dbType string) *Hub {
	// 生成节点 ID
	nodeID := generateNodeID()

	// 初始化节点列表（这里可以从配置文件或环境变量中读取）
	nodes := []string{}

	// 初始化广播通道
	broadcastChan := make(chan []byte)

	log.Printf("节点 %s 初始化完成，将使用基于 HTTP 的多节点模式", nodeID)

	return &Hub{
		clients:             sync.Map{},
		register:            make(chan *Client),
		unregister:          make(chan *Client),
		broadcast:           broadcastChan,
		Broadcast:           broadcastChan,
		userClients:         sync.Map{},
		conversationMembers: make(map[uint]cachedMembers),
		nodes:               nodes,
		nodeID:              nodeID,
		db:                  db,
		dbType:              dbType,
		statusDebouncer:     NewStatusDebouncer(StatusDebounceDelay),
	}
}

// generateNodeID 生成唯一的节点 ID
func generateNodeID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString 生成指定长度的随机字符串
func randomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// 降级方案（极少发生）
		for i := range b {
			b[i] = letterBytes[time.Now().UnixNano()%int64(len(letterBytes))]
		}
		return string(b)
	}
	for i := range b {
		b[i] = letterBytes[int(b[i])%len(letterBytes)]
	}
	return string(b)
}

func (h *Hub) Run() {
	// 启动节点间通信服务
	go h.startNodeCommunication()

	for {
		select {
		case client := <-h.register:
			h.clients.Store(client, true)
			if existingClients, ok := h.userClients.Load(client.userID); ok {
				clients := existingClients.([]*Client)
				clients = append(clients, client)
				h.userClients.Store(client.userID, clients)
			} else {
				h.userClients.Store(client.userID, []*Client{client})
			}
			log.Printf("用户 %d 连接", client.userID)

			// 更新用户在线状态并广播
			h.UpdateUserStatus(client.userID, StatusOnline)

		case client := <-h.unregister:
			if _, ok := h.clients.Load(client); ok {
				h.clients.Delete(client)
				close(client.send)

				if existingClients, ok := h.userClients.Load(client.userID); ok {
					clients := existingClients.([]*Client)
					for i, c := range clients {
						if c == client {
							clients = append(clients[:i], clients[i+1:]...)
							break
						}
					}

					if len(clients) == 0 {
						h.userClients.Delete(client.userID)
						// 更新用户离线状态并广播
						h.UpdateUserStatus(client.userID, StatusOffline)
					} else {
						h.userClients.Store(client.userID, clients)
					}
				}
			}

			// 清理用户的订阅
			h.CleanupUserSubscriptions(client.userID)
			log.Printf("用户 %d 断开连接", client.userID)

		case message := <-h.broadcast:
			// 异步广播，不阻塞事件循环
			go h.asyncBroadcast(message)
		}
	}
}

// asyncBroadcast 异步广播消息给所有客户端，使用并发发送不阻塞事件循环
func (h *Hub) asyncBroadcast(message []byte) {
	// 收集所有客户端到切片
	var clients []*Client
	h.clients.Range(func(key, value interface{}) bool {
		clients = append(clients, key.(*Client))
		return true
	})

	if len(clients) == 0 {
		h.broadcastToOtherNodes(message)
		return
	}

	// 使用 goroutine 池并行发送
	// 每个客户端一个 goroutine，由运行时调度器管理
	var wg sync.WaitGroup
	failedChan := make(chan *Client, len(clients))

	for _, client := range clients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			select {
			case c.send <- message:
				// 发送成功
			default:
				// 发送通道已满，标记为待删除
				failedChan <- c
			}
		}(client)
	}

	// 等待所有发送完成
	wg.Wait()
	close(failedChan)

	// 清理发送失败的客户端
	for client := range failedChan {
		h.clients.Delete(client)
		close(client.send)
	}

	// 广播到其他节点
	h.broadcastToOtherNodes(message)
}

// startNodeCommunication 启动节点间通信服务
func (h *Hub) startNodeCommunication() {
	// 这里可以实现节点发现和心跳检测
	log.Println("节点间通信服务启动")
}

// broadcastToOtherNodes 通过 HTTP 向其他节点广播消息
func (h *Hub) broadcastToOtherNodes(message []byte) {
	for _, node := range h.nodes {
		if node == h.nodeID {
			continue // 跳过自身节点
		}

		// 构建其他节点的 URL
		nodeURL := "http://" + node + "/api/v1/node/broadcast"

		// 发送 HTTP 请求
		go func(url string) {
			resp, err := http.Post(url, "application/json", nil)
			if err != nil {
				log.Printf("向节点 %s 广播失败: %v", url, err)
				return
			}
			defer resp.Body.Close()
		}(nodeURL)
	}
}

func (h *Hub) SendToUser(userID uint, message []byte) {
	if existingClients, ok := h.userClients.Load(userID); ok {
		clients := existingClients.([]*Client)
		for _, client := range clients {
			select {
			case client.send <- message:
			default:
			}
		}
	}

	h.sendToUserToOtherNodes(userID, message)
}

// IsUserOnline 检查用户是否在线
func (h *Hub) IsUserOnline(userID uint) bool {
	if existingClients, ok := h.userClients.Load(userID); ok {
		clients := existingClients.([]*Client)
		return len(clients) > 0
	}
	return false
}

// UpdateConversationMembers 更新会话成员缓存
func (h *Hub) UpdateConversationMembers(convID uint) {
	// 从数据库查询最新的会话成员
	db := h.db
	var members []model.ConversationMember
	result := db.Where("conversation_id = ?", convID).Find(&members)
	if result.Error != nil {
		log.Printf("更新会话成员缓存失败: %v", result.Error)
		return
	}

	// 提取用户ID
	memberIDs := make([]uint, len(members))
	for i, member := range members {
		memberIDs[i] = member.UserID
	}

	// 更新缓存，5分钟过期
	h.mu.Lock()
	h.conversationMembers[convID] = cachedMembers{
		memberIDs: memberIDs,
		expiredAt: time.Now().Add(5 * time.Minute),
	}
	h.mu.Unlock()
	log.Printf("更新会话 %d 成员缓存，成员数量: %d", convID, len(memberIDs))
}

// sendToUserToOtherNodes 通过 HTTP 向其他节点发送用户特定消息
func (h *Hub) sendToUserToOtherNodes(userID uint, message []byte) {
	for _, node := range h.nodes {
		if node == h.nodeID {
			continue // 跳过自身节点
		}

		// 构建其他节点的 URL
		nodeURL := "http://" + node + "/api/v1/node/send-to-user"

		// 构建请求体
		reqBody := map[string]interface{}{
			"user_id": userID,
			"message": string(message),
		}
		jsonBody, _ := json.Marshal(reqBody)

		// 发送 HTTP 请求
		go func(url string, body []byte) {
			resp, err := http.Post(url, "application/json", nil)
			if err != nil {
				log.Printf("向节点 %s 发送用户消息失败: %v", url, err)
				return
			}
			defer resp.Body.Close()
		}(nodeURL, jsonBody)
	}
}

func (h *Hub) SendToConversation(convID uint, excludeUserID uint, message []byte) {
	h.mu.RLock()
	cached, found := h.conversationMembers[convID]
	h.mu.RUnlock()

	var memberIDs []uint
	if found && time.Now().Before(cached.expiredAt) {
		memberIDs = cached.memberIDs
	} else {
		db := h.db
		var members []model.ConversationMember
		result := db.Where("conversation_id = ?", convID).Find(&members)
		if result.Error != nil {
			log.Printf("查询会话成员失败: %v", result.Error)
			return
		}

		memberIDs = make([]uint, len(members))
		for i, member := range members {
			memberIDs[i] = member.UserID
		}

		h.mu.Lock()
		h.conversationMembers[convID] = cachedMembers{
			memberIDs: memberIDs,
			expiredAt: time.Now().Add(5 * time.Minute),
		}
		h.mu.Unlock()
	}

	for _, userID := range memberIDs {
		if userID != excludeUserID {
			h.SendToUser(userID, message)
		}
	}
}

func (h *Hub) SendToConversationAsync(convID uint, excludeUserID uint, message []byte) {
	h.mu.RLock()
	cached, found := h.conversationMembers[convID]
	h.mu.RUnlock()

	var memberIDs []uint
	if found && time.Now().Before(cached.expiredAt) {
		memberIDs = cached.memberIDs
	} else {
		db := h.db
		var members []model.ConversationMember
		result := db.Where("conversation_id = ?", convID).Find(&members)
		if result.Error != nil {
			log.Printf("查询会话成员失败: %v", result.Error)
			return
		}

		memberIDs = make([]uint, len(members))
		for i, member := range members {
			memberIDs[i] = member.UserID
		}

		h.mu.Lock()
		h.conversationMembers[convID] = cachedMembers{
			memberIDs: memberIDs,
			expiredAt: time.Now().Add(5 * time.Minute),
		}
		h.mu.Unlock()
	}

	var wg sync.WaitGroup
	for _, userID := range memberIDs {
		if userID != excludeUserID {
			wg.Add(1)
			go func(uid uint) {
				defer wg.Done()
				h.SendToUser(uid, message)
			}(userID)
		}
	}
	wg.Wait()
}

func (c *Client) readPump() {

	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		var msg WSMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("读取错误: %v", err)
			}
			break
		}

		log.Printf("收到客户端消息: 类型=%s", msg.Type)

		switch msg.Type {
		case "heartbeat":
			// 心跳，无需处理
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
			log.Printf("未知消息类型: %s", msg.Type)
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func handleSendMessage(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	convIDFloat, _ := msgData["conversation_id"].(float64)
	convID := uint(convIDFloat)
	msgType, _ := msgData["type"].(string)
	content, _ := msgData["content"].(string)

	// 处理引用消息ID
	var quotedMessageID *uint
	if quotedID, ok := msgData["quoted_message_id"].(float64); ok {
		quotedIDUint := uint(quotedID)
		quotedMessageID = &quotedIDUint
	}

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		return
	}

	// 创建消息
	msg := model.Message{
		ConversationID:  convID,
		SenderID:        c.userID,
		Type:            msgType,
		Content:         content,
		QuotedMessageID: quotedMessageID,
	}
	db.Create(&msg)

	// 预加载发送者和引用消息
	db.Preload("Sender").Preload("QuotedMessage").First(&msg, msg.ID)

	// 预加载引用消息的发送者
	if msg.QuotedMessage != nil {
		db.Model(&msg.QuotedMessage).Association("Sender").Find(&msg.QuotedMessage.Sender)
	}

	// 更新会话最后消息
	now := time.Now()
	var conv model.Conversation
	db.First(&conv, convID)
	conv.LastMessageID = &msg.ID
	conv.LastMessageAt = &now
	db.Save(&conv)

	// 更新未读数
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id != ?", convID, c.userID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

	mentions := mention.ExtractMentions(msg.Content)
	mentionUserIDs := make([]uint, 0, len(mentions))
	for _, m := range mentions {
		mentionUserIDs = append(mentionUserIDs, m.UserID)
	}

	wsMsg := WSMessage{
		Type: "new_message",
		Data: map[string]interface{}{
			"id":                msg.ID,
			"conversation_id":   msg.ConversationID,
			"sender_id":         msg.SenderID,
			"type":              msg.Type,
			"content":           msg.Content,
			"quoted_message_id": msg.QuotedMessageID,
			"is_recalled":       msg.IsRecalled,
			"is_read":           msg.IsRead,
			"is_avatar_reply":   msg.IsAvatarReply,
			"is_ai_message":     msg.Sender.Type == "bot" || msg.Sender.Type == "system",
			"recalled_at":       msg.RecalledAt,
			"created_at":        msg.CreatedAt,
			"sender":            msg.Sender,
			"quoted_message":    msg.QuotedMessage,
			"mention_user_ids":  mentionUserIDs,
		},
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	// 推送给会话其他成员
	c.hub.SendToConversation(convID, c.userID, jsonMsg)
}

func handleReadMessage(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	convIDFloat, _ := msgData["conversation_id"].(float64)
	convID := uint(convIDFloat)

	// 使用单条 INSERT ... SELECT 语句批量创建已读回执，避免将消息 ID 加载到内存
	// 根据数据库类型使用不同的语法
	if c.hub.dbType == "mysql" {
		db.Exec(`
			INSERT IGNORE INTO message_read_receipts (message_id, conversation_id, user_id, created_at)
			SELECT id, ?, ?, ?
			FROM messages
			WHERE conversation_id = ? AND sender_id != ? AND is_read = false
		`, convID, c.userID, time.Now(), convID, c.userID)
	} else {
		db.Exec(`
			INSERT INTO message_read_receipts (message_id, conversation_id, user_id, created_at)
			SELECT id, ?, ?, ?
			FROM messages
			WHERE conversation_id = ? AND sender_id != ? AND is_read = false
			ON CONFLICT (message_id, user_id) DO NOTHING
		`, convID, c.userID, time.Now(), convID, c.userID)
	}

	// 更新成员未读数和最后读取
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, c.userID).
		Updates(map[string]interface{}{
			"unread_count": 0,
			"last_read_at": time.Now(),
		})

	// 标记消息为已读（只标记非自己发送的消息）
	result := db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND is_read = false", convID, c.userID).
		UpdateColumn("is_read", true)

	// 只有当确实有消息被标记为已读时，才发送已读回执通知给对方
	if result.RowsAffected > 0 {
		var conv model.Conversation
		db.First(&conv, convID)

		readMsg := WSMessage{
			Type: "message_read",
			Data: map[string]interface{}{
				"conversation_id": convID,
				"user_id":         c.userID,
				"timestamp":       time.Now().Unix(),
			},
		}
		jsonMsg, _ := json.Marshal(readMsg)

		if conv.Type == "single" {
			var otherMember model.ConversationMember
			db.Where("conversation_id = ? AND user_id != ?", convID, c.userID).First(&otherMember)
			c.hub.SendToUser(otherMember.UserID, jsonMsg)
		} else if conv.Type == "group" {
			var members []model.ConversationMember
			db.Where("conversation_id = ? AND user_id != ?", convID, c.userID).Find(&members)

			for _, member := range members {
				c.hub.SendToUser(member.UserID, jsonMsg)
			}
		}
	}
}

func handleWebRTCSignal(c *Client, data interface{}, signalType string) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}
	log.Printf("收到 %s 消息: %v", signalType, msgData)
	var targetUserID uint

	// 尝试将 target_user_id 转换为 float64 (数字类型)
	if targetUserIDFloat, ok := msgData["target_user_id"].(float64); ok {
		targetUserID = uint(targetUserIDFloat)
	} else if targetUserIDStr, ok := msgData["target_user_id"].(string); ok {
		// 尝试将 target_user_id 转换为 string 类型，然后转换为 uint
		if id, err := strconv.ParseUint(targetUserIDStr, 10, 32); err == nil {
			targetUserID = uint(id)
		} else {
			return
		}
	} else {
		return
	}

	// 构建转发的信令消息
	// ICE 候选者使用 candidate 字段，其他信令使用 signal 字段
	signalData := msgData["signal"]
	if signalType == "webrtc.ice-candidate" {
		signalData = msgData["candidate"]
	}

	// 构建转发的数据，包含原始消息中的所有字段
	forwardData := map[string]interface{}{
		"from_user_id": c.userID,
		"signal":       signalData,
	}

	// 转发原始消息中的其他字段
	// 优先使用新的 media_type 字段
	if mediaType, ok := msgData["media_type"]; ok {
		forwardData["media_type"] = mediaType
	} else {
		// 兼容旧的 share_type 和 call_type 字段
		// 如果存在 share_type 或 call_type，同时设置 media_type
		if shareType, ok := msgData["share_type"]; ok {
			forwardData["share_type"] = shareType
			forwardData["media_type"] = shareType // 同时设置 media_type
		}
		if callType, ok := msgData["call_type"]; ok {
			forwardData["call_type"] = callType
			forwardData["media_type"] = callType // 同时设置 media_type
		}
	}

	// 如果有 media_type，也转发原始的 share_type 和 call_type（向后兼容）
	if mediaType, ok := forwardData["media_type"]; ok {
		// 如果是新格式（只有 media_type），也设置 share_type 或 call_type
		if _, hasShareType := forwardData["share_type"]; !hasShareType {
			if mediaTypeStr, ok := mediaType.(string); ok {
				if mediaTypeStr == "screen" {
					forwardData["share_type"] = mediaTypeStr
				} else if mediaTypeStr == "video" || mediaTypeStr == "audio" {
					forwardData["call_type"] = mediaTypeStr
				}
			}
		}
	}

	signalMsg := WSMessage{
		Type: signalType,
		Data: forwardData,
	}

	jsonMsg, _ := json.Marshal(signalMsg)

	// 发送给目标用户
	c.hub.SendToUser(targetUserID, jsonMsg)
	log.Printf("转发WebRTC信令 %s 从用户 %d 到用户 %d", signalType, c.userID, targetUserID)
}

func ServeWs(hub *Hub, c *gin.Context) {
	userID, _ := c.Get("user_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 1024), userID: userID.(uint)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

// ServeScreenShare 处理屏幕共享的 WebSocket 连接
func ServeScreenShare(hub *Hub, c *gin.Context) {
	userID, _ := c.Get("user_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 1024), userID: userID.(uint)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

// 处理屏幕共享开始
func handleScreenShareStart(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	// 支持两种命名格式：下划线和驼峰
	var convIDFloat float64
	if val, ok := msgData["conversation_id"].(float64); ok {
		convIDFloat = val
	} else if val, ok := msgData["conversationId"].(float64); ok {
		convIDFloat = val
	} else {
		return
	}
	convID := uint(convIDFloat)

	// 支持两种命名格式：下划线和驼峰
	var userIdFloat float64
	if val, ok := msgData["user_id"].(float64); ok {
		userIdFloat = val
	} else if val, ok := msgData["userId"].(float64); ok {
		userIdFloat = val
	} else {
		// 如果没有提供userId，使用当前用户ID
		userIdFloat = float64(c.userID)
	}
	userId := uint(userIdFloat)

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		return
	}

	// 构建屏幕共享开始消息
	wsMsg := WSMessage{
		Type: "screen-share.start",
		Data: map[string]interface{}{
			"conversation_id": convID,
			"user_id":         userId,
			"timestamp":       time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	// 推送给会话其他成员
	c.hub.SendToConversation(convID, c.userID, jsonMsg)
	log.Printf("用户 %d 开始屏幕共享，会话 %d", c.userID, convID)
}

// 处理屏幕共享停止
func handleScreenShareStop(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	// 支持两种命名格式：下划线和驼峰
	var convIDFloat float64
	if val, ok := msgData["conversation_id"].(float64); ok {
		convIDFloat = val
	} else if val, ok := msgData["conversationId"].(float64); ok {
		convIDFloat = val
	} else {
		return
	}
	convID := uint(convIDFloat)

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		return
	}

	// 构建屏幕共享停止消息
	wsMsg := WSMessage{
		Type: "screen-share.stop",
		Data: map[string]interface{}{
			"conversation_id": convID,
			"user_id":         c.userID,
			"timestamp":       time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	// 推送给会话其他成员
	c.hub.SendToConversation(convID, c.userID, jsonMsg)
	log.Printf("用户 %d 停止屏幕共享，会话 %d", c.userID, convID)
}

// 处理屏幕共享数据
func handleScreenShareData(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		log.Printf("屏幕共享数据格式错误: %v", data)
		return
	}

	// 支持两种命名格式：下划线和驼峰
	var convID uint
	var found bool

	// 尝试从 conversation_id 获取
	if val, ok := msgData["conversation_id"]; ok {
		switch v := val.(type) {
		case float64:
			convID = uint(v)
			found = true
		case int:
			convID = uint(v)
			found = true
		case int64:
			convID = uint(v)
			found = true
		case string:
			if id, err := strconv.Atoi(v); err == nil {
				convID = uint(id)
				found = true
			}
		}
	}

	// 尝试从 conversationId 获取
	if !found && msgData["conversationId"] != nil {
		val := msgData["conversationId"]
		switch v := val.(type) {
		case float64:
			convID = uint(v)
			found = true
		case int:
			convID = uint(v)
			found = true
		case int64:
			convID = uint(v)
			found = true
		case string:
			if id, err := strconv.Atoi(v); err == nil {
				convID = uint(id)
				found = true
			}
		}
	}

	if !found {
		log.Printf("屏幕共享数据缺少会话ID: %v", msgData)
		return
	}

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		return
	}

	// 构建屏幕共享数据消息
	wsMsg := WSMessage{
		Type: "screen-share-data",
		Data: map[string]interface{}{
			"conversation_id": convID,
			"user_id":         c.userID,
			"data":            msgData["data"],
		},
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	// 推送给会话其他成员
	log.Printf("准备向会话 %d 的其他成员推送屏幕共享请求，发送者: %d", convID, c.userID)
	c.hub.SendToConversation(convID, c.userID, jsonMsg)
	log.Printf("用户 %d 请求屏幕共享，会话 %d", c.userID, convID)
}

// 处理屏幕共享请求（支持离线用户）
func handleScreenShareRequest(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		log.Printf("屏幕共享请求数据格式错误: %v", data)
		return
	}

	// 支持两种命名格式：下划线和驼峰
	var convID uint
	var found bool

	// 尝试从 conversation_id 获取
	if val, ok := msgData["conversation_id"]; ok {
		switch v := val.(type) {
		case float64:
			convID = uint(v)
			found = true
		case int:
			convID = uint(v)
			found = true
		case int64:
			convID = uint(v)
			found = true
		case string:
			if id, err := strconv.Atoi(v); err == nil {
				convID = uint(id)
				found = true
			}
		}
	}

	// 尝试从 conversationId 获取
	if !found && msgData["conversationId"] != nil {
		val := msgData["conversationId"]
		switch v := val.(type) {
		case float64:
			convID = uint(v)
			found = true
		case int:
			convID = uint(v)
			found = true
		case int64:
			convID = uint(v)
			found = true
		case string:
			if id, err := strconv.Atoi(v); err == nil {
				convID = uint(id)
				found = true
			}
		}
	}

	if !found {
		log.Printf("屏幕共享请求缺少会话ID: %v", msgData)
		return
	}

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		log.Printf("用户 %d 不是会话 %d 的成员", c.userID, convID)
		return
	}

	// 查询发送者昵称
	var senderNickname string
	if err := db.Model(&model.User{}).Where("id = ?", c.userID).Select("nickname").First(&senderNickname).Error; err != nil {
		log.Printf("查询用户昵称失败: %v，使用默认值", err)
		senderNickname = "未知用户"
	}

	// 构建屏幕共享请求消息
	wsMsg := WSMessage{
		Type: "screen-share.request",
		Data: map[string]interface{}{
			"conversation_id": convID,
			"user_id":         c.userID,
			"from_user_id":    c.userID,
			"from_user_name":  senderNickname,
			"timestamp":       time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	// 推送给会话其他成员（复用原有的 SendToConversation 逻辑）
	log.Printf("准备向会话 %d 的其他成员推送屏幕共享请求，发送者: %d", convID, c.userID)
	c.hub.SendToConversation(convID, c.userID, jsonMsg)
	log.Printf("用户 %d 请求屏幕共享，会话 %d", c.userID, convID)
}

// 处理屏幕共享响应
func handleScreenShareResponse(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	// 支持两种命名格式：下划线和驼峰
	var convIDFloat float64
	if val, ok := msgData["conversation_id"].(float64); ok {
		convIDFloat = val
	} else if val, ok := msgData["conversationId"].(float64); ok {
		convIDFloat = val
	} else {
		return
	}
	convID := uint(convIDFloat)

	// 获取请求者ID
	var requesterIDFloat float64
	if val, ok := msgData["requester_id"].(float64); ok {
		requesterIDFloat = val
	} else if val, ok := msgData["requesterId"].(float64); ok {
		requesterIDFloat = val
	} else {
		return
	}
	requesterID := uint(requesterIDFloat)

	// 获取响应状态
	status, ok := msgData["status"].(string)
	if !ok {
		return
	}

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		return
	}
	log.Printf("用户 %d 响应屏幕共享请求, 会话 %d, 请求者 %d, 状态 %s", c.userID, convID, requesterID, status)
	if status == "accepted" {
		// 向请求者发送接受消息
		acceptMsg := WSMessage{
			Type: "screen-share.accepted",
			Data: map[string]interface{}{
				"conversation_id": convID,
				"user_id":         c.userID,
				"timestamp":       time.Now().Unix(),
			},
		}
		acceptJson, _ := json.Marshal(acceptMsg)
		c.hub.SendToUser(requesterID, acceptJson)

		// 向响应者发送开始消息
		startMsg := WSMessage{
			Type: "screen-share.start",
			Data: map[string]interface{}{
				"conversation_id": convID,
				"user_id":         requesterID,
				"timestamp":       time.Now().Unix(),
			},
		}
		startJson, _ := json.Marshal(startMsg)
		c.hub.SendToUser(c.userID, startJson)
	} else if status == "rejected" {
		// 向请求者发送拒绝消息
		rejectMsg := WSMessage{
			Type: "screen-share.rejected",
			Data: map[string]interface{}{
				"conversation_id": convID,
				"user_id":         c.userID,
				"timestamp":       time.Now().Unix(),
			},
		}
		rejectJson, _ := json.Marshal(rejectMsg)
		c.hub.SendToUser(requesterID, rejectJson)

		log.Printf("用户 %d 拒绝了屏幕共享请求，会话 %d", c.userID, convID)
	}
}

// 处理视频通话邀请
func handleCallInvite(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		log.Printf("通话邀请数据格式错误: %v", data)
		return
	}

	var targetUserID uint
	if targetUserIDFloat, ok := msgData["target_user_id"].(float64); ok {
		targetUserID = uint(targetUserIDFloat)
	} else if targetUserIDStr, ok := msgData["target_user_id"].(string); ok {
		if id, err := strconv.ParseUint(targetUserIDStr, 10, 32); err == nil {
			targetUserID = uint(id)
		} else {
			log.Printf("解析 target_user_id 失败: %v", targetUserIDStr)
			return
		}
	} else {
		log.Printf("通话邀请缺少 target_user_id")
		return
	}

	callType, _ := msgData["call_type"].(string)
	signal := msgData["signal"]

	log.Printf("用户 %d 向用户 %d 发起 %s 通话邀请", c.userID, targetUserID, callType)

	// 转发通话邀请给目标用户
	callMsg := WSMessage{
		Type: "call.start",
		Data: map[string]interface{}{
			"from_user_id": c.userID,
			"call_type":    callType,
			"signal":       signal,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(callMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)
}

// 处理视频通话接听
func handleCallAccept(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		log.Printf("通话接听数据格式错误: %v", data)
		return
	}

	var targetUserID uint
	if targetUserIDFloat, ok := msgData["target_user_id"].(float64); ok {
		targetUserID = uint(targetUserIDFloat)
	} else if targetUserIDStr, ok := msgData["target_user_id"].(string); ok {
		if id, err := strconv.ParseUint(targetUserIDStr, 10, 32); err == nil {
			targetUserID = uint(id)
		} else {
			return
		}
	} else {
		return
	}

	signal := msgData["signal"]

	log.Printf("用户 %d 接听用户 %d 的通话", c.userID, targetUserID)

	// 转发接听消息给发起方
	callMsg := WSMessage{
		Type: "call.answer",
		Data: map[string]interface{}{
			"from_user_id": c.userID,
			"signal":       signal,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(callMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)
}

// 处理视频通话拒绝
func handleCallReject(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		log.Printf("通话拒绝数据格式错误: %v", data)
		return
	}

	var targetUserID uint
	if targetUserIDFloat, ok := msgData["target_user_id"].(float64); ok {
		targetUserID = uint(targetUserIDFloat)
	} else if targetUserIDStr, ok := msgData["target_user_id"].(string); ok {
		if id, err := strconv.ParseUint(targetUserIDStr, 10, 32); err == nil {
			targetUserID = uint(id)
		} else {
			return
		}
	} else {
		return
	}

	log.Printf("用户 %d 拒绝用户 %d 的通话", c.userID, targetUserID)

	// 转发拒绝消息给发起方
	callMsg := WSMessage{
		Type: "call.reject",
		Data: map[string]interface{}{
			"from_user_id": c.userID,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(callMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)
}

// 处理视频通话结束
func handleCallEnd(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		log.Printf("通话结束数据格式错误: %v", data)
		return
	}

	var targetUserID uint
	if targetUserIDFloat, ok := msgData["target_user_id"].(float64); ok {
		targetUserID = uint(targetUserIDFloat)
	} else if targetUserIDStr, ok := msgData["target_user_id"].(string); ok {
		if id, err := strconv.ParseUint(targetUserIDStr, 10, 32); err == nil {
			targetUserID = uint(id)
		} else {
			return
		}
	} else {
		return
	}

	log.Printf("用户 %d 结束与用户 %d 的通话", c.userID, targetUserID)

	// 转发通话结束消息给对方
	callMsg := WSMessage{
		Type: "call.end",
		Data: map[string]interface{}{
			"from_user_id": c.userID,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(callMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)
}

// StatusDebouncer 状态变更防抖器
type StatusDebouncer struct {
	mu     sync.Mutex
	timers map[uint]*time.Timer
	delay  time.Duration
}

func NewStatusDebouncer(delay time.Duration) *StatusDebouncer {
	return &StatusDebouncer{
		timers: make(map[uint]*time.Timer),
		delay:  delay,
	}
}

func (d *StatusDebouncer) Debounce(userID uint, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if timer, exists := d.timers[userID]; exists {
		timer.Stop()
	}

	d.timers[userID] = time.AfterFunc(d.delay, func() {
		fn()
		d.mu.Lock()
		delete(d.timers, userID)
		d.mu.Unlock()
	})
}

// UpdateUserStatus 更新用户状态并广播
func (h *Hub) UpdateUserStatus(userID uint, status string) {
	db := h.db
	now := time.Now()

	result := db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"status":      status,
		"last_online": now,
	})
	if result.Error != nil {
		log.Printf("更新用户状态失败: userID=%d, error=%v", userID, result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("用户 %d 状态变更为 %s", userID, status)
		h.statusDebouncer.Debounce(userID, func() {
			h.BroadcastUserStatus(userID, status)
		})
	}
}

// BroadcastUserStatus 广播用户状态变更
func (h *Hub) BroadcastUserStatus(userID uint, status string) {
	db := h.db
	var user model.User
	if err := db.Select("id", "username", "nickname", "avatar", "status", "last_online").
		First(&user, userID).Error; err != nil {
		log.Printf("获取用户信息失败: userID=%d, error=%v", userID, err)
		return
	}

	msg := WSMessage{
		Type: "user_status_changed",
		Data: map[string]interface{}{
			"user_id":  user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"avatar":   user.Avatar,
			"status":   status,
			"last_online": func() int64 {
				if user.LastOnline != nil {
					return user.LastOnline.Unix()
				}
				return 0
			}(),
			"timestamp": time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(msg)

	if subscribers, ok := h.userSubscribers.Load(userID); ok {
		for _, subscriberID := range subscribers.([]uint) {
			h.SendToUser(subscriberID, jsonMsg)
		}
	}

	h.BroadcastToConversationMembers(userID, jsonMsg)

	log.Printf("已向订阅者广播用户 %d 的状态变更: %s", userID, status)
}

// BroadcastToConversationMembers 向用户所在会话的成员广播状态变更
func (h *Hub) BroadcastToConversationMembers(userID uint, message []byte) {
	db := h.db

	var members []model.ConversationMember
	if err := db.Select("conversation_id").
		Where("user_id = ?", userID).
		Group("conversation_id").
		Find(&members).Error; err != nil {
		log.Printf("获取用户会话失败: userID=%d, error=%v", userID, err)
		return
	}

	for _, member := range members {
		h.SendToConversation(member.ConversationID, userID, message)
	}
}

// SubscribeUserStatus 订阅用户状态变更
func (h *Hub) SubscribeUserStatus(subscriberID, targetUserID uint) {
	h.userSubscribers.Range(func(key, value interface{}) bool {
		if key.(uint) == targetUserID {
			subscribers := value.([]uint)
			for _, sid := range subscribers {
				if sid == subscriberID {
					return false
				}
			}
			subscribers = append(subscribers, subscriberID)
			h.userSubscribers.Store(targetUserID, subscribers)
			return false
		}
		return true
	})

	h.userSubscribers.CompareAndSwap(targetUserID, nil, []uint{subscriberID})
}

// UnsubscribeUserStatus 取消订阅用户状态变更
func (h *Hub) UnsubscribeUserStatus(subscriberID, targetUserID uint) {
	h.userSubscribers.Range(func(key, value interface{}) bool {
		if key.(uint) == targetUserID {
			subscribers := value.([]uint)
			for i, sid := range subscribers {
				if sid == subscriberID {
					subscribers = append(subscribers[:i], subscribers[i+1:]...)
					if len(subscribers) == 0 {
						h.userSubscribers.Delete(key)
					} else {
						h.userSubscribers.Store(key, subscribers)
					}
					return false
				}
			}
		}
		return true
	})
}

// CleanupUserSubscriptions 清理用户的所有订阅
func (h *Hub) CleanupUserSubscriptions(userID uint) {
	h.userSubscribers.Range(func(key, value interface{}) bool {
		subscribers := value.([]uint)
		for i, sid := range subscribers {
			if sid == userID {
				subscribers = append(subscribers[:i], subscribers[i+1:]...)
				if len(subscribers) == 0 {
					h.userSubscribers.Delete(key)
				} else {
					h.userSubscribers.Store(key, subscribers)
				}
				break
			}
		}
		return true
	})
}

// handleSubscribeUserStatus 处理订阅用户状态请求
func handleSubscribeUserStatus(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		log.Printf("订阅用户状态数据格式错误")
		return
	}

	targetUserIDFloat, ok := msgData["user_id"].(float64)
	if !ok {
		log.Printf("订阅用户状态缺少 user_id")
		return
	}

	targetUserID := uint(targetUserIDFloat)
	log.Printf("用户 %d 订阅用户 %d 的状态变更", c.userID, targetUserID)

	c.hub.SubscribeUserStatus(c.userID, targetUserID)

	// 立即返回当前状态
	db := c.hub.db
	var user model.User
	if err := db.Select("id", "username", "nickname", "avatar", "status", "last_online").
		First(&user, targetUserID).Error; err == nil {
		msg := WSMessage{
			Type: "user_status_changed",
			Data: map[string]interface{}{
				"user_id":  user.ID,
				"username": user.Username,
				"nickname": user.Nickname,
				"avatar":   user.Avatar,
				"status":   user.Status,
				"last_online": func() int64 {
					if user.LastOnline != nil {
						return user.LastOnline.Unix()
					}
					return 0
				}(),
				"timestamp": time.Now().Unix(),
			},
		}
		jsonMsg, _ := json.Marshal(msg)
		c.hub.SendToUser(c.userID, jsonMsg)
	}
}

// handleUnsubscribeUserStatus 处理取消订阅用户状态请求
func handleUnsubscribeUserStatus(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		log.Printf("取消订阅用户状态数据格式错误")
		return
	}

	targetUserIDFloat, ok := msgData["user_id"].(float64)
	if !ok {
		log.Printf("取消订阅用户状态缺少 user_id")
		return
	}

	targetUserID := uint(targetUserIDFloat)
	log.Printf("用户 %d 取消订阅用户 %d 的状态变更", c.userID, targetUserID)

	c.hub.UnsubscribeUserStatus(c.userID, targetUserID)
}
