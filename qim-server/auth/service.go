package auth

import (
	"qim-server/auth/provider"
)

var globalAuthChain *AuthChain

func InitAuthChain() {
	globalAuthChain = NewAuthChain()

	localProvider := provider.NewLocalProvider(true, 100)
	globalAuthChain.RegisterProvider(localProvider)

	globalAuthChain.SortProviders()
}

func GetAuthChain() *AuthChain {
	return globalAuthChain
}
