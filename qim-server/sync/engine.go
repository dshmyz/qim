package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/orgsync"
	"qim-server/pkg/logger"
	"qim-server/sync/syncer"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// placeholderHash 是同步创建用户时使用的密码哈希，这些用户无法通过密码登录
var placeholderHash string

func init() {
	hash, err := bcrypt.GenerateFromPassword([]byte("sync_disabled_local_login"), bcrypt.DefaultCost)
	if err != nil {
		placeholderHash = "$2a$10$disabled"
	} else {
		placeholderHash = string(hash)
	}
}

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
	UsersCreated  int    `json:"users_created"`
	UsersUpdated  int    `json:"users_updated"`
	UsersDeleted  int    `json:"users_deleted"`
	DeptsCreated  int    `json:"depts_created"`
	DeptsUpdated  int    `json:"depts_updated"`
	DeptsDeleted  int    `json:"depts_deleted"`
	GroupsCreated int    `json:"groups_created"`
	GroupsUpdated int    `json:"groups_updated"`
	GroupsDeleted int    `json:"groups_deleted"`
	ErrorMessage  string `json:"error_message,omitempty"`
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

	return e.syncToLocal(orgData, config)
}

func (e *Engine) syncToLocal(data *orgsync.OrgData, config *model.OrgSyncConfig) *syncStats {
	stats := &syncStats{}

	// 先同步部门，建立 extID → localID 映射
	extToLocalDept := e.syncDepartments(data, config, stats)
	e.syncUsers(data, extToLocalDept, config, stats)
	e.syncDepartmentEmployees(data, extToLocalDept, config, stats)
	e.syncGroups(data, stats)

	return stats
}

// syncDepartments 同步部门，返回外部ID到本地ID的映射
func (e *Engine) syncDepartments(data *orgsync.OrgData, config *model.OrgSyncConfig, stats *syncStats) map[string]uint {
	extToLocal := make(map[string]uint)
	syncConfig := e.parseSyncConfig(config)

	// 如果配置了不同步部门，直接返回已存在部门的映射
	if !syncConfig["sync_departments"].(bool) {
		logger.WithModule("OrgSync").Info("已配置跳过部门同步")
		for _, extDept := range data.Departments {
			var local model.Department
			if e.db.Where("external_id = ?", extDept.ID).First(&local).Error == nil {
				extToLocal[extDept.ID] = local.ID
			}
		}
		return extToLocal
	}

	sort.Slice(data.Departments, func(i, j int) bool {
		// 按层级升序排列，保证父部门优先处理
		if data.Departments[i].Level != data.Departments[j].Level {
			return data.Departments[i].Level < data.Departments[j].Level
		}
		return data.Departments[i].ID < data.Departments[j].ID
	})

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
				if err := e.db.Model(&local).Updates(updates).Error; err != nil {
					logger.WithModule("OrgSync").Warn("更新部门失败", "id", local.ID, "error", err)
				} else {
					stats.DeptsUpdated++
				}
			}
		} else {
			// 新建
			var parentID *uint
			if extDept.ParentID != "" {
				if pid, ok := extToLocal[extDept.ParentID]; ok {
					parentID = &pid
				}
			}

			newDept := model.Department{
				Name:       extDept.Name,
				ExternalID: extDept.ID,
				Level:      extDept.Level,
				ParentID:   parentID,
			}
			if err := e.db.Create(&newDept).Error; err != nil {
				logger.WithModule("OrgSync").Warn("创建部门失败", "name", extDept.Name, "error", err)
				continue
			}
			stats.DeptsCreated++
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

func (e *Engine) syncUsers(data *orgsync.OrgData, extToLocalDept map[string]uint, config *model.OrgSyncConfig, stats *syncStats) {
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
				PasswordHash: placeholderHash,
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
				if err := e.db.Model(&user).Updates(updates).Error; err != nil {
					logger.WithModule("OrgSync").Warn("更新用户失败", "username", user.Username, "error", err)
				} else {
					stats.UsersUpdated++
				}
			}
		}
	}
}

func (e *Engine) syncDepartmentEmployees(data *orgsync.OrgData, extToLocalDept map[string]uint, config *model.OrgSyncConfig, stats *syncStats) {
	// 按用户分组部门关联，收集每个用户的目标部门ID集合
	userDeptMap := make(map[uint][]uint)

	for _, rel := range data.UserDeptRelations {
		user := e.resolveUserByRelation(rel, data, config.SyncType)
		if user == nil {
			continue
		}

		deptID := e.resolveDepartmentByRelation(rel, extToLocalDept)
		if deptID == 0 {
			continue
		}

		userDeptMap[user.ID] = append(userDeptMap[user.ID], deptID)
	}

	// 逐个用户处理：只创建新关联，删除不再存在于外部系统中的旧关联
	for userID, targetDeptIDs := range userDeptMap {
		for _, deptID := range targetDeptIDs {
			var existingRel int64
			e.db.Model(&model.DepartmentEmployee{}).
				Where("user_id = ? AND department_id = ?", userID, deptID).
				Count(&existingRel)

			if existingRel == 0 {
				empRel := model.DepartmentEmployee{
					UserID:       userID,
					DepartmentID: deptID,
					IsPrimary:    true,
				}
				if err := e.db.Create(&empRel).Error; err != nil {
					logger.WithModule("OrgSync").Warn("创建部门员工关系失败", "user_id", userID, "dept_id", deptID, "error", err)
				}
			}
		}

		// 仅删除该用户在外部系统中不存在的部门关联
		// 即保留手动在 QIM 中配置的其他部门关系
		if len(targetDeptIDs) > 0 {
			if err := e.db.Where("user_id = ? AND department_id NOT IN ?", userID, targetDeptIDs).
				Delete(&model.DepartmentEmployee{}).Error; err != nil {
				logger.WithModule("OrgSync").Warn("删除部门员工关系失败", "user_id", userID, "error", err)
			}
		}
	}
}

func (e *Engine) resolveUserByRelation(rel orgsync.UserDeptRelation, data *orgsync.OrgData, syncSource string) *model.User {
	var user model.User

	// 1. 直接通过 username 查找
	if err := e.db.Where("username = ?", rel.UserID).First(&user).Error; err == nil {
		return &user
	}

	// 2. 从 data.Users 解析 username 后查找
	for _, extUser := range data.Users {
		if extUser.ID == rel.UserID && extUser.Username != "" {
			if err := e.db.Where("username = ?", extUser.Username).First(&user).Error; err == nil {
				return &user
			}
			break
		}
	}

	return nil
}

func (e *Engine) resolveDepartmentByRelation(rel orgsync.UserDeptRelation, extToLocalDept map[string]uint) uint {
	if deptID, ok := extToLocalDept[rel.DepartmentID]; ok {
		return deptID
	}

	var dept model.Department
	if err := e.db.Where("name = ?", rel.DepartmentID).First(&dept).Error; err == nil {
		return dept.ID
	}

	return 0
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

// parseSyncConfig 解析同步配置选项
// 支持的配置项：
// - sync_departments: 是否同步部门（默认 true）
// - sync_users: 是否同步用户（默认 true）
// - sync_relations: 是否同步用户-部门关联（默认 true）
// - create_missing_users: 是否创建缺失的用户（默认 true）
func (e *Engine) parseSyncConfig(config *model.OrgSyncConfig) map[string]interface{} {
	defaultConfig := map[string]interface{}{
		"sync_departments":     true,
		"sync_users":           true,
		"sync_relations":       true,
		"create_missing_users": true,
	}

	if config.Config == "" {
		return defaultConfig
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(config.Config), &cfg); err != nil {
		return defaultConfig
	}

	// 合并配置
	for k, _ := range defaultConfig {
		if val, ok := cfg[k]; ok {
			defaultConfig[k] = val
		}
	}

	return defaultConfig
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
