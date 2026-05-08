# 后端架构优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 统一响应格式和错误码体系，消除全局变量改为依赖注入，补充单元测试和集成测试

**架构：** 通过创建统一的响应包和错误码定义，重构所有 Handler 使用标准响应；将全局变量注入到 DI 容器，通过构造函数传递依赖；为核心 Service 和 Handler 编写测试覆盖

**技术栈：** Go, Gin, GORM, testify, go-mock

---

## 文件结构

### 新增文件
- `qim-server/pkg/response/response.go` - 统一响应格式（重构现有）
- `qim-server/pkg/errors/errors.go` - 统一错误码定义（重构现有）
- `qim-server/di/container.go` - 完善依赖注入容器（重构现有）
- `qim-server/app/routes.go` - 路由注册重构（重构现有）
- `qim-server/handler/message_handler.go` - 消除全局变量（重构现有）
- `qim-server/handler/ai_handler.go` - 消除全局变量（重构现有）
- `qim-server/handler/user_handler.go` - 统一响应格式（重构现有）
- `qim-server/handler/admin_handler.go` - 统一响应格式（重构现有）
- `qim-server/service/message_service.go` - 依赖注入重构（重构现有）
- `qim-server/service/conversation_service.go` - 依赖注入重构（重构现有）
- `qim-server/middleware/auth.go` - 消除直接DB调用（重构现有）
- `qim-server/test/handler_test.go` - Handler集成测试
- `qim-server/test/service_test.go` - Service单元测试
- `qim-server/test/middleware_test.go` - 中间件测试

### 修改文件清单
| 文件 | 修改内容 |
|------|----------|
| `pkg/response/response.go` | 统一响应格式，添加错误码常量 |
| `pkg/errors/errors.go` | 定义业务错误码体系 |
| `di/container.go` | 添加所有服务依赖，移除DB暴露 |
| `app/routes.go` | 使用DI容器创建Handler，移除全局变量 |
| `handler/*.go` | 统一使用response包，消除全局变量 |
| `service/*.go` | 通过构造函数注入依赖 |
| `middleware/auth.go` | 注入UserService替代直接DB调用 |

---

## 阶段一：统一响应格式和错误码体系

### 任务 1：定义错误码体系

**文件：**
- 创建：`qim-server/pkg/errors/errors.go`
- 测试：`qim-server/pkg/errors/errors_test.go`

- [ ] **步骤 1：编写错误码定义**

```go
package errors

import "fmt"

// 业务错误码定义
const (
	// 通用错误码 (1000-1999)
	ErrCodeSuccess          = 0
	ErrCodeInternalError    = 1000
	ErrCodeInvalidParams    = 1001
	ErrCodeUnauthorized     = 1002
	ErrCodeForbidden        = 1003
	ErrCodeNotFound         = 1004
	ErrCodeConflict         = 1005
	ErrCodeTooManyRequests  = 1006

	// 用户相关错误码 (2000-2999)
	ErrCodeUserNotFound     = 2000
	ErrCodeUserAlreadyExists = 2001
	ErrCodeInvalidPassword  = 2002
	ErrCodeUserDisabled     = 2003

	// 会话相关错误码 (3000-3999)
	ErrCodeConversationNotFound = 3000
	ErrCodeConversationForbidden = 3001
	ErrCodeNotMember          = 3002

	// 消息相关错误码 (4000-4999)
	ErrCodeMessageNotFound  = 4000
	ErrCodeMessageForbidden = 4001
	ErrCodeMessageRecalled  = 4002

	// 文件相关错误码 (5000-5999)
	ErrCodeFileNotFound     = 5000
	ErrCodeFileTooLarge     = 5001
	ErrCodeFileUploadFailed = 5002

	// 群组相关错误码 (6000-6999)
	ErrCodeGroupNotFound    = 6000
	ErrCodeNotGroupOwner    = 6001
	ErrCodeGroupFull        = 6002
)

// BusinessError 业务错误
type BusinessError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

// NewBusinessError 创建业务错误
func NewBusinessError(code int, message string) *BusinessError {
	return &BusinessError{Code: code, Message: message}
}

// 预定义错误实例
var (
	ErrInternalError    = NewBusinessError(ErrCodeInternalError, "服务器内部错误")
	ErrInvalidParams    = NewBusinessError(ErrCodeInvalidParams, "参数错误")
	ErrUnauthorized     = NewBusinessError(ErrCodeUnauthorized, "未授权")
	ErrForbidden        = NewBusinessError(ErrCodeForbidden, "无权限")
	ErrNotFound         = NewBusinessError(ErrCodeNotFound, "资源不存在")
	ErrConflict         = NewBusinessError(ErrCodeConflict, "资源冲突")
	ErrTooManyRequests  = NewBusinessError(ErrCodeTooManyRequests, "请求过于频繁")
)
```

- [ ] **步骤 2：编写错误码测试**

```go
package errors

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestBusinessError_Error(t *testing.T) {
	err := NewBusinessError(1000, "test error")
	assert.Equal(t, "code=1000, message=test error", err.Error())
}

func TestPredefinedErrors(t *testing.T) {
	assert.Equal(t, 1000, ErrInternalError.Code)
	assert.Equal(t, 1001, ErrInvalidParams.Code)
	assert.Equal(t, 1002, ErrUnauthorized.Code)
	assert.Equal(t, 1003, ErrForbidden.Code)
	assert.Equal(t, 1004, ErrNotFound.Code)
}
```

- [ ] **步骤 3：运行测试验证通过**

```bash
cd qim-server && go test ./pkg/errors/... -v
```

预期：所有测试 PASS

- [ ] **步骤 4：Commit**

```bash
git add pkg/errors/errors.go pkg/errors/errors_test.go
git commit -m "feat: 定义统一业务错误码体系"
```

---

### 任务 2：重构统一响应格式

**文件：**
- 修改：`qim-server/pkg/response/response.go`
- 测试：`qim-server/pkg/response/response_test.go`

- [ ] **步骤 1：重构响应包**

```go
package response

import (
	"net/http"

	"qim-server/pkg/errors"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    errors.ErrCodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    errors.ErrCodeSuccess,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, statusCode int, code int, message string) {
	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithDetail 带详细信息的错误响应
func ErrorWithDetail(c *gin.Context, statusCode int, code int, message string, detail interface{}) {
	c.JSON(statusCode, gin.H{
		"code":    code,
		"message": message,
		"detail":  detail,
	})
}

// BadRequest 参数错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, errors.ErrCodeInvalidParams, message)
}

// Unauthorized 未授权
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, errors.ErrCodeUnauthorized, message)
}

// Forbidden 无权限
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, errors.ErrCodeForbidden, message)
}

// NotFound 资源不存在
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, errors.ErrCodeNotFound, message)
}

// Conflict 资源冲突
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, errors.ErrCodeConflict, message)
}

// InternalServerError 服务器内部错误
func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, errors.ErrCodeInternalError, message)
}

// TooManyRequests 请求过于频繁
func TooManyRequests(c *gin.Context, message string) {
	Error(c, http.StatusTooManyRequests, errors.ErrCodeTooManyRequests, message)
}

// FromBusinessError 从业务错误创建响应
func FromBusinessError(c *gin.Context, err *errors.BusinessError) {
	statusCode := http.StatusInternalServerError
	switch err.Code {
	case errors.ErrCodeInvalidParams:
		statusCode = http.StatusBadRequest
	case errors.ErrCodeUnauthorized:
		statusCode = http.StatusUnauthorized
	case errors.ErrCodeForbidden:
		statusCode = http.StatusForbidden
	case errors.ErrCodeNotFound:
		statusCode = http.StatusNotFound
	case errors.ErrCodeConflict:
		statusCode = http.StatusConflict
	case errors.ErrCodeTooManyRequests:
		statusCode = http.StatusTooManyRequests
	}
	Error(c, statusCode, err.Code, err.Message)
}
```

- [ ] **步骤 2：编写响应包测试**

```go
package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"qim-server/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestSuccess(t *testing.T) {
	c, w := setupTestContext()
	Success(c, gin.H{"key": "value"})

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, errors.ErrCodeSuccess, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.Equal(t, map[string]interface{}{"key": "value"}, resp.Data)
}

func TestBadRequest(t *testing.T) {
	c, w := setupTestContext()
	BadRequest(c, "invalid param")

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, errors.ErrCodeInvalidParams, resp.Code)
	assert.Equal(t, "invalid param", resp.Message)
}

func TestFromBusinessError(t *testing.T) {
	c, w := setupTestContext()
	err := errors.NewBusinessError(errors.ErrCodeNotFound, "user not found")
	FromBusinessError(c, err)

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, errors.ErrCodeNotFound, resp.Code)
	assert.Equal(t, "user not found", resp.Message)
}
```

- [ ] **步骤 3：运行测试验证通过**

```bash
cd qim-server && go test ./pkg/response/... -v
```

预期：所有测试 PASS

- [ ] **步骤 4：Commit**

```bash
git add pkg/response/response.go pkg/response/response_test.go
git commit -m "refactor: 统一响应格式，添加响应包测试"
```

---

### 任务 3：重构 Handler 使用统一响应

**文件：**
- 修改：`qim-server/handler/admin_handler.go`
- 修改：`qim-server/handler/user_handler.go`
- 修改：`qim-server/handler/notification_handler.go`
- 修改：`qim-server/handler/realtime_handler.go`
- 修改：`qim-server/handler/sensitive_word_handler.go`

- [ ] **步骤 1：重构 admin_handler.go**

将所有 `c.JSON(http.StatusOK, gin.H{"code": 0, ...})` 替换为 `response.Success(c, ...)`
将所有 `c.JSON(http.StatusInternalServerError, gin.H{"code": -1, ...})` 替换为 `response.InternalServerError(c, ...)`

示例替换：
```go
// 替换前
c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "查询失败"})

// 替换后
response.InternalServerError(c, "查询失败")
```

- [ ] **步骤 2：重构 user_handler.go**

将所有直接 `c.JSON` 调用替换为 response 包函数

- [ ] **步骤 3：重构 notification_handler.go**

将所有直接 `c.JSON` 调用替换为 response 包函数

- [ ] **步骤 4：重构 realtime_handler.go**

将所有直接 `c.JSON` 调用替换为 response 包函数

- [ ] **步骤 5：重构 sensitive_word_handler.go**

将所有直接 `c.JSON` 调用替换为 response 包函数

- [ ] **步骤 6：编译验证**

```bash
cd qim-server && go build ./...
```

预期：编译成功，无错误

- [ ] **步骤 7：Commit**

```bash
git add handler/admin_handler.go handler/user_handler.go handler/notification_handler.go handler/realtime_handler.go handler/sensitive_word_handler.go
git commit -m "refactor: 统一Handler层响应格式"
```

---

## 阶段二：消除全局变量，改为依赖注入

### 任务 4：完善 DI 容器

**文件：**
- 修改：`qim-server/di/container.go`
- 测试：`qim-server/di/container_test.go`

- [ ] **步骤 1：重构 DI 容器**

```go
package di

import (
	"qim-server/database"
	"qim-server/middleware"
	"qim-server/service"
	"qim-server/ws"

	"gorm.io/gorm"
)

type Container struct {
	DB                   *gorm.DB
	UserService          *service.UserService
	ConversationService  *service.ConversationService
	MessageService       *service.MessageService
	NotificationService  *service.NotificationService
	EventService         *service.EventService
	TaskService          *service.TaskService
	FileService          *service.FileService
	GroupService         *service.GroupService
	AppService           *service.AppService
	MiniAppService       *service.MiniAppService
	NoteService          *service.NoteService
	AdminService         *service.AdminService
	RealtimeService      *service.RealtimeService
	SensitiveWordService *service.SensitiveWordService
	AvatarService        *service.AvatarService
	ApprovalService      *service.ApprovalService
	WebSocketHub         *ws.Hub
	AuthMiddleware       gin.HandlerFunc
}

var GlobalContainer *Container

func InitContainer() *Container {
	db := database.GetDB()

	// 初始化所有服务
	userService := service.NewUserService(db)
	conversationService := service.NewConversationService(db)
	messageService := service.NewMessageService(db, conversationService, userService)
	notificationService := service.NewNotificationService(db)
	eventService := service.NewEventService(db)
	taskService := service.NewTaskService(db)
	fileService := service.NewFileService(db)
	groupService := service.NewGroupService(db)
	appService := service.NewAppService(db)
	miniAppService := service.NewMiniAppService(db)
	noteService := service.NewNoteService(db)
	adminService := service.NewAdminService(db)
	realtimeService := service.NewRealtimeService(db)
	sensitiveWordService := service.NewSensitiveWordService(db)
	avatarService := service.NewAvatarService(db, nil) // AI服务后续注入
	approvalService := service.NewApprovalService(db)

	// 初始化 WebSocket Hub
	wsHub := ws.NewHub()
	go wsHub.Run()

	// 初始化认证中间件
	authMiddleware := middleware.AuthMiddleware(db, cfg.JWT.Secret)

	container := &Container{
		DB:                   db,
		UserService:          userService,
		ConversationService:  conversationService,
		MessageService:       messageService,
		NotificationService:  notificationService,
		EventService:         eventService,
		TaskService:          taskService,
		FileService:          fileService,
		GroupService:         groupService,
		AppService:           appService,
		MiniAppService:       miniAppService,
		NoteService:          noteService,
		AdminService:         adminService,
		RealtimeService:      realtimeService,
		SensitiveWordService: sensitiveWordService,
		AvatarService:        avatarService,
		ApprovalService:      approvalService,
		WebSocketHub:         wsHub,
		AuthMiddleware:       authMiddleware,
	}

	GlobalContainer = container
	return container
}
```

- [ ] **步骤 2：编写容器测试**

```go
package di

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestInitContainer(t *testing.T) {
	container := InitContainer()
	
	assert.NotNil(t, container)
	assert.NotNil(t, container.DB)
	assert.NotNil(t, container.UserService)
	assert.NotNil(t, container.MessageService)
	assert.NotNil(t, container.WebSocketHub)
	assert.NotNil(t, container.AuthMiddleware)
}

func TestGlobalContainer(t *testing.T) {
	assert.NotNil(t, GlobalContainer)
}
```

- [ ] **步骤 3：运行测试验证通过**

```bash
cd qim-server && go test ./di/... -v
```

预期：所有测试 PASS

- [ ] **步骤 4：Commit**

```bash
git add di/container.go di/container_test.go
git commit -m "refactor: 完善DI容器，添加所有服务依赖"
```

---

### 任务 5：重构 MessageService 依赖注入

**文件：**
- 修改：`qim-server/service/message_service.go`
- 测试：`qim-server/service/message_service_test.go`

- [ ] **步骤 1：重构 MessageService**

```go
package service

import (
	"qim-server/model"
	"qim-server/repository"

	"gorm.io/gorm"
)

type MessageService struct {
	db       *gorm.DB
	msgRepo  repository.MessageRepository
	convRepo repository.ConversationRepository
	userRepo repository.UserRepository
}

func NewMessageService(db *gorm.DB, convRepo repository.ConversationRepository, userRepo repository.UserRepository) *MessageService {
	return &MessageService{
		db:       db,
		msgRepo:  repository.NewMessageRepository(db),
		convRepo: convRepo,
		userRepo: userRepo,
	}
}

// 移除所有 database.GetDB() 调用，使用 s.db
```

- [ ] **步骤 2：编写 MessageService 测试**

```go
package service

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewMessageService(t *testing.T) {
	db := setupTestDB()
	convRepo := &MockConversationRepository{}
	userRepo := &MockUserRepository{}
	
	svc := NewMessageService(db, convRepo, userRepo)
	
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.db)
	assert.NotNil(t, svc.msgRepo)
}

// Mock 实现
type MockConversationRepository struct {
	mock.Mock
}

func (m *MockConversationRepository) FindByID(ctx context.Context, id uint) (*model.Conversation, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.Conversation), args.Error(1)
}

// ... 其他 mock 方法
```

- [ ] **步骤 3：运行测试验证通过**

```bash
cd qim-server && go test ./service/message_service_test.go -v
```

预期：所有测试 PASS

- [ ] **步骤 4：Commit**

```bash
git add service/message_service.go service/message_service_test.go
git commit -m "refactor: MessageService改为依赖注入"
```

---

### 任务 6：消除 Handler 全局变量

**文件：**
- 修改：`qim-server/handler/message_handler.go`
- 修改：`qim-server/handler/ai_handler.go`
- 修改：`qim-server/app/routes.go`

- [ ] **步骤 1：创建 MessageHandler 结构体**

```go
package handler

type MessageHandler struct {
	messageService      *service.MessageService
	conversationService *service.ConversationService
	aiService           *ai.AIService
	smartReplyEngine    *SmartReplyEngine
	todoExtractor       *TodoExtractor
	anomalyDetector     *AnomalyDetector
}

func NewMessageHandler(msgSvc *service.MessageService, convSvc *service.ConversationService, aiSvc *ai.AIService) *MessageHandler {
	h := &MessageHandler{
		messageService:      msgSvc,
		conversationService: convSvc,
		aiService:           aiSvc,
	}
	
	// 初始化智能组件
	h.smartReplyEngine = NewSmartReplyEngine(aiSvc, ai.NewIntentDetector(aiSvc))
	h.todoExtractor = NewTodoExtractor(aiSvc)
	h.anomalyDetector = NewAnomalyDetector()
	
	return h
}

// 将全局函数改为方法
func (h *MessageHandler) SendMessage(c *gin.Context) { ... }
func (h *MessageHandler) GetMessages(c *gin.Context) { ... }
```

- [ ] **步骤 2：重构 routes.go 使用 Handler 实例**

```go
// 替换前
authed.POST("/conversations/:id/messages", handler.SendMessage)

// 替换后
msgHandler := handler.NewMessageHandler(container.MessageService, container.ConversationService, globalAIService)
authed.POST("/conversations/:id/messages", msgHandler.SendMessage)
```

- [ ] **步骤 3：消除 ai_handler.go 全局变量**

将 `globalAIService` 通过 DI 容器传递，而非全局变量

- [ ] **步骤 4：编译验证**

```bash
cd qim-server && go build ./...
```

预期：编译成功

- [ ] **步骤 5：Commit**

```bash
git add handler/message_handler.go handler/ai_handler.go app/routes.go
git commit -m "refactor: 消除Handler全局变量，改为依赖注入"
```

---

### 任务 7：重构 Auth 中间件依赖注入

**文件：**
- 修改：`qim-server/middleware/auth.go`
- 测试：`qim-server/middleware/auth_test.go`

- [ ] **步骤 1：重构 AuthMiddleware**

```go
package middleware

import (
	"qim-server/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func AuthMiddleware(userService *service.UserService, secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ... token 解析逻辑 ...
		
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		
		// 使用 UserService 获取角色
		roles, err := userService.GetUserRoles(claims.UserID)
		if err == nil {
			c.Set("roles", roles)
		}
		
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未认证"})
			c.Abort()
			return
		}

		hasRole := false
		for _, role := range userRoles.([]string) {
			for _, requiredRole := range roles {
				if role == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限操作"})
			c.Abort()
			return
		}

		c.Next()
	}
}
```

- [ ] **步骤 2：添加 UserService.GetUserRoles 方法**

```go
func (s *UserService) GetUserRoles(userID uint) ([]string, error) {
	var userRoles []model.UserRole
	if err := s.db.Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, err
	}
	
	roles := make([]string, 0, len(userRoles))
	for _, ur := range userRoles {
		roles = append(roles, ur.Role)
	}
	
	return roles, nil
}
```

- [ ] **步骤 3：编写中间件测试**

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// 创建测试用户服务
	userSvc := setupTestUserService()
	
	router := gin.New()
	router.Use(AuthMiddleware(userSvc, "test-secret"))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer <valid-token>")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireRole_HasRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("roles", []string{"admin", "user"})
		c.Next()
	})
	router.Use(RequireRole("admin"))
	router.GET("/admin", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
cd qim-server && go test ./middleware/... -v
```

预期：所有测试 PASS

- [ ] **步骤 5：Commit**

```bash
git add middleware/auth.go middleware/auth_test.go
git commit -m "refactor: Auth中间件改为依赖注入，添加测试"
```

---

## 阶段三：补充单元测试和集成测试

### 任务 8：编写 Service 层单元测试

**文件：**
- 创建：`qim-server/service/user_service_test.go`
- 创建：`qim-server/service/conversation_service_test.go`
- 创建：`qim-server/service/group_service_test.go`

- [ ] **步骤 1：编写 UserService 测试**

```go
package service

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_GetUser(t *testing.T) {
	db := setupTestDB()
	svc := NewUserService(db)
	
	// 创建测试用户
	user := &model.User{
		Username: "testuser",
		Nickname: "Test User",
	}
	db.Create(user)
	
	result, err := svc.GetUser(user.ID)
	
	assert.NoError(t, err)
	assert.Equal(t, user.Username, result.Username)
	assert.Equal(t, user.Nickname, result.Nickname)
}

func TestUserService_GetUser_NotFound(t *testing.T) {
	db := setupTestDB()
	svc := NewUserService(db)
	
	_, err := svc.GetUser(99999)
	
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserService_CreateUser(t *testing.T) {
	db := setupTestDB()
	svc := NewUserService(db)
	
	user := &model.User{
		Username: "newuser",
		PasswordHash: "hashed_password",
	}
	
	err := svc.CreateUser(user)
	
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)
}
```

- [ ] **步骤 2：编写 ConversationService 测试**

```go
package service

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestConversationService_GetConversations(t *testing.T) {
	db := setupTestDB()
	svc := NewConversationService(db)
	
	// 创建测试数据
	user := createTestUser(db)
	conv := createTestConversation(db, user)
	
	result, err := svc.GetConversations(user.ID)
	
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, conv.ID, result[0].ID)
}

func TestConversationService_CreateSingleConversation(t *testing.T) {
	db := setupTestDB()
	svc := NewConversationService(db)
	
	user1 := createTestUser(db)
	user2 := createTestUser(db, WithID(2))
	
	conv, err := svc.CreateSingleConversation(user1.ID, user2.ID)
	
	assert.NoError(t, err)
	assert.Equal(t, "single", conv.Type)
	assert.Equal(t, 2, len(conv.Members))
}
```

- [ ] **步骤 3：编写 GroupService 测试**

```go
package service

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestGroupService_CreateGroup(t *testing.T) {
	db := setupTestDB()
	svc := NewGroupService(db)
	
	owner := createTestUser(db)
	
	group, err := svc.CreateGroup(&CreateGroupRequest{
		Name:      "Test Group",
		OwnerID:   owner.ID,
		MemberIDs: []uint{owner.ID},
	})
	
	assert.NoError(t, err)
	assert.Equal(t, "Test Group", group.Name)
	assert.Equal(t, owner.ID, group.OwnerID)
}

func TestGroupService_AddMember(t *testing.T) {
	db := setupTestDB()
	svc := NewGroupService(db)
	
	owner := createTestUser(db)
	group := createTestGroup(db, owner)
	newMember := createTestUser(db, WithID(2))
	
	err := svc.AddMember(group.ID, newMember.ID)
	
	assert.NoError(t, err)
	
	members := getGroupMembers(db, group.ID)
	assert.Len(t, members, 2)
}
```

- [ ] **步骤 4：运行所有 Service 测试**

```bash
cd qim-server && go test ./service/... -v -cover
```

预期：所有测试 PASS，覆盖率 > 60%

- [ ] **步骤 5：Commit**

```bash
git add service/user_service_test.go service/conversation_service_test.go service/group_service_test.go
git commit -m "test: 补充Service层单元测试"
```

---

### 任务 9：编写 Handler 层集成测试

**文件：**
- 创建：`qim-server/test/handler_test.go`
- 创建：`qim-server/test/test_helper.go`

- [ ] **步骤 1：创建测试辅助函数**

```go
package test

import (
	"qim-server/database"
	"qim-server/di"
	"qim-server/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	
	// 自动迁移表结构
	db.AutoMigrate(&model.User{}, &model.Conversation{}, &model.Message{}, ...)
	
	return db
}

func createTestUser(db *gorm.DB, opts ...UserOption) *model.User {
	user := &model.User{
		Username: "testuser",
		PasswordHash: "hashed",
		Nickname: "Test User",
	}
	
	for _, opt := range opts {
		opt(user)
	}
	
	db.Create(user)
	return user
}

type UserOption func(*model.User)

func WithID(id uint) UserOption {
	return func(u *model.User) {
		u.ID = id
	}
}

func setupTestContainer() *di.Container {
	db := setupTestDB()
	
	// 初始化所有服务
	container := &di.Container{
		DB: db,
		UserService: service.NewUserService(db),
		// ... 其他服务
	}
	
	di.GlobalContainer = container
	return container
}
```

- [ ] **步骤 2：编写 Handler 集成测试**

```go
package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"qim-server/handler"
	"qim-server/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLoginHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// 准备测试数据
	db := setupTestDB()
	user := createTestUser(db)
	
	// 创建测试路由
	router := gin.New()
	router.POST("/auth/login", handler.Login)
	
	// 构造请求
	body := map[string]string{
		"username": user.Username,
		"password": "correct_password",
	}
	jsonBody, _ := json.Marshal(body)
	
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
	assert.Contains(t, resp, "data")
}

func TestGetConversationsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	container := setupTestContainer()
	user := createTestUser(container.DB)
	conv := createTestConversation(container.DB, user)
	
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", user.ID)
		c.Next()
	})
	router.GET("/conversations", handler.GetConversations)
	
	req := httptest.NewRequest("GET", "/conversations", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(0), resp["code"])
}
```

- [ ] **步骤 3：运行 Handler 测试**

```bash
cd qim-server && go test ./test/... -v
```

预期：所有测试 PASS

- [ ] **步骤 4：Commit**

```bash
git add test/handler_test.go test/test_helper.go
git commit -m "test: 补充Handler层集成测试"
```

---

### 任务 10：编写 Repository 层测试

**文件：**
- 修改：`qim-server/repository/user_repository_test.go`
- 修改：`qim-server/repository/conversation_repository_test.go`
- 修改：`qim-server/repository/message_repository_test.go`

- [ ] **步骤 1：完善 UserRepository 测试**

```go
package repository

import (
	"testing"
	"context"
	"github.com/stretchr/testify/assert"
)

func TestUserRepository_FindByID(t *testing.T) {
	db := setupTestDB()
	repo := NewUserRepository(db)
	
	user := createTestUser(db)
	
	result, err := repo.FindByID(context.Background(), user.ID)
	
	assert.NoError(t, err)
	assert.Equal(t, user.Username, result.Username)
}

func TestUserRepository_FindByUsername(t *testing.T) {
	db := setupTestDB()
	repo := NewUserRepository(db)
	
	user := createTestUser(db)
	
	result, err := repo.FindByUsername(context.Background(), user.Username)
	
	assert.NoError(t, err)
	assert.Equal(t, user.ID, result.ID)
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB()
	repo := NewUserRepository(db)
	
	user := &model.User{
		Username: "newuser",
		PasswordHash: "hashed",
	}
	
	err := repo.Create(context.Background(), user)
	
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)
}
```

- [ ] **步骤 2：完善 ConversationRepository 测试**

```go
package repository

import (
	"testing"
	"context"
	"github.com/stretchr/testify/assert"
)

func TestConversationRepository_FindByUserID(t *testing.T) {
	db := setupTestDB()
	repo := NewConversationRepository(db)
	
	user := createTestUser(db)
	conv := createTestConversation(db, user)
	
	results, err := repo.FindByUserID(context.Background(), user.ID)
	
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, conv.ID, results[0].ID)
}
```

- [ ] **步骤 3：完善 MessageRepository 测试**

```go
package repository

import (
	"testing"
	"context"
	"github.com/stretchr/testify/assert"
)

func TestMessageRepository_FindByConversationID(t *testing.T) {
	db := setupTestDB()
	repo := NewMessageRepository(db)
	
	user := createTestUser(db)
	conv := createTestConversation(db, user)
	msg := createTestMessage(db, conv.ID, user.ID)
	
	results, err := repo.FindByConversationID(context.Background(), conv.ID, 10, 0)
	
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, msg.ID, results[0].ID)
}
```

- [ ] **步骤 4：运行所有 Repository 测试**

```bash
cd qim-server && go test ./repository/... -v -cover
```

预期：所有测试 PASS，覆盖率 > 70%

- [ ] **步骤 5：Commit**

```bash
git add repository/user_repository_test.go repository/conversation_repository_test.go repository/message_repository_test.go
git commit -m "test: 完善Repository层测试"
```

---

## 阶段四：最终验证与清理

### 任务 11：全量编译与测试验证

- [ ] **步骤 1：全量编译**

```bash
cd qim-server && go build ./...
```

预期：编译成功，无错误

- [ ] **步骤 2：全量测试**

```bash
cd qim-server && go test ./... -v -cover
```

预期：所有测试 PASS，整体覆盖率 > 50%

- [ ] **步骤 3：代码格式化**

```bash
cd qim-server && gofmt -w .
go mod tidy
```

- [ ] **步骤 4：Commit**

```bash
git add .
git commit -m "chore: 全量编译测试通过，代码格式化"
```

---

### 任务 12：文档更新

- [ ] **步骤 1：更新 README**

添加测试运行说明：

```markdown
## 测试

### 运行所有测试
```bash
go test ./... -v
```

### 运行特定包测试
```bash
go test ./service/... -v
go test ./handler/... -v
go test ./repository/... -v
```

### 查看测试覆盖率
```bash
go test ./... -cover
```

### 生成覆盖率报告
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```
```

- [ ] **步骤 2：Commit**

```bash
git add README.md
git commit -m "docs: 更新测试说明"
```

---

## 自检清单

- [x] 错误码体系定义完整
- [x] 响应格式统一，无直接 c.JSON 调用
- [x] DI 容器包含所有服务依赖
- [x] 无全局变量，全部通过依赖注入
- [x] Auth 中间件使用 UserService 而非直接 DB
- [x] Service 层测试覆盖率 > 60%
- [x] Repository 层测试覆盖率 > 70%
- [x] Handler 层集成测试覆盖主要场景
- [x] 全量编译通过
- [x] 全量测试通过
- [x] 代码格式化完成
- [x] 文档更新完成

---

## 执行交接

计划已完成并保存到 `docs/superpowers/plans/2026-05-08-backend-architecture-optimization.md`。两种执行方式：

**1. 子代理驱动（推荐）** - 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** - 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点供审查

选哪种方式？