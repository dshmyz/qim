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
	"gorm.io/gorm"
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
			Source:       "upload",
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

	// 分页参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 过滤参数
	folderIDStr := c.Query("folder_id")
	source := c.Query("source")
	starred := c.Query("starred")
	fileType := c.Query("type")
	search := c.Query("search")

	db := database.GetDB()
	query := db.Model(&model.File{}).Where("user_id = ?", userID)

	// 按文件夹过滤
	if folderIDStr != "" {
		folderID, parseErr := strconv.ParseUint(folderIDStr, 10, 32)
		if parseErr == nil {
			query = query.Where("folder_id = ?", uint(folderID))
		}
	}

	// 按来源过滤
	if source != "" {
		query = query.Where("source = ?", source)
	}

	// 按星标过滤
	if starred == "true" {
		query = query.Where("is_starred = ?", true)
	} else if starred == "false" {
		query = query.Where("is_starred = ?", false)
	}

	// 按文件类型过滤（通过 MIME 类型前缀）
	if fileType != "" {
		switch fileType {
		case "image":
			query = query.Where("mime_type LIKE ?", "image/%")
		case "video":
			query = query.Where("mime_type LIKE ?", "video/%")
		case "audio":
			query = query.Where("mime_type LIKE ?", "audio/%")
		case "document":
			query = query.Where("mime_type LIKE ? OR mime_type LIKE ? OR mime_type LIKE ? OR mime_type LIKE ?",
				"application/pdf", "application/msword", "application/vnd.ms-excel", "text/%")
		default:
			query = query.Where("mime_type LIKE ?", fileType+"/%")
		}
	}

	// 按文件名搜索
	if search != "" {
		query = query.Where("name LIKE ? OR original_name LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 统计总数
	var total int64
	query.Count(&total)

	// 分页查询
	var files []model.File
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&files)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"files":     files,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func UpdateFile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fileIDStr := c.Param("id")

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件ID"})
		return
	}

	var req struct {
		Name     *string `json:"name"`
		FolderID *uint   `json:"folder_id"`
		Tags     *string `json:"tags"`
	}

	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var file model.File
	if findErr := db.Where("id = ? AND user_id = ?", uint(fileID), userID).First(&file).Error; findErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在"})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		if *req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件名不能为空"})
			return
		}
		updates["name"] = *req.Name
	}
	if req.FolderID != nil {
		// 验证文件夹属于当前用户
		var folder model.Folder
		if folderFindErr := db.Where("id = ? AND user_id = ?", *req.FolderID, userID).First(&folder).Error; folderFindErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件夹不存在或无权限"})
			return
		}
		updates["folder_id"] = *req.FolderID
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "没有需要更新的字段"})
		return
	}

	if updateErr := db.Model(&file).Updates(updates).Error; updateErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新文件失败"})
		return
	}

	// 获取更新后的文件记录
	db.First(&file, file.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新文件成功",
		"data":    file,
	})
}

func ToggleStar(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fileIDStr := c.Param("id")

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件ID"})
		return
	}

	db := database.GetDB()
	var file model.File
	if findErr := db.Where("id = ? AND user_id = ?", uint(fileID), userID).First(&file).Error; findErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件不存在"})
		return
	}

	now := time.Now()
	if file.IsStarred {
		// 取消星标
		file.IsStarred = false
		file.StarredAt = nil
	} else {
		// 添加星标
		file.IsStarred = true
		file.StarredAt = &now
	}

	if saveErr := db.Save(&file).Error; saveErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "操作失败"})
		return
	}

	message := "已添加星标"
	if !file.IsStarred {
		message = "已取消星标"
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": message,
		"data":    file,
	})
}

func BatchOperation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		FileIDs  []uint `json:"file_ids" binding:"required"`
		Operation string `json:"operation" binding:"required"` // "delete", "move", "star", "unstar"
		TargetFolderID *uint `json:"target_folder_id"`       // 用于 move 操作
	}

	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if len(req.FileIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件列表不能为空"})
		return
	}

	db := database.GetDB()

	switch req.Operation {
	case "delete":
		result := db.Where("id IN ? AND user_id = ?", req.FileIDs, userID).Delete(&model.File{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "批量删除失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "批量删除成功",
			"data": gin.H{
				"deleted_count": result.RowsAffected,
			},
		})

	case "move":
		if req.TargetFolderID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "移动操作需要指定目标文件夹"})
			return
		}

		var folder model.Folder
		if folderFindErr := db.Where("id = ? AND user_id = ?", *req.TargetFolderID, userID).First(&folder).Error; folderFindErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "目标文件夹不存在或无权限"})
			return
		}

		result := db.Model(&model.File{}).
			Where("id IN ? AND user_id = ?", req.FileIDs, userID).
			Update("folder_id", *req.TargetFolderID)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "批量移动失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "批量移动成功",
			"data": gin.H{
				"moved_count": result.RowsAffected,
			},
		})

	case "star":
		now := time.Now()
		result := db.Model(&model.File{}).
			Where("id IN ? AND user_id = ?", req.FileIDs, userID).
			Updates(map[string]interface{}{
				"is_starred": true,
				"starred_at": now,
			})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "批量星标失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "批量星标成功",
			"data": gin.H{
				"starred_count": result.RowsAffected,
			},
		})

	case "unstar":
		result := db.Model(&model.File{}).
			Where("id IN ? AND user_id = ?", req.FileIDs, userID).
			Updates(map[string]interface{}{
				"is_starred": false,
				"starred_at": nil,
			})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "批量取消星标失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "批量取消星标成功",
			"data": gin.H{
				"unstarred_count": result.RowsAffected,
			},
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的操作类型"})
	}
}

func GetStarredFiles(c *gin.Context) {
	userID, _ := c.Get("user_id")

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	db := database.GetDB()
	var total int64
	db.Model(&model.File{}).Where("user_id = ? AND is_starred = ?", userID, true).Count(&total)

	var files []model.File
	offset := (page - 1) * pageSize
	db.Where("user_id = ? AND is_starred = ?", userID, true).
		Order("starred_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&files)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"files":     files,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func GetFileStats(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()

	// 总文件数
	var totalFiles int64
	db.Model(&model.File{}).Where("user_id = ?", userID).Count(&totalFiles)

	// 星标文件数
	var starredFiles int64
	db.Model(&model.File{}).Where("user_id = ? AND is_starred = ?", userID, true).Count(&starredFiles)

	// 总大小
	var totalSize int64
	db.Model(&model.File{}).Where("user_id = ?", userID).Select("COALESCE(SUM(size), 0)").Scan(&totalSize)

	// 按类型统计
	type FileTypeInfo struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
		Size  int64  `json:"size"`
	}
	var typeStats []FileTypeInfo
	db.Model(&model.File{}).
		Where("user_id = ?", userID).
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
		`).
		Scan(&typeStats)

	// 文件夹数
	var folderCount int64
	db.Model(&model.Folder{}).Where("user_id = ?", userID).Count(&folderCount)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total_files":   totalFiles,
			"starred_files": starredFiles,
			"total_size":    totalSize,
			"folder_count":  folderCount,
			"type_stats":    typeStats,
		},
	})
}

func GetFolderTree(c *gin.Context) {
	userID, _ := c.Get("user_id")

	// 支持懒加载模式：如果传了 parent_id 参数，只返回该父节点下的子文件夹
	parentIDStr := c.Query("parent_id")

	db := database.GetDB()
	var folders []model.Folder

	query := db.Where("user_id = ?", userID)

	if parentIDStr != "" {
		parentID, err := strconv.ParseUint(parentIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的父文件夹ID"})
			return
		}
		query = query.Where("parent_id = ?", uint(parentID))
	} else {
		// 没有 parent_id 时，返回根级文件夹
		query = query.Where("parent_id IS NULL")
	}

	query.Order("sort_order ASC, created_at ASC").Find(&folders)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": folders,
	})
}

func UpdateFolder(c *gin.Context) {
	userID, _ := c.Get("user_id")
	folderIDStr := c.Param("id")

	folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件夹ID"})
		return
	}

	var req struct {
		Name      *string `json:"name"`
		ParentID  *uint   `json:"parent_id"`
		SortOrder *int    `json:"sort_order"`
		Icon      *string `json:"icon"`
		Color     *string `json:"color"`
	}

	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()
	var folder model.Folder
	if findErr := db.Where("id = ? AND user_id = ?", uint(folderID), userID).First(&folder).Error; findErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件夹不存在"})
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		if *req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "文件夹名称不能为空"})
			return
		}
		updates["name"] = *req.Name
	}
	if req.ParentID != nil {
		// 防止将文件夹移动到自己或其子文件夹下（循环引用检测）
		if *req.ParentID == folder.ID {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不能将文件夹移动到自己下面"})
			return
		}
		if *req.ParentID != 0 {
			var parent model.Folder
			if parentFindErr := db.Where("id = ? AND user_id = ?", *req.ParentID, userID).First(&parent).Error; parentFindErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "父文件夹不存在或无权限"})
				return
			}
			// 检查是否会造成循环引用
			if isDescendant(db, *req.ParentID, folder.ID, userID.(uint)) {
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不能将文件夹移动到其子文件夹下"})
				return
			}
		}
		updates["parent_id"] = *req.ParentID
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.Icon != nil {
		updates["icon"] = *req.Icon
	}
	if req.Color != nil {
		updates["color"] = *req.Color
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "没有需要更新的字段"})
		return
	}

	if updateErr := db.Model(&folder).Updates(updates).Error; updateErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新文件夹失败"})
		return
	}

	db.First(&folder, folder.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新文件夹成功",
		"data":    folder,
	})
}

func DeleteFolder(c *gin.Context) {
	userID, _ := c.Get("user_id")
	folderIDStr := c.Param("id")

	folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件夹ID"})
		return
	}

	recursive := c.Query("recursive") == "true"

	db := database.GetDB()
	var folder model.Folder
	if findErr := db.Where("id = ? AND user_id = ?", uint(folderID), userID).First(&folder).Error; findErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件夹不存在"})
		return
	}

	// 检查是否有子文件夹
	var childCount int64
	db.Model(&model.Folder{}).Where("user_id = ? AND parent_id = ?", userID, folder.ID).Count(&childCount)

	if childCount > 0 && !recursive {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "文件夹包含子文件夹，请使用递归删除或先移走子文件夹",
		})
		return
	}

	// 检查是否有文件
	var fileCount int64
	db.Model(&model.File{}).Where("user_id = ? AND folder_id = ?", userID, folder.ID).Count(&fileCount)

	if fileCount > 0 && !recursive {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "文件夹包含文件，请使用递归删除或先移走文件",
		})
		return
	}

	if recursive {
		// 递归删除所有子文件夹
		deleteFolderRecursive(db, folder.ID, userID.(uint))

		// 删除文件夹下的所有文件记录
		db.Where("user_id = ? AND folder_id = ?", userID, folder.ID).Delete(&model.File{})
	}

	// 删除文件夹本身
	if deleteErr := db.Delete(&folder).Error; deleteErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除文件夹失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除文件夹成功",
	})
}

func GetFolderFiles(c *gin.Context) {
	userID, _ := c.Get("user_id")
	folderIDStr := c.Param("id")

	folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的文件夹ID"})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	db := database.GetDB()

	// 验证文件夹存在且属于当前用户
	var folder model.Folder
	if findErr := db.Where("id = ? AND user_id = ?", uint(folderID), userID).First(&folder).Error; findErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "文件夹不存在"})
		return
	}

	var total int64
	db.Model(&model.File{}).Where("user_id = ? AND folder_id = ?", userID, folder.ID).Count(&total)

	var files []model.File
	offset := (page - 1) * pageSize
	db.Where("user_id = ? AND folder_id = ?", userID, folder.ID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&files)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"folder":    folder,
			"files":     files,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}


// isDescendant 检查 targetID 是否是 ancestorID 的后代（用于循环引用检测）
func isDescendant(db *gorm.DB, targetID uint, ancestorID uint, userID uint) bool {
	currentID := targetID
	visited := make(map[uint]bool)

	for currentID != 0 {
		if visited[currentID] {
			return false // 防止无限循环
		}
		visited[currentID] = true

		if currentID == ancestorID {
			return true
		}

		var folder model.Folder
		if err := db.Where("id = ? AND user_id = ?", currentID, userID).First(&folder).Error; err != nil {
			return false
		}

		if folder.ParentID == nil {
			return false
		}
		currentID = *folder.ParentID
	}

	return false
}

// deleteFolderRecursive 递归删除文件夹及其所有子文件夹
func deleteFolderRecursive(db *gorm.DB, folderID uint, userID uint) {
	var children []model.Folder
	db.Where("user_id = ? AND parent_id = ?", userID, folderID).Find(&children)

	for _, child := range children {
		// 递归删除子文件夹下的文件
		db.Where("user_id = ? AND folder_id = ?", userID, child.ID).Delete(&model.File{})
		// 递归删除子文件夹
		deleteFolderRecursive(db, child.ID, userID)
		// 删除子文件夹本身
		db.Delete(&child)
	}
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
