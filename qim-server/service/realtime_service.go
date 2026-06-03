package service

import (
	"context"
	"time"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

type RealtimeService struct {
	db *gorm.DB
}

func NewRealtimeService(db *gorm.DB) *RealtimeService {
	return &RealtimeService{db: db}
}

func (s *RealtimeService) CreateSession(session *model.RealtimeSession) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Create(session).Error
}

func (s *RealtimeService) CreateParticipant(participant *model.RealtimeParticipant) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Create(participant).Error
}

func (s *RealtimeService) GetSession(sessionID string) (*model.RealtimeSession, error) {
	ctx := context.Background()
	var session model.RealtimeSession
	err := s.db.WithContext(ctx).Preload("Initiator").Preload("Participants").Preload("Participants.User").
		First(&session, "id = ?", sessionID).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *RealtimeService) GetParticipant(sessionID string, userID uint) (*model.RealtimeParticipant, error) {
	ctx := context.Background()
	var participant model.RealtimeParticipant
	err := s.db.WithContext(ctx).Where("session_id = ? AND user_id = ?", sessionID, userID).First(&participant).Error
	if err != nil {
		return nil, err
	}
	return &participant, nil
}

func (s *RealtimeService) GetParticipantByID(id string) (*model.RealtimeParticipant, error) {
	ctx := context.Background()
	var participant model.RealtimeParticipant
	err := s.db.WithContext(ctx).Preload("Session").First(&participant, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &participant, nil
}

func (s *RealtimeService) GetActiveSessions(userID uint) ([]model.RealtimeSession, error) {
	ctx := context.Background()

	var participants []model.RealtimeParticipant
	s.db.WithContext(ctx).Where("user_id = ? AND status IN ?", userID, []string{"approved", "joined"}).
		Find(&participants)

	var sessions []model.RealtimeSession
	for _, p := range participants {
		var session model.RealtimeSession
		if err := s.db.WithContext(ctx).Preload("Initiator").Preload("Participants").Preload("Participants.User").
			First(&session, "id = ?", p.SessionID).Error; err == nil {
			if session.Status == "active" || session.Status == "pending" {
				sessions = append(sessions, session)
			}
		}
	}

	return sessions, nil
}

func (s *RealtimeService) UpdateSession(session *model.RealtimeSession) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Save(session).Error
}

func (s *RealtimeService) UpdateParticipant(participant *model.RealtimeParticipant) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Save(participant).Error
}

func (s *RealtimeService) UpdateParticipantsStatus(sessionID string, status string, leftAt time.Time) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Model(&model.RealtimeParticipant{}).
		Where("session_id = ? AND status IN ?", sessionID, []string{"approved", "joined", "pending"}).
		Updates(map[string]interface{}{
			"status":  status,
			"left_at": leftAt,
		}).Error
}

func (s *RealtimeService) GetPendingRequests(userID uint) ([]model.RealtimeParticipant, error) {
	ctx := context.Background()
	var participants []model.RealtimeParticipant
	err := s.db.WithContext(ctx).Where("user_id = ? AND status = ?", userID, "pending").
		Preload("User").
		Preload("Session").
		Preload("Session.Initiator").
		Find(&participants).Error
	return participants, err
}

func (s *RealtimeService) GetParticipantWithSession(id string, userID uint) (*model.RealtimeParticipant, error) {
	ctx := context.Background()
	var participant model.RealtimeParticipant
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).
		Preload("Session").
		First(&participant).Error
	if err != nil {
		return nil, err
	}
	return &participant, nil
}
