package handler

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/middleware"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/gin-gonic/gin"
)

func GetSystemConfig(c *gin.Context) {
	configSvc := di.GlobalContainer.SystemConfigService
	result, err := configSvc.GetAllConfigs()
	if err != nil {
		response.InternalServerError(c, "获取配置失败")
		return
	}

	result = mapConfigToFrontend(result)
	response.Success(c, result)
}

func mapConfigToFrontend(raw map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	rateLimitKeys := map[string]string{
		"rate_limit:global_rate":          "rateLimitGlobalRate",
		"rate_limit:global_window_seconds": "rateLimitGlobalWindow",
		"rate_limit:login_max_attempts":   "rateLimitLoginMaxAttempts",
		"rate_limit:login_window_seconds": "rateLimitLoginWindow",
		"rate_limit:login_ban_seconds":    "rateLimitLoginBan",
	}
	for k, v := range raw {
		switch k {
		case "file_upload:max_size":
			if n, ok := v.(int); ok {
				out["maxFileSize"] = n / (1024 * 1024)
			} else if s, ok := v.(string); ok {
				// 兜底：旧记录可能因类型识别 bug 被写成 type=string，按字符串解析回 int
				if n, err := strconv.Atoi(s); err == nil {
					out["maxFileSize"] = n / (1024 * 1024)
				} else {
					out["maxFileSize"] = 50
				}
			} else {
				out["maxFileSize"] = 50
			}
		case "file_upload:allowed_extensions":
			if s, ok := v.(string); ok {
				out["allowedFileTypes"] = s
			}
		default:
			if fk, ok := rateLimitKeys[k]; ok {
				out[fk] = v
			} else {
				out[k] = v
			}
		}
	}
	if _, ok := out["maxFileSize"]; !ok {
		out["maxFileSize"] = 50
	}
	if _, ok := out["allowedFileTypes"]; !ok {
		out["allowedFileTypes"] = defaultAllowedExtJSON()
	}
	// 速率限制默认值
	if _, ok := out["rateLimitGlobalRate"]; !ok {
		out["rateLimitGlobalRate"] = 500
	}
	if _, ok := out["rateLimitGlobalWindow"]; !ok {
		out["rateLimitGlobalWindow"] = 60
	}
	if _, ok := out["rateLimitLoginMaxAttempts"]; !ok {
		out["rateLimitLoginMaxAttempts"] = 5
	}
	if _, ok := out["rateLimitLoginWindow"]; !ok {
		out["rateLimitLoginWindow"] = 60
	}
	if _, ok := out["rateLimitLoginBan"]; !ok {
		out["rateLimitLoginBan"] = 900
	}
	return out
}

func defaultAllowedExtJSON() string {
	return `[".jpg",".jpeg",".png",".gif",".bmp",".webp",".pdf",".doc",".docx",".xls",".xlsx",".ppt",".pptx",".txt",".md",".csv",".zip",".rar",".7z",".mp3",".wav",".mp4",".avi",".mov"]`
}

func GetPublicSystemConfig(c *gin.Context) {
	configSvc := di.GlobalContainer.SystemConfigService
	result, err := configSvc.GetPublicConfigs()
	if err != nil {
		response.InternalServerError(c, "获取配置失败")
		return
	}

	// vector_enabled 不存数据库（基础设施状态，不是用户配置），
	// 由 handler 运行时注入：VectorService 非 nil 即视为可用。
	// 前端据此显示「知识库开关无效」等提示，避免用户开了开关但实际没生效。
	vectorEnabled := di.GlobalContainer.VectorService != nil
	result = mergeRuntimeFlags(result, vectorEnabled)

	response.Success(c, result)
}

// mergeRuntimeFlags 把运行时基础设施状态合并进配置结果。
// 抽成函数便于单元测试（handler 直接依赖 di.GlobalContainer 不易测）。
func mergeRuntimeFlags(result map[string]interface{}, vectorEnabled bool) map[string]interface{} {
	if result == nil {
		result = map[string]interface{}{}
	}
	result["vector_enabled"] = vectorEnabled
	return result
}

func UpdateSystemConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	req = mapConfigFromFrontend(req)

	configSvc := di.GlobalContainer.SystemConfigService
	if err := configSvc.BatchUpdate(req); err != nil {
		response.InternalServerError(c, "配置保存失败")
		return
	}

	// 外部 MCP 连接配置变更时，在配置落库后触发网关热同步：让新增/修改/删除的连接
	// 立即生效而不必重启服务。异步执行——Sync 每连接 connect+ListTools 各带 15s 超时，
	// 慢连接 × 启用连接数叠加会阻塞请求，与前端 15s axios 超时撞车导致保存报错。
	// 配置已先落库，热同步延后到后台不影响正确性；Sync 自带串行锁，此处再 recover
	// 兜底 panic，可安全脱离请求生命周期运行。
	if _, touched := req["external_mcp"]; touched {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[SystemConfig] ReSyncExternalMCP panic: %v", r)
				}
			}()
			ReSyncExternalMCP()
		}()
	}

	// 动态重新加载速率限制配置
	middleware.ReloadRateLimitFromDB(func(key string) (string, error) {
		cfg, err := configSvc.GetConfig(key)
		if err != nil {
			return "", err
		}
		return cfg.Value, nil
	})

	publicConfigs, _ := configSvc.GetPublicConfigs()
	wsMsg := ws.WSMessage{Type: "system_config_updated", Data: publicConfigs}
	jsonData, _ := json.Marshal(wsMsg)
	if ws.GlobalHub != nil {
		ws.GlobalHub.BroadcastToAllOnlineUsers(jsonData)
	}

	response.Success(c, gin.H{"message": "配置保存成功"})
}

func mapConfigFromFrontend(req map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	rateLimitKeys := map[string]string{
		"rateLimitGlobalRate":       "rate_limit:global_rate",
		"rateLimitGlobalWindow":     "rate_limit:global_window_seconds",
		"rateLimitLoginMaxAttempts": "rate_limit:login_max_attempts",
		"rateLimitLoginWindow":      "rate_limit:login_window_seconds",
		"rateLimitLoginBan":         "rate_limit:login_ban_seconds",
	}
	for k, v := range req {
		switch k {
		case "maxFileSize":
			if n, ok := v.(float64); ok {
				out["file_upload:max_size"] = int64(n) * 1024 * 1024
			}
		case "allowedFileTypes":
			out["file_upload:allowed_extensions"] = v
		default:
			if dbKey, ok := rateLimitKeys[k]; ok {
				out[dbKey] = v
			} else {
				out[k] = v
			}
		}
	}
	return out
}