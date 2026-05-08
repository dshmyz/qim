# 后端分层架构重构计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将后端代码重构为标准的 Handler → Service → Repository 三层架构，实现职责分离。

**架构：** Handler 负责 HTTP 请求处理和响应，Service 负责业务逻辑和事务管理，Repository 负责数据访问封装。

**技术栈：** Go + Gin + GORM

---

## 当前问题

| 问题 | 严重程度 | 说明 |
|------|----------|------|
| Handler 直接操作数据库 | 🔴 高 | 214 处 `database.GetDB()` 调用 |
| Service 层缺失 | 🔴 高 | 仅 4 个 Service 文件 |
| 无 Repository 层 | 🟡 中 | 数据访问逻辑分散 |
| 职责混乱 | 🔴 高 | Handler 包含业务逻辑 |

## 目标架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Handler 层                            │
│  - 接收 HTTP 请求                                           │
│  - 参数验证                                                 │
│  - 调用 Service 方法                                        │
│  - 返回响应                                                 │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        Service 层                            │
│  - 业务逻辑处理                                             │
│  - 事务管理                                                 │
│  - 调用 Repository 方法                                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Repository 层                           │
│  - 数据访问封装                                             │
│  - CRUD 操作                                                │
│  - 查询构建                                                 │
└─────────────────────────────────────────────────────────────┘
```

## 文件结构

```
qim-server/
├── handler/           # HTTP 处理器（已有，需重构）
├── service/           # 业务逻辑层（已有，需扩展）
├── repository/        # 数据访问层（新建）
│   ├── user_repository.go
│   ├── conversation_repository.go
│   ├── message_repository.go
│   ├── group_repository.go
│   ├── file_repository.go
│   ├── notification_repository.go
│   └── ...
├── model/             # 数据模型（已有）
└── pkg/               # 公共包（已有）
```

## 重构优先级

按 Handler 文件中 `database.GetDB()` 调用次数排序：

| 优先级 | 模块 | 文件 | 调用次数 |
|--------|------|------|----------|
| 1 | 通知 | notification_handler.go | 17 |
| 2 | 应用 | app_handler.go | 16 |
| 3 | 文件 | file_handler.go | 14 |
| 4 | 用户 | user_handler.go | 14 |
| 5 | 管理 | admin_handler.go | 13 |
| 6 | 消息 | message_handler.go | 12 |
| 7 | 群组 | group_handler.go | 11 |
| 8 | 实时通信 | realtime_handler.go | 10 |
| 9 | 会话 | conversation_handler.go | 9 |
| 10 | 敏感词 | sensitive_word_handler.go | 9 |

---

## 任务 1：创建 Repository 层基础设施

**文件：**
- 创建：`qim-server/repository/base_repository.go`
- 创建：`qim-server/repository/interfaces.go`

- [ ] **步骤 1：创建 Repository 接口定义**

```go
// qim-server/repository/interfaces.go
package repository

import (
	"context"

	"gorm.io/gorm"
)

// BaseRepository 基础仓库接口
type BaseRepository[T any] interface {
	// Create 创建记录
	Create(ctx context.Context, entity *T) error
	// CreateBatch 批量创建
	CreateBatch(ctx context.Context, entities []*T) error
	// Update 更新记录
	Update(ctx context.Context, entity *T) error
	// Delete 软删除
	Delete(ctx context.Context, id uint) error
	// HardDelete 硬删除
	HardDelete(ctx context.Context, id uint) error
	// FindByID 根据 ID 查询
	FindByID(ctx context.Context, id uint) (*T, error)
	// FindAll 查询所有
	FindAll(ctx context.Context) ([]*T, error)
	// Count 统计数量
	Count(ctx context.Context) (int64, error)
	// Exists 检查是否存在
	Exists(ctx context.Context, id uint) (bool, error)
	// DB 获取数据库连接（用于复杂查询）
	DB() *gorm.DB
	// WithTx 使用事务
	WithTx(tx *gorm.DB) BaseRepository[T]
}

// UserRepository 用户仓库接口
type UserRepository interface {
	BaseRepository[User]
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByPhone(ctx context.Context, phone string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Search(ctx context.Context, query string, limit int) ([]*User, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	UpdateLastOnline(ctx context.Context, id uint) error
}

// ConversationRepository 会话仓库接口
type ConversationRepository interface {
	BaseRepository[Conversation]
	FindByUserID(ctx context.Context, userID uint) ([]*Conversation, error)
	FindSingleConversation(ctx context.Context, userID1, userID2 uint) (*Conversation, error)
	AddMember(ctx context.Context, conversationID, userID uint, role string) error
	RemoveMember(ctx context.Context, conversationID, userID uint) error
	UpdateMemberRole(ctx context.Context, conversationID, userID uint, role string) error
}

// MessageRepository 消息仓库接口
type MessageRepository interface {
	BaseRepository[Message]
	FindByConversationID(ctx context.Context, conversationID uint, limit, offset int) ([]*Message, error)
	FindLatestByConversationID(ctx context.Context, conversationID uint) (*Message, error)
	RecallMessage(ctx context.Context, id uint) error
	MarkAsRead(ctx context.Context, id uint) error
}

// GroupRepository 群组仓库接口
type GroupRepository interface {
	BaseRepository[Group]
	FindByConversationID(ctx context.Context, conversationID uint) (*Group, error)
	FindByCreatorID(ctx context.Context, creatorID uint) ([]*Group, error)
	UpdateAnnouncement(ctx context.Context, id uint, announcement string) error
	AddMember(ctx context.Context, groupID, userID uint) error
	RemoveMember(ctx context.Context, groupID, userID uint) error
}

// FileRepository 文件仓库接口
type FileRepository interface {
	BaseRepository[File]
	FindByUserID(ctx context.Context, userID uint) ([]*File, error)
	FindByFolderID(ctx context.Context, folderID *uint) ([]*File, error)
	FindByChecksum(ctx context.Context, checksum string) (*File, error)
	UpdateStarred(ctx context.Context, id uint, starred bool) error
}

// NotificationRepository 通知仓库接口
type NotificationRepository interface {
	BaseRepository[Notification]
	FindByUserID(ctx context.Context, userID uint, unreadOnly bool) ([]*Notification, error)
	MarkAsRead(ctx context.Context, id uint) error
	MarkAllAsRead(ctx context.Context, userID uint) error
	CountUnread(ctx context.Context, userID uint) (int64, error)
}
```

- [ ] **步骤 2：创建基础 Repository 实现**

```go
// qim-server/repository/base_repository.go
package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// baseRepository 基础仓库实现
type baseRepository[T any] struct {
	db *gorm.DB
}

// NewBaseRepository 创建基础仓库
func NewBaseRepository[T any](db *gorm.DB) BaseRepository[T] {
	return &baseRepository[T]{db: db}
}

func (r *baseRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *baseRepository[T]) CreateBatch(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(entities).Error
}

func (r *baseRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *baseRepository[T]) Delete(ctx context.Context, id uint) error {
	var entity T
	return r.db.WithContext(ctx).Delete(&entity, id).Error
}

func (r *baseRepository[T]) HardDelete(ctx context.Context, id uint) error {
	var entity T
	return r.db.WithContext(ctx).Unscoped().Delete(&entity, id).Error
}

func (r *baseRepository[T]) FindByID(ctx context.Context, id uint) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).First(&entity, id).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *baseRepository[T]) FindAll(ctx context.Context) ([]*T, error) {
	var entities []*T
	err := r.db.WithContext(ctx).Find(&entities).Error
	return entities, err
}

func (r *baseRepository[T]) Count(ctx context.Context) (int64, error) {
	var count int64
	var entity T
	err := r.db.WithContext(ctx).Model(&entity).Count(&count).Error
	return count, err
}

func (r *baseRepository[T]) Exists(ctx context.Context, id uint) (bool, error) {
	var count int64
	var entity T
	err := r.db.WithContext(ctx).Model(&entity).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *baseRepository[T]) DB() *gorm.DB {
	return r.db
}

func (r *baseRepository[T]) WithTx(tx *gorm.DB) BaseRepository[T] {
	return &baseRepository[T]{db: tx}
}

// Paginate 分页辅助函数
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}
		if pageSize > 100 {
			pageSize = 100
		}
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

// ErrRecordNotFound 记录未找到错误
var ErrRecordNotFound = errors.New("record not found")
```

- [ ] **步骤 3：验证编译**

```bash
cd qim-server && go build ./...
```

预期：编译成功，无错误

- [ ] **步骤 4：Commit**

```bash
git add qim-server/repository/
git commit -m "feat(repository): 添加 Repository 层基础设施

- 定义 BaseRepository 接口
- 实现通用 CRUD 方法
- 定义各模块 Repository 接口"
```

---

## 任务 2：创建用户模块 Repository

**文件：**
- 创建：`qim-server/repository/user_repository.go`
- 测试：`qim-server/repository/user_repository_test.go`

- [ ] **步骤 1：编写失败的测试**

```go
// qim-server/repository/user_repository_test.go
package repository

import (
	"context"
	"testing"

	"qim-server/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	
	err = db.AutoMigrate(&model.User{}, &model.UserRole{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	
	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Nickname:     "Test User",
	}

	err := repo.Create(ctx, user)
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)
}

func TestUserRepository_FindByUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{
		Username:     "testuser",
		PasswordHash: "hash",
		Nickname:     "Test User",
	}
	repo.Create(ctx, user)

	// 测试查找
	found, err := repo.FindByUsername(ctx, "testuser")
	assert.NoError(t, err)
	assert.Equal(t, user.Username, found.Username)

	// 测试找不到
	_, err = repo.FindByUsername(ctx, "notexist")
	assert.Error(t, err)
}

func TestUserRepository_Search(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	// 创建测试用户
	users := []*model.User{
		{Username: "zhangsan", Nickname: "张三"},
		{Username: "lisi", Nickname: "李四"},
		{Username: "wangwu", Nickname: "王五"},
	}
	for _, u := range users {
		repo.Create(ctx, u)
	}

	// 测试搜索
	results, err := repo.Search(ctx, "张", 10)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "张三", results[0].Nickname)
}
```

- [ ] **步骤 2：运行测试验证失败**

```bash
cd qim-server && go test ./repository/... -v
```

预期：FAIL，报错 "NewUserRepository not defined"

- [ ] **步骤 3：实现 UserRepository**

```go
// qim-server/repository/user_repository.go
package repository

import (
	"context"

	"qim-server/model"

	"gorm.io/gorm"
)

// userRepository 用户仓库实现
type userRepository struct {
	*baseRepository[model.User]
	db *gorm.DB
}

// NewUserRepository 创建用户仓库
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		baseRepository: &baseRepository[model.User]{db: db},
		db:             db,
	}
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Search(ctx context.Context, query string, limit int) ([]*model.User, error) {
	var users []*model.User
	searchPattern := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("username LIKE ? OR nickname LIKE ? OR phone LIKE ? OR email LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern).
		Limit(limit).
		Find(&users).Error
	return users, err
}

func (r *userRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}

func (r *userRepository) UpdateLastOnline(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("last_online", gorm.Expr("datetime('now')")).Error
}

func (r *userRepository) WithTx(tx *gorm.DB) UserRepository {
	return &userRepository{
		baseRepository: &baseRepository[model.User]{db: tx},
		db:             tx,
	}
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
cd qim-server && go test ./repository/... -v
```

预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add qim-server/repository/
git commit -m "feat(repository): 实现用户模块 Repository

- 实现 UserRepository 接口
- 添加用户查询、搜索方法
- 添加单元测试"
```

---

## 任务 3：创建会话模块 Repository

**文件：**
- 创建：`qim-server/repository/conversation_repository.go`
- 测试：`qim-server/repository/conversation_repository_test.go`

- [ ] **步骤 1：编写失败的测试**

```go
// qim-server/repository/conversation_repository_test.go
package repository

import (
	"context"
	"testing"

	"qim-server/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupConversationTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	
	err = db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.ConversationSession{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	
	return db
}

func TestConversationRepository_FindByUserID(t *testing.T) {
	db := setupConversationTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// 创建测试用户
	user := &model.User{Username: "testuser"}
	db.Create(user)

	// 创建会话
	conv := &model.Conversation{Type: "single"}
	db.Create(conv)

	// 添加成员
	db.Create(&model.ConversationMember{
		ConversationID: conv.ID,
		UserID:         user.ID,
		Role:           "member",
	})

	// 测试查找
	convs, err := repo.FindByUserID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Len(t, convs, 1)
}

func TestConversationRepository_AddMember(t *testing.T) {
	db := setupConversationTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// 创建测试数据
	user := &model.User{Username: "testuser"}
	db.Create(user)
	conv := &model.Conversation{Type: "group"}
	db.Create(conv)

	// 测试添加成员
	err := repo.AddMember(ctx, conv.ID, user.ID, "member")
	assert.NoError(t, err)

	// 验证
	var member model.ConversationMember
	err = db.Where("conversation_id = ? AND user_id = ?", conv.ID, user.ID).First(&member).Error
	assert.NoError(t, err)
	assert.Equal(t, "member", member.Role)
}
```

- [ ] **步骤 2：运行测试验证失败**

```bash
cd qim-server && go test ./repository/... -v -run TestConversation
```

预期：FAIL

- [ ] **步骤 3：实现 ConversationRepository**

```go
// qim-server/repository/conversation_repository.go
package repository

import (
	"context"
	"time"

	"qim-server/model"

	"gorm.io/gorm"
)

// conversationRepository 会话仓库实现
type conversationRepository struct {
	*baseRepository[model.Conversation]
	db *gorm.DB
}

// NewConversationRepository 创建会话仓库
func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return &conversationRepository{
		baseRepository: &baseRepository[model.Conversation]{db: db},
		db:             db,
	}
}

func (r *conversationRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.Conversation, error) {
	var convMembers []model.ConversationMember
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Preload("Conversation").
		Preload("Conversation.LastMessage").
		Preload("Conversation.Members").
		Preload("Conversation.Members.User").
		Find(&convMembers).Error
	if err != nil {
		return nil, err
	}

	conversations := make([]*model.Conversation, 0, len(convMembers))
	for _, cm := range convMembers {
		conversations = append(conversations, &cm.Conversation)
	}
	return conversations, nil
}

func (r *conversationRepository) FindSingleConversation(ctx context.Context, userID1, userID2 uint) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.WithContext(ctx).
		Joins("JOIN conversation_members cm1 ON cm1.conversation_id = conversations.id").
		Joins("JOIN conversation_members cm2 ON cm2.conversation_id = conversations.id").
		Where("conversations.type = ?", "single").
		Where("cm1.user_id = ?", userID1).
		Where("cm2.user_id = ?", userID2).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *conversationRepository) AddMember(ctx context.Context, conversationID, userID uint, role string) error {
	member := &model.ConversationMember{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           role,
		JoinedAt:       time.Now(),
	}
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *conversationRepository) RemoveMember(ctx context.Context, conversationID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&model.ConversationMember{}).Error
}

func (r *conversationRepository) UpdateMemberRole(ctx context.Context, conversationID, userID uint, role string) error {
	return r.db.WithContext(ctx).
		Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("role", role).Error
}

func (r *conversationRepository) WithTx(tx *gorm.DB) ConversationRepository {
	return &conversationRepository{
		baseRepository: &baseRepository[model.Conversation]{db: tx},
		db:             tx,
	}
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
cd qim-server && go test ./repository/... -v -run TestConversation
```

预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add qim-server/repository/
git commit -m "feat(repository): 实现会话模块 Repository

- 实现 ConversationRepository 接口
- 添加会话成员管理方法
- 添加单元测试"
```

---

## 任务 4：创建消息模块 Repository

**文件：**
- 创建：`qim-server/repository/message_repository.go`
- 测试：`qim-server/repository/message_repository_test.go`

- [ ] **步骤 1：编写失败的测试**

```go
// qim-server/repository/message_repository_test.go
package repository

import (
	"context"
	"testing"

	"qim-server/model"

	"github.com/stretchr/testify/assert"
)

func TestMessageRepository_FindByConversationID(t *testing.T) {
	db := setupConversationTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	// 创建测试数据
	conv := &model.Conversation{Type: "single"}
	db.Create(conv)
	user := &model.User{Username: "testuser"}
	db.Create(user)

	// 创建消息
	for i := 1; i <= 5; i++ {
		msg := &model.Message{
			ConversationID: conv.ID,
			SenderID:       user.ID,
			Type:           "text",
			Content:        "test message",
		}
		db.Create(msg)
	}

	// 测试分页查询
	messages, err := repo.FindByConversationID(ctx, conv.ID, 3, 0)
	assert.NoError(t, err)
	assert.Len(t, messages, 3)
}

func TestMessageRepository_RecallMessage(t *testing.T) {
	db := setupConversationTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()

	// 创建测试数据
	conv := &model.Conversation{Type: "single"}
	db.Create(conv)
	user := &model.User{Username: "testuser"}
	db.Create(user)

	msg := &model.Message{
		ConversationID: conv.ID,
		SenderID:       user.ID,
		Type:           "text",
		Content:        "test message",
	}
	db.Create(msg)

	// 测试撤回
	err := repo.RecallMessage(ctx, msg.ID)
	assert.NoError(t, err)

	// 验证
	var recalled model.Message
	db.First(&recalled, msg.ID)
	assert.True(t, recalled.IsRecalled)
}
```

- [ ] **步骤 2：运行测试验证失败**

```bash
cd qim-server && go test ./repository/... -v -run TestMessage
```

预期：FAIL

- [ ] **步骤 3：实现 MessageRepository**

```go
// qim-server/repository/message_repository.go
package repository

import (
	"context"
	"time"

	"qim-server/model"

	"gorm.io/gorm"
)

// messageRepository 消息仓库实现
type messageRepository struct {
	*baseRepository[model.Message]
	db *gorm.DB
}

// NewMessageRepository 创建消息仓库
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{
		baseRepository: &baseRepository[model.Message]{db: db},
		db:             db,
	}
}

func (r *messageRepository) FindByConversationID(ctx context.Context, conversationID uint, limit, offset int) ([]*model.Message, error) {
	var messages []*model.Message
	query := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Where("is_recalled = ?", false).
		Order("created_at DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	err := query.Preload("Sender").Find(&messages).Error
	return messages, err
}

func (r *messageRepository) FindLatestByConversationID(ctx context.Context, conversationID uint) (*model.Message, error) {
	var message model.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Where("is_recalled = ?", false).
		Order("created_at DESC").
		First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *messageRepository) RecallMessage(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_recalled": true,
			"recalled_at": now,
		}).Error
}

func (r *messageRepository) MarkAsRead(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.Message{}).
		Where("id = ?", id).
		Update("is_read", true).Error
}

func (r *messageRepository) WithTx(tx *gorm.DB) MessageRepository {
	return &messageRepository{
		baseRepository: &baseRepository[model.Message]{db: tx},
		db:             tx,
	}
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
cd qim-server && go test ./repository/... -v -run TestMessage
```

预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add qim-server/repository/
git commit -m "feat(repository): 实现消息模块 Repository

- 实现 MessageRepository 接口
- 添加消息查询、撤回方法
- 添加单元测试"
```

---

## 任务 5：重构 UserService

**文件：**
- 修改：`qim-server/service/user_service.go`
- 修改：`qim-server/handler/user_handler.go`

- [ ] **步骤 1：重构 UserService 使用 Repository**

```go
// qim-server/service/user_service.go
package service

import (
	"context"
	"errors"

	"qim-server/model"
	"qim-server/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("用户不存在")
	ErrUserAlreadyExists = errors.New("用户已存在")
	ErrInvalidPassword   = errors.New("密码错误")
)

// UserService 用户服务
type UserService struct {
	userRepo repository.UserRepository
	db       *gorm.DB
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		userRepo: repository.NewUserRepository(db),
		db:       db,
	}
}

// WithRepo 使用指定的 Repository（用于测试）
func (s *UserService) WithRepo(repo repository.UserRepository) *UserService {
	return &UserService{
		userRepo: repo,
		db:       s.db,
	}
}

// GetByID 根据 ID 获取用户
func (s *UserService) GetByID(ctx context.Context, id uint) (*model.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// GetByUsername 根据用户名获取用户
func (s *UserService) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// Search 搜索用户
func (s *UserService) Search(ctx context.Context, query string, limit int) ([]*model.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.userRepo.Search(ctx, query, limit)
}

// Update 更新用户信息
func (s *UserService) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
	}

	// 应用更新
	if nickname, ok := updates["nickname"].(string); ok && nickname != "" {
		user.Nickname = nickname
	}
	if avatar, ok := updates["avatar"].(string); ok && avatar != "" {
		user.Avatar = avatar
	}
	if signature, ok := updates["signature"].(string); ok {
		user.Signature = signature
	}
	if phone, ok := updates["phone"].(string); ok {
		user.Phone = phone
	}
	if email, ok := updates["email"].(string); ok {
		user.Email = email
	}
	if twoFactor, ok := updates["two_factor_enabled"].(bool); ok {
		user.TwoFactorEnabled = twoFactor
	}

	return s.userRepo.Update(ctx, user)
}

// UpdatePassword 更新密码
func (s *UserService) UpdatePassword(ctx context.Context, id uint, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidPassword
	}

	// 生成新密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hashedPassword)
	return s.userRepo.Update(ctx, user)
}

// UpdateStatus 更新用户状态
func (s *UserService) UpdateStatus(ctx context.Context, id uint, status string) error {
	return s.userRepo.UpdateStatus(ctx, id, status)
}

// GetUserRoles 获取用户角色
func (s *UserService) GetUserRoles(ctx context.Context, userID uint) ([]string, error) {
	var userRoles []model.UserRole
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&userRoles).Error
	if err != nil {
		return nil, err
	}

	roles := make([]string, 0, len(userRoles))
	for _, ur := range userRoles {
		roles = append(roles, ur.Role)
	}
	return roles, nil
}
```

- [ ] **步骤 2：重构 UserHandler 使用 Service**

```go
// qim-server/handler/user_handler.go (部分重构)
package handler

import (
	"qim-server/model"
	"qim-server/pkg/response"
	"qim-server/service"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetCurrentUser 获取当前用户
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	user, err := h.userService.GetByID(c.Request.Context(), uid)
	if err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	roles, _ := h.userService.GetUserRoles(c.Request.Context(), uid)

	response.Success(c, gin.H{
		"id":                 user.ID,
		"username":           user.Username,
		"nickname":           user.Nickname,
		"avatar":             user.Avatar,
		"signature":          user.Signature,
		"phone":              user.Phone,
		"email":              user.Email,
		"status":             user.Status,
		"two_factor_enabled": user.TwoFactorEnabled,
		"roles":              roles,
	})
}

// UpdateUser 更新用户
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	var req struct {
		Nickname         string `json:"nickname"`
		Avatar           string `json:"avatar"`
		Signature        string `json:"signature"`
		Phone            string `json:"phone"`
		Email            string `json:"email"`
		TwoFactorEnabled *bool  `json:"two_factor_enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	updates := make(map[string]interface{})
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Signature != "" {
		updates["signature"] = req.Signature
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.TwoFactorEnabled != nil {
		updates["two_factor_enabled"] = *req.TwoFactorEnabled
	}

	if err := h.userService.Update(c.Request.Context(), uid, updates); err != nil {
		response.Error(c, err.Error())
		return
	}

	// 返回更新后的用户
	user, _ := h.userService.GetByID(c.Request.Context(), uid)
	response.Success(c, user)
}

// SearchUsers 搜索用户
func (h *UserHandler) SearchUsers(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		response.BadRequest(c, "搜索关键词不能为空")
		return
	}

	users, err := h.userService.Search(c.Request.Context(), query, 20)
	if err != nil {
		response.Error(c, "搜索失败")
		return
	}

	response.Success(c, users)
}

// 保留原有的函数签名以兼容现有路由
func GetCurrentUser(c *gin.Context) {
	userService := service.NewUserService(database.GetDB())
	handler := NewUserHandler(userService)
	handler.GetCurrentUser(c)
}

func UpdateUser(c *gin.Context) {
	userService := service.NewUserService(database.GetDB())
	handler := NewUserHandler(userService)
	handler.UpdateUser(c)
}

func SearchUsers(c *gin.Context) {
	userService := service.NewUserService(database.GetDB())
	handler := NewUserHandler(userService)
	handler.SearchUsers(c)
}
```

- [ ] **步骤 3：验证编译**

```bash
cd qim-server && go build ./...
```

预期：编译成功

- [ ] **步骤 4：Commit**

```bash
git add qim-server/service/user_service.go qim-server/handler/user_handler.go
git commit -m "refactor(user): 重构用户模块使用分层架构

- UserService 使用 Repository
- UserHandler 使用 Service
- 保持向后兼容"
```

---

## 任务 6：创建其他核心 Repository

**文件：**
- 创建：`qim-server/repository/group_repository.go`
- 创建：`qim-server/repository/file_repository.go`
- 创建：`qim-server/repository/notification_repository.go`

- [ ] **步骤 1：实现 GroupRepository**

```go
// qim-server/repository/group_repository.go
package repository

import (
	"context"

	"qim-server/model"

	"gorm.io/gorm"
)

type groupRepository struct {
	*baseRepository[model.Group]
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{
		baseRepository: &baseRepository[model.Group]{db: db},
		db:             db,
	}
}

func (r *groupRepository) FindByConversationID(ctx context.Context, conversationID uint) (*model.Group, error) {
	var group model.Group
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Preload("Conversation").
		Preload("Conversation.Members").
		Preload("Conversation.Members.User").
		First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *groupRepository) FindByCreatorID(ctx context.Context, creatorID uint) ([]*model.Group, error) {
	var groups []*model.Group
	err := r.db.WithContext(ctx).
		Where("creator_id = ?", creatorID).
		Preload("Conversation").
		Find(&groups).Error
	return groups, err
}

func (r *groupRepository) UpdateAnnouncement(ctx context.Context, id uint, announcement string) error {
	return r.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("id = ?", id).
		Update("announcement", announcement).Error
}

func (r *groupRepository) AddMember(ctx context.Context, groupID, userID uint) error {
	// 这里需要先获取 group 的 conversation_id
	var group model.Group
	if err := r.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return err
	}
	
	// 添加会话成员
	return r.db.WithContext(ctx).Create(&model.ConversationMember{
		ConversationID: group.ConversationID,
		UserID:         userID,
		Role:           "member",
	}).Error
}

func (r *groupRepository) RemoveMember(ctx context.Context, groupID, userID uint) error {
	var group model.Group
	if err := r.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return err
	}
	
	return r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", group.ConversationID, userID).
		Delete(&model.ConversationMember{}).Error
}

func (r *groupRepository) WithTx(tx *gorm.DB) GroupRepository {
	return &groupRepository{
		baseRepository: &baseRepository[model.Group]{db: tx},
		db:             tx,
	}
}
```

- [ ] **步骤 2：实现 FileRepository**

```go
// qim-server/repository/file_repository.go
package repository

import (
	"context"

	"qim-server/model"

	"gorm.io/gorm"
)

type fileRepository struct {
	*baseRepository[model.File]
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) FileRepository {
	return &fileRepository{
		baseRepository: &baseRepository[model.File]{db: db},
		db:             db,
	}
}

func (r *fileRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.File, error) {
	var files []*model.File
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&files).Error
	return files, err
}

func (r *fileRepository) FindByFolderID(ctx context.Context, folderID *uint) ([]*model.File, error) {
	var files []*model.File
	query := r.db.WithContext(ctx)
	
	if folderID == nil {
		query = query.Where("folder_id IS NULL")
	} else {
		query = query.Where("folder_id = ?", *folderID)
	}
	
	err := query.Order("created_at DESC").Find(&files).Error
	return files, err
}

func (r *fileRepository) FindByChecksum(ctx context.Context, checksum string) (*model.File, error) {
	var file model.File
	err := r.db.WithContext(ctx).
		Where("checksum = ?", checksum).
		First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *fileRepository) UpdateStarred(ctx context.Context, id uint, starred bool) error {
	return r.db.WithContext(ctx).
		Model(&model.File{}).
		Where("id = ?", id).
		Update("is_starred", starred).Error
}

func (r *fileRepository) WithTx(tx *gorm.DB) FileRepository {
	return &fileRepository{
		baseRepository: &baseRepository[model.File]{db: tx},
		db:             tx,
	}
}
```

- [ ] **步骤 3：实现 NotificationRepository**

```go
// qim-server/repository/notification_repository.go
package repository

import (
	"context"
	"time"

	"qim-server/model"

	"gorm.io/gorm"
)

type notificationRepository struct {
	*baseRepository[model.Notification]
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{
		baseRepository: &baseRepository[model.Notification]{db: db},
		db:             db,
	}
}

func (r *notificationRepository) FindByUserID(ctx context.Context, userID uint, unreadOnly bool) ([]*model.Notification, error) {
	var notifications []*model.Notification
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	
	if unreadOnly {
		query = query.Where("read = ?", false)
	}
	
	err := query.Order("created_at DESC").Find(&notifications).Error
	return notifications, err
}

func (r *notificationRepository) MarkAsRead(ctx context.Context, id uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": now,
		}).Error
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, userID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Updates(map[string]interface{}{
			"read":    true,
			"read_at": now,
		}).Error
}

func (r *notificationRepository) CountUnread(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ? AND read = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (r *notificationRepository) WithTx(tx *gorm.DB) NotificationRepository {
	return &notificationRepository{
		baseRepository: &baseRepository[model.Notification]{db: tx},
		db:             tx,
	}
}
```

- [ ] **步骤 4：验证编译**

```bash
cd qim-server && go build ./...
```

预期：编译成功

- [ ] **步骤 5：Commit**

```bash
git add qim-server/repository/
git commit -m "feat(repository): 实现群组、文件、通知 Repository

- 实现 GroupRepository
- 实现 FileRepository
- 实现 NotificationRepository"
```

---

## 任务 7：重构 ConversationService

**文件：**
- 修改：`qim-server/service/conversation_service.go`
- 修改：`qim-server/handler/conversation_handler.go`

- [ ] **步骤 1：重构 ConversationService 使用 Repository**

```go
// qim-server/service/conversation_service.go
package service

import (
	"context"
	"errors"
	"time"

	"qim-server/model"
	"qim-server/repository"

	"gorm.io/gorm"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrConversationForbidden = errors.New("access forbidden")
	ErrNotConversationOwner = errors.New("only owner can perform this action")
)

// ConversationService 会话服务
type ConversationService struct {
	convRepo  repository.ConversationRepository
	userRepo  repository.UserRepository
	db        *gorm.DB
}

// NewConversationService 创建会话服务
func NewConversationService(db *gorm.DB) *ConversationService {
	return &ConversationService{
		convRepo: repository.NewConversationRepository(db),
		userRepo: repository.NewUserRepository(db),
		db:       db,
	}
}

// ConversationWithPin 带置顶信息的会话
type ConversationWithPin struct {
	model.Conversation
	IsPinned        bool   `json:"is_pinned"`
	IP              string `json:"ip,omitempty"`
	Status          string `json:"status,omitempty"`
	Signature       string `json:"signature,omitempty"`
	OtherMemberID   uint   `json:"other_member_id,omitempty"`
	OtherMemberName string `json:"other_member_name,omitempty"`
}

// GetConversations 获取用户会话列表
func (s *ConversationService) GetConversations(ctx context.Context, userID uint) ([]ConversationWithPin, error) {
	conversations, err := s.convRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var result []ConversationWithPin
	for _, conv := range conversations {
		var session model.ConversationSession
		s.db.WithContext(ctx).
			Where("user_id = ? AND conversation_id = ?", userID, conv.ID).
			FirstOrCreate(&session, model.ConversationSession{
				IsPinned:      false,
				LastVisitedAt: time.Now(),
			})

		convWithPin := ConversationWithPin{
			Conversation: *conv,
			IsPinned:     session.IsPinned,
		}

		// 单聊会话，获取对方信息
		if conv.Type == "single" {
			for _, m := range conv.Members {
				if m.UserID != userID && m.UserID > 0 {
					convWithPin.IP = m.User.IP
					convWithPin.Status = m.User.Status
					convWithPin.Signature = m.User.Signature
					convWithPin.OtherMemberID = m.User.ID
					convWithPin.OtherMemberName = m.User.Nickname
					break
				}
			}
		}

		result = append(result, convWithPin)
	}

	return result, nil
}

// GetConversation 获取单个会话
func (s *ConversationService) GetConversation(ctx context.Context, convID uint) (*model.Conversation, error) {
	return s.convRepo.FindByID(ctx, convID)
}

// CreateSingleConversation 创建单聊会话
func (s *ConversationService) CreateSingleConversation(ctx context.Context, userID1, userID2 uint) (*model.Conversation, error) {
	// 检查是否已存在
	existing, _ := s.convRepo.FindSingleConversation(ctx, userID1, userID2)
	if existing != nil {
		return existing, nil
	}

	var conv *model.Conversation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建会话
		conv = &model.Conversation{Type: "single"}
		if err := tx.Create(conv).Error; err != nil {
			return err
		}

		// 添加成员
		if err := tx.Create(&model.ConversationMember{
			ConversationID: conv.ID,
			UserID:         userID1,
			Role:           "member",
			JoinedAt:       time.Now(),
		}).Error; err != nil {
			return err
		}

		if err := tx.Create(&model.ConversationMember{
			ConversationID: conv.ID,
			UserID:         userID2,
			Role:           "member",
			JoinedAt:       time.Now(),
		}).Error; err != nil {
			return err
		}

		return nil
	})

	return conv, err
}

// PinConversation 置顶会话
func (s *ConversationService) PinConversation(ctx context.Context, userID, convID uint, pinned bool) error {
	var session model.ConversationSession
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND conversation_id = ?", userID, convID).
		FirstOrCreate(&session, model.ConversationSession{
			UserID:         userID,
			ConversationID: convID,
		}).Error
	if err != nil {
		return err
	}

	now := time.Now()
	return s.db.WithContext(ctx).
		Model(&session).
		Updates(map[string]interface{}{
			"is_pinned": pinned,
			"pinned_at": now,
		}).Error
}
```

- [ ] **步骤 2：重构 ConversationHandler 使用 Service**

```go
// qim-server/handler/conversation_handler.go (部分重构)
package handler

import (
	"strconv"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"
	"qim-server/service"

	"github.com/gin-gonic/gin"
)

// ConversationHandler 会话处理器
type ConversationHandler struct {
	convService *service.ConversationService
}

// NewConversationHandler 创建会话处理器
func NewConversationHandler(convService *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{
		convService: convService,
	}
}

// GetConversations 获取会话列表
func (h *ConversationHandler) GetConversations(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	conversations, err := h.convService.GetConversations(c.Request.Context(), uid)
	if err != nil {
		response.Error(c, "获取会话列表失败")
		return
	}

	response.Success(c, conversations)
}

// GetConversation 获取单个会话
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	convID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的会话ID")
		return
	}

	conv, err := h.convService.GetConversation(c.Request.Context(), uint(convID))
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	response.Success(c, conv)
}

// PinConversation 置顶会话
func (h *ConversationHandler) PinConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	convID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的会话ID")
		return
	}

	var req struct {
		Pinned bool `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.convService.PinConversation(c.Request.Context(), uid, uint(convID), req.Pinned); err != nil {
		response.Error(c, "操作失败")
		return
	}

	response.Success(c, nil)
}

// 保留原有函数签名以兼容现有路由
func GetConversations(c *gin.Context) {
	convService := service.NewConversationService(database.GetDB())
	handler := NewConversationHandler(convService)
	handler.GetConversations(c)
}

func GetConversation(c *gin.Context) {
	convService := service.NewConversationService(database.GetDB())
	handler := NewConversationHandler(convService)
	handler.GetConversation(c)
}

func PinConversation(c *gin.Context) {
	convService := service.NewConversationService(database.GetDB())
	handler := NewConversationHandler(convService)
	handler.PinConversation(c)
}
```

- [ ] **步骤 3：验证编译**

```bash
cd qim-server && go build ./...
```

预期：编译成功

- [ ] **步骤 4：Commit**

```bash
git add qim-server/service/conversation_service.go qim-server/handler/conversation_handler.go
git commit -m "refactor(conversation): 重构会话模块使用分层架构

- ConversationService 使用 Repository
- ConversationHandler 使用 Service
- 保持向后兼容"
```

---

## 任务 8：更新依赖注入

**文件：**
- 修改：`qim-server/app/init.go`
- 修改：`qim-server/app/routes.go`

- [ ] **步骤 1：创建服务容器**

```go
// qim-server/app/container.go
package app

import (
	"qim-server/repository"
	"qim-server/service"

	"gorm.io/gorm"
)

// Container 服务容器
type Container struct {
	DB *gorm.DB

	// Repositories
	UserRepo         repository.UserRepository
	ConversationRepo repository.ConversationRepository
	MessageRepo      repository.MessageRepository
	GroupRepo        repository.GroupRepository
	FileRepo         repository.FileRepository
	NotificationRepo repository.NotificationRepository

	// Services
	UserService         *service.UserService
	ConversationService *service.ConversationService
	MessageService      *service.MessageService
}

// NewContainer 创建服务容器
func NewContainer(db *gorm.DB) *Container {
	c := &Container{DB: db}

	// 初始化 Repositories
	c.UserRepo = repository.NewUserRepository(db)
	c.ConversationRepo = repository.NewConversationRepository(db)
	c.MessageRepo = repository.NewMessageRepository(db)
	c.GroupRepo = repository.NewGroupRepository(db)
	c.FileRepo = repository.NewFileRepository(db)
	c.NotificationRepo = repository.NewNotificationRepository(db)

	// 初始化 Services
	c.UserService = service.NewUserService(db)
	c.ConversationService = service.NewConversationService(db)
	c.MessageService = service.NewMessageService(db)

	return c
}
```

- [ ] **步骤 2：更新初始化代码**

```go
// qim-server/app/init.go
package app

import (
	"qim-server/database"

	"gorm.io/gorm"
)

var (
	DB        *gorm.DB
	Container *Container
)

// Init 初始化应用
func Init() error {
	// 初始化数据库
	var err error
	DB, err = database.InitDB()
	if err != nil {
		return err
	}

	// 初始化服务容器
	Container = NewContainer(DB)

	return nil
}
```

- [ ] **步骤 3：验证编译**

```bash
cd qim-server && go build ./...
```

预期：编译成功

- [ ] **步骤 4：Commit**

```bash
git add qim-server/app/
git commit -m "feat(app): 添加服务容器和依赖注入

- 创建 Container 管理服务和仓库
- 统一初始化流程"
```

---

## 任务 9：编写集成测试

**文件：**
- 创建：`qim-server/tests/integration/user_test.go`
- 创建：`qim-server/tests/integration/conversation_test.go`

- [ ] **步骤 1：编写用户模块集成测试**

```go
// qim-server/tests/integration/user_test.go
package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"qim-server/app"
	"qim-server/handler"
	"qim-server/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestApp(t *testing.T) (*gin.Engine, *app.Container) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	container := app.NewContainer(db)
	router := gin.New()

	// 注册路由
	userHandler := handler.NewUserHandler(container.UserService)
	router.GET("/api/v1/users/me", userHandler.GetCurrentUser)
	router.PUT("/api/v1/users/me", userHandler.UpdateUser)

	return router, container
}

func TestUserAPI_GetCurrentUser(t *testing.T) {
	router, container := setupTestApp(t)

	// 创建测试用户
	user := container.UserService.Create(context.Background(), &model.User{
		Username: "testuser",
		Nickname: "Test User",
	})

	// 创建请求
	req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	req.Header.Set("user_id", strconv.FormatUint(uint64(user.ID), 10))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "testuser")
}
```

- [ ] **步骤 2：运行测试**

```bash
cd qim-server && go test ./tests/integration/... -v
```

预期：PASS

- [ ] **步骤 3：Commit**

```bash
git add qim-server/tests/
git commit -m "test: 添加集成测试

- 用户模块集成测试
- 会话模块集成测试"
```

---

## 任务 10：更新文档

**文件：**
- 创建：`qim-server/docs/ARCHITECTURE.md`

- [ ] **步骤 1：编写架构文档**

```markdown
# qim-server 架构文档

## 分层架构

项目采用经典的三层架构：

### Handler 层
- 职责：处理 HTTP 请求，参数验证，响应格式化
- 位置：`handler/` 目录
- 规则：
  - 不直接操作数据库
  - 只调用 Service 方法
  - 处理 HTTP 相关逻辑

### Service 层
- 职责：业务逻辑处理，事务管理
- 位置：`service/` 目录
- 规则：
  - 不直接操作数据库
  - 只调用 Repository 方法
  - 处理业务逻辑

### Repository 层
- 职责：数据访问封装
- 位置：`repository/` 目录
- 规则：
  - 只处理数据库操作
  - 不包含业务逻辑
  - 提供通用 CRUD 方法

## 依赖注入

使用 Container 模式管理依赖：

```go
container := app.NewContainer(db)
userHandler := handler.NewUserHandler(container.UserService)
```

## 测试策略

- 单元测试：测试 Repository 和 Service
- 集成测试：测试 Handler 和完整流程
- 使用 SQLite 内存数据库进行测试
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/docs/
git commit -m "docs: 添加架构文档

- 分层架构说明
- 依赖注入说明
- 测试策略说明"
```

---

## 后续任务（按优先级继续）

1. 重构 `notification_handler.go` (17 处调用)
2. 重构 `app_handler.go` (16 处调用)
3. 重构 `file_handler.go` (14 处调用)
4. 重构 `admin_handler.go` (13 处调用)
5. 重构 `message_handler.go` (12 处调用)
6. 重构 `group_handler.go` (11 处调用)
7. 重构 `realtime_handler.go` (10 处调用)
8. 重构 `sensitive_word_handler.go` (9 处调用)
9. 重构其他 Handler

每个 Handler 的重构遵循相同的模式：
1. 创建对应的 Repository（如果不存在）
2. 重构 Service 使用 Repository
3. 重构 Handler 使用 Service
4. 编写测试
5. 验证编译
6. Commit

---

## 验收标准

- [ ] 所有 Handler 不再直接调用 `database.GetDB()`
- [ ] 所有 Service 使用 Repository 进行数据访问
- [ ] 所有 Repository 有对应的单元测试
- [ ] 所有 Handler 有对应的集成测试
- [ ] 编译无错误
- [ ] 测试覆盖率 > 80%
