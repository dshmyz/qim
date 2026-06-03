package app

import (
	"github.com/dshmyz/qim/qim-server/config"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/ws"
)

type Container = di.Container

var GlobalContainer = di.GlobalContainer

func InitContainer(cfg *config.Config, hub *ws.Hub) *Container {
	return di.InitContainer(cfg, hub)
}
