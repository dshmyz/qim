package service

import (
	"strconv"
	"sync"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/logger"
)

var aiNameCacheInstance *AINameCache
var aiNameCacheOnce sync.Once

type AINameCache struct {
	cache *AICache
}

func GetAINameCache() *AINameCache {
	aiNameCacheOnce.Do(func() {
		aiNameCacheInstance = &AINameCache{
			cache: NewAICache(),
		}
	})
	return aiNameCacheInstance
}

func groupKey(convID uint) string  { return "group:" + strconv.FormatUint(uint64(convID), 10) }
func avatarKey(userID uint) string { return "avatar:" + strconv.FormatUint(uint64(userID), 10) }

func (c *AINameCache) GetGroupAssistantName(conversationID uint) string {
	key := groupKey(conversationID)
	name, _ := c.cache.GetOrCompute(key, func() (string, error) {
		d := database.GetDB()
		var group model.Group
		if err := d.Select("ai_config").Where("conversation_id = ?", conversationID).First(&group).Error; err != nil {
			return "", nil
		}
		if group.AIConfigJSON == "" {
			return "AI助手", nil
		}
		aiConfig := group.GetAIConfig()
		if aiConfig.AssistantName != "" {
			return aiConfig.AssistantName, nil
		}
		return "AI助手", nil
	}, 10*time.Minute)
	return name
}

func (c *AINameCache) GetAvatarName(userID uint) string {
	key := avatarKey(userID)
	name, _ := c.cache.GetOrCompute(key, func() (string, error) {
		d := database.GetDB()
		var config model.AvatarConfig
		if err := d.Select("name").Where("user_id = ?", userID).First(&config).Error; err != nil {
			return "", nil
		}
		return config.Name, nil
	}, 10*time.Minute)
	return name
}

func (c *AINameCache) InvalidateGroupAssistantName(conversationID uint) {
	c.cache.Delete(groupKey(conversationID))
	logger.WithModule("AINameCache").Info("清除群 AI 名称缓存", "conversationID", conversationID)
}

func (c *AINameCache) InvalidateAvatarName(userID uint) {
	c.cache.Delete(avatarKey(userID))
}