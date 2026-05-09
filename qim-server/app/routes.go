package app

import (
	"time"

	"qim-server/ai"
	"qim-server/config"
	"qim-server/di"
	"qim-server/handler"
	"qim-server/middleware"
	"qim-server/service"
	"qim-server/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// GetAIService returns the global AI service instance
func GetAIService() *ai.AIService {
	return di.GlobalContainer.AIService
}

// SetupRoutes 设置 API 路由
func SetupRoutes(r *gin.Engine, cfg *config.Config, hub *ws.Hub) {
	handler.SetConfig(cfg)

	aiSvc := di.GlobalContainer.AIService

	mcpServer := ai.NewMCPServer(false)

	aiSvc.SetMCPServer(mcpServer)

	handler.RegisterAdminTools(mcpServer)

	handler.InitSmartReplyEngine(aiSvc)

	handler.InitAnomalyDetector()

	go func() {
		if err := mcpServer.Start(":8081"); err != nil {
		}
	}()

	aiHandler := handler.NewAIHandler(aiSvc, mcpServer)

	avatarService := service.NewAvatarService(GetDB(), aiSvc)
	handler.SetAvatarWorkerPool(avatarService.GetWorkerPool())

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

	// 静态文件服务
	r.Static("/uploads", "./uploads")
	r.Static("/miniprograms", "./static/miniprograms")

	// 客户端更新检查（无需认证）
	r.GET("/api/v1/updates/:platform/latest.yml", handler.GetLatestYML)
	r.GET("/api/v1/updates/:platform/files/:filename", handler.GetUpdateFile)

	// 使用静态文件处理函数，并确保CORS中间件应用

	// API路由
	api := r.Group("/api/v1")
	{
		// 认证路由
		auth := api.Group("/auth")
		{
			auth.POST("/login", handler.Login)
			auth.POST("/register", handler.Register)
			auth.POST("/2fa/verify", handler.VerifyTwoFA)
			auth.POST("/check-2fa", handler.CheckTwoFAStatus)
			auth.POST("/check-version", handler.CheckVersion)
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

			// 群聊成员管理
			authed.POST("/conversations/:id/members", handler.AddMemberToGroup)
			// 移除群成员
			authed.DELETE("/conversations/:id/members/:user_id", handler.RemoveMemberFromGroup)
			// 退出群聊
			authed.POST("/conversations/:id/exit", handler.ExitGroup)
			// 申请加入群聊
			authed.POST("/conversations/:id/apply", handler.ApplyJoinGroup)
			// 拒绝加入请求
			authed.DELETE("/conversations/:id/join-requests/:user_id", handler.RejectJoinRequest)
			// 更新群聊信息
			authed.PUT("/conversations/:id", handler.UpdateGroupInfo)
			// 获取群聊 AI 设置
			authed.GET("/conversations/:id/ai-settings", handler.GetGroupAISettings)
			// 更新群聊 AI 设置
			authed.PUT("/conversations/:id/ai-settings", handler.UpdateGroupAISettings)
			// 群知识库管理
			authed.GET("/conversations/:id/ai-documents", handler.GetGroupDocuments)
			authed.POST("/conversations/:id/ai-documents", handler.AddGroupDocument)
			authed.DELETE("/conversations/:id/ai-documents/:file_id", handler.RemoveGroupDocument)
			// 设置/取消管理员
			authed.PUT("/conversations/:id/members/:user_id/role", handler.SetMemberRole)
			// 转让群主
			authed.POST("/conversations/:id/members/:user_id/transfer-owner", handler.TransferOwner)
			// 更新群公告
			authed.PUT("/conversations/:id/announcement", handler.UpdateAnnouncement)

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
			authed.DELETE("/files/:id", handler.DeleteFile)

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
		}
	}

	// 短链接访问路由（不需要认证）
	r.GET("/:code", handler.RedirectShortLink)
}
