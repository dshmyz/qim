package handler

import (
	"encoding/csv"
	"strconv"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

func GetOperationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	module := c.Query("module")
	action := c.Query("action")
	username := c.Query("username")

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

	var total int64
	query.Count(&total)

	var logs []model.OperationLog
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs)

	response.Success(c, gin.H{
		"list":     logs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
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
