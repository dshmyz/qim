package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/pkg/upload"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru/v2"
	"gorm.io/gorm"
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
		".exe": true, ".msi": true, ".dmg": true, ".pkg": true, ".AppImage": true, ".deb": true, ".rpm": true,
	}
	uploadConfigCache   map[string]interface{}
	uploadConfigMu      sync.RWMutex
	uploadConfigExpires time.Time

	// protectedUploadSources 是允许通过 PublicDownloadFile 免鉴权公开下载的文件来源。
	// 这些来源仅可由管理员在上传时设置，普通用户不得借此公开私有文件。
	protectedUploadSources = map[string]bool{
		"client_update": true,
		"version":       true,
	}
)

// hasRole 判断当前请求的认证用户是否具备指定角色（角色由 auth 中间件写入 context）。
func hasRole(c *gin.Context, role string) bool {
	raw, ok := c.Get("roles")
	if !ok {
		return false
	}
	roles, ok := raw.([]string)
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

type uploadConfig struct {
	MaxSize           int64
	AllowedExtensions map[string]bool
	EnableTypeCheck   bool
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
		enableTypeCheck := false
		if v, ok := uploadConfigCache["file_upload:enable_type_check"]; ok {
			if s, ok := v.(string); ok {
				enableTypeCheck = s == "true"
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
		result := &uploadConfig{MaxSize: maxSize, AllowedExtensions: allowed, EnableTypeCheck: enableTypeCheck}
		uploadConfigMu.RUnlock()
		return result
	}
	uploadConfigMu.RUnlock()

	db := database.GetDB()
	var configs []model.SystemConfig
	db.Where("config_key IN ?", []string{"file_upload:max_size", "file_upload:allowed_extensions", "file_upload:enable_type_check"}).Find(&configs)

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
	enableTypeCheck := false
	if v, ok := cache["file_upload:enable_type_check"]; ok {
		if s, ok := v.(string); ok {
			enableTypeCheck = s == "true"
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

	return &uploadConfig{MaxSize: maxSize, AllowedExtensions: allowed, EnableTypeCheck: enableTypeCheck}
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

// fileStatsCache 使用 LRU 缓存避免用户量大时内存无限增长。
// 容量 1000 足够覆盖活跃用户，超出后自动淘汰最久未访问的条目。
var fileStatsCache, _ = lru.New[uint, fileStatsCacheEntry](1000)

func invalidateFileStatsCache(userID uint) {
	fileStatsCache.Remove(userID)
}

// sanitizeFilename 生成安全的 Content-Disposition 头值，使用 RFC 5987 编码文件名。
// 格式：attachment; filename="sanitized"; filename*=UTF-8”encoded
func sanitizeFilename(filename string) string {
	safeName := sanitizeFallbackName(filename)
	encoded := url.PathEscape(filename)
	return fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", safeName, encoded)
}

// sanitizeInlineFilename 生成安全的 inline Content-Disposition 头值
func sanitizeInlineFilename(filename string) string {
	safeName := sanitizeFallbackName(filename)
	encoded := url.PathEscape(filename)
	return fmt.Sprintf("inline; filename=\"%s\"; filename*=UTF-8''%s", safeName, encoded)
}

// sanitizeFallbackName 生成 ASCII 安全的 fallback 文件名，用于不支持 RFC 5987 的客户端
func sanitizeFallbackName(filename string) string {
	var safe strings.Builder
	for _, r := range filename {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			safe.WriteRune(r)
		}
	}
	result := safe.String()
	if result == "" {
		return "download"
	}
	return result
}

func UploadFile(c *gin.Context) {
	// 统一上传开关检查
	if !upload.IsUploadEnabled(func() (string, error) {
		cfg, err := di.GlobalContainer.SystemConfigService.GetConfig("enableFileUpload")
		if err != nil {
			return "", err
		}
		return cfg.Value, nil
	}) {
		response.Forbidden(c, "文件上传功能已关闭")
		return
	}

	userID, _ := c.Get("user_id")

	ucfg := getUploadConfig()
	policy := upload.NewPolicy(ucfg.MaxSize, ucfg.AllowedExtensions, ucfg.EnableTypeCheck)
	maxMB := ucfg.MaxSize / (1024 * 1024)

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, ucfg.MaxSize)

	// source 权限检查提前到文件读取之前，避免无权用户浪费 IO
	source := c.DefaultPostForm("source", "upload")
	if protectedUploadSources[source] && !hasRole(c, "system_admin") {
		response.Forbidden(c, "无权使用该文件来源")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			response.BadRequest(c, fmt.Sprintf("文件过大，最大支持%dMB", maxMB))
			return
		}
		response.BadRequest(c, "文件上传失败")
		return
	}

	// 统一大小校验
	if err := policy.ValidateSize(file.Size); err != nil {
		response.BadRequest(c, fmt.Sprintf("文件过大，最大支持%dMB", maxMB))
		return
	}

	st := di.GlobalContainer.DefaultStorage
	if st == nil {
		response.InternalServerError(c, "存储服务未初始化")
		return
	}

	// 生成存储文件名（时间戳+userID+扩展名）
	uid, _ := userID.(uint)
	now := time.Now()
	ext := strings.ToLower(filepath.Ext(file.Filename))

	// 个人存储配额检查（读取/保存文件之前拦截，超限不浪费 IO 与存储）
	fileSvc := di.GlobalContainer.FileService
	if fileSvc != nil {
		used, err := fileSvc.GetStorageUsage(uid)
		if err == nil {
			if quota, qerr := fileSvc.GetUserQuota(uid); qerr == nil && used+file.Size > quota {
				response.BadRequest(c, fmt.Sprintf("存储空间不足：已用 %s / 配额 %s", formatBytes(used), formatBytes(quota)))
				return
			}
		}
	}

	// 复用公共"读取+校验+存储"函数
	// 受信任的管理员分发来源（version / client_update）跳过类型校验，
	// 允许发布 .exe 等 CLI/MCP/客户端安装包；此来源已在上方强制要求 system_admin 权限。
	saved, err := upload.SaveMultipartFile(file, upload.SaveConfig{
		Policy:        policy,
		Storage:       st,
		KeyPrefix:     fmt.Sprintf("uploads/%s", now.Format("2006/01")),
		SkipTypeCheck: protectedUploadSources[source],
		FilenameFn: func() string {
			return fmt.Sprintf("%s%03d_%d%s", now.Format("20060102150405"), now.UnixMilli()%1000, uid, ext)
		},
		ContextFn: func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(c.Request.Context(), 30*time.Second)
		},
	})
	if err != nil {
		if ve, ok := err.(*upload.ValidateError); ok {
			if ve.Field == "size" {
				response.BadRequest(c, fmt.Sprintf("文件过大，最大支持%dMB", maxMB))
				return
			}
			response.BadRequest(c, ve.Error())
			return
		}
		response.InternalServerError(c, "保存文件失败: "+err.Error())
		return
	}

	svc := di.GlobalContainer.FileService

	// folder_id 可选：上传到指定文件夹（缺省或非法时落根）
	var folderID *uint
	if fidStr := c.PostForm("folder_id"); fidStr != "" {
		if fid, err := strconv.ParseUint(fidStr, 10, 64); err == nil && fid > 0 {
			folderID = new(uint)
			*folderID = uint(fid)
		}
	}
	if folderID != nil {
		if _, err := svc.GetFolder(uid, *folderID); err != nil {
			response.BadRequest(c, "目标文件夹不存在或无权限")
			return
		}
	}

	fileRecord := model.File{
		Name:         saved.SafeName,
		OriginalName: saved.SafeName,
		StoragePath:  saved.StoragePath,
		Size:         saved.Size, // 用实际读取大小，不信任客户端 file.Size
		MimeType:     saved.MimeType,
		UserID:       uid,
		Source:       source,
		FolderID:     folderID,
		CreatedAt:    time.Now(),
	}
	if err := svc.CreateFile(&fileRecord); err != nil {
		// 存储成功但建记录失败，回滚已上传文件，避免孤儿文件
		saved.Cleanup(st)
		response.InternalServerError(c, "创建文件记录失败")
		return
	}

	invalidateFileStatsCache(uid)

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

// GetStorageUsage 查询当前用户的存储用量与配额（侧栏容量条数据源）
func GetStorageUsage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)

	svc := di.GlobalContainer.FileService
	if svc == nil {
		response.InternalServerError(c, "文件服务未初始化")
		return
	}
	used, err := svc.GetStorageUsage(uid)
	if err != nil {
		response.InternalServerError(c, "查询存储用量失败")
		return
	}
	quota, err := svc.GetUserQuota(uid)
	if err != nil {
		response.InternalServerError(c, "查询存储配额失败")
		return
	}
	response.Success(c, gin.H{"used": used, "quota": quota})
}

// formatBytes 人类可读的字节大小
func formatBytes(b int64) string {
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GB", float64(b)/(1024*1024*1024))
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func GetFileStats(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	cached, found := fileStatsCache.Get(uid)

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

	fileStatsCache.Add(uid, fileStatsCacheEntry{
		data:      resultData,
		expiredAt: time.Now().Add(5 * time.Minute),
	})

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
		if *req.ParentID == 0 {
			// 0 视为移回根目录：显式置 NULL（map 更新不会自动转换，否则写入 0 会成树中不可见孤儿）
			updates["parent_id"] = gorm.Expr("NULL")
		} else {
			_, parentErr := svc.GetFolder(userID.(uint), *req.ParentID)
			if parentErr != nil {
				response.BadRequest(c, "父文件夹不存在或无权限")
				return
			}
			if svc.IsDescendant(userID.(uint), *req.ParentID, folder.ID) {
				response.BadRequest(c, "不能将文件夹移动到其子文件夹下")
				return
			}
			updates["parent_id"] = *req.ParentID
		}
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
	// 登录即可下载（auth 中间件已保证 user_id 存在）：
	// 消息里他人发送的聊天文件也须能下载，故不做所有者校验，仅要求文件存在。
	// 路由组 /api/v1/files 已挂 auth 中间件；软删除文件仍由 GORM 自动过滤。
	fileIDStr := c.Param("id")

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文件ID")
		return
	}

	svc := di.GlobalContainer.FileService
	file, err := svc.GetFileByID(uint(fileID))
	if err != nil {
		response.NotFound(c, "文件不存在")
		return
	}

	streamFileResponse(c, file, file.Name)
}

// PublicDownloadFile 公开下载文件（无需认证，用于客户端安装包等）
func PublicDownloadFile(c *gin.Context) {
	fileIDStr := c.Param("id")

	fileID, err := strconv.ParseUint(fileIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的文件ID")
		return
	}

	db := database.GetDB()
	var file model.File
	if err := db.First(&file, uint(fileID)).Error; err != nil {
		response.NotFound(c, "文件不存在")
		return
	}

	// 只允许公开类型的文件下载（如客户端更新包），私有上传文件禁止公开下载。
	// 这些来源在上传时已限制为仅管理员可设置（见 UploadFile）。
	if !protectedUploadSources[file.Source] {
		response.Forbidden(c, "该文件不允许公开下载")
		return
	}

	streamFileResponse(c, &file, file.OriginalName)
}

// streamFileFromStorage 从存储层读取文件并流式写入响应。
// 调用方应在调用前设置 Content-Type 和 Content-Disposition。
// timeout 为0时默认60秒。
func streamFileFromStorage(c *gin.Context, storagePath string, size int64, timeout time.Duration) {
	mgr := di.GlobalContainer.StorageManager
	if mgr == nil {
		response.InternalServerError(c, "存储服务未初始化")
		return
	}
	st, key, ok := mgr.ByPath(storagePath)
	if !ok || st == nil {
		response.InternalServerError(c, "存储类型不支持")
		return
	}

	if timeout == 0 {
		timeout = 60 * time.Second
	}
	// 用请求 context：客户端断开后存储层读取也会中止，避免 goroutine 泄漏
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	reader, err := st.Get(ctx, key)
	if err != nil {
		response.InternalServerError(c, "读取文件失败")
		return
	}
	defer reader.Close()

	c.Header("Content-Length", fmt.Sprintf("%d", size))
	c.Header("X-Content-Type-Options", "nosniff")

	if _, err := io.Copy(c.Writer, reader); err != nil {
		logger.WithModule("FileHandler").Error("下载文件失败", "error", err)
	}
}

// streamFileResponse 统一处理文件流响应。
// filename 用于 Content-Disposition（会被 sanitize）。
// 通过 query 参数控制行为：
//   - inline=true: 预览模式（inline disposition），危险类型仍强制下载
//   - thumbnail=true: 缩略图模式（仅在 inline 模式下不设置 Content-Disposition）
func streamFileResponse(c *gin.Context, file *model.File, filename string) {
	c.Header("Content-Type", file.MimeType)

	inline := c.Query("inline") == "true"
	thumbnail := c.Query("thumbnail") == "true"

	if inline && !upload.ShouldForceDownload(filename) {
		// 预览模式：inline（缩略图模式不设 disposition）
		if !thumbnail {
			c.Header("Content-Disposition", sanitizeInlineFilename(filename))
		}
	} else {
		// 下载模式：attachment（危险类型也强制 attachment）
		c.Header("Content-Disposition", sanitizeFilename(filename))
	}

	streamFileFromStorage(c, file.StoragePath, file.Size, 60*time.Second)
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
	_, err = svc.GetFile(userID.(uint), uint(fileID))
	if err != nil {
		response.NotFound(c, "文件不存在")
		return
	}

	if err := svc.DeleteFile(userID.(uint), uint(fileID)); err != nil {
		logger.WithModule("FileHandler").Error("删除文件失败", "file_id", fileID, "error", err)
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

	// parent_id 归一化：0 视为根目录（与 UpdateFolder 语义一致，避免写入 0 成树中不可见孤儿）
	if req.ParentID != nil && *req.ParentID == 0 {
		req.ParentID = nil
	}
	// 校验父文件夹存在且属于当前用户（GetFolder 走 user_id+scope 三元组过滤）
	if req.ParentID != nil {
		if _, err := svc.GetFolder(userID.(uint), *req.ParentID); err != nil {
			response.BadRequest(c, "父文件夹不存在或无权限")
			return
		}
	}

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
