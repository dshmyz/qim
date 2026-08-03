package handler

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/pkg/upload"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CrashLogHandler struct {
	db *gorm.DB
}

func NewCrashLogHandler(db *gorm.DB) *CrashLogHandler {
	return &CrashLogHandler{db: db}
}

func (h *CrashLogHandler) GetCrashLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	platform := c.Query("platform")
	appVersion := c.Query("appVersion")

	var crashLogs []model.CrashLog
	var total int64

	query := h.db.Model(&model.CrashLog{})
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if appVersion != "" {
		query = query.Where("app_version = ?", appVersion)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&crashLogs).Error; err != nil {
		logger.WithModule("crash").Error("获取崩溃日志失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取崩溃日志失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     crashLogs,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

func (h *CrashLogHandler) GetCrashDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的 ID",
		})
		return
	}

	var crashLog model.CrashLog
	if err := h.db.First(&crashLog, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "崩溃日志不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": crashLog,
	})
}

func (h *CrashLogHandler) CreateCrashLog(c *gin.Context) {
	var req struct {
		Platform     string `json:"platform" binding:"required"`
		AppVersion   string `json:"appVersion" binding:"required"`
		DeviceModel  string `json:"deviceModel"`
		OSVersion    string `json:"osVersion"`
		ErrorStack   string `json:"errorStack"`
		ErrorMessage string `json:"errorMessage"`
		Extra        string `json:"extra"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	userIDAny, exists := c.Get("user_id")
	var userID *uint
	if exists {
		uid := userIDAny.(uint)
		userID = &uid
	}

	crashLog := model.CrashLog{
		UserID:       userID,
		Platform:     req.Platform,
		AppVersion:   req.AppVersion,
		DeviceModel:  req.DeviceModel,
		OSVersion:    req.OSVersion,
		ErrorStack:   req.ErrorStack,
		ErrorMessage: req.ErrorMessage,
		Extra:        req.Extra,
		CreatedAt:    time.Now(),
	}

	if err := h.db.Create(&crashLog).Error; err != nil {
		logger.WithModule("crash").Error("创建崩溃日志失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建崩溃日志失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "提交成功",
		"data":    crashLog,
	})
}

type FeedbackHandler struct {
	db *gorm.DB
}

func NewFeedbackHandler(db *gorm.DB) *FeedbackHandler {
	return &FeedbackHandler{db: db}
}

// fillUserNames 批量填充反馈列表的用户名/昵称/处理人名，避免 N+1 查询
func (h *FeedbackHandler) fillUserNames(feedbacks []model.UserFeedback) {
	if len(feedbacks) == 0 {
		return
	}
	// 收集所有需要查询的 userId（含提交者和处理人）
	idSet := make(map[uint]struct{})
	for i := range feedbacks {
		if feedbacks[i].UserID != nil {
			idSet[*feedbacks[i].UserID] = struct{}{}
		}
		if feedbacks[i].HandlerID != nil {
			idSet[*feedbacks[i].HandlerID] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return
	}
	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	var users []model.User
	if err := h.db.Select("id, username, nickname").Where("id IN ?", ids).Find(&users).Error; err != nil {
		logger.WithModule("feedback").Warn("批量查询反馈用户名失败", "error", err)
		return
	}
	nameMap := make(map[uint]model.User, len(users))
	for _, u := range users {
		nameMap[u.ID] = u
	}
	for i := range feedbacks {
		if feedbacks[i].UserID != nil {
			if u, ok := nameMap[*feedbacks[i].UserID]; ok {
				feedbacks[i].Username = u.Username
				feedbacks[i].Nickname = u.Nickname
			}
		}
		if feedbacks[i].HandlerID != nil {
			if u, ok := nameMap[*feedbacks[i].HandlerID]; ok {
				feedbacks[i].HandlerName = u.Nickname
				if feedbacks[i].HandlerName == "" {
					feedbacks[i].HandlerName = u.Username
				}
			}
		}
	}
}

// fillUserNamesSingle 填充单条反馈的用户名，详情/更新场景使用
func (h *FeedbackHandler) fillUserNamesSingle(feedback *model.UserFeedback) {
	list := []model.UserFeedback{*feedback}
	h.fillUserNames(list)
	*feedback = list[0]
}

func (h *FeedbackHandler) GetFeedbacks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status := c.Query("status")
	feedbackType := c.Query("type")
	userID := c.Query("userId")
	priority := c.Query("priority")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	var feedbacks []model.UserFeedback
	var total int64

	query := h.db.Model(&model.UserFeedback{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if feedbackType != "" {
		query = query.Where("type = ?", feedbackType)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&feedbacks).Error; err != nil {
		logger.WithModule("feedback").Error("获取用户反馈失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取用户反馈失败",
		})
		return
	}

	h.fillUserNames(feedbacks)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     feedbacks,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

func (h *FeedbackHandler) GetFeedbackDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的 ID",
		})
		return
	}

	var feedback model.UserFeedback
	if err := h.db.First(&feedback, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "反馈不存在",
		})
		return
	}

	h.fillUserNamesSingle(&feedback)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": feedback,
	})
}

func (h *FeedbackHandler) UpdateFeedback(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的 ID",
		})
		return
	}

	var req struct {
		Status   *string `json:"status"`
		Priority *string `json:"priority"`
		Reply    *string `json:"reply"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	var feedback model.UserFeedback
	if err := h.db.First(&feedback, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "反馈不存在",
		})
		return
	}

	updates := map[string]interface{}{}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Reply != nil {
		updates["reply"] = *req.Reply
	}
	updates["updated_at"] = time.Now()

	userIDAny, exists := c.Get("user_id")
	if exists {
		uid := userIDAny.(uint)
		updates["handler_id"] = uid
	}

	if err := h.db.Model(&feedback).Updates(updates).Error; err != nil {
		logger.WithModule("feedback").Error("更新用户反馈失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新用户反馈失败",
		})
		return
	}

	h.db.First(&feedback, id)
	h.fillUserNamesSingle(&feedback)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
		"data":    feedback,
	})
}

func (h *FeedbackHandler) CreateFeedback(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	var userID *uint
	if exists {
		uid := userIDAny.(uint)
		userID = &uid
	}

	reqType := c.PostForm("type")
	reqContent := c.PostForm("content")

	if reqType == "" || reqContent == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: type 和 content 不能为空",
		})
		return
	}

	var screenshotPath string
	// MaxBytesReader 保护：防止超大 multipart 请求在解析阶段就耗尽内存
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, upload.DefaultScreenshotMaxSize+64*1024)
	file, err := c.FormFile("screenshot")
	if err == nil {
		// 统一上传策略：截图仅允许图片类型，强制白名单校验
		screenshotPolicy := upload.NewPolicy(
			upload.DefaultScreenshotMaxSize,
			map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true},
			true,
		)

		st := di.GlobalContainer.DefaultStorage
		if st == nil {
			response.InternalServerError(c, "存储服务未初始化")
			return
		}

		var uidVal uint
		if userID != nil {
			uidVal = *userID
		}

		now := time.Now()
		safeExt := strings.ToLower(filepath.Ext(upload.SanitizeFilename(file.Filename)))

		// 复用公共"读取+校验+存储"函数
		saved, saveErr := upload.SaveMultipartFile(file, upload.SaveConfig{
			Policy:    screenshotPolicy,
			Storage:   st,
			KeyPrefix: fmt.Sprintf("uploads/feedbacks/%s", now.Format("2006/01")),
			FilenameFn: func() string {
				return fmt.Sprintf("feedback_%s_%d%s", now.Format("20060102150405"), uidVal, safeExt)
			},
			ContextFn: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(c.Request.Context(), 30*time.Second)
			},
		})
		if saveErr != nil {
			logger.WithModule("feedback").Error("保存截图失败", "error", saveErr)
			if ve, ok := saveErr.(*upload.ValidateError); ok {
				if ve.Field == "size" {
					response.BadRequest(c, "截图大小不能超过 5MB")
					return
				}
				response.BadRequest(c, "不支持的图片格式，仅支持 PNG、JPG、GIF、WebP")
				return
			}
			response.InternalServerError(c, "保存截图失败")
			return
		}

		screenshotPath = saved.StoragePath
	}

	feedback := model.UserFeedback{
		UserID:     userID,
		Type:       reqType,
		Content:    reqContent,
		Screenshot: screenshotPath,
		Status:     "pending",
		Priority:   "normal",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := h.db.Create(&feedback).Error; err != nil {
		logger.WithModule("feedback").Error("创建用户反馈失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建用户反馈失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "提交成功",
		"data":    feedback,
	})
}

// GetMyFeedbacks 查询当前用户提交的反馈列表
func (h *FeedbackHandler) GetMyFeedbacks(c *gin.Context) {
	userIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "未登录",
		})
		return
	}
	userID := userIDAny.(uint)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var feedbacks []model.UserFeedback
	var total int64

	query := h.db.Model(&model.UserFeedback{}).Where("user_id = ?", userID)
	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&feedbacks).Error; err != nil {
		logger.WithModule("feedback").Error("获取我的反馈失败", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取反馈列表失败",
		})
		return
	}

	h.fillUserNames(feedbacks)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":     feedbacks,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}
