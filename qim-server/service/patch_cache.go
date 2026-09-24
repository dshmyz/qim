package service

import (
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/model"

	"gorm.io/gorm"
)

// activePatchCacheTTL 生效补丁的缓存时长：补丁变更不频繁，避免每次构造 system prompt 查库。
const activePatchCacheTTL = 60 * time.Second

var (
	patchCacheMu    sync.RWMutex
	patchCacheRules []string
	patchCacheExp   time.Time
)

// ActivePatchRules 返回当前所有 active 补丁的规则句（注入 system prompt 用）。
// P4 阶段补丁对群回复通用，暂不按场景过滤；db 未就绪返回空。
func ActivePatchRules(db *gorm.DB) []string {
	if db == nil {
		return nil
	}
	patchCacheMu.RLock()
	if time.Now().Before(patchCacheExp) {
		defer patchCacheMu.RUnlock()
		return patchCacheRules
	}
	patchCacheMu.RUnlock()

	patchCacheMu.Lock()
	defer patchCacheMu.Unlock()
	// double check
	if time.Now().Before(patchCacheExp) {
		return patchCacheRules
	}
	var patches []model.AIPromptPatch
	if err := db.Where("state = ?", "active").Find(&patches).Error; err != nil {
		return patchCacheRules
	}
	rules := make([]string, 0, len(patches))
	for _, p := range patches {
		if p.Rule != "" {
			rules = append(rules, p.Rule)
		}
	}
	patchCacheRules = rules
	patchCacheExp = time.Now().Add(activePatchCacheTTL)
	return rules
}

// invalidatePatchCache 在补丁激活/停用时清空缓存，使注入立即生效。
func invalidatePatchCache() {
	patchCacheMu.Lock()
	patchCacheRules = nil
	patchCacheExp = time.Time{}
	patchCacheMu.Unlock()
}
