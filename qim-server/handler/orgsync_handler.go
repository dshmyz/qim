package handler

import (
	"net/http"
	"strconv"

	"qim-server/database"
	"qim-server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrgSyncHandler struct {
	db *gorm.DB
}

func NewOrgSyncHandler() *OrgSyncHandler {
	return &OrgSyncHandler{
		db: database.GetDB(),
	}
}

func (h *OrgSyncHandler) GetConfigs(c *gin.Context) {
	var configs []model.OrgSyncConfig
	if err := h.db.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func (h *OrgSyncHandler) CreateConfig(c *gin.Context) {
	var config model.OrgSyncConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": config})
}

func (h *OrgSyncHandler) UpdateConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var config model.OrgSyncConfig
	if err := h.db.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	var updateData model.OrgSyncConfig
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Model(&config).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": config})
}

func (h *OrgSyncHandler) TriggerSync(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var config model.OrgSyncConfig
	if err := h.db.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "config not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "同步任务已触发", "config_id": id})
}

func (h *OrgSyncHandler) GetLogs(c *gin.Context) {
	configID := c.Query("config_id")
	if configID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config_id is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var logs []model.OrgSyncLog
	var total int64

	h.db.Model(&model.OrgSyncLog{}).Where("config_id = ?", configID).Count(&total)

	if err := h.db.Where("config_id = ?", configID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"total": total,
			"items": logs,
		},
	})
}
