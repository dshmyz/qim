package auth

import (
	"github.com/dshmyz/qim/qim-server/auth/provider"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

var globalAuthChain *AuthChain

func InitAuthChain() {
	globalAuthChain = NewAuthChain()

	db := database.GetDB()

	var authProviders []model.AuthProvider
	if err := db.Where("enabled = ?", true).Order("priority ASC").Find(&authProviders).Error; err != nil {
		logger.WithModule("Auth").Warn("加载认证提供者配置失败，仅使用本地认证", "error", err)
	}

	hasExternal := false
	for _, ap := range authProviders {
		switch ap.Protocol {
		case model.AuthProviderProtocolLDAP:
			ldapProvider, err := provider.NewLDAPProvider(ap.Name, ap.Enabled, ap.Priority, ap.Config)
			if err != nil {
				logger.WithModule("Auth").Error("创建LDAP认证提供者失败", "name", ap.Name, "error", err)
				continue
			}
			globalAuthChain.RegisterProvider(ldapProvider)
			hasExternal = true
			logger.WithModule("Auth").Info("已注册LDAP认证提供者", "name", ap.Name, "priority", ap.Priority)

		case model.AuthProviderProtocolOAuth:
			oauthProvider, err := provider.NewOAuthProvider(ap.Name, ap.Enabled, ap.Priority, ap.Config)
			if err != nil {
				logger.WithModule("Auth").Debug("创建OAuth认证提供者失败", "name", ap.Name, "error", err)
				continue
			}
			globalAuthChain.RegisterProvider(oauthProvider)
			hasExternal = true
			logger.WithModule("Auth").Info("已注册OAuth认证提供者", "name", ap.Name, "priority", ap.Priority)

		case model.AuthProviderProtocolCAS:
			casProvider, err := provider.NewCASProvider(ap.Name, ap.Enabled, ap.Priority, ap.Config)
			if err != nil {
				logger.WithModule("Auth").Debug("创建CAS认证提供者失败", "name", ap.Name, "error", err)
				continue
			}
			globalAuthChain.RegisterProvider(casProvider)
			hasExternal = true
			logger.WithModule("Auth").Info("已注册CAS认证提供者", "name", ap.Name, "priority", ap.Priority)

		default:
			logger.WithModule("Auth").Warn("未知的认证协议", "protocol", ap.Protocol, "name", ap.Name)
		}
	}

	localPriority := 100
	if hasExternal {
		localPriority = 200
	}

	localProvider := provider.NewLocalProvider(true, localPriority)
	globalAuthChain.RegisterProvider(localProvider)

	globalAuthChain.SortProviders()
}

func GetAuthChain() *AuthChain {
	return globalAuthChain
}
