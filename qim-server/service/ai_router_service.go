package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

// RouterConfigKey 是 SystemConfig 表里存储「AI 模型路由」DB 覆盖的键。
// Type 为 json，Value 存放 ai.RouterConfig 序列化后的 JSON。
const RouterConfigKey = "ai_router"

// AIRouterService 负责「AI 模型路由」（任务类型 → provider/model）的 DB 持久化与运行时热更。
// 语义：DB 覆盖优先——存在 ai_router 覆盖时使用 DB 值；否则回退到 config.yaml 的默认路由。
//
// 为避免 service → di 的循环依赖，热更 AIService 的动作由调用方（handler/di）传入
// *ai.AIService 参数，与 AIProviderService.ReloadEnabledProviders 的约定一致。
type AIRouterService struct {
	db          *gorm.DB
	configSvc   *SystemConfigService
	providerSvc *AIProviderService
	// defaultRouter 返回 config.yaml 的默认路由（副本）；测试可注入。
	defaultRouter func() *ai.RouterConfig
}

func NewAIRouterService(db *gorm.DB) *AIRouterService {
	return &AIRouterService{
		db:          db,
		configSvc:   NewSystemConfigService(db),
		providerSvc: NewAIProviderService(db),
	}
}

// SetDefaultRouterFunc 注入 config.yaml 默认路由的读取函数（主要供测试使用）。
// 生产环境由 di 容器注入（读 di.GlobalContainer.Config.AI.Router）。
func (s *AIRouterService) SetDefaultRouterFunc(fn func() *ai.RouterConfig) {
	s.defaultRouter = fn
}

// DefaultRouter 返回当前 config.yaml 默认路由（非 nil 副本）。供前台展示与「恢复默认」使用。
func (s *AIRouterService) DefaultRouter() *ai.RouterConfig {
	var rc *ai.RouterConfig
	if s.defaultRouter != nil {
		rc = s.defaultRouter()
	}
	if rc == nil {
		rc = &ai.RouterConfig{}
	}
	return cloneRouter(rc)
}

// GetDBRouter 返回 DB 中保存的路由覆盖；若无覆盖返回 (nil, nil)。
func (s *AIRouterService) GetDBRouter() (*ai.RouterConfig, error) {
	cfg, err := s.configSvc.GetConfig(RouterConfigKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if cfg.Value == "" {
		return nil, nil
	}
	var rc ai.RouterConfig
	if err := json.Unmarshal([]byte(cfg.Value), &rc); err != nil {
		return nil, fmt.Errorf("解析 router 配置失败: %w", err)
	}
	if rc.Routes == nil {
		rc.Routes = make(map[ai.TaskType]ai.Route)
	}
	return &rc, nil
}

// HasDBRouter 是否已存在 DB 路由覆盖。
func (s *AIRouterService) HasDBRouter() (bool, error) {
	rc, err := s.GetDBRouter()
	if err != nil {
		return false, err
	}
	return rc != nil, nil
}

// GetEffectiveRouter 返回当前生效路由：DB 有覆盖用 DB，否则用 config.yaml 默认。
// 同时返回已配置的 AI 供应商（前台 provider 下拉候选）。
func (s *AIRouterService) GetEffectiveRouter() (*ai.RouterConfig, []model.AIProvider, error) {
	providers, err := s.providerSvc.GetProviders()
	if err != nil {
		return nil, nil, err
	}

	dbRouter, err := s.GetDBRouter()
	if err != nil {
		return nil, nil, err
	}

	if dbRouter != nil {
		return dbRouter, providers, nil
	}
	return s.DefaultRouter(), providers, nil
}

// SaveRouter 将路由保存到 SystemConfig(ai_router, json)，并热更新到传入的 AIService。
//
// 校验语义与运行时对齐（运行时对未配置 provider 的路由会自动回退到已配置 provider）：
//   - provider 为空的路由直接忽略；
//   - 引用未配置供应商的路由：跳过（不写入），收集警告返回，不阻塞其余有效路由的保存；
//   - model 不再做硬校验——模型由 provider 在运行时校验，已登记 models 仅作前端下拉候选
//     （前端模型框允许自由输入自定义模型，硬校验会误拦）。
// 全部路由都被跳过时不写入（避免用空路由覆盖既有配置）。
func (s *AIRouterService) SaveRouter(aiService *ai.AIService, rc *ai.RouterConfig) (warnings []string, err error) {
	if rc == nil {
		return nil, errors.New("路由配置不能为空")
	}
	if rc.Routes == nil {
		rc.Routes = make(map[ai.TaskType]ai.Route)
	}

	// provider 合法集：已启用供应商 name（小写匹配）+ 运行时 provider 池。
	// 运行时池纳入：DB 无已启用供应商时，AIService 回退 config.yaml 供应商作为兜底
	//（见 ReloadProvidersFromDB），这些 provider 运行时可用却不在 DB 登记表里，
	// 不并入会导致纯 config.yaml 部署下默认路由（如 openai）保存被误拒。
	providers, err := s.providerSvc.GetProviders()
	if err != nil {
		return nil, err
	}
	providerSet := map[string]bool{}
	for _, p := range providers {
		if !p.Enabled || p.Name == "" {
			continue
		}
		providerSet[strings.ToLower(p.Name)] = true
	}
	if aiService != nil {
		for _, name := range aiService.ProviderNames() {
			providerSet[name] = true
		}
	}

	valid := make(map[ai.TaskType]ai.Route, len(rc.Routes))
	for taskType, route := range rc.Routes {
		if route.Provider == "" {
			continue
		}
		if !providerSet[strings.ToLower(route.Provider)] {
			warnings = append(warnings, fmt.Sprintf("任务 %q 引用了未配置的供应商 %q，该路由已跳过", taskType, route.Provider))
			continue
		}
		valid[taskType] = route
	}

	if len(valid) == 0 {
		return warnings, errors.New("没有可保存的有效路由（引用的供应商均未配置）")
	}
	rc.Routes = valid

	data, err := json.Marshal(rc)
	if err != nil {
		return nil, fmt.Errorf("序列化路由配置失败: %w", err)
	}

	if err := s.upsertConfig(aiService, RouterConfigKey, string(data)); err != nil {
		return nil, err
	}
	return warnings, nil
}

// ClearRouter 清除 DB 路由覆盖，使运行回到 config.yaml 默认路由，并热更新传入的 AIService。
func (s *AIRouterService) ClearRouter(aiService *ai.AIService) error {
	if err := s.db.Where("config_key = ?", RouterConfigKey).Delete(&model.SystemConfig{}).Error; err != nil {
		return err
	}
	if aiService != nil {
		aiService.UpdateRouter(*s.DefaultRouter())
	}
	return nil
}

// upsertConfig 按 key 写入 SystemConfig（存在则更新，不存在则插入）。Type 固定为 json。
// 写入成功后热更新 AIService。
func (s *AIRouterService) upsertConfig(aiService *ai.AIService, key, value string) error {
	var cfg model.SystemConfig
	err := s.db.Where("config_key = ?", key).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = model.SystemConfig{
			ConfigKey: key,
			Value:     value,
			Type:      "json",
			Desc:      "AI 模型路由（任务类型 → provider/model）DB 覆盖",
		}
		if err := s.db.Create(&cfg).Error; err != nil {
			return err
		}
		if aiService != nil {
			if err := applyRouterValueTo(aiService, value); err != nil {
				return err
			}
		}
		return nil
	}
	if err != nil {
		return err
	}
	cfg.Value = value
	cfg.Type = "json"
	if err := s.db.Save(&cfg).Error; err != nil {
		return err
	}
	if aiService != nil {
		if err := applyRouterValueTo(aiService, value); err != nil {
			return err
		}
	}
	return nil
}

// applyRouterValueTo 将 router JSON 反序列化后热更到 AIService。
func applyRouterValueTo(aiService *ai.AIService, value string) error {
	var rc ai.RouterConfig
	if err := json.Unmarshal([]byte(value), &rc); err != nil {
		return fmt.Errorf("解析 router 配置失败: %w", err)
	}
	if rc.Routes == nil {
		rc.Routes = make(map[ai.TaskType]ai.Route)
	}
	aiService.UpdateRouter(rc)
	return nil
}

func cloneRouter(rc *ai.RouterConfig) *ai.RouterConfig {
	if rc == nil {
		rc = &ai.RouterConfig{}
	}
	out := &ai.RouterConfig{DefaultTask: rc.DefaultTask}
	out.Routes = make(map[ai.TaskType]ai.Route, len(rc.Routes))
	for k, v := range rc.Routes {
		cp := v
		cp.Fallback = append([]string(nil), v.Fallback...)
		out.Routes[k] = cp
	}
	return out
}
