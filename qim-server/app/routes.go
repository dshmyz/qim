package app

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"qim-server/ai"
	"qim-server/config"
	"qim-server/di"
	"qim-server/handler"
	"qim-server/middleware"
	"qim-server/pkg/logger"
	"qim-server/service"
	syncpkg "qim-server/sync"
	"qim-server/utils"
	"qim-server/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// GetAIService returns the global AI service instance
func GetAIService() *ai.AIService {
	return di.GlobalContainer.AIService
}

// SetupRoutes 设置 API 路由
func SetupRoutes(r *gin.Engine, cfg *config.Config, hub *ws.Hub) {
	handler.SetConfig(cfg)
	ws.SetAllowedOrigins(cfg.WS.AllowedOrigins)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"version":   "2.0",
			"timestamp": time.Now().Unix(),
		})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	aiSvc := di.GlobalContainer.AIService

	mcpServer := ai.NewMCPServer(false, aiSvc)

	aiSvc.SetMCPServer(mcpServer)

	handler.RegisterAdminTools(mcpServer)

	groupDocSvc := di.GlobalContainer.GroupDocumentService
	var uk *service.UnifiedKnowledgeService
	if vectorSvc := di.GlobalContainer.VectorService; vectorSvc != nil {
		service.NewUnifiedMCPBridge(mcpServer, vectorSvc.GetDB())

		fallback := &service.LegacyKnowledgeFallback{
			SearchFunc: func(query string, groupID uint, limit int) []service.KnowledgeSnippet {
				return nil
			},
		}
		uk = service.NewUnifiedKnowledgeService(groupDocSvc, fallback)
		uk.SetGraphEnhanced(true)
	}

	handler.InitSmartReplyEngine(aiSvc)

	// 设置依赖（在 InitSmartReplyEngine 之后）
	if uk != nil {
		handler.SetUnifiedKnowledge(uk)
	}
	if avatarMemorySvc := di.GlobalContainer.AvatarMemoryService; avatarMemorySvc != nil {
		handler.SetMemoryService(avatarMemorySvc)
	}

	// 初始化 SmartReplyGraph（使用 Eino 框架编排）
	if err := handler.InitSmartReplyGraph(); err != nil {
		logger.WithModule("Routes").Warn("初始化 SmartReplyGraph 失败，将使用旧方法", "error", err)
	}

	handler.InitAnomalyDetector()

	utils.SafeGoWithLabel("mcp-server", func() {
		if err := mcpServer.Start(":8081"); err != nil {
		}
	})

	aiHandler := handler.NewAIHandler(aiSvc, mcpServer)

	aiCache := service.NewAICache()

	summaryGraph := service.NewSummaryGraph(aiSvc, aiCache)
	if err := summaryGraph.Build(); err != nil {
		logger.WithModule("Routes").Warn("初始化 SummaryGraph 失败", "error", err)
	} else {
		aiHandler.SetSummaryGraph(summaryGraph)
		logger.WithModule("Routes").Info("SummaryGraph 初始化成功")
	}

	textProcessGraph := service.NewTextProcessGraph(aiSvc, aiCache)
	if err := textProcessGraph.Build(); err != nil {
		logger.WithModule("Routes").Warn("初始化 TextProcessGraph 失败", "error", err)
	} else {
		aiHandler.SetTextProcessGraph(textProcessGraph)
		logger.WithModule("Routes").Info("TextProcessGraph 初始化成功")
	}

	noteVectorSvc := di.GlobalContainer.NoteVectorService
	avatarMemorySvc := di.GlobalContainer.AvatarMemoryService
	unifiedSearchGraph := service.NewUnifiedSearchGraph(aiSvc, noteVectorSvc, groupDocSvc, avatarMemorySvc)
	if err := unifiedSearchGraph.Build(); err != nil {
		logger.WithModule("Routes").Warn("初始化 UnifiedSearchGraph 失败", "error", err)
	} else {
		aiHandler.SetUnifiedSearchGraph(unifiedSearchGraph)
		logger.WithModule("Routes").Info("UnifiedSearchGraph 初始化成功")
	}

	smartDigestGraph := service.NewSmartDigestGraph(aiSvc, aiCache)
	if err := smartDigestGraph.Build(); err != nil {
		logger.WithModule("Routes").Warn("Failed to build SmartDigestGraph", "error", err)
	} else {
		aiHandler.SetSmartDigestGraph(smartDigestGraph)
		logger.WithModule("Routes").Info("SmartDigestGraph 初始化成功")
	}

	avatarService := service.NewAvatarService(GetDB(), aiSvc)
	handler.SetAvatarWorkerPool(avatarService.GetWorkerPool())

	// 注入 WebSocket 消息回调，使分身/智能回复在 WebSocket 发送消息时也触发
	hub.OnMessageSent = func(senderID uint, conversationID uint, content string, mentionUserIDs []uint) {
		sre := handler.GetSmartReplyEngine()
		if sre != nil {
			sre.HandleMessage(senderID, conversationID, content, mentionUserIDs)
		}
	}

	avatarService.SetGroupDocumentService(groupDocSvc)

	// 注入 WebSocket 通知回调（分身学习完成时推送）
	avatarService.SetWebSocketNotify(func(userID uint, eventType string, data map[string]interface{}) {
		if ws.GlobalHub != nil {
			payload := map[string]interface{}{
				"type": eventType,
				"data": data,
			}
			jsonData, _ := json.Marshal(payload)
			ws.GlobalHub.SendToUser(userID, jsonData)
		}
	})

	// 自定义CORS中间件，确保所有响应都包含CORS头
	corsMiddleware := cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	})

	// 全局应用CORS中间件
	r.Use(corsMiddleware)

	// 全局中间件
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.LoggerMiddleware())

	// 请求限流（500 请求/分钟/IP）
	rateLimiter := middleware.NewIPRateLimiter(500, time.Minute)
	r.Use(middleware.RateLimitMiddleware(rateLimiter))

	// 静态文件服务（带缓存头 + 路径遍历防护）
	r.GET("/uploads/*filepath", func(c *gin.Context) {
		fp := c.Param("filepath")
		if strings.Contains(fp, "..") {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		baseDir := "./uploads"
		cleanPath := filepath.Clean(filepath.Join(baseDir, fp))
		if !strings.HasPrefix(cleanPath, baseDir) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Header("Cache-Control", "public, max-age=86400")
		c.File(cleanPath)
	})
	r.Static("/miniprograms", "./static/miniprograms")

	// 客户端更新检查（无需认证）
	r.GET("/api/v1/updates/:platform/latest.yml", handler.GetLatestYML)
	r.GET("/api/v1/updates/:platform/files/:filename", handler.GetUpdateFile)

	// 使用静态文件处理函数，并确保CORS中间件应用

	// API路由
	api := r.Group("/api/v1")
	{
		// 组织架构同步Webhook（公开端点，外部系统调用）
		api.POST("/org/sync/webhook", syncpkg.WebhookHandler)

		// 认证路由
		auth := api.Group("/auth")
		loginLimiter := middleware.NewLoginLimiter(5, time.Minute, 15*time.Minute)
		middleware.SetGlobalLoginLimiter(loginLimiter)
		{
			auth.POST("/login", middleware.LoginRateLimitMiddleware(loginLimiter), handler.Login)
			auth.POST("/register", handler.Register)
			auth.POST("/2fa/verify", middleware.LoginRateLimitMiddleware(loginLimiter), handler.VerifyTwoFA)
			auth.POST("/check-2fa", handler.CheckTwoFAStatus)
			auth.POST("/check-version", handler.CheckVersion)

			// 公开的认证提供者列表（无需认证）
			authProviderHandler := handler.NewAuthProviderHandler()
			auth.GET("/providers", authProviderHandler.GetProviders)

			// OAuth回调（无需认证）
			auth.POST("/oauth/callback", handler.OAuthCallback)

			// CAS回调（无需认证）
			auth.POST("/cas/callback", handler.CASCallback)

			// 统一认证回调（支持OAuth和CAS，无需认证）
			auth.POST("/callback", handler.UnifiedAuthCallback)
		}

		// 需要认证的认证相关路由
		authAuthed := api.Group("/auth")
		authAuthed.Use(middleware.AuthMiddleware(cfg.JWT.Secret, di.GlobalContainer.UserService))
		{
			authAuthed.POST("/logout", handler.Logout)
			authAuthed.POST("/refresh", handler.RefreshToken)
		}

		// 需要认证的路由
		authed := api.Group("")
		authed.Use(middleware.AuthMiddleware(cfg.JWT.Secret, di.GlobalContainer.UserService))
		{
			// 公开系统配置（客户端可读）
			authed.GET("/system/public-config", handler.GetPublicSystemConfig)

			// 用户
			authed.GET("/users/me", handler.GetCurrentUser)
			authed.PUT("/users/me", handler.UpdateUser)
			authed.GET("/users/:id", handler.GetUserByID)
			// 用户状态查询
			authed.GET("/users/status", handler.GetUserStatus)
			authed.GET("/users/status/batch", handler.GetUserStatusBatch)
			// AI配置
			authed.GET("/ai/config", handler.GetAIConfig)
			authed.PUT("/ai/config", handler.UpdateAIConfig)

			// 组织架构
			authed.GET("/organization/tree", handler.GetOrganizationTree)
			// 创建部门
			authed.POST("/departments", handler.CreateDepartment)
			// 删除部门
			authed.DELETE("/departments/:id", handler.DeleteDepartment)
			// 获取部门员工
			authed.GET("/departments/:id/employees", handler.GetDepartmentEmployees)
			// 从部门移除员工
			authed.DELETE("/department-employees/:id/:user_id", handler.RemoveEmployeeFromDepartment)
			// 创建用户
			authed.POST("/users", handler.CreateUser)
			// 关联用户和部门
			authed.POST("/department-employees", handler.AddUserToDepartment)

			// 会话
			authed.GET("/conversations", handler.GetConversations)
			authed.POST("/conversations/single", handler.CreateSingleConversation)
			authed.POST("/conversations/bot", handler.CreateBotConversation)
			authed.POST("/conversations/group", handler.CreateGroupConversation)
			authed.POST("/conversations/discussion", handler.CreateDiscussionConversation)
			authed.GET("/conversations/:id", handler.GetConversation)
			// 会话置顶/取消置顶
			authed.PUT("/conversations/:id/pin", handler.PinConversation)
			// 设置免打扰
			authed.PUT("/conversations/:id/mute", handler.SetConversationMute)
			// 解散群聊
			authed.DELETE("/conversations/:id", handler.DeleteConversation)

			// 消息
			authed.GET("/conversations/:id/messages", handler.GetMessages)
			authed.POST("/conversations/:id/messages", handler.SendMessage)
			authed.POST("/conversations/:id/messages/stream", handler.StreamMessage)
			authed.POST("/conversations/:id/read", handler.MarkConversationAsRead)
			authed.GET("/messages/:id/read-users", handler.GetMessageReadUsers)
			authed.POST("/messages/batch/read-users", handler.BatchGetMessageReadUsers)
			// 消息撤回
			authed.POST("/messages/:id/recall", handler.RecallMessage)

			// 消息提醒
			authed.POST("/messages/:id/remind", handler.RemindMessage)

			// 消息已读状态消息删除
			authed.DELETE("/messages/:id", handler.DeleteMessage)
			// 消息搜索
			authed.GET("/messages/search", handler.SearchMessages)
			// 获取消息引用链
			authed.GET("/messages/:id/quote-chain", handler.GetMessageQuoteChain)

			// 群聊管理（群特有功能）
			authed.POST("/groups/:id/members", handler.AddMemberToGroup)
			// 移除群成员
			authed.DELETE("/groups/:id/members/:user_id", handler.RemoveMemberFromGroup)
			// 退出群聊
			authed.POST("/groups/:id/exit", handler.ExitGroup)
			// 申请加入群聊
			authed.POST("/groups/:id/apply", handler.ApplyJoinGroup)
			// 拒绝加入请求
			authed.DELETE("/groups/:id/join-requests/:user_id", handler.RejectJoinRequest)
			// 更新群聊信息
			authed.PUT("/groups/:id", handler.UpdateGroupInfo)
			// 获取群聊 AI 设置
			authed.GET("/groups/:id/ai-settings", handler.GetGroupAISettings)
			// 更新群聊 AI 设置
			authed.PUT("/groups/:id/ai-settings", handler.UpdateGroupAISettings)
			// 群知识库管理（带处理状态）
			authed.GET("/groups/:id/ai-documents", handler.GetGroupDocumentsWithStatus)
			authed.POST("/groups/:id/ai-documents", handler.AddGroupDocument)
			authed.DELETE("/groups/:id/ai-documents/:file_id", handler.RemoveGroupDocument)
			authed.POST("/groups/:id/ai-documents/:file_id/process", handler.ProcessGroupDocument)
			authed.GET("/groups/:id/ai-documents/:file_id/status", handler.GetDocumentProcessStatus)
			authed.POST("/groups/:id/ai-documents/batch-process", handler.BatchProcessDocuments)
			authed.POST("/groups/:id/ai-documents/batch-retry", handler.BatchRetryDocuments)
			// 设置/取消管理员
			authed.PUT("/groups/:id/members/:user_id/role", handler.SetMemberRole)
			// 转让群主
			authed.POST("/groups/:id/members/:user_id/transfer-owner", handler.TransferOwner)
			// 更新群公告
			authed.PUT("/groups/:id/announcement", handler.UpdateAnnouncement)
			// 解散群聊
			authed.DELETE("/groups/:id", handler.DissolveGroup)

			// WebSocket
			authed.GET("/ws", func(c *gin.Context) {
				ws.ServeWs(hub, c)
			})

			// 屏幕共享 WebSocket
			authed.GET("/screen-share", func(c *gin.Context) {
				ws.ServeScreenShare(hub, c)
			})

			// 文件上传
			authed.POST("/upload", handler.UploadFile)

			// 文件管理
			authed.GET("/files", handler.GetFiles)
			authed.GET("/files/starred", handler.GetStarredFiles)
			authed.GET("/files/stats", handler.GetFileStats)
			authed.POST("/files/batch", handler.BatchOperation)
			authed.PUT("/files/:id", handler.UpdateFile)
			authed.PUT("/files/:id/star", handler.ToggleStar)
			authed.GET("/files/:id/download", handler.DownloadFile)
			authed.GET("/files/:id/preview", handler.PreviewFile)
			authed.DELETE("/files/:id", handler.DeleteFile)

			// 分片上传
			authed.POST("/files/upload/init", handler.InitUpload)
			authed.POST("/files/upload/chunk", handler.UploadChunk)
			authed.POST("/files/upload/complete", handler.CompleteUpload)
			authed.POST("/files/upload/cancel", handler.CancelUpload)

			// 笔记管理
			authed.GET("/notes", handler.GetNotes)
			authed.GET("/notes/:id", handler.GetNote)
			authed.POST("/notes", handler.CreateNote)
			authed.PUT("/notes/:id", handler.UpdateNote)
			authed.DELETE("/notes/:id", handler.DeleteNote)
			authed.POST("/notes/:id/analyze", handler.AnalyzeNote)
			authed.GET("/notes/:id/export", handler.ExportNote)
			authed.PATCH("/notes/:id/tags", handler.UpdateNoteTags)
			authed.PATCH("/notes/:id/summary", handler.UpdateNoteSummary)
			authed.POST("/notes/search", handler.NoteVectorSearch)

			// 文件夹管理
			authed.POST("/folders", handler.CreateFolder)
			authed.GET("/folders/tree", handler.GetFolderTree)
			authed.GET("/folders/:id/files", handler.GetFolderFiles)
			authed.PUT("/folders/:id", handler.UpdateFolder)
			authed.DELETE("/folders/:id", handler.DeleteFolder)

			// 机器人管理
			authed.GET("/bots", handler.GetBots)
			authed.GET("/bots/templates", handler.GetTemplates)
			authed.GET("/bots/my", handler.GetMyBots)
			authed.GET("/bots/my-count", handler.GetMyBotCount)
			authed.POST("/bots", handler.CreateBot)
			authed.PUT("/bots/:id", handler.UpdateMyBot)
			authed.DELETE("/bots/:id", handler.DeleteMyBot)

			// 日历事件
			authed.GET("/events", handler.GetEvents)
			authed.POST("/events", handler.CreateEvent)
			authed.GET("/events/:id", handler.GetEvent)
			authed.PUT("/events/:id", handler.UpdateEvent)
			authed.DELETE("/events/:id", handler.DeleteEvent)

			// 系统消息
			authed.GET("/system-messages", handler.GetSystemMessages)
			authed.POST("/system-messages", middleware.RequireRole(di.GlobalContainer.UserService, "system_publisher", "system_admin"), handler.CreateSystemMessage)
			authed.PUT("/system-messages/:id", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.UpdateSystemMessage)

			// 频道
			authed.POST("/channels", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.CreateChannel)
			authed.GET("/channels", handler.GetChannels)
			authed.POST("/channels/:id/subscribe", handler.SubscribeChannel)
			authed.POST("/channels/:id/unsubscribe", handler.UnsubscribeChannel)
			authed.POST("/channels/:id/messages", handler.CreateChannelMessage)
			authed.GET("/channels/:id/messages", handler.GetChannelMessages)
			authed.POST("/channels/messages/:messageId/like", handler.LikeChannelMessage)
			authed.POST("/channels/messages/:messageId/unlike", handler.UnlikeChannelMessage)
			authed.GET("/channels/messages/:messageId/likes", handler.GetChannelMessageLikes)
			authed.POST("/channels/messages/:messageId/comments", handler.CommentChannelMessage)
			authed.GET("/channels/messages/:messageId/comments", handler.GetChannelMessageComments)

			// 消息管理
			authed.GET("/messages", handler.GetMessagesByFilter)

			// 小程序管理
			authed.GET("/mini-apps", handler.GetMiniApps)
			authed.GET("/mini-apps/:id", handler.GetMiniApp)
			authed.POST("/mini-apps", handler.CreateMiniApp)
			authed.PUT("/mini-apps/:id", handler.UpdateMiniApp)
			authed.DELETE("/mini-apps/:id", handler.DeleteMiniApp)

			// 应用管理
			authed.GET("/apps", handler.GetApps)
			authed.GET("/apps/all", handler.GetAllApps)
			authed.GET("/apps/built-in", handler.GetBuiltInApps)
			authed.POST("/apps", handler.CreateApp)
			authed.PUT("/apps/:id", handler.UpdateApp)
			authed.DELETE("/apps/:id", handler.DeleteApp)
			authed.PATCH("/apps/:id/toggle", handler.ToggleAppStatus)

			// 管理员应用管理
			authed.GET("/admin/apps", handler.GetAllApps)

			// 统计报表
			authed.GET("/statistics", handler.GetStatistics)

			// 通知管理
			authed.GET("/notifications", handler.GetNotifications)
			authed.PUT("/notifications/:id/read", handler.MarkNotificationAsRead)
			authed.PUT("/notifications/read-all", handler.MarkAllNotificationsAsRead)
			authed.DELETE("/notifications", handler.ClearAllNotifications)
			authed.PATCH("/notifications/:id/action", handler.HandleNotificationAction)
			authed.PATCH("/notifications/:id/pin", handler.TogglePinNotification)
			authed.PATCH("/notifications/:id/important", handler.ToggleImportantNotification)

			// 任务管理
			authed.GET("/tasks", handler.GetTasks)
			authed.POST("/tasks", handler.CreateTask)
			authed.PUT("/tasks/:id", handler.UpdateTask)
			authed.DELETE("/tasks/:id", handler.DeleteTask)
			authed.PATCH("/tasks/:id/status", handler.UpdateTaskStatus)
			// 实时通信 API
			realtime := authed.Group("/realtime")
			{
				realtime.POST("/sessions", handler.CreateSession)
				realtime.GET("/sessions/:id", handler.GetSession)
				realtime.GET("/sessions", handler.GetActiveSessions)
				realtime.POST("/sessions/:id/end", handler.EndSession)
				realtime.POST("/sessions/:id/participants", handler.RequestJoin)
				realtime.PATCH("/sessions/:id/participants/:user_id", handler.ApproveJoin)
				realtime.DELETE("/sessions/:id/participants/:user_id", handler.RejectJoin)
				realtime.DELETE("/sessions/:id/participants", handler.LeaveSession)
				// 离线共享请求
				realtime.GET("/pending-requests", handler.GetPendingRequests)
				realtime.POST("/pending-requests/:id/respond", handler.RespondToShareRequest)
			}
			// 短链接管理
			authed.POST("/shortlinks", handler.CreateShortLink)
			authed.GET("/shortlinks", handler.GetShortLinks)
			authed.POST("/shortlinks/batch", handler.BatchCreateShortLinks)
			authed.DELETE("/shortlinks/batch", handler.BatchDeleteShortLinks)
			authed.DELETE("/shortlinks/:id", handler.DeleteShortLink)

			// 用户搜索
			authed.GET("/users/search", handler.SearchUsers)
			// 敏感词管理
			handler.RegisterSensitiveWordRoutes(authed)

			// 管理后台路由（自动记录操作日志）
			adminRoutes := authed.Group("")
			adminRoutes.Use(middleware.OperationLogMiddleware())
			{
				// 系统配置
				adminRoutes.GET("/system/config", handler.GetSystemConfig)
				adminRoutes.PUT("/system/config", handler.UpdateSystemConfig)

				// 操作日志
				adminRoutes.GET("/logs/operation", handler.GetOperationLogs)
				adminRoutes.GET("/logs/operation/:id", handler.GetOperationLogDetail)
				adminRoutes.GET("/logs/operation/stats", handler.GetOperationLogStats)
				adminRoutes.GET("/logs/operation/export", handler.ExportOperationLogs)

				// 版本管理
				adminRoutes.GET("/client/versions", handler.GetVersions)
				adminRoutes.POST("/client/versions", handler.CreateVersion)
				adminRoutes.PUT("/client/versions/:id", handler.UpdateVersion)
				adminRoutes.DELETE("/client/versions/:id", handler.DeleteVersion)
				adminRoutes.PATCH("/client/versions/:id/toggle", handler.ToggleVersionStatus)

				// 黑名单管理
				adminRoutes.GET("/users/blacklist", handler.GetBlacklist)
				adminRoutes.POST("/users/blacklist", handler.AddToBlacklist)
				adminRoutes.DELETE("/users/blacklist/:id", handler.RemoveBlacklistEntry)
			}
			// 管理员接口（需要 system_admin 角色）
			admin := authed.Group("/admin")
			admin.Use(middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"))
			{
				admin.GET("/users", handler.AdminGetUsers)
				admin.GET("/groups", handler.AdminGetGroups)
				admin.DELETE("/groups/:id", handler.AdminDeleteGroup)
				admin.GET("/channels", handler.AdminGetChannels)
				admin.PUT("/channels/:id", handler.AdminUpdateChannel)
				admin.DELETE("/channels/:id", handler.AdminDeleteChannel)
				admin.GET("/statistics", handler.AdminGetStatistics)
				admin.GET("/recent-registrations", handler.AdminGetRecentRegistrations)
				admin.GET("/ai-usage-logs", handler.GetAIUsageLogs)

				// 认证提供者管理
				authProviderHandler := handler.NewAuthProviderHandler()
				admin.GET("/auth/providers", authProviderHandler.GetProviders)
				admin.POST("/auth/providers", authProviderHandler.CreateProvider)
				admin.PUT("/auth/providers/:id", authProviderHandler.UpdateProvider)
				admin.DELETE("/auth/providers/:id", authProviderHandler.DeleteProvider)
				admin.POST("/auth/providers/:id/test", authProviderHandler.TestProvider)

				// 角色管理
				admin.GET("/roles", handler.GetRoles)
				admin.POST("/roles", handler.CreateRole)
				admin.PUT("/roles/:id", handler.UpdateRole)
				admin.DELETE("/roles/:id", handler.DeleteRole)
				admin.GET("/roles/:role/users", handler.GetRoleUsers)

				// AI提供商管理
				admin.GET("/ai/providers", handler.GetProviders)
				admin.POST("/ai/providers", handler.CreateProvider)
				admin.PUT("/ai/providers/:id", handler.UpdateProvider)
				admin.DELETE("/ai/providers/:id", handler.DeleteProvider)
				admin.PATCH("/ai/providers/:id/status", handler.ToggleProviderStatus)
				admin.POST("/ai/providers/:id/test", handler.TestProviderConnection)

				// 组织架构同步管理
				orgSyncHandler := handler.NewOrgSyncHandler()
				admin.GET("/org/sync/configs", orgSyncHandler.GetConfigs)
				admin.POST("/org/sync/configs", orgSyncHandler.CreateConfig)
				admin.PUT("/org/sync/configs/:id", orgSyncHandler.UpdateConfig)
				admin.DELETE("/org/sync/configs/:id", orgSyncHandler.DeleteConfig)
				admin.POST("/org/sync/trigger/:id", orgSyncHandler.TriggerSync)
				admin.GET("/org/sync/logs", orgSyncHandler.GetLogs)

				// 文件存储管理
				admin.GET("/files/statistics", handler.GetAdminFileStatistics)
				admin.GET("/files/large", handler.GetAdminLargeFiles)

				// 统一审批 API
				approvalHandler := service.NewApprovalHandler(di.GlobalContainer.ApprovalService)
				approvalHandler.RegisterRoutes(admin)
			}

			// 节点间通信
			authed.POST("/node/broadcast", handler.BroadcastMessage)
			authed.POST("/node/send-to-user", handler.SendToUserMessage)

			// 用户角色管理
			authed.POST("/users/:id/roles", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.AddUserRole)
			authed.DELETE("/users/:id/roles/:role", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.RemoveUserRole)
			authed.POST("/users/:id/roles/batch", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.BatchAssignUserRoles)

			// 用户删除（管理员）
			authed.DELETE("/users/:id", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.DeleteUser)

			// AI相关路由
			userAIConfigHandler := handler.NewUserAIConfigHandler()
			userAIConfigHandler.RegisterRoutes(authed)
			aiHandler.RegisterRoutes(authed)

			// 分身服务路由
			avatarHandler := handler.NewAvatarHandler(GetDB(), avatarService, mcpServer)
			avatarHandler.RegisterRoutes(authed)

			// AI 运维面板（管理员）
			admin.GET("/ai/dashboard", func(c *gin.Context) {
				aiHandler.OpsDashboard(c)
			})

			// MCP 工具管理（管理员）
			admin.GET("/mcp/tools", aiHandler.ListMCPTools)
			admin.PUT("/mcp/tools/:tool_name", aiHandler.UpdateMCPToolConfig)

			// 知识图谱（管理员）
			admin.GET("/knowledge-graph", aiHandler.GetKnowledgeGraph)

			// 管理员操作用户AI配置
			admin.GET("/users/:id/ai-configs", handler.AdminGetUserAIConfigs)
			admin.PUT("/users/:id/ai-configs/:configId", handler.AdminUpdateUserAIConfig)

			// 向量数据库管理（管理员）
			admin.GET("/vector/collections", handler.AdminListVectorCollections)
			admin.GET("/vector/collections/:name", handler.AdminGetVectorCollectionData)
		}
	}

	// 短链接访问路由（不需要认证）
	r.GET("/:code", handler.RedirectShortLink)
}
