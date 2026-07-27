package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"gorm.io/gorm"
)

// webhook outbox：出站 webhook 先落表再异步投递，失败指数退避重试，超阈值进死信。
// 兜底 agent webhook 端点短暂不可用导致的丢消息。

// MaxAttempts 最大重试次数（含首次），超过即死信。
const MaxAttempts = 4

// backoff 重试间隔序列：首次失败后 30s、2m、10m、1h。
var backoff = []time.Duration{
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
	time.Hour,
}

// EnqueueWebhookDelivery 先落 outbox 表（pending），返回 delivery ID。不发起 HTTP。
// 调用方拿到 ID 后可立即 deliverOnce 试一次（best-effort），失败则由调度器后续重试。
func EnqueueWebhookDelivery(db *gorm.DB, botID uint, event, payloadJSON, webhookURL, webhookSecret string) (uint, error) {
	d := model.BotWebhookDelivery{
		BotID:         botID,
		Event:         event,
		Payload:       payloadJSON,
		WebhookURL:    webhookURL,
		WebhookSecret: webhookSecret,
		Status:        "pending",
	}
	if err := db.Create(&d).Error; err != nil {
		return 0, err
	}
	return d.ID, nil
}

// deliverOnce 单次投递：反序列化 payload 调 SendBotWebhook。
// 成功 -> done + delivered_at；失败 -> attempts++ + next_retry_at（按 backoff），
// 超过 MaxAttempts -> dead + last_error。乐观锁防调度器重叠双投。
// 返回 (终态?, err)。
func deliverOnce(db *gorm.DB, d *model.BotWebhookDelivery) (bool, error) {
	// 乐观锁：只在自己的 attempts 版本上推进，防 job 重叠双投
	res := db.Model(&model.BotWebhookDelivery{}).
		Where("id = ? AND attempts = ?", d.ID, d.Attempts).
		Update("attempts", d.Attempts+1)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		// 已被其他 job 推进，放弃本次
		return true, nil
	}
	d.Attempts++

	var payload BotWebhookPayload
	if err := json.Unmarshal([]byte(d.Payload), &payload); err != nil {
		markDead(db, d, "反序列化 payload 失败: "+err.Error())
		return true, nil
	}

	err := SendBotWebhook(d.WebhookURL, d.WebhookSecret, payload)
	if err == nil {
		now := time.Now()
		db.Model(&model.BotWebhookDelivery{}).Where("id = ?", d.ID).Updates(map[string]any{
			"status":       "done",
			"delivered_at": now,
			"next_retry_at": nil,
			"last_error":    "",
		})
		logger.WithModule("BotWebhookOutbox").Info("webhook 投递成功",
			"deliveryID", d.ID, "botID", d.BotID, "event", d.Event, "attempts", d.Attempts)
		return true, nil
	}

	// 失败：记 last_error，决定重试或死信
	if d.Attempts >= MaxAttempts {
		markDead(db, d, err.Error())
		return true, nil
	}
	nextRetry := nextRetryTime(d.Attempts)
	db.Model(&model.BotWebhookDelivery{}).Where("id = ?", d.ID).Updates(map[string]any{
		"status":        "pending",
		"next_retry_at": nextRetry,
		"last_error":    truncateErr(err.Error()),
	})
	logger.WithModule("BotWebhookOutbox").Warn("webhook 投递失败，将重试",
		"deliveryID", d.ID, "botID", d.BotID, "attempts", d.Attempts, "nextRetry", nextRetry, "error", err)
	return false, nil
}

// markDead 标记死信，保留 last_error 供排查。
func markDead(db *gorm.DB, d *model.BotWebhookDelivery, lastErr string) {
	db.Model(&model.BotWebhookDelivery{}).Where("id = ?", d.ID).Updates(map[string]any{
		"status":        "dead",
		"next_retry_at": nil,
		"last_error":    truncateErr(lastErr),
	})
	logger.WithModule("BotWebhookOutbox").Error("webhook 投递进入死信",
		"deliveryID", d.ID, "botID", d.BotID, "event", d.Event, "attempts", d.Attempts, "error", lastErr)
}

// nextRetryTime 按 attempts（已含本次失败）取下次重试时间。
func nextRetryTime(attempts int) time.Time {
	idx := attempts - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(backoff) {
		idx = len(backoff) - 1
	}
	return time.Now().Add(backoff[idx])
}

// ProcessPendingDeliveries 调度器 job：扫到期的 pending 投递记录，逐条投递。
// 到期 = status=pending AND (next_retry_at IS NULL OR next_retry_at <= now)。
// LIMIT 50 按 created_at 升序（防饥饿），单条失败不影响其他。
func ProcessPendingDeliveries(db *gorm.DB) {
	var deliveries []model.BotWebhookDelivery
	now := time.Now()
	err := db.Where("status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)", "pending", now).
		Order("created_at ASC").Limit(50).Find(&deliveries).Error
	if err != nil {
		logger.WithModule("BotWebhookOutbox").Error("扫描待投递记录失败", "error", err)
		return
	}
	for i := range deliveries {
		// 重新加载最新 attempts（防与立即投递竞态）
		var d model.BotWebhookDelivery
		if err := db.First(&d, deliveries[i].ID).Error; err != nil {
			continue
		}
		if d.Status != "pending" {
			continue
		}
		_, _ = deliverOnce(db, &d)
	}
}

// DeliverOnce 暴露给调用方做"立即试一次"（best-effort 同步投递）。
// 成功或进入重试队列后返回 nil；只有落表/乐观锁等本地错误才返回 err。
func DeliverOnce(db *gorm.DB, deliveryID uint) error {
	var d model.BotWebhookDelivery
	if err := db.First(&d, deliveryID).Error; err != nil {
		return err
	}
	if d.Status != "pending" {
		return nil
	}
	_, err := deliverOnce(db, &d)
	return err
}

// Redeliver 手动重投：把 dead（或 pending 跳过退避等待）重置为 pending 后立即投递一次。
// 供管理后台"手动重投"调用。dead 是终态，deliverOnce 不处理，须先重置：
// 归零 attempts、清 next_retry_at/last_error，让 deliverOnce 从头跑。
// 终态 done 不可重投（避免重复投递已成功消息）。
func Redeliver(db *gorm.DB, deliveryID uint) error {
	var d model.BotWebhookDelivery
	if err := db.First(&d, deliveryID).Error; err != nil {
		return err
	}
	if d.Status == "done" {
		return errors.New("已投递成功的记录不可重投")
	}
	// 重置为 pending：dead/pending 均可。乐观条件防与调度器竞态。
	if err := db.Model(&model.BotWebhookDelivery{}).Where("id = ?", d.ID).Updates(map[string]any{
		"status":        "pending",
		"attempts":      0,
		"next_retry_at": nil,
		"last_error":    "",
	}).Error; err != nil {
		return err
	}
	// 重新加载拿归零后的 attempts，再立即投一次
	d = model.BotWebhookDelivery{}
	if err := db.First(&d, deliveryID).Error; err != nil {
		return err
	}
	_, err := deliverOnce(db, &d)
	return err
}

func truncateErr(s string) string {
	if len(s) > 500 {
		return s[:500] + "..."
	}
	return s
}

// 编译期保证 errors 被引用（未来校验扩展用）。
var _ = errors.New
