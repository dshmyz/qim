package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/cache"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/pkg/response"
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
	// 由上方 map 直接求逆派生，避免两张表手写、加平台时漏改一处而漂移。
	minSendVersionFieldToConfig = func() map[string]string {
		m := make(map[string]string, len(minSendVersionConfigToField))
		for k, v := range minSendVersionConfigToField {
			m[v] = k
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

// parseClientUA 从存量客户端（2.0.39 之前）的 User-Agent 解析版本与平台。
// Electron 默认 UA 在 " Chrome/" 前带 "<app 名>/<版本>"，例如：
//
//	Mozilla/5.0 (X11; Linux x86_64) ... qim/1.0.0 Chrome/148.0.7778.218 Electron/42.3.3 ...
//
// 只取 " Chrome/" 前紧邻的 token，避免命中 AppleWebKit/537.36、Electron/42.3.3、Safari/537.36；
// 浏览器 UA 该位置是 "... Gecko)"，解析不出返回空版本。不校验产品名：客户端是独立桌面程序，
// 只要求版本是 x.y.z 三段数字，避免把产品名写死进代码。
func parseClientUA(ua string) (version, platform string) {
	if ua == "" {
		return "", ""
	}
	return parseUAVersion(ua), parseUAPlatform(ua)
}

func parseUAVersion(ua string) string {
	i := strings.Index(ua, " Chrome/")
	if i <= 0 {
		return ""
	}
	token := ua[:i]
	if j := strings.LastIndexAny(token, " ;("); j >= 0 {
		token = token[j+1:]
	}
	slash := strings.LastIndexByte(token, '/')
	if slash <= 0 {
		return ""
	}
	v := token[slash+1:]
	if !service.IsValidVersion(v) {
		return ""
	}
	return v
}

func parseUAPlatform(ua string) string {
	switch {
	case strings.Contains(ua, "Windows"):
		return "windows"
	case strings.Contains(ua, "Macintosh"), strings.Contains(ua, "Mac OS X"):
		return "macos"
	case strings.Contains(ua, "Linux"), strings.Contains(ua, "X11"):
		return "linux"
	default:
		return ""
	}
}

// clientSendBlockedRequest 对一次 HTTP 发送请求做版本门槛判定：
//   - 带版本头（新客户端）：按请求自身版本+平台判定；平台缺失时回退全局门槛。
//   - 无版本头（存量客户端）：先从 Electron 默认 UA 解析发送设备自己的版本/平台；
//     解析不出时再取该账号在线设备（WS 上报的 version/platform，与后台版本分布同源）——
//     只要有一台达到它自己的平台门槛就不拦，全部不达标才拦；无在线设备可证时对全局门槛 fail-closed。
//
// 返回是否应拦截、门槛值（供错误消息展示）与"是否因无法验证版本而拦"（unverifiable）。
// 未配置门槛恒不拦截。REST SendMessage 与频道发消息共用此判定，避免两处逻辑漂移。
func clientSendBlockedRequest(c *gin.Context, uid uint) (blocked bool, minV string, unverifiable bool) {
	// 用户长期令牌（qusr_，CLI/MCP 以本人身份调用）不做客户端版本门槛：
	// 门槛目的是拦截低版本桌面客户端，命令行工具没有客户端版本可报，
	// 若按"无版本头"fail-closed 会被任意门槛无条件拦死，CLI 将完全无法发消息。
	if c.GetString("token_type") == "user_token" {
		return false, "", false
	}

	// qim CLI 自报身份（X-Client-App: cli，doReq 统一注入）同样豁免：
	// qim login 签发的是普通 access JWT，token_type 与桌面客户端相同，无法按令牌区分。
	// 新桌面客户端恒带 X-App-Version，存量老客户端不会带此标记，故不受影响；
	// 门槛本身非安全边界（客户端本可自报任意版本），标记头与其同等强度。
	if c.GetHeader("X-Client-App") == "cli" {
		return false, "", false
	}

	clientVersion := c.GetHeader("X-App-Version")
	clientPlatform := c.GetHeader("X-App-Platform")

	// 存量客户端（2.0.39 之前）不发版本头，但 Electron 默认 UA 自带版本与平台。
	if clientVersion == "" {
		clientVersion, clientPlatform = parseClientUA(c.GetHeader("User-Agent"))
	}

	if clientVersion != "" {
		minV = minSendVersion(clientPlatform)
		return clientSendBlocked(clientVersion, minV), minV, false
	}

	if ws.GlobalHub != nil {
		devices := ws.GlobalHub.GetUserDevices(uid)
		if len(devices) > 0 {
			// 发送设备不可知：任一在线设备达到自己的平台门槛即放行（避免平台专属门槛
			// 误伤同账号其他平台的设备）；全部不达标才拦。
			allBlocked := true
			var firstMinV string
			var unverifiable bool
			for _, d := range devices {
				t := minSendVersion(d.Platform)
				if !clientSendBlocked(d.Version, t) {
					allBlocked = false
					break
				}
				if firstMinV == "" {
					firstMinV = t
				}
				if d.Version == "" || !service.IsValidVersion(d.Version) {
					unverifiable = true
				}
			}
			if allBlocked {
				return true, firstMinV, unverifiable
			}
			return false, "", false
		}
	}
	// 无在线设备可证 → 对全局门槛 fail-closed（无法验证）
	minV = minSendVersion(clientPlatform)
	return clientSendBlocked("", minV), minV, minV != ""
}

// clientRejectSendBlocked 版本门槛拒绝的统一出口：clientSendBlockedRequest 判定应拦截时，
// 按"无法验证"与"版本过低"分别输出统一文案并返回 true；未拦截返回 false。
// 收敛 REST 三处复制块（SendMessage/StreamMessage/CreateChannelMessage），改文案只改这里。
func clientRejectSendBlocked(c *gin.Context, uid uint) bool {
	blocked, minV, unverifiable := clientSendBlockedRequest(c, uid)
	if !blocked {
		return false
	}
	if unverifiable {
		response.Forbidden(c, "当前客户端版本无法验证，请升级至最新客户端并保持网络连接后再发送消息")
	} else {
		response.Forbidden(c, fmt.Sprintf("当前客户端版本过低（需 v%s 及以上），请升级客户端后再发送消息", minV))
	}
	return true
}
