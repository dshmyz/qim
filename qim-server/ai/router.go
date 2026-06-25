package ai

import (
	"fmt"
	"sort"
)

type ModelRouter struct {
	routes      map[TaskType]Route
	defaultTask TaskType
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

	for _, ov := range overrides {
		if ov.TaskType == taskType && ov.Provider != "" {
			if p, ok := pool[ov.Provider]; ok && p.IsConfigured() {
				model := ov.Model
				if model == "" {
					model = route.Model
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
	for _, name := range candidates {
		if name == "" {
			continue
		}
		provider, ok := pool[name]
		if !ok || !provider.IsConfigured() {
			lastErr = fmt.Errorf("provider %s not configured", name)
			continue
		}
		return provider, route.Model, nil
	}

	// Fallback：显式配置的候选都不可用（或根本未配置路由）时，按 name 排序后
	// 选择第一个已配置的 Provider。这样数据库中添加的 Provider（其 pool key 为
	// 显示名，与 config.yaml 的路由名不一致）也能被选中，而不是直接整体失败。
	// 使用排序保证选择结果的确定性（map 遍历顺序不确定）。
	names := make([]string, 0, len(pool))
	for name := range pool {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if provider := pool[name]; provider.IsConfigured() {
			return provider, route.Model, nil
		}
	}

	return nil, "", fmt.Errorf("all providers unavailable for %s: %w", taskType, lastErr)
}
