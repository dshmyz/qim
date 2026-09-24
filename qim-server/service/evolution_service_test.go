package service

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"gorm.io/gorm"
)

func newEvolutionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(
		&model.AIModelCall{}, &model.AIModelHealth{},
		&model.EvolutionAction{}, &model.SystemConfig{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func seedCalls(db *gorm.DB, provider string, success, fail int) {
	now := time.Now()
	for i := 0; i < success; i++ {
		db.Create(&model.AIModelCall{Provider: provider, Success: true, FinishReason: "stop", DurationMs: 100, CreatedAt: now})
	}
	for i := 0; i < fail; i++ {
		db.Create(&model.AIModelCall{Provider: provider, Success: false, DurationMs: 100, CreatedAt: now})
	}
}

func TestEvolution_InsufficientSamplesNoAction(t *testing.T) {
	db := newEvolutionTestDB(t)
	cfgSvc := NewSystemConfigService(db)
	seedCalls(db, "p1", 5, 3) // 仅 8 次 < 20
	svc := NewEvolutionService(db, cfgSvc)

	actions, err := svc.Evaluate()
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 0 {
		t.Fatalf("样本不足不得产生动作，got %d", len(actions))
	}
	var n int64
	db.Model(&model.EvolutionAction{}).Count(&n)
	if n != 0 {
		t.Fatalf("审计也不应写入，got %d", n)
	}
}

func TestEvolution_DryRunProposesDegrade(t *testing.T) {
	db := newEvolutionTestDB(t)
	cfgSvc := NewSystemConfigService(db)
	// 25 次：10 成功 15 失败 → 完成率 40% < 70%
	seedCalls(db, "p1", 10, 15)
	svc := NewEvolutionService(db, cfgSvc)

	if svc.IsEnabled() {
		t.Fatal("默认必须关闭")
	}
	actions, err := svc.Evaluate()
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 {
		t.Fatalf("应产生1个动作，got %d", len(actions))
	}
	a := actions[0]
	if a.Kind != "model_degrade" || a.State != "proposed" || a.Target != "p1" {
		t.Fatalf("dry-run 动作异常: %+v", a)
	}
	// dry-run 不得真正降级
	if svc.IsDegraded("p1") {
		t.Fatal("dry-run 不应改变健康门控")
	}
	// 健康统计表仍写入（观测），state 维持 healthy
	var h model.AIModelHealth
	db.Where("provider='p1'").First(&h)
	if h.State != "healthy" || h.WindowCalls != 25 {
		t.Fatalf("观测统计异常: %+v", h)
	}
}

func TestEvolution_EnabledActuallyDegrades(t *testing.T) {
	db := newEvolutionTestDB(t)
	cfgSvc := NewSystemConfigService(db)
	seedCalls(db, "p1", 8, 17) // 32% 完成率
	svc := NewEvolutionService(db, cfgSvc)

	if err := svc.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if !svc.IsEnabled() {
		t.Fatal("开关应已开启")
	}
	actions, err := svc.Evaluate()
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].State != "active" {
		t.Fatalf("应产生 active 动作: %+v", actions)
	}
	if !svc.IsDegraded("p1") {
		t.Fatal("门控应已生效（缓存失效后立即降级）")
	}
}

func TestEvolution_RestoreWhenHealthyAgain(t *testing.T) {
	db := newEvolutionTestDB(t)
	cfgSvc := NewSystemConfigService(db)
	// 先 degraded（enabled）
	seedCalls(db, "p1", 5, 20) // 20%
	svc := NewEvolutionService(db, cfgSvc)
	svc.SetEnabled(true)
	svc.Evaluate()
	if !svc.IsDegraded("p1") {
		t.Fatal("setup: 应已降级")
	}

	// 删除旧的坏调用，造一批高完成率的新调用（在窗口内）
	db.Where("provider='p1'").Delete(&model.AIModelCall{})
	seedCalls(db, "p1", 30, 2) // 94% ≥ 85%
	actions, err := svc.Evaluate()
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].Kind != "model_restore" || actions[0].State != "active" {
		t.Fatalf("应恢复: %+v", actions)
	}
	if svc.IsDegraded("p1") {
		t.Fatal("门控应解除")
	}
}

func TestEvolution_RevertAction(t *testing.T) {
	db := newEvolutionTestDB(t)
	cfgSvc := NewSystemConfigService(db)
	seedCalls(db, "p1", 5, 20)
	svc := NewEvolutionService(db, cfgSvc)
	svc.SetEnabled(true)
	actions, _ := svc.Evaluate()

	if err := svc.RevertAction(actions[0].ID); err != nil {
		t.Fatalf("revert: %v", err)
	}
	if svc.IsDegraded("p1") {
		t.Fatal("回滚后应恢复 healthy")
	}
	var a model.EvolutionAction
	db.First(&a, actions[0].ID)
	if a.State != "reverted" {
		t.Fatalf("动作应标记 reverted，got %s", a.State)
	}

	// 非 active 动作不可重复回滚
	if err := svc.RevertAction(actions[0].ID); err == nil {
		t.Fatal("reverted 动作不可再次回滚")
	}
}

func TestEvolution_HealthCheckerSatisfiesInterface(t *testing.T) {
	db := newEvolutionTestDB(t)
	svc := NewEvolutionService(db, NewSystemConfigService(db))
	var _ ai.HealthChecker = svc
}
