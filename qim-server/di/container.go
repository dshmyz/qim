package di

import (
	"fmt"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/middleware"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/scheduler"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/service/storage"
	syncpkg "github.com/dshmyz/qim/qim-server/sync"
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Container struct {
	DB                   *gorm.DB
	Config               *config.Config
	AIService            *ai.AIService
	UserService          *service.UserService
	ConversationService  *service.ConversationService
	MessageService       *service.MessageService
	NotificationService  *service.NotificationService
	EventService         *service.EventService
	TaskService          *service.TaskService
	FileService          *service.FileService
	FileSpaceService     *service.FileSpaceService
	StorageManager       *storage.Manager
	DefaultStorage       storage.Storage
	GroupService         *service.GroupService
	AppService           *service.AppService
	MiniAppService       *service.MiniAppService
	NoteService          *service.NoteService
	AdminService         *service.AdminService
	RealtimeService      *service.RealtimeService
	SensitiveWordService *service.SensitiveWordService
	AvatarService        *service.AvatarService
	ApprovalService      *service.ApprovalService
	VersionService       *service.VersionService
	BlacklistService     *service.BlacklistService
	OperationLogService  *service.OperationLogService
	SystemConfigService  *service.SystemConfigService
	ShortLinkService     *service.ShortLinkService
	ChannelService       *service.ChannelService
	RenderRuleService    *service.RenderRuleService
	BotService           *service.BotService
	AIProviderService    *service.AIProviderService
	GroupDocumentService *service.GroupDocumentService
	UserSettingService   *service.UserSettingService
	AIConfigService      *service.AIConfigService
	ChunkService         *service.ChunkService
	VectorService        *service.VectorService
	NoteVectorService    *service.NoteVectorService
	AvatarMemoryService  *service.AvatarMemoryService
	GroupMemoryService   *service.GroupMemoryService
	AvatarTriggerService *service.AvatarTriggerService
	PromptManager        *service.PromptManager
	WebSocketHub         *ws.Hub
	AuthMiddleware       gin.HandlerFunc
	// Scheduler 是统一调度器（robfig/cron/v3），由 main.go 在启动时注入
	Scheduler *scheduler.Scheduler
	// SyncEngine 是 OrgSync 引擎，由 main.go 在启动时注入
	SyncEngine *syncpkg.Engine
}

var GlobalContainer *Container

func InitContainer(cfg *config.Config, hub *ws.Hub) (*Container, error) {
	db := database.GetDB()

	aiService := ai.NewAIService(&cfg.AI)
	aiProviderService := service.NewAIProviderService(db)

	// 从数据库加载已启用的 AI Provider，覆盖配置文件中的设置
	// db 可能为 nil（测试环境未初始化），此时跳过 DB 加载
	if db != nil {
		if _, err := aiProviderService.ReloadEnabledProviders(aiService); err != nil {
			logger.WithModule("DI").Warn("从数据库加载 AI Provider 失败", "error", err)
		}

		// Token 用量异步落库：每次 LLM 调用完成后写入 ai_usage_logs 表
		aiService.SetUsageSink(func(taskType ai.TaskType, providerName, modelName string, usage *ai.TokenUsage, durationMs int64) {
			go func() {
				log := model.AIUsageLog{
					Provider:  providerName,
					Model:     modelName,
					TaskType:  string(taskType),
					CallType:  "chat",
					TokensIn:  usage.PromptTokens,
					TokensOut: usage.CompletionTokens,
					Duration:  durationMs,
					Status:    "success",
				}
				if err := db.Create(&log).Error; err != nil {
					logger.WithModule("AIUsage").Warn("token 用量落库失败", "error", err)
				}
			}()
		})
	}

	userService := service.NewUserService(db)
	conversationService := service.NewConversationService(db)
	messageService := service.NewMessageService(db, hub, aiService)
	notificationService := service.NewNotificationService(db)
	eventService := service.NewEventService(db)
	taskService := service.NewTaskService(db)
	fileService := service.NewFileService(db)

	var s3Svc *service.S3Service
	var defaultStorage storage.Storage
	var s3Storage storage.Storage
	var localStorage storage.Storage

	if cfg.Storage.Type == "s3" {
		var err error
		s3Svc, err = service.NewS3Service(cfg.Storage.S3)
		if err != nil {
			// fail-fast：配置了 type=s3 却不可用，拒绝启动，避免静默降级 local 导致数据写本地而运维无感
			logger.WithModule("DI").Error("S3 存储初始化失败，配置了 type=s3 但不可用，拒绝启动", "error", err)
			return nil, fmt.Errorf("S3 存储初始化失败: %w", err)
		}
		logger.WithModule("DI").Info("S3Service 初始化成功", "bucket", cfg.Storage.S3.Bucket)
		s3Storage = storage.NewS3Storage(s3Svc, cfg.Storage.S3)
		defaultStorage = s3Storage
	}

	localStorage, storageErr := storage.NewLocalStorage(cfg.Storage.Local)
	if storageErr != nil {
		logger.WithModule("DI").Warn("LocalStorage 初始化失败", "error", storageErr)
	}
	if defaultStorage == nil {
		defaultStorage = localStorage
	}

	storageManager := storage.NewManager(defaultStorage, s3Storage, localStorage)
	storageAccessor := NewStorageAccessor(storageManager)
	fileService.SetStorageAccessor(storageAccessor)
	fileSpaceService := service.NewFileSpaceService(db)
	fileSpaceService.SetStorageAccessor(storageAccessor)

	groupService := service.NewGroupService(db)
	appService := service.NewAppService(db)
	miniAppService := service.NewMiniAppService(db)
	noteService := service.NewNoteService(db)
	adminService := service.NewAdminService(db)
	realtimeService := service.NewRealtimeService(db)
	sensitiveWordService := service.NewSensitiveWordService(db)
	avatarService := service.NewAvatarService(db, aiService)
	approvalService := service.NewApprovalService(db)
	versionService := service.NewVersionService(db, storageAccessor)
	blacklistService := service.NewBlacklistService(db)
	operationLogService := service.NewOperationLogService(db)
	systemConfigService := service.NewSystemConfigService(db)
	shortLinkService := service.NewShortLinkService(db)
	channelService := service.NewChannelService(db)
	renderRuleService := service.NewRenderRuleService(db)
	botService := service.NewBotService(db)
	groupDocumentService := service.NewGroupDocumentService(db, storageAccessor)
	userSettingService := service.NewUserSettingService(db)
	aiConfigService := service.NewAIConfigService(db, ai.NewProviderFactory())

	// 初始化 ChunkService
	chunkStoragePath := cfg.Storage.Local.Path
	if chunkStoragePath == "" {
		chunkStoragePath = "./uploads/chunks"
	}
	chunkService := service.NewChunkService(db, chunkStoragePath, storageAccessor)

	authMiddleware := middleware.AuthMiddleware(cfg.JWT.Secret, userService)

	// 初始化 RAG 相关服务
	var vectorSvc *service.VectorService
	var noteVectorSvc *service.NoteVectorService
	var avatarMemorySvc *service.AvatarMemoryService
	var groupMemorySvc *service.GroupMemoryService
	// 智能触发只依赖 AI 与主数据库，不应受向量库是否可用影响。
	avatarTriggerSvc := service.NewAvatarTriggerService(aiService, db)

	vectorPath := cfg.Vector.Path
	embedder := service.NewCortexDBEmbedder(aiService)

	var err error
	logger.WithModule("DI").Info("开始初始化 VectorService", "path", vectorPath)
	vectorSvc, err = service.NewVectorService(vectorPath, embedder)
	if err != nil {
		logger.WithModule("DI").Warn("VectorService 初始化失败，RAG 功能将不可用", "error", err)
	} else {
		logger.WithModule("DI").Info("VectorService 初始化成功")
		noteVectorSvc = service.NewNoteVectorService(vectorSvc, aiService)
		avatarMemorySvc = service.NewAvatarMemoryService(vectorSvc, aiService)
		groupMemorySvc = service.NewGroupMemoryService(vectorSvc, aiService)
	}

	// 注入向量服务到相关服务
	if noteVectorSvc != nil {
		noteService.SetVectorService(noteVectorSvc)
		groupDocumentService.SetVectorServices(vectorSvc)
		avatarService.SetRAGServices(noteVectorSvc, avatarMemorySvc)
	}

	promptManager := service.NewPromptManager()

	container := &Container{
		DB:                   db,
		Config:               cfg,
		AIService:            aiService,
		UserService:          userService,
		ConversationService:  conversationService,
		MessageService:       messageService,
		NotificationService:  notificationService,
		EventService:         eventService,
		TaskService:          taskService,
		FileService:          fileService,
		FileSpaceService:     fileSpaceService,
		StorageManager:       storageManager,
		DefaultStorage:       defaultStorage,
		GroupService:         groupService,
		AppService:           appService,
		MiniAppService:       miniAppService,
		NoteService:          noteService,
		AdminService:         adminService,
		RealtimeService:      realtimeService,
		SensitiveWordService: sensitiveWordService,
		AvatarService:        avatarService,
		ApprovalService:      approvalService,
		VersionService:       versionService,
		BlacklistService:     blacklistService,
		OperationLogService:  operationLogService,
		SystemConfigService:  systemConfigService,
		ShortLinkService:     shortLinkService,
		ChannelService:       channelService,
		RenderRuleService:    renderRuleService,
		BotService:           botService,
		AIProviderService:    aiProviderService,
		GroupDocumentService: groupDocumentService,
		UserSettingService:   userSettingService,
		AIConfigService:      aiConfigService,
		ChunkService:         chunkService,
		VectorService:        vectorSvc,
		NoteVectorService:    noteVectorSvc,
		AvatarMemoryService:  avatarMemorySvc,
		GroupMemoryService:   groupMemorySvc,
		AvatarTriggerService: avatarTriggerSvc,
		PromptManager:        promptManager,
		WebSocketHub:         hub,
		AuthMiddleware:       authMiddleware,
	}

	GlobalContainer = container

	return container, nil
}
