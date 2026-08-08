package service

import (
	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type BotService struct {
	db *gorm.DB
}

func NewBotService(db *gorm.DB) *BotService {
	return &BotService{db: db}
}

func (s *BotService) GetBots() ([]model.Bot, error) {
	var bots []model.Bot
	err := s.db.Where(
		"(creator_id = 0 AND is_active = ?) OR (is_template = ? AND is_active = ? AND approval_status = ?) OR (approval_status = ? AND is_active = ?)",
		true, true, true, "approved", "approved", true,
	).Find(&bots).Error
	return bots, err
}

func (s *BotService) GetBotByID(id uint) (*model.Bot, error) {
	var bot model.Bot
	err := s.db.First(&bot, id).Error
	return &bot, err
}

func (s *BotService) CreateBot(bot *model.Bot) error {
	return s.db.Create(bot).Error
}

func (s *BotService) UpdateBot(bot *model.Bot) error {
	return s.db.Save(bot).Error
}

func (s *BotService) DeleteBot(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var bot model.Bot
		if err := tx.First(&bot, id).Error; err != nil {
			return err
		}

		// 删除关联的 1:1 会话配对（bot_conversations），避免留下悬空关联
		if err := tx.Where("bot_id = ?", id).Delete(&model.BotConversation{}).Error; err != nil {
			return err
		}

		// 关键清理：机器人删除后，其虚拟用户仍可能留在各群 conversation_members 里。
		// 若不清除，前端群成员列表仍会显示该 bot，且对其发起私聊时后端
		// CreateSingleConversation 反查 virtual_user_id 失败返回 404「机器人不存在」。
		if bot.VirtualUserID != nil {
			if err := tx.Where("user_id = ?", *bot.VirtualUserID).Delete(&model.ConversationMember{}).Error; err != nil {
				return err
			}
		}

		return tx.Delete(&bot).Error
	})
}
