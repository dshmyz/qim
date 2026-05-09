package di

import (
	"log"
	"qim-server/ai"
	"qim-server/config"
	"qim-server/database"
	"qim-server/middleware"
	"qim-server/service"
	"qim-server/ws"

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
	// 新增：RAG 和智能分身相关服务
	VectorService        *service.VectorService
	NoteVectorService    *service.NoteVectorService
	AvatarMemoryService  *service.AvatarMemoryService
	AvatarTriggerService *service.AvatarTriggerService
	WebSocketHub         *ws.Hub
	AuthMiddleware       gin.HandlerFunc
}

var GlobalContainer *Container

func InitContainer(cfg *config.Config, hub *ws.Hub) *Container {
	db := database.GetDB()

	aiService := ai.NewAIService(&cfg.AI)

	userService := service.NewUserService(db)
	conversationService := service.NewConversationService(db)
	messageService := service.NewMessageServiceWithDBType(db, hub, aiService, cfg.Database.Type)
	notificationService := service.NewNotificationService(db)
	eventService := service.NewEventService(db)
	taskService := service.NewTaskService(db)
	fileService := service.NewFileService(db)
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

	authMiddleware := middleware.AuthMiddleware(cfg.JWT.Secret, userService)

	// 初始化 RAG 相关服务
	var vectorSvc *service.VectorService
	var noteVectorSvc *service.NoteVectorService
	var avatarMemorySvc *service.AvatarMemoryService
	var avatarTriggerSvc *service.AvatarTriggerService

	// 尝试初始化向量服务（如果 cortexdb 可用）
	vectorPath := "./data/vector.db"

	var err error
	vectorSvc, err = service.NewVectorService(vectorPath)
	if err != nil {
		log.Printf("[DI] Warning: VectorService 初始化失败: %v (RAG 功能将不可用)", err)
	} else {
		noteVectorSvc = service.NewNoteVectorService(vectorSvc, aiService)
		avatarMemorySvc = service.NewAvatarMemoryService(vectorSvc, aiService, db)
		avatarTriggerSvc = service.NewAvatarTriggerService(aiService, db)
	}

	// 注入向量服务到相关服务
	if noteVectorSvc != nil {
		noteService.SetVectorService(noteVectorSvc)
		groupDocumentService.SetVectorServices(vectorSvc, aiService)
		avatarService.SetRAGServices(noteVectorSvc, avatarMemorySvc, avatarTriggerSvc)
	}

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
		VectorService:        vectorSvc,
		NoteVectorService:    noteVectorSvc,
		AvatarMemoryService:  avatarMemorySvc,
		AvatarTriggerService: avatarTriggerSvc,
		WebSocketHub:         hub,
		AuthMiddleware:       authMiddleware,
	}

	GlobalContainer = container

	return container
}
