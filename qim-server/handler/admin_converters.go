package handler

import (
	"github.com/dshmyz/qim/qim-server/model"

	"github.com/gin-gonic/gin"
)

// 时间格式化常量，与 versionToFrontend 保持一致
const frontendTimeFormat = "2006-01-02 15:04:05"

// operationLogToFrontend 将 OperationLog 模型转换为前端期望的 snake_case 格式。
func operationLogToFrontend(log model.OperationLog) gin.H {
	return gin.H{
		"id":           log.ID,
		"user_id":      log.UserID,
		"username":     log.Username,
		"action":       log.Action,
		"module":       log.Module,
		"ip":           log.IP,
		"user_agent":   log.UserAgent,
		"request_url":  log.RequestURL,
		"request_body": log.RequestBody,
		"response":     log.Response,
		"duration":     log.Duration,
		"created_at":   log.CreatedAt.Format(frontendTimeFormat),
	}
}

// operationLogsToFrontend 将 OperationLog 列表转换为前端期望的 camelCase 格式。
func operationLogsToFrontend(logs []model.OperationLog) []gin.H {
	result := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		result = append(result, operationLogToFrontend(log))
	}
	return result
}

// blacklistToFrontend 将 Blacklist 模型转换为前端期望的 camelCase 格式。
// 注意：service.GetBlacklist 中使用了 Preload("User")，entry.User 含有用户信息。
func blacklistToFrontend(entry model.Blacklist) gin.H {
	username := ""
	if entry.User.Username != "" {
		username = entry.User.Username
	} else if entry.User.Nickname != "" {
		username = entry.User.Nickname
	}
	return gin.H{
		"id":        entry.ID,
		"userId":    entry.UserID,
		"username":  username,
		"reason":    entry.Reason,
		"operator":  entry.Operator,
		"status":    "active",
		"createdAt": entry.CreatedAt.Format(frontendTimeFormat),
	}
}

// blacklistsToFrontend 将 Blacklist 列表转换为前端期望的 camelCase 格式。
func blacklistsToFrontend(entries []model.Blacklist) []gin.H {
	result := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		result = append(result, blacklistToFrontend(entry))
	}
	return result
}

// sensitiveWordToFrontend 将 SensitiveWord 模型转换为前端期望的 camelCase 格式。
// Enabled bool 转换为 status string ("active"/"inactive")。
func sensitiveWordToFrontend(word model.SensitiveWord) gin.H {
	status := "inactive"
	if word.Enabled {
		status = "active"
	}
	return gin.H{
		"id":        word.ID,
		"word":      word.Word,
		"level":     word.Level,
		"status":    status,
		"createdAt": word.CreatedAt.Format(frontendTimeFormat),
		"updatedAt": word.UpdatedAt.Format(frontendTimeFormat),
	}
}

// sensitiveWordsToFrontend 将 SensitiveWord 列表转换为前端期望的 camelCase 格式。
func sensitiveWordsToFrontend(words []model.SensitiveWord) []gin.H {
	result := make([]gin.H, 0, len(words))
	for _, word := range words {
		result = append(result, sensitiveWordToFrontend(word))
	}
	return result
}

// departmentToFrontend 将 Department 模型转换为前端期望的 camelCase 格式。
// 递归转换 SubDepartments 子部门。
func departmentToFrontend(dept model.Department) gin.H {
	var parentID interface{}
	if dept.ParentID != nil {
		parentID = *dept.ParentID
	}

	subs := make([]gin.H, 0, len(dept.SubDepartments))
	for _, sub := range dept.SubDepartments {
		subs = append(subs, departmentToFrontend(sub))
	}

	return gin.H{
		"id":             dept.ID,
		"name":           dept.Name,
		"externalId":     dept.ExternalID,
		"parentId":       parentID,
		"level":          dept.Level,
		"path":           dept.Path,
		"sortOrder":      dept.SortOrder,
		"createdAt":      dept.CreatedAt.Format(frontendTimeFormat),
		"updatedAt":      dept.UpdatedAt.Format(frontendTimeFormat),
		"subDepartments": subs,
		"employees":      dept.Employees,
	}
}

// departmentsToFrontend 将 Department 列表转换为前端期望的 camelCase 格式。
func departmentsToFrontend(depts []model.Department) []gin.H {
	result := make([]gin.H, 0, len(depts))
	for _, dept := range depts {
		result = append(result, departmentToFrontend(dept))
	}
	return result
}
