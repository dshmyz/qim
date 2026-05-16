package service

import (
	"context"
	"log"

	"qim-server/model"
	"qim-server/utils"

	"gorm.io/gorm"
)

type NoteService struct {
	db            *gorm.DB
	noteVectorSvc *NoteVectorService
}

func NewNoteService(db *gorm.DB) *NoteService {
	return &NoteService{db: db}
}

func (s *NoteService) SetVectorService(noteVectorSvc *NoteVectorService) {
	s.noteVectorSvc = noteVectorSvc
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
	err := s.db.WithContext(ctx).Create(note).Error
	if err == nil && s.noteVectorSvc != nil && note.Content != "" {
		utils.SafeGoWithLabel("note-vectorize", func() {
			if vecErr := s.noteVectorSvc.VectorizeNote(note.UserID, note.ID, note.Title, note.Content); vecErr != nil {
				log.Printf("[NoteService] 笔记向量化失败 note_id=%d: %v", note.ID, vecErr)
			}
		})
	}
	return err
}

func (s *NoteService) UpdateNote(note *model.Note) error {
	ctx := context.Background()
	err := s.db.WithContext(ctx).Save(note).Error
	if err == nil && s.noteVectorSvc != nil && note.Content != "" {
		utils.SafeGoWithLabel("note-vectorize", func() {
			if vecErr := s.noteVectorSvc.VectorizeNote(note.UserID, note.ID, note.Title, note.Content); vecErr != nil {
				log.Printf("[NoteService] 笔记向量化失败 note_id=%d: %v", note.ID, vecErr)
			}
		})
	}
	return err
}

func (s *NoteService) DeleteNote(noteID, userID uint) error {
	ctx := context.Background()
	if s.noteVectorSvc != nil {
		s.noteVectorSvc.DeleteNoteVectors(userID, noteID)
	}
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ?", noteID, userID).Delete(&model.Note{}).Error
}
