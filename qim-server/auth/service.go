package auth

import (
	"qim-server/auth/provider"
	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/logger"
)

var globalAuthChain *AuthChain

func InitAuthChain() {
	globalAuthChain = NewAuthChain()

	db := database.GetDB()

	var authProviders []model.AuthProvider
	if err := db.Where("type = ? AND enabled = ?", "direct", true).Order("priority ASC").Find(&authProviders).Error; err != nil {
		logger.WithModule("Auth").Warn("加载认证提供者配置失败，仅使用本地认证", "error", err)
	}

	hasLDAP := false
	for _, ap := range authProviders {
		switch ap.Name {
		case "ldap":
			ldapProvider, err := provider.NewLDAPProvider(ap.Name, ap.Enabled, ap.Priority, ap.Config)
			if err != nil {
				logger.WithModule("Auth").Error("创建LDAP认证提供者失败", "error", err)
				continue
			}
			globalAuthChain.RegisterProvider(ldapProvider)
			hasLDAP = true
			logger.WithModule("Auth").Info("已注册LDAP认证提供者", "priority", ap.Priority)
		}
	}

	localPriority := 100
	if hasLDAP {
		localPriority = 200
	}

	localProvider := provider.NewLocalProvider(true, localPriority)
	globalAuthChain.RegisterProvider(localProvider)

	globalAuthChain.SortProviders()
}

func GetAuthChain() *AuthChain {
	return globalAuthChain
}
