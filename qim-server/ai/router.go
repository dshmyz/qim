package ai

import "fmt"

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
    for _, ov := range overrides {
        if ov.TaskType == taskType && ov.Provider != "" {
            if p, ok := pool[ov.Provider]; ok && p.IsConfigured() {
                return p, ov.Model, nil
            }
        }
    }

    route, ok := r.routes[taskType]
    if !ok {
        route = r.routes[r.defaultTask]
    }

    candidates := []string{route.Provider}
    candidates = append(candidates, route.Fallback...)

    var lastErr error
    for _, name := range candidates {
        provider, ok := pool[name]
        if !ok || !provider.IsConfigured() {
            lastErr = fmt.Errorf("provider %s not configured", name)
            continue
        }
        return provider, route.Model, nil
    }

    return nil, "", fmt.Errorf("all providers unavailable for %s: %w", taskType, lastErr)
}