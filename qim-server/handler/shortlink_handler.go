package handler

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func CreateShortLink(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		OriginalURL string     `json:"original_url" binding:"required"`
		CustomCode  string     `json:"custom_code"`  // 可选的自定义后缀
		ExpiresAt   *time.Time `json:"expires_at"`   // 可选的过期时间
		Password    string     `json:"password"`     // 可选的访问密码
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	// 确定短链接代码
	var code string
	if req.CustomCode != "" {
		// 验证自定义后缀的唯一性
		var existingLink model.ShortLink
		if err := db.Where("code = ?", req.CustomCode).First(&existingLink).Error; err == nil {
			response.BadRequest(c, "自定义后缀已被使用")
			return
		}
		code = req.CustomCode
	} else {
		// 生成随机代码,确保唯一性
		for {
			code = generateShortCode()
			var existingLink model.ShortLink
			if err := db.Where("code = ?", code).First(&existingLink).Error; err != nil {
				break
			}
		}
	}

	// 哈希密码(如果提供)
	var passwordHash string
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			response.InternalServerError(c, "密码处理失败")
			return
		}
		passwordHash = string(hashedPassword)
	}

	shortLink := model.ShortLink{
		UserID:      userID.(uint),
		OriginalURL: req.OriginalURL,
		Code:        code,
		CustomCode:  req.CustomCode,
		ExpiresAt:   req.ExpiresAt,
		Password:    passwordHash,
		VisitCount:  0,
	}

	if err := db.Create(&shortLink).Error; err != nil {
		response.InternalServerError(c, "生成短链接失败")
		return
	}

	shortURL := "http://" + c.Request.Host + "/s/" + code

	response := gin.H{
		"id":           shortLink.ID,
		"original_url": shortLink.OriginalURL,
		"short_url":    shortURL,
		"code":         shortLink.Code,
		"custom_code":  shortLink.CustomCode,
		"visit_count":  shortLink.VisitCount,
		"created_at":   shortLink.CreatedAt,
	}
	
	if shortLink.ExpiresAt != nil {
		response["expires_at"] = shortLink.ExpiresAt
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": response,
	})
}

func GetShortLinks(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()

	var shortLinks []model.ShortLink
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&shortLinks).Error; err != nil {
		response.InternalServerError(c, "获取短链接列表失败")
		return
	}

	response := make([]gin.H, len(shortLinks))
	for i, link := range shortLinks {
		shortURL := "http://" + c.Request.Host + "/s/" + link.Code
		item := gin.H{
			"id":           link.ID,
			"original_url": link.OriginalURL,
			"short_url":    shortURL,
			"code":         link.Code,
			"custom_code":  link.CustomCode,
			"visit_count":  link.VisitCount,
			"created_at":   link.CreatedAt,
			"has_password": link.Password != "",
		}
		
		if link.ExpiresAt != nil {
			item["expires_at"] = link.ExpiresAt
		}
		
		response[i] = item
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": response,
	})
}

func RedirectShortLink(c *gin.Context) {
	code := c.Param("code")

	db := database.GetDB()

	var shortLink model.ShortLink
	if err := db.Where("code = ?", code).First(&shortLink).Error; err != nil {
		response.NotFound(c, "短链接不存在")
		return
	}

	// 检查是否过期
	if shortLink.ExpiresAt != nil && time.Now().After(*shortLink.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"code": 410, "message": "短链接已过期"})
		return
	}

	// 检查是否需要密码
	if shortLink.Password != "" {
		// 从查询参数或header获取密码
		password := c.Query("password")
		if password == "" {
			password = c.GetHeader("X-ShortLink-Password")
		}
		
		if password == "" {
			response.Unauthorized(c, "需要访问密码")
			return
		}

		// 验证密码
		if err := bcrypt.CompareHashAndPassword([]byte(shortLink.Password), []byte(password)); err != nil {
			response.Unauthorized(c, "密码错误")
			return
		}
	}

	db.Model(&shortLink).Update("visit_count", shortLink.VisitCount+1)

	c.Redirect(http.StatusFound, shortLink.OriginalURL)
}

func DeleteShortLink(c *gin.Context) {
	userID, _ := c.Get("user_id")
	linkIDStr := c.Param("id")

	linkID, err := strconv.ParseUint(linkIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的短链接ID")
		return
	}

	db := database.GetDB()

	var shortLink model.ShortLink
	if err := db.Where("id = ? AND user_id = ?", linkID, userID).First(&shortLink).Error; err != nil {
		response.NotFound(c, "短链接不存在或无权操作")
		return
	}

	if err := db.Delete(&shortLink).Error; err != nil {
		response.InternalServerError(c, "删除短链接失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "短链接删除成功",
	})
}

// BatchCreateShortLinks 批量创建短链接
func BatchCreateShortLinks(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		URLs []struct {
			OriginalURL string     `json:"original_url" binding:"required"`
			CustomCode  string     `json:"custom_code"`
			ExpiresAt   *time.Time `json:"expires_at"`
			Password    string     `json:"password"`
		} `json:"urls" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if len(req.URLs) == 0 {
		response.BadRequest(c, "URL列表不能为空")
		return
	}

	if len(req.URLs) > 100 {
		response.BadRequest(c, "一次最多创建100个短链接")
		return
	}

	db := database.GetDB()

	type Result struct {
		OriginalURL string     `json:"original_url"`
		ShortURL    string     `json:"short_url,omitempty"`
		Code        string     `json:"code,omitempty"`
		CustomCode  string     `json:"custom_code,omitempty"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
		Success     bool       `json:"success"`
		Error       string     `json:"error,omitempty"`
	}

	results := make([]Result, 0, len(req.URLs))
	successCount := 0
	failCount := 0

	for _, urlItem := range req.URLs {
		result := Result{
			OriginalURL: urlItem.OriginalURL,
		}

		// 确定短链接代码
		var code string
		if urlItem.CustomCode != "" {
			// 验证自定义后缀的唯一性
			var existingLink model.ShortLink
			if err := db.Where("code = ?", urlItem.CustomCode).First(&existingLink).Error; err == nil {
				result.Error = "自定义后缀已被使用"
				result.Success = false
				results = append(results, result)
				failCount++
				continue
			}
			code = urlItem.CustomCode
		} else {
			// 生成随机代码,确保唯一性
			for {
				code = generateShortCode()
				var existingLink model.ShortLink
				if err := db.Where("code = ?", code).First(&existingLink).Error; err != nil {
					break
				}
			}
		}

		// 哈希密码(如果提供)
		var passwordHash string
		if urlItem.Password != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(urlItem.Password), bcrypt.DefaultCost)
			if err != nil {
				result.Error = "密码处理失败"
				result.Success = false
				results = append(results, result)
				failCount++
				continue
			}
			passwordHash = string(hashedPassword)
		}

		shortLink := model.ShortLink{
			UserID:      userID.(uint),
			OriginalURL: urlItem.OriginalURL,
			Code:        code,
			CustomCode:  urlItem.CustomCode,
			ExpiresAt:   urlItem.ExpiresAt,
			Password:    passwordHash,
			VisitCount:  0,
		}

		if err := db.Create(&shortLink).Error; err != nil {
			result.Error = "创建失败"
			result.Success = false
			results = append(results, result)
			failCount++
			continue
		}

		result.ShortURL = "http://" + c.Request.Host + "/" + code
		result.Code = code
		result.CustomCode = shortLink.CustomCode
		result.ExpiresAt = shortLink.ExpiresAt
		result.Success = true
		results = append(results, result)
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"results":       results,
			"success_count": successCount,
			"fail_count":    failCount,
		},
	})
}

// BatchDeleteShortLinks 批量删除短链接
func BatchDeleteShortLinks(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if len(req.IDs) == 0 {
		response.BadRequest(c, "ID列表不能为空")
		return
	}

	if len(req.IDs) > 100 {
		response.BadRequest(c, "一次最多删除100个短链接")
		return
	}

	db := database.GetDB()

	// 验证所有短链接都属于当前用户
	var count int64
	if err := db.Model(&model.ShortLink{}).Where("id IN ? AND user_id = ?", req.IDs, userID).Count(&count).Error; err != nil {
		response.InternalServerError(c, "验证权限失败")
		return
	}

	if count != int64(len(req.IDs)) {
		response.Forbidden(c, "部分短链接不存在或无权操作")
		return
	}

	// 批量删除
	if err := db.Where("id IN ?", req.IDs).Delete(&model.ShortLink{}).Error; err != nil {
		response.InternalServerError(c, "批量删除失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":          0,
		"message":       "批量删除成功",
		"deleted_count": len(req.IDs),
	})
}

func generateShortCode() string {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const codeLength = 6

	code := make([]byte, codeLength)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}

	return string(code)
}
