package handler

import (
	"qim-server/ws"

	"github.com/gin-gonic/gin"
)

func ServeWs(hub *ws.Hub, c *gin.Context) {
	ws.ServeWs(hub, c)
}

func ServeScreenShare(hub *ws.Hub, c *gin.Context) {
	ws.ServeScreenShare(hub, c)
}
