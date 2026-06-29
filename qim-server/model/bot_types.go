package model

const (
	BotTypeAssistant      = "assistant"
	BotTypeGroupAssistant = "group_assistant"
	BotTypeCustom         = "custom"
	BotTypeSystem         = "system"
)

func IsAssistantBotType(botType string) bool {
	return botType == BotTypeAssistant || botType == BotTypeGroupAssistant
}
