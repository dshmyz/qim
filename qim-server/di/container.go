package di

import (
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/middleware"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/service/storage"
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
	AuthService          *service.AuthService
	VersionService       *service.VersionService
	BlacklistService     *service.BlacklistService
	OperationLogService  *service.OperationLogService
	SystemConfigService  *service.SystemConfigService
	ShortLinkService     *service.ShortLinkService
	ChannelService       *service.ChannelService
	BotService           *service.BotService
	AIProviderService    *service.AIProviderService
	GroupDocumentService *service.GroupDocumentService
	AIConfigService      *service.AIConfigService
	ChunkService         *service.ChunkService
	VectorService        *service.VectorService
	NoteVectorService    *service.NoteVectorService
	AvatarMemoryService  *service.AvatarMemoryService
	AvatarTriggerService *service.AvatarTriggerService
	PromptManager        *service.PromptManager
	WebSocketHub         *ws.Hub
	AuthMiddleware       gin.HandlerFunc
}

var GlobalContainer *Container

func InitContainer(cfg *config.Config, hub *ws.Hub) *Container {
	db := database.GetDB()

	aiService := ai.NewAIService(&cfg.AI)

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
			logger.WithModule("DI").Warn("S3Service 初始化失败，S3 存储功能将不可用", "error", err)
		} else {
			logger.WithModule("DI").Info("S3Service 初始化成功", "bucket", cfg.Storage.S3.Bucket)
			s3Storage = storage.NewS3Storage(s3Svc, cfg.Storage.S3)
			defaultStorage = s3Storage
		}
	}

	localStorage, storageErr := storage.NewLocalStorage(cfg.Storage.Local)
	if storageErr != nil {
		logger.WithModule("DI").Warn("LocalStorage 初始化失败", "error", storageErr)
	}
	if defaultStorage == nil {
		defaultStorage = localStorage
	}

	storageManager := storage.NewManager(defaultStorage, s3Storage, localStorage)

	groupService := service.NewGroupService(db)
	appService := service.NewAppService(db)
	miniAppService := service.NewMiniAppService(db)
	noteService := service.NewNoteService(db)
	adminService := service.NewAdminService(db)
	realtimeService := service.NewRealtimeService(db)
	sensitiveWordService := service.NewSensitiveWordService(db)
	avatarService := service.NewAvatarService(db, aiService)
	approvalService := service.NewApprovalService(db)
	authService := service.NewAuthService(db, cfg.JWT.Secret)
	versionService := service.NewVersionService(db)
	blacklistService := service.NewBlacklistService(db)
	operationLogService := service.NewOperationLogService(db)
	systemConfigService := service.NewSystemConfigService(db)
	shortLinkService := service.NewShortLinkService(db)
	channelService := service.NewChannelService(db)
	botService := service.NewBotService(db)
	aiProviderService := service.NewAIProviderService(db)
	groupDocumentService := service.NewGroupDocumentService(db)
	aiConfigService := service.NewAIConfigService(db, ai.NewProviderFactory())

	// 初始化 ChunkService
	chunkStoragePath := cfg.Storage.Local.Path
	if chunkStoragePath == "" {
		chunkStoragePath = "./uploads/chunks"
	}
	chunkService := service.NewChunkService(db, chunkStoragePath)

	authMiddleware := middleware.AuthMiddleware(cfg.JWT.Secret, userService)

	// 初始化 RAG 相关服务
	var vectorSvc *service.VectorService
	var noteVectorSvc *service.NoteVectorService
	var avatarMemorySvc *service.AvatarMemoryService
	var avatarTriggerSvc *service.AvatarTriggerService

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
		avatarTriggerSvc = service.NewAvatarTriggerService(aiService, db)
	}

	// 注入向量服务到相关服务
	if noteVectorSvc != nil {
		noteService.SetVectorService(noteVectorSvc)
		groupDocumentService.SetVectorServices(vectorSvc)
		avatarService.SetRAGServices(noteVectorSvc, avatarMemorySvc, avatarTriggerSvc)
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
		AuthService:          authService,
		VersionService:       versionService,
		BlacklistService:     blacklistService,
		OperationLogService:  operationLogService,
		SystemConfigService:  systemConfigService,
		ShortLinkService:     shortLinkService,
		ChannelService:       channelService,
		BotService:           botService,
		AIProviderService:    aiProviderService,
		GroupDocumentService: groupDocumentService,
		AIConfigService:      aiConfigService,
		ChunkService:         chunkService,
		VectorService:        vectorSvc,
		NoteVectorService:    noteVectorSvc,
		AvatarMemoryService:  avatarMemorySvc,
		AvatarTriggerService: avatarTriggerSvc,
		PromptManager:        promptManager,
		WebSocketHub:         hub,
		AuthMiddleware:       authMiddleware,
	}

	GlobalContainer = container

	return container
}
