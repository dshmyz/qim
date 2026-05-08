package app

import "qim-server/di"

type Container = di.Container

var GlobalContainer = di.GlobalContainer

func InitContainer() *Container {
	return di.InitContainer()
}
