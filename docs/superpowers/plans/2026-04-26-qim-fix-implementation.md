# QIM 项目功能修复实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 修复 QIM 即时通讯项目的安全漏洞、完善未实现功能、优化代码质量

**架构：** 分三个阶段执行——阶段一（安全修复）、阶段二（核心功能完善）、阶段三（代码质量优化）。每阶段独立推进，涉及文件操作前先备份。

**技术栈：** Go (Gin) + Vue 3 + TypeScript + Electron + WebSocket + MySQL

---

## 文件结构概览

| 文件 | 职责 | 涉及任务 |
|------|------|----------|
| `qim-server/handler/auth_handler.go` | JWT 生成、2FA 逻辑 | 1, 2 |
| `qim-server/config/config.go` | 配置读取 | 1 |
| `qim-server/app/routes.go` | 路由注册 | 8 |
| `qim-server/handler/message_handler.go` | 消息处理逻辑 | 7 |
| `qim-server/handler/file_handler.go` | 文件上传/下载 | 4, 17 |
| `qim-server/middleware/auth.go` | JWT 中间件 | 1 |
| `qim-server/pkg/response/response.go` | 统一响应格式 | 11 |
| `qim-client/src/views/Main.vue` | 主视图（过大） | 20 |
| `qim-client/src/components/chat/ChatWindow.vue` | 聊天窗口（过大） | 21 |
| `qim-client/src/views/Login.vue` | 登录页 | 2 |
| `qim-client/src/utils/miniAppUtils.ts` | 小程序工具 | 5 |
| `qim-client/src/composables/useCurrentUser.ts` | 用户状态 | 6 |
| `qim-client/src/config.ts` | 前端配置 | 8 |
| `qim-client/src/components/chat/ChatWindow copy.vue` | 残留文件 | 22 |
| `qim-client/src/views/Main.vue.backup` | 残留文件 | 22 |

---

### 任务 1：修复 JWT Secret 硬编码

**文件：**
- 创建：`qim-server/config/config.go` 中添加 JWT Secret 配置项
- 修改：`qim-server/handler/auth_handler.go:284-296` — generateToken 函数
- 修改：`qim-server/config.yaml` — 添加 jwt.secret 配置

**步骤：**

- [ ] **步骤 1：备份文件**
```bash
cp qim-server/handler/auth_handler.go qim-server/handler/auth_handler.go.bak
cp qim-server/config/config.yaml qim-server/config/config.yaml.bak
```

- [ ] **步骤 2：在 config.go 中添加 JWT 配置结构**

在 `qim-server/config/config.go` 中，修改 JWTConfig 结构体，确保包含 Secret 字段：
```go
type JWTConfig struct {
    Secret string `mapstructure:"secret"`
    Expire int    `mapstructure:"expire"`
}
```

- [ ] **步骤 3：在 config.yaml 中添加 JWT 配置**

在 `qim-server/config.yaml` 中添加：
```yaml
jwt:
  secret: "${QIM_JWT_SECRET:change-me-to-random-string}"
  expire: 7200
```

- [ ] **步骤 4：修改 generateToken 使用配置中的 Secret**

在 `qim-server/handler/auth_handler.go` 的 generateToken 函数中，将硬编码的 `"your-secret-key-change-in-production"` 替换为 `cfg.JWT.Secret`：
```go
func generateToken(userID uint, username string) string {
    claims := middleware.Claims{
        UserID:   userID,
        Username: username,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JWT.Expire) * time.Second)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, _ := token.SignedString([]byte(cfg.JWT.Secret))
    return tokenString
}
```

- [ ] **步骤 5：修改 RefreshToken 的过期时间也使用配置**

同样修改 RefreshToken 函数中的 token 过期时间。

- [ ] **步骤 6：验证编译通过**
```bash
cd qim-server && go build .
```

---

### 任务 2：实现 2FA 验证（TOTP）

**文件：**
- 创建：`qim-server/handler/totp_helper.go` — TOTP 工具函数
- 修改：`qim-server/handler/auth_handler.go:110-146` — VerifyTwoFA
- 修改：`qim-server/handler/auth_handler.go:30-86` — Login 增加 2FA session 返回
- 修改：`qim-client/src/views/Login.vue` — 添加 2FA 验证步骤

**步骤：**

- [ ] **步骤 1：备份文件**
```bash
cp qim-server/handler/auth_handler.go qim-server/handler/auth_handler.go.2fa.bak
cp qim-client/src/views/Login.vue qim-client/src/views/Login.vue.2fa.bak
```

- [ ] **步骤 2：添加 TOTP 依赖**
```bash
cd qim-server && go get github.com/pquerna/otp
```

- [ ] **步骤 3：创建 TOTP helper 工具**

创建 `qim-server/handler/totp_helper.go`：
```go
package handler

import (
    "crypto/rand"
    "encoding/base32"
    "github.com/pquerna/otp/totp"
)

func GenerateTOTPSecret() (string, error) {
    b := make([]byte, 20)
    _, err := rand.Read(b)
    if err != nil {
        return "", err
    }
    return base32.StdEncoding.EncodeToString(b), nil
}

func VerifyTOTPCode(secret, code string) bool {
    return totp.Validate(code, secret)
}
```

- [ ] **步骤 4：修改 Login 返回 2FA session**

在 `auth_handler.go` 的 Login 函数中，当 `user.TwoFactorEnabled` 为 true 时，生成 session 并返回，而不是直接报"需要双因素认证"：
```go
if user.TwoFactorEnabled {
    session := generateSessionID()
    response.Success(c, gin.H{
        "need_2fa": true,
        "session":  session,
        "message":  "需要双因素认证",
    })
    return
}
```

- [ ] **步骤 5：实现 VerifyTwoFA 真正的验证逻辑**

修改 VerifyTwoFA 函数，使用 TOTP 验证：
```go
func VerifyTwoFA(c *gin.Context) {
    var req struct {
        Session  string `json:"session" binding:"required"`
        Code     string `json:"code" binding:"required"`
        Username string `json:"username" binding:"required"`
    }
    // ... 验证 TOTP 代码，成功后返回 token
}
```

- [ ] **步骤 6：前端添加 2FA 输入界面**

修改 `Login.vue`，在检测到 `need_2fa: true` 时，显示 6 位验证码输入框，调用 `/auth/2fa/verify` 接口。

---

### 任务 3：修复 CORS 配置

**文件：**
- 修改：`qim-server/app/routes.go:38-47` — CORS 中间件配置
- 修改：`qim-server/config/config.go` — 添加 CORS 配置项
- 修改：`qim-server/config.yaml` — 添加 allowed_origins

**步骤：**

- [ ] **步骤 1：备份**
```bash
cp qim-server/app/routes.go qim-server/app/routes.go.bak
```

- [ ] **步骤 2：在 config.yaml 中添加 CORS 配置**
```yaml
cors:
  allowed_origins:
    - "http://localhost:5173"
    - "app://localhost"
```

- [ ] **步骤 3：修改 routes.go 使用动态配置**

```go
corsConfig := cors.Config{
    AllowOrigins:     cfg.CORS.AllowedOrigins,
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
}
```

---

### 任务 4：文件上传添加类型/大小限制

**文件：**
- 修改：`qim-server/handler/file_handler.go:17-83` — UploadFile 函数
- 修改：`qim-server/config/config.go` — 添加上传限制配置
- 修改：`qim-server/config.yaml` — 添加 upload 配置

**步骤：**

- [ ] **步骤 1：备份**
```bash
cp qim-server/handler/file_handler.go qim-server/handler/file_handler.go.bak
```

- [ ] **步骤 2：添加上传配置到 config.yaml**
```yaml
upload:
  max_size_mb: 100
  allowed_types:
    - "image/jpeg"
    - "image/png"
    - "image/gif"
    - "application/pdf"
    - "application/zip"
    - "application/msword"
    - "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    - "text/plain"
    - "video/mp4"
```

- [ ] **步骤 3：在 UploadFile 中添加验证**

```go
func UploadFile(c *gin.Context) {
    // 检查文件大小
    if file.Size > int64(cfg.Upload.MaxSizeMB)*1024*1024 {
        c.JSON(http.StatusBadRequest, gin.H{
            "code": 400, 
            "message": fmt.Sprintf("文件大小超过限制(%dMB)", cfg.Upload.MaxSizeMB),
        })
        return
    }
    
    // 检查文件类型
    allowed := make(map[string]bool)
    for _, t := range cfg.Upload.AllowedTypes {
        allowed[t] = true
    }
    mimeType := http.DetectContentType(fileBytes)
    if !allowed[mimeType] && !strings.HasPrefix(mimeType, "image/") {
        c.JSON(http.StatusBadRequest, gin.H{
            "code": 400,
            "message": "不支持的文件类型",
        })
        return
    }
    // ... 其余逻辑
}
```

---

### 任务 5：修复前端 XSS 风险

**文件：**
- 修改：`qim-client/src/utils/miniAppUtils.ts` — innerHTML 改为安全渲染

**步骤：**

- [ ] **步骤 1：备份**
```bash
cp qim-client/src/utils/miniAppUtils.ts qim-client/src/utils/miniAppUtils.ts.bak
```

- [ ] **步骤 2：搜索所有 innerHTML 使用**

```bash
grep -r "innerHTML\|v-html" qim-client/src/
```

- [ ] **步骤 3：替换为安全的 DOM 操作**

将 `element.innerHTML = content` 改为使用 `textContent` 或经过消毒的 HTML。对 v-html 指令，使用 DOMPurify 等库进行消毒。

---

### 任务 6：前端密码处理安全加固

**文件：**
- 修改：`qim-client/src/composables/useCurrentUser.ts` — 移除密码明文存储

**步骤：**

- [ ] **步骤 1：备份**
```bash
cp qim-client/src/composables/useCurrentUser.ts qim-client/src/composables/useCurrentUser.ts.bak
```

- [ ] **步骤 2：审查密码处理逻辑**

确保：
1. 不在 localStorage 中存储密码
2. 不在 console.log 中输出密码
3. 不在内存中保留不必要的密码副本

---

### 任务 7：消息撤回添加时间限制

**文件：**
- 修改：`qim-server/handler/message_handler.go:437-482` — RecallMessage 函数

**步骤：**

- [ ] **步骤 1：备份**
```bash
cp qim-server/handler/message_handler.go qim-server/handler/message_handler.go.bak
```

- [ ] **步骤 2：添加时间检查**

```go
func RecallMessage(c *gin.Context) {
    // ... 现有验证 ...
    
    // 检查是否在2分钟内
    if time.Since(msg.CreatedAt) > 2*time.Minute {
        c.JSON(http.StatusBadRequest, gin.H{
            "code": 400,
            "message": "超过2分钟，无法撤回消息",
        })
        return
    }
    
    // ... 撤回逻辑 ...
}
```

---

### 任务 8：统一后端响应格式

**文件：**
- 修改：`qim-server/handler/message_handler.go` — 使用 response 包
- 修改：`qim-server/handler/conversation_handler.go` — 使用 response 包
- 修改：`qim-server/handler/group_handler.go` — 使用 response 包
- 修改：`qim-server/handler/file_handler.go` — 使用 response 包
- 修改：`qim-server/handler/channel_handler.go` — 使用 response 包

**步骤：**

- [ ] **步骤 1：审查 response 包定义**

读取 `qim-server/pkg/response/response.go` 确认可用的函数：
```go
func Success(c *gin.Context, data interface{})
func BadRequest(c *gin.Context, msg string)
func Unauthorized(c *gin.Context, msg string)
func Forbidden(c *gin.Context, msg string)
func NotFound(c *gin.Context, msg string)
func InternalServerError(c *gin.Context, msg string)
```

- [ ] **步骤 2：逐文件替换**

在每个 handler 文件中，将：
```go
c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": xxx})
```
替换为：
```go
response.Success(c, xxx)
```

将：
```go
c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "xxx"})
```
替换为：
```go
response.BadRequest(c, "xxx")
```

以此类推。

---

### 任务 9：实现消息提醒功能

**文件：**
- 修改：`qim-server/handler/message_handler.go:484-511` — RemindMessage 函数

**步骤：**

- [ ] **步骤 1：备份**

- [ ] **步骤 2：实现提醒逻辑**

消息提醒功能应该是：在群组中提醒某个人注意某条消息。实现：
1. 获取消息所在会话
2. 查询被提醒用户的 WebSocket 连接
3. 发送提醒通知
4. 同时创建系统通知记录

```go
func RemindMessage(c *gin.Context) {
    userID, _ := c.Get("user_id")
    msgIDStr := c.Param("id")
    
    var req struct {
        TargetUserID uint `json:"target_user_id"`
    }
    // 解析请求体
    
    // 验证权限（群主/管理员可以提醒任何成员，普通成员可以提醒任何人）
    // 获取消息信息
    // 获取会话信息
    // 发送 WebSocket 提醒通知
    // 创建通知记录
    
    if ws.GlobalHub != nil {
        remindMsg := ws.WSMessage{
            Type: "message_remind",
            Data: gin.H{
                "message_id":      msg.ID,
                "conversation_id": msg.ConversationID,
                "sender_id":       userID,
                "content_preview": msg.Content,
            },
        }
        jsonMsg, _ := json.Marshal(remindMsg)
        ws.GlobalHub.SendToUser(req.TargetUserID, jsonMsg)
    }
}
```

---

### 任务 10：文件夹树结构正确构建

**文件：**
- 修改：`qim-server/handler/file_handler.go:215-243` — GetFolderTree 函数

**步骤：**

- [ ] **步骤 1：修复树结构构建**

```go
func GetFolderTree(c *gin.Context) {
    userID, _ := c.Get("user_id")
    db := database.GetDB()
    
    var folders []model.Folder
    db.Where("user_id = ?", userID).Order("created_at ASC").Find(&folders)
    
    // 构建 ID -> Folder 映射
    folderMap := make(map[uint]*model.Folder)
    for i := range folders {
        folderMap[folders[i].ID] = &folders[i]
    }
    
    // 构建树结构
    type FolderWithChildren struct {
        model.Folder
        Children []FolderWithChildren `json:"children,omitempty"`
    }
    
    folderTreeMap := make(map[uint]*FolderWithChildren)
    var roots []FolderWithChildren
    
    for i := range folders {
        f := &FolderWithChildren{
            Folder:   folders[i],
            Children: nil,
        }
        folderTreeMap[f.ID] = f
        
        if f.ParentID == nil {
            roots = append(roots, *f)
        }
    }
    
    // 组装父子关系
    for _, folder := range folders {
        if folder.ParentID != nil {
            if parent, exists := folderTreeMap[*folder.ParentID]; exists {
                if child, exists := folderTreeMap[folder.ID]; exists {
                    parent.Children = append(parent.Children, *child)
                }
            }
        }
    }
    
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": roots,
    })
}
```

---

### 任务 11：版本检查从配置读取

**文件：**
- 修改：`qim-server/handler/auth_handler.go:256-282` — CheckVersion 函数

**步骤：**

- [ ] **步骤 1：在 config 中添加版本配置**

在 `config.yaml` 中添加：
```yaml
app:
  version: "1.0.0"
  update_url: "https://example.com/download/qim-latest"
  force_update: false
  release_notes: "1. 新功能说明..."
```

- [ ] **步骤 2：修改 CheckVersion 使用配置值**

```go
func CheckVersion(c *gin.Context) {
    var req struct {
        Version string `json:"version" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "参数错误")
        return
    }
    
    response.Success(c, gin.H{
        "latest_version":  cfg.App.Version,
        "current_version": req.Version,
        "need_update":     req.Version != cfg.App.Version,
        "force_update":    cfg.App.ForceUpdate,
        "update_url":      cfg.App.UpdateURL,
        "release_notes":   cfg.App.ReleaseNotes,
    })
}
```

---

### 任务 12：清理前端残留文件

**文件：**
- 删除：`qim-client/src/components/chat/ChatWindow copy.vue`
- 删除：`qim-client/src/components/chat/modal/ChatWindow copy.vue`
- 删除：`qim-client/src/views/Main.vue.backup`

**步骤：**

- [ ] **步骤 1：备份后删除**
```bash
mkdir -p .backup-cleanup
cp "qim-client/src/components/chat/ChatWindow copy.vue" .backup-cleanup/
cp "qim-client/src/components/chat/modal/ChatWindow copy.vue" .backup-cleanup/
cp qim-client/src/views/Main.vue.backup .backup-cleanup/

rm "qim-client/src/components/chat/ChatWindow copy.vue"
rm "qim-client/src/components/chat/modal/ChatWindow copy.vue"
rm qim-client/src/views/Main.vue.backup
```

- [ ] **步骤 2：验证无引用**
```bash
grep -r "ChatWindow copy\|Main.vue.backup" qim-client/src/
# 应该无结果
```

---

### 任务 13：清理 console.log 调试代码

**文件：**
- 修改：`qim-client/src/` 下的多个文件

**步骤：**

- [ ] **步骤 1：统计 console.log 数量**
```bash
grep -r "console\.log" qim-client/src/ | wc -l
```

- [ ] **步骤 2：保留必要的 error/warn，移除调试用 log**

保留 `console.error` 和 `console.warn`，移除纯调试用的 `console.log`。

---

### 任务 14：统一前端 request 函数

**文件：**
- 创建：`qim-client/src/utils/request.ts` — 统一请求函数
- 修改：`qim-client/src/composables/useRequest.ts` — 使用统一请求
- 修改：`qim-client/src/composables/useChatRequest.ts` — 使用统一请求

**步骤：**

- [ ] **步骤 1：检查现有 request 函数重复情况**

```bash
grep -r "async function request\|const request" qim-client/src/composables/
```

- [ ] **步骤 2：创建统一 request.ts**

```typescript
import axios from 'axios'
import { API_BASE_URL } from '@/config'

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
})

apiClient.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

apiClient.interceptors.response.use(
  response => response.data,
  error => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default apiClient
```

- [ ] **步骤 3：修改各 composable 使用统一 client**

将各 composable 中的独立 fetch/axios 调用替换为使用 `apiClient`。

---

## 执行顺序建议

```
任务 1 (JWT) → 任务 3 (CORS) → 任务 4 (上传限制) → 任务 7 (撤回限制)
→ 任务 11 (版本配置) → 任务 10 (文件夹树) → 任务 9 (消息提醒) → 任务 8 (统一响应)
→ 任务 12 (清理残留) → 任务 13 (清理log) → 任务 14 (统一request)
→ 任务 2 (2FA) → 任务 5 (XSS) → 任务 6 (密码安全)
```

**理由：** 先做简单配置类修复（影响小），再做功能完善，最后做前端清理和安全加固。2FA 和 XSS 修复涉及较大改动放在后面。
