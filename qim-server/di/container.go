package di

import (
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
	groupService := service.NewGroupService(db)
	appService := service.NewAppService(db)
	miniAppService := service.NewMiniAppService(db)
	noteService := service.NewNoteService(db)
	adminService := service.NewAdminService(db)
	realtimeService := service.NewRealtimeService(db)
	sensitiveWordService := service.NewSensitiveWordService(db)
	avatarService := service.NewAvatarService(db, aiService)
	approvalService := service.NewApprovalService(db)

	authMiddleware := middleware.AuthMiddleware(cfg.JWT.Secret, userService)

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
		WebSocketHub:         hub,
		AuthMiddleware:       authMiddleware,
	}

	GlobalContainer = container

	return container
}
