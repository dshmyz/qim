# QIM 全栈优化与功能增强 Spec

**日期:** 2026-05-09  
**状态:** 待实现  
**范围:** qim-client + qim-admin + qim-server

## 一、背景与目标

基于对 QIM 项目的全面代码审查，发现以下问题与改进机会：

1. **qim-client**: 群聊 @ 我缺少视觉提示、分身设置数据不回显
2. **qim-admin**: 配置分散、审计日志功能薄弱、缺少官网式产品介绍页
3. **qim-server**: 大量重复代码模式（分页解析、ID 解析、CRUD）、缺失通用 middleware、数据访问方式不统一

**目标：** 提升代码质量、完善用户体验、建立可复用的基础设施。

## 二、需求总览

### 2.1 qim-client（客户端）

| 编号 | 需求 | 优先级 |
|------|------|--------|
| C1 | 群聊 @ 我特殊视觉提示 | P1 |
| C2 | 分身设置数据回显修复 + 审批流程优化 | P0 |

### 2.2 qim-admin（管理后台）

| 编号 | 需求 | 优先级 |
|------|------|--------|
| A1 | 统一配置项管理 | P1 |
| A2 | 审计日志查询增强（高级筛选、导出、详情） | P1 |
| A3 | 官网式产品介绍首页（无需登录） | P1 |

### 2.3 qim-server（后端服务）

| 编号 | 需求 | 优先级 |
|------|------|--------|
| S1 | 分页参数解析统一抽象 | P0 |
| S2 | URL 参数 ID 解析统一抽象 | P0 |
| S3 | 统一分页响应格式 | P0 |
| S4 | 操作日志自动记录 middleware | P1 |
| S5 | Request ID + Recovery middleware | P2 |
| S6 | 请求限流 middleware | P2 |
| S7 | 统一数据访问方式（禁止 handler 直连 DB） | P1 |

## 三、详细设计

### 3.1 C1: 群聊 @ 我特殊视觉提示

**问题：** 群聊中有人 @ 我时，消息列表和聊天界面缺少明显的视觉提示。

**设计方案：**

#### 3.1.1 消息列表提示
- 在会话列表项中，如果包含未读的 @ 消息，显示特殊的 "@" 徽标（红色背景 + 白色文字）
- 徽标显示在消息预览文字旁边，与未读数量徽标并列
- 会话名称高亮（加粗或变色）

#### 3.1.2 聊天界面提示
- 在聊天窗口顶部显示横幅："你有 N 条未读的 @ 消息"
- 点击横幅自动滚动到第一条 @ 消息
- @ 消息本身添加特殊背景色（淡黄色/淡蓝色）
- @ 消息左侧显示 "@" 图标

#### 3.1.3 实现要点
- 复用现有 `MessageItem.vue` 组件，添加 `isAtMention` prop
- 在消息数据模型中增加 `is_at_mention` 字段（后端需支持）
- 使用项目已有的 Badge/Tag 组件保持视觉一致性
- 样式使用 CSS 变量，支持主题切换

#### 3.1.4 后端支持
- `message_handler.go` 中判断消息内容是否包含当前用户的 @
- 在消息返回数据中增加 `is_at_mention` 字段
- 支持查询某会话中未读的 @ 消息数量

---

### 3.2 C2: 分身设置数据回显修复 + 审批流程优化

**问题：** 
1. 打开分身设置面板时，已保存的配置不回显
2. 审批未通过时无法保存配置草稿

**根因分析：**
- `AvatarBasicSettings.vue` 使用 `v-model` 绑定 `modelValue`，但父组件 `AvatarSettingsPanel.vue` 加载数据后可能未正确更新
- 当前逻辑要求审批通过才能启用，但不允许审批前保存配置

**设计方案：**

#### 3.2.1 回显修复
- 确保 `useAvatar.ts` composable 在 `onMounted` 时调用 `getConfig()` 获取最新数据
- 数据加载完成后正确更新 `config` 响应式对象
- 添加 loading 状态，数据加载期间显示加载指示器

#### 3.2.2 审批流程优化
- **允许先保存草稿**：用户可以填写所有配置并保存，但启用开关需要审批通过后才可打开
- **审批不通过时保留已填配置**：审批通过后无需重新填写
- 增加"保存草稿"按钮，与"提交审批"按钮区分
- 后端支持 `draft` 状态的配置保存

#### 3.2.3 状态流转
```
无配置 → 填写配置 → 保存草稿 → 提交审批 → 审批中 → 审批通过 → 可启用
                                    ↓
                              审批拒绝 → 可修改后重新提交
```

---

### 3.3 A1: 统一配置项管理

**问题：** 前端配置分散在多处（`request.ts`、`env.d.ts`、各组件本地配置），缺乏统一管理。

**设计方案：**

#### 3.3.1 配置中心
创建 `src/config/index.ts` 集中管理所有配置：

```typescript
// 环境配置
export const ENV = {
  API_BASE_URL: import.meta.env.VITE_API_BASE_URL || '/api',
  WS_URL: import.meta.env.VITE_WS_URL || 'ws://localhost:8080',
  MODE: import.meta.env.MODE,
}

// API 配置
export const API_CONFIG = {
  TIMEOUT: 15000,
  RETRY_COUNT: 3,
  RETRY_DELAY: 1000,
}

// 分页默认配置
export const PAGINATION_CONFIG = {
  DEFAULT_PAGE: 1,
  DEFAULT_PAGE_SIZE: 20,
  MAX_PAGE_SIZE: 100,
}

// 功能开关
export const FEATURE_FLAGS = {
  ENABLE_AI: true,
  ENABLE_AVATAR: true,
  ENABLE_REALTIME: true,
}
```

#### 3.3.2 类型定义
```typescript
export interface AppConfig {
  api: ApiConfig
  pagination: PaginationConfig
  features: FeatureFlags
}
```

#### 3.3.3 使用方式
- 所有组件通过 `import { API_CONFIG } from '@/config'` 获取配置
- `request.ts` 使用配置中心的超时时间
- 列表页面使用配置中心的分页默认值

---

### 3.4 A2: 审计日志查询增强

**问题：** 现有操作日志只有基础列表，缺少高级筛选、详情展开、统计图表。

**设计方案：**

#### 3.4.1 高级筛选
- 时间范围选择器（日期范围）
- 操作类型筛选（增/删/改/查）
- 操作用户搜索
- IP 地址筛选
- 模块筛选（用户管理、群组管理、系统配置等）
- 请求状态筛选（成功/失败）

#### 3.4.2 详情展开
- 点击日志行展开详情面板
- 显示：请求方法、请求 URL、请求参数、响应结果、耗时、变更前后对比
- 支持 JSON 格式化展示

#### 3.4.3 导出功能
- 支持导出当前筛选结果为 CSV/Excel
- 导出文件名包含时间范围

#### 3.4.4 统计图表
- 操作频次趋势图（折线图）
- 操作类型分布（饼图）
- 异常操作统计（柱状图）

#### 3.4.5 后端支持
- 增强 `operation_log_handler.go` 支持更多筛选参数
- 增加统计接口 `/api/v1/logs/operation/stats`
- 日志记录增加 `request_body`、`response_body`、`changes` 字段

---

### 3.5 A3: 官网式产品介绍首页

**问题：** 缺少面向所有用户（无需登录）的产品介绍页面。

**设计方案：**

#### 3.5.1 页面结构
```
┌─────────────────────────────────────────┐
│  Hero Section                           │
│  - 产品名称 + Slogan                    │
│  - 主要 CTA 按钮（下载/了解更多）        │
├─────────────────────────────────────────┤
│  核心功能展示                           │
│  - 即时通讯                             │
│  - AI 助手                              │
│  - 数字分身                             │
│  - 团队协作                             │
├─────────────────────────────────────────┤
│  下载区域                               │
│  - Windows / macOS / Linux              │
│  - 显示最新版本号                       │
│  - 下载按钮链接到最新版本               │
├─────────────────────────────────────────┤
│  功能点详细说明                         │
│  - 每个功能模块的图文介绍               │
│  - 快速导航到管理后台对应模块           │
├─────────────────────────────────────────┤
│  Footer                                 │
│  - 版权信息                             │
│  - 管理后台入口                         │
└─────────────────────────────────────────┘
```

#### 3.5.2 路由配置
- 路径：`/home` 或 `/`（如果管理后台改为 `/admin`）
- `meta: { requiresAuth: false }`
- 独立布局（不使用 AdminLayout）

#### 3.5.3 数据来源
- 版本信息从 `/api/v1/client/versions` 获取最新版本
- 功能说明可硬编码或从系统配置读取

---

### 3.6 S1-S3: 后端分页与参数解析统一抽象

**问题：** 分页参数解析、URL ID 解析、分页响应格式在 20-30+ 处重复。

**设计方案：**

#### 3.6.1 分页参数解析 (`pkg/pagination/pagination.go`)

```go
package pagination

type Pagination struct {
    Page     int
    PageSize int
    Offset   int
}

func Parse(c *gin.Context) Pagination {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
    
    if page < 1 { page = 1 }
    if pageSize < 1 || pageSize > 100 { pageSize = 20 }
    
    return Pagination{
        Page:     page,
        PageSize: pageSize,
        Offset:   (page - 1) * pageSize,
    }
}

// 可选：middleware 方式
func Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        p := Parse(c)
        c.Set("pagination", p)
        c.Next()
    }
}
```

#### 3.6.2 URL 参数解析 (`pkg/params/params.go`)

```go
package params

func GetUintParam(c *gin.Context, key string) (uint, error) {
    idStr := c.Param(key)
    id, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        return 0, fmt.Errorf("无效的%s: %s", key, idStr)
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

#### 3.6.3 统一分页响应 (`pkg/response/response.go` 扩展)

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

---

### 3.7 S4: 操作日志自动记录 Middleware

**问题：** 当前操作日志只提供查询，没有自动记录机制。

**设计方案：**

#### 3.7.1 Middleware 实现 (`middleware/operation_log.go`)

```go
func OperationLogMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        startTime := time.Now()
        
        // 执行请求
        c.Next()
        
        // 记录日志
        duration := time.Since(startTime).Milliseconds()
        
        userID, _ := c.Get("user_id")
        username, _ := c.Get("username")
        
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
        
        // 异步保存（避免阻塞响应）
        go saveOperationLog(log)
    }
}
```

#### 3.7.2 模块提取规则
```go
func extractModule(path string) string {
    // /api/v1/users -> 用户管理
    // /api/v1/conversations -> 会话管理
    // /api/v1/admin/roles -> 角色管理
    // ...
}
```

#### 3.7.3 配置选项
- 可配置跳过某些路径（如健康检查、静态文件）
- 可配置是否记录请求体和响应体
- 可配置日志保留天数

---

### 3.8 S5: Request ID + Recovery Middleware

**设计方案：**

#### 3.8.1 Request ID (`middleware/request_id.go`)
- 为每个请求生成唯一 ID（UUID）
- 注入到 `gin.Context` 和响应头 `X-Request-ID`
- 日志中携带 Request ID 便于追踪

#### 3.8.2 Recovery (`middleware/recovery.go`)
- 捕获 panic，记录完整堆栈
- 返回友好错误响应
- 可选：发送告警通知

---

### 3.9 S6: 请求限流 Middleware

**设计方案：**

#### 3.9.1 限流策略
- 基于 IP 的限流（默认 100 req/min）
- 基于用户的限流（认证用户 200 req/min）
- 特殊接口限流（登录接口 10 req/min）

#### 3.9.2 实现方式
- 使用令牌桶算法或滑动窗口
- 内存存储（单机）或 Redis（集群）
- 超过限制返回 429 Too Many Requests

---

### 3.10 S7: 统一数据访问方式

**问题：** handler 层存在三种数据访问方式混用。

**设计方案：**

#### 3.10.1 规范
- **禁止** handler 直接调用 `database.GetDB()` 或 `di.GlobalContainer.DB`
- **必须** 通过 Service 层访问数据
- Service 层通过 Repository 层访问数据库

#### 3.10.2 迁移策略
- 逐步将直接调用 DB 的 handler 改为调用 Service
- 为缺失 Service 的模块创建 Service（如 blacklist、version）
- 添加代码审查规则，禁止新的直连 DB 代码

---

## 四、影响范围

### 4.1 受影响文件

| 模块 | 新增文件 | 修改文件 |
|------|----------|----------|
| qim-client | 消息 @ 提示组件、配置中心 | MessageItem.vue、AvatarSettingsPanel.vue、useAvatar.ts |
| qim-admin | 官网首页、配置中心 | OperationLogs.vue、router/index.ts、request.ts |
| qim-server | pagination.go、params.go、middleware/operation_log.go、middleware/request_id.go | 20+ handler 文件、routes.go |

### 4.2 数据库变更

- `operation_logs` 表可能需要增加字段：`request_body`、`response_body`、`changes`
- `avatar_configs` 表可能需要增加 `status` 字段（draft/pending/approved/rejected）

### 4.3 API 变更

- 新增：`GET /api/v1/logs/operation/stats`（操作日志统计）
- 新增：`GET /api/v1/conversations/:id/at-mentions`（@ 消息查询）
- 修改：消息返回数据增加 `is_at_mention` 字段
- 修改：分身配置 API 支持 `draft` 状态

---

## 五、实施优先级与顺序

### 阶段一：基础设施（P0）
1. S1-S3: 后端分页与参数解析统一抽象
2. C2: 分身设置回显修复

### 阶段二：核心功能（P1）
3. C1: 群聊 @ 我特殊提示
4. A1: 统一配置项管理
5. A2: 审计日志查询增强
6. S4: 操作日志自动记录 middleware
7. S7: 统一数据访问方式

### 阶段三：体验增强（P1-P2）
8. A3: 官网式产品介绍首页
9. S5: Request ID + Recovery middleware
10. S6: 请求限流 middleware

---

## 六、风险与注意事项

1. **分页抽象风险**：Go 泛型在 HTTP handler 层的使用有限制，建议先提供辅助函数而非完整泛型框架
2. **操作日志性能**：自动记录 middleware 应使用异步保存，避免阻塞响应
3. **分身审批流程**：需要与现有审批系统兼容，避免破坏已有逻辑
4. **官网首页 SEO**：如需 SEO 优化，可能需要 SSR 或预渲染，当前方案为纯 SPA
5. **限流中间件**：集群部署需要 Redis 支持，单机可用内存存储

---

## 七、验收标准

1. 所有重复代码模式已被抽象，新增 handler 不再需要手写分页/ID 解析
2. 群聊 @ 消息有明显的视觉提示，用户不会错过
3. 分身设置打开后正确回显已保存的配置
4. 管理后台配置集中在 `src/config/index.ts`
5. 审计日志支持高级筛选、详情查看、导出
6. 官网首页可无需登录访问，展示产品信息和下载链接
7. 操作日志自动记录所有 API 调用
8. 所有 handler 通过 Service 层访问数据，无直连 DB 代码
