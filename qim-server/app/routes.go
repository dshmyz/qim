package app

import (
	"qim-server/config"
	"qim-server/handler"
	"qim-server/middleware"
	"qim-server/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置 API 路由
func SetupRoutes(r *gin.Engine, cfg *config.Config, hub *ws.Hub) {
	// CORS配置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 静态文件服务
	r.Static("/uploads", "./uploads")

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
		authAuthed.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			authAuthed.POST("/logout", handler.Logout)
			authAuthed.POST("/refresh", handler.RefreshToken)
		}

		// 需要认证的路由
		authed := api.Group("")
		authed.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			// 用户
			authed.GET("/users/me", handler.GetCurrentUser)
			authed.PUT("/users/me", handler.UpdateUser)

			// 组织架构
			authed.GET("/organization/tree", handler.GetOrganizationTree)
			// 创建部门
			authed.POST("/departments", handler.CreateDepartment)
			// 创建用户
			authed.POST("/users", handler.CreateUser)
			// 关联用户和部门
			authed.POST("/department-employees", handler.AddUserToDepartment)

			// 会话
			authed.GET("/conversations", handler.GetConversations)
			authed.POST("/conversations/single", handler.CreateSingleConversation)
			authed.POST("/conversations/group", handler.CreateGroupConversation)
			authed.POST("/conversations/discussion", handler.CreateDiscussionConversation)
			authed.GET("/conversations/:id", handler.GetConversation)
			// 会话置顶/取消置顶
			authed.PUT("/conversations/:id/pin", handler.PinConversation)
			// 设置免打扰
			authed.PUT("/conversations/:id/mute", handler.SetConversationMute)

			// 消息
			authed.GET("/conversations/:id/messages", handler.GetMessages)
			authed.POST("/conversations/:id/messages", handler.SendMessage)
			authed.POST("/conversations/:id/read", handler.MarkConversationAsRead)
			authed.GET("/messages/:id/read-users", handler.GetMessageReadUsers)
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
			// 更新群聊信息
			authed.PUT("/conversations/:id", handler.UpdateGroupInfo)
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

			// 文件上传
			authed.POST("/upload", handler.UploadFile)

			// 文件管理
			authed.GET("/files", handler.GetFiles)
			authed.GET("/files/:id/download", handler.DownloadFile)
			authed.DELETE("/files/:id", handler.DeleteFile)

			// 笔记管理
			authed.GET("/notes", handler.GetNotes)
			authed.GET("/notes/:id", handler.GetNote)
			authed.POST("/notes", handler.CreateNote)
			authed.PUT("/notes/:id", handler.UpdateNote)
			authed.DELETE("/notes/:id", handler.DeleteNote)

			// 文件夹管理
			authed.POST("/folders", handler.CreateFolder)
			authed.GET("/folders/tree", handler.GetFolderTree)

			// 机器人管理
			authed.GET("/bots", handler.GetBots)

			// 日历事件
			authed.GET("/events", handler.GetEvents)
			authed.POST("/events", handler.CreateEvent)
			authed.GET("/events/:id", handler.GetEvent)
			authed.PUT("/events/:id", handler.UpdateEvent)
			authed.DELETE("/events/:id", handler.DeleteEvent)

			// 系统消息
			authed.GET("/system-messages", handler.GetSystemMessages)
			authed.POST("/system-messages", middleware.RequireRole("system_publisher", "system_admin"), handler.CreateSystemMessage)
			authed.PUT("/system-messages/:id", middleware.RequireRole("system_admin"), handler.UpdateSystemMessage)

			// 频道
			authed.POST("/channels", middleware.RequireRole("system_admin"), handler.CreateChannel)
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

			// 统计报表
			authed.GET("/statistics", handler.GetStatistics)

			// 通知管理
			authed.GET("/notifications", handler.GetNotifications)
			authed.PUT("/notifications/:id/read", handler.MarkNotificationAsRead)
			authed.PUT("/notifications/read-all", handler.MarkAllNotificationsAsRead)

			// 短链接管理
			authed.POST("/shortlinks", handler.CreateShortLink)
			authed.GET("/shortlinks", handler.GetShortLinks)
		}
	}

	// 短链接访问路由（不需要认证）
	r.GET("/:code", handler.RedirectShortLink)
}
