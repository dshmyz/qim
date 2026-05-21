// @title QIM Server API
// @version 2.0
// @description QIM 智能办公平台后端 API 文档
// @contact.name QIM Team
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

package main

import (
	"context"
	"os"
	"os/signal"
	"qim-server/app"
	"qim-server/database"
	"qim-server/handler"
	"qim-server/pkg/logger"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化应用
	cfg, db, hub := app.InitApp()
	defer database.Close(db)

	// 注册 WS 消息处理回调，统一使用 MessageService
	handler.InitWSHandlers()

	// 启动群聊总结定时任务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	summaryJob := handler.NewGroupSummaryJob(app.GetAIService())
	go runDailyJob(ctx, "22:00", func() {
		summaryJob.GenerateDailySummaries()
	})
	logger.L().Info("定时任务已启动：群聊总结 (每天 22:00)")

	// 优化：使用 gin.New() 替代 gin.Default()，避免 Logger 中间件的 stdout IO 瓶颈
	r := gin.New()
	r.Use(gin.Recovery())

	// 设置路由
	app.SetupRoutes(r, cfg, hub)

	// 优雅退出：监听信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		logger.L().Info("收到退出信号，正在优雅关闭...")
		cancel()
		hub.Broadcast = nil
		os.Exit(0)
	}()

	// 启动服务器
	logger.L().Info("服务器启动在端口", "port", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		logger.WithModule("main").Error("服务器启动失败", "error", err)
		os.Exit(1)
	}
}

// runDailyJob runs a job daily at the specified time (format: "HH:MM")
// 支持 context 优雅退出
func runDailyJob(ctx context.Context, timeStr string, job func()) {
	for {
		now := time.Now()
		scheduledTime, err := time.ParseInLocation("15:04", timeStr, time.Local)
		if err != nil {
			logger.WithModule("main").Error("定时任务时间解析失败", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Hour):
				continue
			}
		}

		scheduledTime = scheduledTime.AddDate(now.Year(), int(now.Month())-1, now.Day()-1)
		if scheduledTime.Before(now) {
			scheduledTime = scheduledTime.Add(24 * time.Hour)
		}

		select {
		case <-ctx.Done():
			logger.WithModule("main").Info("定时任务收到退出信号，停止运行")
			return
		case <-time.After(scheduledTime.Sub(now)):
			job()
		}
	}
}
