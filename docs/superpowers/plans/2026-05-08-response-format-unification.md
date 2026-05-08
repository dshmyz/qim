# 前后端响应格式统一实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将所有 Handler 中的 `c.JSON()` 直接响应替换为统一的 `response` 包函数，确保前后端响应格式完全一致，避免重构导致前端解析错误。

**架构：** 后端使用 `pkg/response/response.go` 提供统一的响应格式 `{code, message, data}`，前端通过 axios 拦截器解析 `response.data`。当前部分 Handler 仍使用旧的 `c.JSON(http.StatusOK, gin.H{...})` 方式，需要统一。

**技术栈：** Go, Gin, Vue 3, TypeScript, Axios

---

## 文件清单

### 需要修改的后端文件（按优先级排序）
- `qim-server/handler/group_handler.go` — 70+ 处 `c.JSON()` 调用，最复杂
- `qim-server/handler/note_handler.go` — 15 处 `c.JSON()` 调用
- `qim-server/handler/misc_handler.go` — 12 处 `c.JSON()` 调用
- 其他 17 个 handler 文件 — 需要逐个检查

### 已确认无需修改的文件
- `qim-server/handler/auth_handler.go` — 已使用 `response` 包 ✅
- `qim-server/handler/message_handler.go` — 已使用 `response` 包 ✅
- `qim-server/handler/notification_handler.go` — 已使用 `response` 包 ✅
- `qim-server/handler/admin_handler.go` — 已使用 `response` 包 ✅

### 前端文件（仅验证，不修改）
- `qim-admin/src/utils/request.ts` — axios 拦截器，已兼容 ✅
- `qim-admin/src/types/index.ts` — `ApiResponse` 接口定义 ✅

---

## 响应格式对照表

### 旧格式（需要替换）
```go
// 成功响应
c.JSON(http.StatusOK, gin.H{"code": 0, "message": "成功", "data": result})

// 错误响应
c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "不存在"})
c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务器错误"})
```

### 新格式（统一使用）
```go
// 成功响应
response.Success(c, data)
response.SuccessWithMessage(c, "成功", data)

// 错误响应
response.BadRequest(c, "参数错误")
response.NotFound(c, "不存在")
response.InternalServerError(c, "服务器错误")
response.Forbidden(c, "无权限")
response.Unauthorized(c, "未认证")
response.Conflict(c, "资源冲突")
```

### 特殊状态码映射
| 旧状态码 | 新函数 | 说明 |
|---------|--------|------|
| 503 ServiceUnavailable | `response.Error(c, 503, errors.ErrCodeServiceUnavailable, msg)` | 服务不可用 |
| 409 Conflict | `response.Conflict(c, msg)` | 资源冲突 |
| 429 TooManyRequests | `response.TooManyRequests(c, msg)` | 请求过多 |

---

## 前端兼容性分析

### 前端解析路径
前端 axios 响应拦截器返回的是完整的 AxiosResponse，所以组件中访问数据的路径是：
```typescript
const res = await apiCall()
res.data.data        // 实际业务数据
res.data.data.list   // 列表数据
res.data.data.total  // 总数
```

### 兼容性结论
- ✅ 前端 `ApiResponse<T>` 接口：`{ code: number, message: string, data: T }`
- ✅ 后端 `Response` 结构：`{ code: int, message: string, data: interface{} }`
- ✅ 成功判断：前端检查 `res.code !== 0`，后端成功码 `ErrCodeSuccess = 0`
- ⚠️ **风险点**：部分旧代码返回 `gin.H{"code": 0, "message": "xxx"}` 没有 `data` 字段，前端访问 `res.data.data` 会得到 `undefined`

---

### 任务 1：重构 note_handler.go

**文件：**
- 修改：`qim-server/handler/note_handler.go`

**错误码映射：**
- `c.JSON(400, ...)` → `response.BadRequest(c, ...)`
- `c.JSON(404, ...)` → `response.NotFound(c, ...)`
- `c.JSON(500, ...)` → `response.InternalServerError(c, ...)`
- `c.JSON(503, ...)` → `response.Error(c, 503, errors.ErrCodeServiceUnavailable, ...)`
- `c.JSON(200, gin.H{"code": 0, ...})` → `response.Success(c, ...)` 或 `response.SuccessWithMessage(c, ...)`

- [ ] **步骤 1：分析 note_handler.go 所有 c.JSON 调用**

读取文件，标记每处需要替换的位置：
- L37: `c.JSON(400, gin.H{"code": 400, "message": "无效的笔记ID"})` → `response.BadRequest(c, "无效的笔记ID")`
- L44: `c.JSON(404, gin.H{"code": 404, "message": "笔记不存在"})` → `response.NotFound(c, "笔记不存在")`
- L50: `c.JSON(503, gin.H{"code": 503, "message": "AI 服务未配置"})` → `response.Error(c, 503, errors.ErrCodeServiceUnavailable, "AI 服务未配置")`
- L68: `c.JSON(500, gin.H{"code": 500, "message": "AI 分析失败"})` → `response.InternalServerError(c, "AI 分析失败")`
- L89: `c.JSON(200, gin.H{...})` → `response.Success(c, gin.H{...})`
- L101: `c.JSON(400, ...)` → `response.BadRequest(c, "无效的笔记ID")`
- L108: `c.JSON(404, ...)` → `response.NotFound(c, "笔记不存在")`
- L127: `c.JSON(400, ...)` → `response.BadRequest(c, "无效的笔记ID")`
- L133: `c.JSON(400, ...)` → `response.BadRequest(c, "参数错误")`
- L141: `c.JSON(500, ...)` → `response.InternalServerError(c, "更新失败")`
- L145: `c.JSON(200, gin.H{"code": 0, "message": "更新成功"})` → `response.SuccessWithMessage(c, "更新成功", nil)`
- L154: `c.JSON(400, ...)` → `response.BadRequest(c, "无效的笔记ID")`
- L160: `c.JSON(400, ...)` → `response.BadRequest(c, "参数错误")`
- L166: `c.JSON(500, ...)` → `response.InternalServerError(c, "更新失败")`
- L170: `c.JSON(200, gin.H{"code": 0, "message": "更新成功"})` → `response.SuccessWithMessage(c, "更新成功", nil)`

- [ ] **步骤 2：执行替换**

使用 SearchReplace 工具逐个替换所有 `c.JSON()` 调用为对应的 `response.*()` 函数。

- [ ] **步骤 3：验证编译**

运行：`cd qim-server && go build ./...`
预期：编译成功，无错误

- [ ] **步骤 4：Commit**

```bash
cd qim-server
git add handler/note_handler.go
git commit -m "refactor: 统一 note_handler 响应格式使用 response 包"
```

---

### 任务 2：重构 misc_handler.go

**文件：**
- 修改：`qim-server/handler/misc_handler.go`

- [ ] **步骤 1：分析 misc_handler.go 所有 c.JSON 调用**

读取文件，标记每处需要替换的位置：
- L31: `c.JSON(200, gin.H{...})` → `response.Success(c, gin.H{...})`
- L64: `c.JSON(200, gin.H{...})` → `response.Success(c, gin.H{...})`
- L85: `c.JSON(400, gin.H{"code": 400, "message": "参数错误"})` → `response.BadRequest(c, "参数错误")`
- L101: `c.JSON(500, gin.H{"code": 500, "message": "创建系统消息失败"})` → `response.InternalServerError(c, "创建系统消息失败")`
- L151: `c.JSON(200, gin.H{...})` → `response.Success(c, gin.H{...})`
- L162: `c.JSON(400, gin.H{"code": 400, "message": "无效的消息ID"})` → `response.BadRequest(c, "无效的消息ID")`
- L171: `c.JSON(400, gin.H{"code": 400, "message": "参数错误"})` → `response.BadRequest(c, "参数错误")`
- L179: `c.JSON(404, gin.H{"code": 404, "message": "消息不存在"})` → `response.NotFound(c, "消息不存在")`
- L185: `c.JSON(500, gin.H{"code": 500, "message": "更新消息状态失败"})` → `response.InternalServerError(c, "更新消息状态失败")`
- L189: `c.JSON(200, gin.H{...})` → `response.Success(c, gin.H{...})`
- L200: `c.JSON(400, gin.H{"code": 400, "message": "请求参数错误"})` → `response.BadRequest(c, "请求参数错误")`
- L208: `c.JSON(200, gin.H{"code": 0, "message": "消息广播成功"})` → `response.SuccessWithMessage(c, "消息广播成功", nil)`
- L217: `c.JSON(400, gin.H{"code": 400, "message": "请求参数错误"})` → `response.BadRequest(c, "请求参数错误")`
- L225: `c.JSON(200, gin.H{"code": 0, "message": "消息发送成功"})` → `response.SuccessWithMessage(c, "消息发送成功", nil)`

- [ ] **步骤 2：执行替换**

使用 SearchReplace 工具逐个替换所有 `c.JSON()` 调用。

- [ ] **步骤 3：验证编译**

运行：`cd qim-server && go build ./...`
预期：编译成功，无错误

- [ ] **步骤 4：Commit**

```bash
cd qim-server
git add handler/misc_handler.go
git commit -m "refactor: 统一 misc_handler 响应格式使用 response 包"
```

---

### 任务 3：重构 group_handler.go（核心）

**文件：**
- 修改：`qim-server/handler/group_handler.go`

这是最复杂的文件，包含 70+ 处 `c.JSON()` 调用。需要按函数分组处理。

- [ ] **步骤 1：读取文件分析所有调用**

读取 `group_handler.go`，按函数分组标记需要替换的位置。

- [ ] **步骤 2：替换错误响应（400/403/404/500）**

批量替换所有错误响应：
- `c.JSON(400, gin.H{"code": 400, "message": "xxx"})` → `response.BadRequest(c, "xxx")`
- `c.JSON(403, gin.H{"code": 403, "message": "xxx"})` → `response.Forbidden(c, "xxx")`
- `c.JSON(404, gin.H{"code": 404, "message": "xxx"})` → `response.NotFound(c, "xxx")`
- `c.JSON(500, gin.H{"code": 500, "message": "xxx"})` → `response.InternalServerError(c, "xxx")`

- [ ] **步骤 3：替换成功响应（200）**

批量替换所有成功响应：
- `c.JSON(200, gin.H{"code": 0, "message": "xxx"})` → `response.SuccessWithMessage(c, "xxx", nil)`
- `c.JSON(200, gin.H{...data...})` → `response.Success(c, gin.H{...data...})`

- [ ] **步骤 4：验证编译**

运行：`cd qim-server && go build ./...`
预期：编译成功，无错误

- [ ] **步骤 5：Commit**

```bash
cd qim-server
git add handler/group_handler.go
git commit -m "refactor: 统一 group_handler 响应格式使用 response 包"
```

---

### 任务 4：重构其他 Handler 文件

**文件：**
- 修改：`qim-server/handler/app_handler.go`
- 修改：`qim-server/handler/file_handler.go`
- 修改：`qim-server/handler/avatar_handler.go`
- 修改：`qim-server/handler/group_document_handler.go`
- 修改：`qim-server/handler/admin_file_handler.go`
- 修改：`qim-server/handler/ai_provider_handler.go`
- 修改：`qim-server/handler/bot_creation_handler.go`
- 修改：`qim-server/handler/channel_handler.go`
- 修改：`qim-server/handler/shortlink_handler.go`
- 修改：`qim-server/handler/user_ai_config_handler.go`
- 修改：`qim-server/handler/ai_usage_handler.go`
- 修改：`qim-server/handler/ai_handler.go`
- 修改：`qim-server/handler/ai_search_handler.go`
- 修改：`qim-server/handler/ai_summary_handler.go`
- 修改：`qim-server/handler/ai_text_handler.go`
- 修改：`qim-server/handler/organization_handler.go`
- 修改：`qim-server/handler/statistics_handler.go`

- [ ] **步骤 1：批量检查所有文件**

对每个文件执行：
```bash
grep -n "c\.JSON(" handler/<file>.go
```

- [ ] **步骤 2：逐个文件替换**

对每个有 `c.JSON()` 调用的文件，按任务 1-3 的模式进行替换。

- [ ] **步骤 3：验证编译**

运行：`cd qim-server && go build ./...`
预期：编译成功，无错误

- [ ] **步骤 4：Commit**

```bash
cd qim-server
git add handler/
git commit -m "refactor: 统一所有 handler 响应格式使用 response 包"
```

---

### 任务 5：添加缺失的错误码常量

**文件：**
- 修改：`qim-server/pkg/errors/errors.go`

- [ ] **步骤 1：检查是否需要新增错误码**

检查是否有使用 503 等非标准状态码的地方，需要添加对应的错误码常量：
```go
const ErrCodeServiceUnavailable = 503
```

- [ ] **步骤 2：添加常量**

在 `errors.go` 中添加缺失的错误码常量。

- [ ] **步骤 3：Commit**

```bash
cd qim-server
git add pkg/errors/errors.go
git commit -m "feat: 添加 ErrCodeServiceUnavailable 错误码常量"
```

---

### 任务 6：端到端验证

- [ ] **步骤 1：运行后端测试**

运行：`cd qim-server && go test ./... -v`
预期：所有测试通过

- [ ] **步骤 2：运行后端编译**

运行：`cd qim-server && go build ./...`
预期：编译成功

- [ ] **步骤 3：检查前端编译**

运行：`cd qim-admin && npm run build`
预期：编译成功，无类型错误

- [ ] **步骤 4：最终 Commit**

```bash
git add -A
git commit -m "chore: 完成前后端响应格式统一重构"
```

---

## 自检清单

**1. 规格覆盖度：**
- ✅ note_handler.go 重构
- ✅ misc_handler.go 重构
- ✅ group_handler.go 重构
- ✅ 其他 17 个 handler 文件重构
- ✅ 错误码常量补充
- ✅ 端到端验证

**2. 占位符扫描：**
- ✅ 无 "TODO"、"待定" 等占位符
- ✅ 每个步骤都有具体的代码示例和命令

**3. 类型一致性：**
- ✅ response 包函数签名一致
- ✅ 错误码常量引用一致
- ✅ 前端 ApiResponse 接口与后端 Response 结构一致
