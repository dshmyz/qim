package service

import (
	"context"
	"time"

	"qim-server/model"
	"qim-server/repository"

	"gorm.io/gorm"
)

type EventService struct {
	repo repository.EventRepository
	db   *gorm.DB
}

func NewEventService(db *gorm.DB) *EventService {
	return &EventService{
		repo: repository.NewEventRepository(db),
		db:   db,
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

	if err := s.repo.Update(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *EventService) DeleteEvent(userID, eventID uint) error {
	ctx := context.Background()
	return s.repo.DeleteByUserIDAndID(ctx, userID, eventID)
}

func (s *EventService) GetDB() *gorm.DB {
	return s.db
}

func (s *EventService) CreateReminderNotification(userID uint, event *model.Event) {
	reminderTime := event.Start.Add(-time.Duration(event.Reminder) * time.Minute)
	now := time.Now()

	if reminderTime.Before(now) {
		return
	}

	waitDuration := reminderTime.Sub(now)
	timer := time.NewTimer(waitDuration)
	<-timer.C

	var currentEvent model.Event
	if err := s.db.First(&currentEvent, event.ID).Error; err != nil {
		return
	}

	if time.Now().After(currentEvent.End) {
		return
	}
}
