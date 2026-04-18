package main

import (
	"log"
	"qim-server/app"
	"qim-server/database"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化应用
	cfg, db, hub := app.InitApp()
	defer database.Close(db)

	// 初始化Gin
	r := gin.Default()

	// 设置路由
	app.SetupRoutes(r, cfg, hub)

	// 启动服务器
	log.Println("服务器启动在端口", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}