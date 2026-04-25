package handler

import (
	"fmt"
	"net/http"
	"time"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
)

func GetStatistics(c *gin.Context) {
	userID, _ := c.Get("user_id")
	period := c.DefaultQuery("period", "week")

	db := database.GetDB()

	now := time.Now()
	var startDate time.Time

	switch period {
	case "day":
		startDate = now.AddDate(0, 0, -1)
	case "week":
		startDate = now.AddDate(0, 0, -7)
	case "month":
		startDate = now.AddDate(0, -1, 0)
	case "year":
		startDate = now.AddDate(-1, 0, 0)
	default:
		startDate = now.AddDate(0, 0, -7)
	}

	var totalMessages int64
	db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ?", userID, startDate).Count(&totalMessages)

	var totalFiles int64
	db.Model(&model.File{}).Where("user_id = ? AND created_at >= ?", userID, startDate).Count(&totalFiles)

	var totalNotes int64
	db.Model(&model.Note{}).Where("user_id = ? AND created_at >= ?", userID, startDate).Count(&totalNotes)

	totalTasks := int64(0)
	completedTasks := int64(0)
	pendingTasks := int64(0)
	taskCompletionRate := 0.0

	messageTrend := []map[string]interface{}{}

	switch period {
	case "day":
		for i := 23; i >= 0; i-- {
			hour := now.Add(-time.Duration(i) * time.Hour)
			hourStr := fmt.Sprintf("%d:00", hour.Hour())

			var count int64
			db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ? AND created_at < ?",
				userID, hour, hour.Add(time.Hour)).Count(&count)

			messageTrend = append(messageTrend, map[string]interface{}{
				"date":  hourStr,
				"count": count,
			})
		}
	case "week":
		for i := 6; i >= 0; i-- {
			date := now.AddDate(0, 0, -i)
			dateStr := fmt.Sprintf("%d/%d", date.Month(), date.Day())

			var count int64
			db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ? AND created_at < ?",
				userID, date, date.AddDate(0, 0, 1)).Count(&count)

			messageTrend = append(messageTrend, map[string]interface{}{
				"date":  dateStr,
				"count": count,
			})
		}
	case "month":
		for i := 3; i >= 0; i-- {
			weekStart := now.AddDate(0, 0, -i*7)
			weekStr := fmt.Sprintf("第%d周", 4-i)

			var count int64
			db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ? AND created_at < ?",
				userID, weekStart, weekStart.AddDate(0, 0, 7)).Count(&count)

			messageTrend = append(messageTrend, map[string]interface{}{
				"date":  weekStr,
				"count": count,
			})
		}
	case "year":
		for i := 11; i >= 0; i-- {
			month := now.AddDate(0, -i, 0)
			monthStr := fmt.Sprintf("%d月", month.Month())

			var count int64
			db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ? AND created_at < ?",
				userID, month, month.AddDate(0, 1, 0)).Count(&count)

			messageTrend = append(messageTrend, map[string]interface{}{
				"date":  monthStr,
				"count": count,
			})
		}
	default:
		for i := 6; i >= 0; i-- {
			date := now.AddDate(0, 0, -i)
			dateStr := fmt.Sprintf("%d/%d", date.Month(), date.Day())

			var count int64
			db.Model(&model.Message{}).Where("sender_id = ? AND created_at >= ? AND created_at < ?",
				userID, date, date.AddDate(0, 0, 1)).Count(&count)

			messageTrend = append(messageTrend, map[string]interface{}{
				"date":  dateStr,
				"count": count,
			})
		}
	}

	fileTypes := []map[string]interface{}{
		{"type": "文档", "count": int64(120), "percentage": 35},
		{"type": "图片", "count": int64(80), "percentage": 23},
		{"type": "视频", "count": int64(60), "percentage": 17},
		{"type": "音频", "count": int64(40), "percentage": 12},
		{"type": "其他", "count": int64(45), "percentage": 13},
	}

	maxMessages := int64(0)
	for _, item := range messageTrend {
		if count, ok := item["count"].(int64); ok && count > maxMessages {
			maxMessages = count
		}
	}

	statisticsData := map[string]interface{}{
		"totalMessages":      totalMessages,
		"totalFiles":         totalFiles,
		"totalNotes":         totalNotes,
		"totalTasks":         totalTasks,
		"completedTasks":     completedTasks,
		"pendingTasks":       pendingTasks,
		"taskCompletionRate": taskCompletionRate,
		"maxMessages":        maxMessages,
		"messageTrend":       messageTrend,
		"fileTypes":          fileTypes,
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": statisticsData,
	})
}
