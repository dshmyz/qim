package service

import (
	"fmt"
	"time"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

// GovernanceMetrics 治理看板聚合结果。
type GovernanceMetrics struct {
	WindowDays       int                `json:"window_days"`
	GeneratedAt      time.Time          `json:"generated_at"`
	Scenes           []SceneGovernance  `json:"scenes"`
	InterruptReasons []NamedCount       `json:"interrupt_reasons"`
	DailyTrend       []DailySceneCount  `json:"daily_trend"`
	Feedback         FeedbackGovernance `json:"feedback"`
	Signals          []NamedCount       `json:"signals"`
}

// SceneGovernance 单场景的回复分布。
type SceneGovernance struct {
	Scene        string `json:"scene"`
	Total        int64  `json:"total"`
	Completed    int64  `json:"completed"`
	Interrupted  int64  `json:"interrupted"`
	NotTriggered int64  `json:"not_triggered"`
	// CompletionRate 完成数 /（完成+中断）——分母排除"判断不需要"的拦截
	CompletionRate float64 `json:"completion_rate"`
}

// NamedCount 通用名称-计数对。
type NamedCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// DailySceneCount 每日每场景完成数趋势。
type DailySceneCount struct {
	Day   string `json:"day"`
	Scene string `json:"scene"`
	Count int64  `json:"count"`
}

// FeedbackGovernance 显式反馈聚合。
type FeedbackGovernance struct {
	Total       int64        `json:"total"`
	Up          int64        `json:"up"`
	Down        int64        `json:"down"`
	DownReasons []NamedCount `json:"down_reasons"`
}

// AIGovernanceService 治理看板聚合查询。
type AIGovernanceService struct {
	db *gorm.DB
}

func NewAIGovernanceService(db *gorm.DB) *AIGovernanceService {
	return &AIGovernanceService{db: db}
}

// GetMetrics 聚合最近 windowDays 天的治理指标。
func (s *AIGovernanceService) GetMetrics(windowDays int) (*GovernanceMetrics, error) {
	if s.db == nil {
		return nil, fmt.Errorf("db not initialized")
	}
	if windowDays <= 0 || windowDays > 90 {
		windowDays = 7
	}
	since := time.Now().AddDate(0, 0, -(windowDays - 1))
	dayStart := time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, time.Local)

	out := &GovernanceMetrics{
		WindowDays:  windowDays,
		GeneratedAt: time.Now(),
	}

	type outcomeCount struct {
		Scene   string
		Outcome string
		Count   int64
	}
	var rows []outcomeCount
	if err := s.db.Model(&model.AIReplyEvent{}).
		Select("scene, outcome, count(*) as count").
		Where("created_at >= ?", dayStart).
		Group("scene, outcome").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	byScene := map[string]*SceneGovernance{}
	for _, r := range rows {
		sg := byScene[r.Scene]
		if sg == nil {
			sg = &SceneGovernance{Scene: r.Scene}
			byScene[r.Scene] = sg
		}
		switch r.Outcome {
		case "completed":
			sg.Completed = r.Count
		case "interrupted":
			sg.Interrupted = r.Count
		case "not_triggered":
			sg.NotTriggered = r.Count
		}
		sg.Total += r.Count
	}
	for _, sg := range byScene {
		if denom := sg.Completed + sg.Interrupted; denom > 0 {
			sg.CompletionRate = float64(sg.Completed) / float64(denom)
		}
	}
	out.Scenes = make([]SceneGovernance, 0, len(byScene))
	for _, name := range []string{
		ReplySceneBotDM, ReplySceneGroupMention, ReplySceneGroupAuto, ReplySceneAvatar,
	} {
		if sg, ok := byScene[name]; ok {
			out.Scenes = append(out.Scenes, *sg)
		}
	}

	// 中断原因分布
	if err := s.db.Model(&model.AIReplyEvent{}).
		Select("reason as name, count(*) as count").
		Where("created_at >= ? AND outcome = ?", dayStart, "interrupted").
		Group("reason").
		Order("count DESC").
		Find(&out.InterruptReasons).Error; err != nil {
		return nil, err
	}

	// 每日完成趋势
	var trend []struct {
		Day   string `gorm:"column:day"`
		Scene string
		Count int64
	}
	if err := s.db.Model(&model.AIReplyEvent{}).
		Select("DATE(created_at) as day, scene, count(*) as count").
		Where("created_at >= ? AND outcome = ?", dayStart, "completed").
		Group("DATE(created_at), scene").
		Order("day ASC").
		Find(&trend).Error; err != nil {
		return nil, err
	}
	out.DailyTrend = make([]DailySceneCount, 0, len(trend))
	for _, t := range trend {
		out.DailyTrend = append(out.DailyTrend, DailySceneCount{
			Day: t.Day, Scene: t.Scene, Count: t.Count,
		})
	}

	// 反馈聚合
	var fbTotal, fbUp, fbDown int64
	s.db.Model(&model.AIMessageFeedback{}).Where("created_at >= ?", dayStart).Count(&fbTotal)
	s.db.Model(&model.AIMessageFeedback{}).Where("created_at >= ? AND rating > 0", dayStart).Count(&fbUp)
	s.db.Model(&model.AIMessageFeedback{}).Where("created_at >= ? AND rating < 0", dayStart).Count(&fbDown)
	var downReasons []NamedCount
	if err := s.db.Model(&model.AIMessageFeedback{}).
		Select("reason_category as name, count(*) as count").
		Where("created_at >= ? AND rating < 0 AND reason_category != ''", dayStart).
		Group("reason_category").
		Order("count DESC").
		Find(&downReasons).Error; err != nil {
		return nil, err
	}
	out.Feedback = FeedbackGovernance{
		Total: fbTotal, Up: fbUp, Down: fbDown, DownReasons: downReasons,
	}

	// 学习信号分布
	if err := s.db.Model(&model.AILearningSignal{}).
		Select("signal_type as name, count(*) as count").
		Where("created_at >= ?", dayStart).
		Group("signal_type").
		Order("count DESC").
		Find(&out.Signals).Error; err != nil {
		return nil, err
	}

	return out, nil
}
