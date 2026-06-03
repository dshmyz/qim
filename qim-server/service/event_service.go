package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/repository"
	"github.com/dshmyz/qim/qim-server/ws"

	"gorm.io/gorm"
)

type EventService struct {
	repo   repository.EventRepository
	db     *gorm.DB
	stopCh chan struct{}
}

func NewEventService(db *gorm.DB) *EventService {
	return &EventService{
		repo:   repository.NewEventRepository(db),
		db:     db,
		stopCh: make(chan struct{}),
	}
}

func (s *EventService) GetEvents(userID uint) ([]model.Event, error) {
	ctx := context.Background()
	events, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]model.Event, len(events))
	for i, e := range events {
		result[i] = *e
	}
	return result, nil
}

func (s *EventService) CreateEvent(event *model.Event) error {
	ctx := context.Background()
	return s.repo.Create(ctx, event)
}

func (s *EventService) GetEvent(userID, eventID uint) (*model.Event, error) {
	ctx := context.Background()
	return s.repo.FindByUserIDAndID(ctx, userID, eventID)
}

func (s *EventService) UpdateEvent(userID, eventID uint, updates *model.Event) (*model.Event, error) {
	ctx := context.Background()
	event, err := s.repo.FindByUserIDAndID(ctx, userID, eventID)
	if err != nil {
		return nil, err
	}

	event.Title = updates.Title
	event.Description = updates.Description
	event.Start = updates.Start
	event.End = updates.End
	event.AllDay = updates.AllDay
	event.Reminder = updates.Reminder
	event.ReminderSent = false

	if err := s.repo.Update(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *EventService) DeleteEvent(userID, eventID uint) error {
	ctx := context.Background()
	return s.repo.DeleteByUserIDAndID(ctx, userID, eventID)
}

func (s *EventService) StartReminderScheduler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.processReminders()

	for {
		select {
		case <-ticker.C:
			s.processReminders()
		case <-s.stopCh:
			return
		}
	}
}

func (s *EventService) StopReminderScheduler() {
	close(s.stopCh)
}

func (s *EventService) processReminders() {
	now := time.Now()

	var events []model.Event
	if err := s.db.Where(
		"reminder > 0 AND reminder_sent = ? AND start_time > ?",
		false, now,
	).Find(&events).Error; err != nil {
		return
	}

	for _, event := range events {
		reminderTime := event.Start.Add(-time.Duration(event.Reminder) * time.Minute)
		if now.After(reminderTime) || now.Equal(reminderTime) {
			s.sendReminderNotification(&event)
			s.db.Model(&event).Update("reminder_sent", true)
		}
	}
}

func (s *EventService) sendReminderNotification(event *model.Event) {
	if ws.GlobalHub == nil {
		return
	}

	msg, _ := json.Marshal(map[string]interface{}{
		"type":       "event_reminder",
		"event_id":   event.ID,
		"title":      event.Title,
		"start":      event.Start,
		"reminder":   event.Reminder,
		"created_at": time.Now().Unix(),
	})

	ws.GlobalHub.SendToUser(event.UserID, msg)
}
