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
		CreatedAt  time.Time  `json:"created_at"`
		LastUsedAt *time.Time `json:"last_used_at"`
	}
	out := make([]tokenInfo, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, tokenInfo{ID: t.ID, Name: t.Name, CreatedAt: t.CreatedAt, LastUsedAt: t.LastUsedAt})
	}
	response.Success(c, gin.H{"tokens": out})
}

// IssueUserToken 签发用户长期令牌。明文 token 仅此次返回，需前端提示用户保存。
// POST /api/v1/user-tokens  (authed)
func IssueUserToken(c *gin.Context) {
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	var req struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req)

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
