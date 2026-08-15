package ws

import (
	"encoding/json"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

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
		delete(d.timers, userID)
	}

	d.timers[userID] = time.AfterFunc(d.delay, func() {
		fn()
		d.mu.Lock()
		delete(d.timers, userID)
		d.mu.Unlock()
	})
}

// UpdateUserStatus 更新用户状态并广播

func (h *Hub) UpdateUserStatus(userID uint, username, status string) {
	db := h.db
	now := time.Now()

	result := db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"status":      status,
		"last_online": now,
	})
	if result.Error != nil {
		logger.WithModule("WS").Error("更新用户状态失败", "userID", userID, "username", username, "error", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		logger.WithModule("WS").Info("用户状态变更", "userID", userID, "username", username, "status", status)
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
		logger.WithModule("WS").Error("获取用户信息失败", "userID", userID, "error", err)
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

	logger.WithModule("WS").Debug("已向订阅者广播用户状态变更", "userID", userID, "status", status)
}

// BroadcastToConversationMembers 向用户所在会话的成员广播状态变更

func (h *Hub) BroadcastToConversationMembers(userID uint, message []byte) {
	db := h.db

	var members []model.ConversationMember
	if err := db.Select("conversation_id").
		Where("user_id = ?", userID).
		Group("conversation_id").
		Find(&members).Error; err != nil {
		logger.WithModule("WS").Error("获取用户会话失败", "userID", userID, "error", err)
		return
	}

	for _, member := range members {
		h.SendToConversationAsync(member.ConversationID, userID, message)
	}
}

// SubscribeUserStatus 订阅用户状态变更

func (h *Hub) SubscribeUserStatus(subscriberID, targetUserID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.userSubscribers.Load(targetUserID); ok {
		subscribers := existing.([]uint)
		for _, sid := range subscribers {
			if sid == subscriberID {
				return // 已订阅，跳过
			}
		}
		subscribers = append(subscribers, subscriberID)
		h.userSubscribers.Store(targetUserID, subscribers)
	} else {
		h.userSubscribers.Store(targetUserID, []uint{subscriberID})
	}
}

// UnsubscribeUserStatus 取消订阅用户状态变更

func (h *Hub) UnsubscribeUserStatus(subscriberID, targetUserID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()

	existing, ok := h.userSubscribers.Load(targetUserID)
	if !ok {
		return
	}
	subscribers := existing.([]uint)
	for i, sid := range subscribers {
		if sid == subscriberID {
			subscribers = append(subscribers[:i], subscribers[i+1:]...)
			if len(subscribers) == 0 {
				h.userSubscribers.Delete(targetUserID)
			} else {
				h.userSubscribers.Store(targetUserID, subscribers)
			}
			break
		}
	}
}

// CleanupUserSubscriptions 清理用户的所有订阅

func (h *Hub) CleanupUserSubscriptions(userID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var toDelete []uint
	var toUpdate []struct {
		key         uint
		subscribers []uint
	}

	h.userSubscribers.Range(func(key, value interface{}) bool {
		subscribers := value.([]uint)
		for i, sid := range subscribers {
			if sid == userID {
				newSubs := append(subscribers[:i], subscribers[i+1:]...)
				if len(newSubs) == 0 {
					toDelete = append(toDelete, key.(uint))
				} else {
					toUpdate = append(toUpdate, struct {
						key         uint
						subscribers []uint
					}{key.(uint), newSubs})
				}
				break
			}
		}
		return true
	})

	for _, k := range toDelete {
		h.userSubscribers.Delete(k)
	}
	for _, u := range toUpdate {
		h.userSubscribers.Store(u.key, u.subscribers)
	}
}

// handleSubscribeUserStatus 处理订阅用户状态请求

func handleSubscribeUserStatus(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("WS").Warn("订阅用户状态数据格式错误")
		return
	}

	targetUserIDFloat, ok := msgData["user_id"].(float64)
	if !ok {
		logger.WithModule("WS").Warn("订阅用户状态缺少user_id")
		return
	}

	targetUserID := uint(targetUserIDFloat)
	logger.WithModule("WS").Info("用户订阅状态变更", "subscriberID", c.userID, "targetUserID", targetUserID)

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
		logger.WithModule("WS").Warn("取消订阅用户状态数据格式错误")
		return
	}

	targetUserIDFloat, ok := msgData["user_id"].(float64)
	if !ok {
		logger.WithModule("WS").Warn("取消订阅用户状态缺少user_id")
		return
	}

	targetUserID := uint(targetUserIDFloat)
	logger.WithModule("WS").Info("用户取消订阅状态变更", "subscriberID", c.userID, "targetUserID", targetUserID)

	c.hub.UnsubscribeUserStatus(c.userID, targetUserID)
}

// versionStatsKey 生成版本统计的 map key
