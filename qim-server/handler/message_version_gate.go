package handler

import (
	"time"

	"github.com/dshmyz/qim/qim-server/cache"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/gin-gonic/gin"
)

// minSendVersionCache 缓存各平台最低发消息版本（SystemConfig key: client:min_send_version[:平台]）。
// 5s TTL：管理后台改配置后最多 5 秒生效，避免每条消息都读 DB。
var minSendVersionCache = cache.NewCacheWithTTL(8, 5*time.Second)

const minSendVersionGlobalKey = "client:min_send_version"

// minSendVersionCacheKey 平台专属门槛的缓存 key；空平台表示全局。
func minSendVersionCacheKey(platform string) string {
	return "min_send_version:" + platform
}

// minSendVersionConfigKey 平台专属门槛的配置 key；空平台表示全局。
func minSendVersionConfigKey(platform string) string {
	if platform == "" {
		return minSendVersionGlobalKey
	}
	return minSendVersionGlobalKey + ":" + platform
}

// clientMinSendVersionPlatforms 最低发消息版本覆盖的平台 → 后台字段名（空平台=全局）。
// 单一事实源：门槛缓存失效、系统配置双向映射均由它驱动；新增平台只改这一处，
// 避免各处以字面量重复导致漏改一处就静默漂移。
var clientMinSendVersionPlatforms = []struct {
	Platform string
	Field    string
}{
	{"", "clientMinSendVersion"},
	{"windows", "clientMinSendVersionWindows"},
	{"macos", "clientMinSendVersionMacos"},
	{"linux", "clientMinSendVersionLinux"},
}

var (
	// minSendVersionConfigToField 配置 key → 后台字段名（含平台专属），供 GetSystemConfig 映射。
	minSendVersionConfigToField = func() map[string]string {
		m := make(map[string]string, len(clientMinSendVersionPlatforms))
		for _, p := range clientMinSendVersionPlatforms {
			m[minSendVersionConfigKey(p.Platform)] = p.Field
		}
		return m
	}()
	// minSendVersionFieldToConfig 后台字段名 → 配置 key，供 UpdateSystemConfig 映射。
	minSendVersionFieldToConfig = func() map[string]string {
		m := make(map[string]string, len(clientMinSendVersionPlatforms))
		for _, p := range clientMinSendVersionPlatforms {
			m[p.Field] = minSendVersionConfigKey(p.Platform)
		}
		return m
	}()
)

// minSendVersion 返回指定平台的最低发消息版本：平台专属配置优先，未配置回退全局；空表示不限制。
// 缓存"已解析"后的最终值（而非原始配置）：热路径每次发送一次缓存查找；
// 任一平台配置变更经 invalidateMinSendVersionCache 全量失效，无需 per-key 区分。
func minSendVersion(platform string) string {
	platform = service.NormalizePlatform(platform)
	key := minSendVersionCacheKey(platform)
	if v, ok := minSendVersionCache.Get(key); ok {
		return v.(string)
	}
	val := readMinSendVersionConfig(minSendVersionConfigKey(platform))
	if val == "" {
		val = readMinSendVersionConfig(minSendVersionConfigKey(""))
	}
	minSendVersionCache.Put(key, val)
	return val
}

func readMinSendVersionConfig(configKey string) string {
	svc := di.GlobalContainer.SystemConfigService
	if svc == nil {
		// 测试环境/未初始化容器：视为未配置门槛，不拦截
		return ""
	}
	cfg, err := svc.GetConfig(configKey)
	if err != nil {
		return ""
	}
	return cfg.Value
}

// invalidateMinSendVersionCache 清除缓存，供配置保存后即时感知。
func invalidateMinSendVersionCache() {
	for _, p := range clientMinSendVersionPlatforms {
		minSendVersionCache.Delete(minSendVersionCacheKey(p.Platform))
	}
}

// clientSendBlocked 判断客户端版本是否被发送门槛拦截：
// - 门槛未配置或格式非法 → 不拦截
// - 客户端未上报版本或格式非法（老客户端无法证明达门槛）→ 拦截
// - 客户端版本低于门槛 → 拦截
func clientSendBlocked(clientVersion, minVersion string) bool {
	if minVersion == "" || !service.IsValidVersion(minVersion) {
		return false
	}
	if clientVersion == "" || !service.IsValidVersion(clientVersion) {
		return true
	}
	return service.CompareVersions(clientVersion, minVersion) < 0
}

// clientSendBlockedRequest 对一次 HTTP 发送请求做版本门槛判定：
//   - 带版本头（新客户端）：按请求自身版本+平台判定；平台缺失时回退全局门槛。
//   - 无版本头（老客户端/未知设备）：发送设备不可知，按"用户任一在线设备低于其自身平台门槛即拦"
//     判定——不折叠多设备、确定性，用户在跑旧客户端即受门槛约束；无在线设备可证时对全局门槛 fail-closed。
//
// 返回是否应拦截、门槛值（供错误消息展示）与"是否因无法验证版本而拦"（unverifiable）。
// 未配置门槛恒不拦截。REST SendMessage 与频道发消息共用此判定，避免两处逻辑漂移。
func clientSendBlockedRequest(c *gin.Context, uid uint) (blocked bool, minV string, unverifiable bool) {
	clientVersion := c.GetHeader("X-App-Version")
	clientPlatform := c.GetHeader("X-App-Platform")

	if clientVersion != "" {
		minV = minSendVersion(clientPlatform)
		return clientSendBlocked(clientVersion, minV), minV, false
	}

	if ws.GlobalHub != nil {
		devices := ws.GlobalHub.GetUserDevices(uid)
		for _, d := range devices {
			t := minSendVersion(d.Platform)
			if clientSendBlocked(d.Version, t) {
				// 设备未上报版本或低于门槛，均因无法归属到发送设备而属"无法验证/可能过低"
				return true, t, d.Version == "" || !service.IsValidVersion(d.Version)
			}
		}
		if len(devices) > 0 {
			return false, "", false
		}
	}
	// 无在线设备可证 → 对全局门槛 fail-closed（无法验证）
	minV = minSendVersion(clientPlatform)
	return clientSendBlocked("", minV), minV, minV != ""
}
