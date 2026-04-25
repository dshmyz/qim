package handler

import (
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

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件上传失败"})
		return
	}

	ext := filepath.Ext(file.Filename)
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
				"name": file.Filename,
				"size": file.Size,
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
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存文件失败"})
			return
		}

		db := database.GetDB()
		fileRecord := model.File{
			Name:         file.Filename,
			OriginalName: file.Filename,
			StoragePath:  "/uploads/" + filename,
			Size:         file.Size,
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
