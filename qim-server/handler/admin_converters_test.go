package handler

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestOperationLogToFrontend_ReturnsCamelCaseFields 验证 OperationLog 转换为 camelCase
func TestOperationLogToFrontend_ReturnsCamelCaseFields(t *testing.T) {
	log := model.OperationLog{
		ID:          1,
		UserID:      100,
		Username:    "alice",
		Action:      "login",
		Module:      "auth",
		IP:          "127.0.0.1",
		UserAgent:   "Mozilla/5.0",
		RequestURL:  "/api/v1/login",
		RequestBody: `{"username":"alice"}`,
		Response:    `{"code":0}`,
		Duration:    42,
	}
	log.CreatedAt = time.Now()

	result := operationLogToFrontend(log)

	assert.Equal(t, uint(1), result["id"])
	assert.Equal(t, uint(100), result["userId"])
	assert.Equal(t, "alice", result["username"])
	assert.Equal(t, "login", result["action"])
	assert.Equal(t, "auth", result["module"])
	assert.Equal(t, "127.0.0.1", result["ip"])
	assert.Equal(t, "Mozilla/5.0", result["userAgent"])
	assert.Equal(t, "/api/v1/login", result["requestUrl"])
	assert.Equal(t, `{"username":"alice"}`, result["requestBody"])
	assert.Equal(t, `{"code":0}`, result["response"])
	assert.Equal(t, 42, result["duration"])
	assert.NotNil(t, result["createdAt"])

	// 确保没有 snake_case 字段泄漏
	assert.NotContains(t, result, "user_id")
	assert.NotContains(t, result, "user_agent")
	assert.NotContains(t, result, "request_url")
	assert.NotContains(t, result, "request_body")
	assert.NotContains(t, result, "created_at")
}

// TestOperationLogsToFrontend_ReturnsCamelCaseList 验证列表转换
func TestOperationLogsToFrontend_ReturnsCamelCaseList(t *testing.T) {
	logs := []model.OperationLog{
		{ID: 1, UserID: 100, UserAgent: "UA1", RequestURL: "/a"},
		{ID: 2, UserID: 200, UserAgent: "UA2", RequestURL: "/b"},
	}

	result := operationLogsToFrontend(logs)

	assert.Len(t, result, 2)
	assert.Equal(t, uint(100), result[0]["userId"])
	assert.Equal(t, "UA1", result[0]["userAgent"])
	assert.Equal(t, "/a", result[0]["requestUrl"])
	assert.Equal(t, uint(200), result[1]["userId"])
	assert.Equal(t, "UA2", result[1]["userAgent"])
	assert.Equal(t, "/b", result[1]["requestUrl"])
}

// TestOperationLogsToFrontend_ReturnsEmptyListForEmptyInput 验证空列表
func TestOperationLogsToFrontend_ReturnsEmptyListForEmptyInput(t *testing.T) {
	result := operationLogsToFrontend([]model.OperationLog{})
	assert.Len(t, result, 0)

	resultNil := operationLogsToFrontend(nil)
	assert.Len(t, resultNil, 0)
}

// TestBlacklistToFrontend_ReturnsCamelCaseFields 验证 Blacklist 转换为 camelCase
func TestBlacklistToFrontend_ReturnsCamelCaseFields(t *testing.T) {
	entry := model.Blacklist{
		ID:       1,
		UserID:   100,
		Reason:   "违规操作",
		Operator: "admin",
	}
	entry.CreatedAt = time.Now()

	result := blacklistToFrontend(entry)

	assert.Equal(t, uint(1), result["id"])
	assert.Equal(t, uint(100), result["userId"])
	assert.Equal(t, "违规操作", result["reason"])
	assert.Equal(t, "admin", result["operator"])
	assert.NotNil(t, result["createdAt"])

	// 确保没有 snake_case 字段泄漏
	assert.NotContains(t, result, "user_id")
	assert.NotContains(t, result, "created_at")
}

// TestBlacklistsToFrontend_ReturnsCamelCaseList 验证列表转换
func TestBlacklistsToFrontend_ReturnsCamelCaseList(t *testing.T) {
	entries := []model.Blacklist{
		{ID: 1, UserID: 100, Reason: "r1"},
		{ID: 2, UserID: 200, Reason: "r2"},
	}

	result := blacklistsToFrontend(entries)

	assert.Len(t, result, 2)
	assert.Equal(t, uint(100), result[0]["userId"])
	assert.Equal(t, uint(200), result[1]["userId"])
}

// TestSensitiveWordToFrontend_ReturnsCamelCaseFields 验证 SensitiveWord 转换
// Enabled bool 转换为 status string ("active"/"inactive")
func TestSensitiveWordToFrontend_ReturnsCamelCaseFields(t *testing.T) {
	word := model.SensitiveWord{
		ID:      1,
		Word:    "敏感词",
		Level:   "high",
		Enabled: true,
	}
	word.CreatedAt = time.Now()
	word.UpdatedAt = time.Now()

	result := sensitiveWordToFrontend(word)

	assert.Equal(t, uint(1), result["id"])
	assert.Equal(t, "敏感词", result["word"])
	assert.Equal(t, "high", result["level"])
	assert.Equal(t, "active", result["status"])
	assert.NotNil(t, result["createdAt"])
	assert.NotNil(t, result["updatedAt"])

	// 确保没有 snake_case / 原始字段泄漏
	assert.NotContains(t, result, "created_at")
	assert.NotContains(t, result, "updated_at")
	assert.NotContains(t, result, "enabled")
}

// TestSensitiveWordToFrontend_DisabledReturnsInactiveStatus 验证禁用状态
func TestSensitiveWordToFrontend_DisabledReturnsInactiveStatus(t *testing.T) {
	word := model.SensitiveWord{
		ID:      2,
		Word:    "test",
		Level:   "low",
		Enabled: false,
	}

	result := sensitiveWordToFrontend(word)

	assert.Equal(t, "inactive", result["status"])
}

// TestSensitiveWordsToFrontend_ReturnsCamelCaseList 验证列表转换
func TestSensitiveWordsToFrontend_ReturnsCamelCaseList(t *testing.T) {
	words := []model.SensitiveWord{
		{ID: 1, Word: "w1", Enabled: true},
		{ID: 2, Word: "w2", Enabled: false},
	}

	result := sensitiveWordsToFrontend(words)

	assert.Len(t, result, 2)
	assert.Equal(t, "active", result[0]["status"])
	assert.Equal(t, "inactive", result[1]["status"])
}

// TestDepartmentToFrontend_ReturnsCamelCaseFields 验证 Department 转换为 camelCase
func TestDepartmentToFrontend_ReturnsCamelCaseFields(t *testing.T) {
	parentID := uint(10)
	dept := model.Department{
		ID:         1,
		Name:       "技术部",
		ExternalID: "ext-001",
		ParentID:   &parentID,
		Level:      2,
		Path:       "/root/tech",
		SortOrder:  5,
	}
	dept.CreatedAt = time.Now()
	dept.UpdatedAt = time.Now()

	result := departmentToFrontend(dept)

	assert.Equal(t, uint(1), result["id"])
	assert.Equal(t, "技术部", result["name"])
	assert.Equal(t, "ext-001", result["externalId"])
	assert.Equal(t, parentID, result["parentId"])
	assert.Equal(t, 2, result["level"])
	assert.Equal(t, "/root/tech", result["path"])
	assert.Equal(t, 5, result["sortOrder"])
	assert.NotNil(t, result["createdAt"])
	assert.NotNil(t, result["updatedAt"])

	// 确保没有 snake_case 字段泄漏
	assert.NotContains(t, result, "external_id")
	assert.NotContains(t, result, "parent_id")
	assert.NotContains(t, result, "sort_order")
	assert.NotContains(t, result, "created_at")
	assert.NotContains(t, result, "updated_at")
}

// TestDepartmentToFrontend_NilParentIDReturnsNil 验证 nil ParentID 处理
func TestDepartmentToFrontend_NilParentIDReturnsNil(t *testing.T) {
	dept := model.Department{
		ID:       1,
		Name:     "root",
		ParentID: nil,
	}

	result := departmentToFrontend(dept)

	assert.Nil(t, result["parentId"])
}

// TestDepartmentToFrontend_RecursivelyConvertsSubDepartments 验证递归转换子部门
func TestDepartmentToFrontend_RecursivelyConvertsSubDepartments(t *testing.T) {
	childParentID := uint(1)
	dept := model.Department{
		ID:         1,
		Name:       "技术部",
		ExternalID: "ext-root",
		SortOrder:  1,
		SubDepartments: []model.Department{
			{
				ID:         2,
				Name:       "前端组",
				ExternalID: "ext-child",
				ParentID:   &childParentID,
				SortOrder:  10,
			},
		},
	}

	result := departmentToFrontend(dept)

	subs, ok := result["subDepartments"].([]gin.H)
	assert.True(t, ok, "subDepartments 应为 []gin.H 类型")
	assert.Len(t, subs, 1)
	assert.Equal(t, uint(2), subs[0]["id"])
	assert.Equal(t, "前端组", subs[0]["name"])
	assert.Equal(t, "ext-child", subs[0]["externalId"])
	assert.Equal(t, childParentID, subs[0]["parentId"])
	assert.Equal(t, 10, subs[0]["sortOrder"])

	// 确保子部门也没有 snake_case 字段泄漏
	assert.NotContains(t, subs[0], "external_id")
	assert.NotContains(t, subs[0], "parent_id")
	assert.NotContains(t, subs[0], "sort_order")
}

// TestDepartmentToFrontend_EmptySubDepartmentsReturnsEmptyList 验证空子部门
func TestDepartmentToFrontend_EmptySubDepartmentsReturnsEmptyList(t *testing.T) {
	dept := model.Department{
		ID:             1,
		Name:           "leaf",
		SubDepartments: []model.Department{},
	}

	result := departmentToFrontend(dept)

	subs, ok := result["subDepartments"].([]gin.H)
	assert.True(t, ok)
	assert.Len(t, subs, 0)
}

// TestDepartmentToFrontend_NilSubDepartmentsReturnsEmptyList 验证 nil 子部门
func TestDepartmentToFrontend_NilSubDepartmentsReturnsEmptyList(t *testing.T) {
	dept := model.Department{
		ID:             1,
		Name:           "leaf",
		SubDepartments: nil,
	}

	result := departmentToFrontend(dept)

	subs, ok := result["subDepartments"].([]gin.H)
	assert.True(t, ok)
	assert.Len(t, subs, 0)
}

// TestDepartmentsToFrontend_ReturnsCamelCaseList 验证部门列表转换
func TestDepartmentsToFrontend_ReturnsCamelCaseList(t *testing.T) {
	depts := []model.Department{
		{ID: 1, Name: "a", ExternalID: "ea"},
		{ID: 2, Name: "b", ExternalID: "eb"},
	}

	result := departmentsToFrontend(depts)

	assert.Len(t, result, 2)
	assert.Equal(t, "ea", result[0]["externalId"])
	assert.Equal(t, "eb", result[1]["externalId"])
}

// TestDepartmentsToFrontend_ReturnsEmptyListForEmptyInput 验证空列表
func TestDepartmentsToFrontend_ReturnsEmptyListForEmptyInput(t *testing.T) {
	result := departmentsToFrontend([]model.Department{})
	assert.Len(t, result, 0)

	resultNil := departmentsToFrontend(nil)
	assert.Len(t, resultNil, 0)
}
