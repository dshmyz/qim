package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"gorm.io/gorm"
)

// RenderRule 渲染增强规则（前后端共用结构）
type RenderRule struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Enabled  bool         `json:"enabled"`
	Priority int          `json:"priority"`
	Scope    RenderScope  `json:"scope"`
	Match    RenderMatch  `json:"match"`
	Render   RenderConfig `json:"render"`
}

// RenderScope 规则作用域
type RenderScope struct {
	Groups            []string `json:"groups"`
	ExcludeGroups     []string `json:"exclude_groups"`
	ConversationTypes []string `json:"conversation_types"`
}

// RenderMatch 匹配规则
type RenderMatch struct {
	Pattern       string         `json:"pattern"`
	Flags         string         `json:"flags"`
	CaptureGroups map[string]int `json:"capture_groups"`
}

// RenderConfig 渲染配置
type RenderConfig struct {
	Type          string `json:"type"`
	URLTemplate   string `json:"url_template"`
	LabelTemplate string `json:"label_template"`
	TitleTemplate string `json:"title_template"`
	Icon          string `json:"icon"`
	Target        string `json:"target"`
	Class         string `json:"class"`
}

// TestRuleResult 测试规则的匹配结果
type TestRuleResult struct {
	Matched string `json:"matched"`
	URL     string `json:"url"`
	Label   string `json:"label"`
}

const renderRulesConfigKey = "render_rules"

// 嵌套量词 ReDoS 简易检测：匹配 (xxx+)+ (xxx*)* 等模式
var reDoSPattern = regexp.MustCompile(`\([^)]*[+*?][^)]*\)[+*?]`)

// class 白名单：仅允许字母数字连字符
var classWhitelist = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// 合法的 render.type
var validRenderTypes = map[string]bool{
	"link":      true,
	"link_card": true,
	"text_chip": true,
}

type RenderRuleService struct {
	db *gorm.DB

	// 内存缓存
	mu          sync.RWMutex
	rules       []RenderRule
	version     int64
	lastRefresh time.Time
}

func NewRenderRuleService(db *gorm.DB) *RenderRuleService {
	return &RenderRuleService{db: db}
}

// GetRules 获取所有规则（含禁用的），带 5 分钟内存缓存
func (s *RenderRuleService) GetRules() ([]RenderRule, error) {
	s.mu.RLock()
	if s.rules != nil && time.Since(s.lastRefresh) < 5*time.Minute {
		s.mu.RUnlock()
		return s.rules, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// double check
	if s.rules != nil && time.Since(s.lastRefresh) < 5*time.Minute {
		return s.rules, nil
	}

	rules, version, err := s.loadFromDB()
	if err != nil {
		return nil, err
	}
	s.rules = rules
	s.version = version
	s.lastRefresh = time.Now()
	return rules, nil
}

// loadFromDB 从 system_configs 读取规则
func (s *RenderRuleService) loadFromDB() ([]RenderRule, int64, error) {
	var cfg model.SystemConfig
	err := s.db.Where("config_key = ?", renderRulesConfigKey).First(&cfg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return []RenderRule{}, 0, nil
		}
		return nil, 0, err
	}

	var rules []RenderRule
	if err := json.Unmarshal([]byte(cfg.Value), &rules); err != nil {
		return nil, 0, fmt.Errorf("规则 JSON 解析失败: %w", err)
	}
	return rules, cfg.UpdatedAt.Unix(), nil
}

// GetVersion 获取当前规则版本号
func (s *RenderRuleService) GetVersion() (int64, error) {
	// 先确保规则已加载（GetRules 内部处理锁）
	if _, err := s.GetRules(); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version, nil
}

// GetAllRules 管理后台用：获取全部规则（含禁用的）
func (s *RenderRuleService) GetAllRules() ([]RenderRule, error) {
	return s.GetRules()
}

// SaveRules 管理后台用：批量覆盖保存规则
func (s *RenderRuleService) SaveRules(rules []RenderRule) error {
	for i, rule := range rules {
		if err := s.validateRule(rule); err != nil {
			return fmt.Errorf("规则 #%d (%s) 校验失败: %w", i+1, rule.ID, err)
		}
	}

	data, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	var cfg model.SystemConfig
	result := s.db.Where("config_key = ?", renderRulesConfigKey).First(&cfg)
	if result.Error == gorm.ErrRecordNotFound {
		cfg = model.SystemConfig{
			ConfigKey: renderRulesConfigKey,
			Value:     string(data),
			Type:      "json",
			Desc:      "消息渲染增强规则",
		}
		if err := s.db.Create(&cfg).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	} else {
		cfg.Value = string(data)
		if err := s.db.Save(&cfg).Error; err != nil {
			return err
		}
	}

	// 失效缓存
	s.mu.Lock()
	s.rules = nil
	s.version = 0
	s.lastRefresh = time.Time{}
	s.mu.Unlock()

	return nil
}

// validateRule 校验单条规则安全性
func (s *RenderRuleService) validateRule(rule RenderRule) error {
	if rule.ID == "" {
		return fmt.Errorf("规则 ID 不能为空")
	}
	if len(rule.ID) > 64 {
		return fmt.Errorf("规则 ID 过长（≤64 字符）")
	}
	if rule.Priority < 1 || rule.Priority > 100 {
		return fmt.Errorf("priority 必须在 1-100 之间")
	}

	// 正则校验
	if rule.Match.Pattern == "" {
		return fmt.Errorf("match.pattern 不能为空")
	}
	if len(rule.Match.Pattern) > 200 {
		return fmt.Errorf("正则长度超限（≤200 字符）")
	}
	if len(rule.Match.CaptureGroups) > 5 {
		return fmt.Errorf("捕获组数量超限（≤5 个）")
	}
	if _, err := regexp.Compile(rule.Match.Pattern); err != nil {
		return fmt.Errorf("正则编译失败: %w", err)
	}
	if reDoSPattern.MatchString(rule.Match.Pattern) {
		return fmt.Errorf("正则疑似包含嵌套量词，有 ReDoS 风险")
	}

	// URL 模板校验
	if rule.Render.URLTemplate != "" {
		lower := strings.ToLower(rule.Render.URLTemplate)
		if strings.Contains(lower, "javascript:") {
			return fmt.Errorf("url_template 禁止 javascript 协议")
		}
		if !strings.HasPrefix(rule.Render.URLTemplate, "http://") &&
			!strings.HasPrefix(rule.Render.URLTemplate, "https://") &&
			!strings.HasPrefix(rule.Render.URLTemplate, "/") {
			return fmt.Errorf("url_template 必须以 http://、https:// 或 / 开头")
		}
	}

	// class 白名单
	if rule.Render.Class != "" && !classWhitelist.MatchString(rule.Render.Class) {
		return fmt.Errorf("class 仅允许字母数字连字符")
	}

	// type 枚举
	if !validRenderTypes[rule.Render.Type] {
		return fmt.Errorf("render.type 必须是 link/link_card/text_chip")
	}

	return nil
}

// TestRule 测试单条规则在样例文本上的匹配效果
func (s *RenderRuleService) TestRule(rule RenderRule, sampleText string) ([]TestRuleResult, error) {
	if err := s.validateRule(rule); err != nil {
		return nil, err
	}

	re := regexp.MustCompile(rule.Match.Pattern)
	matches := re.FindAllStringSubmatch(sampleText, -1)

	results := make([]TestRuleResult, 0, len(matches))
	for _, m := range matches {
		ctx := map[string]string{}
		for name, idx := range rule.Match.CaptureGroups {
			if idx < len(m) {
				ctx[name] = m[idx]
			}
		}
		results = append(results, TestRuleResult{
			Matched: m[0],
			URL:     fillTemplate(rule.Render.URLTemplate, ctx),
			Label:   fillTemplate(rule.Render.LabelTemplate, ctx),
		})
	}
	return results, nil
}

// fillTemplate 用捕获组上下文填充模板
func fillTemplate(tmpl string, ctx map[string]string) string {
	result := tmpl
	for k, v := range ctx {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}
