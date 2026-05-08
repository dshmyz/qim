package service

import (
	"context"

	"qim-server/model"

	"gorm.io/gorm"
)

type NoteService struct {
	db *gorm.DB
}

func NewNoteService(db *gorm.DB) *NoteService {
	return &NoteService{db: db}
}

func (s *NoteService) GetNotes(userID uint) ([]model.Note, error) {
	ctx := context.Background()
	var notes []model.Note
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at DESC").Find(&notes).Error
	return notes, err
}

func (s *NoteService) GetNote(noteID, userID uint) (*model.Note, error) {
	ctx := context.Background()
	var note model.Note
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func (s *NoteService) CreateNote(note *model.Note) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Create(note).Error
}

func (s *NoteService) UpdateNote(note *model.Note) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Save(note).Error
}

func (s *NoteService) DeleteNote(noteID, userID uint) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ?", noteID, userID).Delete(&model.Note{}).Error
}
