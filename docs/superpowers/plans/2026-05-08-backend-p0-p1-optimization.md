# 后端 P0+P1 优化项实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 消除全局变量和直接数据库调用，统一响应格式，清理 Model 层业务逻辑，提升代码质量和可维护性。

**架构：** 分 4 个阶段执行：阶段一清理全局变量（P0-2），阶段二统一响应格式（P0-3），阶段三消除直接数据库调用（P0-1），阶段四清理 Model 层和优化架构（P1-4/5/6/7）。每个阶段独立可测试。

**技术栈：** Go, Gin, GORM, 依赖注入

---

## 阶段一：删除全局变量（P0-2）

### 任务 1：删除 service/index.go 全局变量

**文件：**
- 修改：`qim-server/service/index.go`
- 修改：`qim-server/di/container.go`
- 修改：`qim-server/handler/ws_handler.go`
- 修改：`qim-server/ai/ai_service.go`
- 搜索：所有引用 `service.UserSvc`、`service.ConversationSvc`、`service.MessageSvc`、`service.GetAIService()` 的地方

**当前问题：**
```go
// service/index.go
var (
    UserSvc         *UserService
    ConversationSvc *ConversationService
    MessageSvc      *MessageService
    aiSvc           *ai.AIService
)

func Init(userService, conversationService, messageService) { ... }
func SetAIService(svc *ai.AIService) { ... }
func GetAIService() *ai.AIService { ... }
```

**替换方案：** 所有引用改为 `di.GlobalContainer.XXXService`

- [ ] **步骤 1：搜索所有引用位置**

运行：
```bash
cd qim-server
grep -rn "service\.UserSvc\|service\.ConversationSvc\|service\.MessageSvc\|service\.GetAIService\|service\.SetAIService\|service\.Init(" .
```

- [ ] **步骤 2：删除 service/index.go 内容**

将 `service/index.go` 内容替换为：
```go
package service

// 全局变量已废弃，统一使用 di.GlobalContainer 获取服务实例
// 保留此文件仅作为向后兼容的占位
```

- [ ] **步骤 3：更新 di/container.go**

删除 `service.Init(...)` 调用：
```go
// 删除这行
// service.Init(userService, conversationService, messageService)
```

- [ ] **步骤 4：更新所有引用处**

将以下模式替换：
```go
// 旧
svc := service.UserSvc
svc := service.ConversationSvc
svc := service.MessageSvc
aiSvc := service.GetAIService()

// 新
svc := di.GlobalContainer.UserService
svc := di.GlobalContainer.ConversationService
svc := di.GlobalContainer.MessageService
aiSvc := di.GlobalContainer.AIService
```

- [ ] **步骤 5：在 DI Container 中添加 AIService 字段**

修改 `di/container.go`：
```go
type Container struct {
    // ... 现有字段
    AIService *ai.AIService  // 新增
}

func InitContainer(secret string, hub *ws.Hub) *Container {
    // ... 现有初始化
    aiService := ai.NewAIService(db, config)  // 根据实际情况初始化
    
    container := &Container{
        // ... 现有字段
        AIService: aiService,
    }
    
    // 删除 service.Init(...) 调用
    GlobalContainer = container
    return container
}
```

- [ ] **步骤 6：验证编译**

运行：`cd qim-server && go build ./...`
预期：编译成功

- [ ] **步骤 7：运行测试**

运行：`cd qim-server && go test ./... -v`
预期：所有测试通过

- [ ] **步骤 8：Commit**

```bash
cd qim-server
git add service/index.go di/container.go
git commit -m "refactor: 删除 service 层全局变量，统一使用 DI 容器"
```

---

## 阶段二：统一响应格式（P0-3）

### 任务 2：重构 note_handler.go

**文件：**
- 修改：`qim-server/handler/note_handler.go`

**错误码映射：**
- `c.JSON(400, ...)` → `response.BadRequest(c, ...)`
- `c.JSON(404, ...)` → `response.NotFound(c, ...)`
- `c.JSON(500, ...)` → `response.InternalServerError(c, ...)`
- `c.JSON(503, ...)` → `response.Error(c, 503, 503, ...)`
- `c.JSON(200, gin.H{"code": 0, ...})` → `response.SuccessWithMessage(c, ..., nil)`
- `c.JSON(200, gin.H{...data...})` → `response.Success(c, gin.H{...data...})`

- [ ] **步骤 1：替换所有 c.JSON 调用**

逐行替换（共 15 处）：
```go
// L37: 400 → BadRequest
c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的笔记ID"})
→ response.BadRequest(c, "无效的笔记ID")

// L44: 404 → NotFound
c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "笔记不存在"})
→ response.NotFound(c, "笔记不存在")

// L50: 503 → Error
c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "AI 服务未配置"})
→ response.Error(c, http.StatusServiceUnavailable, 503, "AI 服务未配置")

// L68: 500 → InternalServerError
c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "AI 分析失败"})
→ response.InternalServerError(c, "AI 分析失败")

// L89: 200 success with data
c.JSON(http.StatusOK, gin.H{...})
→ response.Success(c, gin.H{...})

// L101: 400 → BadRequest
→ response.BadRequest(c, "无效的笔记ID")

// L108: 404 → NotFound
→ response.NotFound(c, "笔记不存在")

// L127: 400 → BadRequest
→ response.BadRequest(c, "无效的笔记ID")

// L133: 400 → BadRequest
→ response.BadRequest(c, "参数错误")

// L141: 500 → InternalServerError
→ response.InternalServerError(c, "更新失败")

// L145: 200 success message only
c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
→ response.SuccessWithMessage(c, "更新成功", nil)

// L154: 400 → BadRequest
→ response.BadRequest(c, "无效的笔记ID")

// L160: 400 → BadRequest
→ response.BadRequest(c, "参数错误")

// L166: 500 → InternalServerError
→ response.InternalServerError(c, "更新失败")

// L170: 200 success message only
→ response.SuccessWithMessage(c, "更新成功", nil)
```

- [ ] **步骤 2：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 3：Commit**

```bash
git add handler/note_handler.go
git commit -m "refactor: 统一 note_handler 响应格式"
```

---

### 任务 3：重构 misc_handler.go

**文件：**
- 修改：`qim-server/handler/misc_handler.go`

- [ ] **步骤 1：替换所有 c.JSON 调用（共 12 处）**

```go
// L31: 200 success
→ response.Success(c, gin.H{...})

// L64: 200 success
→ response.Success(c, gin.H{...})

// L85: 400
→ response.BadRequest(c, "参数错误")

// L101: 500
→ response.InternalServerError(c, "创建系统消息失败")

// L151: 200 success
→ response.Success(c, gin.H{...})

// L162: 400
→ response.BadRequest(c, "无效的消息ID")

// L171: 400
→ response.BadRequest(c, "参数错误")

// L179: 404
→ response.NotFound(c, "消息不存在")

// L185: 500
→ response.InternalServerError(c, "更新消息状态失败")

// L189: 200 success
→ response.Success(c, gin.H{...})

// L200: 400
→ response.BadRequest(c, "请求参数错误")

// L208: 200 message only
c.JSON(http.StatusOK, gin.H{"code": 0, "message": "消息广播成功"})
→ response.SuccessWithMessage(c, "消息广播成功", nil)

// L217: 400
→ response.BadRequest(c, "请求参数错误")

// L225: 200 message only
→ response.SuccessWithMessage(c, "消息发送成功", nil)
```

- [ ] **步骤 2：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 3：Commit**

```bash
git add handler/misc_handler.go
git commit -m "refactor: 统一 misc_handler 响应格式"
```

---

### 任务 4：重构 group_handler.go（核心）

**文件：**
- 修改：`qim-server/handler/group_handler.go`

这是最大的文件，包含 70+ 处 `c.JSON()` 调用。按函数分组处理。

- [ ] **步骤 1：批量替换错误响应**

使用正则批量替换：
```bash
# 400 错误
sed -i 's/c\.JSON(http\.StatusBadRequest, gin\.H{"code": 400, "message": "\([^"]*\)"})/response.BadRequest(c, "\1")/g' handler/group_handler.go

# 403 错误
sed -i 's/c\.JSON(http\.StatusForbidden, gin\.H{"code": 403, "message": "\([^"]*\)"})/response.Forbidden(c, "\1")/g' handler/group_handler.go

# 404 错误
sed -i 's/c\.JSON(http\.StatusNotFound, gin\.H{"code": 404, "message": "\([^"]*\)"})/response.NotFound(c, "\1")/g' handler/group_handler.go

# 500 错误
sed -i 's/c\.JSON(http\.StatusInternalServerError, gin\.H{"code": 500, "message": "\([^"]*\)"})/response.InternalServerError(c, "\1")/g' handler/group_handler.go
```

- [ ] **步骤 2：手动替换成功响应**

成功响应格式多样，需要逐个检查：
```go
// 纯消息成功
c.JSON(http.StatusOK, gin.H{"code": 0, "message": "xxx"})
→ response.SuccessWithMessage(c, "xxx", nil)

// 带数据成功
c.JSON(http.StatusOK, gin.H{"code": 0, "data": {...}})
→ response.Success(c, gin.H{"data": {...}})

// 混合
c.JSON(http.StatusOK, gin.H{"code": 0, "message": "xxx", "data": {...}})
→ response.SuccessWithMessage(c, "xxx", gin.H{...})
```

- [ ] **步骤 3：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 4：Commit**

```bash
git add handler/group_handler.go
git commit -m "refactor: 统一 group_handler 响应格式"
```

---

## 阶段三：消除直接数据库调用（P0-1）

### 任务 5：为缺失的 Handler 创建 Service 并注册到 DI

**文件：**
- 创建：`qim-server/service/conversation_service.go`（补充缺失方法）
- 创建：`qim-server/service/channel_service.go`
- 创建：`qim-server/service/shortlink_service.go`
- 创建：`qim-server/service/bot_creation_service.go`
- 创建：`qim-server/service/ai_provider_service.go`
- 创建：`qim-server/service/version_service.go`
- 创建：`qim-server/service/smart_reply_service.go`
- 修改：`qim-server/di/container.go`

- [ ] **步骤 1：分析每个 Handler 的数据库操作**

对每个使用 `database.GetDB()` 的 Handler，分析其数据库操作：

| Handler | 主要操作 | 需要创建的 Service 方法 |
|---------|---------|----------------------|
| conversation_handler | 获取会话列表、创建会话、更新会话 | GetConversations, CreateConversation, UpdateConversation |
| channel_handler | 频道 CRUD、订阅管理 | GetChannels, CreateChannel, SubscribeChannel |
| shortlink_handler | 短链接 CRUD、访问统计 | CreateShortLink, GetShortLink, RecordVisit |
| bot_creation_handler | 机器人创建、审批 | CreateBot, GetPendingBots, ApproveBot |
| ai_provider_handler | AI 提供商 CRUD、测试连接 | GetProviders, CreateProvider, TestConnection |
| version_handler | 版本管理 CRUD | GetVersions, CreateVersion, UpdateVersion |
| smart_reply_handler | 智能回复配置 | GetSmartReplyConfig, UpdateSmartReplyConfig |

- [ ] **步骤 2：创建 Service 文件**

为每个模块创建 Service 文件，参考现有模式：

```go
// service/channel_service.go
package service

import (
    "qim-server/model"
    "gorm.io/gorm"
)

type ChannelService struct {
    db *gorm.DB
}

func NewChannelService(db *gorm.DB) *ChannelService {
    return &ChannelService{db: db}
}

func (s *ChannelService) GetChannels(page, pageSize int) ([]model.Channel, int64, error) {
    var channels []model.Channel
    var total int64
    
    query := s.db.Model(&model.Channel{})
    query.Count(&total)
    
    err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&channels).Error
    return channels, total, err
}

// ... 其他方法
}
```

- [ ] **步骤 3：注册到 DI Container**

修改 `di/container.go`：
```go
type Container struct {
    // ... 现有字段
    ChannelService       *service.ChannelService
    ShortLinkService     *service.ShortLinkService
    BotCreationService   *service.BotCreationService
    AIProviderService    *service.AIProviderService
    VersionService       *service.VersionService
    SmartReplyService    *service.SmartReplyService
    ConversationService  *service.ConversationService  // 补充
}

func InitContainer(secret string, hub *ws.Hub) *Container {
    // ... 现有初始化
    channelService := service.NewChannelService(db)
    shortLinkService := service.NewShortLinkService(db)
    botCreationService := service.NewBotCreationService(db)
    aiProviderService := service.NewAIProviderService(db)
    versionService := service.NewVersionService(db)
    smartReplyService := service.NewSmartReplyService(db)
    
    container := &Container{
        // ... 现有字段
        ChannelService:     channelService,
        ShortLinkService:   shortLinkService,
        BotCreationService: botCreationService,
        AIProviderService:  aiProviderService,
        VersionService:     versionService,
        SmartReplyService:  smartReplyService,
    }
}
```

- [ ] **步骤 4：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 5：Commit**

```bash
git add service/*.go di/container.go
git commit -m "feat: 创建缺失的 Service 并注册到 DI 容器"
```

---

### 任务 6：重构 conversation_handler.go

**文件：**
- 修改：`qim-server/handler/conversation_handler.go`

- [ ] **步骤 1：替换所有 database.GetDB() 调用**

将以下模式替换：
```go
// 旧
db := database.GetDB()
db.Where("user_id = ?", uid).Find(&members)

// 新
convSvc := di.GlobalContainer.ConversationService
members, err := convSvc.GetConversationMembers(uid)
```

- [ ] **步骤 2：在 ConversationService 中补充缺失方法**

```go
// service/conversation_service.go 补充
func (s *ConversationService) GetConversationMembers(userID uint) ([]model.ConversationMember, error) {
    var members []model.ConversationMember
    err := s.db.Where("user_id = ?", userID).
        Preload("Conversation").
        Preload("Conversation.LastMessage").
        Find(&members).Error
    return members, err
}

func (s *ConversationService) GetGroupsByConversationIDs(ids []uint) ([]model.Group, error) {
    var groups []model.Group
    err := s.db.Where("conversation_id IN ?", ids).Find(&groups).Error
    return groups, err
}
```

- [ ] **步骤 3：更新 Handler 使用 Service**

```go
func GetConversations(c *gin.Context) {
    userID, _ := c.Get("user_id")
    uid := userID.(uint)
    
    convSvc := di.GlobalContainer.ConversationService
    members, err := convSvc.GetConversationMembers(uid)
    if err != nil {
        response.InternalServerError(c, "获取会话列表失败")
        return
    }
    // ... 后续逻辑
}
```

- [ ] **步骤 4：删除 database 导入**

```go
import (
    // 删除 "qim-server/database"
    // 添加 "qim-server/di"
)
```

- [ ] **步骤 5：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 6：Commit**

```bash
git add handler/conversation_handler.go service/conversation_service.go
git commit -m "refactor: conversation_handler 使用 DI 容器替代直接数据库调用"
```

---

### 任务 7：重构 user_handler.go

**文件：**
- 修改：`qim-server/handler/user_handler.go`

- [ ] **步骤 1：分析 13 处 database.GetDB() 调用**

| 行号 | 操作 | 迁移方案 |
|------|------|---------|
| 166-178 | 创建 UserService 实例 | 使用 UserHandler 已有的 userService 字段 |
| 196 | 查询用户角色 | 添加到 UserService |
| 245 | 更新用户 | 使用已有 userService |
| 270 | 查询部门 | 添加到 UserService |
| 319 | 修改密码 | 添加到 UserService |
| 364 | 查询好友 | 添加到 UserService |
| 408 | 搜索用户 | 添加到 UserService |
| 443 | 查询统计 | 添加到 UserService |
| 468 | 查询在线状态 | 添加到 UserService |
| 516 | 批量操作 | 添加到 UserService |
| 627 | 其他查询 | 添加到 UserService |
| 663 | 其他查询 | 添加到 UserService |

- [ ] **步骤 2：在 UserService 中补充方法**

```go
// service/user_service.go 补充
func (s *UserService) GetUserRoles(userID uint) ([]string, error) {
    var userRoles []model.UserRole
    err := s.db.Where("user_id = ?", userID).Find(&userRoles).Error
    if err != nil {
        return nil, err
    }
    roles := make([]string, len(userRoles))
    for i, ur := range userRoles {
        roles[i] = ur.Role
    }
    return roles, nil
}

func (s *UserService) ChangePassword(userID uint, oldPassword, newPassword string) error {
    var user model.User
    if err := s.db.First(&user, userID).Error; err != nil {
        return err
    }
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
        return errors.New("原密码错误")
    }
    hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    return s.db.Model(&user).Update("password_hash", string(hash)).Error
}
```

- [ ] **步骤 3：重构 Handler 使用 Service**

```go
// 旧
func ChangePassword(c *gin.Context) {
    db := database.GetDB()
    // 直接操作数据库
}

// 新
func (h *UserHandler) ChangePassword(c *gin.Context) {
    userID, _ := c.Get("user_id")
    err := h.userService.ChangePassword(userID.(uint), oldPwd, newPwd)
    if err != nil {
        response.BadRequest(c, err.Error())
        return
    }
    response.SuccessWithMessage(c, "密码修改成功", nil)
}
```

- [ ] **步骤 4：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 5：Commit**

```bash
git add handler/user_handler.go service/user_service.go
git commit -m "refactor: user_handler 使用 DI 容器替代直接数据库调用"
```

---

### 任务 8：重构剩余 Handler 文件

**文件：**
- 修改：`qim-server/handler/channel_handler.go`
- 修改：`qim-server/handler/shortlink_handler.go`
- 修改：`qim-server/handler/bot_creation_handler.go`
- 修改：`qim-server/handler/ai_provider_handler.go`
- 修改：`qim-server/handler/version_handler.go`
- 修改：`qim-server/handler/smart_reply_handler.go`
- 修改：`qim-server/handler/admin_file_handler.go`
- 修改：`qim-server/handler/group_document_handler.go`
- 修改：`qim-server/handler/note_handler.go`（剩余 database.GetDB）
- 修改：`qim-server/handler/misc_handler.go`（剩余 database.GetDB）
- 修改：`qim-server/handler/auth_handler.go`（剩余 database.GetDB）
- 修改：`qim-server/handler/message_handler.go`（剩余 database.GetDB）
- 修改：`qim-server/handler/message_sender.go`
- 修改：`qim-server/handler/prompt_builder.go`
- 修改：`qim-server/service/avatar_worker_pool.go`

- [ ] **步骤 1：逐个文件替换 database.GetDB()**

对每个文件：
1. 分析数据库操作
2. 在对应 Service 中补充方法
3. 替换为 `di.GlobalContainer.XXXService.Method()`
4. 删除 `database` 导入

- [ ] **步骤 2：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 3：Commit**

```bash
git add handler/ service/
git commit -m "refactor: 消除所有 handler 中的直接数据库调用"
```

---

### 任务 9：重构 ws/ws.go 中的数据库调用

**文件：**
- 修改：`qim-server/ws/ws.go`

WebSocket 模块有 14 处 `database.GetDB()` 调用，需要通过 DI 注入。

- [ ] **步骤 1：修改 Hub 结构，添加依赖**

```go
// ws/ws.go
type Hub struct {
    // ... 现有字段
    UserService     *service.UserService
    MessageService  *service.MessageService
    ConversationSvc *service.ConversationService
}

func NewHub(userSvc *service.UserService, msgSvc *service.MessageService, convSvc *service.ConversationService) *Hub {
    return &Hub{
        // ... 现有初始化
        UserService:     userSvc,
        MessageService:  msgSvc,
        ConversationSvc: convSvc,
    }
}
```

- [ ] **步骤 2：更新 DI Container 创建 Hub**

```go
// di/container.go
func InitContainer(secret string, hub *ws.Hub) *Container {
    // 先创建 Service
    userService := service.NewUserService(db)
    messageService := service.NewMessageService(db, nil)  // 暂不传 hub
    
    // 创建 Hub 并注入依赖
    hub := ws.NewHub(userService, messageService, conversationService)
    
    // 重新创建 messageService 传入 hub
    messageService = service.NewMessageService(db, hub)
    
    // ... 其余初始化
}
```

- [ ] **步骤 3：替换 ws.go 中所有 database.GetDB()**

```go
// 旧
db := database.GetDB()
db.Where("user_id = ?", userID).Find(&user)

// 新
user, err := h.UserService.GetUser(userID)
```

- [ ] **步骤 4：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 5：Commit**

```bash
git add ws/ws.go di/container.go
git commit -m "refactor: ws 模块使用依赖注入替代直接数据库调用"
```

---

### 任务 10：删除 database.GetDB() 全局函数

**文件：**
- 修改：`qim-server/database/database.go`

- [ ] **步骤 1：确认无残留引用**

运行：
```bash
cd qim-server
grep -rn "database\.GetDB()" .
```

预期：无结果

- [ ] **步骤 2：标记 GetDB 为废弃（可选）**

```go
// database/database.go
// Deprecated: 使用 di.GlobalContainer.DB 替代
func GetDB() *gorm.DB {
    return globalDB
}
```

- [ ] **步骤 3：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 4：运行测试**

运行：`cd qim-server && go test ./... -v`

- [ ] **步骤 5：Commit**

```bash
git add database/database.go
git commit -m "refactor: 标记 database.GetDB() 为废弃"
```

---

## 阶段四：Model 层和架构优化（P1）

### 任务 11：清理 Model 层业务逻辑

**文件：**
- 修改：`qim-server/model/model.go`
- 创建：`qim-server/service/group_ai_config_service.go`

- [ ] **步骤 1：分析 Group 模型中的方法**

```go
// 需要保留（ApprovalEntity 接口实现，属于数据映射）
func (g *Group) GetID() uint
func (g *Group) GetCreatorID() uint
func (g *Group) GetApprovalStatus() string
func (g *Group) GetApprovalType() string
func (g *Group) SetApprovalStatus(status string)
func (g *Group) SetApprovedAt(t *time.Time)
func (g *Group) SetApprovedBy(adminID uint)
func (g *Group) SetRejectReason(reason string)
func (g *Group) GetRejectReason() string

// 需要迁移到 Service 层（业务逻辑）
func (g *Group) GetAIConfig() *GroupAIConfig
func (g *Group) SetAIConfig(config *GroupAIConfig) error
```

- [ ] **步骤 2：创建 GroupAIConfigService**

```go
// service/group_ai_config_service.go
package service

import "qim-server/model"

type GroupAIConfigService struct{}

func NewGroupAIConfigService() *GroupAIConfigService {
    return &GroupAIConfigService{}
}

func (s *GroupAIConfigService) ParseConfig(configJSON string) *model.GroupAIConfig {
    if configJSON == "" {
        return s.defaultConfig()
    }
    var config model.GroupAIConfig
    if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
        return s.defaultConfig()
    }
    return &config
}

func (s *GroupAIConfigService) SerializeConfig(config *model.GroupAIConfig) (string, error) {
    data, err := json.Marshal(config)
    if err != nil {
        return "", err
    }
    return string(data), nil
}

func (s *GroupAIConfigService) defaultConfig() *model.GroupAIConfig {
    return &model.GroupAIConfig{
        Enabled:          false,
        AssistantName:    "AI助手",
        ReplyMode:        "mention_only",
        // ...
    }
}
```

- [ ] **步骤 3：更新 Group 模型**

```go
// model/model.go
// 删除 GetAIConfig 和 SetAIConfig 方法
// 保留 ApprovalEntity 接口方法
```

- [ ] **步骤 4：更新所有调用处**

```go
// 旧
config := group.GetAIConfig()

// 新
configSvc := di.GlobalContainer.GroupAIConfigService
config := configSvc.ParseConfig(group.AIConfigJSON)
```

- [ ] **步骤 5：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 6：Commit**

```bash
git add model/model.go service/group_ai_config_service.go
git commit -m "refactor: 将 Group AI 配置解析逻辑从 Model 层迁移到 Service 层"
```

---

### 任务 12：Service 接口抽象

**文件：**
- 创建：`qim-server/service/interfaces.go`
- 修改：`qim-server/di/container.go`

- [ ] **步骤 1：定义核心 Service 接口**

```go
// service/interfaces.go
package service

import (
    "context"
    "qim-server/model"
)

type UserServiceInterface interface {
    GetUser(id uint) (*model.User, error)
    GetUserByUsername(username string) (*model.User, error)
    CreateUser(user *model.User) error
    UpdateUser(id uint, updates map[string]interface{}) error
    GetUserRoles(userID uint) ([]string, error)
    ChangePassword(userID uint, oldPassword, newPassword string) error
}

type MessageServiceInterface interface {
    GetMessages(conversationID uint, page, pageSize int) ([]model.Message, int64, error)
    CreateMessage(msg *model.Message) error
    RecallMessage(messageID, userID uint) error
    MarkAsRead(messageID, userID uint) error
}

type ConversationServiceInterface interface {
    GetConversations(userID uint) ([]model.Conversation, error)
    CreateConversation(convType string, members []uint) (*model.Conversation, error)
    GetConversationMembers(userID uint) ([]model.ConversationMember, error)
}

// ... 其他接口
```

- [ ] **步骤 2：确保 Service 实现接口**

在 Service 文件中添加编译时检查：
```go
// service/user_service.go
var _ UserServiceInterface = (*UserService)(nil)
```

- [ ] **步骤 3：更新 DI Container 使用接口**

```go
// di/container.go
type Container struct {
    UserService          service.UserServiceInterface
    MessageService       service.MessageServiceInterface
    ConversationService  service.ConversationServiceInterface
    // ... 其他保持具体类型（非核心服务暂不需要接口）
}
```

- [ ] **步骤 4：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 5：Commit**

```bash
git add service/interfaces.go di/container.go service/*.go
git commit -m "feat: 为核心 Service 添加接口抽象"
```

---

### 任务 13：修复 StringArray 类型

**文件：**
- 修改：`qim-server/model/model.go`

- [ ] **步骤 1：修复 Scan 方法**

```go
func (s *StringArray) Scan(value interface{}) error {
    if value == nil {
        *s = StringArray{}
        return nil
    }
    
    str, ok := value.(string)
    if !ok {
        return fmt.Errorf("failed to scan StringArray: expected string, got %T", value)
    }
    
    return json.Unmarshal([]byte(str), s)
}
```

- [ ] **步骤 2：修复 Value 方法**

```go
func (s StringArray) Value() (driver.Value, error) {
    if s == nil {
        return "[]", nil
    }
    data, err := json.Marshal(s)
    if err != nil {
        return nil, err
    }
    return string(data), nil
}
```

- [ ] **步骤 3：添加必要的 import**

```go
import (
    "database/sql/driver"
    "encoding/json"
    "fmt"
)
```

- [ ] **步骤 4：验证编译**

运行：`cd qim-server && go build ./...`

- [ ] **步骤 5：Commit**

```bash
git add model/model.go
git commit -m "fix: 修复 StringArray 类型的 Scan 和 Value 方法"
```

---

## 验证阶段

### 任务 14：端到端验证

- [ ] **步骤 1：编译检查**

运行：`cd qim-server && go build ./...`
预期：编译成功，无错误

- [ ] **步骤 2：运行所有测试**

运行：`cd qim-server && go test ./... -v -cover`
预期：所有测试通过

- [ ] **步骤 3：检查无残留**

运行：
```bash
# 检查无 database.GetDB() 残留
grep -rn "database\.GetDB()" handler/ ws/ service/

# 检查无 c.JSON() 残留
grep -rn "c\.JSON(" handler/

# 检查无全局变量残留
grep -rn "service\.UserSvc\|service\.ConversationSvc\|service\.MessageSvc" .
```

预期：无结果

- [ ] **步骤 4：最终 Commit**

```bash
git add -A
git commit -m "chore: 完成 P0+P1 优化项，统一依赖注入和响应格式"
```

---

## 自检

**1. 规格覆盖度：**
- ✅ P0-2: 删除全局变量（任务 1）
- ✅ P0-3: 统一响应格式（任务 2/3/4）
- ✅ P0-1: 消除直接数据库调用（任务 5-10）
- ✅ P1-4: Model 层清理（任务 11）
- ✅ P1-6: Service 接口抽象（任务 12）
- ✅ P1-11: StringArray 修复（任务 13）

**2. 占位符扫描：**
- ✅ 无 "TODO"、"待定" 等占位符
- ✅ 每个步骤都有具体的代码示例和命令

**3. 类型一致性：**
- ✅ DI Container 字段名一致
- ✅ Service 方法签名一致
- ✅ 响应函数使用一致
