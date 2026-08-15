package handler

import (
	"strconv"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
)

// 外部 Bot 运维视图：管理后台查看外部 agent webhook bot 与其 outbox 投递状态。
// bot 创建审批已在 UnifiedApprovalPanel 覆盖，此处仅做只读监控 + 手动重投。

// externalBotRow 外部 bot 列表行（脱敏：不回 webhook_secret）。
type externalBotRow struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	CreatorName  string `json:"creator_name"`
	IsActive     bool   `json:"is_active"`
	Mode         string `json:"mode"`
	WebhookURL   string `json:"webhook_url"`
	PendingCount int64  `json:"pending_count"`
	DeadCount    int64  `json:"dead_count"`
	CreatedAt    string `json:"created_at"`
}

// AdminGetExternalBots 列出 external_webhook 模式的 bot（含各 bot 的 pending/dead 投递计数）。
func AdminGetExternalBots(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	keyword := c.Query("keyword")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	db := database.GetDB()

	// 外部 bot：Config 里 mode=="external_webhook"。SQLite/MySQL JSON 提取不一致，
	// 这里全量取 bot 后在 Go 层用 ParseBotConfig 过滤（外部 bot 数量有限，可接受）。
	var allBots []model.Bot
	query := db.Model(&model.Bot{}).Where("is_template = ?", false)
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	query.Order("created_at DESC").Find(&allBots)

	var external []model.Bot
	for i := range allBots {
		if service.ParseBotConfig(allBots[i].Config).IsExternalWebhook() {
			external = append(external, allBots[i])
		}
	}

	total := int64(len(external))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}
	pageBots := external[start:end]

	// 批量取本页 bot 的投递计数
	botIDs := make([]uint, 0, len(pageBots))
	for i := range pageBots {
		botIDs = append(botIDs, pageBots[i].ID)
	}
	counts := make(map[uint]map[string]int64, len(pageBots))
	if len(botIDs) > 0 {
		type cnt struct {
			BotID  uint
			Status string
			N      int64
		}
		var rows []cnt
		db.Model(&model.BotWebhookDelivery{}).
			Select("bot_id, status, count(*) as n").
			Where("bot_id IN ?", botIDs).
			Group("bot_id, status").
			Find(&rows)
		for _, r := range rows {
			if counts[r.BotID] == nil {
				counts[r.BotID] = map[string]int64{}
			}
			counts[r.BotID][r.Status] = r.N
		}
	}

	rows2 := make([]externalBotRow, 0, len(pageBots))
	for i := range pageBots {
		b := pageBots[i]
		cfg := service.ParseBotConfig(b.Config)
		var pending, dead int64
		if counts[b.ID] != nil {
			pending = counts[b.ID]["pending"]
			dead = counts[b.ID]["dead"]
		}
		rows2 = append(rows2, externalBotRow{
			ID:           b.ID,
			Name:         b.Name,
			CreatorName:  b.CreatorName,
			IsActive:     b.IsActive,
			Mode:         cfg.Mode,
			WebhookURL:   cfg.WebhookURL,
			PendingCount: pending,
			DeadCount:    dead,
			CreatedAt:    b.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	response.Success(c, gin.H{
		"list":  rows2,
		"total": total,
	})
}

// AdminGetWebhookDeliveries 分页列出 webhook 投递记录。支持 bot_id/event/status 过滤。
func AdminGetWebhookDeliveries(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	db := database.GetDB()
	q := db.Model(&model.BotWebhookDelivery{})

	if botIDStr := c.Query("bot_id"); botIDStr != "" {
		if botID, err := strconv.ParseUint(botIDStr, 10, 64); err == nil {
			q = q.Where("bot_id = ?", botID)
		}
	}
	if event := c.Query("event"); event != "" {
		q = q.Where("event = ?", event)
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	q.Count(&total)

	var deliveries []model.BotWebhookDelivery
	q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&deliveries)

	// 批量取 bot 名（避免 N+1）
	botIDs := make([]uint, 0, len(deliveries))
	for i := range deliveries {
		botIDs = append(botIDs, deliveries[i].BotID)
	}
	botNames := make(map[uint]string, len(deliveries))
	if len(botIDs) > 0 {
		var bots []model.Bot
		db.Select("id, name").Where("id IN ?", botIDs).Find(&bots)
		for i := range bots {
			botNames[bots[i].ID] = bots[i].Name
		}
	}

	// payload 截断预览，详情走单条端点；secret 字段模型已 json:"-"
	type row struct {
		model.BotWebhookDelivery
		BotName        string `json:"bot_name"`
		PayloadPreview string `json:"payload_preview"`
	}
	list := make([]row, 0, len(deliveries))
	for i := range deliveries {
		preview := deliveries[i].Payload
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		list = append(list, row{
			BotWebhookDelivery: deliveries[i],
			BotName:            botNames[deliveries[i].BotID],
			PayloadPreview:     preview,
		})
	}

	response.Success(c, gin.H{
		"list":  list,
		"total": total,
	})
}

// AdminGetWebhookDelivery 单条投递详情（含完整 payload）。
func AdminGetWebhookDelivery(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的投递 ID")
		return
	}

	db := database.GetDB()
	var d model.BotWebhookDelivery
	if err := db.First(&d, id).Error; err != nil {
		response.NotFound(c, "投递记录不存在")
		return
	}

	var botName string
	db.Model(&model.Bot{}).Select("name").Where("id = ?", d.BotID).Scan(&botName)

	response.Success(c, gin.H{
		"delivery": d,
		"bot_name": botName,
	})
}

// AdminRedeliverWebhook 手动重投一条投递记录（通常用于死信）。
func AdminRedeliverWebhook(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的投递 ID")
		return
	}

	db := database.GetDB()
	// 先看当前状态用于反馈
	var before model.BotWebhookDelivery
	if err := db.First(&before, id).Error; err != nil {
		response.NotFound(c, "投递记录不存在")
		return
	}

	if err := service.Redeliver(db, uint(id)); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 重投后 reload 实际状态反馈给前端
	var after model.BotWebhookDelivery
	db.First(&after, id)
	response.SuccessWithMessage(c, "已触发重投", gin.H{
		"status":        after.Status,
		"attempts":      after.Attempts,
		"last_error":    after.LastError,
		"next_retry_at": after.NextRetryAt,
	})
}
