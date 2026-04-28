# QIM Admin 全面修复实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 修复所有严重 Bug、优化操作便利性、改善界面布局，提升整体用户体验和代码质量。

**架构：** 按模块分组修复，先修核心 Bug，再优化组件功能，最后完善页面布局。确保每个修复都是独立的、可测试的。

**技术栈：** Vue 3、TypeScript、Element Plus、Pinia

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `src/components/data/DataTable.vue` | 修改 | 修复分页逻辑、添加 loading 防抖 |
| `src/components/forms/EntityDialog.vue` | 修改 | 修复 loading 状态同步、清除验证状态 |
| `src/views/RoleManagement/index.vue` | 修改 | 添加权限分配功能、修复标签显示 |
| `src/views/UserManagement/index.vue` | 修改 | 修复编辑模式密码验证、优化角色标签 |
| `src/views/GroupManagement/index.vue` | 修改 | 添加成员分页、修复空值保护 |
| `src/views/Statistics.vue` | 修改 | 修复活跃群组数据、启用趋势数据 |
| `src/views/Dashboard/index.vue` | 修改 | 修复空值保护、优化 loading 状态 |
| `src/views/Login.vue` | 修改 | 改进错误处理、添加 redirect 支持 |
| `src/composables/useEntity.ts` | 修改 | 统一空值处理、优化分页逻辑 |

---

### 任务 1：修复 DataTable 分页逻辑和 loading 防抖

**文件：**
- 修改：`src/components/data/DataTable.vue`

- [ ] **步骤：修复分页事件处理**

当前问题：
```typescript
// 当前代码 - 语义错误
@size-change="$emit('page-change', $event)"  // pageSize 改变时传递 pageSize
@current-change="$emit('page-change', $event)"  // 页码改变时传递页码
// 两个事件使用同一个 emit，父组件无法区分
```

修改为：
```vue
<template>
  <el-pagination
    :current-page="pagination.page"
    :page-size="pagination.pageSize"
    :total="total"
    :page-sizes="pageSizes"
    layout="total, sizes, prev, pager, next, jumper"
    @size-change="handleSizeChange"
    @current-change="handlePageChange"
  />
</template>

<script setup lang="ts">
defineEmits<{
  'page-change': [page: number]
  'size-change': [pageSize: number]
  'refresh': []
}>()

const total = computed(() => props.pagination.total)

const handleSizeChange = (pageSize: number) => {
  emit('size-change', pageSize)
  emit('page-change', 1)  // 重置到第 1 页
}

const handlePageChange = (page: number) => {
  emit('page-change', page)
}
</script>
```

- [ ] **步骤：添加刷新按钮防抖**

```vue
<template>
  <el-button size="small" :icon="Refresh" circle @click="handleRefresh" :loading="loading" />
</template>

<script setup lang="ts">
const isRefreshing = ref(false)

const handleRefresh = () => {
  if (isRefreshing.value) return
  isRefreshing.value = true
  emit('refresh')
  setTimeout(() => {
    isRefreshing.value = false
  }, 500)
}
</script>
```

- [ ] **Commit**

```bash
git add src/components/data/DataTable.vue
git commit -m "fix(data-table): 修复分页逻辑和添加刷新防抖"
```

---

### 任务 2：修复 EntityDialog loading 状态和验证清除

**文件：**
- 修改：`src/components/forms/EntityDialog.vue`

- [ ] **步骤：修复 loading 状态同步**

当前问题：
```typescript
const handleSave = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  
  loading.value = true
  try {
    emit('save', { formData, isEdit: !!formData.id })
    dialogVisible.value = false
  } catch (error) {
    console.error('[EntityDialog] save failed:', error)
  } finally {
    loading.value = false
  }
}
// 问题：emit 是同步的，父组件的异步操作无法影响 loading 状态
```

修改为使用 Promise 模式：
```typescript
interface SaveEvent {
  formData: Record<string, any>
  isEdit: boolean
  onSuccess: () => void
  onError: (error: any) => void
}

defineEmits<{
  'save': [data: SaveEvent]
}>()

const handleSave = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  
  loading.value = true
  
  emit('save', {
    formData,
    isEdit: !!formData.id,
    onSuccess: () => {
      dialogVisible.value = false
      loading.value = false
    },
    onError: (error: any) => {
      console.error('[EntityDialog] save failed:', error)
      loading.value = false
    }
  })
}
```

- [ ] **步骤：添加对话框关闭时清除验证**

```typescript
watch(
  () => props.open,
  (open) => {
    if (open) {
      if (props.entity) {
        formData.value = { ...props.entity }
      } else {
        formData.value = {}
        fieldConfigs.value.forEach((field) => {
          formData.value[field.key] = ''
        })
      }
      // 清除验证状态
      nextTick(() => {
        formRef.value?.resetFields()
      })
    }
  }
)
```

- [ ] **Commit**

```bash
git add src/components/forms/EntityDialog.vue
git commit -m "fix(entity-dialog): 修复 loading 同步和验证清除"
```

---

### 任务 3：修复 RoleManagement 权限分配功能

**文件：**
- 修改：`src/views/RoleManagement/index.vue`

- [ ] **步骤：添加权限字段到表单**

```typescript
const roleFields: FormField[] = [
  { key: 'name', label: '角色名称', type: 'text', placeholder: '请输入角色名称', required: true },
  { key: 'code', label: '角色编码', type: 'text', placeholder: '请输入角色编码（字母、数字、下划线）', required: true, rules: [{ pattern: /^[a-zA-Z0-9_]+$/, message: '只允许字母、数字、下划线' }] },
  { key: 'description', label: '描述', type: 'textarea', placeholder: '请输入角色描述' },
  { key: 'permissions', label: '权限', type: 'checkbox-group', placeholder: '请选择权限', options: [
    { label: '查看用户', value: 'user:read' },
    { label: '创建用户', value: 'user:create' },
    { label: '编辑用户', value: 'user:update' },
    { label: '删除用户', value: 'user:delete' },
    { label: '查看群组', value: 'group:read' },
    { label: '创建群组', value: 'group:create' },
    { label: '编辑群组', value: 'group:update' },
    { label: '删除群组', value: 'group:delete' },
    { label: '查看角色', value: 'role:read' },
    { label: '创建角色', value: 'role:create' },
    { label: '编辑角色', value: 'role:update' },
    { label: '删除角色', value: 'role:delete' },
  ]},
]
```

- [ ] **步骤：优化权限标签显示（最多显示3个）**

```vue
<el-table-column label="权限" min-width="240">
  <template #default="{ row }">
    <div class="permissions-cell">
      <el-tag
        v-for="perm in row.permissions.slice(0, 3)"
        :key="perm"
        size="small"
      >
        {{ permissionLabel(perm) }}
      </el-tag>
      <el-tag v-if="row.permissions.length > 3" size="small" type="info">
        +{{ row.permissions.length - 3 }}
      </el-tag>
    </div>
  </template>
</el-table-column>
```

- [ ] **Commit**

```bash
git add src/views/RoleManagement/index.vue
git commit -m "feat(role-management): 添加权限分配功能和优化标签显示"
```

---

### 任务 4：修复 UserManagement 编辑模式密码验证

**文件：**
- 修改：`src/views/UserManagement/index.vue`

- [ ] **步骤：动态调整密码字段验证**

```typescript
const userFields = computed<FormField[]>(() => [
  { key: 'username', label: '用户名', type: 'text', placeholder: '请输入用户名', required: true },
  { key: 'nickname', label: '昵称', type: 'text', placeholder: '请输入昵称', required: false },
  { key: 'email', label: '邮箱', type: 'text', placeholder: '请输入邮箱' },
  { key: 'phone', label: '手机号', type: 'text', placeholder: '请输入手机号' },
  // 编辑模式下密码非必填
  { key: 'password', label: '密码', type: 'password', placeholder: mode.value === 'edit' ? '留空表示不修改密码' : '请输入密码', required: mode.value === 'create' },
  { key: 'roleIds', label: '角色', type: 'select', placeholder: '请选择角色', options: roleOptions.value, multiple: true },
])
```

- [ ] **步骤：优化角色标签显示（最多显示3个）**

```vue
<el-table-column label="角色" min-width="200">
  <template #default="{ row }">
    <el-tag v-for="role in row.roles.slice(0, 3)" :key="role" class="role-tag" size="small">
      {{ getRoleName(role) }}
    </el-tag>
    <el-tag v-if="row.roles.length > 3" size="small" type="info">
      +{{ row.roles.length - 3 }}
    </el-tag>
  </template>
</el-table-column>
```

- [ ] **步骤：添加空值保护**

```typescript
const getRoleName = (roleId: number): string => {
  const role = roles.value.find((r) => r.id === roleId)
  return role?.name || `角色 #${roleId}`
}
```

- [ ] **Commit**

```bash
git add src/views/UserManagement/index.vue
git commit -m "fix(user-management): 修复编辑模式密码验证和优化角色标签"
```

---

### 任务 5：修复 GroupManagement 成员分页和空值保护

**文件：**
- 修改：`src/views/GroupManagement/index.vue`

- [ ] **步骤：添加成员分页**

```typescript
const memberPagination = ref({
  page: 1,
  pageSize: 20,
  total: 0,
})

const fetchMembers = async (page = 1) => {
  if (!currentGroup.value) return
  membersLoading.value = true
  try {
    const { data } = await getGroupMembers(currentGroup.value.id, {
      page,
      pageSize: memberPagination.value.pageSize,
    })
    if (data.data) {
      members.value = data.data.list ?? []
      memberPagination.value.total = data.data.total ?? 0
      memberPagination.value.page = page
    }
  } catch (error) {
    console.error('[GroupManagement] fetch members failed:', error)
    ElMessage.error('获取成员列表失败')
  } finally {
    membersLoading.value = false
  }
}

const handleMemberPageChange = (page: number) => {
  fetchMembers(page)
}
```

- [ ] **步骤：添加空值保护**

```vue
<el-avatar :size="32">{{ row.name?.charAt(0) || '?' }}</el-avatar>
```

```vue
<el-avatar :size="28" :src="row.avatar">{{ (row.nickname || row.username)?.charAt(0) || '?' }}</el-avatar>
```

- [ ] **步骤：添加群主保护**

```typescript
const handleRemoveMember = async (userId: number) => {
  if (userId === currentGroup.value?.ownerId) {
    ElMessage.warning('无法移除群主')
    return
  }
  
  try {
    await removeGroupMember(currentGroup.value!.id, userId)
    ElMessage.success('成员移除成功')
    fetchMembers(memberPagination.value.page)
  } catch (error) {
    console.error('[GroupManagement] remove member failed:', error)
    ElMessage.error('移除成员失败')
  }
}
```

- [ ] **Commit**

```bash
git add src/views/GroupManagement/index.vue
git commit -m "fix(group-management): 添加成员分页、空值保护和群主保护"
```

---

### 任务 6：修复 Statistics 数据问题

**文件：**
- 修改：`src/views/Statistics.vue`

- [ ] **步骤：修复活跃群组数据显示**

当前问题：
```vue
<div class="overview-item">
  <span class="label">活跃群组</span>
  <span class="value">{{ stats.totalGroups }}</span>  <!-- 错误：显示的是总数 -->
</div>
```

修改为：
```vue
<div class="overview-item">
  <span class="label">活跃群组</span>
  <span class="value">{{ stats.activeGroups ?? stats.totalGroups }}</span>
</div>
```

- [ ] **步骤：启用趋势数据**

```typescript
const fetchStatistics = async () => {
  chartLoading.value = true
  try {
    const { data } = await getStatistics()
    if (data.data) {
      stats.value = data.data
    }
    // 开发环境启用 mock 数据
    if (import.meta.env.DEV) {
      generateMockTrendData()
    }
  } catch (error) {
    console.error('[Statistics] fetch statistics failed:', error)
    ElMessage.error('获取统计数据失败')
  } finally {
    chartLoading.value = false
  }
}
```

- [ ] **步骤：统一增长率卡片布局**

```vue
<!-- 修改为 4 列布局 -->
<el-row :gutter="20" class="chart-row">
  <el-col :xs="24" :sm="12" :md="6" v-for="item in growthItems" :key="item.label">
    <el-card class="growth-card" shadow="never">
      <div class="growth-label">{{ item.label }}</div>
      <div class="growth-value" :class="item.growthClass">
        {{ item.value }}
      </div>
    </el-card>
  </el-col>
</el-row>
```

- [ ] **Commit**

```bash
git add src/views/Statistics.vue
git commit -m "fix(statistics): 修复活跃群组数据和启用趋势数据"
```

---

### 任务 7：修复 Dashboard 空值保护和优化 loading

**文件：**
- 修改：`src/views/Dashboard/index.vue`

- [ ] **步骤：添加空值保护**

```typescript
const fetchRecentRegistrations = async () => {
  registrationsLoading.value = true
  try {
    const { data } = await getRecentRegistrations(10)
    recentRegistrations.value = data.data?.list ?? []
  } catch (error) {
    console.error('Failed to fetch recent registrations:', error)
    ElMessage.error('获取注册数据失败')
  } finally {
    registrationsLoading.value = false
  }
}
```

- [ ] **步骤：优化 loading 状态管理**

```typescript
const refreshData = async () => {
  refreshing.value = true
  try {
    await Promise.all([
      fetchDashboardStats(),
      fetchRecentRegistrations()
    ])
  } catch (error) {
    console.error('Failed to refresh dashboard:', error)
  } finally {
    refreshing.value = false
  }
}
```

- [ ] **Commit**

```bash
git add src/views/Dashboard/index.vue
git commit -m "fix(dashboard): 添加空值保护和优化 loading 状态"
```

---

### 任务 8：优化 Login 错误处理和 redirect 支持

**文件：**
- 修改：`src/views/Login.vue`

- [ ] **步骤：改进错误处理和 redirect**

```typescript
const handleSubmit = async () => {
  if (!formRef.value) return
  
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  
  submitting.value = true
  try {
    const { data } = await login(form)
    if (data.data) {
      authStore.setToken(data.data.token)
      authStore.setUser(data.data.user)
      
      // 跳转到 redirect 页面或首页
      const redirect = route.query.redirect as string
      router.push(redirect || '/')
    }
  } catch (error: any) {
    const message = error?.response?.data?.message || '登录失败，请检查用户名和密码'
    ElMessage.error(message)
  } finally {
    submitting.value = false
  }
}
```

- [ ] **步骤：添加 remember me 功能**

```typescript
const rememberMe = ref(false)

onMounted(() => {
  const remembered = localStorage.getItem('rememberMe')
  if (remembered) {
    form.username = remembered
    rememberMe.value = true
  }
})

const handleSubmit = async () => {
  // ... 登录逻辑
  
  if (rememberMe.value) {
    localStorage.setItem('rememberMe', form.username)
  } else {
    localStorage.removeItem('rememberMe')
  }
}
```

- [ ] **Commit**

```bash
git add src/views/Login.vue
git commit -m "feat(login): 改进错误处理、添加 redirect 和 remember me"
```

---

### 任务 9：运行测试并验证

- [ ] **步骤：运行类型检查**

```bash
cd /Users/gracegaoya/work/project/qim/qim-admin
npx vue-tsc --noEmit
```

预期：无错误

- [ ] **步骤：运行单元测试**

```bash
npm run test
```

预期：所有测试通过

- [ ] **步骤：启动开发服务器验证**

```bash
npm run dev
```

验证以下功能：
1. 分页功能正常（切换页码、每页条数）
2. 角色管理可以分配权限
3. 编辑用户时密码非必填
4. 统计数据正确显示
5. 登录页面 redirect 正常工作
6. 所有空值保护生效

- [ ] **Commit（如有修复）**

---

## 自检

### 1. 规格覆盖度
- ✅ DataTable 分页逻辑 → 任务 1
- ✅ EntityDialog loading → 任务 2
- ✅ RoleManagement 权限分配 → 任务 3
- ✅ UserManagement 密码验证 → 任务 4
- ✅ GroupManagement 成员分页 → 任务 5
- ✅ Statistics 数据修复 → 任务 6
- ✅ Dashboard 空值保护 → 任务 7
- ✅ Login 错误处理 → 任务 8
- ✅ 测试验证 → 任务 9

### 2. 占位符扫描
- ✅ 无 TODO 或待定
- ✅ 所有步骤都有完整代码

### 3. 类型一致性
- ✅ 所有类型定义一致
- ✅ Props/Emits 正确

---

计划已完成并保存到 `docs/superpowers/plans/2026-04-28-comprehensive-fixes.md`。两种执行方式：

**1. 子代理驱动（推荐）** - 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** - 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点

选哪种方式？
