package service

import (
	"context"
	"sync"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/utils"

	"gorm.io/gorm"
)

type NoteService struct {
	db            *gorm.DB
	noteVectorSvc *NoteVectorService
	// vecLocks 按笔记 ID 分条带的互斥锁：串行化同一笔记的「向量化 / 删向量」，
	// 避免异步向量化与关闭开关/清空内容之间的竞态把向量重新写回。
	vecLocks [64]sync.Mutex
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
	// 显式赋 true：带 default:true 的 bool 零值在 INSERT 时被 GORM 省略、走 DB 默认值，
	// 且仅 SQLite（RETURNING）会把默认值回填到结构体——MySQL 下不显式赋值会导致
	// 内存中 AiAccessible=false，新笔记既不向量化、创建响应也误报「分身不可见」。
	note.AiAccessible = true
	err := s.db.WithContext(ctx).Create(note).Error
	if err == nil {
		s.syncNoteVectors(note)
	}
	return err
}

func (s *NoteService) UpdateNote(note *model.Note) error {
	ctx := context.Background()
	err := s.db.WithContext(ctx).Save(note).Error
	if err == nil {
		// 打开暴露 → 向量化（delete-then-add 可安全重复）；关闭暴露或内容清空 → 移除向量。
		// 存量的、未显式设置过的笔记 AiAccessible 默认 true，行为与改造前一致。
		s.syncNoteVectors(note)
	}
	return err
}

// noteVecLock 返回该笔记对应的分条带锁。
func (s *NoteService) noteVecLock(noteID uint) *sync.Mutex {
	return &s.vecLocks[noteID%uint(len(s.vecLocks))]
}

// syncNoteVectors 按笔记当前状态同步向量库：可见且内容非空 → 异步重新向量化；
// 不可见或内容为空 → 同步移除既有向量（否则旧内容残留，分身仍能检索到）。
func (s *NoteService) syncNoteVectors(note *model.Note) {
	if s.noteVectorSvc == nil {
		return
	}
	if !note.AiAccessible || note.Content == "" {
		s.deleteNoteVectors(note.UserID, note.ID)
		return
	}
	utils.SafeGoWithLabel("note-vectorize", func() {
		lock := s.noteVecLock(note.ID)
		lock.Lock()
		defer lock.Unlock()
		// 持锁后回读最新状态再向量化：本 goroutine 排队期间开关可能已关闭、
		// 内容可能已清空或再次编辑，回读保证向量与 DB 最终状态一致。
		fresh, err := s.GetNote(note.ID, note.UserID)
		if err != nil || !fresh.AiAccessible || fresh.Content == "" {
			return
		}
		if vecErr := s.noteVectorSvc.VectorizeNote(fresh.UserID, fresh.ID, fresh.Title, fresh.Content); vecErr != nil {
			logger.WithModule("NoteService").Error("笔记向量化失败", "note_id", note.ID, "error", vecErr)
		}
	})
}

// deleteNoteVectors 在同笔记分条带锁内删除向量，与在途的异步向量化互斥。
func (s *NoteService) deleteNoteVectors(userID, noteID uint) {
	lock := s.noteVecLock(noteID)
	lock.Lock()
	defer lock.Unlock()
	if err := s.noteVectorSvc.DeleteNoteVectors(userID, noteID); err != nil {
		logger.WithModule("NoteService").Error("删除笔记向量失败", "note_id", noteID, "error", err)
	}
}

// SetNoteAiAccessible 切换笔记的「允许分身读取」状态并同步向量：
//   - 打开  → 落库后把笔记向量化进集合（分身即可检索到）
//   - 关闭  → 落库后移除该笔记既有向量（分身检索不到）
//
// 返回更新后的笔记；笔记不存在或无归属时返回 (nil, gorm.ErrRecordNotFound)。
func (s *NoteService) SetNoteAiAccessible(userID, noteID uint, accessible bool) (*model.Note, error) {
	note, err := s.GetNote(noteID, userID)
	if err != nil {
		return nil, err
	}
	note.AiAccessible = accessible
	if err := s.db.WithContext(context.Background()).Save(note).Error; err != nil {
		return nil, err
	}
	s.syncNoteVectors(note)
	return note, nil
}

func (s *NoteService) DeleteNote(noteID, userID uint) error {
	ctx := context.Background()
	if s.noteVectorSvc != nil {
		s.deleteNoteVectors(userID, noteID)
	}
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ?", noteID, userID).Delete(&model.Note{}).Error
}
