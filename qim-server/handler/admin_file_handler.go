package handler

import (
	"net/http"
	"strconv"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
)

type FileStatistics struct {
	TotalSize int64  `json:"totalSize"`
	UsedSize  int64  `json:"usedSize"`
	FileCount int64  `json:"fileCount"`
	TypeStats []struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
		Size  int64  `json:"size"`
	} `json:"typeStats"`
}

type LargeFile struct {
	ID           uint   `json:"id"`
	FileName     string `json:"fileName"`
	FileSize     int64  `json:"fileSize"`
	UploaderID   uint   `json:"uploaderId"`
	UploaderName string `json:"uploaderName"`
	CreatedAt    string `json:"createdAt"`
}

func GetAdminFileStatistics(c *gin.Context) {
	db := database.GetDB()

	var totalSize int64
	db.Model(&model.File{}).Select("COALESCE(SUM(size), 0)").Scan(&totalSize)

	var fileCount int64
	db.Model(&model.File{}).Count(&fileCount)

	typeStats := make([]struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
		Size  int64  `json:"size"`
	}, 0)

	db.Model(&model.File{}).
		Select(`
			CASE
				WHEN mime_type LIKE 'image/%' THEN 'image'
				WHEN mime_type LIKE 'video/%' THEN 'video'
				WHEN mime_type LIKE 'audio/%' THEN 'audio'
				WHEN mime_type LIKE 'application/%' OR mime_type LIKE 'text/%' THEN 'document'
				ELSE 'other'
			END as type,
			COUNT(*) as count,
			COALESCE(SUM(size), 0) as size
		`).
		Group(`
			CASE
				WHEN mime_type LIKE 'image/%' THEN 'image'
				WHEN mime_type LIKE 'video/%' THEN 'video'
				WHEN mime_type LIKE 'audio/%' THEN 'audio'
				WHEN mime_type LIKE 'application/%' OR mime_type LIKE 'text/%' THEN 'document'
				ELSE 'other'
			END
		`).Scan(&typeStats)

	stat := FileStatistics{
		TotalSize: totalSize,
		UsedSize:  totalSize,
		FileCount: fileCount,
		TypeStats: typeStats,
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": stat,
	})
}

func GetAdminLargeFiles(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	db := database.GetDB()

	type Result struct {
		ID           uint   `json:"id"`
		Name         string `json:"name"`
		Size         int64  `json:"size"`
		UserID       uint   `json:"user_id"`
		Username     string `json:"username"`
		Nickname     string `json:"nickname"`
		CreatedAt    string `json:"created_at"`
	}

	var results []Result
	db.Model(&model.File{}).
		Select("files.id, files.name, files.size, files.user_id, users.username, users.nickname, files.created_at").
		Joins("LEFT JOIN users ON files.user_id = users.id").
		Order("files.size DESC").
		Limit(limit).
		Scan(&results)

	files := make([]LargeFile, 0, len(results))
	for _, r := range results {
		uploaderName := r.Username
		if r.Nickname != "" {
			uploaderName = r.Nickname
		}

		files = append(files, LargeFile{
			ID:           r.ID,
			FileName:     r.Name,
			FileSize:     r.Size,
			UploaderID:   r.UserID,
			UploaderName: uploaderName,
			CreatedAt:    r.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": files,
	})
}
