package handler

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/middleware"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/gin-gonic/gin"
)

// generateUserToken 生成随机用户访问令牌明文（仅此一次返回）。
// 误用 bot 令牌前缀（qbot_）或其他字段名会导致鉴权分支错配，务必保持 qusr_ 前缀。
func generateUserToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "qusr_" + hex.EncodeToString(b), nil
}

// ListUserTokens 列出当前用户的长期令牌。不返回明文/hash。
// GET /api/v1/user-tokens  (authed)
func ListUserTokens(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	var tokens []model.UserToken
	if err := db.Where("user_id = ?", userID.(uint)).Order("created_at DESC").Find(&tokens).Error; err != nil {
		response.InternalServerError(c, "查询令牌失败")
		return
	}

	type tokenInfo struct {
		ID         uint       `json:"id"`
		Name       string     `json:"name"`
		ExpiresAt  *time.Time `json:"expires_at"` // nil = 永久
		Expired    bool       `json:"expired"`    // 到期即真（过期令牌仍可撤销，鉴权已拒绝）
		CreatedAt  time.Time  `json:"created_at"`
		LastUsedAt *time.Time `json:"last_used_at"`
	}
	now := time.Now()
	out := make([]tokenInfo, 0, len(tokens))
	for _, t := range tokens {
		expired := t.ExpiresAt != nil && now.After(*t.ExpiresAt)
		out = append(out, tokenInfo{ID: t.ID, Name: t.Name, ExpiresAt: t.ExpiresAt, Expired: expired, CreatedAt: t.CreatedAt, LastUsedAt: t.LastUsedAt})
	}
	response.Success(c, gin.H{"tokens": out})
}

// IssueUserToken 签发用户长期令牌。明文 token 仅此次返回，需前端提示用户保存。
// POST /api/v1/user-tokens  (authed)
// body.expires_days 可选：0/缺省 = 永久，1..3650 = 有效天数。
func IssueUserToken(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	var req struct {
		Name        string `json:"name"`
		ExpiresDays int    `json:"expires_days"` // 0 = 永久
	}
	// 空 body（无 Content-Length）视为全部缺省——兼容无参调用；
	// 非空但绑定失败必须 400：expires_days 传错类型时若静默按 0 处理，
	// 会把用户想要的限期令牌降级成永久令牌。
	if err := c.ShouldBindJSON(&req); err != nil && c.Request.ContentLength != 0 {
		response.BadRequest(c, "请求体格式错误: "+err.Error())
		return
	}

	if req.ExpiresDays < 0 || req.ExpiresDays > 3650 {
		response.BadRequest(c, "有效天数须在 1~3650 之间（0 表示永久）")
		return
	}

	plain, err := generateUserToken()
	if err != nil {
		response.InternalServerError(c, "生成令牌失败")
		return
	}

	token := model.UserToken{
		UserID:    userID.(uint),
		TokenHash: middleware.HashBotToken(plain),
		Name:      req.Name,
	}
	if req.ExpiresDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, req.ExpiresDays)
		token.ExpiresAt = &expiresAt
	}
	if err := db.Create(&token).Error; err != nil {
		response.InternalServerError(c, "保存令牌失败")
		return
	}

	if svc := di.GlobalContainer.OperationLogService; svc != nil {
		svc.LogUserOperation(c, "user", "issue_token")
	}

	response.Success(c, gin.H{
		"token":      plain,
		"token_id":   token.ID,
		"name":       token.Name,
		"expires_at": token.ExpiresAt,
		"created_at": token.CreatedAt,
	})
}

// RevokeUserToken 撤销用户长期令牌（仅本人）。软删除即时生效。
// DELETE /api/v1/user-tokens/:tid  (authed)
func RevokeUserToken(c *gin.Context) {
	userID, _ := c.Get("user_id")
	tokenID, err := strconv.ParseUint(c.Param("tid"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的令牌 ID")
		return
	}

	db := database.GetDB()
	result := db.Where("id = ? AND user_id = ?", tokenID, userID.(uint)).Delete(&model.UserToken{})
	if result.Error != nil || result.RowsAffected == 0 {
		response.NotFound(c, "令牌不存在或已撤销")
		return
	}

	if svc := di.GlobalContainer.OperationLogService; svc != nil {
		svc.LogUserOperation(c, "user", "revoke_token")
	}

	response.SuccessWithMessage(c, "令牌已撤销", nil)
}
