package handler

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"qim-server/di"
	"qim-server/model"
	"qim-server/pkg/logger"
	"qim-server/pkg/response"
	"qim-server/service/storage"

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

func (h *FeedbackHandler) GetFeedbacks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	status := c.Query("status")
	feedbackType := c.Query("type")

	var feedbacks []model.UserFeedback
	var total int64

	query := h.db.Model(&model.UserFeedback{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if feedbackType != "" {
		query = query.Where("type = ?", feedbackType)
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
	file, err := c.FormFile("screenshot")
	if err == nil {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		allowedExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true}
		if !allowedExts[ext] {
			response.BadRequest(c, "不支持的图片格式，仅支持 PNG、JPG、GIF、WebP")
			return
		}

		if file.Size > 5*1024*1024 {
			response.BadRequest(c, "截图大小不能超过 5MB")
			return
		}

		st := di.GlobalContainer.DefaultStorage
		if st == nil {
			response.InternalServerError(c, "存储服务未初始化")
			return
		}

		var uidVal uint
		if userID != nil {
			uidVal = *userID
		}
		filename := fmt.Sprintf("feedback_%s_%d%s", time.Now().Format("20060102150405"), uidVal, ext)
		key := "uploads/feedbacks/" + filename

		fileData, err := file.Open()
		if err != nil {
			logger.WithModule("feedback").Error("打开截图失败", "error", err)
			response.InternalServerError(c, "保存截图失败")
			return
		}
		defer fileData.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		mimeType := file.Header.Get("Content-Type")
		if err := st.Put(ctx, key, fileData, file.Size, mimeType); err != nil {
			logger.WithModule("feedback").Error("保存截图失败", "error", err)
			response.InternalServerError(c, "保存截图失败")
			return
		}

		screenshotPath = storage.BuildPath(st.Kind(), key)
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
