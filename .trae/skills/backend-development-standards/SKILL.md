---
name: "backend-development-standards"
description: "Enforces qim-server backend development standards including layered architecture, DI container usage, and code organization. Invoke when implementing new features, creating handlers/services, or modifying backend code in qim-server."
---

# qim-server 后端研发规范

## 核心原则

**严格遵循分层架构，Model 层只定义数据结构，禁止包含业务逻辑。**

## 项目架构

```
qim-server/
├── model/          # 数据模型层（仅 struct 定义）
├── repository/     # 数据访问层（数据库 CRUD）
├── service/        # 业务逻辑层（核心业务）
├── handler/        # HTTP 处理层（请求/响应）
├── middleware/     # 中间件（认证/日志等）
├── di/             # 依赖注入容器
├── ai/             # AI 服务相关
├── config/         # 配置管理
├── database/       # 数据库连接
├── cache/          # 缓存管理
├── ws/             # WebSocket 相关
├── pkg/            # 公共工具包
└── utils/          # 工具函数
```

## 分层职责

### 1. Model 层（model/）

**职责：** 仅定义数据结构（struct、字段、GORM 标签、关联关系）

**禁止：**
- ❌ 包含数据库查询逻辑
- ❌ 包含业务判断函数
- ❌ 包含任何方法（除了 GORM 回调钩子）

**示例：**
```go
// ✅ 正确：纯数据结构
type User struct {
    ID       uint   `json:"id" gorm:"primarykey"`
    Username string `json:"username" gorm:"uniqueIndex;size:50;not null"`
    Type     string `json:"type" gorm:"size:20;default:'user'"`
}

// ❌ 错误：Model 层包含业务逻辑
func GetSystemUser(db *gorm.DB) *User { ... }
func IsAIMessage(db *gorm.DB, senderID uint) bool { ... }
```

### 2. Repository 层（repository/）

**职责：** 封装数据库 CRUD 操作，提供基础数据访问接口

**规范：**
- 继承 `BaseRepository` 或使用接口定义
- 不包含业务逻辑判断
- 返回原始数据对象

**示例：**
```go
type UserRepository struct {
    BaseRepository[User]
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
    var user User
    err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
    return &user, err
}
```

### 3. Service 层（service/）

**职责：** 核心业务逻辑处理、事务管理、调用 Repository

**规范：**
- 所有业务逻辑必须在此层实现
- 通过 DI Container 获取依赖
- 处理事务、缓存、权限校验
- 返回业务结果或错误

**示例：**
```go
type UserService struct {
    db       *gorm.DB
    userRepo repository.UserRepository
}

func (s *UserService) GetSystemUser() *model.User {
    var systemUser model.User
    if err := s.db.Where("type = ?", "system").First(&systemUser).Error; err != nil {
        return nil
    }
    return &systemUser
}

func (s *UserService) GetSystemUserID() uint {
    systemUser := s.GetSystemUser()
    if systemUser != nil {
        return systemUser.ID
    }
    return 0
}
```

### 4. Handler 层（handler/）

**职责：** 接收 HTTP 请求、参数验证、调用 Service、返回响应

**规范：**
- 只处理 HTTP 相关逻辑（参数解析、响应格式化）
- 通过 `di.GlobalContainer` 获取 Service
- 不包含业务逻辑
- 统一错误处理

**示例：**
```go
func GetUsers(c *gin.Context) {
    userID, _ := c.Get("user_id")
    
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
    
    if page < 1 {
        page = 1
    }
    if pageSize < 1 || pageSize > 100 {
        pageSize = 10
    }
    
    userSvc := di.GlobalContainer.UserService
    users, total, err := userSvc.GetUsers(userID.(uint), page, pageSize)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取用户列表失败"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": gin.H{
            "list":  users,
            "total": total,
        },
    })
}
```

## 依赖注入规范

### DI Container 使用

**必须通过 DI Container 获取 Service 实例：**

```go
// ✅ 正确
svc := di.GlobalContainer.UserService
msgSvc := di.GlobalContainer.MessageService

// ❌ 错误：直接创建 Service 实例（除非特殊情况）
svc := service.NewUserService(database.GetDB())
```

### 新增 Service 注册流程

1. 在 `service/` 创建 Service 文件
2. 在 `di/container.go` 的 `Container` struct 添加字段
3. 在 `InitContainer` 中初始化并注册

```go
// di/container.go
type Container struct {
    // ... 现有字段
    NewService *service.NewService
}

func InitContainer(secret string, hub *ws.Hub) *Container {
    // ... 现有初始化
    newService := service.NewNewService(db)
    
    container := &Container{
        // ... 现有字段
        NewService: newService,
    }
}
```

## 新功能开发流程

### 步骤 1：定义数据模型

在 `model/model.go` 或新建文件中定义 struct：

```go
type NewFeature struct {
    ID        uint      `json:"id" gorm:"primarykey"`
    Name      string    `json:"name" gorm:"size:100;not null"`
    UserID    uint      `json:"user_id" gorm:"index"`
    Status    string    `json:"status" gorm:"size:20;default:'active'"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 步骤 2：创建 Repository（可选）

如果需要封装复杂查询，在 `repository/` 创建：

```go
type NewFeatureRepository struct {
    BaseRepository[NewFeature]
}

func NewNewFeatureRepository(db *gorm.DB) *NewFeatureRepository {
    return &NewFeatureRepository{
        BaseRepository: BaseRepository[NewFeature]{db: db},
    }
}
```

### 步骤 3：创建 Service

在 `service/` 创建业务逻辑：

```go
type NewFeatureService struct {
    db   *gorm.DB
    repo *repository.NewFeatureRepository
}

func NewNewFeatureService(db *gorm.DB) *NewFeatureService {
    return &NewFeatureService{
        db:   db,
        repo: repository.NewNewFeatureRepository(db),
    }
}

func (s *NewFeatureService) CreateFeature(...) error {
    // 业务逻辑
}
```

### 步骤 4：注册到 DI Container

在 `di/container.go` 中注册。

### 步骤 5：创建 Handler

在 `handler/` 创建 HTTP 处理函数：

```go
func CreateNewFeature(c *gin.Context) {
    // 参数解析
    // 调用 Service
    // 返回响应
}
```

### 步骤 6：注册路由

在 `app/routes.go` 中注册路由。

## 代码规范检查清单

开发新功能时，检查以下项：

- [ ] Model 层只包含 struct 定义，无业务逻辑函数
- [ ] 业务逻辑在 Service 层实现
- [ ] Handler 层只处理 HTTP 请求/响应
- [ ] 通过 DI Container 获取 Service
- [ ] Repository 层封装数据库操作（可选）
- [ ] 分页参数限制上限（pageSize <= 100）
- [ ] 统一错误响应格式 `{"code": xxx, "message": "xxx"}`
- [ ] 用户权限校验（通过 middleware 或 Service）

## 常见反模式

### ❌ 反模式 1：Model 层包含业务逻辑

```go
// model/model.go
func GetSystemUser(db *gorm.DB) *User { ... }  // 错误！
```

**正确做法：** 移到 `service/user_service.go`

### ❌ 反模式 2：Handler 层包含业务逻辑

```go
func CreateOrder(c *gin.Context) {
    // 直接操作数据库，跳过 Service 层
    db.Create(&order)  // 错误！
}
```

**正确做法：** 调用 `di.GlobalContainer.OrderService.CreateOrder()`

### ❌ 反模式 3：直接创建 Service 实例

```go
func SomeHandler(c *gin.Context) {
    svc := service.NewUserService(database.GetDB())  // 不推荐！
}
```

**正确做法：** `svc := di.GlobalContainer.UserService`

### ❌ 反模式 4：无分页上限

```go
pageSize, _ := strconv.Atoi(c.Query("page_size"))
// 未限制上限，可能导致性能问题
```

**正确做法：**
```go
if pageSize < 1 || pageSize > 100 {
    pageSize = 20
}
```

## 文件命名规范

| 类型 | 命名规则 | 示例 |
|------|---------|------|
| Model | 单数名词 | `model.go`, `approval.go` |
| Repository | `{entity}_repository.go` | `user_repository.go` |
| Service | `{entity}_service.go` | `user_service.go` |
| Handler | `{feature}_handler.go` | `file_handler.go` |

## 总结

> **"分层清晰，职责单一，依赖注入，避免混乱。"**

每次开发新功能前，回顾本规范，确保代码结构一致性和可维护性。
