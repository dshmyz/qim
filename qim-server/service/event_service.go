package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/repository"
	"github.com/dshmyz/qim/qim-server/ws"

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

// ProcessReminders 扫描所有需要提醒的事件并发送通知。
// 由统一调度器（pkg/scheduler）每 30 秒调用一次。
func (s *EventService) ProcessReminders() {
	now := time.Now()

	// SQLite 下 start_time 以 TEXT 存储，比较是字典序。
	// 经 API 创建的事件 start 以 UTC 串落库（前端 toISOString -> Go 解析为 UTC time.Time），
	// 故查询参数须用 UTC，保证与存储同 offset、同格式，避免本地与 UTC 日期错位把到点事件滤掉。
	// Go 层 now.After(reminderTime) 用绝对时刻比较，不受时区影响，保持本地 now 即可。
	var events []model.Event
	if err := s.db.Where(
		"reminder > 0 AND reminder_sent = ? AND start_time > ?",
		false, now.UTC(),
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

	// ① event_reminder：原系统通知通道（showReminder -> 系统/Electron 横幅）。
	// 走标准 WSMessage{type, data} 包裹，与 new_message/message_updated 等一致：
	// 前端 handler 接收的是 message.data，扁平 map 会让 data 为 undefined 而报错。
	msg, _ := json.Marshal(ws.WSMessage{
		Type: "event_reminder",
		Data: map[string]interface{}{
			"event_id":   event.ID,
			"title":      event.Title,
			"start":      event.Start,
			"reminder":   event.Reminder,
			"created_at": time.Now().Unix(),
		},
	})
	ws.GlobalHub.SendToUser(event.UserID, msg)

	// ② new_notification：应用内通知中心通道（留痕 + 红点 + 闪烁 + toast）。
	// 系统横幅一闪即逝且依赖权限；通知中心让用户错过也能回看。
	// 与 todo_extractor.notifyTodoAssigned 范式一致：直接落库 + 推 WS。
	timeStr := event.Start.Local().Format("01/02 15:04")
	notification := model.Notification{
		UserID:        event.UserID,
		Type:          "event_reminder",
		Title:         "日历提醒",
		Content:       fmt.Sprintf("事件: %s\n时间: %s", event.Title, timeStr),
		Priority:      "normal",
		ActionType:    "view_event",
		ActionPayload: fmt.Sprintf(`{"event_id":%d}`, event.ID),
	}
	if db := database.GetDB(); db != nil {
		if err := db.Create(&notification).Error; err != nil {
			logger.WithModule("EventService").Warn("日历提醒通知落库失败",
				"eventID", event.ID, "userID", event.UserID, "error", err)
			// 落库失败仍尝试推送（前端仍能实时看到），只是刷新后丢失
		}
	}
	notifMsg, _ := json.Marshal(ws.WSMessage{
		Type: "new_notification",
		Data: notification,
	})
	ws.GlobalHub.SendToUser(event.UserID, notifMsg)
}
