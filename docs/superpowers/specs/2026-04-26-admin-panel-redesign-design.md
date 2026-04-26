# QIM 管理后台重新设计文档

**日期:** 2026-04-26  
**状态:** 已批准

## 一、设计目标

基于 QIM 项目已实现的所有功能，对管理后台进行信息架构重构和功能补充，打造更清晰、更完整的企业级管理后台。

**核心原则：**
1. 接口复用优先：能复用现有接口的功能不新增接口
2. 模块化设计：按业务域分组，每个模块职责明确
3. 渐进式实施：分阶段实施，不影响现有功能

## 二、信息架构

### 2.1 侧边栏菜单结构（7 大模块，18 个页面）

```
📊 数据概览
  ├─ 仪表盘 (/)
  └─ 数据统计 (/statistics)

👥 用户与组织
  ├─ 用户管理 (/users)
  ├─ 组织架构 (/organization)
  └─ 角色权限 (/roles) - 新增

💬 会话与群组
  ├─ 群组管理 (/groups)
  ├─ 会话管理 (/conversations)
  └─ 频道管理 (/channels)

📦 应用生态
  ├─ 应用管理 (/apps)
  ├─ 小程序管理 (/mini-apps)
  └─ AI 助手 (/ai-assistant) - 新增

📨 消息与通知
  ├─ 系统消息 (/messages)
  └─ 通知管理 (/notifications)

🛡️ 安全与合规
  ├─ 黑名单管理 (/blacklist)
  ├─ 敏感词管理 (/sensitive-words) - 新增
  └─ 操作日志 (/operation-logs) - 新增

⚙️ 系统设置
  ├─ 系统配置 (/system-config) - 新增
  └─ 版本管理 (/version-management) - 新增
```

### 2.2 路由完整映射表

| 路由 | 页面组件 | 状态 | 所属模块 |
|------|---------|------|---------|
| `/` | Dashboard | 现有 | 数据概览 |
| `/statistics` | Statistics | 现有 | 数据概览 |
| `/users` | Users | 现有 | 用户与组织 |
| `/organization` | Organization | 现有 | 用户与组织 |
| `/roles` | Roles | 新增 | 用户与组织 |
| `/groups` | Groups | 现有 | 会话与群组 |
| `/conversations` | Conversations | 现有 | 会话与群组 |
| `/channels` | Channels | 现有 | 会话与群组 |
| `/apps` | Apps | 现有 | 应用生态 |
| `/mini-apps` | MiniApps | 现有 | 应用生态 |
| `/ai-assistant` | AIAssistant | 新增 | 应用生态 |
| `/messages` | SystemMessages | 现有 | 消息与通知 |
| `/notifications` | Notifications | 现有 | 消息与通知 |
| `/blacklist` | Blacklist | 现有 | 安全与合规 |
| `/sensitive-words` | SensitiveWords | 新增 | 安全与合规 |
| `/operation-logs` | OperationLogs | 新增 | 安全与合规 |
| `/system-config` | SystemConfig | 新增 | 系统设置 |
| `/version-management` | VersionManagement | 新增 | 系统设置 |

## 三、模块详细设计

### 3.1 数据概览模块

#### 仪表盘（现有）
- 统计卡片：总用户数、在线用户、总群组、总消息
- 最近注册用户列表
- 复用接口：`getDashboardStats`, `getRecentRegistrations`
- 复用组件：`StatCards`, `RecentActivityTable`, `ChartPlaceholders`

#### 数据统计（现有）
- 用户/群组/消息统计卡片
- 用户增长趋势、消息发送趋势
- 活动数据统计、系统概览
- 复用接口：`getStatistics`

### 3.2 用户与组织模块

#### 用户管理（现有，保持不变）
- 功能：用户列表、搜索、创建、编辑、删除、角色分配
- 复用接口：`getUsers`, `createUser`, `updateUser`, `deleteUser`, `assignRoles`

#### 组织架构（现有，保持不变）
- 功能：部门树管理、创建/删除部门、添加/移除员工
- 复用接口：`getOrganizationTree`, `createDepartment`, `updateDepartment`, `deleteDepartment`, `addEmployeeToDepartment`, `removeEmployeeFromDepartment`, `getDepartmentEmployees`

#### 角色权限（新增）
- 功能：
  - 角色列表展示（系统管理员、系统发布者、系统审核员、系统运营）
  - 按角色筛选用户
  - 角色权限说明
- 数据来源：从用户管理中提取角色信息
- 复用接口：`getUsers`（通过角色筛选）、`assignRoles`
- 组件设计：复用 `Users.vue` 的筛选和表格逻辑，创建子组件 `RoleFilter`

### 3.3 会话与群组模块

#### 群组管理（现有，保持不变）
- 功能：群组列表、搜索、查看成员、移除成员、删除群组
- 复用接口：`getGroups`, `getGroupMembers`, `removeGroupMember`, `deleteGroup`

#### 会话管理（现有，保持不变）
- 功能：会话列表查看
- 复用接口：`getConversations`

#### 频道管理（现有，保持不变）
- 功能：频道列表、搜索、创建、编辑、删除
- 复用接口：`getChannels`, `createChannel`, `updateChannel`, `deleteChannel`

### 3.4 应用生态模块

#### 应用管理（现有，保持不变）
- 功能：应用 CRUD、分类管理、状态控制
- 复用接口：`getApps`, `createApp`, `updateApp`, `deleteApp`
- 复用组件：`AppDialog`

#### 小程序管理（现有，保持不变）
- 功能：小程序列表、配置管理
- 复用接口：`getMiniApps`, `createMiniApp`, `updateMiniApp`, `deleteMiniApp`
- 复用组件：`MiniAppDialog`

#### AI 助手（新增）
- 功能：
  - 机器人列表展示
  - 机器人配置（名称、头像、描述、系统提示词）
  - 对话记录查看（按机器人筛选）
- 复用接口：复用客户端已有的 AI 助手相关接口
- 依赖：需后端提供机器人管理和对话记录接口

### 3.5 消息与通知模块

#### 系统消息（现有，保持不变）
- 功能：消息发送、历史记录查看
- 复用接口：`getSystemMessages`, `sendSystemMessage`

#### 通知管理（现有，保持不变）
- 功能：通知推送、模板管理
- 复用接口：`getNotifications`, `sendNotification`

### 3.6 安全与合规模块

#### 黑名单管理（现有，保持不变）
- 功能：黑名单列表、移出黑名单
- 复用接口：`getBlacklist`, `removeBlacklistEntry`

#### 敏感词管理（新增）
- 功能：
  - 敏感词列表（支持分页、搜索）
  - 添加/删除敏感词
  - 敏感词分类（广告、辱骂、政治等）
- 新增接口：需后端提供敏感词 CRUD 接口
  - `GET /api/admin/sensitive-words` - 获取敏感词列表
  - `POST /api/admin/sensitive-words` - 添加敏感词
  - `PUT /api/admin/sensitive-words/:id` - 更新敏感词
  - `DELETE /api/admin/sensitive-words/:id` - 删除敏感词

#### 操作日志（新增）
- 功能：
  - 管理员操作记录列表
  - 按操作人/时间/操作类型筛选
  - 操作详情查看（操作人、时间、操作类型、目标、详情）
- 复用接口：如后端已有日志接口可复用
- 新增接口（如需）：
  - `GET /api/admin/operation-logs` - 获取操作日志列表

### 3.7 系统设置模块

#### 系统配置（新增）
- 功能：
  - 全局参数配置：
    - 消息撤回时间限制（默认 120 秒）
    - 文件大小限制
    - 图片质量设置
  - 开关配置：
    - 用户注册开关
    - 双因素认证开关
    - 文件上传开关
- 新增接口：需后端提供系统配置接口
  - `GET /api/admin/config` - 获取系统配置
  - `PUT /api/admin/config` - 更新系统配置

#### 版本管理（新增）
- 功能：
  - 客户端版本列表（版本号、发布日期、更新说明、强制更新）
  - 发布新版本
  - 设置强制更新
- 复用接口：如客户端已有 `checkVersion` 接口可复用
- 新增接口（如需）：
  - `GET /api/admin/versions` - 获取版本列表
  - `POST /api/admin/versions` - 发布新版本
  - `PUT /api/admin/versions/:id` - 更新版本信息

## 四、技术实现方案

### 4.1 侧边栏重构

**文件：** `src/layouts/AdminLayout.vue`

**修改内容：**
- 将现有的扁平菜单改为分组菜单（el-sub-menu）
- 按 7 大模块重新组织菜单项
- 保持现有的折叠/展开、主题切换功能不变

### 4.2 路由配置更新

**文件：** `src/router/index.ts`

**修改内容：**
- 添加 7 个新路由
- 保持现有的路由守卫逻辑不变

### 4.3 新增页面实现策略

**复用策略：**
1. 角色权限管理：复用 `Users.vue` 的表格和筛选逻辑
2. 敏感词管理：复用 `Apps.vue` 的 CRUD 模式
3. 操作日志：复用 `Users.vue` 的表格展示模式
4. 系统配置：使用 Element Plus 表单组件
5. 版本管理：复用 `Apps.vue` 的列表模式
6. AI 助手：复用客户端 AI 助手相关组件结构

**组件抽取计划：**
- 抽取通用表格布局组件 `BaseTableLayout`（搜索栏 + 表格 + 分页）
- 抽取通用对话框组件 `BaseDialog`（表单对话框模板）

## 五、实施优先级

### P0 - 现有功能重构（第 1 阶段）
- 侧边栏菜单重新分组
- 路由配置调整
- 代码规范检查（确保现有功能不受影响）

### P1 - 高价值低复杂度功能（第 2 阶段）
- 角色权限管理
- 操作日志

### P2 - 需要后端配合的功能（第 3 阶段）
- 敏感词管理
- 系统配置
- 版本管理

### P3 - 依赖较多的功能（第 4 阶段）
- AI 助手管理

## 六、风险评估

| 风险 | 影响 | 应对措施 |
|------|------|---------|
| 后端接口未就绪 | P2/P3 功能无法实现 | 前端先完成 UI 和 mock 数据 |
| 侧边栏重构影响现有路由 | 用户访问旧路由失败 | 保持路由 path 不变，仅调整菜单分组 |
| 新增页面风格不一致 | 用户体验差 | 抽取通用组件，复用现有样式 |

## 七、成功标准

1. 侧边栏菜单按 7 大模块清晰分组
2. 新增 7 个页面全部实现
3. 所有现有功能保持正常
4. 代码复用率高，无重复逻辑
5. 新增页面风格统一，符合设计规范
