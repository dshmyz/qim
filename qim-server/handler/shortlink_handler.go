package handler

import (
	"math/rand"
	"net/http"
	"strconv"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
)

func CreateShortLink(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		OriginalURL string `json:"original_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	code := generateShortCode()

	shortLink := model.ShortLink{
		UserID:      userID.(uint),
		OriginalURL: req.OriginalURL,
		Code:        code,
		VisitCount:  0,
	}

	if err := db.Create(&shortLink).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成短链接失败"})
		return
	}

	shortURL := "http://" + c.Request.Host + "/" + code

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":           shortLink.ID,
			"original_url": shortLink.OriginalURL,
			"short_url":    shortURL,
			"code":         shortLink.Code,
			"visit_count":  shortLink.VisitCount,
			"created_at":   shortLink.CreatedAt,
		},
	})
}

func GetShortLinks(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()

	var shortLinks []model.ShortLink
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&shortLinks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取短链接列表失败"})
		return
	}

	response := make([]gin.H, len(shortLinks))
	for i, link := range shortLinks {
		shortURL := "http://" + c.Request.Host + "/" + link.Code
		response[i] = gin.H{
			"id":           link.ID,
			"original_url": link.OriginalURL,
			"short_url":    shortURL,
			"code":         link.Code,
			"visit_count":  link.VisitCount,
			"created_at":   link.CreatedAt,
		}
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
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "短链接不存在"})
		return
	}

	db.Model(&shortLink).Update("visit_count", shortLink.VisitCount+1)

	c.Redirect(http.StatusFound, shortLink.OriginalURL)
}

func DeleteShortLink(c *gin.Context) {
	userID, _ := c.Get("user_id")
	linkIDStr := c.Param("id")

	linkID, err := strconv.ParseUint(linkIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的短链接ID"})
		return
	}

	db := database.GetDB()

	var shortLink model.ShortLink
	if err := db.Where("id = ? AND user_id = ?", linkID, userID).First(&shortLink).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "短链接不存在或无权操作"})
		return
	}

	if err := db.Delete(&shortLink).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除短链接失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "短链接删除成功",
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
