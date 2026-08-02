package handler

import (
	"encoding/json"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// UserSettingHandler 通用用户设置接口
// 路由：/api/v1/user-settings/:key（GET 读取、PUT 保存）
// 适合轻量偏好类配置（quick_replies、quick_command_panel_enabled 等）。
// value 字段接受任意 JSON 值（string/array/object/boolean/number/null），
// 存储时序列化为字符串写入 user_settings.setting_value。
type UserSettingHandler struct{}

func NewUserSettingHandler() *UserSettingHandler {
	return &UserSettingHandler{}
}

func (h *UserSettingHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/user-settings")
	{
		group.GET("/:key", h.Get)
		group.PUT("/:key", h.Save)
	}
}

// GetResponse 读取响应
type GetResponse struct {
	// Value 设置值（任意 JSON 类型）；未设置时为 nil
	Value interface{} `json:"value"`
	// HasValue 是否已设置（区分 "未设置" 与 "显式设置为 null"）
	HasValue bool `json:"has_value"`
}

// Get 读取当前用户某项设置
func (h *UserSettingHandler) Get(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	key := c.Param("key")

	svc := di.GlobalContainer.UserSettingService
	raw, err := svc.GetOrDefault(userID, key, "")
	if err != nil {
		response.InternalServerError(c, "读取设置失败")
		return
	}
	if raw == "" {
		// 未设置：返回 value=null + has_value=false
		response.Success(c, GetResponse{Value: nil, HasValue: false})
		return
	}
	// 尝试反序列化为 JSON 值；非 JSON 字符串则原样返回
	var value interface{}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		response.Success(c, GetResponse{Value: raw, HasValue: true})
		return
	}
	response.Success(c, GetResponse{Value: value, HasValue: true})
}

// SaveRequest 保存请求体
// Value 接受任意 JSON 值：string/array/object/boolean/number/null
type SaveRequest struct {
	Value interface{} `json:"value"`
}

// MaxUserSettingValueSize 单项设置值序列化后的最大字节数（防止滥用，默认 64KB）
const MaxUserSettingValueSize = 64 * 1024

// Save 保存当前用户某项设置（整存整取）
func (h *UserSettingHandler) Save(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	key := c.Param("key")

	var req SaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	// 序列化为字符串存储
	data, err := json.Marshal(req.Value)
	if err != nil {
		response.BadRequest(c, "参数序列化失败")
		return
	}
	if len(data) > MaxUserSettingValueSize {
		response.BadRequest(c, "设置值过大")
		return
	}

	svc := di.GlobalContainer.UserSettingService
	if err := svc.Upsert(userID, key, string(data)); err != nil {
		response.InternalServerError(c, "保存设置失败")
		return
	}
	response.Success(c, gin.H{"message": "保存成功"})
}
