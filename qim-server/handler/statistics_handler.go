package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"

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

	var totalTasks int64
	db.Model(&model.Task{}).Where("user_id = ? AND created_at >= ?", userID, startDate).Count(&totalTasks)

	var completedTasks int64
	db.Model(&model.Task{}).Where("user_id = ? AND status = ? AND created_at >= ?", userID, "done", startDate).Count(&completedTasks)

	var pendingTasks int64
	db.Model(&model.Task{}).Where("user_id = ? AND status IN ? AND created_at >= ?", userID, []string{"todo", "in_progress"}, startDate).Count(&pendingTasks)

	taskCompletionRate := 0.0
	if totalTasks > 0 {
		taskCompletionRate = float64(completedTasks) / float64(totalTasks) * 100
	}

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

	fileTypes := []map[string]interface{}{}
	var fileTypeRows []struct {
		Type  string
		Count int64
		Size  int64
	}
	db.Model(&model.File{}).Where("user_id = ? AND created_at >= ?", userID, startDate).
		Select(`
			CASE
				WHEN mime_type LIKE 'image/%' THEN '图片'
				WHEN mime_type LIKE 'video/%' THEN '视频'
				WHEN mime_type LIKE 'audio/%' THEN '音频'
				WHEN mime_type LIKE 'application/%' OR mime_type LIKE 'text/%' THEN '文档'
				ELSE '其他'
			END as type,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size
		`).
		Group(`
			CASE
				WHEN mime_type LIKE 'image/%' THEN '图片'
				WHEN mime_type LIKE 'video/%' THEN '视频'
				WHEN mime_type LIKE 'audio/%' THEN '音频'
				WHEN mime_type LIKE 'application/%' OR mime_type LIKE 'text/%' THEN '文档'
				ELSE '其他'
			END
		`).Scan(&fileTypeRows)

	var totalFileCount int64
	for _, row := range fileTypeRows {
		totalFileCount += row.Count
	}
	for _, row := range fileTypeRows {
		pct := 0
		if totalFileCount > 0 {
			pct = int(row.Count * 100 / totalFileCount)
		}
		fileTypes = append(fileTypes, map[string]interface{}{
			"type":       row.Type,
			"count":      row.Count,
			"percentage": pct,
		})
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
