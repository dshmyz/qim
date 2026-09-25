package service

import (
	"strings"
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

// ActivePatchRules 返回当前 active 补丁中适用于指定场景的规则句（注入 system prompt 用）。
// scene 为该 AI 入口的场景标识（group_mention/group_auto/bot_dm…）；Scene 为空的补丁
// 视为通用、全场景注入，否则仅注入 scene 匹配的补丁——bot 案例学到的规则不应约束群助手。
// db 未就绪返回空。
func ActivePatchRules(db *gorm.DB, scene string) []string {
	if db == nil {
		return nil
	}
	patchCacheMu.RLock()
	if time.Now().Before(patchCacheExp) {
		defer patchCacheMu.RUnlock()
		return filterPatchRules(patchCacheRules, scene)
	}
	patchCacheMu.RUnlock()

	patchCacheMu.Lock()
	defer patchCacheMu.Unlock()
	// double check
	if time.Now().Before(patchCacheExp) {
		return filterPatchRules(patchCacheRules, scene)
	}
	var patches []model.AIPromptPatch
	if err := db.Where("state = ?", "active").Find(&patches).Error; err != nil {
		return filterPatchRules(patchCacheRules, scene)
	}
	rules := make([]string, 0, len(patches))
	for _, p := range patches {
		if p.Rule != "" {
			// 缓存条目携带补丁的场景（\x00 分隔），过滤在读取侧按调用方 scene 进行
			rules = append(rules, p.Rule+"\x00"+p.Scene)
		}
	}
	patchCacheRules = rules
	patchCacheExp = time.Now().Add(activePatchCacheTTL)
	return filterPatchRules(rules, scene)
}

// filterPatchRules 按场景过滤缓存的规则。缓存条目格式 "rule\x00scene"（scene 空=通用）。
func filterPatchRules(cached []string, scene string) []string {
	out := make([]string, 0, len(cached))
	for _, entry := range cached {
		rule, patchScene := entry, ""
		if idx := strings.IndexByte(entry, 0); idx >= 0 {
			rule, patchScene = entry[:idx], entry[idx+1:]
		}
		if patchScene == "" || patchScene == scene {
			out = append(out, rule)
		}
	}
	return out
}

// invalidatePatchCache 在补丁激活/停用时清空缓存，使注入立即生效。
func invalidatePatchCache() {
	patchCacheMu.Lock()
	patchCacheRules = nil
	patchCacheExp = time.Time{}
	patchCacheMu.Unlock()
}
