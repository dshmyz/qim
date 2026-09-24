package ai

import (
	"fmt"
	"sort"
)

type ModelRouter struct {
	routes      map[TaskType]Route
	defaultTask TaskType
	// healthChecker 可选：报告 provider 是否被自动降权（degraded）。
	// 选择候选时，有其它健康替代则跳过 degraded；全部 degraded 时仍可降级使用。
	healthChecker HealthChecker
}

// HealthChecker provider 健康门控查询（由 service 层实现并注入，避免 ai 包依赖 DB）。
type HealthChecker interface {
	IsDegraded(provider string) bool
}

// SetHealthChecker 注入健康门控。
func (r *ModelRouter) SetHealthChecker(hc HealthChecker) {
	r.healthChecker = hc
}

func NewModelRouter(cfg RouterConfig) *ModelRouter {
	if cfg.DefaultTask == "" {
		cfg.DefaultTask = TaskTypeChat
	}
	if cfg.Routes == nil {
		cfg.Routes = make(map[TaskType]Route)
	}
	return &ModelRouter{
		routes:      cfg.Routes,
		defaultTask: cfg.DefaultTask,
	}
}

func (r *ModelRouter) SelectProvider(
	pool map[string]Provider,
	taskType TaskType,
	overrides ...Override,
) (Provider, string, error) {
	route, ok := r.routes[taskType]
	if !ok {
		route = r.routes[r.defaultTask]
	}
	// embedding 且无显式 embedding 路由时，会回退到 defaultTask（通常为 chat）路由，
	// 此时 route.Model 是文本模型名，绝不能经 WithModel 塞给 embedding 端点（会把 gpt-4o
	// 之类发送到 /embeddings 导致 400/404）。返回空模型让 Embed() 走 provider.Embedding()
	// 使用 provider 自身配置的 embedding 模型。显式配置了 embedding 路由时不受影响。
	routeModel := route.Model
	if taskType == TaskTypeEmbedding && !ok {
		routeModel = ""
	}

	for _, ov := range overrides {
		if ov.TaskType == taskType && ov.Provider != "" {
			if p, ok := pool[ov.Provider]; ok && p.IsConfigured() {
				model := ov.Model
				if model == "" {
					model = routeModel
				}
				return p, model, nil
			}
		}
	}

	candidates := make([]string, 0, 1+len(route.Fallback))
	if route.Provider != "" {
		candidates = append(candidates, route.Provider)
	}
	candidates = append(candidates, route.Fallback...)

	var lastErr error
	// degradedFallback 记录唯一可用但被降权的 provider，仅在无健康替代时启用
	var degradedFallback Provider
	for _, name := range candidates {
		if name == "" {
			continue
		}
		provider, ok := pool[name]
		if !ok || !provider.IsConfigured() {
			lastErr = fmt.Errorf("provider %s not configured", name)
			continue
		}
		// embedding 任务：跳过不支持 embedding 的 provider（如 Anthropic），
		// 避免选中后 Embedding() 返回错误导致语义搜索整体失败。
		if taskType == TaskTypeEmbedding && !provider.SupportsEmbedding() {
			lastErr = fmt.Errorf("provider %s does not support embedding", name)
			continue
		}
		if r.healthChecker != nil && r.healthChecker.IsDegraded(name) {
			if degradedFallback == nil {
				degradedFallback = provider
			}
			continue
		}
		return provider, routeModel, nil
	}
	// 无健康候选：降级使用被降权的 provider（有替代总比整体失败好）
	if degradedFallback != nil {
		return degradedFallback, routeModel, nil
	}

	// Fallback：显式配置的候选都不可用（或根本未配置路由）时，按 name 排序后
	// 选择第一个已配置的 Provider，同样优先健康。
	names := make([]string, 0, len(pool))
	for name := range pool {
		names = append(names, name)
	}
	sort.Strings(names)
	var poolDegraded Provider
	for _, name := range names {
		provider := pool[name]
		if !provider.IsConfigured() {
			continue
		}
		// embedding 任务：跳过不支持 embedding 的 provider
		if taskType == TaskTypeEmbedding && !provider.SupportsEmbedding() {
			continue
		}
		if r.healthChecker != nil && r.healthChecker.IsDegraded(name) {
			if poolDegraded == nil {
				poolDegraded = provider
			}
			continue
		}
		return provider, routeModel, nil
	}
	if poolDegraded != nil {
		return poolDegraded, routeModel, nil
	}

	return nil, "", fmt.Errorf("all providers unavailable for %s: %w", taskType, lastErr)
}

// HasExplicitRoute 报告 taskType 是否配置了独立的显式路由（而不是回退到 defaultTask）。
// 用于判定多模态任务（如 vision）是否有专门的视觉模型：未配置时会回退到 defaultTask
// （通常为纯文本 chat 模型），此时把图片 base64 发给它必然 400。调用方据此选择
// 诚实降级（"当前模型不支持看图"）而非把图片塞给不支持视觉的模型。
func (r *ModelRouter) HasExplicitRoute(taskType TaskType) bool {
	route, ok := r.routes[taskType]
	if !ok {
		return false
	}
	return route.Provider != "" || len(route.Fallback) > 0
}
