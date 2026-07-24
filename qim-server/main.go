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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dshmyz/qim/qim-server/app"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/handler"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/scheduler"
	"github.com/dshmyz/qim/qim-server/service"
	syncpkg "github.com/dshmyz/qim/qim-server/sync"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化应用
	cfg, db, hub := app.InitApp()

	// 注册 WS 消息处理回调，统一使用 MessageService
	handler.InitWSHandlers()

	// 启动统一调度器（基于 robfig/cron/v3，管理所有定时任务）
	sched := scheduler.New()
	if err := sched.AddDailyJob("group-summary", "22:00", func(ctx context.Context) {
		summaryJob := handler.NewGroupSummaryJob(app.GetAIService())
		summaryJob.GenerateDailySummaries()
	}); err != nil {
		logger.L().Error("注册群聊总结 cron job 失败", "error", err)
	}
	// 事件提醒：每 30 秒扫描需要提醒的事件
	if err := sched.AddIntervalJob("event-reminder", 30*time.Second, func(ctx context.Context) {
		di.GlobalContainer.EventService.ProcessReminders()
	}); err != nil {
		logger.L().Error("注册事件提醒 cron job 失败", "error", err)
	}
	// Bot webhook 重试：每 15 秒扫描 outbox 待投递记录，指数退避重试，超阈值死信
	if err := sched.AddIntervalJob("bot-webhook-retry", 15*time.Second, func(ctx context.Context) {
		service.ProcessPendingDeliveries(db)
	}); err != nil {
		logger.L().Error("注册 bot webhook 重试 cron job 失败", "error", err)
	}
	// 组织架构同步：从 DB 加载 OrgSyncConfig 并注册为 cron job
	syncEngine := syncpkg.NewEngine()
	syncpkg.SharedEngine = syncEngine
	if err := syncpkg.LoadOrgSyncJobs(sched, syncEngine, db); err != nil {
		logger.L().Error("加载 OrgSync cron jobs 失败", "error", err)
	}
	// 注入到 DI 容器，让 handler 通过 di.GlobalContainer 访问
	di.GlobalContainer.Scheduler = sched
	di.GlobalContainer.SyncEngine = syncEngine

	jobCtx, jobCancel := context.WithCancel(context.Background())
	sched.Start(jobCtx)
	logger.L().Info("统一调度器已启动（群聊总结 22:00 / 事件提醒 30s / OrgSync 配置驱动）")

	// 使用 gin.New() 替代 gin.Default()，避免 Logger 中间件的 stdout IO 瓶颈
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(loggerRequestMiddleware())

	// 设置路由
	app.SetupRoutes(r, cfg, hub)

	// 优雅退出：使用 http.Server.Shutdown 等待连接自然结束
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		logger.L().Info("收到退出信号，正在优雅关闭...")
		jobCancel()  // 停止统一调度器（所有 Job：群聊总结 / 事件提醒 / OrgSync）
		sched.Stop() // 等待所有 Job 退出

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.L().Error("HTTP 优雅关闭失败", "error", err)
		}
	}()

	// 启动服务器
	logger.L().Info("服务器启动在端口", "port", cfg.Server.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.WithModule("main").Error("服务器启动失败", "error", err)
		os.Exit(1)
	}

	// Shutdown 完成后关闭 DB
	database.Close(db)
	logger.L().Info("服务器已关闭")
	os.Exit(0)
}

// loggerRequestMiddleware 轻量请求日志，替代 gin.Default() 的 Logger
func loggerRequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		logger.L().Debug("HTTP",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", latency.Milliseconds(),
			"ip", c.ClientIP(),
		)
	}
}
