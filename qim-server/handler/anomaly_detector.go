package handler

import (
	"encoding/json"
	"log"
	"math"
	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"
	"strings"
	"sync"
	"time"
)

// AnomalyDetector 异常检测器
type AnomalyDetector struct {
	mu                  sync.RWMutex
	messageCounts       map[uint]int         // conversationID -> 最近窗口消息数
	messageTimestamps   map[uint][]time.Time // conversationID -> 消息时间戳
	windowSize          time.Duration
	thresholdMultiplier float64
	avgCounts           map[uint]float64 // conversationID -> 历史平均消息数
}

// NewAnomalyDetector 创建异常检测器
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		messageCounts:       make(map[uint]int),
		messageTimestamps:   make(map[uint][]time.Time),
		windowSize:          5 * time.Minute,
		thresholdMultiplier: 3.0,
		avgCounts:           make(map[uint]float64),
	}
}

// RecordMessage 记录消息用于检测
func (d *AnomalyDetector) RecordMessage(conversationID uint) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	// 记录时间戳
	d.messageTimestamps[conversationID] = append(
		d.messageTimestamps[conversationID], now,
	)

	// 清理旧时间戳
	cutoff := now.Add(-d.windowSize)
	timestamps := d.messageTimestamps[conversationID]
	validTimestamps := make([]time.Time, 0, len(timestamps))
	for _, t := range timestamps {
		if t.After(cutoff) {
			validTimestamps = append(validTimestamps, t)
		}
	}
	d.messageTimestamps[conversationID] = validTimestamps
	d.messageCounts[conversationID] = len(validTimestamps)
}

// CheckMessageAnomaly 检查消息异常
func (d *AnomalyDetector) CheckMessageAnomaly(conversationID uint) *AnomalyAlert {
	d.mu.RLock()
	defer d.mu.RUnlock()

	currentCount := d.messageCounts[conversationID]
	avgCount := d.avgCounts[conversationID]

	// 如果历史数据不足，跳过检测
	if avgCount == 0 {
		return nil
	}

	threshold := avgCount * d.thresholdMultiplier

	if float64(currentCount) > threshold {
		return &AnomalyAlert{
			Type:           "message_spike",
			Level:          "warning",
			ConversationID: conversationID,
			Message:        "检测到异常消息量增长，可能存在突发事件或刷屏行为",
			Details: map[string]interface{}{
				"current_count": currentCount,
				"avg_count":     int(avgCount),
				"threshold":     int(threshold),
				"multiplier":    float64(currentCount) / avgCount,
			},
		}
	}

	return nil
}

// UpdateBaseline 更新基线数据
func (d *AnomalyDetector) UpdateBaseline() {
	d.mu.Lock()
	defer d.mu.Unlock()

	db := database.GetDB()

	var conversations []model.Conversation
	db.Where("type = ?", "group").Find(&conversations)

	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	tenMinutesAgo := time.Now().Add(-10 * time.Minute)

	for _, conv := range conversations {
		// 统计前一个 5 分钟窗口的消息数（作为基线）
		var count10 int64
		db.Model(&model.Message{}).
			Where("conversation_id = ? AND created_at >= ? AND created_at < ?",
				conv.ID, tenMinutesAgo, fiveMinutesAgo).
			Count(&count10)

		// 统计最近 5 分钟的消息数
		var count5 int64
		db.Model(&model.Message{}).
			Where("conversation_id = ? AND created_at >= ?",
				conv.ID, fiveMinutesAgo).
			Count(&count5)

		// 平滑更新基线
		if d.avgCounts[conv.ID] == 0 {
			d.avgCounts[conv.ID] = float64(count10)
		} else {
			d.avgCounts[conv.ID] = d.avgCounts[conv.ID]*0.7 + float64(count10)*0.3
		}

		d.messageCounts[conv.ID] = int(count5)
	}
}

// AnomalyAlert 异常告警
type AnomalyAlert struct {
	Type           string                 `json:"type"`
	Level          string                 `json:"level"`
	ConversationID uint                   `json:"conversation_id,omitempty"`
	Message        string                 `json:"message"`
	Details        map[string]interface{} `json:"details,omitempty"`
}

// CheckSensitiveContent 检测敏感内容爆发
func (d *AnomalyDetector) CheckSensitiveContent(content string) *AnomalyAlert {
	// 简单的敏感词检测（实际应该用更复杂的 NLP）
	sensitivePatterns := []string{
		"暴力", "色情", "赌博", "诈骗", "传销",
		"violence", "porn", "gambling", "fraud",
	}

	contentLower := strings.ToLower(content)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(contentLower, pattern) {
			return &AnomalyAlert{
				Type:    "sensitive_content",
				Level:   "high",
				Message: "检测到敏感内容",
				Details: map[string]interface{}{
					"pattern": pattern,
					"preview": previewText(content, 50),
				},
			}
		}
	}

	return nil
}

// CheckMessageFrequency 检测单用户消息频率异常
func (d *AnomalyDetector) CheckMessageFrequency(userID uint, conversationID uint) *AnomalyAlert {
	db := database.GetDB()

	oneMinuteAgo := time.Now().Add(-1 * time.Minute)

	var count int64
	db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id = ? AND created_at >= ?",
			conversationID, userID, oneMinuteAgo).
		Count(&count)

	if count > 20 { // 1 分钟内超过 20 条消息
		return &AnomalyAlert{
			Type:    "user_message_flood",
			Level:   "warning",
			Message: "检测到用户短时间内发送大量消息，可能存在刷屏行为",
			Details: map[string]interface{}{
				"user_id": userID,
				"count":   count,
				"window":  "1 minute",
			},
		}
	}

	return nil
}

// CheckInactiveGroup 检测不活跃群组突然活跃
func (d *AnomalyDetector) CheckInactiveGroup(conversationID uint) *AnomalyAlert {
	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, conversationID).Error; err != nil {
		return nil
	}

	if conv.LastMessageAt == nil || conv.LastMessageAt.IsZero() {
		return nil
	}

	daysSinceLastMessage := time.Since(*conv.LastMessageAt).Hours() / 24

	if daysSinceLastMessage > 7 {
		// 获取群聊信息
		var group model.Group
		db.Where("conversation_id = ?", conversationID).First(&group)

		// 超过 7 天没有消息的群突然活跃
		return &AnomalyAlert{
			Type:    "inactive_group_activated",
			Level:   "info",
			Message: "沉寂已久的群组突然活跃",
			Details: map[string]interface{}{
				"group_name":      group.Name,
				"days_inactive":   int(daysSinceLastMessage),
				"conversation_id": conversationID,
			},
		}
	}

	return nil
}

// SendAlert 发送告警通知
func (d *AnomalyDetector) SendAlert(adminID uint, alert *AnomalyAlert) {
	db := database.GetDB()

	notification := model.Notification{
		UserID:  adminID,
		Type:    "anomaly_alert",
		Title:   "⚠️ 异常检测告警",
		Content: alert.Message,
		Read:    false,
	}
	db.Create(&notification)

	// WebSocket 推送
	wsMsg := ws.WSMessage{
		Type: "anomaly_alert",
		Data: alert,
	}
	jsonMsg, _ := json.Marshal(wsMsg)
	ws.GlobalHub.SendToUser(adminID, jsonMsg)

	log.Printf("[AnomalyDetector] 告警已发送: type=%s, level=%s, message=%s",
		alert.Type, alert.Level, alert.Message)
}

func previewText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// StartAnomalyDetection 启动异常检测定时任务
func StartAnomalyDetection(detector *AnomalyDetector) {
	go func() {
		// 每分钟更新基线并检测异常
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			detector.UpdateBaseline()
			log.Printf("[AnomalyDetector] 基线已更新，监控 %d 个群组", len(detector.avgCounts))
		}
	}()
}

// CalculateMessageVariance 计算消息方差（用于判断异常波动）
func (d *AnomalyDetector) CalculateMessageVariance(conversationID uint) float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	avg := d.avgCounts[conversationID]
	current := float64(d.messageCounts[conversationID])

	if avg == 0 {
		return 0
	}

	return math.Abs(current-avg) / avg
}

// GetDetectionStats 获取检测统计信息
func (d *AnomalyDetector) GetDetectionStats() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return map[string]interface{}{
		"monitored_groups": len(d.avgCounts),
		"window_size":      d.windowSize.String(),
		"threshold":        d.thresholdMultiplier,
	}
}
