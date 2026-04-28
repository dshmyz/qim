# 移动端响应式增强实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为 QIM Admin 管理后台实现完整的移动端响应式布局，将侧边栏在移动端改为抽屉模式，提升小屏幕设备的使用体验。

**架构：** 在 AdminLayout 中通过 `window.matchMedia` 监听屏幕宽度，移动端（≤768px）时将固定侧边栏切换为抽屉模式，添加遮罩层和汉堡菜单按钮，支持触摸手势。

**技术栈：** Vue 3 Composition API、CSS 响应式设计、Element Plus

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `src/components/layout/Sidebar/index.vue` | 修改 | 增加移动端样式适配 |
| `src/components/layout/Header/index.vue` | 修改 | 添加汉堡菜单按钮 |
| `src/components/layout/SidebarDrawer.vue` | 创建 | 移动端抽屉包装组件 |
| `src/components/layout/MobileOverlay.vue` | 创建 | 遮罩层组件 |
| `src/layouts/AdminLayout.vue` | 修改 | 响应式布局逻辑切换 |
| `src/styles/main.css` | 修改 | 补充全局移动端样式 |

---

### 任务 1：创建 MobileOverlay 遮罩层组件

**文件：**
- 创建：`src/components/layout/MobileOverlay.vue`
- 测试：无需单元测试

- [ ] **步骤 1：创建 MobileOverlay 组件**

```vue
<!-- src/components/layout/MobileOverlay.vue -->
<template>
  <Transition name="fade">
    <div v-if="visible" class="mobile-overlay" @click="$emit('close')" />
  </Transition>
</template>

<script setup lang="ts">
defineProps<{
  visible: boolean
}>()

defineEmits<{
  close: []
}>()
</script>

<style scoped>
.mobile-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(15, 23, 42, 0.5);
  z-index: 999;
  backdrop-filter: blur(2px);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/layout/MobileOverlay.vue
git commit -m "feat(layout): 添加移动端遮罩层组件"
```

---

### 任务 2：修改 Header 添加汉堡菜单按钮

**文件：**
- 修改：`src/components/layout/Header/index.vue`

- [ ] **步骤 1：修改 Header 模板，添加汉堡菜单**

```vue
<!-- src/components/layout/Header/index.vue -->
<template>
  <el-header class="admin-header">
    <div class="header-left">
      <button v-if="showHamburger" class="hamburger-btn" @click="$emit('toggleSidebar')">
        <el-icon :size="20">
          <Fold v-if="!sidebarOpen" />
          <Expand v-else />
        </el-icon>
      </button>
      <slot name="breadcrumb"></slot>
    </div>
    <div class="header-right">
      <ThemeToggle />
      <UserDropdown />
    </div>
  </el-header>
</template>

<script setup lang="ts">
import { Fold, Expand } from '@element-plus/icons-vue'
import ThemeToggle from './ThemeToggle.vue'
import UserDropdown from './UserDropdown.vue'

defineProps<{
  showHamburger?: boolean
  sidebarOpen?: boolean
}>()

defineEmits<{
  toggleSidebar: []
}>()
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
  width: 100%;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.hamburger-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  background: transparent;
  color: var(--color-text-secondary);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

.hamburger-btn:hover {
  background-color: var(--color-primary-lighter);
  color: var(--color-primary);
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/layout/Header/index.vue
git commit -m "feat(layout): Header添加汉堡菜单按钮"
```

---

### 任务 3：创建 SidebarDrawer 抽屉包装组件

**文件：**
- 创建：`src/components/layout/SidebarDrawer.vue`
- 测试：无需单元测试

- [ ] **步骤 1：创建 SidebarDrawer 组件**

```vue
<!-- src/components/layout/SidebarDrawer.vue -->
<template>
  <Transition name="slide">
    <div v-if="visible" class="sidebar-drawer">
      <Sidebar :collapsed="false" @toggle="$emit('close')" />
    </div>
  </Transition>
</template>

<script setup lang="ts">
import Sidebar from './Sidebar/index.vue'

defineProps<{
  visible: boolean
}>()

defineEmits<{
  close: []
}>()
</script>

<style scoped>
.sidebar-drawer {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 260px;
  z-index: 1000;
  box-shadow: 4px 0 24px rgba(0, 0, 0, 0.15);
}

.slide-enter-active,
.slide-leave-active {
  transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.slide-enter-from {
  transform: translateX(-100%);
}

.slide-leave-to {
  transform: translateX(-100%);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/layout/SidebarDrawer.vue
git commit -m "feat(layout): 创建移动端侧边栏抽屉组件"
```

---

### 任务 4：重构 AdminLayout 增加响应式逻辑

**文件：**
- 修改：`src/layouts/AdminLayout.vue`

- [ ] **步骤 1：修改 AdminLayout 模板**

```vue
<!-- src/layouts/AdminLayout.vue -->
<template>
  <el-container class="admin-layout">
    <Sidebar v-if="!isMobile" :collapsed="isCollapsed" @toggle="isCollapsed = !isCollapsed" />
    <SidebarDrawer :visible="isDrawerOpen" @close="isDrawerOpen = false" />
    <MobileOverlay :visible="isDrawerOpen" @close="isDrawerOpen = false" />
    <el-container class="main-container">
      <Header 
        :show-hamburger="isMobile" 
        :sidebar-open="isDrawerOpen" 
        @toggle-sidebar="isDrawerOpen = !isDrawerOpen"
      >
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import Sidebar from '@/components/layout/Sidebar/index.vue'
import Header from '@/components/layout/Header/index.vue'
import Breadcrumb from '@/components/layout/Breadcrumb/index.vue'
import SidebarDrawer from '@/components/layout/SidebarDrawer.vue'
import MobileOverlay from '@/components/layout/MobileOverlay.vue'

const route = useRoute()
const isCollapsed = ref(false)
const isMobile = ref(false)
const isDrawerOpen = ref(false)

const MOBILE_BREAKPOINT = 768

const checkMobile = () => {
  isMobile.value = window.innerWidth <= MOBILE_BREAKPOINT
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})

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
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: var(--color-bg-page);
}

.admin-main {
  background-color: var(--color-bg-page);
  padding: var(--space-6);
  overflow-y: auto;
  flex: 1;
}

:deep(.el-main) {
  padding: 0;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/layouts/AdminLayout.vue
git commit -m "feat(layout): AdminLayout增加响应式布局切换逻辑"
```

---

### 任务 5：优化全局移动端样式

**文件：**
- 修改：`src/styles/main.css`

- [ ] **步骤 1：在 main.css 末尾添加移动端优化样式**

在现有 `@media (max-width: 768px)` 之后添加：

```css
/* ==========================================
   8. 移动端增强
   ========================================== */
@media (max-width: 768px) {
  html {
    font-size: 14px;
  }

  :root {
    --sidebar-width: 200px;
  }

  /* 主内容区减少 padding */
  .admin-main {
    padding: var(--space-4) !important;
  }

  /* 页面标题区适配 */
  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-3);
  }

  /* 对话框适配 */
  .el-dialog {
    width: 95vw !important;
    margin: var(--space-4) auto !important;
  }

  /* 表格操作列按钮紧凑化 */
  .el-table .el-button--small {
    padding: 0 var(--space-2) !important;
  }

  /* 数据表格头部适配 */
  .table-header {
    flex-direction: column;
    align-items: stretch;
    gap: var(--space-3);
  }

  .table-header .left-actions {
    justify-content: space-between;
  }
}

@media (max-width: 640px) {
  .hide-mobile {
    display: none !important;
  }

  /* Header 紧凑化 */
  .admin-header {
    padding: 0 var(--space-4) !important;
  }

  /* 统计卡片单列 */
  .stat-cards .el-col {
    max-width: 100% !important;
    flex: 0 0 100% !important;
  }
}

/* 移动端锁定 body 滚动 */
body.mobile-drawer-open {
  overflow: hidden;
}
```

- [ ] **步骤 2：在 AdminLayout 中添加 body 滚动锁定逻辑**

在 `src/layouts/AdminLayout.vue` 的 script 中添加 watch：

```typescript
import { watch } from 'vue'

watch(isDrawerOpen, (open) => {
  if (open) {
    document.body.classList.add('mobile-drawer-open')
  } else {
    document.body.classList.remove('mobile-drawer-open')
  }
})
```

- [ ] **步骤 3：Commit**

```bash
git add src/styles/main.css src/layouts/AdminLayout.vue
git commit -m "feat(styles): 优化全局移动端响应式样式"
```

---

### 任务 6：添加触摸手势支持

**文件：**
- 修改：`src/layouts/AdminLayout.vue`

- [ ] **步骤 1：添加触摸手势检测**

在 `src/layouts/AdminLayout.vue` 的 script 中添加：

```typescript
const touchStartX = ref(0)
const touchStartY = ref(0)

const onTouchStart = (e: TouchEvent) => {
  touchStartX.value = e.touches[0].clientX
  touchStartY.value = e.touches[0].clientY
}

const onTouchEnd = (e: TouchEvent) => {
  const deltaX = e.changedTouches[0].clientX - touchStartX.value
  const deltaY = Math.abs(e.changedTouches[0].clientY - touchStartY.value)
  
  // 仅当水平滑动大于垂直滑动，且从左边缘开始
  if (deltaX > 80 && deltaY < 100 && touchStartX.value < 40) {
    isDrawerOpen.value = true
  }
  
  // 从右向左滑动关闭抽屉
  if (deltaX < -80 && deltaY < 100 && isDrawerOpen.value) {
    isDrawerOpen.value = false
  }
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
  window.addEventListener('touchstart', onTouchStart, { passive: true })
  window.addEventListener('touchend', onTouchEnd, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
  window.removeEventListener('touchstart', onTouchStart)
  window.removeEventListener('touchend', onTouchEnd)
})
```

- [ ] **步骤 2：Commit**

```bash
git add src/layouts/AdminLayout.vue
git commit -m "feat(layout): 添加移动端触摸手势支持"
```

---

### 任务 7：运行测试并验证

- [ ] **步骤 1：运行类型检查**

```bash
cd /Users/gracegaoya/work/project/qim/qim-admin
npx vue-tsc --noEmit
```

预期：无错误输出

- [ ] **步骤 2：运行单元测试**

```bash
npm run test
```

预期：所有测试通过

- [ ] **步骤 3：启动开发服务器验证视觉效果**

```bash
npm run dev
```

打开浏览器，访问 `http://localhost:3008`（或实际端口），调整窗口宽度测试：
- 桌面端（>768px）：侧边栏固定显示
- 移动端（≤768px）：点击汉堡菜单打开抽屉，点击遮罩关闭
- 触摸滑动：从左边缘向右滑动打开抽屉

- [ ] **步骤 4：Commit（如有修复）**

---

## 规格自检

### 1. 规格覆盖度
- ✅ 移动端侧边栏抽屉模式 → 任务 3、4
- ✅ 汉堡菜单按钮 → 任务 2
- ✅ 遮罩层 → 任务 1
- ✅ 响应式断点 → 任务 5
- ✅ 触摸手势 → 任务 6
- ✅ 滚动锁定 → 任务 5

### 2. 占位符扫描
- ✅ 无 "TODO" 或 "待定"
- ✅ 所有步骤都有完整代码
- ✅ 无模糊需求

### 3. 类型一致性
- ✅ Props/Emits 定义一致
- ✅ 组件导入路径正确
- ✅ CSS 变量使用统一

---

计划已完成并保存到 `docs/superpowers/plans/2026-04-28-mobile-responsive-enhancement.md`。两种执行方式：

**1. 子代理驱动（推荐）** - 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** - 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点

选哪种方式？
