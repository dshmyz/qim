package app

import (
	"qim-server/config"
	"qim-server/di"
	"qim-server/ws"
)

type Container = di.Container

var GlobalContainer = di.GlobalContainer

func InitContainer(cfg *config.Config, hub *ws.Hub) *Container {
	return di.InitContainer(cfg, hub)
}
