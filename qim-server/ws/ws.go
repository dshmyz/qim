package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"qim-server/database"
	"qim-server/model"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var GlobalHub *Hub

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	clients     map[*Client]bool
	register    chan *Client
	unregister  chan *Client
	broadcast   chan []byte
	Broadcast   chan []byte
	userClients map[uint][]*Client
	mu          sync.RWMutex
	nodes       []string
	nodeID      string
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

func NewHub() *Hub {
	// 生成节点 ID
	nodeID := generateNodeID()

	// 初始化节点列表（这里可以从配置文件或环境变量中读取）
	nodes := []string{}

	// 初始化广播通道
	broadcastChan := make(chan []byte)

	log.Printf("节点 %s 初始化完成，将使用基于 HTTP 的多节点模式", nodeID)

	return &Hub{
		clients:     make(map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   broadcastChan,
		Broadcast:   broadcastChan,
		userClients: make(map[uint][]*Client),
		nodes:       nodes,
		nodeID:      nodeID,
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
	for i := range b {
		b[i] = letterBytes[time.Now().UnixNano()%int64(len(letterBytes))]
	}
	return string(b)
}

func (h *Hub) Run() {
	// 启动节点间通信服务
	go h.startNodeCommunication()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.userClients[client.userID] = append(h.userClients[client.userID], client)
			h.mu.Unlock()
			log.Printf("用户 %d 连接", client.userID)

			// 更新用户在线状态
			db := database.GetDB()
			db.Model(&model.User{}).Where("id = ?", client.userID).Update("status", "online")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				// 从userClients中移除
				clients := h.userClients[client.userID]
				for i, c := range clients {
					if c == client {
						h.userClients[client.userID] = append(clients[:i], clients[i+1:]...)
						break
					}
				}

				// 如果没有其他连接，更新为离线
				if len(h.userClients[client.userID]) == 0 {
					db := database.GetDB()
					db.Model(&model.User{}).Where("id = ?", client.userID).Update("status", "offline")
					delete(h.userClients, client.userID)
				}
			}
			h.mu.Unlock()
			log.Printf("用户 %d 断开连接", client.userID)

		case message := <-h.broadcast:
			// 广播给本地客户端
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()

			// 通过 HTTP 广播给其他节点
			h.broadcastToOtherNodes(message)
		}
	}
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
	// 向本地客户端发送消息
	h.mu.RLock()
	clients := h.userClients[userID]
	h.mu.RUnlock()

	log.Printf("找到用户 %d 的本地WebSocket连接数量: %d", userID, len(clients))

	for i, client := range clients {
		log.Printf("向用户 %d 的第 %d 个本地连接发送WebSocket消息", userID, i+1)
		select {
		case client.send <- message:
			log.Printf("消息发送成功")
		default:
			log.Printf("消息发送失败，连接可能已关闭")
		}
	}

	// 通过 HTTP 向其他节点发送消息
	h.sendToUserToOtherNodes(userID, message)
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
	log.Printf("开始向会话 %d 发送WebSocket消息，排除用户 %d", convID, excludeUserID)

	db := database.GetDB()
	var members []model.ConversationMember
	result := db.Where("conversation_id = ?", convID).Find(&members)
	log.Printf("找到会话 %d 的成员数量: %d", convID, len(members))
	if result.Error != nil {
		log.Printf("查询会话成员失败: %v", result.Error)
		return
	}

	for _, member := range members {
		if member.UserID != excludeUserID {
			log.Printf("向用户 %d 发送WebSocket消息", member.UserID)
			h.SendToUser(member.UserID, message)
		} else {
			log.Printf("排除用户 %d，不发送消息", member.UserID)
		}
	}
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

		switch msg.Type {
		case "heartbeat":
			// 心跳，无需处理
		case "send_message":
			handleSendMessage(c, msg.Data)
		case "read_message":
			handleReadMessage(c, msg.Data)
		case "webrtc_offer":
			handleWebRTCSignal(c, msg.Data, "webrtc_offer")
		case "webrtc_answer":
			handleWebRTCSignal(c, msg.Data, "webrtc_answer")
		case "webrtc_ice_candidate":
			handleWebRTCSignal(c, msg.Data, "webrtc_ice_candidate")
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
	db := database.GetDB()

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

	// 构建推送消息
	wsMsg := WSMessage{
		Type: "new_message",
		Data: msg,
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	// 推送给会话其他成员
	c.hub.SendToConversation(convID, c.userID, jsonMsg)
}

func handleReadMessage(c *Client, data interface{}) {
	db := database.GetDB()

	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	convIDFloat, _ := msgData["conversation_id"].(float64)
	convID := uint(convIDFloat)
	msgIDFloat, _ := msgData["message_id"].(float64)
	msgID := uint(msgIDFloat)

	// 更新成员未读数和最后读取
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, c.userID).
		Updates(map[string]interface{}{
			"unread_count": 0,
			"last_read_at": time.Now(),
		})

	// 记录已读回执
	db.Create(&model.MessageReadReceipt{
		MessageID:      msgID,
		ConversationID: convID,
		UserID:         c.userID,
	})

	// 标记消息为已读（只标记非自己发送的消息）
	result := db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND is_read = false", convID, c.userID).
		UpdateColumn("is_read", true)

	// 只有当确实有消息被标记为已读时，才发送已读回执通知给对方
	if result.RowsAffected > 0 {
		// 发送已读回执通知给对方
		var conv model.Conversation
		db.First(&conv, convID)

		// 构建已读回执消息
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
			// 对于单聊，找到对方用户
			var otherMember model.ConversationMember
			db.Where("conversation_id = ? AND user_id != ?", convID, c.userID).First(&otherMember)

			// 发送给对方用户
			c.hub.SendToUser(otherMember.UserID, jsonMsg)
		} else if conv.Type == "group" {
			// 对于群聊，发送给所有其他成员
			var members []model.ConversationMember
			db.Where("conversation_id = ? AND user_id != ?", convID, c.userID).Find(&members)

			for _, member := range members {
				// 发送给每个成员
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

	targetUserIDFloat, ok := msgData["target_user_id"].(float64)
	if !ok {
		return
	}
	targetUserID := uint(targetUserIDFloat)

	// 构建转发的信令消息
	signalMsg := WSMessage{
		Type: signalType,
		Data: map[string]interface{}{
			"from_user_id": c.userID,
			"signal":       msgData["signal"],
			"call_id":      msgData["call_id"],
		},
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

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), userID: userID.(uint)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
