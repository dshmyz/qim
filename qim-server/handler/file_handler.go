package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"qim-server/database"
	"qim-server/di"
	"qim-server/model"
	"qim-server/pkg/logger"
	"qim-server/pkg/response"
	"qim-server/service/storage"

	"github.com/gin-gonic/gin"
)

const defaultMaxUploadSize = 500 * 1024 * 1024 // 500MB default

var (
	defaultAllowedExtensions = map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".webp": true,
		".pdf": true,
		".doc": true, ".docx": true,
		".xls": true, ".xlsx": true,
		".ppt": true, ".pptx": true,
		".txt": true, ".md": true, ".csv": true,
		".zip": true, ".rar": true, ".7z": true,
		".mp3": true, ".wav": true, ".mp4": true, ".avi": true, ".mov": true,
	}
	uploadConfigCache   map[string]interface{}
	uploadConfigMu      sync.RWMutex
	uploadConfigExpires time.Time
)

type uploadConfig struct {
	MaxSize           int64
	AllowedExtensions map[string]bool
}

func getUploadConfig() *uploadConfig {
	uploadConfigMu.RLock()
	if uploadConfigCache != nil && time.Now().Before(uploadConfigExpires) {
		maxSize := int64(defaultMaxUploadSize)
		if v, ok := uploadConfigCache["file_upload:max_size"]; ok {
			if n, err := strconv.ParseInt(fmt.Sprintf("%v", v), 10, 64); err == nil {
				maxSize = n
			}
		}
		var allowed map[string]bool
		if v, ok := uploadConfigCache["file_upload:allowed_extensions"]; ok {
			allowed = map[string]bool{}
			if s, ok := v.(string); ok {
				var exts []string
				if err := json.Unmarshal([]byte(s), &exts); err == nil {
					for _, e := range exts {
						allowed[strings.ToLower(e)] = true
					}
				}
			}
		}
		if allowed == nil {
			allowed = defaultAllowedExtensions
		}
		result := &uploadConfig{MaxSize: maxSize, AllowedExtensions: allowed}
		uploadConfigMu.RUnlock()
		return result
	}
	uploadConfigMu.RUnlock()

	db := database.GetDB()
	var configs []model.SystemConfig
	db.Where("config_key IN ?", []string{"file_upload:max_size", "file_upload:allowed_extensions"}).Find(&configs)

	cache := map[string]interface{}{}
	for _, c := range configs {
		cache[c.ConfigKey] = c.Value
	}

	uploadConfigMu.Lock()
	uploadConfigCache = cache
	uploadConfigExpires = time.Now().Add(5 * time.Minute)
	uploadConfigMu.Unlock()

	maxSize := int64(defaultMaxUploadSize)
	if v, ok := cache["file_upload:max_size"]; ok {
		if n, err := strconv.ParseInt(fmt.Sprintf("%v", v), 10, 64); err == nil {
			maxSize = n
		}
	}
	allowed := defaultAllowedExtensions
	if v, ok := cache["file_upload:allowed_extensions"]; ok {
		if s, ok := v.(string); ok {
			var exts []string
			if err := json.Unmarshal([]byte(s), &exts); err == nil {
				allowed = map[string]bool{}
				for _, e := range exts {
					allowed[strings.ToLower(e)] = true
				}
			}
		}
	}

	return &uploadConfig{MaxSize: maxSize, AllowedExtensions: allowed}
}

func invalidateUploadConfigCache() {
	uploadConfigMu.Lock()
	uploadConfigCache = nil
	uploadConfigMu.Unlock()
}

type fileStatsCacheEntry struct {
	data      gin.H
	expiredAt time.Time
}

var fileStatsCache = make(map[uint]fileStatsCacheEntry)
var fileStatsCacheMu sync.RWMutex

func invalidateFileStatsCache(userID uint) {
	fileStatsCacheMu.Lock()
	delete(fileStatsCache, userID)
	fileStatsCacheMu.Unlock()
}

func UploadFile(c *gin.Context) {
	// 检查文件上传开关
	if cfg, err := di.GlobalContainer.SystemConfigService.GetConfig("enableFileUpload"); err == nil && cfg.Value == "false" {
		response.Forbidden(c, "文件上传功能已关闭")
		return
	}

	userID, _ := c.Get("user_id")

	ucfg := getUploadConfig()
	maxMB := ucfg.MaxSize / (1024 * 1024)

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, ucfg.MaxSize)

	file, err := c.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			response.BadRequest(c, fmt.Sprintf("文件过大，最大支持%dMB", maxMB))
			return
		}
		response.BadRequest(c, "文件上传失败")
		return
	}

	if file.Size > ucfg.MaxSize {
		response.BadRequest(c, fmt.Sprintf("文件过大，最大支持%dMB", maxMB))
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !ucfg.AllowedExtensions[ext] {
		response.BadRequest(c, "不支持的文件类型")
		return
	}

	source := c.DefaultPostForm("source", "upload")

	filename := time.Now().Format("20060102150405") + "_" + strconv.FormatUint(uint64(userID.(uint)), 10) + ext
	key := "uploads/" + filename
	mimeType := file.Header.Get("Content-Type")

	st := di.GlobalContainer.DefaultStorage
	if st == nil {
		response.InternalServerError(c, "存储服务未初始化")
		return
	}

	fileData, err := file.Open()
	if err != nil {
		response.InternalServerError(c, "打开文件失败")
		return
	}
	defer fileData.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := st.Put(ctx, key, fileData, file.Size, mimeType); err != nil {
		response.InternalServerError(c, "存储文件失败")
		return
	}

	svc := di.GlobalContainer.FileService
	fileRecord := model.File{
		Name:         file.Filename,
		OriginalName: file.Filename,
		StoragePath:  storage.BuildPath(st.Kind(), key),
		Size:         file.Size,
		MimeType:     mimeType,
		UserID:       userID.(uint),
		Source:       source,
		CreatedAt:    time.Now(),
	}
	if err := svc.CreateFile(&fileRecord); err != nil {
		response.InternalServerError(c, "创建文件记录失败")
		return
	}

	invalidateFileStatsCache(userID.(uint))

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

func GetFiles(c *gin.Context) {
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

	filters := map[string]string{
		"folder_id":  c.Query("folder_id"),
		"source":     c.Query("source"),
		"starred":    c.Query("starred"),
		"type":       c.Query("type"),
		"search":     c.Query("search"),
		"sort_by":    c.DefaultQuery("sort_by", "created_at"),
		"sort_order": c.DefaultQuery("sort_order", "desc"),
		"date_from":  c.Query("date_from"),
		"date_to":    c.Query("date_to"),
	}

	svc := di.GlobalContainer.FileService
	files, total, err := svc.GetFiles(userID.(uint), page, pageSize, filters)
	if err != nil {
		response.InternalServerError(c, "获取文件列表失败")
		return
	}

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
		response.BadRequest(c, "无效的文件ID")
		return
	}

	var req struct {
		Name     *string `json:"name"`
		FolderID *uint   `json:"folder_id"`
		Tags     *string `json:"tags"`
	}

	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := di.GlobalContainer.FileService
	_, err = svc.GetFile(userID.(uint), uint(fileID))
	if err != nil {
		response.NotFound(c, "文件不存在")
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		if *req.Name == "" {
			response.BadRequest(c, "文件名不能为空")
			return
		}
		updates["name"] = *req.Name
	}
	if req.FolderID != nil {
		_, folderErr := svc.GetFolder(userID.(uint), *req.FolderID)
		if folderErr != nil {
			response.BadRequest(c, "文件夹不存在或无权限")
			return
		}
		updates["folder_id"] = *req.FolderID
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}

	if len(updates) == 0 {
		response.BadRequest(c, "没有需要更新的字段")
		return
	}

	file, updateErr := svc.UpdateFile(userID.(uint), uint(fileID), updates)
	if updateErr != nil {
		response.InternalServerError(c, "更新文件失败")
		return
	}

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
		response.BadRequest(c, "无效的文件ID")
		return
	}

	svc := di.GlobalContainer.FileService
	file, err := svc.ToggleStar(userID.(uint), uint(fileID))
	if err != nil {
		response.NotFound(c, "文件不存在")
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
		FileIDs        []uint `json:"file_ids" binding:"required"`
		Operation      string `json:"operation" binding:"required"`
		TargetFolderID *uint  `json:"target_folder_id"`
	}

	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if len(req.FileIDs) == 0 {
		response.BadRequest(c, "文件列表不能为空")
		return
	}

	svc := di.GlobalContainer.FileService

	switch req.Operation {
	case "delete":
		count, err := svc.BatchDelete(userID.(uint), req.FileIDs)
		if err != nil {
			response.InternalServerError(c, "批量删除失败")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "批量删除成功",
			"data":    gin.H{"deleted_count": count},
		})

	case "move":
		if req.TargetFolderID == nil {
			response.BadRequest(c, "移动操作需要指定目标文件夹")
			return
		}
		_, folderErr := svc.GetFolder(userID.(uint), *req.TargetFolderID)
		if folderErr != nil {
			response.BadRequest(c, "目标文件夹不存在或无权限")
			return
		}
		count, err := svc.BatchMove(userID.(uint), req.FileIDs, *req.TargetFolderID)
		if err != nil {
			response.InternalServerError(c, "批量移动失败")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "批量移动成功",
			"data":    gin.H{"moved_count": count},
		})

	case "star":
		count, err := svc.BatchStar(userID.(uint), req.FileIDs, true)
		if err != nil {
			response.InternalServerError(c, "批量星标失败")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "批量星标成功",
			"data":    gin.H{"starred_count": count},
		})

	case "unstar":
		count, err := svc.BatchStar(userID.(uint), req.FileIDs, false)
		if err != nil {
			response.InternalServerError(c, "批量取消星标失败")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "批量取消星标成功",
			"data":    gin.H{"unstarred_count": count},
		})

	default:
		response.BadRequest(c, "不支持的操作类型")
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

	svc := di.GlobalContainer.FileService
	files, total, err := svc.GetStarredFiles(userID.(uint), page, pageSize)
	if err != nil {
		response.InternalServerError(c, "获取星标文件失败")
		return
	}

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
	uid := userID.(uint)

	fileStatsCacheMu.RLock()
	cached, found := fileStatsCache[uid]
	fileStatsCacheMu.RUnlock()

	if found && time.Now().Before(cached.expiredAt) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": cached.data,
		})
		return
	}

	svc := di.GlobalContainer.FileService
	stats, err := svc.GetFileStats(uid)
	if err != nil {
		response.InternalServerError(c, "获取文件统计失败")
		return
	}

	resultData := gin.H{
		"total_files":   stats.TotalFiles,
		"starred_files": stats.StarredFiles,
		"total_size":    stats.TotalSize,
		"folder_count":  stats.FolderCount,
		"type_stats":    stats.TypeStats,
	}

	fileStatsCacheMu.Lock()
	fileStatsCache[uid] = fileStatsCacheEntry{
		data:      resultData,
		expiredAt: time.Now().Add(5 * time.Minute),
	}
	fileStatsCacheMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": resultData,
	})
}

func GetFolderTree(c *gin.Context) {
	userID, _ := c.Get("user_id")

	parentIDStr := c.Query("parent_id")
	svc := di.GlobalContainer.FileService

	var parentID *uint
	if parentIDStr != "" {
		pid, err := strconv.ParseUint(parentIDStr, 10, 32)
		if err != nil {
			response.BadRequest(c, "无效的父文件夹ID")
			return
		}
		pidVal := uint(pid)
		parentID = &pidVal
	}

	folders, err := svc.GetFolderTree(userID.(uint), parentID)
	if err != nil {
		response.InternalServerError(c, "获取文件夹树失败")
		return
	}

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
		response.BadRequest(c, "无效的文件夹ID")
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
		response.BadRequest(c, "参数错误")
		return
	}

	svc := di.GlobalContainer.FileService
	folder, err := svc.GetFolder(userID.(uint), uint(folderID))
	if err != nil {
		response.NotFound(c, "文件夹不存在")
		return
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		if *req.Name == "" {
			response.BadRequest(c, "文件夹名称不能为空")
			return
		}
		updates["name"] = *req.Name
	}
	if req.ParentID != nil {
		if *req.ParentID == folder.ID {
			response.BadRequest(c, "不能将文件夹移动到自己下面")
			return
		}
		if *req.ParentID != 0 {
			_, parentErr := svc.GetFolder(userID.(uint), *req.ParentID)
			if parentErr != nil {
				response.BadRequest(c, "父文件夹不存在或无权限")
				return
			}
			if svc.IsDescendant(userID.(uint), *req.ParentID, folder.ID) {
				response.BadRequest(c, "不能将文件夹移动到其子文件夹下")
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
		response.BadRequest(c, "没有需要更新的字段")
		return
	}

	updatedFolder, updateErr := svc.UpdateFolder(userID.(uint), uint(folderID), updates)
	if updateErr != nil {
		response.InternalServerError(c, "更新文件夹失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新文件夹成功",
		"data":    updatedFolder,
	})
}

func DeleteFolder(c *gin.Context) {
	userID, _ := c.Get("user_id")
	folderIDStr := c.Param("id")

	folderID, err := strconv.ParseUint(folderIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文件夹ID")
		return
	}

	recursive := c.Query("recursive") == "true"
	svc := di.GlobalContainer.FileService
	uid := userID.(uint)
	fid := uint(folderID)

	_, err = svc.GetFolder(uid, fid)
	if err != nil {
		response.NotFound(c, "文件夹不存在")
		return
	}

	childCount, _ := svc.GetFolderChildCount(uid, fid)
	if childCount > 0 && !recursive {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "文件夹包含子文件夹，请使用递归删除或先移走子文件夹",
		})
		return
	}

	fileCount, _ := svc.GetFolderFileCount(uid, fid)
	if fileCount > 0 && !recursive {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "文件夹包含文件，请使用递归删除或先移走文件",
		})
		return
	}

	if recursive {
		svc.DeleteFolderRecursive(uid, fid)
		svc.DeleteFolderFiles(uid, fid)
	}

	if deleteErr := svc.DeleteFolder(uid, fid); deleteErr != nil {
		response.InternalServerError(c, "删除文件夹失败")
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
		response.BadRequest(c, "无效的文件夹ID")
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

	svc := di.GlobalContainer.FileService
	uid := userID.(uint)
	fid := uint(folderID)

	folder, err := svc.GetFolder(uid, fid)
	if err != nil {
		response.NotFound(c, "文件夹不存在")
		return
	}

	files, total, err := svc.GetFolderFiles(uid, fid, page, pageSize)
	if err != nil {
		response.InternalServerError(c, "获取文件夹文件失败")
		return
	}

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

func DownloadFile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fileIDStr := c.Param("id")

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文件ID")
		return
	}

	svc := di.GlobalContainer.FileService
	file, err := svc.GetFile(userID.(uint), uint(fileID))
	if err != nil {
		response.NotFound(c, "文件不存在")
		return
	}

	mgr := di.GlobalContainer.StorageManager
	st, key, ok := mgr.ByPath(file.StoragePath)
	if !ok || st == nil {
		response.InternalServerError(c, "存储类型不支持")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	reader, err := st.Get(ctx, key)
	if err != nil {
		response.InternalServerError(c, "读取文件失败")
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Length", fmt.Sprintf("%d", file.Size))

	if _, err := io.Copy(c.Writer, reader); err != nil {
		logger.WithModule("FileHandler").Error("下载文件失败", "error", err)
		return
	}
}

func PreviewFile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fileIDStr := c.Param("id")

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文件ID")
		return
	}

	svc := di.GlobalContainer.FileService
	file, err := svc.GetFile(userID.(uint), uint(fileID))
	if err != nil {
		response.NotFound(c, "文件不存在")
		return
	}

	thumbnail := c.Query("thumbnail") == "true"

	c.Header("Cache-Control", "public, max-age=86400")

	mgr := di.GlobalContainer.StorageManager
	st, key, ok := mgr.ByPath(file.StoragePath)
	if !ok || st == nil {
		response.InternalServerError(c, "存储类型不支持")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	reader, err := st.Get(ctx, key)
	if err != nil {
		response.InternalServerError(c, "读取文件失败")
		return
	}
	defer reader.Close()

	c.Header("Content-Type", file.MimeType)
	if !thumbnail {
		c.Header("Content-Disposition", "inline; filename=\""+file.Name+"\"")
	}

	if _, err := io.Copy(c.Writer, reader); err != nil {
		logger.WithModule("FileHandler").Error("预览文件失败", "error", err)
		return
	}
}

func DeleteFile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	fileIDStr := c.Param("id")

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文件ID")
		return
	}

	svc := di.GlobalContainer.FileService
	file, err := svc.GetFile(userID.(uint), uint(fileID))
	if err != nil {
		response.NotFound(c, "文件不存在")
		return
	}

	mgr := di.GlobalContainer.StorageManager
	st, key, ok := mgr.ByPath(file.StoragePath)
	if !ok || st == nil {
		response.InternalServerError(c, "存储类型不支持")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := st.Delete(ctx, key); err != nil {
		logger.WithModule("FileHandler").Error("删除物理文件失败", "error", err)
		response.InternalServerError(c, "删除文件失败")
		return
	}

	if err := svc.DeleteFile(userID.(uint), uint(fileID)); err != nil {
		logger.WithModule("FileHandler").Error("删除文件记录失败，但物理文件已删除", "file_id", fileID, "error", err)
		response.InternalServerError(c, "删除文件记录失败")
		return
	}

	invalidateFileStatsCache(userID.(uint))
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除文件成功",
	})
}

func CreateFolder(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name     string `json:"name" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	svc := di.GlobalContainer.FileService
	folder := &model.Folder{
		UserID:   userID.(uint),
		Name:     req.Name,
		ParentID: req.ParentID,
	}
	if err := svc.CreateFolder(folder); err != nil {
		response.InternalServerError(c, "创建文件夹失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": folder,
	})
}
