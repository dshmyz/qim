package sync

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var webhookPlaceholderHash string

func init() {
	hash, err := bcrypt.GenerateFromPassword([]byte("sync_disabled_local_login"), bcrypt.DefaultCost)
	if err != nil {
		webhookPlaceholderHash = "$2a$10$disabled"
	} else {
		webhookPlaceholderHash = string(hash)
	}
}

type WebhookEvent struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type WebhookPayload struct {
	UserCreated     *UserEvent     `json:"user_created,omitempty"`
	UserUpdated     *UserEvent     `json:"user_updated,omitempty"`
	UserDeleted     *UserEvent     `json:"user_deleted,omitempty"`
	DeptCreated     *DeptEvent     `json:"department_created,omitempty"`
	DeptUpdated     *DeptEvent     `json:"department_updated,omitempty"`
	DeptDeleted     *DeptEvent     `json:"department_deleted,omitempty"`
}

type UserEvent struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	DeptID   string `json:"department_id"`
}

type DeptEvent struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
}

func WebhookHandler(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	var events []WebhookEvent
	if err := json.Unmarshal(body, &events); err != nil {
		var single WebhookEvent
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体格式"})
			return
		}
		events = []WebhookEvent{single}
	}

	db := database.GetDB()
	var configs []model.OrgSyncConfig
	if err := db.Where("sync_type = ? AND enabled = ?", "realtime", true).Find(&configs).Error; err != nil || len(configs) == 0 {
		logger.WithModule("Webhook").Warn("未找到实时同步配置", "error", err)
		c.JSON(http.StatusOK, gin.H{"status": "skipped", "message": "无可用的实时同步配置"})
		return
	}

	for _, event := range events {
		if err := processEvent(event, db); err != nil {
			logger.WithModule("Webhook").Error("处理事件失败", "event", event.Event, "error", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func processEvent(event WebhookEvent, db *gorm.DB) error {
	switch event.Event {
	case "user_created", "user_updated":
		var userEvent UserEvent
		if err := json.Unmarshal(event.Data, &userEvent); err != nil {
			return fmt.Errorf("解析用户事件失败: %w", err)
		}
		return processUserEvent(event.Event, &userEvent, db)

	case "user_deleted":
		var userEvent UserEvent
		if err := json.Unmarshal(event.Data, &userEvent); err != nil {
			return fmt.Errorf("解析用户事件失败: %w", err)
		}

		var user model.User
		result := db.Where("username = ?", userEvent.Username).First(&user)
		if result.Error != nil {
			return nil
		}

		// 清理部门关联后软删除用户
		db.Where("user_id = ?", user.ID).Delete(&model.DepartmentEmployee{})
		return db.Delete(&user).Error

	case "department_created":
		var deptEvent DeptEvent
		if err := json.Unmarshal(event.Data, &deptEvent); err != nil {
			return fmt.Errorf("解析部门事件失败: %w", err)
		}
		return processDeptCreated(&deptEvent, db)

	case "department_updated":
		var deptEvent DeptEvent
		if err := json.Unmarshal(event.Data, &deptEvent); err != nil {
			return fmt.Errorf("解析部门事件失败: %w", err)
		}
		return processDeptUpdated(&deptEvent, db)

	case "department_deleted":
		var deptEvent DeptEvent
		if err := json.Unmarshal(event.Data, &deptEvent); err != nil {
			return fmt.Errorf("解析部门事件失败: %w", err)
		}

		return processDeptDeleted(&deptEvent, db)

	default:
		logger.WithModule("Webhook").Warn("未知事件类型", "event", event.Event)
		return nil
	}
}

func processUserEvent(eventType string, ue *UserEvent, db *gorm.DB) error {
	var user model.User
	result := db.Where("username = ?", ue.Username).First(&user)

	if result.Error != nil && eventType == "user_created" {
		user = model.User{
			Username:     ue.Username,
			Nickname:     ue.Nickname,
			Email:        ue.Email,
			Phone:        ue.Phone,
			Status:       "offline",
			Type:         "user",
			PasswordHash: webhookPlaceholderHash,
		}
		if user.Nickname == "" {
			user.Nickname = user.Username
		}
		return db.Create(&user).Error
	}

	if result.Error == nil {
		updates := make(map[string]interface{})
		if ue.Nickname != "" {
			updates["nickname"] = ue.Nickname
		}
		if ue.Email != "" {
			updates["email"] = ue.Email
		}
		if ue.Phone != "" {
			updates["phone"] = ue.Phone
		}
		if len(updates) > 0 {
			return db.Model(&user).Updates(updates).Error
		}
	}

	return nil
}

func processDeptCreated(de *DeptEvent, db *gorm.DB) error {
	// 先按 ExternalID 查找
	if de.ID != "" {
		var existing model.Department
		if err := db.Where("external_id = ?", de.ID).First(&existing).Error; err == nil {
			return nil // 已存在，跳过
		}
	}

	// 新建部门
	dept := model.Department{
		Name:       de.Name,
		ExternalID: de.ID,
	}
	return db.Create(&dept).Error
}

func processDeptUpdated(de *DeptEvent, db *gorm.DB) error {
	var dept model.Department

	// 优先按 ExternalID 查找
	if de.ID != "" {
		if err := db.Where("external_id = ?", de.ID).First(&dept).Error; err != nil {
			// ExternalID 未找到，按名称查找并设置 ExternalID
			if err := db.Where("name = ?", de.Name).First(&dept).Error; err != nil {
				// 完全找不到，创建
				newDept := model.Department{
					Name:       de.Name,
					ExternalID: de.ID,
				}
				return db.Create(&newDept).Error
			}
			// 找到后回填 ExternalID
			db.Model(&dept).Update("external_id", de.ID)
		}
	} else {
		if err := db.Where("name = ?", de.Name).First(&dept).Error; err != nil {
			return nil
		}
	}

	updates := make(map[string]interface{})
	if de.Name != "" && de.Name != dept.Name {
		updates["name"] = de.Name
	}
	if de.ParentID != "" {
		var parentDept model.Department
		if err := db.Where("external_id = ?", de.ParentID).First(&parentDept).Error; err == nil {
			if dept.ParentID == nil || *dept.ParentID != parentDept.ID {
				updates["parent_id"] = parentDept.ID
			}
		}
	}
	if len(updates) > 0 {
		return db.Model(&dept).Updates(updates).Error
	}

	return nil
}

func processDeptDeleted(de *DeptEvent, db *gorm.DB) error {
	var dept model.Department

	if de.ID != "" {
		if err := db.Where("external_id = ?", de.ID).First(&dept).Error; err != nil {
			return nil
		}
	} else {
		if err := db.Where("name = ?", de.Name).First(&dept).Error; err != nil {
			return nil
		}
	}

	// 清理员工关联后软删除部门
	db.Where("department_id = ?", dept.ID).Delete(&model.DepartmentEmployee{})
	return db.Delete(&dept).Error
}
