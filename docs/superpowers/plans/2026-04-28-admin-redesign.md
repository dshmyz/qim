# QIM Admin 管理后台全面重构实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 全面重构 qim-admin 管理后台，建立可复用组件系统、RBAC 权限控制、统一数据层封装

**架构：** 分层架构（视图层 → 组件层 → Composables → 数据层），通过 DataTable + EntityDialog + useEntity 实现页面代码量减少 70%+

**技术栈：** Vue 3 + TypeScript + Element Plus + Pinia + Vue Router + Vite

---

## 文件结构

### 新增文件

**权限系统：**
- `qim-admin/src/stores/permission.ts` - 权限状态管理
- `qim-admin/src/directives/permission.ts` - v-permission 指令
- `qim-admin/src/router/guards.ts` - 路由守卫
- `qim-admin/src/views/Forbidden.vue` - 403 页面

**组件系统：**
- `qim-admin/src/components/layout/Sidebar/index.vue` - 侧边栏主体
- `qim-admin/src/components/layout/Header/index.vue` - 顶部栏
- `qim-admin/src/components/layout/Header/ThemeToggle.vue` - 主题切换
- `qim-admin/src/components/layout/Header/UserDropdown.vue` - 用户下拉
- `qim-admin/src/components/layout/Breadcrumb/index.vue` - 面包屑
- `qim-admin/src/components/data/DataTable.vue` - 核心数据表格
- `qim-admin/src/components/data/SearchForm.vue` - 搜索表单
- `qim-admin/src/components/data/SearchField.vue` - 搜索字段
- `qim-admin/src/components/data/StatusTag.vue` - 状态标签
- `qim-admin/src/components/forms/EntityDialog.vue` - 实体对话框
- `qim-admin/src/components/forms/FieldRenderer.vue` - 字段渲染器
- `qim-admin/src/components/common/ActionButton.vue` - 权限按钮
- `qim-admin/src/components/common/EmptyState.vue` - 空状态

**Composables：**
- `qim-admin/src/composables/useEntity.ts` - CRUD 通用逻辑
- `qim-admin/src/composables/usePermission.ts` - 权限判断
- `qim-admin/src/composables/useSearch.ts` - 搜索逻辑

**页面重构（文件夹结构）：**
- `qim-admin/src/views/Dashboard/index.vue`
- `qim-admin/src/views/UserManagement/index.vue`
- `qim-admin/src/views/UserManagement/config.ts` - 列定义、表单配置
- `qim-admin/src/views/GroupManagement/index.vue`
- `qim-admin/src/views/RoleManagement/index.vue`

### 修改文件

- `qim-admin/src/router/index.ts` - 更新路由路径，添加路由守卫
- `qim-admin/src/layouts/AdminLayout.vue` - 重构为组合式组件
- `qim-admin/src/main.ts` - 注册权限指令
- `qim-admin/src/stores/auth.ts` - 补充权限加载
- `qim-admin/src/types/index.ts` - 补充 Permission、Role 类型
- `qim-admin/src/utils/request.ts` - 增强错误处理

---

## 阶段 1 - 基础设施

### 任务 1：类型定义补充

**文件：**
- 修改：`qim-admin/src/types/index.ts`

- [ ] **步骤 1：在 types/index.ts 末尾添加权限和角色类型**

在现有 `AIBot` 接口后添加：

```typescript
// 权限相关
export interface Permission {
  resource: string
  actions: string[]
}

export interface RouteMeta {
  title: string
  requiresAuth?: boolean
  permission?: string  // 格式: resource:action
  icon?: string
}
```

- [ ] **步骤 2：Commit**

```bash
cd qim-admin
git add src/types/index.ts
git commit -m "feat(types): 添加 Permission、RouteMeta 类型定义"
```

---

### 任务 2：权限 Store

**文件：**
- 创建：`qim-admin/src/stores/permission.ts`
- 测试：`qim-admin/tests/unit/stores/permission.test.ts`

- [ ] **步骤 1：编写权限 store 测试**

```typescript
// tests/unit/stores/permission.test.ts
import { createPinia, setActivePinia } from 'pinia'
import { usePermissionStore } from '@/stores/permission'
import type { Permission, Role } from '@/types'

describe('permission store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should initialize with empty permissions', () => {
    const store = usePermissionStore()
    expect(store.permissions).toEqual([])
    expect(store.roles).toEqual([])
  })

  it('should set permissions and roles', () => {
    const store = usePermissionStore()
    const perm: Permission = { resource: 'user', actions: ['create', 'read', 'update', 'delete'] }
    const role: Role = { id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }
    store.setPermissions([perm])
    store.setRoles([role])
    expect(store.permissions.length).toBe(1)
    expect(store.roles.length).toBe(1)
  })

  it('hasPermission returns true when permission exists', () => {
    const store = usePermissionStore()
    store.setPermissions([{ resource: 'user', actions: ['create', 'read'] }])
    expect(store.hasPermission('user', 'create')).toBe(true)
    expect(store.hasPermission('user', 'delete')).toBe(false)
  })

  it('hasAnyPermission returns true if any check passes', () => {
    const store = usePermissionStore()
    store.setPermissions([{ resource: 'user', actions: ['read'] }])
    expect(store.hasAnyPermission(['user:delete', 'user:read'])).toBe(true)
    expect(store.hasAnyPermission(['user:delete', 'group:create'])).toBe(false)
  })

  it('isRole returns true for matching role', () => {
    const store = usePermissionStore()
    store.setRoles([{ id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }])
    expect(store.isRole('system_admin')).toBe(true)
    expect(store.isRole('system_publisher')).toBe(false)
  })

  it('system_admin has all permissions', () => {
    const store = usePermissionStore()
    store.setPermissions([])
    store.setRoles([{ id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }])
    expect(store.hasPermission('anything', 'anything')).toBe(true)
  })

  it('reset clears all state', () => {
    const store = usePermissionStore()
    store.setPermissions([{ resource: 'user', actions: ['read'] }])
    store.setRoles([{ id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }])
    store.markInitialized()
    store.reset()
    expect(store.permissions).toEqual([])
    expect(store.roles).toEqual([])
    expect(store.isInitialized).toBe(false)
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

```bash
cd qim-admin
npm run test -- tests/unit/stores/permission.test.ts
```
预期：FAIL - module not found

- [ ] **步骤 3：创建 permission store**

```typescript
// src/stores/permission.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Permission, Role } from '@/types'

export const usePermissionStore = defineStore('permission', () => {
  const permissions = ref<Permission[]>([])
  const roles = ref<Role[]>([])
  const isInitialized = ref(false)

  function setPermissions(newPermissions: Permission[]) {
    permissions.value = newPermissions
  }

  function setRoles(newRoles: Role[]) {
    roles.value = newRoles
  }

  function hasPermission(resource: string, action: string): boolean {
    if (roles.value.some(r => r.code === 'system_admin')) {
      return true
    }
    return permissions.value.some(
      p => p.resource === resource && p.actions.includes(action)
    )
  }

  function hasAnyPermission(checks: string[]): boolean {
    return checks.some(check => {
      const [resource, action] = check.split(':')
      return hasPermission(resource, action)
    })
  }

  function isRole(roleCode: string): boolean {
    return roles.value.some(r => r.code === roleCode)
  }

  function markInitialized() {
    isInitialized.value = true
  }

  function reset() {
    permissions.value = []
    roles.value = []
    isInitialized.value = false
  }

  return {
    permissions,
    roles,
    isInitialized,
    setPermissions,
    setRoles,
    hasPermission,
    hasAnyPermission,
    isRole,
    markInitialized,
    reset,
  }
})
```

- [ ] **步骤 4：运行测试验证通过**

```bash
cd qim-admin
npm run test -- tests/unit/stores/permission.test.ts
```
预期：PASS (7 tests)

- [ ] **步骤 5：Commit**

```bash
cd qim-admin
git add src/stores/permission.ts tests/unit/stores/permission.test.ts
git commit -m "feat(store): 添加权限状态管理"
```

---

### 任务 3：权限指令

**文件：**
- 创建：`qim-admin/src/directives/permission.ts`
- 测试：`qim-admin/tests/unit/directives/permission.test.ts`

- [ ] **步骤 1：编写指令测试**

```typescript
// tests/unit/directives/permission.test.ts
import { nextTick } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import type { DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'
import { permissionDirective } from '@/directives/permission'

describe('v-permission directive', () => {
  let store: ReturnType<typeof usePermissionStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    store = usePermissionStore()
  })

  it('should remove element when no permission', async () => {
    store.setPermissions([{ resource: 'user', actions: ['read'] }])
    const parent = document.createElement('div')
    const el = document.createElement('button')
    parent.appendChild(el)
    const binding = { value: 'user:create' } as DirectiveBinding
    permissionDirective.mounted!(el, binding)
    await nextTick()
    expect(el.parentNode).toBeNull()
  })

  it('should keep element when has permission', async () => {
    store.setPermissions([{ resource: 'user', actions: ['create'] }])
    const parent = document.createElement('div')
    const el = document.createElement('button')
    parent.appendChild(el)
    const binding = { value: 'user:create' } as DirectiveBinding
    permissionDirective.mounted!(el, binding)
    await nextTick()
    expect(el.parentNode).toBe(parent)
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

```bash
cd qim-admin
npm run test -- tests/unit/directives/permission.test.ts
```
预期：FAIL - module not found

- [ ] **步骤 3：创建权限指令**

```typescript
// src/directives/permission.ts
import type { Directive, DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'

export const permissionDirective: Directive = {
  mounted(el: HTMLElement, binding: DirectiveBinding<string>) {
    const store = usePermissionStore()
    const permission = binding.value
    const [resource, action] = permission.split(':')

    if (!store.hasPermission(resource, action)) {
      el.parentNode?.removeChild(el)
    }
  }
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
cd qim-admin
npm run test -- tests/unit/directives/permission.test.ts
```
预期：PASS (2 tests)

- [ ] **步骤 5：Commit**

```bash
cd qim-admin
git add src/directives/permission.ts tests/unit/directives/permission.test.ts
git commit -m "feat(directive): 添加 v-permission 权限指令"
```

---

### 任务 4：路由守卫和 403 页面

**文件：**
- 创建：`qim-admin/src/router/guards.ts`
- 创建：`qim-admin/src/views/Forbidden.vue`

- [ ] **步骤 1：创建 403 页面**

```vue
<!-- src/views/Forbidden.vue -->
<template>
  <div class="forbidden-page">
    <div class="forbidden-content">
      <h1 class="error-code">403</h1>
      <h2 class="error-title">权限不足</h2>
      <p class="error-message">您没有访问此页面的权限。请联系管理员获取相应权限。</p>
      <el-button type="primary" @click="router.push('/')">返回首页</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
const router = useRouter()
</script>

<style scoped>
.forbidden-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--color-bg-page);
}
.forbidden-content {
  text-align: center;
}
.error-code {
  font-size: 96px;
  font-weight: 900;
  background: var(--gradient-primary);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin: 0;
}
.error-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: var(--space-4) 0 var(--space-2);
}
.error-message {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: var(--space-6);
}
</style>
```

- [ ] **步骤 2：创建路由守卫**

```typescript
// src/router/guards.ts
import type { RouteLocationNormalized, NavigationGuardNext } from 'vue-router'
import { usePermissionStore } from '@/stores/permission'

export function setupPermissionGuard() {
  return (
    to: RouteLocationNormalized,
    _from: RouteLocationNormalized,
    next: NavigationGuardNext
  ) => {
    const token = localStorage.getItem('token')
    const requiresAuth = to.meta.requiresAuth !== false

    if (requiresAuth && !token) {
      next('/login')
      return
    }

    if (to.path === '/login' && token) {
      next('/')
      return
    }

    if (requiresAuth && token) {
      const permissionStore = usePermissionStore()
      const requiredPermission = to.meta.permission as string | undefined

      if (requiredPermission && !permissionStore.isInitialized) {
        next('/login')
        return
      }

      if (requiredPermission) {
        const [resource, action] = requiredPermission.split(':')
        if (!permissionStore.hasPermission(resource, action)) {
          next('/403')
          return
        }
      }
    }

    next()
  }
}
```

- [ ] **步骤 3：Commit**

```bash
cd qim-admin
git add src/views/Forbidden.vue src/router/guards.ts
git commit -m "feat(router): 添加权限路由守卫和 403 页面"
```

---

### 任务 5：增强请求拦截器

**文件：**
- 修改：`qim-admin/src/utils/request.ts`

- [ ] **步骤 1：增强 request.ts，添加权限集成**

将现有的 `request.ts` 替换为：

```typescript
import axios from 'axios'
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiResponse } from '@/types'
import { usePermissionStore } from '@/stores/permission'

const service: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

service.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    console.error('[Request] error:', error)
    return Promise.reject(error)
  }
)

service.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const res = response.data
    if (res.code !== 0) {
      ElMessage.error(res.message || '请求失败')
      if (res.code === 401) {
        const permStore = usePermissionStore()
        permStore.reset()
        localStorage.removeItem('token')
        window.location.href = '/login'
      }
      return Promise.reject(new Error(res.message || '请求失败'))
    }
    return response
  },
  (error) => {
    console.error('[Response] error:', error)
    const status = error.response?.status
    
    if (status === 401) {
      const permStore = usePermissionStore()
      permStore.reset()
      localStorage.removeItem('token')
      window.location.href = '/login'
    } else if (status === 403) {
      ElMessage.error('权限不足，无法执行此操作')
    } else {
      const message = error.response?.data?.message || error.message || '网络异常'
      ElMessage.error(message)
    }
    
    return Promise.reject(error)
  }
)

export default service

export const request = <T = unknown>(config: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return service(config)
}
```

- [ ] **步骤 2：Commit**

```bash
cd qim-admin
git add src/utils/request.ts
git commit -m "refactor(request): 增强错误处理和权限集成"
```

---

### 任务 6：usePermission Composable

**文件：**
- 创建：`qim-admin/src/composables/usePermission.ts`
- 测试：`qim-admin/tests/unit/composables/usePermission.test.ts`

- [ ] **步骤 1：编写测试**

```typescript
// tests/unit/composables/usePermission.test.ts
import { createPinia, setActivePinia } from 'pinia'
import { usePermission } from '@/composables/usePermission'

describe('usePermission composable', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('canAccess returns true for permitted action', () => {
    const { canAccess } = usePermission()
    const permStore = usePermissionStore()
    permStore.setPermissions([{ resource: 'user', actions: ['create', 'read'] }])
    expect(canAccess('user:create')).toBe(true)
    expect(canAccess('user:delete')).toBe(false)
  })

  it('canAccessAny returns true if any permission matches', () => {
    const { canAccessAny } = usePermission()
    const permStore = usePermissionStore()
    permStore.setPermissions([{ resource: 'user', actions: ['read'] }])
    expect(canAccessAny(['user:delete', 'user:read'])).toBe(true)
    expect(canAccessAny(['user:delete', 'group:create'])).toBe(false)
  })

  it('isAdmin returns true for system_admin', () => {
    const { isAdmin } = usePermission()
    const permStore = usePermissionStore()
    permStore.setRoles([{ id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }])
    expect(isAdmin()).toBe(true)
  })
})
```

需要添加 import: `import { usePermissionStore } from '@/stores/permission'`

- [ ] **步骤 2：运行测试验证失败**

```bash
cd qim-admin
npm run test -- tests/unit/composables/usePermission.test.ts
```
预期：FAIL - module not found

- [ ] **步骤 3：创建 usePermission composable**

```typescript
// src/composables/usePermission.ts
import { usePermissionStore } from '@/stores/permission'

export function usePermission() {
  const store = usePermissionStore()

  function canAccess(permission: string): boolean {
    const [resource, action] = permission.split(':')
    return store.hasPermission(resource, action)
  }

  function canAccessAny(permissions: string[]): boolean {
    return store.hasAnyPermission(permissions)
  }

  function isAdmin(): boolean {
    return store.isRole('system_admin')
  }

  return { canAccess, canAccessAny, isAdmin }
}
```

- [ ] **步骤 4：运行测试验证通过**

```bash
cd qim-admin
npm run test -- tests/unit/composables/usePermission.test.ts
```
预期：PASS (3 tests)

- [ ] **步骤 5：Commit**

```bash
cd qim-admin
git add src/composables/usePermission.ts tests/unit/composables/usePermission.test.ts
git commit -m "feat(composable): 添加 usePermission 组合式函数"
```

---

## 阶段 2 - 核心组件

### 任务 7：DataTable 组件

**文件：**
- 创建：`qim-admin/src/components/data/DataTable.vue`

- [ ] **步骤 1：创建 DataTable 组件**

```vue
<!-- src/components/data/DataTable.vue -->
<template>
  <div class="data-table">
    <div class="table-header">
      <div class="search-area">
        <slot name="search"></slot>
      </div>
      <div class="actions-area">
        <slot name="actions"></slot>
        <el-button type="primary" @click="$emit('refresh')">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <el-table :data="data" v-loading="loading" stripe>
      <slot></slot>
    </el-table>

    <div class="pagination-area" v-if="showPagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="pageSizes"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="$emit('page-change')"
        @current-change="$emit('page-change')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Refresh } from '@element-plus/icons-vue'

interface PaginationConfig {
  page: number
  pageSize: number
  total: number
}

interface Props {
  data: unknown[]
  loading: boolean
  pagination: PaginationConfig
  showPagination?: boolean
  pageSizes?: number[]
}

withDefaults(defineProps<Props>(), {
  showPagination: true,
  pageSizes: () => [10, 20, 50, 100]
})

const emit = defineEmits<{
  'page-change': []
  'refresh': []
}>()

const currentPage = computed({
  get: () => props.pagination.page,
  set: (val: number) => {}
})

const pageSize = computed({
  get: () => props.pagination.pageSize,
  set: (val: number) => {}
})

const total = computed(() => props.pagination.total)
</script>

<style scoped>
.data-table {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  padding: var(--space-6);
  box-shadow: var(--shadow-card);
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--space-5);
  gap: var(--space-4);
  flex-wrap: wrap;
}

.search-area {
  flex: 1;
  min-width: 0;
}

.actions-area {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-shrink: 0;
}

.pagination-area {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
cd qim-admin
git add src/components/data/DataTable.vue
git commit -m "feat(component): 添加 DataTable 数据表格组件"
```

---

### 任务 8：SearchForm 和 SearchField 组件

**文件：**
- 创建：`qim-admin/src/components/data/SearchForm.vue`
- 创建：`qim-admin/src/components/data/SearchField.vue`

- [ ] **步骤 1：创建 SearchField 组件**

```vue
<!-- src/components/data/SearchField.vue -->
<template>
  <el-form-item :label="label">
    <el-input
      v-if="type === 'input'"
      :model-value="modelValue"
      :placeholder="placeholder"
      clearable
      @update:model-value="$emit('update:modelValue', $event)"
      @keyup.enter="$emit('search')"
    />
    <el-select
      v-else-if="type === 'select'"
      :model-value="modelValue"
      :placeholder="placeholder"
      clearable
      @update:model-value="$emit('update:modelValue', $event)"
    >
      <el-option
        v-for="opt in options"
        :key="opt.value"
        :label="opt.label"
        :value="opt.value"
      />
    </el-select>
  </el-form-item>
</template>

<script setup lang="ts">
interface Option {
  label: string
  value: string | number
}

interface Props {
  modelValue?: string | number
  label: string
  type?: 'input' | 'select'
  placeholder?: string
  options?: Option[]
}

withDefaults(defineProps<Props>(), {
  type: 'input',
  placeholder: '请输入',
  options: () => []
})

defineEmits<{
  'update:modelValue': [value: string | number]
  'search': []
}>()
</script>
```

- [ ] **步骤 2：创建 SearchForm 组件**

```vue
<!-- src/components/data/SearchForm.vue -->
<template>
  <el-form :model="{}" inline>
    <slot></slot>
    <el-form-item>
      <el-button type="primary" @click="$emit('search')">搜索</el-button>
      <el-button @click="$emit('reset')">重置</el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
defineEmits<{
  'search': []
  'reset': []
}>()
</script>
```

- [ ] **步骤 3：Commit**

```bash
cd qim-admin
git add src/components/data/SearchForm.vue src/components/data/SearchField.vue
git commit -m "feat(component): 添加 SearchForm 和 SearchField 组件"
```

---

### 任务 9：StatusTag 和 ActionButton 组件

**文件：**
- 创建：`qim-admin/src/components/data/StatusTag.vue`
- 创建：`qim-admin/src/components/common/ActionButton.vue`

- [ ] **步骤 1：创建 StatusTag 组件**

```vue
<!-- src/components/data/StatusTag.vue -->
<template>
  <el-tag :type="tagType" size="small" round>
    {{ label }}
  </el-tag>
</template>

<script setup lang="ts">
interface Props {
  status: string
  map?: Record<string, { label: string; type: 'success' | 'warning' | 'danger' | 'info' }>
}

const props = withDefaults(defineProps<Props>(), {
  map: () => ({
    active: { label: '正常', type: 'success' },
    inactive: { label: '停用', type: 'info' },
    banned: { label: '封禁', type: 'danger' },
    published: { label: '已发布', type: 'success' },
    draft: { label: '草稿', type: 'info' },
  })
})

const tagType = computed(() => props.map[props.status]?.type || 'info')
const label = computed(() => props.map[props.status]?.label || props.status)
</script>

<script lang="ts">
import { computed } from 'vue'
export default {}
</script>
```

修正：将 computed 合并到 setup 中：

```vue
<!-- src/components/data/StatusTag.vue -->
<template>
  <el-tag :type="tagType" size="small" round>
    {{ label }}
  </el-tag>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface StatusMap {
  [key: string]: { label: string; type: 'success' | 'warning' | 'danger' | 'info' }
}

interface Props {
  status: string
  map?: StatusMap
}

const props = withDefaults(defineProps<Props>(), {
  map: () => ({
    active: { label: '正常', type: 'success' },
    inactive: { label: '停用', type: 'info' },
    banned: { label: '封禁', type: 'danger' },
    published: { label: '已发布', type: 'success' },
    draft: { label: '草稿', type: 'info' },
  })
})

const tagType = computed(() => props.map[props.status]?.type || 'info')
const label = computed(() => props.map[props.status]?.label || props.status)
</script>
```

- [ ] **步骤 2：创建 ActionButton 组件**

```vue
<!-- src/components/common/ActionButton.vue -->
<template>
  <el-button :type="type" size="small" @click="$emit('click', $event)">
    <slot></slot>
  </el-button>
</template>

<script setup lang="ts">
interface Props {
  type?: 'primary' | 'success' | 'warning' | 'danger' | 'info'
}

withDefaults(defineProps<Props>(), {
  type: 'default'
})

defineEmits<{
  'click': [event: MouseEvent]
}>()
</script>
```

- [ ] **步骤 3：Commit**

```bash
cd qim-admin
git add src/components/data/StatusTag.vue src/components/common/ActionButton.vue
git commit -m "feat(component): 添加 StatusTag 和 ActionButton 组件"
```

---

### 任务 10：EntityDialog 组件

**文件：**
- 创建：`qim-admin/src/components/forms/EntityDialog.vue`
- 创建：`qim-admin/src/components/forms/FieldRenderer.vue`

- [ ] **步骤 1：创建 FieldRenderer 组件**

```vue
<!-- src/components/forms/FieldRenderer.vue -->
<template>
  <el-form-item :label="field.label" :prop="field.name" :rules="rules">
    <el-input
      v-if="field.type === 'input'"
      v-model="model[field.name]"
      v-bind="field.props"
    />
    <el-input
      v-else-if="field.type === 'textarea'"
      v-model="model[field.name]"
      type="textarea"
      v-bind="field.props"
    />
    <el-input
      v-else-if="field.type === 'password'"
      v-model="model[field.name]"
      type="password"
      show-password
      v-bind="field.props"
    />
    <el-select
      v-else-if="field.type === 'select'"
      v-model="model[field.name]"
      v-bind="field.props"
    >
      <el-option
        v-for="opt in field.options"
        :key="opt.value"
        :label="opt.label"
        :value="opt.value"
      />
    </el-select>
    <el-switch
      v-else-if="field.type === 'switch'"
      v-model="model[field.name]"
      v-bind="field.props"
    />
    <el-input-number
      v-else-if="field.type === 'number'"
      v-model="model[field.name]"
      v-bind="field.props"
    />
  </el-form-item>
</template>

<script setup lang="ts">
import type { FormItemRule } from 'element-plus'

interface FieldOption {
  label: string
  value: string | number | boolean
}

interface FormField {
  name: string
  label: string
  type: 'input' | 'textarea' | 'password' | 'select' | 'switch' | 'number'
  props?: Record<string, unknown>
  options?: FieldOption[]
  required?: boolean
}

interface Props {
  field: FormField
  model: Record<string, unknown>
  rules?: FormItemRule[]
}

defineProps<Props>()
</script>
```

- [ ] **步骤 2：创建 EntityDialog 组件**

```vue
<!-- src/components/forms/EntityDialog.vue -->
<template>
  <el-dialog
    :model-value="modelValue"
    :title="mode === 'create' ? createTitle : editTitle"
    width="500px"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="80px"
    >
      <FieldRenderer
        v-for="field in fields"
        :key="field.name"
        :field="field"
        :model="formData"
      />
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="handleSave">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import FieldRenderer from './FieldRenderer.vue'

interface FieldOption {
  label: string
  value: string | number | boolean
}

interface FormField {
  name: string
  label: string
  type: 'input' | 'textarea' | 'password' | 'select' | 'switch' | 'number'
  props?: Record<string, unknown>
  options?: FieldOption[]
  required?: boolean
}

interface Props {
  modelValue: boolean
  mode: 'create' | 'edit'
  fields: FormField[]
  rules?: FormRules
  initialData?: Record<string, unknown>
  createTitle?: string
  editTitle?: string
}

const props = withDefaults(defineProps<Props>(), {
  rules: () => ({}),
  initialData: () => ({}),
  createTitle: '创建',
  editTitle: '编辑',
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'save': [data: Record<string, unknown>]
}>()

const formRef = ref<FormInstance>()
const formData = ref<Record<string, unknown>>({})
const loading = ref(false)

watch(
  () => props.modelValue,
  (val) => {
    if (val) {
      formData.value = { ...props.initialData }
    }
  }
)

async function handleSave() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    emit('save', { ...formData.value })
  } finally {
    loading.value = false
  }
}
</script>
```

- [ ] **步骤 3：Commit**

```bash
cd qim-admin
git add src/components/forms/EntityDialog.vue src/components/forms/FieldRenderer.vue
git commit -m "feat(component): 添加 EntityDialog 和 FieldRenderer 表单组件"
```

---

### 任务 11：useEntity Composable

**文件：**
- 创建：`qim-admin/src/composables/useEntity.ts`

- [ ] **步骤 1：创建 useEntity composable**

```typescript
// src/composables/useEntity.ts
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

interface PaginationState {
  page: number
  pageSize: number
  total: number
}

interface EntityAPI<T> {
  getList: (params: Record<string, unknown>) => Promise<{ data: { data: { list: T[]; total: number } } }>
  create: (data: Record<string, unknown>) => Promise<void>
  update: (id: number, data: Record<string, unknown>) => Promise<void>
  delete: (id: number) => Promise<void>
}

interface FormField {
  name: string
  label: string
  type: 'input' | 'textarea' | 'password' | 'select' | 'switch' | 'number'
  props?: Record<string, unknown>
  options?: Array<{ label: string; value: unknown }>
  required?: boolean
}

interface UseEntityOptions<T> {
  api: EntityAPI<T>
  searchFields: string[]
  formFields: FormField[]
  successMessages?: {
    create?: string
    update?: string
    delete?: string
  }
}

export function useEntity<T extends { id: number }>(options: UseEntityOptions<T>) {
  const { api, searchFields, formFields, successMessages = {} } = options

  // 列表状态
  const list = ref<T[]>([])
  const loading = ref(false)
  const pagination = reactive<PaginationState>({ page: 1, pageSize: 10, total: 0 })

  // 搜索
  const searchForm = reactive<Record<string, unknown>>({})
  for (const field of searchFields) {
    searchForm[field] = ''
  }

  // 对话框
  const dialogVisible = ref(false)
  const dialogMode = ref<'create' | 'edit'>('create')
  const currentRow = ref<T | null>(null)
  const formData = ref<Record<string, unknown>>({})
  const submitting = ref(false)

  // 获取数据
  async function fetchData() {
    loading.value = true
    try {
      const params: Record<string, unknown> = {
        page: pagination.page,
        pageSize: pagination.pageSize,
      }
      for (const field of searchFields) {
        if (searchForm[field]) {
          params[field] = searchForm[field]
        }
      }
      const { data } = await api.getList(params)
      list.value = data.data.list
      pagination.total = data.data.total
    } catch (error) {
      console.error('[useEntity] fetch failed:', error)
    } finally {
      loading.value = false
    }
  }

  // 搜索
  function handleSearch() {
    pagination.page = 1
    fetchData()
  }

  function handleReset() {
    for (const field of searchFields) {
      searchForm[field] = ''
    }
    handleSearch()
  }

  // 创建
  function handleCreate() {
    dialogMode.value = 'create'
    currentRow.value = null
    formData.value = {}
    dialogVisible.value = true
  }

  // 编辑
  function handleEdit(row: T) {
    dialogMode.value = 'edit'
    currentRow.value = row
    formData.value = { ...row }
    dialogVisible.value = true
  }

  // 删除
  async function handleDelete(id: number) {
    try {
      await ElMessageBox.confirm('确定删除吗？', '确认', {
        type: 'warning',
      })
      await api.delete(id)
      ElMessage.success(successMessages.delete || '删除成功')
      fetchData()
    } catch (error) {
      if (error !== 'cancel') {
        console.error('[useEntity] delete failed:', error)
      }
    }
  }

  // 保存
  async function handleSave(data: Record<string, unknown>) {
    submitting.value = true
    try {
      if (dialogMode.value === 'create') {
        await api.create(data)
        ElMessage.success(successMessages.create || '创建成功')
      } else if (currentRow.value) {
        await api.update(currentRow.value.id, data)
        ElMessage.success(successMessages.update || '更新成功')
      }
      dialogVisible.value = false
      fetchData()
    } catch (error) {
      console.error('[useEntity] save failed:', error)
    } finally {
      submitting.value = false
    }
  }

  return {
    list,
    loading,
    pagination,
    searchForm,
    dialogVisible,
    dialogMode,
    currentRow,
    formData,
    submitting,
    formFields,
    handleSearch,
    handleReset,
    handleCreate,
    handleEdit,
    handleDelete,
    handleSave,
    fetchData,
  }
}
```

- [ ] **步骤 2：Commit**

```bash
cd qim-admin
git add src/composables/useEntity.ts
git commit -m "feat(composable): 添加 useEntity CRUD 组合式函数"
```

---

### 任务 12：布局组件拆分

**文件：**
- 创建：`qim-admin/src/components/layout/Sidebar/index.vue`
- 创建：`qim-admin/src/components/layout/Header/index.vue`
- 创建：`qim-admin/src/components/layout/Header/ThemeToggle.vue`
- 创建：`qim-admin/src/components/layout/Header/UserDropdown.vue`
- 创建：`qim-admin/src/components/layout/Breadcrumb/index.vue`
- 修改：`qim-admin/src/layouts/AdminLayout.vue`

- [ ] **步骤 1：创建 Sidebar 组件**

从现有 AdminLayout.vue 的侧边栏部分提取：

```vue
<!-- src/components/layout/Sidebar/index.vue -->
<template>
  <el-aside :width="collapsed ? '64px' : '240px'" class="sidebar" :class="{ 'is-collapsed': collapsed }">
    <div class="logo-container">
      <div class="logo-icon">
        <el-icon :size="28"><ChatDotRound /></el-icon>
      </div>
      <h2 class="logo-text" v-show="!collapsed">QIM Admin</h2>
    </div>

    <div class="menu-wrapper">
      <el-menu :default-active="activeMenu" :collapse="collapsed" router class="sidebar-menu">
        <el-sub-menu index="dashboard-group">
          <template #title>
            <el-icon><DataAnalysis /></el-icon>
            <span>数据概览</span>
          </template>
          <el-menu-item index="/">
            <el-icon><HomeFilled /></el-icon>
            <template #title>仪表盘</template>
          </el-menu-item>
          <el-menu-item index="/statistics">
            <el-icon><TrendCharts /></el-icon>
            <template #title>数据统计</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="user-group">
          <template #title>
            <el-icon><User /></el-icon>
            <span>用户与组织</span>
          </template>
          <el-menu-item index="/users" v-permission="'user:read'">
            <el-icon><User /></el-icon>
            <template #title>用户管理</template>
          </el-menu-item>
          <el-menu-item index="/organization" v-permission="'organization:read'">
            <el-icon><School /></el-icon>
            <template #title>组织架构</template>
          </el-menu-item>
          <el-menu-item index="/roles" v-permission="'role:read'">
            <el-icon><Key /></el-icon>
            <template #title>角色权限</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="chat-group">
          <template #title>
            <el-icon><ChatDotRound /></el-icon>
            <span>会话与群组</span>
          </template>
          <el-menu-item index="/groups" v-permission="'group:read'">
            <el-icon><UserFilled /></el-icon>
            <template #title>群组管理</template>
          </el-menu-item>
          <el-menu-item index="/conversations" v-permission="'conversation:read'">
            <el-icon><ChatLineSquare /></el-icon>
            <template #title>会话管理</template>
          </el-menu-item>
          <el-menu-item index="/channels" v-permission="'channel:read'">
            <el-icon><Connection /></el-icon>
            <template #title>频道管理</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="app-group">
          <template #title>
            <el-icon><Grid /></el-icon>
            <span>应用生态</span>
          </template>
          <el-menu-item index="/apps" v-permission="'app:read'">
            <el-icon><Monitor /></el-icon>
            <template #title>应用管理</template>
          </el-menu-item>
          <el-menu-item index="/mini-apps" v-permission="'miniapp:read'">
            <el-icon><Cellphone /></el-icon>
            <template #title>小程序管理</template>
          </el-menu-item>
          <el-menu-item index="/ai-assistant" v-permission="'ai:read'">
            <el-icon><Cpu /></el-icon>
            <template #title>AI 助手</template>
          </el-menu-item>
          <el-menu-item index="/ai-ops" v-permission="'ai:read'">
            <el-icon><Monitor /></el-icon>
            <template #title>AI 运维面板</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="msg-group">
          <template #title>
            <el-icon><Bell /></el-icon>
            <span>消息与通知</span>
          </template>
          <el-menu-item index="/messages" v-permission="'message:read'">
            <el-icon><ChatDotRound /></el-icon>
            <template #title>系统消息</template>
          </el-menu-item>
          <el-menu-item index="/notifications" v-permission="'notification:read'">
            <el-icon><BellFilled /></el-icon>
            <template #title>通知管理</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="security-group">
          <template #title>
            <el-icon><Lock /></el-icon>
            <span>安全与合规</span>
          </template>
          <el-menu-item index="/blacklist" v-permission="'blacklist:read'">
            <el-icon><CircleCloseFilled /></el-icon>
            <template #title>黑名单管理</template>
          </el-menu-item>
          <el-menu-item index="/sensitive-words" v-permission="'sensitive:read'">
            <el-icon><Warning /></el-icon>
            <template #title>敏感词管理</template>
          </el-menu-item>
          <el-menu-item index="/operation-logs" v-permission="'log:read'">
            <el-icon><Document /></el-icon>
            <template #title>操作日志</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="system-group">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>系统设置</span>
          </template>
          <el-menu-item index="/system-config" v-permission="'config:read'">
            <el-icon><Tools /></el-icon>
            <template #title>系统配置</template>
          </el-menu-item>
          <el-menu-item index="/version-management" v-permission="'version:read'">
            <el-icon><Upload /></el-icon>
            <template #title>版本管理</template>
          </el-menu-item>
        </el-sub-menu>
      </el-menu>
    </div>

    <button class="collapse-btn" @click="$emit('toggle')" :title="collapsed ? '展开' : '收起'">
      <el-icon :size="18">
        <Fold v-if="!collapsed" />
        <Expand v-else />
      </el-icon>
    </button>
  </el-aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  HomeFilled, User, UserFilled, ChatDotRound, Bell,
  CircleCloseFilled, TrendCharts, School, ChatLineSquare,
  Connection, Grid, Monitor, Cellphone, BellFilled,
  Fold, Expand, DataAnalysis, Key, Cpu, Warning, Document,
  Lock, Setting, Tools, Upload,
} from '@element-plus/icons-vue'

defineEmits<{
  'toggle': []
}>()

defineProps<{
  collapsed: boolean
}>()

const route = useRoute()
const activeMenu = computed(() => route.path)
</script>

<style scoped>
.sidebar {
  background: var(--sidebar-bg);
  overflow: hidden;
  position: relative;
  transition: width var(--duration-normal) var(--ease-out);
  box-shadow: 4px 0 16px rgba(0, 0, 0, 0.08);
  z-index: 10;
}

.logo-container {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 0 var(--space-4);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.logo-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--gradient-primary);
  border-radius: var(--radius-md);
  color: white;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(14, 165, 233, 0.25);
}

.logo-text {
  color: white;
  font-size: 18px;
  font-weight: 800;
  margin: 0;
  white-space: nowrap;
  letter-spacing: -0.02em;
}

.menu-wrapper {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: var(--space-2) 0;
}

.sidebar-menu {
  background: transparent !important;
  border-right: none !important;
}

:deep(.el-menu-item),
:deep(.el-sub-menu__title) {
  color: var(--sidebar-text) !important;
  height: 44px !important;
  line-height: 44px !important;
  font-weight: 500 !important;
}

:deep(.el-menu-item:hover),
:deep(.el-sub-menu__title:hover) {
  background-color: rgba(255, 255, 255, 0.06) !important;
  color: var(--sidebar-text-active) !important;
}

:deep(.el-menu-item.is-active) {
  background: rgba(255, 255, 255, 0.1) !important;
  color: var(--sidebar-text-active) !important;
  font-weight: 700 !important;
}

:deep(.el-sub-menu .el-menu-item) {
  min-width: auto !important;
  margin: 2px 8px !important;
  background: rgba(255, 255, 255, 0.03) !important;
  border-radius: var(--radius-sm) !important;
}

:deep(.el-sub-menu .el-menu-item:hover) {
  background: rgba(255, 255, 255, 0.08) !important;
}

:deep(.el-sub-menu .el-menu-item.is-active) {
  background: rgba(255, 255, 255, 0.15) !important;
  color: white !important;
}

:deep(.el-sub-menu .el-menu) {
  background: rgba(0, 0, 0, 0.12) !important;
  border-radius: var(--radius-lg);
  margin: 4px 8px;
}

.collapse-btn {
  position: absolute;
  bottom: var(--space-4);
  left: 50%;
  transform: translateX(-50%);
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: var(--radius-md);
  color: rgba(255, 255, 255, 0.7);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

.collapse-btn:hover {
  background: rgba(255, 255, 255, 0.15);
  color: white;
  transform: translateX(-50%) scale(1.05);
}
</style>
```

- [ ] **步骤 2：创建 Header 组件**

```vue
<!-- src/components/layout/Header/index.vue -->
<template>
  <el-header class="admin-header">
    <div class="header-left">
      <slot name="breadcrumb"></slot>
    </div>
    <div class="header-right">
      <ThemeToggle />
      <UserDropdown />
    </div>
  </el-header>
</template>

<script setup lang="ts">
import ThemeToggle from './ThemeToggle.vue'
import UserDropdown from './UserDropdown.vue'
</script>

<style scoped>
.admin-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  background-color: var(--color-surface);
  padding: 0 var(--space-6);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
</style>
```

- [ ] **步骤 3：创建 ThemeToggle 组件**

```vue
<!-- src/components/layout/Header/ThemeToggle.vue -->
<template>
  <button class="theme-toggle" @click="toggleTheme" :title="isDark ? '切换到亮色' : '切换到暗色'">
    <el-icon :size="20">
      <Sunny v-if="isDark" />
      <Moon v-else />
    </el-icon>
  </button>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Sunny, Moon } from '@element-plus/icons-vue'

const isDark = ref(false)

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.setAttribute('data-theme', isDark.value ? 'dark' : 'light')
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  const saved = localStorage.getItem('theme')
  if (saved === 'dark') {
    isDark.value = true
    document.documentElement.setAttribute('data-theme', 'dark')
  }
})
</script>

<style scoped>
.theme-toggle {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

.theme-toggle:hover {
  background-color: var(--color-primary-lighter);
  color: var(--color-primary);
  border-color: var(--color-primary);
  transform: scale(1.05);
}
</style>
```

- [ ] **步骤 4：创建 UserDropdown 组件**

```vue
<!-- src/components/layout/Header/UserDropdown.vue -->
<template>
  <el-dropdown trigger="click" @command="handleCommand">
    <span class="user-dropdown">
      <el-avatar :size="34">
        {{ authStore.user?.username?.charAt(0) || 'A' }}
      </el-avatar>
      <span class="username">{{ authStore.user?.username || 'Admin' }}</span>
      <el-icon :size="14"><ArrowDown /></el-icon>
    </span>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item command="logout">
          <el-icon><SwitchButton /></el-icon>
          <span>退出登录</span>
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { ArrowDown, SwitchButton } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

function handleCommand(command: string) {
  if (command === 'logout') {
    authStore.logout()
    router.push('/login')
  }
}
</script>

<style scoped>
.user-dropdown {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-lg);
  transition: background-color var(--duration-fast) var(--ease-out);
}

.user-dropdown:hover {
  background-color: var(--color-surface-hover);
}

.username {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

@media (max-width: 768px) {
  .username { display: none; }
}
</style>
```

- [ ] **步骤 5：创建 Breadcrumb 组件**

```vue
<!-- src/components/layout/Breadcrumb/index.vue -->
<template>
  <el-breadcrumb separator="/">
    <el-breadcrumb-item :to="{ path: '/' }">首页</el-breadcrumb-item>
    <el-breadcrumb-item>{{ title }}</el-breadcrumb-item>
  </el-breadcrumb>
</template>

<script setup lang="ts">
defineProps<{
  title: string
}>()
</script>
```

- [ ] **步骤 6：重构 AdminLayout.vue**

将现有的 AdminLayout.vue 替换为使用新组件：

```vue
<!-- src/layouts/AdminLayout.vue -->
<template>
  <el-container class="admin-layout">
    <Sidebar :collapsed="isCollapsed" @toggle="isCollapsed = !isCollapsed" />
    <el-container class="main-container">
      <Header>
        <template #breadcrumb>
          <Breadcrumb :title="currentTitle" />
        </template>
      </Header>
      <el-main class="admin-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import Sidebar from '@/components/layout/Sidebar/index.vue'
import Header from '@/components/layout/Header/index.vue'
import Breadcrumb from '@/components/layout/Breadcrumb/index.vue'

const route = useRoute()
const isCollapsed = ref(false)

const titleMap: Record<string, string> = {
  '/': '仪表盘',
  '/statistics': '数据统计',
  '/users': '用户管理',
  '/organization': '组织架构',
  '/roles': '角色权限',
  '/groups': '群组管理',
  '/conversations': '会话管理',
  '/channels': '频道管理',
  '/apps': '应用管理',
  '/mini-apps': '小程序管理',
  '/ai-assistant': 'AI 助手',
  '/ai-ops': 'AI 运维面板',
  '/messages': '系统消息',
  '/notifications': '通知管理',
  '/blacklist': '黑名单管理',
  '/sensitive-words': '敏感词管理',
  '/operation-logs': '操作日志',
  '/system-config': '系统配置',
  '/version-management': '版本管理',
}

const currentTitle = computed(() => titleMap[route.path] || '仪表盘')
</script>

<style scoped>
.admin-layout {
  height: 100vh;
  background-color: var(--color-bg-page);
}

.main-container {
  background-color: var(--color-bg-page);
}

.admin-main {
  background-color: var(--color-bg-page);
  padding: var(--space-6);
  overflow-y: auto;
}
</style>
```

- [ ] **步骤 7：Commit**

```bash
cd qim-admin
git add src/components/layout/ src/layouts/AdminLayout.vue
git commit -m "refactor(layout): 拆分 AdminLayout 为独立组件"
```

---

## 阶段 3 - 页面重构

### 任务 13：重构 Dashboard

**文件：**
- 创建：`qim-admin/src/views/Dashboard/index.vue`
- 删除：`qim-admin/src/views/Dashboard.vue`

- [ ] **步骤 1：创建新的 Dashboard**

将现有 Dashboard.vue 移动为 views/Dashboard/index.vue，保持原有逻辑不变（Dashboard 不需要 DataTable）。

```bash
cd qim-admin
mkdir -p src/views/Dashboard
mv src/views/Dashboard.vue src/views/Dashboard/index.vue
```

更新 import 路径确保组件引用正确。

- [ ] **步骤 2：验证构建**

```bash
cd qim-admin
npm run build
```
预期：PASS - 无错误

- [ ] **步骤 3：Commit**

```bash
cd qim-admin
git add src/views/Dashboard/
git rm src/views/Dashboard.vue
git commit -m "refactor(view): 重构 Dashboard 为文件夹结构"
```

---

### 任务 14：重构 UserManagement

**文件：**
- 创建：`qim-admin/src/views/UserManagement/index.vue`
- 创建：`qim-admin/src/views/UserManagement/config.ts`
- 删除：`qim-admin/src/views/Users.vue`

- [ ] **步骤 1：创建 config.ts**

```typescript
// src/views/UserManagement/config.ts
import type { FormRules } from 'element-plus'
import type { FormField } from '@/composables/useEntity'

export const userColumns = [
  { prop: 'id', label: 'ID', width: '80' },
  { label: '用户名', minWidth: '150', slot: 'user-info' },
  { prop: 'email', label: '邮箱', minWidth: '180' },
  { prop: 'phone', label: '手机号', minWidth: '140' },
  { label: '角色', minWidth: '200', slot: 'roles' },
  { label: '状态', width: '100', slot: 'status' },
  { prop: 'createdAt', label: '创建时间', width: '180' },
  { label: '操作', width: '200', slot: 'actions', fixed: 'right' },
]

export const userFields: FormField[] = [
  { name: 'username', label: '用户名', type: 'input', required: true, props: { disabled: true } },
  { name: 'password', label: '密码', type: 'password', required: true, props: { showPassword: true } },
  { name: 'nickname', label: '昵称', type: 'input' },
  { name: 'email', label: '邮箱', type: 'input', required: true },
  { name: 'avatar', label: '头像', type: 'input', props: { placeholder: '请输入头像URL' } },
  { name: 'phone', label: '手机号', type: 'input' },
]

export const userRules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效邮箱', trigger: 'blur' },
  ],
}

export const roleOptions = [
  { label: '系统管理员', value: 'system_admin' },
  { label: '系统发布者', value: 'system_publisher' },
  { label: '系统审核员', value: 'system_moderator' },
  { label: '系统运营', value: 'system_operator' },
]
```

- [ ] **步骤 2：创建 index.vue**

```vue
<!-- src/views/UserManagement/index.vue -->
<template>
  <DataTable :data="list" :loading="loading" :pagination="pagination"
    @search="handleSearch" @page-change="fetchData" @refresh="fetchData">
    <template #search>
      <SearchForm @search="handleSearch" @reset="handleReset">
        <SearchField v-model="searchForm.keyword" label="关键词" placeholder="用户名或昵称" />
      </SearchForm>
    </template>
    <template #actions>
      <el-button v-permission="'user:create'" type="success" @click="handleCreate">创建用户</el-button>
    </template>

    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column label="用户名" min-width="150">
      <template #default="{ row }">
        <div class="user-cell">
          <el-avatar :size="32" :src="row.avatar">{{ (row.nickname || row.username).charAt(0) }}</el-avatar>
          <div class="user-info">
            <span class="username">{{ row.username }}</span>
            <span class="nickname">{{ row.nickname || '-' }}</span>
          </div>
        </div>
      </template>
    </el-table-column>
    <el-table-column prop="email" label="邮箱" min-width="180" />
    <el-table-column prop="phone" label="手机号" min-width="140" />
    <el-table-column label="角色" min-width="200">
      <template #default="{ row }">
        <el-tag v-for="role in (row.roles || [])" :key="role" size="small" class="role-tag">
          {{ roleLabel(role) }}
        </el-tag>
        <span v-if="!row.roles || row.roles.length === 0" class="text-muted">未分配</span>
      </template>
    </el-table-column>
    <el-table-column label="状态" width="100">
      <template #default="{ row }">
        <StatusTag :status="row.status" />
      </template>
    </el-table-column>
    <el-table-column prop="createdAt" label="创建时间" width="180" />
    <el-table-column label="操作" width="260" fixed="right">
      <template #default="{ row }">
        <ActionButton v-permission="'user:update'" @click="handleEdit(row)">编辑</ActionButton>
        <ActionButton v-permission="'user:update'" @click="handleManageRoles(row)">管理角色</ActionButton>
        <el-popconfirm title="确定删除该用户吗？" @confirm="handleDelete(row.id)">
          <template #reference>
            <ActionButton v-permission="'user:delete'" type="danger">删除</ActionButton>
          </template>
        </el-popconfirm>
      </template>
    </el-table-column>
  </DataTable>

  <EntityDialog
    v-model="dialogVisible"
    :mode="dialogMode"
    :fields="userFields"
    :rules="userRules"
    :initial-data="formData"
    :create-title="'创建用户'"
    :edit-title="'编辑用户'"
    @save="handleSave"
  />

  <el-dialog v-model="roleDialogVisible" title="管理角色" width="400px">
    <p class="role-dialog-hint">为用户 <strong>{{ currentUser?.username }}</strong> 分配角色</p>
    <el-checkbox-group v-model="selectedRoles">
      <el-checkbox v-for="role in roleOptions" :key="role.value" :label="role.value">
        {{ role.label }}
      </el-checkbox>
    </el-checkbox-group>
    <template #footer>
      <el-button @click="roleDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSaveRoles">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import DataTable from '@/components/data/DataTable.vue'
import SearchForm from '@/components/data/SearchForm.vue'
import SearchField from '@/components/data/SearchField.vue'
import StatusTag from '@/components/data/StatusTag.vue'
import ActionButton from '@/components/common/ActionButton.vue'
import EntityDialog from '@/components/forms/EntityDialog.vue'
import { useEntity } from '@/composables/useEntity'
import { getUsers, createUser, updateUser, deleteUser, assignRoles } from '@/api/users'
import { userFields, userRules, roleOptions } from './config'
import type { User } from '@/types'

const userApi = {
  getList: (params: Record<string, unknown>) => getUsers(params as any),
  create: (data: Record<string, unknown>) => createUser(data as any),
  update: (id: number, data: Record<string, unknown>) => updateUser(id, data as any),
  delete: (id: number) => deleteUser(id),
}

const {
  list, loading, pagination, searchForm,
  dialogVisible, dialogMode, formData, submitting,
  handleSearch, handleReset, handleCreate, handleEdit, handleDelete, handleSave,
  fetchData
} = useEntity<User>({
  api: userApi,
  searchFields: ['keyword'],
  formFields: userFields,
  successMessages: { create: '创建成功', update: '更新成功', delete: '删除成功' }
})

const roleDialogVisible = ref(false)
const currentUser = ref<User | null>(null)
const selectedRoles = ref<string[]>([])

const roleLabel = (role: string): string => {
  const map: Record<string, string> = {
    system_admin: '系统管理员',
    system_publisher: '系统发布者',
    system_moderator: '系统审核员',
    system_operator: '系统运营',
  }
  return map[role] || role
}

const handleManageRoles = (row: User) => {
  currentUser.value = row
  selectedRoles.value = row.roles ? [...row.roles] : []
  roleDialogVisible.value = true
}

const handleSaveRoles = async () => {
  if (!currentUser.value) return
  submitting.value = true
  try {
    await assignRoles(currentUser.value.id, selectedRoles.value)
    ElMessage.success('角色更新成功')
    roleDialogVisible.value = false
    fetchData()
  } catch (error) {
    console.error('[UserManagement] save roles failed:', error)
  } finally {
    submitting.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.user-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-info {
  display: flex;
  flex-direction: column;
}

.username {
  font-weight: 600;
  color: var(--color-text-primary);
}

.nickname {
  font-size: 12px;
  color: var(--color-text-muted);
}

.role-tag {
  margin-right: 4px;
}

.role-dialog-hint {
  margin-bottom: var(--space-4);
  color: var(--color-text-secondary);
  font-weight: 500;
}
</style>
```

- [ ] **步骤 3：删除旧文件**

```bash
cd qim-admin
git rm src/views/Users.vue
```

- [ ] **步骤 4：更新路由**

修改 `src/router/index.ts`，将 Users.vue 路径改为 UserManagement/index.vue：

```typescript
// 修改前
{ path: 'users', component: () => import('@/views/Users.vue') }
// 修改后
{ path: 'users', component: () => import('@/views/UserManagement/index.vue') }
```

- [ ] **步骤 5：验证构建**

```bash
cd qim-admin
npm run build
```

- [ ] **步骤 6：Commit**

```bash
cd qim-admin
git add src/views/UserManagement/ src/router/index.ts
git rm src/views/Users.vue
git commit -m "refactor(view): 重构用户管理页面"
```

---

### 任务 15：重构 GroupManagement

**文件：**
- 创建：`qim-admin/src/views/GroupManagement/index.vue`
- 删除：`qim-admin/src/views/Groups.vue`

- [ ] **步骤 1：创建 GroupManagement/index.vue**

```vue
<!-- src/views/GroupManagement/index.vue -->
<template>
  <DataTable :data="list" :loading="loading" :pagination="pagination"
    @search="handleSearch" @page-change="fetchData" @refresh="fetchData">
    <template #search>
      <SearchForm @search="handleSearch" @reset="handleReset">
        <SearchField v-model="searchForm.keyword" label="群组名称" placeholder="请输入群组名称" />
      </SearchForm>
    </template>

    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column label="群组名称" min-width="180">
      <template #default="{ row }">
        <div class="group-cell">
          <el-avatar :size="32" :src="row.avatar">{{ row.name.charAt(0) }}</el-avatar>
          <span class="group-name">{{ row.name }}</span>
        </div>
      </template>
    </el-table-column>
    <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
    <el-table-column prop="memberCount" label="成员数" width="100" />
    <el-table-column label="状态" width="100">
      <template #default="{ row }">
        <StatusTag :status="row.status" />
      </template>
    </el-table-column>
    <el-table-column prop="createdAt" label="创建时间" width="180" />
    <el-table-column label="操作" width="180" fixed="right">
      <template #default="{ row }">
        <el-button size="small" type="primary" @click="handleViewMembers(row)">查看成员</el-button>
        <el-popconfirm title="确定删除该群组吗？" @confirm="handleDelete(row.id)">
          <template #reference>
            <ActionButton type="danger">删除</ActionButton>
          </template>
        </el-popconfirm>
      </template>
    </el-table-column>
  </DataTable>

  <el-dialog v-model="memberDialogVisible" :title="`群组成员 - ${currentGroup?.name || ''}`" width="600px">
    <el-table :data="members" v-loading="membersLoading" size="small" max-height="400">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column label="成员" min-width="150">
        <template #default="{ row }">
          <div class="member-cell">
            <el-avatar :size="28" :src="row.avatar">{{ (row.nickname || row.username).charAt(0) }}</el-avatar>
            <span>{{ row.nickname || row.username }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="角色" width="100">
        <template #default="{ row }">
          <el-tag size="small" v-if="row.role">{{ roleLabel(row.role) }}</el-tag>
          <span v-else class="text-muted">普通成员</span>
        </template>
      </el-table-column>
      <el-table-column prop="joinedAt" label="加入时间" width="180" />
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-popconfirm title="确定移除该成员吗？" @confirm="handleRemoveMember(row.userId)">
            <template #reference>
              <el-button size="small" type="danger" text>移除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import DataTable from '@/components/data/DataTable.vue'
import SearchForm from '@/components/data/SearchForm.vue'
import SearchField from '@/components/data/SearchField.vue'
import StatusTag from '@/components/data/StatusTag.vue'
import ActionButton from '@/components/common/ActionButton.vue'
import { useEntity } from '@/composables/useEntity'
import { getGroups, getGroupMembers, removeGroupMember, deleteGroup } from '@/api/groups'
import type { Group, ConversationMember } from '@/types'

const groupApi = {
  getList: (params: Record<string, unknown>) => getGroups(params as any),
  create: async () => {},
  update: async () => {},
  delete: (id: number) => deleteGroup(id),
}

const {
  list, loading, pagination, searchForm,
  handleSearch, handleReset, handleDelete, fetchData
} = useEntity<Group>({
  api: groupApi,
  searchFields: ['keyword'],
  formFields: [],
})

const memberDialogVisible = ref(false)
const currentGroup = ref<Group | null>(null)
const members = ref<ConversationMember[]>([])
const membersLoading = ref(false)

const roleLabel = (role: string): string => {
  const map: Record<string, string> = { owner: '群主', admin: '管理员', member: '成员' }
  return map[role] || role
}

const handleViewMembers = async (row: Group) => {
  currentGroup.value = row
  memberDialogVisible.value = true
  await fetchMembers(row.id)
}

const fetchMembers = async (conversationId: number) => {
  membersLoading.value = true
  try {
    const { data } = await getGroupMembers(conversationId, { page: 1, pageSize: 100 })
    members.value = data.data.list
  } catch (error) {
    console.error('[GroupManagement] fetch members failed:', error)
  } finally {
    membersLoading.value = false
  }
}

const handleRemoveMember = async (userId: number) => {
  if (!currentGroup.value) return
  try {
    await removeGroupMember(currentGroup.value.id, userId)
    ElMessage.success('移除成功')
    fetchMembers(currentGroup.value.id)
  } catch (error) {
    console.error('[GroupManagement] remove member failed:', error)
  }
}

onMounted(fetchData)
</script>

<style scoped>
.group-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.group-name {
  font-weight: 600;
  color: var(--color-text-primary);
}

.member-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
```

- [ ] **步骤 2：更新路由**

```typescript
// router/index.ts 修改
{ path: 'groups', component: () => import('@/views/GroupManagement/index.vue') }
```

- [ ] **步骤 3：删除旧文件**

```bash
cd qim-admin
git rm src/views/Groups.vue
```

- [ ] **步骤 4：验证构建并 Commit**

```bash
cd qim-admin
npm run build
git add src/views/GroupManagement/ src/router/index.ts
git commit -m "refactor(view): 重构群组管理页面"
```

---

### 任务 16：重构 RoleManagement

**文件：**
- 创建：`qim-admin/src/views/RoleManagement/index.vue`
- 删除：`qim-admin/src/views/Roles.vue`

- [ ] **步骤 1：读取现有 Roles.vue**

```bash
cd qim-admin
cat src/views/Roles.vue
```

根据现有内容重构，使用 DataTable + useEntity 模式。

- [ ] **步骤 2：创建 RoleManagement/index.vue**

```vue
<!-- src/views/RoleManagement/index.vue -->
<template>
  <DataTable :data="list" :loading="loading" :pagination="pagination"
    @search="handleSearch" @page-change="fetchData" @refresh="fetchData">
    <template #search>
      <SearchForm @search="handleSearch" @reset="handleReset">
        <SearchField v-model="searchForm.keyword" label="角色名称" placeholder="请输入角色名称" />
      </SearchForm>
    </template>
    <template #actions>
      <el-button v-permission="'role:create'" type="success" @click="handleCreate">创建角色</el-button>
    </template>

    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column prop="name" label="角色名称" min-width="150" />
    <el-table-column prop="code" label="角色代码" min-width="150" />
    <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
    <el-table-column label="用户数" width="100">
      <template #default="{ row }">
        {{ row.userCount || 0 }}
      </template>
    </el-table-column>
    <el-table-column prop="createdAt" label="创建时间" width="180" />
    <el-table-column label="操作" width="200" fixed="right">
      <template #default="{ row }">
        <ActionButton v-permission="'role:update'" @click="handleEdit(row)">编辑</ActionButton>
        <el-popconfirm title="确定删除该角色吗？" @confirm="handleDelete(row.id)">
          <template #reference>
            <ActionButton v-permission="'role:delete'" type="danger">删除</ActionButton>
          </template>
        </el-popconfirm>
      </template>
    </el-table-column>
  </DataTable>

  <EntityDialog
    v-model="dialogVisible"
    :mode="dialogMode"
    :fields="roleFields"
    :rules="roleRules"
    :initial-data="formData"
    :create-title="'创建角色'"
    :edit-title="'编辑角色'"
    @save="handleSave"
  />
</template>

<script setup lang="ts">
import DataTable from '@/components/data/DataTable.vue'
import SearchForm from '@/components/data/SearchForm.vue'
import SearchField from '@/components/data/SearchField.vue'
import ActionButton from '@/components/common/ActionButton.vue'
import EntityDialog from '@/components/forms/EntityDialog.vue'
import { useEntity } from '@/composables/useEntity'
import type { FormField } from '@/composables/useEntity'
import { getRoles, createRole, updateRole, deleteRole } from '@/api/roles'
import type { Role } from '@/types'

const roleFields: FormField[] = [
  { name: 'name', label: '角色名称', type: 'input', required: true },
  { name: 'code', label: '角色代码', type: 'input', required: true },
  { name: 'description', label: '描述', type: 'textarea' },
]

const roleRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入角色代码', trigger: 'blur' }],
}

const roleApi = {
  getList: (params: Record<string, unknown>) => getRoles(params as any),
  create: (data: Record<string, unknown>) => createRole(data as any),
  update: (id: number, data: Record<string, unknown>) => updateRole(id, data as any),
  delete: (id: number) => deleteRole(id),
}

const {
  list, loading, pagination, searchForm,
  dialogVisible, dialogMode, formData,
  handleSearch, handleReset, handleCreate, handleEdit, handleDelete, handleSave,
  fetchData
} = useEntity<Role>({
  api: roleApi,
  searchFields: ['keyword'],
  formFields: roleFields,
  successMessages: { create: '创建成功', update: '更新成功', delete: '删除成功' }
})

onMounted(fetchData)
</script>
```

- [ ] **步骤 3：验证并 Commit**

```bash
cd qim-admin
npm run build
git add src/views/RoleManagement/ src/router/index.ts
git rm src/views/Roles.vue
git commit -m "refactor(view): 重构角色管理页面"
```

---

## 阶段 4 - 注册集成与清理

### 任务 17：注册权限指令和路由守卫

**文件：**
- 修改：`qim-admin/src/main.ts`
- 修改：`qim-admin/src/router/index.ts`

- [ ] **步骤 1：在 main.ts 注册权限指令**

```typescript
// src/main.ts 中添加
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'
import { permissionDirective } from './directives/permission'
import { setupPermissionGuard } from './router/guards'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(ElementPlus)

// 注册权限指令
app.directive('permission', permissionDirective)

// 设置路由守卫
router.beforeEach(setupPermissionGuard())

app.mount('#app')
```

- [ ] **步骤 2：更新 router/index.ts**

将现有的 beforeEach 替换为使用 guards.ts 中的守卫：

```typescript
// router/index.ts 中删除原有的 beforeEach
// 改为：
router.beforeEach(() => {}) // 占位，实际守卫在 main.ts 中设置
```

- [ ] **步骤 3：在路由中添加权限元信息**

为每个路由添加 `meta.permission`：

```typescript
{
  path: 'users',
  name: 'Users',
  component: () => import('@/views/UserManagement/index.vue'),
  meta: { title: '用户管理', permission: 'user:read' },
},
```

为所有 19 个路由添加权限。

- [ ] **步骤 4：验证构建**

```bash
cd qim-admin
npm run build
```

- [ ] **步骤 5：Commit**

```bash
cd qim-admin
git add src/main.ts src/router/index.ts
git commit -m "feat(integration): 注册权限指令和路由守卫"
```

---

### 任务 18：运行全部测试并修复

- [ ] **步骤 1：运行全部测试**

```bash
cd qim-admin
npm run test
```

- [ ] **步骤 2：修复任何失败**

根据测试输出修复问题。

- [ ] **步骤 3：运行构建验证**

```bash
cd qim-admin
npm run build
```

- [ ] **步骤 4：Commit**

```bash
cd qim-admin
git add .
git commit -m "fix: 修复测试和构建问题"
```

---

## 自检

### 规格覆盖检查

| 规格需求 | 对应任务 | 状态 |
|----------|----------|------|
| 权限 Store | 任务 2 | ✅ |
| 权限指令 | 任务 3 | ✅ |
| 路由守卫 | 任务 4 | ✅ |
| 403 页面 | 任务 4 | ✅ |
| 请求拦截增强 | 任务 5 | ✅ |
| usePermission | 任务 6 | ✅ |
| DataTable 组件 | 任务 7 | ✅ |
| SearchForm/SearchField | 任务 8 | ✅ |
| StatusTag/ActionButton | 任务 9 | ✅ |
| EntityDialog/FieldRenderer | 任务 10 | ✅ |
| useEntity | 任务 11 | ✅ |
| 布局组件拆分 | 任务 12 | ✅ |
| Dashboard 重构 | 任务 13 | ✅ |
| UserManagement 重构 | 任务 14 | ✅ |
| GroupManagement 重构 | 任务 15 | ✅ |
| RoleManagement 重构 | 任务 16 | ✅ |
| 指令注册 | 任务 17 | ✅ |
| 测试验证 | 任务 18 | ✅ |

### 类型一致性检查

- `FormField` 接口在 useEntity.ts 和 config.ts 中一致
- `EntityAPI<T>` 接口在 useEntity.ts 和页面中一致
- `Permission`、`Role` 类型在 types/index.ts 中统一定义

---

计划已完成并保存到 `docs/superpowers/plans/2026-04-28-admin-redesign.md`。两种执行方式：

**1. 子代理驱动（推荐）** - 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** - 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点

选哪种方式？
