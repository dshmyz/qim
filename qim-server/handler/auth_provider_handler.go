package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dshmyz/qim/qim-server/auth"
	"github.com/dshmyz/qim/qim-server/auth/provider"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// sensitiveConfigKeys 是 AuthProvider.Config 中需要脱敏的密钥字段名。
// 对应 LDAP 的 bind_password、OAuth 的 client_secret 等，防止通过接口回显泄露。
var sensitiveConfigKeys = map[string]bool{
	"bind_password": true,
	"client_secret": true,
	"secret_key":    true,
	"api_key":       true,
}

// maskProviderConfig 将 provider.Config 中的敏感字段替换为 ***，返回脱敏后的副本。
// Config 为空或解析失败时原样返回，不影响非敏感字段（便于管理员查看连接地址等配置）。
func maskProviderConfig(p model.AuthProvider) model.AuthProvider {
	if p.Config == "" {
		return p
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(p.Config), &cfg); err != nil {
		// Config 不是合法 JSON，原样返回（不泄露但也不破坏）
		return p
	}
	for k := range cfg {
		if sensitiveConfigKeys[k] {
			cfg[k] = "***"
		}
	}
	masked, err := json.Marshal(cfg)
	if err != nil {
		return p
	}
	p.Config = string(masked)
	return p
}

// preserveSecretFields 在更新时保护敏感字段：如果新 Config 中敏感字段值为 "***"
// （脱敏占位符，说明前端未修改密钥），则从原 Config 保留原值，避免把真实密钥覆盖成 "***"。
func preserveSecretFields(originalConfig, newConfig string) string {
	if newConfig == "" {
		return originalConfig
	}
	if originalConfig == "" {
		return newConfig
	}
	var origCfg, newCfg map[string]interface{}
	if err := json.Unmarshal([]byte(originalConfig), &origCfg); err != nil {
		return newConfig
	}
	if err := json.Unmarshal([]byte(newConfig), &newCfg); err != nil {
		return newConfig
	}
	for k, newVal := range newCfg {
		if sensitiveConfigKeys[k] {
			if s, ok := newVal.(string); ok && s == "***" {
				// 前端未修改密钥，保留原值
				newCfg[k] = origCfg[k]
			}
		}
	}
	result, err := json.Marshal(newCfg)
	if err != nil {
		return newConfig
	}
	return string(result)
}

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
			"message": "查询认证提供者失败",
			"data":    nil,
		})
		return
	}
	// 脱敏：Config 中含 LDAP bind_password / OAuth client_secret 等密钥，不可回显
	masked := make([]model.AuthProvider, len(providers))
	for i, p := range providers {
		masked[i] = maskProviderConfig(p)
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    masked,
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

	if !model.ValidAuthProviderProtocols[provider.Protocol] {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "不支持的协议类型，支持: ldap, oauth, cas",
			"data":    nil,
		})
		return
	}

	if provider.Type == "" {
		switch provider.Protocol {
		case model.AuthProviderProtocolLDAP:
			provider.Type = "direct"
		case model.AuthProviderProtocolOAuth, model.AuthProviderProtocolCAS:
			provider.Type = "redirect"
		}
	}

	if err := h.db.Create(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "创建认证提供者失败",
			"data":    nil,
		})
		return
	}

	// 配置变更后立即重建认证链，使新增的认证方式即时生效
	auth.InitAuthChain()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    maskProviderConfig(provider),
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
			"message": "参数错误",
			"data":    nil,
		})
		return
	}

	if updateData.Protocol != "" && !model.ValidAuthProviderProtocols[updateData.Protocol] {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "不支持的协议类型，支持: ldap, oauth, cas",
			"data":    nil,
		})
		return
	}

	// 保护敏感字段：如果前端提交的 Config 中密钥是 "***"（脱敏占位符），
	// 说明未修改密钥，从原 Config 保留原值，避免覆盖成 "***"
	updateData.Config = preserveSecretFields(provider.Config, updateData.Config)

	if err := h.db.Model(&provider).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "更新认证提供者失败",
			"data":    nil,
		})
		return
	}

	h.db.First(&provider, id)

	// 配置变更后立即重建认证链，使修改后的认证方式即时生效
	auth.InitAuthChain()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    maskProviderConfig(provider),
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
			"message": "删除认证提供者失败",
			"data":    nil,
		})
		return
	}

	// 配置变更后立即重建认证链，使删除后的认证方式即时失效
	auth.InitAuthChain()

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
		switch authProvider.Protocol {
		case model.AuthProviderProtocolLDAP:
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
		switch authProvider.Protocol {
		case model.AuthProviderProtocolOAuth:
			oauthProvider, err := provider.NewOAuthProvider(authProvider.Name, authProvider.Enabled, authProvider.Priority, authProvider.Config)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    1,
					"message": "创建OAuth提供者失败: " + err.Error(),
					"data":    nil,
				})
				return
			}

			results := gin.H{"provider": authProvider.Name}

			authURL := oauthProvider.GetAuthURL("test-state")
			results["auth_url"] = authURL

			if oauthProvider.GetConfig().TokenURL != "" {
				resp, err := http.Head(oauthProvider.GetConfig().TokenURL)
				if err != nil {
					c.JSON(http.StatusOK, gin.H{
						"code":    1,
						"message": "Token端点不可达: " + err.Error(),
						"data":    results,
					})
					return
				}
				resp.Body.Close()
				results["token_url_reachable"] = true
			}

			if oauthProvider.GetConfig().UserInfoURL != "" {
				resp, err := http.Head(oauthProvider.GetConfig().UserInfoURL)
				if err != nil {
					results["user_info_url_reachable"] = false
					results["user_info_url_error"] = err.Error()
				} else {
					resp.Body.Close()
					results["user_info_url_reachable"] = true
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "OAuth配置验证通过",
				"data":    results,
			})

		case model.AuthProviderProtocolCAS:
			casProvider, err := provider.NewCASProvider(authProvider.Name, authProvider.Enabled, authProvider.Priority, authProvider.Config)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    1,
					"message": "创建CAS提供者失败: " + err.Error(),
					"data":    nil,
				})
				return
			}

			results := gin.H{"provider": authProvider.Name}

			loginURL := casProvider.GetLoginURL()
			results["login_url"] = loginURL

			resp, err := http.Head(casProvider.GetConfig().ServerURL)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"code":    1,
					"message": "CAS服务器不可达: " + err.Error(),
					"data":    results,
				})
				return
			}
			resp.Body.Close()
			results["server_reachable"] = true

			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "CAS配置验证通过",
				"data":    results,
			})

		default:
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "该重定向认证类型暂不支持连接测试",
				"data":    gin.H{"provider": authProvider.Name},
			})
		}

	default:
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "未知的认证类型",
			"data":    gin.H{"provider": authProvider.Name},
		})
	}
}

func (h *AuthProviderHandler) GetProviderLoginURL(c *gin.Context) {
	providerName := c.Param("name")

	var authProvider model.AuthProvider
	if err := h.db.Where("name = ? AND enabled = ?", providerName, true).First(&authProvider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    1,
			"message": "认证提供者不存在或未启用",
			"data":    nil,
		})
		return
	}

	switch authProvider.Protocol {
	case model.AuthProviderProtocolOAuth:
		oauthProvider, err := provider.NewOAuthProvider(authProvider.Name, authProvider.Enabled, authProvider.Priority, authProvider.Config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    1,
				"message": "创建OAuth提供者失败: " + err.Error(),
				"data":    nil,
			})
			return
		}

		state := c.Query("state")
		if state == "" {
			state = "auth"
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"provider":  authProvider.Name,
				"protocol":  authProvider.Protocol,
				"login_url": oauthProvider.GetAuthURL(state),
				"type":      "redirect",
			},
		})

	case model.AuthProviderProtocolCAS:
		casProvider, err := provider.NewCASProvider(authProvider.Name, authProvider.Enabled, authProvider.Priority, authProvider.Config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    1,
				"message": "创建CAS提供者失败: " + err.Error(),
				"data":    nil,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"provider":   authProvider.Name,
				"protocol":   authProvider.Protocol,
				"login_url":  casProvider.GetLoginURL(),
				"logout_url": casProvider.GetLogoutURL(),
				"type":       "redirect",
			},
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "该认证类型不支持获取登录URL",
			"data":    nil,
		})
	}
}
