package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
)

func UploadFile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	form, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件上传失败"})
		return
	}

	// 检查文件大小
	maxSize := int64(cfg.Upload.MaxSizeMB) * 1024 * 1024
	if cfg.Upload.MaxSizeMB > 0 && form.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("文件大小超过限制(%dMB)", cfg.Upload.MaxSizeMB),
		})
		return
	}

	// 打开文件读取头部用于MIME类型检测
	file, err := form.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件打开失败"})
		return
	}
	defer file.Close()

	// 只读取前512字节用于MIME检测
	header := make([]byte, 512)
	n, _ := file.Read(header)
	mimeType := http.DetectContentType(header[:n])

	// 检查MIME类型是否在允许列表中
	allowedMap := make(map[string]bool)
	for _, t := range cfg.Upload.AllowedTypes {
		allowedMap[t] = true
	}

	// 允许image/*类型，或者在明确允许的列表中
	isAllowed := allowedMap[mimeType] || strings.HasPrefix(mimeType, "image/")
	if len(cfg.Upload.AllowedTypes) > 0 && !isAllowed {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("不支持的文件类型: %s", mimeType),
		})
		return
	}

	// 重置文件指针回到开头
	file.Seek(0, io.SeekStart)

	ext := filepath.Ext(form.Filename)
	filename := time.Now().Format("20060102150405") + "_" + strconv.FormatUint(uint64(userID.(uint)), 10) + ext
	var storagePath string

	if cfg.Storage.Type == "s3" {
		storagePath = "/s3/" + filename
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "S3存储功能暂未实现",
			"data": gin.H{
				"id":   0,
				"url":  storagePath,
				"name": form.Filename,
				"size": form.Size,
			},
		})
		return
	} else {
		uploadDir := cfg.Storage.Local.Path
		if uploadDir == "" {
			uploadDir = "./uploads"
		}
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建上传目录失败"})
			return
		}

		filePath := filepath.Join(uploadDir, filename)
		if err := c.SaveUploadedFile(form, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存文件失败"})
			return
		}

		db := database.GetDB()
		fileRecord := model.File{
			Name:         form.Filename,
			OriginalName: form.Filename,
			StoragePath:  "/uploads/" + filename,
			Size:         form.Size,
			UserID:       userID.(uint),
			CreatedAt:    time.Now(),
		}
		if err := db.Create(&fileRecord).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建文件记录失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"id":   fileRecord.ID,
				"url":  fileRecord.StoragePath,
				"name": fileRecord.Name,
				"size": fileRecord.Size,
			},
		})
	}
}

func GetFiles(c *gin.Context) {
	userID, _ := c.Get("user_id")

	folderIDStr := c.Query("folder_id")

	db := database.GetDB()
	var files []model.File

	query := db.Where("user_id = ?", userID)
	if folderIDStr != "" {
		folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
		if err == nil {
			query = query.Where("folder_id = ?", uint(folderID))
		}
	}

	query.Order("created_at DESC").Find(&files)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": files,
	})
}

func DownloadFile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fileIDStr := c.Param("id")

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件ID"})
		return
	}

	db := database.GetDB()
	var file model.File
	if err := db.Where("id = ? AND user_id = ?", uint(fileID), userID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在"})
		return
	}

	if strings.HasPrefix(file.StoragePath, "/s3/") {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "S3存储文件下载功能暂未实现",
			"data":    gin.H{"file_id": file.ID},
		})
		return
	} else {
		filePath := "." + file.StoragePath

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在"})
			return
		}

		c.FileAttachment(filePath, file.Name)
	}
}

func DeleteFile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fileIDStr := c.Param("id")

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件ID"})
		return
	}

	db := database.GetDB()
	var file model.File
	if err := db.Where("id = ? AND user_id = ?", uint(fileID), userID).First(&file).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在"})
		return
	}

	if strings.HasPrefix(file.StoragePath, "/s3/") {
		if err := db.Delete(&file).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除文件失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "S3存储文件删除功能暂未完全实现，仅删除了文件记录",
		})
		return
	} else {
		filePath := "." + file.StoragePath
		os.Remove(filePath)

		if err := db.Delete(&file).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除文件失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "删除文件成功",
		})
	}
}

func CreateFolder(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name     string `json:"name" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	folder := model.Folder{
		UserID:   userID.(uint),
		Name:     req.Name,
		ParentID: req.ParentID,
	}
	db.Create(&folder)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": folder,
	})
}

func GetFolderTree(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()

	var folders []model.Folder
	db.Where("user_id = ?", userID).Find(&folders)

	folderMap := make(map[uint]*model.Folder)
	var rootFolders []model.Folder

	for i := range folders {
		folderMap[folders[i].ID] = &folders[i]
	}

	for i := range folders {
		if folders[i].ParentID == nil {
			rootFolders = append(rootFolders, folders[i])
		} else {
			if _, exists := folderMap[*folders[i].ParentID]; exists {
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": folders,
	})
}
