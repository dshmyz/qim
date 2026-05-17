package main

// @title QIM Server API
// @version 2.0
// @description QIM 智能办公平台后端 API 文档
// @contact.name QIM Team
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

import (
	"os"
	"qim-server/app"
	"qim-server/database"
	"qim-server/handler"
	"qim-server/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化应用
	cfg, db, hub := app.InitApp()
	defer database.Close(db)

	// 启动群聊总结定时任务
	summaryJob := handler.NewGroupSummaryJob(app.GetAIService())
	go runDailyJob("22:00", func() {
		summaryJob.GenerateDailySummaries()
	})
	logger.L().Info("定时任务已启动：群聊总结 (每天 22:00)")

	// 初始化Gin
	r := gin.Default()

	// 设置路由
	app.SetupRoutes(r, cfg, hub)

	// 启动服务器
	logger.L().Info("服务器启动在端口", "port", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		logger.WithModule("main").Error("服务器启动失败", "error", err)
		os.Exit(1)
	}
}

// runDailyJob runs a job daily at the specified time (format: "HH:MM")
func runDailyJob(timeStr string, job func()) {
	for {
		now := time.Now()
		scheduledTime, err := time.ParseInLocation("15:04", timeStr, time.Local)
		if err != nil {
			logger.WithModule("main").Error("定时任务时间解析失败", "error", err)
			time.Sleep(1 * time.Hour)
			continue
		}

		scheduledTime = scheduledTime.AddDate(now.Year(), int(now.Month())-1, now.Day()-1)
		if scheduledTime.Before(now) {
			scheduledTime = scheduledTime.Add(24 * time.Hour)
		}

		time.Sleep(scheduledTime.Sub(now))
		job()
	}
}
