# QIM 管理后台重新设计实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 重新设计 QIM 管理后台的信息架构，重构侧边栏菜单，补充缺失的管理功能模块

**架构：** 基于现有的 qim-admin 项目，对侧边栏进行模块化重构，新增 7 个管理页面，保持所有现有功能不变

**技术栈：** Vue 3 + TypeScript + Element Plus + Vue Router + Pinia

---

## 文件结构

### 新增文件
- `qim-admin/src/components/common/BaseTableLayout.vue`
- `qim-admin/src/components/common/BaseDialog.vue`
- `qim-admin/src/views/Roles.vue`
- `qim-admin/src/views/SensitiveWords.vue`
- `qim-admin/src/views/OperationLogs.vue`
- `qim-admin/src/views/SystemConfig.vue`
- `qim-admin/src/views/VersionManagement.vue`
- `qim-admin/src/views/AIAssistant.vue`
- `qim-admin/src/api/sensitiveWords.ts`
- `qim-admin/src/api/operationLogs.ts`
- `qim-admin/src/api/systemConfig.ts`
- `qim-admin/src/api/versions.ts`
- `qim-admin/src/api/aiAssistant.ts`

### 修改文件
- `qim-admin/src/router/index.ts`
- `qim-admin/src/layouts/AdminLayout.vue`
- `qim-admin/src/types/index.ts`

---

## 任务 1：抽取通用组件

创建 BaseTableLayout 和 BaseDialog 组件用于后续页面复用。

**文件：**
- 创建：`qim-admin/src/components/common/BaseTableLayout.vue`
- 创建：`qim-admin/src/components/common/BaseDialog.vue`

**代码详见原计划文档任务 1**

## 任务 2：更新类型定义

在 types/index.ts 中添加新页面所需的类型定义。

**文件：**
- 修改：`qim-admin/src/types/index.ts`

**代码详见原计划文档任务 2**

## 任务 3：重构侧边栏菜单

将 AdminLayout.vue 中的侧边栏菜单改为 7 大模块分组结构。

**文件：**
- 修改：`qim-admin/src/layouts/AdminLayout.vue`

**代码详见原计划文档任务 3**

## 任务 4：添加新路由

在 router/index.ts 中添加 7 个新路由。

**文件：**
- 修改：`qim-admin/src/router/index.ts`

**代码详见原计划文档任务 4**

## 任务 5：创建角色权限管理页面

复用现有用户接口实现角色权限展示和按角色筛选用户。

**文件：**
- 创建：`qim-admin/src/views/Roles.vue`

**代码详见原计划文档任务 5**

## 任务 6：创建敏感词管理页面

实现敏感词 CRUD 管理，含分类和等级。

**文件：**
- 创建：`qim-admin/src/api/sensitiveWords.ts`
- 创建：`qim-admin/src/views/SensitiveWords.vue`

**代码详见原计划文档任务 6**

## 任务 7：创建操作日志页面

实现操作日志查看和筛选功能。

**文件：**
- 创建：`qim-admin/src/api/operationLogs.ts`
- 创建：`qim-admin/src/views/OperationLogs.vue`

**代码详见原计划文档任务 7**

## 任务 8：创建系统配置页面

实现系统全局参数和开关配置。

**文件：**
- 创建：`qim-admin/src/api/systemConfig.ts`
- 创建：`qim-admin/src/views/SystemConfig.vue`

**代码详见原计划文档任务 8**

## 任务 9：创建版本管理页面

实现客户端版本发布和管理功能。

**文件：**
- 创建：`qim-admin/src/api/versions.ts`
- 创建：`qim-admin/src/views/VersionManagement.vue`

**代码详见原计划文档任务 9**

## 任务 10：创建 AI 助手管理页面

实现机器人配置和管理功能。

**文件：**
- 创建：`qim-admin/src/api/aiAssistant.ts`
- 创建：`qim-admin/src/views/AIAssistant.vue`

**代码详见原计划文档任务 10**
