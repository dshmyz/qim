package handler

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	syncpkg "github.com/dshmyz/qim/qim-server/sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrgSyncHandler struct {
	db      *gorm.DB
	engine  *syncpkg.Engine
	running map[uint]bool
	mu      sync.Mutex
}

func NewOrgSyncHandler() *OrgSyncHandler {
	engine := syncpkg.SharedEngine
	if engine == nil {
		engine = syncpkg.NewEngine()
	}
	return &OrgSyncHandler{
		db:      database.GetDB(),
		engine:  engine,
		running: make(map[uint]bool),
	}
}

func (h *OrgSyncHandler) GetConfigs(c *gin.Context) {
	var configs []model.OrgSyncConfig
	if err := h.db.Find(&configs).Error; err != nil {
		response.InternalServerError(c, "查询同步配置失败")
		return
	}
	response.Success(c, configs)
}

func (h *OrgSyncHandler) CreateConfig(c *gin.Context) {
	var req model.OrgSyncConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.db.Create(&req).Error; err != nil {
		response.InternalServerError(c, "创建同步配置失败")
		return
	}

	response.Success(c, req)
}

func (h *OrgSyncHandler) UpdateConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var config model.OrgSyncConfig
	if err := h.db.First(&config, id).Error; err != nil {
		response.NotFound(c, "同步配置不存在")
		return
	}

	var updateData model.OrgSyncConfig
	if err := c.ShouldBindJSON(&updateData); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.db.Model(&config).Updates(updateData).Error; err != nil {
		response.InternalServerError(c, "更新同步配置失败")
		return
	}

	response.Success(c, config)
}

func (h *OrgSyncHandler) DeleteConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var config model.OrgSyncConfig
	if err := h.db.First(&config, id).Error; err != nil {
		response.NotFound(c, "同步配置不存在")
		return
	}

	if err := h.db.Delete(&config).Error; err != nil {
		response.InternalServerError(c, "删除同步配置失败")
		return
	}

	response.Success(c, nil)
}

func (h *OrgSyncHandler) TriggerSync(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	configID := uint(id)

	h.mu.Lock()
	if h.running[configID] {
		h.mu.Unlock()
		response.BadRequest(c, "该配置的同步任务正在运行中，请稍后再试")
		return
	}
	h.running[configID] = true
	h.mu.Unlock()

	var config model.OrgSyncConfig
	if err := h.db.First(&config, configID).Error; err != nil {
		h.mu.Lock()
		delete(h.running, configID)
		h.mu.Unlock()
		response.NotFound(c, "同步配置不存在")
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.WithModule("OrgSync").Error("同步任务panic", "config_id", configID, "recover", r)
			}
			h.mu.Lock()
			delete(h.running, configID)
			h.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		h.engine.Sync(ctx, &config)
	}()

	response.SuccessWithMessage(c, "同步任务已启动", gin.H{
		"config_id": configID,
	})
}

func (h *OrgSyncHandler) GetLogs(c *gin.Context) {
	configID := c.Query("config_id")
	if configID == "" {
		response.BadRequest(c, "config_id 不能为空")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	db := database.GetDB()

	var total int64
	db.Model(&model.OrgSyncLog{}).Where("config_id = ?", configID).Count(&total)

	var logs []model.OrgSyncLog
	if err := db.Where("config_id = ?", configID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		response.InternalServerError(c, "查询同步日志失败")
		return
	}

	response.Success(c, gin.H{
		"total": total,
		"items": logs,
	})
}
