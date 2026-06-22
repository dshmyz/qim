package handler

import (
	"encoding/csv"
	"encoding/json"
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

func GetOperationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	module := c.Query("module")
	action := c.Query("action")
	username := c.Query("username")
	status := c.Query("status")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	db := database.GetDB()

	query := db.Model(&model.OperationLog{})
	if module != "" {
		query = query.Where("module = ?", module)
	}
	if action != "" {
		query = query.Where("action LIKE ?", "%"+action+"%")
	}
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	var total int64
	query.Count(&total)

	var logs []model.OperationLog
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs)

	if status != "" {
		var filtered []model.OperationLog
		for _, log := range logs {
			isSuccess := isResponseSuccess(log.Response)
			if (status == "success" && isSuccess) || (status == "failed" && !isSuccess) {
				filtered = append(filtered, log)
			}
		}
		logs = filtered
	}

	response.Success(c, gin.H{
		"list":     operationLogsToFrontend(logs),
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func GetOperationLogDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	db := database.GetDB()
	var log model.OperationLog
	if err := db.First(&log, id).Error; err != nil {
		response.NotFound(c, "日志记录不存在")
		return
	}

	response.Success(c, operationLogToFrontend(log))
}

func GetOperationLogStats(c *gin.Context) {
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")
	trend := c.Query("trend") == "true"

	db := database.GetDB()

	query := db.Model(&model.OperationLog{})
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	var total int64
	query.Count(&total)

	var successCount int64
	var failedCount int64
	var avgDuration float64

	var allLogs []model.OperationLog
	query.Select("response", "duration").Find(&allLogs)

	for _, log := range allLogs {
		if isResponseSuccess(log.Response) {
			successCount++
		} else {
			failedCount++
		}
		avgDuration += float64(log.Duration)
	}

	if len(allLogs) > 0 {
		avgDuration = avgDuration / float64(len(allLogs))
	}

	result := gin.H{
		"total":        total,
		"success":      successCount,
		"failed":       failedCount,
		"avgDuration":  int(avgDuration),
	}

	if trend {
		var trendData []gin.H
		now := time.Now()
		for i := 6; i >= 0; i-- {
			day := now.AddDate(0, 0, -i)
			dayStart := day.Format("2006-01-02")
			dayEnd := dayStart + " 23:59:59"

			var count int64
			db.Model(&model.OperationLog{}).
				Where("created_at >= ? AND created_at <= ?", dayStart, dayEnd).
				Count(&count)

			trendData = append(trendData, gin.H{
				"date":  day.Format("01-02"),
				"count": count,
			})
		}
		result["trend"] = trendData
	}

	response.Success(c, result)
}

func isResponseSuccess(resp string) bool {
	if resp == "" {
		return true
	}
	var result struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return true
	}
	return result.Code == 0
}

func ExportOperationLogs(c *gin.Context) {
	db := database.GetDB()
	var logs []model.OperationLog
	db.Order("created_at DESC").Limit(10000).Find(&logs)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=operation_logs.csv")

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"ID", "用户", "操作", "模块", "IP", "请求URL", "耗时(ms)", "时间"})

	for _, log := range logs {
		w.Write([]string{
			strconv.FormatUint(uint64(log.ID), 10),
			log.Username,
			log.Action,
			log.Module,
			log.IP,
			log.RequestURL,
			strconv.Itoa(log.Duration),
			log.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()
}
