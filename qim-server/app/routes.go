package app

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/handler"
	"github.com/dshmyz/qim/qim-server/middleware"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/mention"
	"github.com/dshmyz/qim/qim-server/pkg/upload"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/service/storage"
	syncpkg "github.com/dshmyz/qim/qim-server/sync"
	"github.com/dshmyz/qim/qim-server/web"
	"github.com/dshmyz/qim/qim-server/ws"

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

	// 校验 CORS 配置：AllowCredentials=true 时不允许 AllowedOrigins 含 "*"
	cfg.CORS.AllowCredentials = true
	corsAllowAllOrigins := cfg.ValidateCORS()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"version":   "2.0",
			"timestamp": time.Now().Unix(),
		})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 公开文档接口已迁至 VitePress 静态站点（/docs/*），不再提供 API

	aiSvc := di.GlobalContainer.AIService

	toolRegistry := ai.NewToolRegistry(aiSvc)

	aiSvc.SetToolRegistry(toolRegistry)

	handler.RegisterAdminTools(toolRegistry)

	// 外部 MCP 客户端网关：读 system_configs 的 external_mcp 配置并注册外部工具。
	// Sync 失败不阻塞启动——外部 server 不可达只影响对应工具，不影响主路径。
	mcpGateway := service.NewMCPClientGateway(
		di.GlobalContainer.SystemConfigService,
		toolRegistry,
	)
	mcpGateway.AllowPrivate = cfg.MCP.AllowPrivate
	mcpGateway.Sync()

	groupDocSvc := di.GlobalContainer.GroupDocumentService
	var uk *service.UnifiedKnowledgeService
	if vectorSvc := di.GlobalContainer.VectorService; vectorSvc != nil {
		service.NewUnifiedToolBridge(toolRegistry, vectorSvc.GetDB(), aiSvc)

		fallback := &service.LegacyKnowledgeFallback{
			SearchFunc: func(query string, groupID uint, limit int) []service.KnowledgeSnippet {
				return nil
			},
		}
		uk = service.NewUnifiedKnowledgeService(groupDocSvc, fallback, di.GlobalContainer.AiThresholdService, aiSvc)
	}

	handler.InitSmartReplyEngine(aiSvc)

	// 设置依赖（在 InitSmartReplyEngine 之后）
	if uk != nil {
		handler.SetUnifiedKnowledge(uk)
	}
	if avatarMemorySvc := di.GlobalContainer.AvatarMemoryService; avatarMemorySvc != nil {
		handler.SetMemoryService(avatarMemorySvc)
	}
	if groupMemorySvc := di.GlobalContainer.GroupMemoryService; groupMemorySvc != nil {
		handler.SetGroupMemoryService(groupMemorySvc)
	}

	// 初始化 SmartReplyGraph（使用 Eino 框架编排）
	if err := handler.InitSmartReplyGraph(); err != nil {
		logger.WithModule("Routes").Warn("初始化 SmartReplyGraph 失败，将使用旧方法", "error", err)
	}

	// 注入外部 MCP 客户端网关。必须在 InitSmartReplyGraph 之后调用——
	// SetMCPGateway 只在 smartReplyGraph 非 nil 时才会传播网关，先于初始化调用会被静默丢弃，
	// 导致普通提问 HasExternalTools() 恒为 false、外部工具永不生效。
	handler.SetMCPGateway(mcpGateway)

	aiHandler := handler.NewAIHandler(aiSvc, toolRegistry)

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

	// 统一上下文预制：侧边栏 current 模式的历史注入经它声明式装配（与 bot 笔记注入同一套抽象）。
	contextAsm := service.NewContextAssembler(di.GlobalContainer.DB)
	contextAsm.SetNoteSearcher(noteVectorSvc) // 可为 nil（向量库未配时安全降级）
	aiHandler.SetContextAssembler(contextAsm)

	// 注册用户侧 AI 工具（依赖 TaskService/MessageService/SearchGraph/SummaryGraph）
	service.RegisterUserTools(toolRegistry, di.GlobalContainer.TaskService, di.GlobalContainer.MessageService, unifiedSearchGraph, summaryGraph)

	// 给专属机器人 1:1 回复注入流式 AI 消息发送器，使其复用群 @AI 同款流式逐 token +
	// 工具调用基建（SendStreamingAIMessage / GetCompletionWithToolsStreamMultiStep / SendToolCallEvent）。
	// 与 SmartReplyEngine 同源 sender（同一 hub + UserService）；未注入则 bot 走非流式老路径降级。
	di.GlobalContainer.MessageService.SetStreamingAISender(
		handler.NewWebSocketMessageSender(ws.GlobalHub, di.GlobalContainer.UserService),
	)

	smartDigestGraph := service.NewSmartDigestGraph(aiSvc, aiCache)
	if err := smartDigestGraph.Build(); err != nil {
		logger.WithModule("Routes").Warn("Failed to build SmartDigestGraph", "error", err)
	} else {
		aiHandler.SetSmartDigestGraph(smartDigestGraph)
		logger.WithModule("Routes").Info("SmartDigestGraph 初始化成功")
	}

	avatarService := di.GlobalContainer.AvatarService
	aiHandler.SetAvatarService(avatarService) // 帮我回复草稿模式复用分身生成
	handler.SetAvatarWorkerPool(avatarService.GetWorkerPool())
	if avatarTriggerSvc := di.GlobalContainer.AvatarTriggerService; avatarTriggerSvc != nil {
		handler.GetSmartReplyEngine().SetAvatarTriggerService(avatarTriggerSvc)
	}

	// 注入 WebSocket 消息回调，使分身/智能回复在 WebSocket 发送消息时也触发
	hub.OnMessageSent = func(msg *model.Message, _ []uint) {
		sre := handler.GetSmartReplyEngine()
		senderID := msg.SenderID
		conversationID := msg.ConversationID
		content := msg.Content
		// @all 不触发 AI，避免噪音
		if mention.HasAnyMention(content) && mention.IsAllMentioned(mention.Parse(content)) {
			return
		}
		msgSvc := di.GlobalContainer.MessageService
		mentionUserIDs := msgSvc.MentionUserIDsForAI(conversationID, content)

		// 群聊外部 agent：群消息 @ 到已入群的 webhook bot 时转发给 agent
		if msgSvc != nil {
			msgSvc.HandleGroupBotMention(conversationID, senderID, mentionUserIDs, content)
		}

		// 待办提取：独立于智能回复，只看群聊 ExtractTodos 配置
		handler.TryExtractTodos(senderID, conversationID, content)

		// 智能回复：受 Enabled/ReplyMode 等控制。
		// 透传完整消息，使 AI 路径能读取引用消息（QuotedMessageID）以提取被引用文件正文。
		if sre != nil && msgSvc != nil {
			sre.HandleMessage(msg, mentionUserIDs)
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
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Node-Secret"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}
	if corsAllowAllOrigins {
		// 通配符模式：动态返回请求 Origin（兼容 AllowCredentials=true）
		corsConfig.AllowOriginFunc = func(origin string) bool {
			return true
		}
	} else {
		corsConfig.AllowOrigins = cfg.CORS.AllowedOrigins
	}
	corsMiddleware := cors.New(corsConfig)

	// 全局应用CORS中间件
	r.Use(corsMiddleware)

	// 全局中间件
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.LoggerMiddleware())

	// 请求限流（默认 500 请求/分钟/IP，可通过后台系统配置动态调整）
	rateLimiter := middleware.NewIPRateLimiter(500, time.Minute)
	middleware.SetGlobalIPRateLimiter(rateLimiter)
	// 从数据库加载已保存的速率限制配置
	middleware.ReloadRateLimitFromDB(func(key string) (string, error) {
		cfgSvc := di.GlobalContainer.SystemConfigService
		cfg, err := cfgSvc.GetConfig(key)
		if err != nil {
			return "", err
		}
		return cfg.Value, nil
	})
	r.Use(middleware.RateLimitMiddleware(rateLimiter))

	// 静态资源服务（统一入口，走 StorageManager，自动适配 local/s3 后端）
	// 路径格式：/static/<key>，如 /static/uploads/2026/01/xxx.png
	r.GET("/static/*filepath", func(c *gin.Context) {
		fp := c.Param("filepath")
		if strings.Contains(fp, "..") {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		serveStorageFile(c, storage.StaticPrefix+strings.TrimPrefix(fp, "/"))
	})
	// 【待清理 TBD-DEPRECATE】历史文件下载兼容路由，计划 2027-01 下线。
	// 背景：旧客户端/历史消息里存的是 /uploads/<key>，现统一迁移到 /static/ 前缀。
	// 保留此路由让旧格式 URL 仍可下载，无需逐个改写历史消息；ParsePath 会解析 /uploads/ 旧格式。
	// 下线前置条件：完成 scripts/migrate_message_storage_paths.sql 对历史消息 content 的迁移，
	// 并确认旧客户端无存量 /uploads/ URL 后，方可连同本路由一并移除。
	r.GET("/uploads/*filepath", func(c *gin.Context) {
		fp := c.Param("filepath")
		if strings.Contains(fp, "..") {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		serveStorageFile(c, "/uploads/"+strings.TrimPrefix(fp, "/"))
	})
	r.GET("/miniapps/*filepath", func(c *gin.Context) {
		fp := c.Param("filepath")
		if strings.Contains(fp, "..") {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		baseDir := cfg.Static.MiniAppsDir
		cleanPath := filepath.Clean(filepath.Join(baseDir, fp))
		if !strings.HasPrefix(cleanPath, baseDir) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// 如果路径以斜杠结尾，尝试访问 index.html
		if strings.HasSuffix(fp, "/") {
			cleanPath = filepath.Join(cleanPath, "index.html")
		}

		// 小程序每次打开都应由宿主强刷（前端带 ?_t= 缓存破坏），这里再声明 no-cache，
		// 避免浏览器在对同一 src 的重复加载中命中本地缓存而回不到服务端新版本。
		c.Header("Cache-Control", "no-cache")
		c.File(cleanPath)
	})

	// CLI 自动更新（无需认证）
	r.GET("/api/v1/cli/version", handler.CLIVersion)
	r.GET("/api/v1/cli/download", handler.CLIDownload)

	// 客户端更新检查（无需认证）
	// electron-updater 会请求 latest.yml 或 latest-{platform}.yml
	r.GET("/api/v1/updates/:platform/*action", handler.HandleUpdateRequest)
	// 更新服务健康检查
	r.GET("/api/v1/updates/health", handler.CheckUpdateHealth)

	// 公开文件下载（无需认证，用于客户端安装包等）
	r.GET("/api/v1/public/files/:id/download", handler.PublicDownloadFile)

	// WebSocket 路由（无需 HTTP 认证，通过首条消息认证）
	r.GET("/api/v1/ws", func(c *gin.Context) {
		ws.ServeWs(hub, c)
	})
	r.GET("/api/v1/screen-share", func(c *gin.Context) {
		ws.ServeScreenShare(hub, c)
	})

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
		// 从数据库加载已保存的登录限流配置
		middleware.ReloadRateLimitFromDB(func(key string) (string, error) {
			cfgSvc := di.GlobalContainer.SystemConfigService
			cfg, err := cfgSvc.GetConfig(key)
			if err != nil {
				return "", err
			}
			return cfg.Value, nil
		})
		{
			auth.POST("/login", middleware.LoginRateLimitMiddleware(loginLimiter), handler.Login)
			auth.POST("/register", handler.Register)
			auth.POST("/2fa/verify", middleware.LoginRateLimitMiddleware(loginLimiter), handler.VerifyTwoFA)
			auth.POST("/check-2fa", handler.CheckTwoFAStatus)
			// 公开的认证提供者列表（无需认证）
			authProviderHandler := handler.NewAuthProviderHandler()
			auth.GET("/providers", authProviderHandler.GetProviders)
			auth.GET("/providers/:name/login-url", authProviderHandler.GetProviderLoginURL)

			// 统一认证回调（前端统一调用，无需认证）
			auth.POST("/callback", handler.UnifiedAuthCallback)
		}

		// 客户端版本查询（公开，无需认证）
		api.GET("/client/versions", handler.GetVersions)

		// Bot API：外部 agent 出站消息（Bot 令牌鉴权，非 JWT）
		botAPIHandler := handler.NewBotAPIHandler(service.NewBotMessagingService(GetDB(), hub))
		// 600/min：agent 典型 1~3s 轮询 + 流式分段（stream-stdin 每行一次 POST），60/min 会被打满。
		botAPI := api.Group("/bot", middleware.BotAuthMiddleware(), middleware.BotRateLimitMiddleware(middleware.NewBotRateLimiter(600, time.Minute)))
		botAPI.POST("/messages", botAPIHandler.SendMessage)
		botAPI.GET("/messages", botAPIHandler.GetBotMessages)
		botAPI.GET("/groups", botAPIHandler.ListBotGroups)
		botAPI.POST("/messages/:id/stream", botAPIHandler.StreamChunk)
		botAPI.PUT("/messages/:id", botAPIHandler.UpdateMessage)

		// 需要认证的认证相关路由（access token）
		authAuthed := api.Group("/auth")
		authAuthed.Use(middleware.AuthMiddleware(cfg.JWT.Secret, di.GlobalContainer.UserService))
		{
			authAuthed.POST("/logout", handler.Logout)
			// refresh-token 用于刷新第三方 OAuth provider 的 access_token，请求体携带的是
			// 第三方 refresh_token，与 QIM JWT 无关，需要 access token 认证用户身份，故留在 authAuthed 组
			authAuthed.POST("/refresh-token", handler.RefreshOAuthToken)
		}

		// refresh token 端点：仅允许 refresh token，access token 不可用。
		// 修复破坏性问题：原先 /refresh 在 authAuthed 组下用 AuthMiddleware，
		// AuthMiddleware 拒绝 refresh token（TokenType="refresh" != "access"），
		// 导致 token 刷新完全断裂，用户 access token 过期后必须重新登录。
		refreshAuthed := api.Group("/auth")
		refreshAuthed.Use(middleware.RefreshAuthMiddleware(cfg.JWT.Secret, di.GlobalContainer.UserService))
		{
			refreshAuthed.POST("/refresh", handler.RefreshToken)
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
			// AI配置
			authed.GET("/ai/config", handler.GetAIConfig)
			authed.PUT("/ai/config", handler.UpdateAIConfig)

			// 组织架构
			authed.GET("/organization/tree", handler.GetOrganizationTree)
			// 获取部门员工
			authed.GET("/departments/:id/employees", handler.GetDepartmentEmployees)
			// 创建用户（管理员）
			authed.POST("/users", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.CreateUser)
			// 关联用户和部门
			authed.POST("/department-employees", handler.AddUserToDepartment)

			// 会话
			authed.GET("/conversations", handler.GetConversations)
			authed.POST("/conversations", handler.CreateConversation)

			// 当前用户的群聊列表
			authed.GET("/users/groups", handler.GetUserGroups)

			authed.GET("/conversations/:id", handler.GetConversation)
			// 搜索会话（按群名或单聊对方昵称）
			authed.GET("/conversations/search", handler.SearchConversations)
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
			handler.RegisterGroupFileRoutes(authed)
			authed.POST("/groups/:id/members", handler.AddMemberToGroup)
			// 拉外部 agent bot 进群（建 BotConversation 关联，使 @ 触发/群内出站可工作）
			authed.POST("/groups/:id/bots", handler.AddBotToGroup)
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
			// 群助手群级记忆管理
			authed.GET("/groups/:id/group-memories", handler.GetGroupMemories)
			authed.DELETE("/groups/:id/group-memories/:memory_id", handler.DeleteGroupMemory)
			authed.PUT("/groups/:id/group-memories/:memory_id", handler.UpdateGroupMemory)
			authed.DELETE("/groups/:id/group-memories", handler.ClearGroupMemories)
			authed.POST("/groups/:id/group-memories/search", handler.SearchGroupMemories)
			// 群知识库管理（带处理状态）
			authed.GET("/groups/:id/ai-documents", handler.GetGroupDocumentsWithStatus)
			authed.POST("/groups/:id/ai-documents", handler.AddGroupDocument)
			authed.DELETE("/groups/:id/ai-documents/:file_id", handler.RemoveGroupDocument)
			authed.POST("/groups/:id/ai-documents/:file_id/process", handler.ProcessGroupDocument)
			authed.GET("/groups/:id/ai-documents/:file_id/status", handler.GetDocumentProcessStatus)
			authed.POST("/groups/:id/ai-documents/batch-process", handler.BatchProcessDocuments)
			authed.POST("/groups/:id/ai-documents/batch-retry", handler.BatchRetryDocuments)
			// 群知识图谱（非管理员，群成员可查看自己群的知识图谱）
			authed.GET("/groups/:id/knowledge-graph", handler.GetGroupKnowledgeGraph)
			// 设置/取消管理员
			authed.PUT("/groups/:id/members/:user_id/role", handler.SetMemberRole)
			// 转让群主
			authed.POST("/groups/:id/members/:user_id/transfer-owner", handler.TransferOwner)
			// 更新群公告
			authed.PUT("/groups/:id/announcement", handler.UpdateAnnouncement)
			// 解散群聊
			authed.DELETE("/groups/:id", handler.DissolveGroup)

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
			authed.PATCH("/notes/:id/access", handler.SetNoteAiAccessible)
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
			// Bot 令牌与配置管理（创建者或 system_admin）
			authed.GET("/bots/:id/tokens", botAPIHandler.ListBotTokens)
			authed.POST("/bots/:id/token", botAPIHandler.IssueToken)
			authed.DELETE("/bots/:id/token/:tid", botAPIHandler.RevokeToken)
			authed.PUT("/bots/:id/config", botAPIHandler.UpdateBotConfig)
			// 用户长期令牌（qim CLI / qim-mcp 经 Bearer 以本人身份调用用户 API）
			authed.GET("/user-tokens", handler.ListUserTokens)
			authed.POST("/user-tokens", handler.IssueUserToken)
			authed.DELETE("/user-tokens/:tid", handler.RevokeUserToken)
			// Bot 卡片按钮回调（JWT 用户鉴权：点击按钮的人类用户，非 bot 令牌）
			authed.POST("/messages/:id/card-action", botAPIHandler.SubmitCardAction)

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
			authed.DELETE("/system-messages/:id", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.DeleteSystemMessage)
			// 全员私聊：以系统账号向用户单聊会话发送普通私聊消息（进最近会话），仅 system_admin
			authed.POST("/system-messages/broadcast-chat", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.BroadcastChatMessage)

			// 频道
			authed.POST("/channels", handler.CreateChannel)
			authed.GET("/channels", handler.GetChannels)
			authed.POST("/channels/:id/subscribe", handler.SubscribeChannel)
			authed.DELETE("/channels/:id/subscribe", handler.UnsubscribeChannel)
			authed.POST("/channels/:id/messages", handler.CreateChannelMessage)
			authed.GET("/channels/:id/messages", handler.GetChannelMessages)
			authed.POST("/channels/messages/:messageId/like", handler.LikeChannelMessage)
			authed.DELETE("/channels/messages/:messageId/like", handler.UnlikeChannelMessage)
			authed.GET("/channels/messages/:messageId/likes", handler.GetChannelMessageLikes)
			authed.POST("/channels/messages/:messageId/comments", handler.CommentChannelMessage)
			authed.GET("/channels/messages/:messageId/comments", handler.GetChannelMessageComments)

			// 消息管理
			authed.GET("/messages", handler.GetMessagesByFilter)

			// 小程序管理
			authed.GET("/mini-apps", handler.GetMiniApps)
			authed.GET("/mini-apps/:id", handler.GetMiniApp)

			// 应用管理
			authed.GET("/apps", handler.GetApps)
			authed.GET("/apps/all", handler.GetAllApps)
			authed.GET("/apps/built-in", handler.GetBuiltInApps)
			authed.POST("/apps", handler.CreateApp)
			authed.PUT("/apps/:id", handler.UpdateApp)
			authed.DELETE("/apps/:id", handler.DeleteApp)
			authed.PATCH("/apps/:id/toggle", handler.ToggleAppStatus)

			// 统计报表
			authed.GET("/statistics", handler.GetStatistics)

			// 通知管理
			authed.GET("/notifications", handler.GetNotifications)
			authed.PUT("/notifications/:id/read", handler.MarkNotificationAsRead)
			authed.PUT("/notifications/:id/unread", handler.MarkNotificationAsUnread)
			authed.PATCH("/notifications/:id", handler.PatchNotification)
			authed.DELETE("/notifications/:id", handler.DeleteNotification)

			authed.PUT("/notifications/read-all", handler.MarkAllNotificationsAsRead)
			authed.DELETE("/notifications", handler.ClearAllNotifications)

			// 任务管理
			authed.GET("/tasks", handler.GetTasks)
			authed.GET("/tasks/:id", handler.GetTaskByID)
			authed.POST("/tasks", handler.CreateTask)
			authed.PUT("/tasks/:id", handler.UpdateTask)
			authed.DELETE("/tasks/:id", handler.DeleteTask)
			authed.PUT("/tasks/:id/reorder", handler.ReorderTask)
			authed.PATCH("/tasks/:id/status", handler.UpdateTaskStatus)

			// 消息渲染增强规则（登录用户可拉取，管理后台维护）
			authed.GET("/render-rules", handler.GetRenderRules)

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
			// 敏感词检查（普通用户可用，用于前端预览）
			authed.POST("/sensitive-words/check", handler.CheckSensitiveWords)

			// 管理后台路由（需要 system_admin 角色，自动记录操作日志）
			adminRoutes := authed.Group("")
			adminRoutes.Use(middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"))
			adminRoutes.Use(middleware.OperationLogMiddleware())
			{
				// 敏感词管理（仅管理员）
				handler.RegisterSensitiveWordRoutes(adminRoutes)
				// 部门管理（仅管理员）
				adminRoutes.POST("/departments", handler.CreateDepartment)
				adminRoutes.PUT("/departments/:id", handler.UpdateDepartment)
				adminRoutes.PUT("/departments/:id/move", handler.MoveDepartment)
				adminRoutes.DELETE("/departments/:id", handler.DeleteDepartment)
				// 从部门移除员工（仅管理员）
				adminRoutes.DELETE("/department-employees/:id/:user_id", handler.RemoveEmployeeFromDepartment)
				// 小程序管理（仅管理员）
				adminRoutes.POST("/mini-apps", handler.CreateMiniApp)
				adminRoutes.PUT("/mini-apps/:id", handler.UpdateMiniApp)
				adminRoutes.DELETE("/mini-apps/:id", handler.DeleteMiniApp)

				// 系统配置
				adminRoutes.GET("/system/config", handler.GetSystemConfig)
				adminRoutes.PUT("/system/config", handler.UpdateSystemConfig)

				// AI 阈值配置
				adminRoutes.GET("/ai/thresholds", handler.GetAIThresholds)
				adminRoutes.GET("/ai/thresholds/schema", handler.GetAIThresholdSchema)
				adminRoutes.PUT("/ai/thresholds", handler.UpdateAIThresholds)

				// 操作日志
				adminRoutes.GET("/logs/operation", handler.GetOperationLogs)
				adminRoutes.GET("/logs/operation/:id", handler.GetOperationLogDetail)
				adminRoutes.GET("/logs/operation/stats", handler.GetOperationLogStats)
				adminRoutes.GET("/logs/operation/export", handler.ExportOperationLogs)

				// 版本管理
				adminRoutes.POST("/client/versions", handler.CreateVersion)
				adminRoutes.PUT("/client/versions/:id", handler.UpdateVersion)
				adminRoutes.DELETE("/client/versions/:id", handler.DeleteVersion)
				adminRoutes.PATCH("/client/versions/:id/toggle", handler.ToggleVersionStatus)
				adminRoutes.POST("/client/versions/:id/rollback", handler.RollbackVersion)
				adminRoutes.GET("/client/versions/distribution", func(c *gin.Context) {
					c.Set("hub", hub)
					handler.GetVersionDistribution(c)
				})

				// CLI 版本管理（复用 VersionService，app_type="cli"）
				adminRoutes.GET("/cli/versions", handler.GetCLIVersions)
				adminRoutes.POST("/cli/versions", handler.CreateCLIVersion)
				adminRoutes.PUT("/cli/versions/:id", handler.UpdateCLIVersion)
				adminRoutes.DELETE("/cli/versions/:id", handler.DeleteCLIVersion)
				adminRoutes.PATCH("/cli/versions/:id/toggle", handler.ToggleCLIVersionStatus)

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
				admin.PUT("/users/:id", handler.AdminUpdateUser)
				admin.GET("/groups", handler.AdminGetGroups)
				admin.POST("/groups", handler.AdminCreateGroup)
				admin.PUT("/groups/:id", handler.AdminUpdateGroup)
				admin.DELETE("/groups/:id", handler.AdminDeleteGroup)
				admin.GET("/groups/:id/members", handler.AdminGetGroupMembers)
				admin.GET("/conversations", handler.AdminGetConversations)
				admin.GET("/conversations/:id/members", handler.AdminGetConversationMembers)
				admin.DELETE("/conversations/:id", handler.AdminDeleteConversation)
				admin.GET("/messages/search", handler.AdminSearchMessages)
				admin.GET("/statistics", handler.AdminGetStatistics)
				admin.GET("/statistics/trend", handler.AdminGetStatisticsTrend)
				admin.GET("/dashboard/stats", handler.AdminGetDashboardStats)
				admin.GET("/dashboard/trend", handler.AdminGetDashboardTrend)
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
				admin.GET("/ai/providers", handler.GetAIProviders)
				admin.POST("/ai/providers", handler.CreateAIProvider)
				admin.PUT("/ai/providers/:id", handler.UpdateAIProvider)
				admin.DELETE("/ai/providers/:id", handler.DeleteAIProvider)
				admin.PATCH("/ai/providers/:id/status", handler.ToggleAIProviderStatus)
				admin.POST("/ai/providers/:id/test", handler.TestAIProviderConnection)

				// AI模型路由管理
				admin.GET("/ai/router", handler.GetAIRouter)
				admin.PUT("/ai/router", handler.SaveAIRouter)
				admin.DELETE("/ai/router", handler.ClearAIRouter)

				// 组织架构同步管理
				orgSyncHandler := handler.NewOrgSyncHandler()
				// 注入 reload 回调：在 Create/Update/Delete 后立即让调度器重载 OrgSyncConfig
				// 这样管理员修改调度配置后无需重启服务即可生效
				if di.GlobalContainer.Scheduler != nil && di.GlobalContainer.SyncEngine != nil {
					orgSyncHandler.SetReloadFn(func() error {
						return syncpkg.ReloadOrgSyncJobs(
							di.GlobalContainer.Scheduler,
							di.GlobalContainer.SyncEngine,
							di.GlobalContainer.DB,
						)
					})
				}
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

				// 外部 Bot 运维：外部 agent bot 列表 + webhook 投递监控/重投
				admin.GET("/bots/external", handler.AdminGetExternalBots)
				admin.GET("/webhook-deliveries", handler.AdminGetWebhookDeliveries)
				admin.GET("/webhook-deliveries/:id", handler.AdminGetWebhookDelivery)
				admin.POST("/webhook-deliveries/:id/redeliver", handler.AdminRedeliverWebhook)

				// 外部 MCP 工具预览（管理后台勾选用）
				admin.POST("/external-mcp/tools", handler.PreviewMCPTools)

				// 监控相关 API
				monitorHandler := handler.NewMonitorHandler()
				alertHandler := handler.NewAlertHandler(GetDB())
				monitor := admin.Group("/monitor")
				{
					monitor.GET("/server", monitorHandler.GetServerMetrics)
					monitor.GET("/server/history", monitorHandler.GetServerMetricsHistory)
					monitor.GET("/services", monitorHandler.GetServiceStatus)
					monitor.POST("/services/health-check", monitorHandler.HealthCheck)
					monitor.GET("/alerts", alertHandler.GetAlertRules)
					monitor.POST("/alerts", alertHandler.CreateAlertRule)
					monitor.PUT("/alerts/:id", alertHandler.UpdateAlertRule)
					monitor.DELETE("/alerts/:id", alertHandler.DeleteAlertRule)
					monitor.GET("/alerts/history", alertHandler.GetAlertHistory)
				}

				// 崩溃日志和用户反馈
				crashHandler := handler.NewCrashLogHandler(GetDB())
				feedbackHandler := handler.NewFeedbackHandler(GetDB())
				admin.GET("/crashes", crashHandler.GetCrashLogs)
				admin.GET("/crashes/:id", crashHandler.GetCrashDetail)
				admin.GET("/feedbacks", feedbackHandler.GetFeedbacks)
				admin.PUT("/feedbacks/:id", feedbackHandler.UpdateFeedback)
				authed.POST("/feedbacks", feedbackHandler.CreateFeedback)
				authed.GET("/my-feedbacks", feedbackHandler.GetMyFeedbacks)
			}

			// 频道管理后台接口（system_admin 或 channel_manager）
			adminChannel := authed.Group("/admin")
			adminChannel.Use(middleware.RequireRole(di.GlobalContainer.UserService, "system_admin", "channel_manager"))
			{
				adminChannel.GET("/channels", handler.AdminGetChannels)
				adminChannel.PUT("/channels/:id", handler.AdminUpdateChannel)
				adminChannel.DELETE("/channels/:id", handler.AdminDeleteChannel)
			}

			// 节点间通信（需要节点内部认证）
			// 注意：必须挂在 api 组而非 authed 组——跨节点请求只携带 Node-Secret，
			// 挂在 authed 下会被 AuthMiddleware 以无 Bearer token 为由先 401，
			// NodeAuthMiddleware 永远执行不到，导致跨节点传输全线不通。
			node := api.Group("", middleware.NodeAuthMiddleware(cfg.Node.Secret))
			node.POST("/node/broadcast", handler.BroadcastMessage)
			node.POST("/node/send-to-user", handler.SendToUserMessage)

			// 用户角色管理
			authed.DELETE("/users/:id/roles/:role", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.RemoveUserRole)
			authed.PUT("/users/:id/roles", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.BatchAssignUserRoles)

			// 用户删除（管理员）
			authed.DELETE("/users/:id", middleware.RequireRole(di.GlobalContainer.UserService, "system_admin"), handler.DeleteUser)

			// AI相关路由
			userAIConfigHandler := handler.NewUserAIConfigHandler()
			userAIConfigHandler.RegisterRoutes(authed)
			aiHandler.RegisterRoutes(authed)

			// 用户个人设置（通用 key-value，如 quick_replies、quick_command_panel_enabled）
			userSettingHandler := handler.NewUserSettingHandler()
			userSettingHandler.RegisterRoutes(authed)

			// AI 机器人管理
			aiBotHandler := handler.NewAIBotHandler(GetDB())
			aiBots := authed.Group("/ai-bots")
			{
				aiBots.GET("", aiBotHandler.GetAIBots)
				aiBots.POST("", aiBotHandler.CreateAIBot)
				aiBots.PUT("/:id", aiBotHandler.UpdateAIBot)
				aiBots.DELETE("/:id", aiBotHandler.DeleteAIBot)
				aiBots.PATCH("/:id/status", aiBotHandler.ToggleAIBotStatus)
			}

			// 分身服务路由
			avatarHandler := handler.NewAvatarHandler(GetDB(), avatarService, toolRegistry, di.GlobalContainer.ApprovalService)
			avatarHandler.RegisterRoutes(authed)

			// AI 运维面板（管理员）
			admin.GET("/ai/dashboard", func(c *gin.Context) {
				aiHandler.OpsDashboard(c)
			})

			// AI 工具注册表管理（管理员）
			admin.GET("/tool-registry/tools", aiHandler.ListToolRegistryTools)
			admin.PUT("/tool-registry/tools/:tool_name", aiHandler.UpdateToolRegistryConfig)

			// 知识图谱（管理员）
			admin.GET("/knowledge-graph", aiHandler.GetKnowledgeGraph)

			// 管理员操作用户分身配置
			admin.GET("/users/:id/avatar-config", handler.AdminGetUserAvatarConfig)
			admin.PUT("/users/:id/avatar-config", handler.AdminUpdateUserAvatarConfig)

			// 向量数据库管理（管理员）
			admin.GET("/vector/collections", handler.AdminListVectorCollections)
			admin.GET("/vector/collections/:name", handler.AdminGetVectorCollectionData)

			// 消息渲染增强规则管理（管理员）
			admin.GET("/render-rules", handler.AdminGetRenderRules)
			admin.PUT("/render-rules", handler.AdminSaveRenderRules)
			admin.POST("/render-rules/test", handler.AdminTestRenderRule)
		}
	}

	// 短链接访问路由（不需要认证）
	r.GET("/s/:code", handler.RedirectShortLink)

	// 管理后台 SPA（必须放在 API 路由之后，NoRoute 之前）
	// 老链接 /admin/docs/cli、/admin/docs/mcp 重定向到新 /docs/*
	// 注意：gin 不允许 catch-all /admin/*filepath 与同层级静态路由 /admin/docs/* 共存，
	// 故把重定向逻辑合并进 catch-all handler，避免路由树 panic。
	r.GET("/admin/*filepath", func(c *gin.Context) {
		fp := c.Param("filepath")
		if fp == "/docs/cli" {
			c.Redirect(http.StatusMovedPermanently, "/docs/cli")
			return
		}
		if fp == "/docs/mcp" {
			c.Redirect(http.StatusMovedPermanently, "/docs/mcp")
			return
		}
		web.ServeAdmin()(c)
	})

	// VitePress 静态文档站点（/docs/cli、/docs/mcp 等）
	r.GET("/docs/*filepath", web.ServeDocs())

	// Landing 首页 SPA（只处理非 API 请求，避免覆盖 API 的 404 响应）
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "接口不存在",
			})
			return
		}
		web.ServeLanding()(c)
	})
}

// serveStorageFile 通过 StorageManager 根据 storagePath 读取并输出文件。
// 被 /static/* 与 /uploads/* 两个路由共用；ParsePath 同时兼容新旧格式前缀。
func serveStorageFile(c *gin.Context, storagePath string) {
	mgr := di.GlobalContainer.StorageManager
	st, key, ok := mgr.ByPath(storagePath)
	if !ok || st == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	reader, err := st.Get(ctx, key)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	defer reader.Close()
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	if ct := mime.TypeByExtension(filepath.Ext(storagePath)); ct != "" {
		c.Header("Content-Type", ct)
	}
	// 危险类型（html/svg/js等）强制下载，防止存储型 XSS
	if upload.ShouldForceDownload(storagePath) {
		c.Header("Content-Disposition", "attachment")
	}
	// 设置 Content-Length：否则 Go 对该大小未知的流会回退到 Transfer-Encoding: chunked，
	// Electron 下载器的 getTotalBytes() 读到 0，无法计算下载进度（前端进度条卡 0%）。
	if size, ok := storageObjectSize(reader); ok && size >= 0 {
		c.Header("Content-Length", strconv.FormatInt(size, 10))
	}
	if _, err := io.Copy(c.Writer, reader); err != nil {
		return
	}
}

// storageObjectSize 尝试获取存储流的总大小（字节）。优先用 fs.File.Stat（不改动读游标），
// 兜底用 io.Seeker 跳到末尾量长度再复位。无法获知时返回 ok=false，调用方不设 Content-Length。
func storageObjectSize(r io.Reader) (int64, bool) {
	if f, ok := r.(fs.File); ok {
		if fi, err := f.Stat(); err == nil {
			return fi.Size(), true
		}
	}
	if s, ok := r.(io.Seeker); ok {
		cur, err := s.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, false
		}
		end, err := s.Seek(0, io.SeekEnd)
		if err != nil {
			return 0, false
		}
		if _, err := s.Seek(cur, io.SeekStart); err != nil {
			return 0, false
		}
		return end, true
	}
	return 0, false
}
