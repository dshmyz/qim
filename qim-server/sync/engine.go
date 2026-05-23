package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/orgsync"
	"qim-server/pkg/logger"
	"qim-server/sync/syncer"

	"gorm.io/gorm"
)

var SharedEngine *Engine

type Engine struct {
	db *gorm.DB
}

func NewEngine() *Engine {
	return &Engine{
		db: database.GetDB(),
	}
}

func (e *Engine) Sync(ctx context.Context, config *model.OrgSyncConfig) {
	log := model.OrgSyncLog{
		ConfigID:  config.ID,
		Status:    "running",
		StartedAt: time.Now(),
	}
	if err := e.db.Create(&log).Error; err != nil {
		logger.WithModule("OrgSync").Error("创建同步日志失败", "error", err, "config_id", config.ID)
		return
	}

	result := e.executeSync(ctx, config)

	statsJSON, _ := json.Marshal(result)
	log.Status = "success"
	log.FinishedAt = nowPtr()
	if result.ErrorMessage != "" {
		log.Status = "failed"
		log.ErrorMessage = result.ErrorMessage
	}
	log.Stats = string(statsJSON)
	if err := e.db.Model(&log).Updates(map[string]interface{}{
		"status":        log.Status,
		"finished_at":   log.FinishedAt,
		"stats":         log.Stats,
		"error_message": log.ErrorMessage,
	}).Error; err != nil {
		logger.WithModule("OrgSync").Error("更新同步日志失败", "error", err)
	}

	now := time.Now()
	updates := map[string]interface{}{
		"last_sync_at":     now,
		"last_sync_status": log.Status,
	}
	if err := e.db.Model(config).Updates(updates).Error; err != nil {
		logger.WithModule("OrgSync").Error("更新同步配置状态失败", "error", err)
	}

	logger.WithModule("OrgSync").Info("同步完成",
		"config_id", config.ID,
		"config_name", config.Name,
		"status", log.Status,
		"users_created", result.UsersCreated,
		"users_updated", result.UsersUpdated,
		"users_deleted", result.UsersDeleted,
		"depts_created", result.DeptsCreated,
		"depts_updated", result.DeptsUpdated,
		"depts_deleted", result.DeptsDeleted,
	)
}

type syncStats struct {
	UsersCreated   int    `json:"users_created"`
	UsersUpdated   int    `json:"users_updated"`
	UsersDeleted   int    `json:"users_deleted"`
	DeptsCreated   int    `json:"depts_created"`
	DeptsUpdated   int    `json:"depts_updated"`
	DeptsDeleted   int    `json:"depts_deleted"`
	GroupsCreated  int    `json:"groups_created"`
	GroupsUpdated  int    `json:"groups_updated"`
	GroupsDeleted  int    `json:"groups_deleted"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

func (e *Engine) executeSync(ctx context.Context, config *model.OrgSyncConfig) *syncStats {
	provider, err := syncer.NewProvider(config)
	if err != nil {
		return &syncStats{ErrorMessage: fmt.Sprintf("创建同步器失败: %v", err)}
	}

	orgData, err := provider.Fetch(ctx, config.Config)
	if err != nil {
		return &syncStats{ErrorMessage: fmt.Sprintf("获取外部数据失败: %v", err)}
	}

	return e.syncToLocal(orgData)
}

func (e *Engine) syncToLocal(data *orgsync.OrgData) *syncStats {
	stats := &syncStats{}

	// 先同步部门，建立 extID → localID 映射
	extToLocalDept := e.syncDepartments(data, stats)
	e.syncUsers(data, extToLocalDept, stats)
	e.syncDepartmentEmployees(data, extToLocalDept, stats)
	e.syncGroups(data, stats)

	return stats
}

// syncDepartments 同步部门，返回外部ID到本地ID的映射
func (e *Engine) syncDepartments(data *orgsync.OrgData, stats *syncStats) map[string]uint {
	extToLocal := make(map[string]uint)

	// 按 ExternalID 查找已有部门
	for _, extDept := range data.Departments {
		var local model.Department
		err := e.db.Where("external_id = ?", extDept.ID).First(&local).Error

		if err == nil {
			// 已存在，更新
			extToLocal[extDept.ID] = local.ID
			updates := make(map[string]interface{})
			if extDept.Name != "" && extDept.Name != local.Name {
				updates["name"] = extDept.Name
			}
			if extDept.Level > 0 && extDept.Level != local.Level {
				updates["level"] = extDept.Level
			}
			if extDept.ParentID != "" {
				if parentLocalID, ok := extToLocal[extDept.ParentID]; ok && (local.ParentID == nil || *local.ParentID != parentLocalID) {
					updates["parent_id"] = parentLocalID
				}
			}
			if len(updates) > 0 {
				e.db.Model(&local).Updates(updates)
				stats.DeptsUpdated++
			}
		} else {
			// 新建
			newDept := model.Department{
				Name:       extDept.Name,
				ExternalID: extDept.ID,
				Level:      extDept.Level,
			}
			stats.DeptsCreated++
			e.db.Create(&newDept)
			extToLocal[extDept.ID] = newDept.ID
		}
	}

	// 第二遍：设置 parent_id（需要所有部门的映射就绪后）
	for _, extDept := range data.Departments {
		if extDept.ParentID == "" {
			continue
		}
		localID, ok := extToLocal[extDept.ID]
		if !ok {
			continue
		}
		parentLocalID, ok := extToLocal[extDept.ParentID]
		if !ok {
			continue
		}
		var local model.Department
		if e.db.First(&local, localID).Error == nil {
			if local.ParentID == nil || *local.ParentID != parentLocalID {
				e.db.Model(&local).Update("parent_id", parentLocalID)
			}
		}
	}

	return extToLocal
}

func (e *Engine) syncUsers(data *orgsync.OrgData, extToLocalDept map[string]uint, stats *syncStats) {
	for _, extUser := range data.Users {
		if extUser.Username == "" {
			continue
		}

		var user model.User
		result := e.db.Where("username = ?", extUser.Username).First(&user)

		if result.Error != nil {
			newUser := model.User{
				Username:     extUser.Username,
				Nickname:     extUser.Nickname,
				Email:        extUser.Email,
				Phone:        extUser.Phone,
				Avatar:       extUser.Avatar,
				Status:       "offline",
				Type:         "user",
				PasswordHash: "$2a$10$placeholder",
			}
			if newUser.Nickname == "" {
				newUser.Nickname = newUser.Username
			}
			if err := e.db.Create(&newUser).Error; err != nil {
				logger.WithModule("OrgSync").Warn("创建用户失败", "username", extUser.Username, "error", err)
				continue
			}
			stats.UsersCreated++
		} else {
			updates := make(map[string]interface{})
			if extUser.Nickname != "" && extUser.Nickname != user.Nickname {
				updates["nickname"] = extUser.Nickname
			}
			if extUser.Email != "" && extUser.Email != user.Email {
				updates["email"] = extUser.Email
			}
			if extUser.Phone != "" && extUser.Phone != user.Phone {
				updates["phone"] = extUser.Phone
			}
			if extUser.Avatar != "" && extUser.Avatar != user.Avatar {
				updates["avatar"] = extUser.Avatar
			}
			if extUser.Position != "" {
				updates["signature"] = extUser.Position
			}
			if len(updates) > 0 {
				e.db.Model(&user).Updates(updates)
				stats.UsersUpdated++
			}
		}
	}
}

func (e *Engine) syncDepartmentEmployees(data *orgsync.OrgData, extToLocalDept map[string]uint, stats *syncStats) {
	for _, rel := range data.UserDeptRelations {
		// 查找本地用户
		var user model.User
		if err := e.db.Where("username = ?", rel.UserID).First(&user).Error; err != nil {
			// 尝试通过外部数据中的 username 查找
			var found bool
			for _, extUser := range data.Users {
				if extUser.ID == rel.UserID {
					e.db.Where("username = ?", extUser.Username).First(&user)
					if user.ID != 0 {
						found = true
					}
					break
				}
			}
			if !found {
				continue
			}
		}

		// 查找本地部门
		deptID, ok := extToLocalDept[rel.DepartmentID]
		if !ok {
			// 尝试按名称查找
			var dept model.Department
			if err := e.db.Where("name = ?", rel.DepartmentID).First(&dept).Error; err == nil {
				deptID = dept.ID
			} else {
				continue
			}
		}

		// 删除该用户的旧部门关联（处理员工调岗）
		e.db.Where("user_id = ? AND department_id != ?", user.ID, deptID).Delete(&model.DepartmentEmployee{})

		// 创建或跳过已存在的关联
		var existingRel int64
		e.db.Model(&model.DepartmentEmployee{}).
			Where("user_id = ? AND department_id = ?", user.ID, deptID).
			Count(&existingRel)

		if existingRel == 0 {
			empRel := model.DepartmentEmployee{
				UserID:       user.ID,
				DepartmentID: deptID,
				Position:     rel.UserID,
				IsPrimary:    true,
			}
			e.db.Create(&empRel)
		}
	}
}

func (e *Engine) syncGroups(data *orgsync.OrgData, stats *syncStats) {
	for _, extGroup := range data.Groups {
		var existing model.Group
		result := e.db.Where("name = ?", extGroup.Name).First(&existing)

		if result.Error != nil {
			conv := model.Conversation{
				Type: "group",
			}
			if err := e.db.Create(&conv).Error; err != nil {
				logger.WithModule("OrgSync").Warn("创建群聊会话失败", "name", extGroup.Name, "error", err)
				continue
			}

			group := model.Group{
				ConversationID: conv.ID,
				GroupType:      "group",
				Name:           extGroup.Name,
			}
			if err := e.db.Create(&group).Error; err != nil {
				logger.WithModule("OrgSync").Warn("创建群组失败", "name", extGroup.Name, "error", err)
				continue
			}
			stats.GroupsCreated++
		}
	}
}

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}

type ParseCronResult struct {
	Interval time.Duration
	NextRun  time.Duration
}

func ParseCron(schedule string) (*ParseCronResult, error) {
	if schedule == "" {
		return nil, fmt.Errorf("cron表达式为空")
	}

	var interval time.Duration
	switch schedule {
	case "@hourly":
		interval = time.Hour
	case "@daily", "@midnight":
		interval = 24 * time.Hour
	case "@weekly":
		interval = 7 * 24 * time.Hour
	case "@monthly":
		interval = 30 * 24 * time.Hour
	default:
		d, err := parseMinuteInterval(schedule)
		if err != nil {
			return nil, err
		}
		interval = d
	}

	now := time.Now()
	next := now.Add(interval)
	nextRounded := next.Truncate(time.Minute)
	if nextRounded.Before(now) {
		nextRounded = nextRounded.Add(time.Minute)
	}

	return &ParseCronResult{
		Interval: interval,
		NextRun:  nextRounded.Sub(now),
	}, nil
}

func parseMinuteInterval(schedule string) (time.Duration, error) {
	var minutes int
	if _, err := fmt.Sscanf(schedule, "every %d minutes", &minutes); err == nil && minutes > 0 {
		return time.Duration(minutes) * time.Minute, nil
	}
	if _, err := fmt.Sscanf(schedule, "every %d hours", &minutes); err == nil && minutes > 0 {
		return time.Duration(minutes) * time.Hour, nil
	}

	if len(schedule) >= 5 && strings.Count(schedule, " ") == 4 {
		return 24 * time.Hour, nil
	}

	return 0, fmt.Errorf("不支持的调度格式: %s", schedule)
}
