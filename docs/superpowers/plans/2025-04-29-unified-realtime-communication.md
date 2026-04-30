# 统一实时通信架构实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 重构屏幕共享功能，实现统一实时通信架构，支持多人加入、审批机制、状态持久化、跨会话可见。

**架构：** 统一的 RealtimeSession 领域模型，事件驱动架构，Mesh 模式 WebRTC，Pinia 状态管理。

**技术栈：** Go (GORM) + Vue 3 (Pinia) + WebRTC + WebSocket

---

## 文件结构

### 服务端新增文件

```
qim-server/
├── model/
│   └── realtime.go              # RealtimeSession 和 RealtimeParticipant 模型
├── handler/
│   └── realtime_handler.go      # REST API handlers
└── ws/
    └── realtime.go              # WebSocket 事件处理
```

### 服务端修改文件

```
qim-server/
├── model/model.go               # 注册新模型到 AutoMigrate
├── app/routes.go                # 添加 REST API 路由
└── ws/ws.go                     # 添加 WebSocket 事件处理
```

### 前端新增文件

```
qim-client/src/
├── types/
│   └── realtime.ts              # TypeScript 类型定义
├── stores/
│   └── realtime.ts              # Pinia store
├── components/realtime/
│   ├── RealtimeSessionCard.vue  # 统一会话卡片
│   ├── JoinRequestModal.vue     # 加入请求弹窗
│   └── ViewerList.vue           # 观看者列表
└── utils/
    └── realtimeConnection.ts    # WebRTC 连接管理
```

### 前端修改文件

```
qim-client/src/
├── components/chat/
│   ├── ChatWindow.vue           # 集成实时通信
│   └── MessageManager.vue       # 渲染系统消息卡片
└── utils/
    └── webrtc.js                # 重构 ScreenShareSender
```

---

## 任务 1：数据库模型

**文件：**
- 创建：`qim-server/model/realtime.go`
- 修改：`qim-server/model/model.go`

### 步骤 1.1：创建 RealtimeSession 模型

- [ ] **创建 `qim-server/model/realtime.go`**

```go
package model

import "time"

// RealtimeSession 实时会话
type RealtimeSession struct {
	ID             string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Type           string     `json:"type" gorm:"type:varchar(20);not null;index"` // screen_share, voice_call, video_call
	InitiatorID    uint       `json:"initiator_id" gorm:"not null;index"`
	ConversationID uint       `json:"conversation_id" gorm:"not null;index"`
	Status         string     `json:"status" gorm:"type:varchar(20);default:'pending'"` // pending, active, paused, ended
	StartedAt      *time.Time `json:"started_at"`
	EndedAt        *time.Time `json:"ended_at"`
	Metadata       string     `json:"metadata" gorm:"type:text"` // JSON 扩展字段
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	
	Initiator      User       `json:"initiator,omitempty" gorm:"foreignKey:InitiatorID"`
	Participants   []RealtimeParticipant `json:"participants,omitempty" gorm:"foreignKey:SessionID"`
}

// RealtimeParticipant 实时会话参与者
type RealtimeParticipant struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	SessionID   string     `json:"session_id" gorm:"not null;index"`
	UserID      uint       `json:"user_id" gorm:"not null;index"`
	Role        string     `json:"role" gorm:"type:varchar(20);default:'viewer'"` // initiator, viewer
	Status      string     `json:"status" gorm:"type:varchar(20);default:'pending'"` // pending, approved, rejected, joined, left
	RequestedAt time.Time  `json:"requested_at"`
	ApprovedAt  *time.Time `json:"approved_at"`
	JoinedAt    *time.Time `json:"joined_at"`
	LeftAt      *time.Time `json:"left_at"`
	
	User        User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (RealtimeSession) TableName() string {
	return "realtime_sessions"
}

func (RealtimeParticipant) TableName() string {
	return "realtime_participants"
}
```

### 步骤 1.2：注册模型到 AutoMigrate

- [ ] **修改 `qim-server/model/model.go`，在文件末尾添加**

找到 `AutoMigrate` 调用的位置，添加新模型：

```go
// 在 AutoMigrate 中添加新模型
db.AutoMigrate(
	&User{},
	&Conversation{},
	&ConversationMember{},
	&Message{},
	&FriendRequest{},
	&UserAIConfig{},
	&RealtimeSession{},      // 新增
	&RealtimeParticipant{},  // 新增
)
```

### 步骤 1.3：验证数据库迁移

- [ ] **运行服务端，确认表创建成功**

```bash
cd qim-server && go run main.go
```

预期：服务启动时自动创建 `realtime_sessions` 和 `realtime_participants` 表。

---

## 任务 2：服务端 REST API

**文件：**
- 创建：`qim-server/handler/realtime_handler.go`
- 修改：`qim-server/app/routes.go`

### 步骤 2.1：创建 REST API handlers

- [ ] **创建 `qim-server/handler/realtime_handler.go`**

```go
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"qim-server/database"
	"qim-server/model"
)

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	Type           string `json:"type" binding:"required"`
	ConversationID uint   `json:"conversation_id" binding:"required"`
}

// CreateSession 创建实时会话
func CreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	db := database.GetDB()

	// 检查用户是否已有活跃会话
	var existingSession model.RealtimeSession
	if err := db.Where("initiator_id = ? AND status IN ?", userID, []string{"pending", "active"}).First(&existingSession).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "已有活跃会话"})
		return
	}

	// 创建会话
	session := model.RealtimeSession{
		ID:             uuid.New().String(),
		Type:           req.Type,
		InitiatorID:    userID,
		ConversationID: req.ConversationID,
		Status:         "pending",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败"})
		return
	}

	// 创建发起者参与者记录
	participant := model.RealtimeParticipant{
		ID:          uuid.New().String(),
		SessionID:   session.ID,
		UserID:      userID,
		Role:        "initiator",
		Status:      "joined",
		RequestedAt: time.Now(),
		JoinedAt:    ptrTime(time.Now()),
	}
	db.Create(&participant)

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// GetSession 获取会话详情
func GetSession(c *gin.Context) {
	sessionID := c.Param("id")
	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.Preload("Initiator").Preload("Participants.User").First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// GetActiveSessions 获取活跃会话列表
func GetActiveSessions(c *gin.Context) {
	initiatorID := c.Query("initiator_id")
	db := database.GetDB()

	var sessions []model.RealtimeSession
	query := db.Where("status IN ?", []string{"pending", "active"})
	
	if initiatorID != "" {
		query = query.Where("initiator_id = ?", initiatorID)
	}
	
	if err := query.Preload("Initiator").Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// EndSession 结束会话
func EndSession(c *gin.Context) {
	sessionID := c.Param("id")
	userID := c.GetUint("user_id")
	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 只有发起者可以结束会话
	if session.InitiatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return
	}

	now := time.Now()
	session.Status = "ended"
	session.EndedAt = &now
	session.UpdatedAt = now

	db.Save(&session)

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// RequestJoin 申请加入会话
func RequestJoin(c *gin.Context) {
	sessionID := c.Param("id")
	userID := c.GetUint("user_id")
	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 检查是否已经是参与者
	var existingParticipant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&existingParticipant).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "已申请或已加入"})
		return
	}

	participant := model.RealtimeParticipant{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		UserID:      userID,
		Role:        "viewer",
		Status:      "pending",
		RequestedAt: time.Now(),
	}

	if err := db.Create(&participant).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "申请失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"participant": participant})
}

// ApproveJoin 审批加入请求
func ApproveJoin(c *gin.Context) {
	sessionID := c.Param("id")
	targetUserID := c.Param("user_id")
	userID := c.GetUint("user_id")
	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 只有发起者可以审批
	if session.InitiatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return
	}

	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, targetUserID).First(&participant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "参与者不存在"})
		return
	}

	now := time.Now()
	participant.Status = "approved"
	participant.ApprovedAt = &now

	db.Save(&participant)

	c.JSON(http.StatusOK, gin.H{"participant": participant})
}

// RejectJoin 拒绝加入请求
func RejectJoin(c *gin.Context) {
	sessionID := c.Param("id")
	targetUserID := c.Param("user_id")
	userID := c.GetUint("user_id")
	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	if session.InitiatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return
	}

	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, targetUserID).First(&participant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "参与者不存在"})
		return
	}

	participant.Status = "rejected"
	db.Save(&participant)

	c.JSON(http.StatusOK, gin.H{"participant": participant})
}

// LeaveSession 离开会话
func LeaveSession(c *gin.Context) {
	sessionID := c.Param("id")
	userID := c.GetUint("user_id")
	db := database.GetDB()

	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&participant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "参与者不存在"})
		return
	}

	now := time.Now()
	participant.Status = "left"
	participant.LeftAt = &now
	db.Save(&participant)

	c.JSON(http.StatusOK, gin.H{"participant": participant})
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
```

### 步骤 2.2：注册路由

- [ ] **修改 `qim-server/app/routes.go`，添加路由**

在路由注册部分添加：

```go
// 实时通信 API
realtime := api.Group("/realtime")
{
	realtime.POST("/sessions", handler.CreateSession)
	realtime.GET("/sessions/:id", handler.GetSession)
	realtime.GET("/sessions", handler.GetActiveSessions)
	realtime.PATCH("/sessions/:id", handler.EndSession)
	realtime.POST("/sessions/:id/participants", handler.RequestJoin)
	realtime.PATCH("/sessions/:id/participants/:user_id", handler.ApproveJoin)
	realtime.DELETE("/sessions/:id/participants/:user_id", handler.RejectJoin)
	realtime.DELETE("/sessions/:id/participants", handler.LeaveSession)
}
```

### 步骤 2.3：验证 API

- [ ] **启动服务端，测试 API**

```bash
curl -X POST http://localhost:8080/api/realtime/sessions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"type":"screen_share","conversation_id":1}'
```

预期：返回创建的会话信息。

---

## 任务 3：服务端 WebSocket 事件处理

**文件：**
- 创建：`qim-server/ws/realtime.go`
- 修改：`qim-server/ws/ws.go`

### 步骤 3.1：创建 WebSocket 事件处理

- [ ] **创建 `qim-server/ws/realtime.go`**

```go
package ws

import (
	"encoding/json"
	"log"
	"time"

	"qim-server/database"
	"qim-server/model"
)

// HandleRealtimeSessionCreate 处理创建实时会话
func HandleRealtimeSessionCreate(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	sessionID, _ := msgData["session_id"].(string)
	conversationID, _ := msgData["conversation_id"].(float64)
	sessionType, _ := msgData["type"].(string)

	db := database.GetDB()

	// 获取会话成员
	var members []model.ConversationMember
	db.Where("conversation_id = ?", uint(conversationID)).Preload("User").Find(&members)

	// 广播给会话成员
	broadcastMsg := WSMessage{
		Type: "realtime:session:created",
		Data: map[string]interface{}{
			"session_id":      sessionID,
			"type":            sessionType,
			"initiator_id":    c.userID,
			"conversation_id": uint(conversationID),
			"timestamp":       time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(broadcastMsg)

	for _, member := range members {
		if member.UserID != c.userID {
			c.hub.SendToUser(member.UserID, jsonMsg)
		}
	}

	log.Printf("用户 %d 创建实时会话 %s", c.userID, sessionID)
}

// HandleRealtimeJoinRequest 处理申请加入
func HandleRealtimeJoinRequest(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	sessionID, _ := msgData["session_id"].(string)

	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		return
	}

	// 通知发起者
	notifyMsg := WSMessage{
		Type: "realtime:join:requested",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"user_id":    c.userID,
			"timestamp":  time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(notifyMsg)
	c.hub.SendToUser(session.InitiatorID, jsonMsg)

	log.Printf("用户 %d 申请加入会话 %s", c.userID, sessionID)
}

// HandleRealtimeJoinApprove 处理批准加入
func HandleRealtimeJoinApprove(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	sessionID, _ := msgData["session_id"].(string)
	viewerID, _ := msgData["user_id"].(float64)

	// 通知观看者
	notifyMsg := WSMessage{
		Type: "realtime:join:approved",
		Data: map[string]interface{}{
			"session_id":   sessionID,
			"initiator_id": c.userID,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(notifyMsg)
	c.hub.SendToUser(uint(viewerID), jsonMsg)

	log.Printf("用户 %d 批准用户 %d 加入会话 %s", c.userID, uint(viewerID), sessionID)
}

// HandleRealtimeJoinReject 处理拒绝加入
func HandleRealtimeJoinReject(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	sessionID, _ := msgData["session_id"].(string)
	viewerID, _ := msgData["user_id"].(float64)

	notifyMsg := WSMessage{
		Type: "realtime:join:rejected",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"timestamp":  time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(notifyMsg)
	c.hub.SendToUser(uint(viewerID), jsonMsg)

	log.Printf("用户 %d 拒绝用户 %d 加入会话 %s", c.userID, uint(viewerID), sessionID)
}

// HandleRealtimeLeave 处理离开会话
func HandleRealtimeLeave(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	sessionID, _ := msgData["session_id"].(string)

	// 通知发起者
	db := database.GetDB()
	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err == nil {
		notifyMsg := WSMessage{
			Type: "realtime:participant:left",
			Data: map[string]interface{}{
				"session_id": sessionID,
				"user_id":    c.userID,
				"timestamp":  time.Now().Unix(),
			},
		}
		jsonMsg, _ := json.Marshal(notifyMsg)
		c.hub.SendToUser(session.InitiatorID, jsonMsg)
	}

	log.Printf("用户 %d 离开会话 %s", c.userID, sessionID)
}

// HandleRealtimeSessionEnd 处理结束会话
func HandleRealtimeSessionEnd(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	sessionID, _ := msgData["session_id"].(string)

	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		return
	}

	// 获取所有参与者
	var participants []model.RealtimeParticipant
	db.Where("session_id = ? AND status = ?", sessionID, "joined").Find(&participants)

	// 通知所有参与者
	notifyMsg := WSMessage{
		Type: "realtime:session:ended",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"timestamp":  time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(notifyMsg)

	for _, p := range participants {
		c.hub.SendToUser(p.UserID, jsonMsg)
	}

	log.Printf("用户 %d 结束会话 %s", c.userID, sessionID)
}

// HandleRealtimeWebRTCOffer 处理 WebRTC offer
func HandleRealtimeWebRTCOffer(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	targetUserID, _ := msgData["target_user_id"].(float64)
	sessionID, _ := msgData["session_id"].(string)
	signal := msgData["signal"]

	offerMsg := WSMessage{
		Type: "realtime:webrtc:offer",
		Data: map[string]interface{}{
			"session_id":    sessionID,
			"from_user_id": c.userID,
			"signal":       signal,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(offerMsg)
	c.hub.SendToUser(uint(targetUserID), jsonMsg)
}

// HandleRealtimeWebRTCAnswer 处理 WebRTC answer
func HandleRealtimeWebRTCAnswer(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	targetUserID, _ := msgData["target_user_id"].(float64)
	sessionID, _ := msgData["session_id"].(string)
	signal := msgData["signal"]

	answerMsg := WSMessage{
		Type: "realtime:webrtc:answer",
		Data: map[string]interface{}{
			"session_id":    sessionID,
			"from_user_id": c.userID,
			"signal":       signal,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(answerMsg)
	c.hub.SendToUser(uint(targetUserID), jsonMsg)
}

// HandleRealtimeWebRTCIce 处理 WebRTC ICE candidate
func HandleRealtimeWebRTCIce(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	targetUserID, _ := msgData["target_user_id"].(float64)
	sessionID, _ := msgData["session_id"].(string)
	signal := msgData["signal"]

	iceMsg := WSMessage{
		Type: "realtime:webrtc:ice",
		Data: map[string]interface{}{
			"session_id":    sessionID,
			"from_user_id": c.userID,
			"signal":       signal,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(iceMsg)
	c.hub.SendToUser(uint(targetUserID), jsonMsg)
}
```

### 步骤 3.2：注册 WebSocket 事件处理

- [ ] **修改 `qim-server/ws/ws.go`，添加事件处理**

在 `messageTypeHandlers` map 中添加：

```go
var messageTypeHandlers = map[string]MessageHandler{
	// ... 现有 handlers ...
	
	// 实时通信事件
	"realtime:session:create":  HandleRealtimeSessionCreate,
	"realtime:session:end":     HandleRealtimeSessionEnd,
	"realtime:join:request":    HandleRealtimeJoinRequest,
	"realtime:join:approve":    HandleRealtimeJoinApprove,
	"realtime:join:reject":     HandleRealtimeJoinReject,
	"realtime:leave":           HandleRealtimeLeave,
	"realtime:webrtc:offer":    HandleRealtimeWebRTCOffer,
	"realtime:webrtc:answer":   HandleRealtimeWebRTCAnswer,
	"realtime:webrtc:ice":      HandleRealtimeWebRTCIce,
}
```

---

## 任务 4：前端类型定义

**文件：**
- 创建：`qim-client/src/types/realtime.ts`

### 步骤 4.1：创建类型定义

- [ ] **创建 `qim-client/src/types/realtime.ts`**

```typescript
export type SessionType = 'screen_share' | 'voice_call' | 'video_call';

export type SessionStatus = 'pending' | 'active' | 'paused' | 'ended';

export type ParticipantRole = 'initiator' | 'viewer';

export type ParticipantStatus = 'pending' | 'approved' | 'rejected' | 'joined' | 'left';

export interface RealtimeSession {
  id: string;
  type: SessionType;
  initiator_id: number;
  conversation_id: number;
  status: SessionStatus;
  started_at: string | null;
  ended_at: string | null;
  metadata: string | null;
  created_at: string;
  updated_at: string;
  initiator?: {
    id: number;
    nickname: string;
    avatar: string;
  };
  participants?: RealtimeParticipant[];
}

export interface RealtimeParticipant {
  id: string;
  session_id: string;
  user_id: number;
  role: ParticipantRole;
  status: ParticipantStatus;
  requested_at: string;
  approved_at: string | null;
  joined_at: string | null;
  left_at: string | null;
  user?: {
    id: number;
    nickname: string;
    avatar: string;
  };
}

export interface JoinRequest {
  session_id: string;
  user_id: number;
  user?: {
    id: number;
    nickname: string;
    avatar: string;
  };
  timestamp: number;
}

export interface WebRTCOfferData {
  session_id: string;
  from_user_id: number;
  signal: RTCSessionDescriptionInit;
  timestamp: number;
}

export interface WebRTCAnswerData {
  session_id: string;
  from_user_id: number;
  signal: RTCSessionDescriptionInit;
  timestamp: number;
}

export interface WebRTCIceData {
  session_id: string;
  from_user_id: number;
  signal: RTCIceCandidateInit;
  timestamp: number;
}
```

---

## 任务 5：前端 Pinia Store

**文件：**
- 创建：`qim-client/src/stores/realtime.ts`

### 步骤 5.1：创建 Pinia Store

- [ ] **创建 `qim-client/src/stores/realtime.ts`**

```typescript
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { RealtimeSession, RealtimeParticipant, JoinRequest } from '../types/realtime';

export const useRealtimeStore = defineStore('realtime', () => {
  // State
  const activeSessions = ref<RealtimeSession[]>([]);
  const mySession = ref<RealtimeSession | null>(null);
  const pendingRequests = ref<JoinRequest[]>([]);
  const currentViewingSession = ref<RealtimeSession | null>(null);

  // Getters
  const isSharing = computed(() => {
    return mySession.value?.type === 'screen_share' && 
           (mySession.value?.status === 'active' || mySession.value?.status === 'pending');
  });

  const isViewing = computed(() => {
    return currentViewingSession.value !== null;
  });

  const getActiveSessionByUser = computed(() => {
    return (userId: number) => {
      return activeSessions.value.find(s => s.initiator_id === userId);
    };
  });

  const getActiveSessionByConversation = computed(() => {
    return (conversationId: number) => {
      return activeSessions.value.find(s => s.conversation_id === conversationId);
    };
  });

  // Actions
  async function createSession(type: 'screen_share' | 'voice_call' | 'video_call', conversationId: number) {
    try {
      const response = await fetch('/api/realtime/sessions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({ type, conversation_id: conversationId })
      });

      const data = await response.json();
      if (data.session) {
        mySession.value = data.session;
        activeSessions.value.push(data.session);
      }
      return data.session;
    } catch (error) {
      console.error('创建会话失败:', error);
      throw error;
    }
  }

  async function fetchActiveSessions(initiatorId?: number) {
    try {
      const url = initiatorId 
        ? `/api/realtime/sessions?initiator_id=${initiatorId}`
        : '/api/realtime/sessions';
      
      const response = await fetch(url, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });

      const data = await response.json();
      if (data.sessions) {
        activeSessions.value = data.sessions;
      }
      return data.sessions;
    } catch (error) {
      console.error('获取活跃会话失败:', error);
      return [];
    }
  }

  async function fetchSession(sessionId: string) {
    try {
      const response = await fetch(`/api/realtime/sessions/${sessionId}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });

      const data = await response.json();
      return data.session;
    } catch (error) {
      console.error('获取会话详情失败:', error);
      return null;
    }
  }

  async function requestJoin(sessionId: string) {
    try {
      const response = await fetch(`/api/realtime/sessions/${sessionId}/participants`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });

      const data = await response.json();
      return data.participant;
    } catch (error) {
      console.error('申请加入失败:', error);
      throw error;
    }
  }

  async function approveJoin(sessionId: string, userId: number) {
    try {
      const response = await fetch(`/api/realtime/sessions/${sessionId}/participants/${userId}`, {
        method: 'PATCH',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });

      const data = await response.json();
      
      // 移除待处理请求
      pendingRequests.value = pendingRequests.value.filter(
        r => !(r.session_id === sessionId && r.user_id === userId)
      );
      
      return data.participant;
    } catch (error) {
      console.error('批准加入失败:', error);
      throw error;
    }
  }

  async function rejectJoin(sessionId: string, userId: number) {
    try {
      const response = await fetch(`/api/realtime/sessions/${sessionId}/participants/${userId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });

      // 移除待处理请求
      pendingRequests.value = pendingRequests.value.filter(
        r => !(r.session_id === sessionId && r.user_id === userId)
      );
    } catch (error) {
      console.error('拒绝加入失败:', error);
    }
  }

  async function leaveSession(sessionId: string) {
    try {
      await fetch(`/api/realtime/sessions/${sessionId}/participants`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });

      currentViewingSession.value = null;
    } catch (error) {
      console.error('离开会话失败:', error);
    }
  }

  async function endSession(sessionId: string) {
    try {
      await fetch(`/api/realtime/sessions/${sessionId}`, {
        method: 'PATCH',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });

      mySession.value = null;
      activeSessions.value = activeSessions.value.filter(s => s.id !== sessionId);
    } catch (error) {
      console.error('结束会话失败:', error);
    }
  }

  function addPendingRequest(request: JoinRequest) {
    const exists = pendingRequests.value.some(
      r => r.session_id === request.session_id && r.user_id === request.user_id
    );
    if (!exists) {
      pendingRequests.value.push(request);
    }
  }

  function updateSession(session: RealtimeSession) {
    const index = activeSessions.value.findIndex(s => s.id === session.id);
    if (index >= 0) {
      activeSessions.value[index] = session;
    }
    if (mySession.value?.id === session.id) {
      mySession.value = session;
    }
  }

  function removeSession(sessionId: string) {
    activeSessions.value = activeSessions.value.filter(s => s.id !== sessionId);
    if (mySession.value?.id === sessionId) {
      mySession.value = null;
    }
    if (currentViewingSession.value?.id === sessionId) {
      currentViewingSession.value = null;
    }
  }

  function setCurrentViewingSession(session: RealtimeSession | null) {
    currentViewingSession.value = session;
  }

  return {
    // State
    activeSessions,
    mySession,
    pendingRequests,
    currentViewingSession,
    
    // Getters
    isSharing,
    isViewing,
    getActiveSessionByUser,
    getActiveSessionByConversation,
    
    // Actions
    createSession,
    fetchActiveSessions,
    fetchSession,
    requestJoin,
    approveJoin,
    rejectJoin,
    leaveSession,
    endSession,
    addPendingRequest,
    updateSession,
    removeSession,
    setCurrentViewingSession
  };
});
```

---

## 任务 6：前端 WebRTC 连接管理

**文件：**
- 创建：`qim-client/src/utils/realtimeConnection.ts`

### 步骤 6.1：创建 WebRTC 连接管理器

- [ ] **创建 `qim-client/src/utils/realtimeConnection.ts`**

```typescript
import { logger } from './logger';

interface ConnectionInfo {
  peerConnection: RTCPeerConnection;
  viewerId: number;
  sessionId: string;
}

export class RealtimeConnectionManager {
  private connections: Map<string, ConnectionInfo> = new Map();
  private localStream: MediaStream | null = null;
  private sessionId: string | null = null;
  private onViewerJoined?: (viewerId: number) => void;
  private onViewerLeft?: (viewerId: number) => void;

  getPeerConfig(): RTCConfiguration {
    return {
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' },
        { urls: 'stun:stun1.l.google.com:19302' },
        { urls: 'stun:stun2.l.google.com:19302' }
      ],
      iceCandidatePoolSize: 10
    };
  }

  setLocalStream(stream: MediaStream) {
    this.localStream = stream;
  }

  setSessionId(sessionId: string) {
    this.sessionId = sessionId;
  }

  setCallbacks(callbacks: {
    onViewerJoined?: (viewerId: number) => void;
    onViewerLeft?: (viewerId: number) => void;
  }) {
    this.onViewerJoined = callbacks.onViewerJoined;
    this.onViewerLeft = callbacks.onViewerLeft;
  }

  async createConnectionForViewer(viewerId: number): Promise<RTCPeerConnection | null> {
    if (!this.localStream || !this.sessionId) {
      logger.error('本地流或会话ID未设置');
      return null;
    }

    const key = `${this.sessionId}:${viewerId}`;
    
    // 如果已存在连接，先关闭
    if (this.connections.has(key)) {
      this.closeConnection(viewerId);
    }

    const peerConnection = new RTCPeerConnection(this.getPeerConfig());

    // 添加本地流
    this.localStream.getTracks().forEach(track => {
      peerConnection.addTrack(track, this.localStream!);
    });

    // 设置事件处理
    peerConnection.onicecandidate = (event) => {
      if (event.candidate) {
        this.sendSignalingMessage('realtime:webrtc:ice', viewerId, {
          signal: event.candidate
        });
      }
    };

    peerConnection.onconnectionstatechange = () => {
      const state = peerConnection.connectionState;
      logger.log(`与观看者 ${viewerId} 的连接状态:`, state);

      if (state === 'connected') {
        if (this.onViewerJoined) {
          this.onViewerJoined(viewerId);
        }
      } else if (state === 'disconnected' || state === 'failed') {
        this.closeConnection(viewerId);
        if (this.onViewerLeft) {
          this.onViewerLeft(viewerId);
        }
      }
    };

    // 创建 offer
    const offer = await peerConnection.createOffer();
    await peerConnection.setLocalDescription(offer);

    // 发送 offer
    this.sendSignalingMessage('realtime:webrtc:offer', viewerId, {
      signal: offer
    });

    // 保存连接
    this.connections.set(key, {
      peerConnection,
      viewerId,
      sessionId: this.sessionId
    });

    logger.log(`为观看者 ${viewerId} 创建连接，当前连接数:`, this.connections.size);

    return peerConnection;
  }

  async handleAnswer(viewerId: number, answer: RTCSessionDescriptionInit) {
    const key = `${this.sessionId}:${viewerId}`;
    const conn = this.connections.get(key);
    
    if (conn) {
      await conn.peerConnection.setRemoteDescription(new RTCSessionDescription(answer));
      logger.log(`设置观看者 ${viewerId} 的远程描述成功`);
    }
  }

  async handleIceCandidate(viewerId: number, candidate: RTCIceCandidateInit) {
    const key = `${this.sessionId}:${viewerId}`;
    const conn = this.connections.get(key);
    
    if (conn && conn.peerConnection.remoteDescription) {
      const iceCandidate = new RTCIceCandidate({
        candidate: candidate.candidate,
        sdpMid: candidate.sdpMid || '',
        sdpMLineIndex: candidate.sdpMLineIndex || 0
      });
      await conn.peerConnection.addIceCandidate(iceCandidate);
    }
  }

  closeConnection(viewerId: number) {
    const key = `${this.sessionId}:${viewerId}`;
    const conn = this.connections.get(key);
    
    if (conn) {
      conn.peerConnection.close();
      this.connections.delete(key);
      logger.log(`关闭观看者 ${viewerId} 的连接，剩余连接数:`, this.connections.size);
    }
  }

  closeAllConnections() {
    this.connections.forEach((conn) => {
      conn.peerConnection.close();
    });
    this.connections.clear();
    logger.log('关闭所有连接');
  }

  getViewerIds(): number[] {
    const ids: number[] = [];
    this.connections.forEach((conn) => {
      ids.push(conn.viewerId);
    });
    return ids;
  }

  getConnectionCount(): number {
    return this.connections.size;
  }

  private sendSignalingMessage(type: string, targetUserId: number, data: Record<string, unknown>) {
    const message = {
      type,
      data: {
        session_id: this.sessionId,
        target_user_id: targetUserId,
        ...data
      }
    };

    if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
      window.ws.send(JSON.stringify(message));
    } else if (window.electron && window.electron.websocket) {
      window.electron.websocket.send(message);
    } else {
      logger.error('WebSocket 连接不可用');
    }
  }
}

// 观看者端连接管理
export class RealtimeViewerConnection {
  private peerConnection: RTCPeerConnection | null = null;
  private sessionId: string | null = null;
  private initiatorId: number | null = null;
  private onStreamReceived?: (stream: MediaStream) => void;

  getPeerConfig(): RTCConfiguration {
    return {
      iceServers: [
        { urls: 'stun:stun.l.google.com:19302' },
        { urls: 'stun:stun1.l.google.com:19302' },
        { urls: 'stun:stun2.l.google.com:19302' }
      ],
      iceCandidatePoolSize: 10
    };
  }

  setCallbacks(callbacks: {
    onStreamReceived?: (stream: MediaStream) => void;
  }) {
    this.onStreamReceived = callbacks.onStreamReceived;
  }

  async handleOffer(sessionId: string, initiatorId: number, offer: RTCSessionDescriptionInit) {
    this.sessionId = sessionId;
    this.initiatorId = initiatorId;

    this.peerConnection = new RTCPeerConnection(this.getPeerConfig());

    // 处理远程流
    this.peerConnection.ontrack = (event) => {
      if (event.streams && event.streams.length > 0) {
        const stream = event.streams[0];
        if (this.onStreamReceived) {
          this.onStreamReceived(stream);
        }
      }
    };

    // ICE candidate
    this.peerConnection.onicecandidate = (event) => {
      if (event.candidate) {
        this.sendSignalingMessage('realtime:webrtc:ice', initiatorId, {
          signal: event.candidate
        });
      }
    };

    // 设置远程描述
    await this.peerConnection.setRemoteDescription(new RTCSessionDescription(offer));

    // 创建 answer
    const answer = await this.peerConnection.createAnswer();
    await this.peerConnection.setLocalDescription(answer);

    // 发送 answer
    this.sendSignalingMessage('realtime:webrtc:answer', initiatorId, {
      signal: answer
    });

    logger.log('已处理 offer 并发送 answer');
  }

  async handleIceCandidate(candidate: RTCIceCandidateInit) {
    if (this.peerConnection && this.peerConnection.remoteDescription) {
      const iceCandidate = new RTCIceCandidate({
        candidate: candidate.candidate,
        sdpMid: candidate.sdpMid || '',
        sdpMLineIndex: candidate.sdpMLineIndex || 0
      });
      await this.peerConnection.addIceCandidate(iceCandidate);
    }
  }

  close() {
    if (this.peerConnection) {
      this.peerConnection.close();
      this.peerConnection = null;
    }
    logger.log('观看者连接已关闭');
  }

  private sendSignalingMessage(type: string, targetUserId: number, data: Record<string, unknown>) {
    const message = {
      type,
      data: {
        session_id: this.sessionId,
        target_user_id: targetUserId,
        ...data
      }
    };

    if (typeof window !== 'undefined' && window.ws && window.ws.readyState === WebSocket.OPEN) {
      window.ws.send(JSON.stringify(message));
    } else if (window.electron && window.electron.websocket) {
      window.electron.websocket.send(message);
    }
  }
}

// 导出单例
export const realtimeConnectionManager = new RealtimeConnectionManager();
export const realtimeViewerConnection = new RealtimeViewerConnection();
```

---

## 任务 7：前端组件 - 会话卡片

**文件：**
- 创建：`qim-client/src/components/realtime/RealtimeSessionCard.vue`

### 步骤 7.1：创建会话卡片组件

- [ ] **创建 `qim-client/src/components/realtime/RealtimeSessionCard.vue`**

```vue
<template>
  <div class="realtime-session-card" :class="{ 'is-active': isActive }">
    <div class="session-header">
      <div class="session-icon">
        <i :class="iconClass"></i>
      </div>
      <div class="session-info">
        <div class="session-title">{{ title }}</div>
        <div class="session-meta">
          <span v-if="session.status === 'active'" class="status-badge active">
            <span class="pulse-dot"></span>
            进行中
          </span>
          <span v-else-if="session.status === 'pending'" class="status-badge pending">
            等待中
          </span>
          <span v-else-if="session.status === 'ended'" class="status-badge ended">
            已结束
          </span>
        </div>
      </div>
    </div>

    <div v-if="showViewers && viewers.length > 0" class="viewers-section">
      <div class="viewers-label">当前观看者:</div>
      <div class="viewers-list">
        <span v-for="viewer in viewers" :key="viewer.user_id" class="viewer-tag">
          {{ viewer.user?.nickname || `用户${viewer.user_id}` }}
        </span>
      </div>
    </div>

    <div class="session-actions">
      <button 
        v-if="canJoin" 
        class="btn-primary" 
        @click="handleJoin"
        :disabled="isJoining"
      >
        {{ isJoining ? '申请中...' : '加入观看' }}
      </button>
      <button 
        v-if="canLeave" 
        class="btn-secondary" 
        @click="handleLeave"
      >
        离开
      </button>
      <button 
        v-if="canEnd" 
        class="btn-danger" 
        @click="handleEnd"
      >
        结束共享
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { RealtimeSession, RealtimeParticipant } from '../../types/realtime';
import { useRealtimeStore } from '../../stores/realtime';

const props = defineProps<{
  session: RealtimeSession;
  currentUserId: number;
  showViewers?: boolean;
}>();

const emit = defineEmits<{
  join: [sessionId: string];
  leave: [sessionId: string];
  end: [sessionId: string];
}>();

const realtimeStore = useRealtimeStore();
const isJoining = ref(false);

const isActive = computed(() => props.session.status === 'active');

const iconClass = computed(() => {
  switch (props.session.type) {
    case 'screen_share':
      return 'fas fa-desktop';
    case 'video_call':
      return 'fas fa-video';
    case 'voice_call':
      return 'fas fa-phone';
    default:
      return 'fas fa-broadcast-tower';
  }
});

const title = computed(() => {
  const initiatorName = props.session.initiator?.nickname || '用户';
  switch (props.session.type) {
    case 'screen_share':
      return `${initiatorName} 正在共享屏幕`;
    case 'video_call':
      return `${initiatorName} 发起了视频通话`;
    case 'voice_call':
      return `${initiatorName} 发起了语音通话`;
    default:
      return `${initiatorName} 发起了实时会话`;
  }
});

const viewers = computed(() => {
  return props.session.participants?.filter(
    p => p.role === 'viewer' && p.status === 'joined'
  ) || [];
});

const isInitiator = computed(() => props.session.initiator_id === props.currentUserId);

const isParticipant = computed(() => {
  return props.session.participants?.some(
    p => p.user_id === props.currentUserId && p.status === 'joined'
  );
});

const canJoin = computed(() => {
  return !isInitiator.value && !isParticipant.value && props.session.status !== 'ended';
});

const canLeave = computed(() => {
  return isParticipant.value;
});

const canEnd = computed(() => {
  return isInitiator.value && props.session.status !== 'ended';
});

async function handleJoin() {
  isJoining.value = true;
  try {
    emit('join', props.session.id);
  } finally {
    isJoining.value = false;
  }
}

function handleLeave() {
  emit('leave', props.session.id);
}

function handleEnd() {
  emit('end', props.session.id);
}
</script>

<style scoped>
.realtime-session-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 12px;
  padding: 16px;
  color: white;
  margin: 12px 0;
}

.realtime-session-card.is-active {
  box-shadow: 0 4px 20px rgba(102, 126, 234, 0.4);
}

.session-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.session-icon {
  width: 40px;
  height: 40px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}

.session-info {
  flex: 1;
}

.session-title {
  font-weight: 600;
  font-size: 15px;
  margin-bottom: 4px;
}

.session-meta {
  font-size: 12px;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 500;
}

.status-badge.active {
  background: rgba(34, 197, 94, 0.2);
}

.status-badge.pending {
  background: rgba(251, 191, 36, 0.2);
}

.status-badge.ended {
  background: rgba(156, 163, 175, 0.2);
}

.pulse-dot {
  width: 8px;
  height: 8px;
  background: #22c55e;
  border-radius: 50%;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.viewers-section {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.2);
}

.viewers-label {
  font-size: 12px;
  opacity: 0.8;
  margin-bottom: 6px;
}

.viewers-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.viewer-tag {
  background: rgba(255, 255, 255, 0.15);
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
}

.session-actions {
  margin-top: 12px;
  display: flex;
  gap: 8px;
}

.btn-primary,
.btn-secondary,
.btn-danger {
  padding: 8px 16px;
  border-radius: 8px;
  border: none;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: white;
  color: #667eea;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.2);
  color: white;
}

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.3);
}

.btn-danger {
  background: rgba(239, 68, 68, 0.8);
  color: white;
}

.btn-danger:hover {
  background: rgba(220, 38, 38, 1);
}
</style>
```

---

## 任务 8：前端组件 - 加入请求弹窗

**文件：**
- 创建：`qim-client/src/components/realtime/JoinRequestModal.vue`

### 步骤 8.1：创建加入请求弹窗组件

- [ ] **创建 `qim-client/src/components/realtime/JoinRequestModal.vue`**

```vue
<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible && currentRequest" class="modal-overlay" @click.self="handleReject">
        <div class="modal-content">
          <div class="modal-header">
            <i class="fas fa-user-plus"></i>
            <span>加入请求</span>
          </div>
          
          <div class="modal-body">
            <div class="user-info">
              <img 
                v-if="currentRequest.user?.avatar" 
                :src="currentRequest.user.avatar" 
                class="avatar"
              />
              <div v-else class="avatar-placeholder">
                <i class="fas fa-user"></i>
              </div>
              <div class="user-details">
                <div class="user-name">
                  {{ currentRequest.user?.nickname || `用户 ${currentRequest.user_id}` }}
                </div>
                <div class="request-info">想加入观看你的屏幕共享</div>
              </div>
            </div>
          </div>
          
          <div class="modal-actions">
            <button class="btn-reject" @click="handleReject">
              <i class="fas fa-times"></i>
              拒绝
            </button>
            <button class="btn-approve" @click="handleApprove">
              <i class="fas fa-check"></i>
              同意
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { JoinRequest } from '../../types/realtime';

const props = defineProps<{
  visible: boolean;
  requests: JoinRequest[];
}>();

const emit = defineEmits<{
  approve: [sessionId: string, userId: number];
  reject: [sessionId: string, userId: number];
}>();

const currentRequest = computed(() => {
  return props.requests.length > 0 ? props.requests[0] : null;
});

function handleApprove() {
  if (currentRequest.value) {
    emit('approve', currentRequest.value.session_id, currentRequest.value.user_id);
  }
}

function handleReject() {
  if (currentRequest.value) {
    emit('reject', currentRequest.value.session_id, currentRequest.value.user_id);
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
}

.modal-content {
  background: white;
  border-radius: 16px;
  width: 320px;
  overflow: hidden;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

.modal-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 600;
}

.modal-body {
  padding: 20px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.avatar,
.avatar-placeholder {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  object-fit: cover;
}

.avatar-placeholder {
  background: #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #9ca3af;
}

.user-details {
  flex: 1;
}

.user-name {
  font-weight: 600;
  font-size: 15px;
  color: #1f2937;
}

.request-info {
  font-size: 13px;
  color: #6b7280;
  margin-top: 2px;
}

.modal-actions {
  display: flex;
  border-top: 1px solid #e5e7eb;
}

.btn-reject,
.btn-approve {
  flex: 1;
  padding: 14px;
  border: none;
  background: none;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: background 0.2s;
}

.btn-reject {
  color: #ef4444;
  border-right: 1px solid #e5e7eb;
}

.btn-reject:hover {
  background: #fef2f2;
}

.btn-approve {
  color: #22c55e;
}

.btn-approve:hover {
  background: #f0fdf4;
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-active .modal-content,
.modal-leave-active .modal-content {
  transition: transform 0.2s ease;
}

.modal-enter-from .modal-content,
.modal-leave-to .modal-content {
  transform: scale(0.95);
}
</style>
```

---

## 任务 9：集成到 ChatWindow

**文件：**
- 修改：`qim-client/src/components/chat/ChatWindow.vue`

### 步骤 9.1：导入和初始化 Store

- [ ] **在 ChatWindow.vue 的 script 部分添加导入**

```typescript
import { useRealtimeStore } from '../../stores/realtime';
import { realtimeConnectionManager, realtimeViewerConnection } from '../../utils/realtimeConnection';
import JoinRequestModal from '../realtime/JoinRequestModal.vue';
import RealtimeSessionCard from '../realtime/RealtimeSessionCard.vue';
```

### 步骤 9.2：添加 WebSocket 事件监听

- [ ] **在 setupWebsocketHandlers 函数中添加事件监听**

```typescript
// 在 screenShareMessageTypes 数组后添加
const realtimeMessageTypes = [
  'realtime:session:created',
  'realtime:join:requested',
  'realtime:join:approved',
  'realtime:join:rejected',
  'realtime:participant:left',
  'realtime:session:ended',
  'realtime:webrtc:offer',
  'realtime:webrtc:answer',
  'realtime:webrtc:ice'
];

// 在 handlerMap 中添加处理
realtimeMessageTypes.forEach(type => {
  handlerMap[type] = (data: any) => handleRealtimeMessage(type, data);
});
```

### 步骤 9.3：添加消息处理函数

- [ ] **添加 handleRealtimeMessage 函数**

```typescript
const handleRealtimeMessage = (type: string, data: any) => {
  const realtimeStore = useRealtimeStore();

  switch (type) {
    case 'realtime:session:created':
      // 有新的共享会话创建
      realtimeStore.fetchActiveSessions();
      break;

    case 'realtime:join:requested':
      // 有人申请加入
      realtimeStore.addPendingRequest({
        session_id: data.session_id,
        user_id: data.user_id,
        timestamp: data.timestamp
      });
      break;

    case 'realtime:join:approved':
      // 加入请求被批准
      if (realtimeStore.mySession?.id === data.session_id) {
        // 发起者为观看者创建连接
        realtimeConnectionManager.createConnectionForViewer(data.viewer_id);
      }
      break;

    case 'realtime:join:rejected':
      // 加入请求被拒绝
      $message.warning('共享观看请求被拒绝');
      break;

    case 'realtime:participant:left':
      // 参与者离开
      realtimeConnectionManager.closeConnection(data.user_id);
      break;

    case 'realtime:session:ended':
      // 会话结束
      realtimeStore.removeSession(data.session_id);
      realtimeViewerConnection.close();
      $message.info('屏幕共享已结束');
      break;

    case 'realtime:webrtc:offer':
      // 收到 WebRTC offer（观看者端）
      realtimeViewerConnection.handleOffer(
        data.session_id,
        data.from_user_id,
        data.signal
      );
      break;

    case 'realtime:webrtc:answer':
      // 收到 WebRTC answer（发起者端）
      realtimeConnectionManager.handleAnswer(
        data.from_user_id,
        data.signal
      );
      break;

    case 'realtime:webrtc:ice':
      // 收到 ICE candidate
      if (realtimeStore.mySession) {
        // 发起者端
        realtimeConnectionManager.handleIceCandidate(
          data.from_user_id,
          data.signal
        );
      } else {
        // 观看者端
        realtimeViewerConnection.handleIceCandidate(data.signal);
      }
      break;
  }
};
```

---

## 任务 10：修改屏幕共享发起流程

**文件：**
- 修改：`qim-client/src/components/shared/ScreenShare.vue`

### 步骤 10.1：集成新的实时通信架构

- [ ] **修改 startSharing 函数**

```typescript
const startSharing = async () => {
  if (!selectedSource.value) return;

  showSourcePicker.value = false;
  screenShareName.value = selectedSource.value.name;
  isSharing.value = true;
  isInitiator.value = true;

  const realtimeStore = useRealtimeStore();
  
  // 1. 创建实时会话
  const session = await realtimeStore.createSession('screen_share', props.conversationId);
  
  if (!session) {
    $message.error('创建共享会话失败');
    isSharing.value = false;
    isInitiator.value = false;
    return;
  }

  // 2. 获取屏幕流
  try {
    const stream = await navigator.mediaDevices.getDisplayMedia({
      video: true,
      audio: true
    });

    screenStream = stream;
    
    // 3. 设置连接管理器
    realtimeConnectionManager.setLocalStream(stream);
    realtimeConnectionManager.setSessionId(session.id);
    realtimeConnectionManager.setCallbacks({
      onViewerJoined: (viewerId) => {
        logger.log(`观看者 ${viewerId} 已加入`);
        // 更新观看者列表
        fetchSessionDetails();
      },
      onViewerLeft: (viewerId) => {
        logger.log(`观看者 ${viewerId} 已离开`);
        fetchSessionDetails();
      }
    });

    // 4. 广播会话创建事件
    sendWebSocketMessage('realtime:session:create', {
      session_id: session.id,
      type: 'screen_share',
      conversation_id: props.conversationId
    });

    // 5. 更新会话状态为 active
    await fetch(`/api/realtime/sessions/${session.id}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({ status: 'active' })
    });

    logger.log('屏幕共享已开始');
  } catch (error) {
    console.error('开始屏幕共享失败:', error);
    isSharing.value = false;
    isInitiator.value = false;
    realtimeStore.endSession(session.id);
  }
};
```

---

## 执行选项

计划已完成并保存到 `docs/superpowers/plans/2025-04-29-unified-realtime-communication.md`。

**两种执行方式：**

**1. 子代理驱动（推荐）** - 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** - 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点供审查

**选哪种方式？**
