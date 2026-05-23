package handler

import (
	"net/http"
	"strconv"

	"qim-server/auth/provider"
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

	var authProvider model.AuthProvider
	if err := h.db.First(&authProvider, id).Error; err != nil {
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
	c.ShouldBindJSON(&testData)

	switch authProvider.Type {
	case "direct":
		switch authProvider.Name {
		case "ldap":
			ldapProvider, err := provider.NewLDAPProvider(authProvider.Name, authProvider.Enabled, authProvider.Priority, authProvider.Config)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    1,
					"message": "创建LDAP提供者失败: " + err.Error(),
					"data":    nil,
				})
				return
			}

			if err := ldapProvider.TestConnection(); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"code":    1,
					"message": "连接测试失败: " + err.Error(),
					"data":    gin.H{"provider": authProvider.Name, "status": "failed"},
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "连接测试成功",
				"data":    gin.H{"provider": authProvider.Name, "status": "connected"},
			})
		default:
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "该认证类型暂不支持连接测试",
				"data":    gin.H{"provider": authProvider.Name},
			})
		}

	case "redirect":
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "重定向认证类型无需连接测试，请检查OAuth配置",
			"data":    gin.H{"provider": authProvider.Name},
		})

	default:
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "未知的认证类型",
			"data":    gin.H{"provider": authProvider.Name},
		})
	}
}
