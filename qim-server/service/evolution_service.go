package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

// 进化执行器的配置门控。
const (
	// evolutionEnabledKey 全局 kill switch，存于 system_configs。默认 false：
	// 未显式开启时执行器只做 dry-run（写 proposed 建议），不改变任何状态。
	evolutionEnabledKey = "ai_evolution_enabled"

	// 健康评估窗口与判定门槛
	healthWindowDays = 7
	minSampleCalls   = 20   // 最小样本：不足不评估（不被个例带偏）
	degradeThreshold = 0.7  // 完成率低于此 → 建议/执行降权
	restoreThreshold = 0.85 // 完成率回升超过此 → 建议/执行恢复
)

// EvolutionService AI 自我进化执行器：聚合模型调用事实 → 更新健康状态 → 审计。
type EvolutionService struct {
	db        *gorm.DB
	configSvc *SystemConfigService
}

func NewEvolutionService(db *gorm.DB, configSvc *SystemConfigService) *EvolutionService {
	return &EvolutionService{db: db, configSvc: configSvc}
}

// degradedCacheTTL 降权集合的内存缓存时长：避免每次 SelectProvider 都查库。
const degradedCacheTTL = 30 * time.Second

var (
	degradedCacheMu sync.Mutex
	degradedSet     map[string]struct{}
	degradedCacheAt time.Time
)

// IsDegraded 实现 ai.HealthChecker：报告 provider 当前是否被自动降权。
func (s *EvolutionService) IsDegraded(provider string) bool {
	set := s.loadDegradedSet()
	_, ok := set[provider]
	return ok
}

func (s *EvolutionService) loadDegradedSet() map[string]struct{} {
	degradedCacheMu.Lock()
	defer degradedCacheMu.Unlock()
	if degradedSet != nil && time.Since(degradedCacheAt) < degradedCacheTTL {
		return degradedSet
	}
	var rows []model.AIModelHealth
	s.db.Select("provider").Where("state = ?", "degraded").Find(&rows)
	set := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		set[r.Provider] = struct{}{}
	}
	degradedSet = set
	degradedCacheAt = time.Now()
	return set
}

// invalidateDegradedCache 在状态变更后清空缓存，使门控立即生效。
func invalidateDegradedCache() {
	degradedCacheMu.Lock()
	degradedSet = nil
	degradedCacheAt = time.Time{}
	degradedCacheMu.Unlock()
}

// providerAgg 单 provider 窗口聚合。
type providerAgg struct {
	Provider    string
	Total       int64
	Successful  int64
	AvgDuration int64
}

// IsEnabled 读取全局 kill switch。
func (s *EvolutionService) IsEnabled() bool {
	if s.configSvc == nil {
		return false
	}
	cfg, err := s.configSvc.GetConfig(evolutionEnabledKey)
	if err != nil || cfg == nil || cfg.Value == "" {
		return false
	}
	return cfg.Value == "true" || cfg.Value == "1"
}

// SetEnabled 切换全局 kill switch。
func (s *EvolutionService) SetEnabled(enabled bool) error {
	if s.configSvc == nil {
		return fmt.Errorf("config service unavailable")
	}
	v := "false"
	if enabled {
		v = "true"
	}
	return s.configSvc.UpsertConfig(evolutionEnabledKey, v, "bool",
		"AI 自我进化执行器总开关：开启后允许系统在窄域内自动降权/恢复模型；关闭时仅产出建议")
}

// Evaluate 执行一次健康评估。返回本次产生的审计动作。
// 未开启时为 dry-run：只写 proposed，不更新健康状态。
func (s *EvolutionService) Evaluate() ([]model.EvolutionAction, error) {
	since := time.Now().AddDate(0, 0, -healthWindowDays)
	dayStart := time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, time.Local)

	var aggs []providerAgg
	if err := s.db.Model(&model.AIModelCall{}).
		Select("provider, count(*) as total, sum(case when success then 1 else 0 end) as successful, avg(duration_ms) as avg_duration").
		Where("created_at >= ? AND provider != ''", dayStart).
		Group("provider").
		Find(&aggs).Error; err != nil {
		return nil, err
	}

	enabled := s.IsEnabled()
	var actions []model.EvolutionAction
	for _, agg := range aggs {
		if agg.Total < minSampleCalls {
			// 样本不足：不动作（防止小样本抖动），也不写动作
			continue
		}
		score := float64(agg.Successful) / float64(agg.Total)

		// 当前健康状态（无记录视为 healthy）
		var health model.AIModelHealth
		found := true
		if err := s.db.Where("provider = ?", agg.Provider).First(&health).Error; err != nil {
			found = false
		}
		currentState := "healthy"
		if found && health.State == "degraded" {
			currentState = "degraded"
		}

		// 期望状态
		desired := currentState
		if currentState == "healthy" && score < degradeThreshold {
			desired = "degraded"
		} else if currentState == "degraded" && score >= restoreThreshold {
			desired = "healthy"
		}
		if desired == currentState {
			// 始终刷新健康统计表（观测用），但不产生动作
			s.upsertHealth(agg, score, currentState, found, health, currentState == "degraded")
			continue
		}

		kind := "model_degrade"
		if desired == "healthy" {
			kind = "model_restore"
		}
		basis := fmt.Sprintf("窗口%d天 样本%d 成功%d 完成率%.1f%% 平均延迟%dms",
			healthWindowDays, agg.Total, agg.Successful, score*100, agg.AvgDuration)

		before, _ := json.Marshal(map[string]interface{}{"state": currentState})
		after, _ := json.Marshal(map[string]interface{}{"state": desired, "score": score})

		state := "proposed"
		if enabled {
			state = "active"
			s.upsertHealth(agg, score, desired, found, health, desired == "degraded")
		} else {
			// dry-run 仍把观测统计写入（state 维持现状）
			s.upsertHealth(agg, score, currentState, found, health, currentState == "degraded")
		}

		action := model.EvolutionAction{
			Kind: kind, Target: agg.Provider,
			BeforeJSON: string(before), AfterJSON: string(after),
			Basis: basis, State: state, CreatedAt: time.Now(),
		}
		if err := s.db.Create(&action).Error; err != nil {
			return actions, err
		}
		actions = append(actions, action)
	}

	return actions, nil
}

// upsertHealth 更新/插入 provider 健康状态。
func (s *EvolutionService) upsertHealth(agg providerAgg, score float64, state string, found bool, existing model.AIModelHealth, degraded bool) {
	reason := ""
	if degraded {
		reason = fmt.Sprintf("近%d天完成率 %.0f%% 低于阈值 %.0f%%（%d 次样本）",
			healthWindowDays, score*100, degradeThreshold*100, agg.Total)
	}
	fields := map[string]interface{}{
		"window_calls":    agg.Total,
		"completed":       agg.Successful,
		"interrupted":     agg.Total - agg.Successful,
		"avg_duration_ms": agg.AvgDuration,
		"score":           score,
		"state":           state,
		"reason":          reason,
		"updated_at":      time.Now(),
	}
	if found {
		s.db.Model(&model.AIModelHealth{}).Where("id = ?", existing.ID).Updates(fields)
	} else {
		// 新建也要带全部统计字段（此前只写 Provider/State 导致观测列为 0）
		row := model.AIModelHealth{
			Provider:      agg.Provider,
			WindowCalls:   int(agg.Total),
			Completed:     int(agg.Successful),
			Interrupted:   int(agg.Total - agg.Successful),
			AvgDurationMs: agg.AvgDuration,
			Score:         score,
			State:         state,
			Reason:        reason,
		}
		s.db.Create(&row)
	}
	invalidateDegradedCache()
}

// ListActions 列出最近的进化动作（最新在前）。
func (s *EvolutionService) ListActions(limit int) ([]model.EvolutionAction, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var actions []model.EvolutionAction
	err := s.db.Order("created_at DESC").Limit(limit).Find(&actions).Error
	return actions, err
}

// ListHealth 列出全部 provider 健康状态。
func (s *EvolutionService) ListHealth() ([]model.AIModelHealth, error) {
	var health []model.AIModelHealth
	err := s.db.Order("updated_at DESC").Find(&health).Error
	return health, err
}

// RevertAction 手动回滚一个 active 动作：degraded→healthy（或反之）。
func (s *EvolutionService) RevertAction(actionID uint) error {
	var action model.EvolutionAction
	if err := s.db.First(&action, actionID).Error; err != nil {
		return err
	}
	if action.State != "active" {
		return fmt.Errorf("仅生效中的动作可回滚（当前 %s）", action.State)
	}

	revertState := "healthy"
	if action.Kind == "model_restore" {
		revertState = "degraded"
	}
	if err := s.db.Model(&model.AIModelHealth{}).
		Where("provider = ?", action.Target).
		Updates(map[string]interface{}{"state": revertState, "updated_at": time.Now()}).Error; err != nil {
		return err
	}
	invalidateDegradedCache()
	return s.db.Model(&model.EvolutionAction{}).Where("id = ?", actionID).
		Update("state", "reverted").Error
}

// ensure unused import referenced (ai used by HealthChecker signature)
var _ ai.HealthChecker = (*EvolutionService)(nil)
