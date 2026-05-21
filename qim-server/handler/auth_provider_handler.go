package handler

import (
	"net/http"
	"strconv"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthProviderHandler struct {
	db *gorm.DB
}

func NewAuthProviderHandler() *AuthProviderHandler {
	return &AuthProviderHandler{
		db: database.GetDB(),
	}
}

func (h *AuthProviderHandler) GetProviders(c *gin.Context) {
	var providers []model.AuthProvider
	if err := h.db.Order("priority ASC").Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    providers,
	})
}

func (h *AuthProviderHandler) CreateProvider(c *gin.Context) {
	var provider model.AuthProvider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	if err := h.db.Create(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    provider,
	})
}

func (h *AuthProviderHandler) UpdateProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "invalid id",
			"data":    nil,
		})
		return
	}

	var provider model.AuthProvider
	if err := h.db.First(&provider, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    1,
			"message": "provider not found",
			"data":    nil,
		})
		return
	}

	var updateData model.AuthProvider
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	if err := h.db.Model(&provider).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    provider,
	})
}

func (h *AuthProviderHandler) DeleteProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "invalid id",
			"data":    nil,
		})
		return
	}

	if err := h.db.Delete(&model.AuthProvider{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
		"data":    nil,
	})
}

func (h *AuthProviderHandler) TestProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "invalid id",
			"data":    nil,
		})
		return
	}

	var provider model.AuthProvider
	if err := h.db.First(&provider, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    1,
			"message": "provider not found",
			"data":    nil,
		})
		return
	}

	var testData struct {
		TestUsername string `json:"test_username"`
		TestPassword string `json:"test_password"`
	}
	if err := c.ShouldBindJSON(&testData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "认证测试功能待实现",
		"data": gin.H{
			"provider": provider.Name,
		},
	})
}
