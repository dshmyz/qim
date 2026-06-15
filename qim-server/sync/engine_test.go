package sync

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/orgsync"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"

	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.Department{},
		&model.DepartmentEmployee{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestResolveUserByRelation_ByUsername(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	// 本地用户 username = "zhangsan"
	db.Create(&model.User{
		Username:     "zhangsan",
		PasswordHash: "hash",
		Nickname:     "张三",
	})

	data := &orgsync.OrgData{
		Users: []orgsync.UserInfo{
			{ID: "ext_001", Username: "zhangsan", Nickname: "张三"},
		},
	}

	rel := orgsync.UserDeptRelation{
		UserID:       "ext_001",
		DepartmentID: "dept_001",
	}

	user := e.resolveUserByRelation(rel, data, "ldap")
	assert.NotNil(t, user)
	assert.Equal(t, "zhangsan", user.Username)
}

func TestResolveUserByRelation_ByEmail(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	// 本地用户 username 不同，但 email 一致
	db.Create(&model.User{
		Username:     "zs_official",
		PasswordHash: "hash",
		Email:        "zhangsan@company.com",
	})

	data := &orgsync.OrgData{
		Users: []orgsync.UserInfo{
			{ID: "ext_002", Username: "zhangsan", Email: "zhangsan@company.com"},
		},
	}

	rel := orgsync.UserDeptRelation{
		UserID:       "ext_002",
		DepartmentID: "dept_001",
	}

	user := e.resolveUserByRelation(rel, data, "ldap")
	assert.NotNil(t, user)
	assert.Equal(t, "zs_official", user.Username)
}

func TestResolveUserByRelation_ByNickname(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	// 本地用户 username 和 email 都不同，但 nickname 一致
	db.Create(&model.User{
		Username:     "user_zhang",
		PasswordHash: "hash",
		Nickname:     "张三",
	})

	data := &orgsync.OrgData{
		Users: []orgsync.UserInfo{
			{ID: "ext_003", Username: "zhangsan", Email: "zhangsan@company.com", Nickname: "张三"},
		},
	}

	rel := orgsync.UserDeptRelation{
		UserID:       "ext_003",
		DepartmentID: "dept_001",
	}

	user := e.resolveUserByRelation(rel, data, "ldap")
	assert.NotNil(t, user)
	assert.Equal(t, "user_zhang", user.Username)
}

func TestResolveUserByRelation_NoMatch(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	// 本地用户
	db.Create(&model.User{
		Username:     "lisi",
		PasswordHash: "hash",
		Nickname:     "李四",
	})

	// 外部用户 username/email/nickname 都不匹配
	data := &orgsync.OrgData{
		Users: []orgsync.UserInfo{
			{ID: "ext_004", Username: "wangwu", Email: "wangwu@company.com", Nickname: "王五"},
		},
	}

	rel := orgsync.UserDeptRelation{
		UserID:       "ext_004",
		DepartmentID: "dept_001",
	}

	user := e.resolveUserByRelation(rel, data, "ldap")
	assert.Nil(t, user)
}

func TestResolveUserByRelation_ExternalIDAsUsername(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	// 外部系统 ID 恰好就是本地 username
	db.Create(&model.User{
		Username:     "emp_123",
		PasswordHash: "hash",
	})

	data := &orgsync.OrgData{
		Users: []orgsync.UserInfo{
			{ID: "emp_123", Username: "emp_123"},
		},
	}

	rel := orgsync.UserDeptRelation{
		UserID:       "emp_123",
		DepartmentID: "dept_001",
	}

	// 第一步直接 username 匹配
	user := e.resolveUserByRelation(rel, data, "api")
	assert.NotNil(t, user)
	assert.Equal(t, "emp_123", user.Username)
}

func TestResolveUserByRelation_SoftDeletedUser(t *testing.T) {
	db := setupTestDB(t)
	e := &Engine{db: db}

	// 软删除的用户不应被匹配到（由调用方判断，这里只验证能查到）
	db.Create(&model.User{
		Username:     "zhangsan",
		PasswordHash: "hash",
	})

	data := &orgsync.OrgData{
		Users: []orgsync.UserInfo{
			{ID: "ext_005", Username: "zhangsan"},
		},
	}

	rel := orgsync.UserDeptRelation{
		UserID:       "ext_005",
		DepartmentID: "dept_001",
	}

	user := e.resolveUserByRelation(rel, data, "ldap")
	assert.NotNil(t, user)
}
