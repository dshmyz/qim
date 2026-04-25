package service

import (
	"qim-server/ai"
)

var (
	UserSvc         *UserService
	ConversationSvc *ConversationService
	MessageSvc     *MessageService
	aiSvc          *ai.AIService
)

func Init(userService *UserService, conversationService *ConversationService, messageService *MessageService) {
	UserSvc = userService
	ConversationSvc = conversationService
	MessageSvc = messageService
}

func SetAIService(svc *ai.AIService) {
	aiSvc = svc
}

func GetAIService() *ai.AIService {
	return aiSvc
}
