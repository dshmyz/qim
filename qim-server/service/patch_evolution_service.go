package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"gorm.io/gorm"
)

// 补丁进化的参数门控。
const (
	minCasesForPatch = 3   // 一个类别至少累计多少条失败案例才催生补丁
	maxReplayCases   = 5   // 回放最多取多少条案例（控制 LLM 成本/时长）
	replayPassBar    = 0.6 // 回放通过率 ≥ 此 → validated，可提交人工批准
)

// PatchEvolutionService 从失败案例聚类 → LLM 生成补丁 → 黄金回放 → 批准 → 注入。
type PatchEvolutionService struct {
	db        *gorm.DB
	aiService *ai.AIService
	llm       PatchLLM // 生成规则/回放判定，默认走真实 LLM；测试可替换
}

// PatchLLM 补丁的 LLM 交互抽象：规则生成与单条回放判定。
type PatchLLM interface {
	MakeRule(category string, cases []model.AIFailureCase) (rule, rationale string, err error)
	// JudgeReplay 针对单个案例：用补丁规则生成改进回复并判定是否修复该类失败。
	JudgeReplay(patch *model.AIPromptPatch, c model.AIFailureCase) (fixed bool, err error)
}

func NewPatchEvolutionService(db *gorm.DB, aiSvc *ai.AIService) *PatchEvolutionService {
	svc := &PatchEvolutionService{db: db, aiService: aiSvc}
	svc.llm = &llmPatchLLM{ai: aiSvc}
	return svc
}

// SetLLM 仅供测试替换 LLM 交互。
func (s *PatchEvolutionService) SetLLM(p PatchLLM) { s.llm = p }

// GenerateProposals 聚类窗口内未消费的失败案例，对达阈值的类别生成补丁提案。
// windowDays 控制案例回看范围。返回新建的补丁。
func (s *PatchEvolutionService) GenerateProposals(windowDays int) ([]model.AIPromptPatch, error) {
	if windowDays <= 0 {
		windowDays = 30
	}
	since := time.Now().AddDate(0, 0, -windowDays)

	// 按类别取未消费案例
	var cases []model.AIFailureCase
	if err := s.db.Where("created_at >= ? AND consumed = ?", since, false).
		Order("created_at DESC").Find(&cases).Error; err != nil {
		return nil, err
	}

	byCategory := map[string][]model.AIFailureCase{}
	for _, c := range cases {
		// 只对有完整 question 的案例聚类（回放需要问题）
		if strings.TrimSpace(c.Question) == "" {
			continue
		}
		byCategory[c.Category] = append(byCategory[c.Category], c)
	}

	var out []model.AIPromptPatch
	for category, group := range byCategory {
		if len(group) < minCasesForPatch {
			continue
		}
		// 已有 active/validated/proposed 的同类别补丁则不重复提案
		var existing int64
		s.db.Model(&model.AIPromptPatch{}).
			Where("category = ? AND state IN ?", category, []string{"proposed", "validated", "active"}).
			Count(&existing)
		if existing > 0 {
			continue
		}

		rule, rationale, gerr := s.llm.MakeRule(category, group)
		if gerr != nil {
			logger.WithModule("PatchEvolution").Warn("生成补丁失败", "category", category, "error", gerr)
			continue
		}
		scene := group[0].Scene
		patch := model.AIPromptPatch{
			Category: category, Scene: scene, Rule: rule, Rationale: rationale,
			CaseCount: len(group), State: "proposed",
		}
		if err := s.db.Create(&patch).Error; err != nil {
			return out, err
		}
		// 该批案例已转化为补丁，标记消费：防止补丁被回放拒绝后同批案例反复催生新提案
		s.MarkCasesConsumed(category, time.Now())
		out = append(out, patch)
	}
	return out, nil
}

// llmPatchLLM PatchLLM 的真实 LLM 实现。
type llmPatchLLM struct {
	ai *ai.AIService
}

func (l *llmPatchLLM) MakeRule(category string, cases []model.AIFailureCase) (string, string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("以下是 AI 助手反复出现的「%s」类失败案例：\n\n", category))
	for i, c := range cases {
		if i >= 6 {
			break
		}
		sb.WriteString(fmt.Sprintf("案例%d\n问题：%s\n错误回复：%s\n\n", i+1,
			truncateRunes(c.Question, 300), truncateRunes(c.BadReply, 300)))
	}
	sb.WriteString("请提炼一条简洁、可执行、通用的规则，加入 AI 的系统提示后能避免这类失败。" +
		"规则必须是对 AI 行为的直接约束（一句到两句话），不要提及具体案例，不要使用否定堆叠。\n" +
		"只返回 JSON：{\"rule\": \"规则句\", \"rationale\": \"为什么这条规则有效\"}")

	msgs := []ai.Message{
		{Role: "system", Content: aiprompt.CurrentTimeLine() + "\n你是提示工程专家，负责从失败案例中总结行为规则。"},
		{Role: "user", Content: sb.String()},
	}
	result, err := l.ai.GetCompletion(ai.TaskTypeAnalysis, msgs)
	if err != nil {
		return "", "", err
	}
	var parsed struct {
		Rule      string `json:"rule"`
		Rationale string `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(result)), &parsed); err != nil {
		return "", "", fmt.Errorf("解析补丁生成结果失败: %w", err)
	}
	rule := strings.TrimSpace(parsed.Rule)
	if rule == "" {
		return "", "", fmt.Errorf("生成的规则为空")
	}
	return rule, strings.TrimSpace(parsed.Rationale), nil
}

// JudgeReplay 用补丁规则生成改进回复，再让 judge 判定是否修复该类失败。
func (l *llmPatchLLM) JudgeReplay(patch *model.AIPromptPatch, c model.AIFailureCase) (bool, error) {
	sysPrompt := fmt.Sprintf("%s\n\n你是企业 IM 的 AI 助手。%s\n【修正规则，必须遵守】\n- %s",
		aiprompt.CurrentTimeLine(), aiprompt.BuildPersona(""), patch.Rule)
	newReply, err := l.ai.GetCompletion(ai.TaskTypeChat, []ai.Message{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: c.Question},
	})
	if err != nil {
		return false, err
	}
	judgePrompt := fmt.Sprintf(
		"问题：%s\n\n原回复（被判定为%s）：%s\n\n候选回复：%s\n\n"+
			"请只判断候选回复是否避免了「%s」这类失败、是否比原回复更好。"+
			"只返回JSON：{\"fixed\": true或false, \"reason\": \"一句话理由\"}",
		c.Question, patch.Category, truncateRunes(c.BadReply, 400),
		truncateRunes(newReply, 400), patch.Category)
	judgeResult, err := l.ai.GetCompletion(ai.TaskTypeAnalysis, []ai.Message{
		{Role: "system", Content: "你是严格的回复质量评审，只依据给定的失败类别判断。"},
		{Role: "user", Content: judgePrompt},
	})
	if err != nil {
		return false, err
	}
	var verdict struct {
		Fixed bool `json:"fixed"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(judgeResult)), &verdict); err != nil {
		return false, err
	}
	return verdict.Fixed, nil
}

// Replay 对补丁做黄金回放：取该类别案例，用补丁规则重新生成回复，judge 判定
// 是否修复该类失败。统计通过率并更新补丁状态：≥replayPassBar → validated；否则 rejected。
func (s *PatchEvolutionService) Replay(patchID uint) (*model.AIPromptPatch, error) {
	var patch model.AIPromptPatch
	if err := s.db.First(&patch, patchID).Error; err != nil {
		return nil, err
	}

	var cases []model.AIFailureCase
	s.db.Where("category = ? AND question != ''", patch.Category).
		Order("created_at DESC").Limit(maxReplayCases).Find(&cases)
	if len(cases) == 0 {
		return nil, fmt.Errorf("没有可用于回放的案例")
	}

	passed := 0
	for _, c := range cases {
		ok, err := s.replayOne(&patch, c)
		if err != nil {
			// 单条回放出错按不通过计，不中断整体
			logger.WithModule("PatchEvolution").Warn("单条回放失败", "error", err)
			continue
		}
		if ok {
			passed++
		}
	}

	rate := float64(passed) / float64(len(cases))
	patch.ReplayTotal = len(cases)
	patch.ReplayPassed = passed
	if rate >= replayPassBar {
		patch.State = "validated"
	} else {
		patch.State = "rejected"
	}
	if err := s.db.Model(&model.AIPromptPatch{}).Where("id = ?", patch.ID).
		Updates(map[string]interface{}{
			"replay_total":  patch.ReplayTotal,
			"replay_passed": patch.ReplayPassed,
			"state":         patch.State,
			"updated_at":    time.Now(),
		}).Error; err != nil {
		return nil, err
	}
	return &patch, nil
}

// replayOne 委托 LLM：用补丁规则生成改进回复并判定是否修复该类失败。
func (s *PatchEvolutionService) replayOne(patch *model.AIPromptPatch, c model.AIFailureCase) (bool, error) {
	return s.llm.JudgeReplay(patch, c)
}

// Approve 人工批准 validated 补丁 → active，立即注入 system prompt。
func (s *PatchEvolutionService) Approve(patchID uint) error {
	if err := s.transition(patchID, "validated", "active"); err != nil {
		return err
	}
	invalidatePatchCache()
	return nil
}

// Deactivate 停用 active 补丁（回滚）→ reverted。
func (s *PatchEvolutionService) Deactivate(patchID uint) error {
	if err := s.transition(patchID, "active", "reverted"); err != nil {
		return err
	}
	invalidatePatchCache()
	return nil
}

func (s *PatchEvolutionService) transition(patchID uint, from, to string) error {
	var patch model.AIPromptPatch
	if err := s.db.First(&patch, patchID).Error; err != nil {
		return err
	}
	if patch.State != from {
		return fmt.Errorf("当前状态为 %s，无法从 %s 流转到 %s", patch.State, from, to)
	}
	return s.db.Model(&model.AIPromptPatch{}).Where("id = ?", patchID).
		Updates(map[string]interface{}{"state": to, "updated_at": time.Now()}).Error
}

// ListPatches 列出补丁（可按状态过滤，空=全部），最新在前。
func (s *PatchEvolutionService) ListPatches(state string) ([]model.AIPromptPatch, error) {
	q := s.db.Order("created_at DESC")
	if state != "" {
		q = q.Where("state = ?", state)
	}
	var patches []model.AIPromptPatch
	err := q.Find(&patches).Error
	return patches, err
}

// MarkCasesConsumed 把某类别在提案生成前的案例标记为已消费。
func (s *PatchEvolutionService) MarkCasesConsumed(category string, before time.Time) {
	s.db.Model(&model.AIFailureCase{}).
		Where("category = ? AND created_at <= ?", category, before).
		Update("consumed", true)
}
