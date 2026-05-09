# QIM 全栈优化与功能增强 - 任务列表

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 提升代码质量、完善用户体验、建立可复用的基础设施。

**架构：** 分三个阶段执行——阶段一（基础设施）、阶段二（核心功能）、阶段三（体验增强）。每阶段独立推进，涉及文件操作前先理解现有代码模式。

**技术栈：** Go (Gin) + Vue 3 + TypeScript + Electron + WebSocket + MySQL

---

## 文件结构

### qim-server 新增文件
| 文件 | 职责 |
|------|------|
| `qim-server/pkg/pagination/pagination.go` | 分页参数解析工具 |
| `qim-server/pkg/params/params.go` | URL 参数解析工具 |
| `qim-server/middleware/operation_log.go` | 操作日志自动记录中间件 |
| `qim-server/middleware/request_id.go` | Request ID 中间件 |
| `qim-server/middleware/recovery.go` | Panic 恢复中间件 |
| `qim-server/middleware/rate_limit.go` | 请求限流中间件 |

### qim-server 修改文件
| 文件 | 变更 |
|------|------|
| `qim-server/pkg/response/response.go` | 增加 SuccessWithPagination 方法 |
| `qim-server/app/routes.go` | 注册新中间件、更新路由 |
| `qim-server/handler/*.go` | 使用新的分页/参数解析工具 |
| `qim-server/model/model.go` | OperationLog 增加字段 |

### qim-client 新增文件
| 文件 | 职责 |
|------|------|
| `qim-client/src/components/message/AtMentionBadge.vue` | @ 消息徽标组件 |
| `qim-client/src/components/message/AtMentionBanner.vue` | @ 消息提示横幅 |

### qim-client 修改文件
| 文件 | 变更 |
|------|------|
| `qim-client/src/components/message/MessageItem.vue` | 添加 @ 消息视觉样式 |
| `qim-client/src/components/conversation/ConversationList.vue` | 会话列表 @ 提示 |
| `qim-client/src/components/avatar/AvatarSettingsPanel.vue` | 修复回显逻辑 |
| `qim-client/src/composables/useAvatar.ts` | 确保数据加载正确 |
| `qim-client/src/config/index.ts` | 统一配置中心 |

### qim-admin 新增文件
| 文件 | 职责 |
|------|------|
| `qim-admin/src/config/index.ts` | 统一配置中心 |
| `qim-admin/src/views/LandingPage.vue` | 官网式产品介绍首页 |
| `qim-admin/src/layouts/LandingLayout.vue` | 首页独立布局 |

### qim-admin 修改文件
| 文件 | 变更 |
|------|------|
| `qim-admin/src/views/OperationLogs.vue` | 增加高级筛选、详情、导出 |
| `qim-admin/src/api/operationLogs.ts` | 增加统计 API |
| `qim-admin/src/router/index.ts` | 添加首页路由 |
| `qim-admin/src/utils/request.ts` | 使用配置中心 |

---

## 阶段一：基础设施（P0）

### 任务 1：后端 - 分页参数解析统一抽象

**文件：**
- 创建：`qim-server/pkg/pagination/pagination.go`
- 修改：`qim-server/app/routes.go`（可选：注册 middleware）

- [ ] **步骤 1：创建分页解析工具**

```go
package pagination

import (
	"strconv"
	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Page     int
	PageSize int
	Offset   int
}

func Parse(c *gin.Context) Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	
	return Pagination{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := Parse(c)
		c.Set("pagination", p)
		c.Next()
	}
}
```

- [ ] **步骤 2：运行测试验证**

运行：`cd qim-server && go build ./...`
预期：编译通过

- [ ] **步骤 3：Commit**

```bash
git add qim-server/pkg/pagination/pagination.go
git commit -m "feat: add pagination parsing utility"
```

---

### 任务 2：后端 - URL 参数 ID 解析统一抽象

**文件：**
- 创建：`qim-server/pkg/params/params.go`

- [ ] **步骤 1：创建参数解析工具**

```go
package params

import (
	"fmt"
	"strconv"
	"qim-server/pkg/response"
	"github.com/gin-gonic/gin"
)

func GetUintParam(c *gin.Context, key string) (uint, error) {
	idStr := c.Param(key)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("无效的%s", key)
	}
	return uint(id), nil
}

func MustGetUintParam(c *gin.Context, key string) (uint, bool) {
	id, err := GetUintParam(c, key)
	if err != nil {
		response.BadRequest(c, err.Error())
		return 0, false
	}
	return id, true
}
```

- [ ] **步骤 2：运行测试验证**

运行：`cd qim-server && go build ./...`
预期：编译通过

- [ ] **步骤 3：Commit**

```bash
git add qim-server/pkg/params/params.go
git commit -m "feat: add URL parameter parsing utility"
```

---

### 任务 3：后端 - 统一分页响应格式

**文件：**
- 修改：`qim-server/pkg/response/response.go`

- [ ] **步骤 1：扩展 response 包**

在 `response.go` 中添加：

```go
func SuccessWithPagination(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	Success(c, gin.H{
		"list":     list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
```

- [ ] **步骤 2：运行测试验证**

运行：`cd qim-server && go test ./pkg/response/... -v`
预期：PASS

- [ ] **步骤 3：Commit**

```bash
git add qim-server/pkg/response/response.go
git commit -m "feat: add SuccessWithPagination response helper"
```

---

### 任务 4：客户端 - 分身设置回显修复

**文件：**
- 修改：`qim-client/src/components/avatar/AvatarSettingsPanel.vue`
- 修改：`qim-client/src/composables/useAvatar.ts`
- 修改：`qim-client/src/components/avatar/AvatarBasicSettings.vue`

- [ ] **步骤 1：检查 useAvatar.ts 数据加载逻辑**

确保 `onMounted` 时调用 `getConfig()` 并正确更新 `config` 响应式对象。

- [ ] **步骤 2：修复 AvatarSettingsPanel.vue 数据绑定**

确保数据加载完成后正确传递给子组件，添加 loading 状态。

- [ ] **步骤 3：修复 AvatarBasicSettings.vue 回显**

检查 `v-model` 绑定是否正确反映最新数据。

- [ ] **步骤 4：运行开发服务器验证**

运行：`cd qim-client && npm run dev`
预期：打开分身设置面板，已保存配置正确回显

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/components/avatar/*.vue qim-client/src/composables/useAvatar.ts
git commit -m "fix: avatar settings data not echoing correctly"
```

---

## 阶段二：核心功能（P1）

### 任务 5：客户端 - 群聊 @ 我特殊视觉提示

**文件：**
- 创建：`qim-client/src/components/message/AtMentionBadge.vue`
- 创建：`qim-client/src/components/message/AtMentionBanner.vue`
- 修改：`qim-client/src/components/message/MessageItem.vue`
- 修改：`qim-client/src/components/conversation/ConversationList.vue`
- 修改：`qim-server/handler/message_handler.go`

- [ ] **步骤 1：后端 - 消息返回增加 is_at_mention 字段**

在 `message_handler.go` 中，判断消息内容是否包含当前用户的 @，在返回数据中增加 `is_at_mention` 字段。

- [ ] **步骤 2：创建 AtMentionBadge 组件**

```vue
<template>
  <span v-if="isAtMention" class="at-mention-badge">@</span>
</template>

<script setup lang="ts">
defineProps<{
  isAtMention: boolean
}>()
</script>

<style scoped>
.at-mention-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 4px;
  background: #ef4444;
  color: white;
  font-size: 11px;
  font-weight: 700;
  margin-left: 6px;
}
</style>
```

- [ ] **步骤 3：修改 MessageItem.vue 添加 @ 消息样式**

- [ ] **步骤 4：修改 ConversationList.vue 添加 @ 提示**

- [ ] **步骤 5：运行开发服务器验证**

运行：`cd qim-client && npm run dev`
预期：群聊中 @ 我的消息有特殊视觉提示

- [ ] **步骤 6：Commit**

```bash
git add qim-client/src/components/message/*.vue qim-server/handler/message_handler.go
git commit -m "feat: add @ mention visual indicator in group chat"
```

---

### 任务 6：管理后台 - 统一配置项管理

**文件：**
- 创建：`qim-admin/src/config/index.ts`
- 修改：`qim-admin/src/utils/request.ts`

- [ ] **步骤 1：创建配置中心**

```typescript
export const ENV = {
  API_BASE_URL: import.meta.env.VITE_API_BASE_URL || '/api',
  WS_URL: import.meta.env.VITE_WS_URL || 'ws://localhost:8080',
  MODE: import.meta.env.MODE,
}

export const API_CONFIG = {
  TIMEOUT: 15000,
  RETRY_COUNT: 3,
  RETRY_DELAY: 1000,
}

export const PAGINATION_CONFIG = {
  DEFAULT_PAGE: 1,
  DEFAULT_PAGE_SIZE: 20,
  MAX_PAGE_SIZE: 100,
}

export const FEATURE_FLAGS = {
  ENABLE_AI: true,
  ENABLE_AVATAR: true,
  ENABLE_REALTIME: true,
}
```

- [ ] **步骤 2：修改 request.ts 使用配置中心**

- [ ] **步骤 3：运行构建验证**

运行：`cd qim-admin && npm run build`
预期：构建通过

- [ ] **步骤 4：Commit**

```bash
git add qim-admin/src/config/index.ts qim-admin/src/utils/request.ts
git commit -m "feat: add centralized configuration management"
```

---

### 任务 7：管理后台 - 审计日志查询增强

**文件：**
- 修改：`qim-admin/src/views/OperationLogs.vue`
- 修改：`qim-admin/src/api/operationLogs.ts`
- 修改：`qim-server/handler/operation_log_handler.go`

- [ ] **步骤 1：后端 - 增强操作日志筛选**

在 `operation_log_handler.go` 中增加时间范围、操作类型、IP 等筛选参数。

- [ ] **步骤 2：前端 - 添加高级筛选表单**

- [ ] **步骤 3：前端 - 添加详情展开面板**

- [ ] **步骤 4：前端 - 添加导出功能**

- [ ] **步骤 5：运行开发服务器验证**

运行：`cd qim-admin && npm run dev`
预期：审计日志支持高级筛选、详情查看、导出

- [ ] **步骤 6：Commit**

```bash
git add qim-admin/src/views/OperationLogs.vue qim-admin/src/api/operationLogs.ts qim-server/handler/operation_log_handler.go
git commit -m "feat: enhance audit log query with advanced filtering and export"
```

---

### 任务 8：后端 - 操作日志自动记录 Middleware

**文件：**
- 创建：`qim-server/middleware/operation_log.go`
- 修改：`qim-server/app/routes.go`
- 修改：`qim-server/model/model.go`（OperationLog 增加字段）

- [ ] **步骤 1：创建操作日志中间件**

```go
package middleware

import (
	"time"
	"qim-server/model"
	"qim-server/database"
	"github.com/gin-gonic/gin"
)

func OperationLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		
		duration := time.Since(startTime).Milliseconds()
		
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")
		
		if userID == nil {
			return
		}
		
		log := model.OperationLog{
			UserID:     userID.(uint),
			Username:   username.(string),
			Action:     c.Request.Method,
			Module:     extractModule(c.Request.URL.Path),
			IP:         c.ClientIP(),
			RequestURL: c.Request.URL.Path,
			Duration:   int(duration),
			StatusCode: c.Writer.Status(),
		}
		
		go func() {
			db := database.GetDB()
			db.Create(&log)
		}()
	}
}

func extractModule(path string) string {
	// 根据路径提取模块名称
	// /api/v1/users -> 用户管理
	// /api/v1/conversations -> 会话管理
	// ...
}
```

- [ ] **步骤 2：在 routes.go 中注册中间件**

- [ ] **步骤 3：运行测试验证**

运行：`cd qim-server && go build ./...`
预期：编译通过

- [ ] **步骤 4：Commit**

```bash
git add qim-server/middleware/operation_log.go qim-server/app/routes.go
git commit -m "feat: add operation log auto-recording middleware"
```

---

### 任务 9：后端 - 统一数据访问方式

**文件：**
- 修改：`qim-server/handler/blacklist_handler.go`
- 修改：`qim-server/handler/version_handler.go`
- 修改：`qim-server/handler/system_config_handler.go`

- [ ] **步骤 1：为 blacklist 创建 Service**

- [ ] **步骤 2：为 version 创建 Service**

- [ ] **步骤 3：修改 handler 使用 Service 而非直连 DB**

- [ ] **步骤 4：运行测试验证**

运行：`cd qim-server && go build ./...`
预期：编译通过

- [ ] **步骤 5：Commit**

```bash
git add qim-server/handler/blacklist_handler.go qim-server/handler/version_handler.go
git commit -m "refactor: unify data access through service layer"
```

---

## 阶段三：体验增强（P1-P2）

### 任务 10：管理后台 - 官网式产品介绍首页

**文件：**
- 创建：`qim-admin/src/views/LandingPage.vue`
- 创建：`qim-admin/src/layouts/LandingLayout.vue`
- 修改：`qim-admin/src/router/index.ts`

- [ ] **步骤 1：创建 LandingLayout 布局**

- [ ] **步骤 2：创建 LandingPage 页面**

包含 Hero Section、核心功能展示、下载区域、功能点说明、Footer。

- [ ] **步骤 3：添加路由配置**

```typescript
{
  path: '/home',
  name: 'LandingPage',
  component: () => import('@/views/LandingPage.vue'),
  meta: { requiresAuth: false },
}
```

- [ ] **步骤 4：运行开发服务器验证**

运行：`cd qim-admin && npm run dev`
预期：访问 /home 显示产品介绍页面

- [ ] **步骤 5：Commit**

```bash
git add qim-admin/src/views/LandingPage.vue qim-admin/src/layouts/LandingLayout.vue qim-admin/src/router/index.ts
git commit -m "feat: add landing page for product introduction"
```

---

### 任务 11：后端 - Request ID + Recovery Middleware

**文件：**
- 创建：`qim-server/middleware/request_id.go`
- 创建：`qim-server/middleware/recovery.go`
- 修改：`qim-server/app/routes.go`

- [ ] **步骤 1：创建 Request ID 中间件**

- [ ] **步骤 2：创建 Recovery 中间件**

- [ ] **步骤 3：注册中间件**

- [ ] **步骤 4：运行测试验证**

运行：`cd qim-server && go build ./...`
预期：编译通过

- [ ] **步骤 5：Commit**

```bash
git add qim-server/middleware/request_id.go qim-server/middleware/recovery.go
git commit -m "feat: add request ID and recovery middleware"
```

---

### 任务 12：后端 - 请求限流 Middleware

**文件：**
- 创建：`qim-server/middleware/rate_limit.go`
- 修改：`qim-server/app/routes.go`

- [ ] **步骤 1：创建限流中间件**

- [ ] **步骤 2：注册中间件**

- [ ] **步骤 3：运行测试验证**

运行：`cd qim-server && go build ./...`
预期：编译通过

- [ ] **步骤 4：Commit**

```bash
git add qim-server/middleware/rate_limit.go
git commit -m "feat: add rate limiting middleware"
```

---

## 任务依赖关系

```
任务 1 (分页解析) ──┐
任务 2 (ID 解析) ───┤── 任务 7 (审计日志增强) ── 任务 8 (操作日志中间件)
任务 3 (分页响应) ──┘
                                     
任务 4 (分身回显) ── 独立任务

任务 5 (@ 提示) ── 独立任务

任务 6 (配置管理) ── 独立任务

任务 9 (统一数据访问) ── 依赖任务 1-3

任务 10 (官网首页) ── 独立任务

任务 11 (Request ID) ── 独立任务

任务 12 (限流) ── 独立任务
```
