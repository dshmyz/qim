# QIM Admin 管理后台全面重构设计文档

> 日期: 2026-04-28
> 状态: 待实现
> 方案: 全面重构（方案 B）

## 概述

对 qim-admin 管理后台进行全面重构，解决现有代码重复、缺乏权限控制、错误处理不完善等核心问题，建立可复用、易维护的前端架构。

## 问题分析

### 严重问题
1. **代码重复严重**：19 个页面各自编写表格、搜索、分页逻辑，未使用现有 BaseTableLayout 组件
2. **缺乏权限控制**：路由和菜单无 RBAC 实现，所有用户可见所有功能
3. **错误处理掩盖**：catch 块仅注释"已在拦截器处理"，无用户可见反馈

### 警告问题
4. **类型定义不完整**：BaseTableLayout 使用 any[] 丧失 TypeScript 保障
5. **对话框硬编码**：角色选项等写死在组件内，未从 API 获取
6. **缺乏数据刷新机制**：仅 Dashboard 有刷新按钮
7. **响应式不完善**：移动端侧边栏处理简陋，表格可能溢出

### 现有优势
- 设计系统完善（CSS 变量、主题切换、Element Plus 覆盖）
- 菜单分组合理（7 个功能模块分组清晰）
- 类型定义丰富（types/index.ts 覆盖大部分业务实体）

## 架构设计

### 分层架构

```
┌─────────────────────────────────────────┐
│              视图层 (Views)              │
│  Dashboard/ UserManagement/ Groups/ ...  │
├─────────────────────────────────────────┤
│           组件层 (Components)            │
│  layout/ data/ forms/ common/           │
├─────────────────────────────────────────┤
│         业务逻辑层 (Composables)         │
│  useEntity/ usePermission/ useSearch     │
├─────────────────────────────────────────┤
│           数据层 (API + Store)           │
│  api/modules/ stores/ request.ts         │
└─────────────────────────────────────────┘
```

### 数据流

组件 → Composable → API → Pinia Store → 响应式更新

## 文件结构

```
qim-admin/src/
├── api/
│   ├── modules/                    # 按模块划分的 API 文件
│   │   ├── users.ts
│   │   ├── groups.ts
│   │   └── ...
│   └── request.ts                  # 统一请求封装（拦截器、错误处理）
│
├── components/
│   ├── layout/                     # 布局组件
│   │   ├── AdminLayout.vue         # 主布局容器
│   │   ├── Sidebar/
│   │   │   ├── index.vue           # 侧边栏主体
│   │   │   ├── MenuGroup.vue       # 菜单分组
│   │   │   └── MenuItem.vue        # 菜单项
│   │   ├── Header/
│   │   │   ├── index.vue           # 顶部栏
│   │   │   ├── ThemeToggle.vue     # 主题切换
│   │   │   └── UserDropdown.vue    # 用户下拉
│   │   └── Breadcrumb/
│   │       └── index.vue           # 面包屑导航
│   │
│   ├── data/                       # 数据展示组件
│   │   ├── DataTable.vue           # 核心数据表格
│   │   ├── SearchForm.vue          # 搜索表单
│   │   ├── SearchField.vue         # 搜索字段
│   │   ├── Pagination/
│   │   │   └── index.vue           # 分页组件
│   │   ├── StatCards.vue           # 统计卡片
│   │   └── StatusTag.vue           # 状态标签
│   │
│   ├── forms/                      # 表单组件
│   │   ├── EntityDialog.vue        # 实体创建/编辑对话框
│   │   └── FieldRenderer.vue       # 字段渲染器
│   │
│   └── common/                     # 通用组件
│       ├── EmptyState.vue          # 空状态
│       ├── LoadingSpinner.vue      # 加载状态
│       ├── ActionButton.vue        # 权限按钮
│       └── ConfirmDialog.vue       # 确认对话框
│
├── composables/
│   ├── useEntity.ts                # CRUD 通用逻辑
│   ├── usePermission.ts            # 权限判断
│   ├── useSearch.ts                # 搜索逻辑
│   └── useDialog.ts                # 对话框管理
│
├── stores/
│   ├── auth.ts                     # 认证状态（保留）
│   ├── permission.ts               # 权限状态（新增）
│   └── app.ts                      # 应用状态（新增）
│
├── views/
│   ├── Dashboard/
│   │   └── index.vue
│   ├── UserManagement/
│   │   ├── index.vue               # 页面入口
│   │   ├── UserTable.vue           # 用户表格
│   │   └── UserDialog.vue          # 用户对话框
│   ├── Organization/
│   │   └── index.vue
│   ├── RoleManagement/
│   │   └── index.vue
│   ├── GroupManagement/
│   │   └── index.vue
│   ├── ConversationManagement/
│   │   └── index.vue
│   ├── ChannelManagement/
│   │   └── index.vue
│   ├── AppManagement/
│   │   └── index.vue
│   ├── MiniAppManagement/
│   │   └── index.vue
│   ├── AIAssistant/
│   │   └── index.vue
│   ├── AIOps/
│   │   └── index.vue
│   ├── SystemMessage/
│   │   └── index.vue
│   ├── Notification/
│   │   └── index.vue
│   ├── Blacklist/
│   │   └── index.vue
│   ├── SensitiveWord/
│   │   └── index.vue
│   ├── OperationLog/
│   │   └── index.vue
│   ├── SystemConfig/
│   │   └── index.vue
│   └── VersionManagement/
│       └── index.vue
│
├── router/
│   ├── index.ts                    # 路由入口
│   ├── routes.ts                   # 路由定义
│   └── guards.ts                   # 路由守卫
│
├── directives/
│   └── permission.ts               # v-permission 指令
│
├── types/
│   └── index.ts                    # 类型定义（保留+补充）
│
├── styles/
│   └── main.css                    # 全局样式（保留）
│
├── App.vue
├── main.ts
└── env.d.ts
```

## 权限系统设计

### 数据结构

```typescript
interface Permission {
  resource: string    // user, group, message, app...
  actions: string[]   // create, read, update, delete
}

interface Role {
  id: number
  name: string
  code: string        // system_admin, system_publisher, system_moderator, system_operator
  permissions: Permission[]
}
```

### 预设角色权限

| 角色 | 权限范围 |
|------|----------|
| system_admin | 全部权限 |
| system_publisher | 消息/通知的 create/read/update |
| system_moderator | 用户/群组的 read、敏感词的 read/update、黑名单管理 |
| system_operator | 数据统计 read、操作日志 read、版本管理 read |

### 权限控制点

1. **路由守卫**：无权限路由自动跳转 403 页面
2. **动态菜单**：侧边栏仅渲染有权限的菜单项
3. **按钮级控制**：`v-permission="'user:create'"` 控制按钮显示/禁用
4. **API 拦截**：无权限的 API 调用在前端拦截

### usePermission Composable

```typescript
function usePermission() {
  const hasPermission = (resource: string, action: string): boolean
  const hasAnyPermission = (checks: string[]): boolean
  const isRole = (roleCode: string): boolean
}
```

### v-permission 指令

```vue
<!-- 显示控制 -->
<el-button v-permission="'user:create'">创建用户</el-button>

<!-- 禁用控制 -->
<el-button v-permission:disabled="'user:delete'">删除用户</el-button>
```

## 核心组件 API

### DataTable

```vue
<DataTable
  :columns="Column[]"
  :data="T[]"
  :loading="boolean"
  :pagination="PaginationConfig"
  @search="SearchParams => void"
  @page-change="PaginationParams => void"
  @refresh="() => void"
>
  <template #search>搜索表单插槽</template>
  <template #actions>操作按钮插槽</template>
  <template #cell-{columnName}="{ row }">自定义列插槽</template>
</DataTable>
```

**Column 接口**：
```typescript
interface Column<T = any> {
  prop?: keyof T
  label: string
  width?: number | string
  sortable?: boolean
  slot?: string           // 使用 slot 渲染
  formatter?: (row: T) => string
}
```

### SearchForm

```vue
<SearchForm @search="handleSearch" @reset="handleReset">
  <SearchField name="keyword" label="关键词" type="input" placeholder="搜索..." />
  <SearchField name="status" label="状态" type="select" :options="statusOptions" />
</SearchForm>
```

### EntityDialog

通过配置自动生成表单：

```vue
<EntityDialog
  v-model="visible"
  :mode="'create' | 'edit'"
  :fields="FormField[]"
  :rules="FormRules"
  :initial-data="Record<string, any>"
  @save="FormData => Promise<void>"
/>
```

**FormField 接口**：
```typescript
interface FormField {
  name: string
  label: string
  type: 'input' | 'textarea' | 'select' | 'upload' | 'switch' | 'number'
  props?: Record<string, any>
  options?: { label: string; value: any }[]  // for select
  required?: boolean
}
```

## useEntity Composable

封装所有列表页的 CRUD 通用逻辑：

```typescript
interface UseEntityOptions<T> {
  api: EntityAPI<T>              // API 对象
  searchFields: string[]         // 搜索字段名
  formFields: FormField[]        // 表单字段配置
  formRules?: FormRules          // 表单验证规则
  successMessages?: {            // 成功提示文案
    create?: string
    update?: string
    delete?: string
  }
}

function useEntity<T>(options: UseEntityOptions<T>) {
  // 列表状态
  const list = ref<T[]>([])
  const loading = ref(false)
  const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
  
  // 搜索
  const searchForm = reactive<Record<string, any>>({})
  const handleSearch = () => void
  const handleReset = () => void
  
  // 对话框
  const dialogVisible = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')
  const currentRow = ref<T | null>(null)
  
  // 操作
  const handleCreate = () => void
  const handleEdit = (row: T) => void
  const handleDelete = (id: number) => Promise<void>
  const handleSave = (formData: Record<string, any>) => Promise<void>
  
  // 数据获取
  const fetchData = () => Promise<void>
  
  return {
    list, loading, pagination,
    searchForm, handleSearch, handleReset,
    dialogVisible, dialogMode, currentRow,
    handleCreate, handleEdit, handleDelete, handleSave,
    fetchData
  }
}
```

## 重构前后代码对比

### Users.vue 重构前（~330 行）

- 搜索栏 HTML + 表单逻辑
- 表格 HTML + 列定义
- 分页 HTML + 事件处理
- 创建/编辑对话框 HTML + 表单
- 角色管理对话框 HTML + 逻辑
- 所有 CRUD 操作函数
- 样式

### Users.vue 重构后（~80 行）

```vue
<template>
  <DataTable :columns="columns" :data="list" :loading="loading" :pagination="pagination"
    @search="handleSearch" @page-change="fetchData" @refresh="fetchData">
    <template #search>
      <SearchForm>
        <SearchField name="keyword" label="关键词" placeholder="用户名或昵称" />
      </SearchForm>
    </template>
    <template #actions>
      <el-button v-permission="'user:create'" type="success" @click="handleCreate">创建用户</el-button>
    </template>
    <template #cell-actions="{ row }">
      <ActionButton v-permission="'user:update'" @click="handleEdit(row)">编辑</ActionButton>
      <ActionButton v-permission="'user:delete'" type="danger" @click="handleDelete(row.id)">删除</ActionButton>
    </template>
  </DataTable>
  <EntityDialog v-model="dialogVisible" :mode="dialogMode" :fields="userFields"
    :rules="userRules" :initial-data="currentRow" @save="handleSave" />
</template>

<script setup lang="ts">
import { useEntity } from '@/composables/useEntity'
import { userApi, userColumns, userFields, userRules } from './config'

const { list, loading, pagination, searchForm, handleSearch,
        dialogVisible, dialogMode, currentRow,
        handleCreate, handleEdit, handleDelete, handleSave, fetchData } = useEntity({
  api: userApi, searchFields: ['keyword'], formFields: userFields, formRules: userRules
})

onMounted(fetchData)
</script>
```

## 错误处理方案

### 全局请求拦截器

```typescript
// 响应拦截
axios.interceptors.response.use(
  response => response,
  error => {
    const status = error.response?.status
    const message = errorMessages[status] || error.message
    ElMessage.error(message)
    
    if (status === 401) {
      authStore.logout()
      router.push('/login')
    }
    if (status === 403) {
      router.push('/403')
    }
    
    return Promise.reject(error)
  }
)
```

### 组件层错误处理

- catch 块中至少记录日志：`console.error('[ModuleName] operation failed:', error)`
- 不掩盖错误，让全局拦截器处理用户提示
- 可选：组件级别提供自定义错误回调

## 实施计划

### 阶段 1 - 基础设施

1. 创建 `stores/permission.ts` - 权限状态管理
2. 创建 `directives/permission.ts` - v-permission 指令
3. 创建 `router/guards.ts` - 路由守卫
4. 创建 `api/request.ts` - 统一请求封装
5. 创建核心组件：
   - `components/data/DataTable.vue`
   - `components/data/SearchForm.vue`
   - `components/data/SearchField.vue`
   - `components/forms/EntityDialog.vue`
   - `components/forms/FieldRenderer.vue`
   - `components/common/ActionButton.vue`
   - `components/common/StatusTag.vue`
6. 创建 composables：
   - `composables/useEntity.ts`
   - `composables/usePermission.ts`
   - `composables/useSearch.ts`

### 阶段 2 - 核心页面重构

1. Dashboard - 验证 StatCards、布局组件
2. UserManagement - 完整验证所有组件和 useEntity
3. GroupManagement - 验证嵌套对话框场景
4. RoleManagement - 验证权限分配场景

### 阶段 3 - 批量迁移

重构剩余 16 个页面：
- Organization, ConversationManagement, ChannelManagement
- AppManagement, MiniAppManagement, AIAssistant, AIOps
- SystemMessage, Notification, Blacklist, SensitiveWord
- OperationLog, SystemConfig, VersionManagement
- Statistics

完成响应式优化和性能优化。

## 响应式优化

- 侧边栏：桌面端可折叠，移动端抽屉式（遮罩+滑出）
- 表格：移动端隐藏次要列，支持横向滚动
- 搜索栏：移动端堆叠布局
- 对话框：移动端全屏模式

## 性能优化

- 路由懒加载（已实现，保留）
- 组件异步加载（大型组件使用 defineAsyncComponent）
- 表格虚拟滚动（万级数据使用 el-table-v2）
- 数据缓存策略（搜索条件、分页状态保持）
