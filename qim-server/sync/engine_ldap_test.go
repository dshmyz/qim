package sync

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/orgsync"
	"github.com/stretchr/testify/assert"
)

// 模拟真实 LDAP 同步场景：
// - 本地已有用户（通过注册或之前创建的）
// - LDAP 返回部门、用户、用户-部门关系
// - 验证用户能正确挂到部门下

func TestSyncToLocal_ExistingUserMatchByUsername(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	// 本地已有用户
	db.Create(&model.User{Username: "zhangsan", PasswordHash: "hash", Nickname: "张三"})

	config := &model.OrgSyncConfig{SyncType: "ldap"}
	data := &orgsync.OrgData{
		Departments: []orgsync.DepartmentInfo{
			{ID: "ldap-dept-1", Name: "研发部", Level: 1},
		},
		Users: []orgsync.UserInfo{
			{ID: "ldap-uuid-001", Username: "zhangsan", Nickname: "张三"},
		},
		UserDeptRelations: []orgsync.UserDeptRelation{
			{UserID: "ldap-uuid-001", DepartmentID: "ldap-dept-1"},
		},
	}

	stats := e.syncToLocal(data, config)

	// 验证部门创建
	assert.Equal(t, 1, stats.DeptsCreated)

	// 验证用户被挂到部门下
	var dept model.Department
	db.Where("external_id = ?", "ldap-dept-1").First(&dept)

	var empCount int64
	db.Model(&model.DepartmentEmployee{}).Where("department_id = ?", dept.ID).Count(&empCount)
	assert.Equal(t, int64(1), empCount)

	var user model.User
	db.Where("username = ?", "zhangsan").First(&user)

	var emp model.DepartmentEmployee
	db.Where("user_id = ? AND department_id = ?", user.ID, dept.ID).First(&emp)
	assert.True(t, emp.IsPrimary)
}

func TestSyncToLocal_ExistingUserMatchByEmail(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	// 本地用户 username 不同，但 email 一致
	db.Create(&model.User{Username: "zs_account", PasswordHash: "hash", Email: "zhangsan@company.com"})

	config := &model.OrgSyncConfig{SyncType: "ldap"}
	data := &orgsync.OrgData{
		Departments: []orgsync.DepartmentInfo{
			{ID: "ldap-dept-1", Name: "研发部", Level: 1},
		},
		Users: []orgsync.UserInfo{
			{ID: "ldap-uuid-002", Username: "zhangsan", Email: "zhangsan@company.com"},
		},
		UserDeptRelations: []orgsync.UserDeptRelation{
			{UserID: "ldap-uuid-002", DepartmentID: "ldap-dept-1"},
		},
	}

	stats := e.syncToLocal(data, config)
	assert.Equal(t, 1, stats.DeptsCreated)

	// 验证通过 email 匹配到了本地用户
	var dept model.Department
	db.Where("external_id = ?", "ldap-dept-1").First(&dept)

	var user model.User
	db.Where("username = ?", "zs_account").First(&user)

	var empCount int64
	db.Model(&model.DepartmentEmployee{}).Where("user_id = ? AND department_id = ?", user.ID, dept.ID).Count(&empCount)
	assert.Equal(t, int64(1), empCount)
}

func TestSyncToLocal_ExistingUserMatchByNickname(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	// 本地用户 username 和 email 都不同，但 nickname 一致
	db.Create(&model.User{Username: "user_1001", PasswordHash: "hash", Nickname: "张三"})

	config := &model.OrgSyncConfig{SyncType: "ldap"}
	data := &orgsync.OrgData{
		Departments: []orgsync.DepartmentInfo{
			{ID: "ldap-dept-1", Name: "研发部", Level: 1},
		},
		Users: []orgsync.UserInfo{
			{ID: "ldap-uuid-003", Username: "zhangsan", Nickname: "张三"},
		},
		UserDeptRelations: []orgsync.UserDeptRelation{
			{UserID: "ldap-uuid-003", DepartmentID: "ldap-dept-1"},
		},
	}

	stats := e.syncToLocal(data, config)
	assert.Equal(t, 1, stats.DeptsCreated)

	var dept model.Department
	db.Where("external_id = ?", "ldap-dept-1").First(&dept)

	var user model.User
	db.Where("username = ?", "user_1001").First(&user)

	var empCount int64
	db.Model(&model.DepartmentEmployee{}).Where("user_id = ? AND department_id = ?", user.ID, dept.ID).Count(&empCount)
	assert.Equal(t, int64(1), empCount)
}

func TestSyncToLocal_MultipleUsersAndDepartments(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	// 本地已有两个用户
	db.Create(&model.User{Username: "zhangsan", PasswordHash: "hash"})
	db.Create(&model.User{Username: "lisi", PasswordHash: "hash"})

	config := &model.OrgSyncConfig{SyncType: "ldap"}
	data := &orgsync.OrgData{
		Departments: []orgsync.DepartmentInfo{
			{ID: "ldap-dept-1", Name: "公司", Level: 0},
			{ID: "ldap-dept-2", Name: "研发部", ParentID: "ldap-dept-1", Level: 1},
			{ID: "ldap-dept-3", Name: "产品部", ParentID: "ldap-dept-1", Level: 1},
		},
		Users: []orgsync.UserInfo{
			{ID: "uuid-001", Username: "zhangsan"},
			{ID: "uuid-002", Username: "lisi"},
		},
		UserDeptRelations: []orgsync.UserDeptRelation{
			{UserID: "uuid-001", DepartmentID: "ldap-dept-2"}, // zhangsan → 研发部
			{UserID: "uuid-002", DepartmentID: "ldap-dept-3"}, // lisi → 产品部
		},
	}

	stats := e.syncToLocal(data, config)

	assert.Equal(t, 3, stats.DeptsCreated)

	// 验证 zhangsan 在研发部
	var devDept model.Department
	db.Where("external_id = ?", "ldap-dept-2").First(&devDept)

	var user1 model.User
	db.Where("username = ?", "zhangsan").First(&user1)

	var emp1Count int64
	db.Model(&model.DepartmentEmployee{}).Where("user_id = ? AND department_id = ?", user1.ID, devDept.ID).Count(&emp1Count)
	assert.Equal(t, int64(1), emp1Count)

	// 验证 lisi 在产品部
	var prodDept model.Department
	db.Where("external_id = ?", "ldap-dept-3").First(&prodDept)

	var user2 model.User
	db.Where("username = ?", "lisi").First(&user2)

	var emp2Count int64
	db.Model(&model.DepartmentEmployee{}).Where("user_id = ? AND department_id = ?", user2.ID, prodDept.ID).Count(&emp2Count)
	assert.Equal(t, int64(1), emp2Count)
}

func TestSyncToLocal_UserNotInLocalDB_Skipped(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	// 本地没有 wangwu
	config := &model.OrgSyncConfig{SyncType: "ldap"}
	data := &orgsync.OrgData{
		Departments: []orgsync.DepartmentInfo{
			{ID: "ldap-dept-1", Name: "研发部", Level: 1},
		},
		Users: []orgsync.UserInfo{
			{ID: "uuid-001", Username: "wangwu"},
		},
		UserDeptRelations: []orgsync.UserDeptRelation{
			{UserID: "uuid-001", DepartmentID: "ldap-dept-1"},
		},
	}

	stats := e.syncToLocal(data, config)

	// 部门创建了，但没有员工关联
	assert.Equal(t, 1, stats.DeptsCreated)

	var empCount int64
	db.Model(&model.DepartmentEmployee{}).Count(&empCount)
	assert.Equal(t, int64(0), empCount)
}

func TestSyncToLocal_DuplicateSync_NoDuplicateRecords(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	db.Create(&model.User{Username: "zhangsan", PasswordHash: "hash"})

	config := &model.OrgSyncConfig{SyncType: "ldap"}
	data := &orgsync.OrgData{
		Departments: []orgsync.DepartmentInfo{
			{ID: "ldap-dept-1", Name: "研发部", Level: 1},
		},
		Users: []orgsync.UserInfo{
			{ID: "uuid-001", Username: "zhangsan"},
		},
		UserDeptRelations: []orgsync.UserDeptRelation{
			{UserID: "uuid-001", DepartmentID: "ldap-dept-1"},
		},
	}

	// 第一次同步
	e.syncToLocal(data, config)
	// 第二次同步
	e.syncToLocal(data, config)

	// 不应产生重复的员工关联
	var empCount int64
	db.Model(&model.DepartmentEmployee{}).Count(&empCount)
	assert.Equal(t, int64(1), empCount)

	// 不应产生重复的部门
	var deptCount int64
	db.Model(&model.Department{}).Where("external_id = ?", "ldap-dept-1").Count(&deptCount)
	assert.Equal(t, int64(1), deptCount)
}
