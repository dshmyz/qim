package app

import (
	"qim-server/di"
	"qim-server/ws"
)

type Container = di.Container

var GlobalContainer = di.GlobalContainer

func InitContainer(secret string, hub *ws.Hub) *Container {
	return di.InitContainer(secret, hub)
}
