package service

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"gorm.io/gorm"
)

func newPatchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.AIFailureCase{}, &model.AIPromptPatch{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedFailureCase(db *gorm.DB, category, question, reply string) {
	db.Create(&model.AIFailureCase{
		Category: category, Question: question, BadReply: reply,
		Scene: ReplySceneGroupAuto, CreatedAt: time.Now(),
	})
}

// fakePatchLLM 可控的 LLM：MakeRule 返回固定规则；JudgeReplay 按 caseIndex 控制通过率。
type fakePatchLLM struct {
	rule        string
	fixedResult map[string]bool // question -> fixed
	calls       int
}

func (f *fakePatchLLM) MakeRule(category string, cases []model.AIFailureCase) (string, string, error) {
	return f.rule, "测试依据", nil
}

func (f *fakePatchLLM) JudgeReplay(patch *model.AIPromptPatch, c model.AIFailureCase) (bool, error) {
	f.calls++
	if f.fixedResult == nil {
		return true, nil
	}
	return f.fixedResult[c.Question], nil
}

func TestPatchGenerate_InsufficientCases(t *testing.T) {
	db := newPatchTestDB(t)
	svc := NewPatchEvolutionService(db, nil)
	seedFailureCase(db, "quality", "问题1", "坏回复1")
	seedFailureCase(db, "quality", "问题2", "坏回复2")

	patches, err := svc.GenerateProposals(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(patches) != 0 {
		t.Fatalf("案例不足 minCasesForPatch(%d)，不应生成，got %d", minCasesForPatch, len(patches))
	}
}

func TestPatchGenerate_SuccessCreatesProposed(t *testing.T) {
	db := newPatchTestDB(t)
	svc := NewPatchEvolutionService(db, nil)
	fake := &fakePatchLLM{rule: "回答前先核对问题主语"}
	svc.SetLLM(fake)
	for i := 0; i < 3; i++ {
		seedFailureCase(db, "quality", "问题"+string(rune('a'+i)), "坏回复")
	}

	patches, err := svc.GenerateProposals(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(patches) != 1 {
		t.Fatalf("应生成1个补丁，got %d", len(patches))
	}
	p := patches[0]
	if p.State != "proposed" || p.Rule != "回答前先核对问题主语" || p.CaseCount != 3 {
		t.Fatalf("补丁异常: %+v", p)
	}

	// 再次生成：同类别已有 proposed，不重复
	again, _ := svc.GenerateProposals(30)
	if len(again) != 0 {
		t.Fatalf("存在进行中的同类别补丁，不应重复提案，got %d", len(again))
	}

	// 生成补丁后该批案例应被标记 consumed（防 rejected 后反复催生）
	var unconsumed int64
	db.Model(&model.AIFailureCase{}).Where("consumed = ?", false).Count(&unconsumed)
	if unconsumed != 0 {
		t.Fatalf("案例应全部标记已消费，未消费 %d 条", unconsumed)
	}
}

func TestPatchReplay_PassValidates(t *testing.T) {
	db := newPatchTestDB(t)
	svc := NewPatchEvolutionService(db, nil)
	fake := &fakePatchLLM{fixedResult: map[string]bool{
		"q1": true, "q2": true, "q3": false,
	}}
	svc.SetLLM(fake)

	patch := model.AIPromptPatch{
		Category: "quality", Rule: "规则", State: "proposed",
	}
	db.Create(&patch)
	seedFailureCase(db, "quality", "q1", "b1")
	seedFailureCase(db, "quality", "q2", "b2")
	seedFailureCase(db, "quality", "q3", "b3")

	result, err := svc.Replay(patch.ID)
	if err != nil {
		t.Fatal(err)
	}
	// 2/3 = 0.67 ≥ 0.6 → validated
	if result.State != "validated" || result.ReplayPassed != 2 || result.ReplayTotal != 3 {
		// 注意回放案例是全局取的（含 setup），用比例判断
		t.Fatalf("result: %+v", result)
	}
}

func TestPatchReplay_FailRejects(t *testing.T) {
	db := newPatchTestDB(t)
	svc := NewPatchEvolutionService(db, nil)
	fake := &fakePatchLLM{fixedResult: map[string]bool{
		"q1": false, "q2": false, "q3": true,
	}}
	svc.SetLLM(fake)

	patch := model.AIPromptPatch{Category: "quality", Rule: "r", State: "proposed"}
	db.Create(&patch)
	seedFailureCase(db, "quality", "q1", "b1")
	seedFailureCase(db, "quality", "q2", "b2")
	seedFailureCase(db, "quality", "q3", "b3")

	result, err := svc.Replay(patch.ID)
	if err != nil {
		t.Fatal(err)
	}
	// 1/3 = 0.33 < 0.6 → rejected
	if result.State != "rejected" {
		t.Fatalf("应 rejected，got %s (passed=%d/%d)", result.State, result.ReplayPassed, result.ReplayTotal)
	}
}

func TestActivePatchRules_FilteredByScene(t *testing.T) {
	invalidatePatchCache() // 包级缓存跨测试共享，先清避免读到其他测试库的数据
	db := newPatchTestDB(t)
	db.Create(&model.AIPromptPatch{Category: "a", Scene: ReplySceneGroupAuto, Rule: "群自动规则", State: "active"})
	db.Create(&model.AIPromptPatch{Category: "b", Scene: ReplySceneBotDM, Rule: "bot 规则", State: "active"})
	db.Create(&model.AIPromptPatch{Category: "c", Scene: "", Rule: "通用规则", State: "active"})
	// 非 active 不注入
	db.Create(&model.AIPromptPatch{Category: "d", Scene: "", Rule: "已停用规则", State: "reverted"})

	// 触发缓存加载
	got := ActivePatchRules(db, ReplySceneGroupAuto)
	if len(got) != 2 || got[0] != "群自动规则" || got[1] != "通用规则" {
		t.Fatalf("群自动场景应得 [群自动规则 通用规则]，got %v", got)
	}

	botRules := ActivePatchRules(db, ReplySceneBotDM)
	if len(botRules) != 2 || botRules[0] != "bot 规则" {
		t.Fatalf("bot 场景应得 [bot 规则 通用规则]，got %v", botRules)
	}

	// 未知场景只得通用规则
	other := ActivePatchRules(db, "avatar")
	if len(other) != 1 || other[0] != "通用规则" {
		t.Fatalf("其他场景应只得通用规则，got %v", other)
	}
}

func TestPatchApproveAndDeactivate(t *testing.T) {
	invalidatePatchCache()
	db := newPatchTestDB(t)
	svc := NewPatchEvolutionService(db, nil)

	patch := model.AIPromptPatch{Category: "quality", Rule: "r", State: "validated"}
	db.Create(&patch)

	// 非 validated 不能批准
	bad := model.AIPromptPatch{Category: "x", Rule: "r", State: "proposed"}
	db.Create(&bad)
	if err := svc.Approve(bad.ID); err == nil {
		t.Fatal("proposed 不能直接批准（须先回放 validated）")
	}

	if err := svc.Approve(patch.ID); err != nil {
		t.Fatalf("批准失败: %v", err)
	}
	var p model.AIPromptPatch
	db.First(&p, patch.ID)
	if p.State != "active" {
		t.Fatalf("应 active，got %s", p.State)
	}
	// active 补丁应被 ActivePatchRules 查到（缓存已失效）；补丁 Scene 为空=通用，任意场景可得
	rules := ActivePatchRules(db, ReplySceneGroupAuto)
	if len(rules) != 1 || rules[0] != "r" {
		t.Fatalf("注入规则异常: %v", rules)
	}

	if err := svc.Deactivate(patch.ID); err != nil {
		t.Fatal(err)
	}
	db.First(&p, patch.ID)
	if p.State != "reverted" {
		t.Fatalf("应 reverted，got %s", p.State)
	}
	if len(ActivePatchRules(db, ReplySceneGroupAuto)) != 0 {
		t.Fatal("停用后不应再注入")
	}
}
