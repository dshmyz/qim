package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dshmyz/qim/qim-server/cache"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/utils"
	"github.com/golang-jwt/jwt/v5"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

const (
	StatusOnline  = "online"
	StatusOffline = "offline"
	StatusBusy    = "busy"

	// 状态变更防抖延迟
	StatusDebounceDelay = 500 * time.Millisecond

	// syncHintTimeout 是 sync_hint 重试窗口：自首次成功发送 hint 起，
	// 客户端仍未 acknowledge_sync（老客户端不支持 / 已失联）则停止重发。
	syncHintTimeout = 30 * time.Second
)

var GlobalHub *Hub

// wsAllowedOrigins 及其锁：CheckOrigin 在每个 WS 连接时并发读，
// SetAllowedOrigins 在配置重载时写。用 RWMutex 保护防止并发 map 读写 panic。

var (
	wsAllowedOrigins   map[string]bool
	wsAllowedOriginsMu sync.RWMutex
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		wsAllowedOriginsMu.RLock()
		origins := wsAllowedOrigins
		wsAllowedOriginsMu.RUnlock()
		if origins == nil {
			return true
		}
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		return origins[origin]
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
	nodeScheme          string // 节点间通信协议：http 或 https
	db                  *gorm.DB
	jwtSecret           string

	statusDebouncer *StatusDebouncer
	userSubscribers sync.Map
	versionStats    sync.Map // key: "version|platform" → *int64，用于版本分布统计

	// OnMessageSent 回调：消息发送后触发，用于智能回复/分身触发。
	// 透传完整消息对象（含 Type/Content/QuotedMessageID/QuotedMessage），
	// 使下游 AI 触发路径能按需读取消息属性（content、引用、文件等），
	// 而非每次新增需求就扩回调参数。
	OnMessageSent func(msg *model.Message, mentionUserIDs []uint)

	// HandleMessage 回调：处理 WebSocket 发送消息请求，由外部注入 MessageService 逻辑
	HandleMessage func(convID, senderID uint, msgType, content string, quotedMessageID *uint) (*model.Message, error)

	// HandleReadMessage 回调：处理 WebSocket 已读消息请求
	HandleReadMessage func(convID, userID uint) error

	// 并发发送信号量，限制同时发送的 goroutine 数量
	sendSem chan struct{}
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   uint
	username string // 用户名（来自 JWT，认证时缓存，用于状态变更日志）
	authed   bool   // 是否已通过认证
	jwtToken string // 待认证的 token
	version  string // 客户端版本号（用于版本分布统计）
	platform string // 客户端平台（windows/macos/linux）

	// needsSync 标记客户端发送缓冲区溢出，需要拉取离线消息补偿。
	// 由 SendToUser 在 channel 满时置位，由客户端 acknowledge_sync 后清除。
	needsSync atomic.Bool
	// lastSyncHintAt 最近一次成功发送 sync_hint 的 UnixNano 时间戳。
	// 用于超时兜底：syncHintTimeout 内客户端未 ack 则停止重发。
	lastSyncHintAt atomic.Int64
}

type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id,omitempty"`
}

// safeCloseSend 安全关闭 channel，防止重复 close 导致 panic

type wsClaims struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// handleAuth 处理 WebSocket 首条认证消息

type StatusDebouncer struct {
	mu     sync.Mutex
	timers map[uint]*time.Timer
	delay  time.Duration
}

type VersionStat struct {
	Version  string `json:"version"`
	Platform string `json:"platform"`
	Count    int64  `json:"count"`
}

// GetVersionStats 返回版本分布统计快照

func SetAllowedOrigins(origins []string) {
	wsAllowedOriginsMu.Lock()
	defer wsAllowedOriginsMu.Unlock()
	if len(origins) == 0 {
		wsAllowedOrigins = nil
		return
	}
	wsAllowedOrigins = map[string]bool{}
	for _, o := range origins {
		wsAllowedOrigins[o] = true
	}
}

func safeCloseSend(ch chan []byte) {
	defer func() { recover() }()
	close(ch)
}

// sendJSON 将 WSMessage 序列化后写入 c.send，交由 writePump 统一写出。
// 认证阶段 readPump 与 writePump 并发运行，不能直接 c.conn.WriteXXX，
// 否则会与 writePump 竞争同一连接，触发 gorilla 的 concurrent write panic。
// 非阻塞入队：认证失败/即将断连时入队失败直接丢弃即可。

func NewHub(db *gorm.DB, jwtSecret string, nodeScheme string) *Hub {
	// 生成节点 ID
	nodeID := generateNodeID()

	// 节点间通信协议默认 http
	if nodeScheme != "https" {
		nodeScheme = "http"
	}

	// 初始化广播通道
	broadcastChan := make(chan []byte)

	logger.WithModule("WS").Info("节点初始化完成", "nodeID", nodeID, "scheme", nodeScheme)

	return &Hub{
		clients:             sync.Map{},
		register:            make(chan *Client),
		unregister:          make(chan *Client),
		broadcast:           broadcastChan,
		Broadcast:           broadcastChan,
		userClients:         sync.Map{},
		conversationMembers: make(map[uint]cachedMembers),
		nodeID:              nodeID,
		nodeScheme:          nodeScheme,
		db:                  db,
		jwtSecret:           jwtSecret,
		statusDebouncer:     NewStatusDebouncer(StatusDebounceDelay),
		sendSem:             make(chan struct{}, 50),
	}
}

// SetNodes 设置其他节点地址列表，由应用初始化时从配置注入（cluster.nodes）。
// 未调用或传空时保持单节点模式：跨节点推送为空循环（当前默认）。
// 节点列表变更需在注册任何客户端前完成。
func (h *Hub) SetNodes(nodes []string) {
	h.nodes = nodes
	logger.WithModule("WS").Info("节点列表已注入", "count", len(nodes), "nodes", nodes)
}

// generateNodeID 生成唯一的节点 ID

func generateNodeID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString 生成指定长度的随机字符串（小写+大写+数字）。
//
// 修复点：原实现 rand.Read 后用 `%len(letterBytes)` 映射到 62 字符的字母表，
// 因 256 不是 62 整数倍（256=4*62+8），前 8 个字母概率偏高（mod bias），
// 实际上破坏了 rand.Read 的均匀分布。
//
// 现统一改用 utils.RandomString（基于 crand.Int 在 [0, 62) 上严格均匀采样），
// 与项目内 handler.generateShortCode 复用同一份实现。

func randomString(n int) string {
	return utils.RandomString(n, utils.Alphanumeric)
}

func (h *Hub) Run() {
	// 启动节点间通信服务
	utils.SafeGoWithLabel("node-comm", func() { h.startNodeCommunication() })

	// 定期向缓冲区溢出的客户端发送 sync_hint，触发增量拉取补偿。
	// 间隔 3s：比消息投递频率低，不会造成 sync 风暴；比 30s ping 间隔短，
	// 确保慢客户端能在合理时间内收到补偿提示。
	utils.SafeGoWithLabel("sync-hint", func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			h.sendSyncHints()
		}
	})

	// 定期清理过期的会话成员缓存
	utils.SafeGoWithLabel("cache-cleanup", func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			h.cleanExpiredConversationCache()
		}
	})

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
			h.incVersionStats(client.version, client.platform)
			logger.WithModule("WS").Info("用户连接", "userID", client.userID)

			// 更新用户在线状态并广播
			h.UpdateUserStatus(client.userID, client.username, StatusOnline)

		case client := <-h.unregister:
			h.clients.LoadAndDelete(client)
			safeCloseSend(client.send)
			h.decVersionStats(client.version, client.platform)

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
					h.UpdateUserStatus(client.userID, client.username, StatusOffline)
				} else {
					h.userClients.Store(client.userID, clients)
				}
			}

			h.CleanupUserSubscriptions(client.userID)

			// userID=0 表示连接从未完成认证（未发合法的 auth 消息）就断开。
			// 这类连接从未被 register，断开是无害空操作，但会刷日志。带出对端地址便于定位来源
			// （浏览器/探活脚本直接访问 ws、或某个把 token 放 URL 且不发 auth 的老连接）。
			if client.userID == 0 {
				remote := "unknown"
				if client.conn != nil && client.conn.RemoteAddr() != nil {
					remote = client.conn.RemoteAddr().String()
				}
				logger.WithModule("WS").Warn("未认证连接断开", "remote", remote, "version", client.version, "platform", client.platform)
			} else {
				logger.WithModule("WS").Info("用户断开连接", "userID", client.userID)
			}

		case message := <-h.broadcast:
			// 异步广播，不阻塞事件循环
			utils.SafeGoWithLabel("broadcast", func() { h.asyncBroadcast(message) })
		}
	}
}

// sendSyncHints 遍历所有客户端，向 needsSync 标记的客户端发送 sync_hint。
// 客户端收到后通过 REST API 增量拉取离线消息（after_id 参数）补偿丢失的推送。
// 发送成功不清除标记——由客户端 acknowledge_sync 清除，保证拉取失败时后续轮询重发；
// 超过 syncHintTimeout 仍无 ack（老客户端不支持 / 已失联）则放弃重发，避免无限推送。
func (h *Hub) sendSyncHints() {
	syncMsg := WSMessage{
		Type: "sync_hint",
		Data: map[string]interface{}{
			"reason":  "buffer_overflow",
			"message": "消息队列已满，请拉取离线消息",
		},
	}
	jsonMsg, _ := json.Marshal(syncMsg)

	h.clients.Range(func(key, value interface{}) bool {
		client := key.(*Client)
		if !client.needsSync.Load() {
			return true
		}

		now := time.Now()
		lastSent := client.lastSyncHintAt.Load()
		if lastSent > 0 {
			// 超时兜底：重试窗口内客户端仍未 ack，停止重发
			if now.Sub(time.Unix(0, lastSent)) > syncHintTimeout {
				client.needsSync.Store(false)
				return true
			}
		}

		select {
		case client.send <- jsonMsg:
			// 发送成功只记录时间，不清标记（可靠性：等 ack）
			client.lastSyncHintAt.Store(now.UnixNano())
		default:
			// channel 仍然满，保持标记，下次轮询再试
		}
		return true
	})
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

	var wg sync.WaitGroup
	failedChan := make(chan *Client, len(clients))

	for _, client := range clients {
		h.sendSem <- struct{}{} // 获取信号量，超过容量时阻塞排队
		wg.Add(1)
		c := client
		utils.SafeGo(func() {
			defer wg.Done()
			defer func() { <-h.sendSem }() // 释放信号量
			select {
			case c.send <- message:
			default:
				failedChan <- c
			}
		})
	}

	wg.Wait()
	close(failedChan)

	for client := range failedChan {
		h.clients.Delete(client)
		safeCloseSend(client.send)
		// 与 unregister 路径一致：清理失败客户端时同步扣减版本统计，避免分布虚高
		h.decVersionStats(client.version, client.platform)

		// 同步清理 userClients，防止悬空
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
				h.UpdateUserStatus(client.userID, client.username, StatusOffline)
			} else {
				h.userClients.Store(client.userID, clients)
			}
		}
	}

	h.broadcastToOtherNodes(message)
}

// startNodeCommunication 启动节点间通信服务

func (h *Hub) SendToUser(userID uint, message []byte) {
	if existingClients, ok := h.userClients.Load(userID); ok {
		clients := existingClients.([]*Client)
		for _, client := range clients {
			select {
			case client.send <- message:
			default:
				// 缓冲区满：标记客户端需要同步，触发后续 sync_hint 增量拉取补偿。
				// 不再静默丢弃——消息已落库，客户端收到 sync_hint 后可按 after_id 拉回。
				if !client.needsSync.Load() {
					client.needsSync.Store(true)
					logger.WithModule("WS").Warn("客户端发送缓冲区溢出，标记需要同步",
						"userID", userID, "channel_len", len(client.send), "channel_cap", cap(client.send))
				}
			}
		}
	}

	h.sendToUserToOtherNodes(userID, message)
}

func (h *Hub) BroadcastToAllOnlineUsers(message []byte) {
	h.userClients.Range(func(key, value interface{}) bool {
		userID := key.(uint)
		h.SendToUser(userID, message)
		return true
	})
}

// BroadcastNewVersion 新版本发布时主动推送给所有在线客户端
// 预留接口：客户端收到后可立即触发 checkForUpdates

func (h *Hub) BroadcastNewVersion(version, platform string, forceUpdate bool) {
	msg, _ := json.Marshal(map[string]interface{}{
		"type": "new_version_available",
		"data": map[string]interface{}{
			"version":      version,
			"platform":     platform,
			"force_update": forceUpdate,
			"timestamp":    time.Now().Format(time.RFC3339),
		},
	})
	h.Broadcast <- msg
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
		logger.WithModule("WS").Error("更新会话成员缓存失败", "error", result.Error)
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

	// 同步失效 service 层的会话成员缓存（GetConversationMembers），
	// 避免成员变更后 TTL 内读到陈旧成员列表。所有成员变更调用点都经由本方法。
	cache.InvalidateConversationMemberCache(convID)

	logger.WithModule("WS").Info("更新会话成员缓存", "convID", convID, "memberCount", len(memberIDs))
}

func (h *Hub) cleanExpiredConversationCache() {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for convID, cached := range h.conversationMembers {
		if now.After(cached.expiredAt) {
			delete(h.conversationMembers, convID)
		}
	}
}

// GetCachedMemberIDs 返回指定会话的成员 ID 列表缓存。
// 缓存命中且未过期时直接返回；否则返回 nil，由调用方 fallback 到 DB。
// 用于 handler 层（如 GetMessages 的 @all 展开）避免每次请求都查 DB。
func (h *Hub) GetCachedMemberIDs(convID uint) []uint {
	h.mu.RLock()
	cached, found := h.conversationMembers[convID]
	h.mu.RUnlock()
	if found && time.Now().Before(cached.expiredAt) {
		return cached.memberIDs
	}
	return nil
}

// sendToUserToOtherNodes 通过 HTTP 向其他节点发送用户特定消息

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
			logger.WithModule("WS").Error("查询会话成员失败", "error", result.Error)
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
			logger.WithModule("WS").Error("查询会话成员失败", "error", result.Error)
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

	// 真正异步发送：不等待完成，但用信号量限制并发
	for _, userID := range memberIDs {
		if userID != excludeUserID {
			uid := userID
			h.sendSem <- struct{}{}
			utils.SafeGo(func() {
				defer func() { <-h.sendSem }()
				h.SendToUser(uid, message)
			})
		}
	}
}

// wsClaims WebSocket 认证用的 JWT Claims（避免循环导入 middleware 包）
