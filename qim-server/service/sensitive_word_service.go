package service

import (
	"context"

	"qim-server/model"

	"gorm.io/gorm"
)

type SensitiveWordService struct {
	db *gorm.DB
}

func NewSensitiveWordService(db *gorm.DB) *SensitiveWordService {
	return &SensitiveWordService{db: db}
}

type SensitiveWordQuery struct {
	Page     int
	PageSize int
	Keyword  string
}

type SensitiveWordResult struct {
	List     []model.SensitiveWord `json:"list"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

func (s *SensitiveWordService) GetSensitiveWords(q SensitiveWordQuery) (*SensitiveWordResult, error) {
	ctx := context.Background()

	query := s.db.WithContext(ctx).Model(&model.SensitiveWord{})
	if q.Keyword != "" {
		query = query.Where("word LIKE ?", "%"+q.Keyword+"%")
	}

	var total int64
	query.Count(&total)

	var words []model.SensitiveWord
	offset := (q.Page - 1) * q.PageSize
	query.Order("created_at DESC").Offset(offset).Limit(q.PageSize).Find(&words)

	return &SensitiveWordResult{
		List:     words,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (s *SensitiveWordService) GetByWord(word string) (*model.SensitiveWord, error) {
	ctx := context.Background()
	var sw model.SensitiveWord
	err := s.db.WithContext(ctx).Where("word = ? AND deleted_at IS NULL", word).First(&sw).Error
	if err != nil {
		return nil, err
	}
	return &sw, nil
}

func (s *SensitiveWordService) Create(word *model.SensitiveWord) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Create(word).Error
}

func (s *SensitiveWordService) GetByID(id uint) (*model.SensitiveWord, error) {
	ctx := context.Background()
	var word model.SensitiveWord
	err := s.db.WithContext(ctx).First(&word, id).Error
	if err != nil {
		return nil, err
	}
	return &word, nil
}

func (s *SensitiveWordService) Update(word *model.SensitiveWord) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Save(word).Error
}

func (s *SensitiveWordService) Delete(id uint) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Delete(&model.SensitiveWord{}, id).Error
}

func (s *SensitiveWordService) GetAllEnabled() ([]model.SensitiveWord, error) {
	ctx := context.Background()
	var words []model.SensitiveWord
	err := s.db.WithContext(ctx).Where("enabled = ?", true).Find(&words).Error
	return words, err
}

func (s *SensitiveWordService) GetAll() ([]model.SensitiveWord, error) {
	ctx := context.Background()
	var words []model.SensitiveWord
	err := s.db.WithContext(ctx).Find(&words).Error
	return words, err
}
