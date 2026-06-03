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
	return s.db.Delete(&model.Bot{}, id).Error
}
