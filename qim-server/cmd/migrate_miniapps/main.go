package main

import (
	"fmt"
	"os"

	"qim-server/pkg/logger"

	"qim-server/pkg/sqlite"
	"gorm.io/gorm"
)

type MiniApp struct {
	ID          uint   `gorm:"primarykey"`
	AppID       string `gorm:"size:100;uniqueIndex;not null"`
	Name        string `gorm:"size:200;not null"`
	Description string `gorm:"type:text"`
	Icon        string `gorm:"size:500"`
	Path        string `gorm:"size:500"`
	Status      string `gorm:"size:20;default:'inactive'"`
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "qim.db"
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		logger.WithModule("MigrateMiniapps").Error("打开数据库失败", "error", err)
		os.Exit(1)
	}

	var miniApps []MiniApp
	db.Find(&miniApps)

	fmt.Printf("找到 %d 个小程序记录，开始检查 path 字段...\n", len(miniApps))

	pathMapping := map[string]string{
		"calculator":         "/miniprograms/calculator/index.html",
		"notepad":            "/miniprograms/notepad/index.html",
		"todo":               "/miniprograms/todo/index.html",
		"password-generator": "/miniprograms/password-generator/index.html",
		"short-link":         "/miniprograms/short-link/index.html",
	}

	updated := 0
	for _, app := range miniApps {
		newPath, ok := pathMapping[app.AppID]
		if ok && app.Path != newPath {
			oldPath := app.Path
			db.Model(&MiniApp{}).Where("id = ?", app.ID).Update("path", newPath)
			fmt.Printf("  [%s] %s -> %s\n", app.Name, oldPath, newPath)
			updated++
		} else if !ok && app.Path != "" && app.Path[0] == '/' && len(app.Path) < 20 {
			fmt.Printf("  [跳过] %s: path=%s (未知 appID，请手动配置)\n", app.Name, app.Path)
		}
	}

	fmt.Printf("迁移完成，更新了 %d 条记录。\n", updated)
}
