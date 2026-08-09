package handler

import (
	"encoding/json"
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

	// 外部 MCP 连接配置变更时，在配置落库后触发网关热同步（必须先于落库结束再同步，
	// 否则网关会读到旧配置）：让新增/修改/删除的连接立即生效而不必重启服务。
	//
	// 已知代价（刻意取舍）：ReSyncExternalMCP() 同步执行网络 IO（每连接 connect+ListTools，
	// 各带 15s 超时，慢连接 × 启用连接数叠加），本配置保存请求会被阻塞直到同步结束。
	// mcp_gateway.Sync 刻意不持锁跑网络，价值在于不阻塞 ListExternalToolNames 等热路径读取者；
	// 配置保存是低频管理操作且有 15s/连接兜底，故接受此同步阻塞而非异步化。若未来连接数
	// 增多或期望更快的保存反馈，可改为后台异步 Sync。
	if _, touched := req["external_mcp"]; touched {
		ReSyncExternalMCP()
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