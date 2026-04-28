# 极简白科技风设计实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 QIM Admin 管理后台从扁平饱满风格重构为极简白科技风，保持功能完整的同时提升视觉品质。

**架构：** 通过修改 CSS 变量和全局样式实现整体风格切换，保留暗色主题选项，逐步优化各组件的视觉细节。

**技术栈：** CSS 变量、Vue 3、Element Plus

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `src/styles/main.css` | 修改 | 设计令牌更新、全局样式重构 |
| `src/components/layout/Sidebar/index.vue` | 修改 | 侧边栏极简白风格 |
| `src/components/layout/Header/index.vue` | 修改 | Header 极简风格 |
| `src/components/layout/AdminLayout.vue` | 修改 | 布局样式调整 |
| `src/components/dashboard/StatCards.vue` | 修改 | 统计卡片极简风格 |
| `src/components/data/DataTable.vue` | 修改 | 数据表格极简风格 |
| `src/views/UserManagement/index.vue` | 修改 | 用户管理页面优化 |
| `src/views/GroupManagement/index.vue` | 修改 | 群组管理页面优化 |
| `src/views/RoleManagement/index.vue` | 修改 | 角色管理页面优化 |
| `src/views/Statistics.vue` | 修改 | 数据统计页面优化 |
| `src/views/Login.vue` | 修改 | 登录页极简风格 |

---

### 任务 1：更新 CSS 变量 - 极简白设计令牌

**文件：**
- 修改：`src/styles/main.css`

- [ ] **步骤：更新根 CSS 变量**

```css
/* 修改前 */
--sidebar-bg: var(--gradient-sidebar);
--sidebar-text: rgba(255, 255, 255, 0.7);
--sidebar-bg-active: rgba(255, 255, 255, 0.1);

/* 修改后 - 极简白侧边栏 */
--sidebar-bg: #ffffff;
--sidebar-text: #6b7280;
--sidebar-text-active: #0ea5e9;
--sidebar-bg-active: #f0f9ff;
--sidebar-border: #f0f0f0;
```

- [ ] **步骤：更新渐变和阴影**

```css
/* 修改前 */
--gradient-primary: linear-gradient(135deg, #0ea5e9 0%, #6366f1 100%);
--gradient-sidebar: linear-gradient(180deg, #0f172a 0%, #1e293b 100%);

/* 修改后 - 极简风格减少渐变 */
--gradient-primary: #0ea5e9;
--gradient-sidebar: none;

/* 修改前 - 阴影较重 */
--shadow-card: 0 2px 8px 0 rgb(0 0 0 / 0.04);
--shadow-card-hover: 0 8px 24px -4px rgb(0 0 0 / 0.08);

/* 修改后 - 极简轻阴影 */
--shadow-card: 0 1px 4px 0 rgb(0 0 0 / 0.02);
--shadow-card-hover: 0 4px 16px -2px rgb(0 0 0 / 0.04);
--shadow-button: 0 1px 3px -1px rgb(0 0 0 / 0.04);
--shadow-button-hover: 0 4px 12px -2px rgb(0 0 0 / 0.06);
```

- [ ] **步骤：更新圆角（更简洁）**

```css
/* 修改前 - 较大圆角 */
--radius-xs: 6px;
--radius-sm: 8px;
--radius-md: 10px;
--radius-lg: 12px;
--radius-xl: 16px;
--radius-2xl: 20px;

/* 修改后 - 较小圆角更极简 */
--radius-xs: 4px;
--radius-sm: 6px;
--radius-md: 8px;
--radius-lg: 10px;
--radius-xl: 12px;
--radius-2xl: 16px;
```

- [ ] **Commit**

```bash
git add src/styles/main.css
git commit -m "style(design-tokens): 更新极简白科技风设计令牌"
```

---

### 任务 2：重构侧边栏极简白风格

**文件：**
- 修改：`src/components/layout/Sidebar/index.vue`

- [ ] **步骤：修改侧边栏样式**

```css
.sidebar {
  background: var(--sidebar-bg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  transition: width var(--duration-normal) var(--ease-out);
  box-shadow: 0 2px 8px rgb(0 0 0 / 0.03);
  z-index: 10;
  height: 100vh;
  border-right: 1px solid var(--sidebar-border);
}

/* Logo 区域 */
.logo {
  display: flex;
  align-items: center;
  padding: 0 var(--space-5);
  height: 64px;
  gap: var(--space-3);
  font-weight: 700;
  font-size: 18px;
  color: var(--color-text-primary);
  border-bottom: 1px solid var(--sidebar-border);
  flex-shrink: 0;
}

/* 菜单项 */
.menu-item {
  display: flex;
  align-items: center;
  padding: var(--space-3) var(--space-5);
  margin: 0 var(--space-2);
  border-radius: var(--radius-sm);
  color: var(--sidebar-text);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
  gap: var(--space-3);
  font-size: 14px;
  font-weight: 500;
  position: relative;
}

.menu-item:hover {
  color: var(--color-text-primary);
  background: var(--color-surface-hover);
}

.menu-item.active {
  color: var(--sidebar-text-active);
  background: var(--sidebar-bg-active);
}

.menu-item.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 20px;
  background: var(--sidebar-text-active);
  border-radius: 0 2px 2px 0;
}

/* 折叠按钮 */
.collapse-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  position: absolute;
  bottom: 16px;
  left: 50%;
  transform: translateX(-50%);
  border-radius: var(--radius-sm);
  color: var(--sidebar-text);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
  background: transparent;
  border: 1px solid var(--sidebar-border);
}

.collapse-btn:hover {
  color: var(--color-text-primary);
  background: var(--color-surface-hover);
}
```

- [ ] **Commit**

```bash
git add src/components/layout/Sidebar/index.vue
git commit -m "style(layout): 侧边栏重构为极简白风格"
```

---

### 任务 3：优化卡片组件极简风格

**文件：**
- 修改：`src/components/dashboard/StatCards.vue`
- 修改：`src/components/dashboard/ChartPlaceholders.vue`

- [ ] **步骤：更新 StatCards 样式**

```css
.stat-card {
  background: var(--color-surface);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  border: 1px solid var(--color-border-light);
  transition: all var(--duration-normal) var(--ease-out);
  margin-bottom: var(--space-4);
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-card-hover);
  border-color: transparent;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
  letter-spacing: -0.01em;
}

.stat-label {
  font-size: 13px;
  color: var(--color-text-muted);
  margin-top: var(--space-1);
  font-weight: 500;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: #fff;
}

/* 图标使用纯色而非渐变 */
.stat-icon.blue { background: var(--color-primary); }
.stat-icon.purple { background: var(--color-accent); }
.stat-icon.green { background: var(--color-success); }
.stat-icon.orange { background: var(--color-warning); }
```

- [ ] **步骤：更新 ChartPlaceholders 样式**

```css
.chart-container {
  height: 280px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-surface-hover);
  border-radius: var(--radius-md);
}

.chart-card {
  margin-bottom: var(--space-4);
  border: 1px solid var(--color-border-light);
}
```

- [ ] **Commit**

```bash
git add src/components/dashboard/StatCards.vue src/components/dashboard/ChartPlaceholders.vue
git commit -m "style(dashboard): 卡片组件极简风格优化"
```

---

### 任务 4：优化数据表格极简风格

**文件：**
- 修改：`src/components/data/DataTable.vue`

- [ ] **步骤：更新 DataTable 样式**

```css
.data-table {
  background: var(--color-surface);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  border: 1px solid var(--color-border-light);
  box-shadow: none;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--color-border-light);
}

.table-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
}

/* 表格行去除斑马纹 */
:deep(.el-table__row) {
  border-bottom: 1px solid var(--color-border-light);
}

:deep(.el-table__row:hover) {
  background: var(--color-surface-hover) !important;
}

:deep(.el-table--striped .el-table__body tr.el-table__row--striped) {
  background: transparent;
}

/* 分页区域 */
.pagination-area {
  display: flex;
  justify-content: center;
  padding-top: var(--space-4);
  border-top: 1px solid var(--color-border-light);
  margin-top: var(--space-4);
}
```

- [ ] **Commit**

```bash
git add src/components/data/DataTable.vue
git commit -m "style(data): 数据表格极简风格优化"
```

---

### 任务 5：优化页面组件

**文件：**
- 修改：`src/views/UserManagement/index.vue`
- 修改：`src/views/GroupManagement/index.vue`
- 修改：`src/views/RoleManagement/index.vue`
- 修改：`src/views/Statistics.vue`

- [ ] **步骤：优化 UserManagement 样式**

```css
.user-cell {
  display: flex;
  align-items: center;
  gap: var(--space-3);
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
  margin-right: var(--space-1);
}

.role-dialog-hint {
  margin-bottom: var(--space-4);
  color: var(--color-text-secondary);
  font-weight: 500;
}
```

- [ ] **步骤：优化 GroupManagement 样式**

```css
.group-cell {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.group-name {
  font-weight: 600;
  color: var(--color-text-primary);
}

.member-cell {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
```

- [ ] **步骤：优化 RoleManagement 样式**

```css
.permissions-cell {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}
```

- [ ] **步骤：优化 Statistics 样式**

```css
.statistics-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.stat-card {
  margin-bottom: var(--space-4);
  transition: all var(--duration-normal) var(--ease-out);
  border: 1px solid var(--color-border-light);
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-card-hover);
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
  letter-spacing: -0.01em;
}

.stat-label {
  font-size: 14px;
  color: var(--color-text-muted);
  margin-top: var(--space-1);
  font-weight: 500;
}
```

- [ ] **Commit**

```bash
git add src/views/UserManagement/index.vue src/views/GroupManagement/index.vue src/views/RoleManagement/index.vue src/views/Statistics.vue
git commit -m "style(views): 页面组件极简风格优化"
```

---

### 任务 6：优化登录页极简风格

**文件：**
- 修改：`src/views/Login.vue`

- [ ] **步骤：简化登录页样式**

```css
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--color-bg-base);
  padding: var(--space-8);
}

.login-card {
  background: var(--color-surface);
  padding: var(--space-10);
  border-radius: var(--radius-2xl);
  width: 100%;
  max-width: 440px;
  border: 1px solid var(--color-border-light);
  box-shadow: var(--shadow-card);
}

.login-header {
  text-align: center;
  margin-bottom: var(--space-8);
}

.login-title {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
  letter-spacing: -0.02em;
}

.login-subtitle {
  font-size: 14px;
  color: var(--color-text-muted);
  margin-top: var(--space-2);
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.login-btn {
  width: 100%;
  height: 48px;
  background: var(--color-primary);
  color: var(--color-text-inverse);
  border: none;
  border-radius: var(--radius-md);
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

.login-btn:hover {
  background: var(--color-primary-hover);
  transform: translateY(-1px);
}

.login-btn:disabled {
  background: var(--color-text-placeholder);
  cursor: not-allowed;
  transform: none;
}
```

- [ ] **Commit**

```bash
git add src/views/Login.vue
git commit -m "style(login): 登录页极简风格优化"
```

---

### 任务 7：优化按钮和输入框全局样式

**文件：**
- 修改：`src/styles/main.css`

- [ ] **步骤：更新按钮样式**

```css
/* 极简按钮样式覆盖 Element Plus */
.el-button--primary {
  background: var(--color-primary);
  border-color: var(--color-primary);
  box-shadow: none;
  border-radius: var(--radius-md);
}

.el-button--primary:hover {
  background: var(--color-primary-hover);
  border-color: var(--color-primary-hover);
  transform: translateY(-1px);
}

.el-button--primary:active {
  transform: translateY(0);
}

/* 减小所有按钮圆角 */
.el-button {
  border-radius: var(--radius-md);
  font-weight: 500;
}
```

- [ ] **步骤：更新输入框样式**

```css
:deep(.el-input__wrapper) {
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-input);
  border: 1px solid var(--color-border);
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px var(--color-text-muted) inset;
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2.5px rgb(14 165 233 / 0.1) inset;
  border-color: var(--color-primary);
}
```

- [ ] **Commit**

```bash
git add src/styles/main.css
git commit -m "style(elements): 按钮和输入框极简风格优化"
```

---

### 任务 8：优化动画和交互

**文件：**
- 修改：`src/styles/main.css`

- [ ] **步骤：简化动画**

```css
/* 移除或简化过于花哨的动画 */
@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 卡片 hover 效果简化 */
.el-card {
  transition: transform var(--duration-normal) var(--ease-out),
              box-shadow var(--duration-normal) var(--ease-out);
}

.el-card:hover {
  transform: translateY(-2px);
}
```

- [ ] **Commit**

```bash
git add src/styles/main.css
git commit -m "style(animations): 简化动画效果"
```

---

## 自检

### 1. 规格覆盖度
- ✅ CSS 变量更新 → 任务 1
- ✅ 侧边栏风格 → 任务 2
- ✅ 卡片组件 → 任务 3
- ✅ 数据表格 → 任务 4
- ✅ 页面组件 → 任务 5
- ✅ 登录页 → 任务 6
- ✅ 按钮和输入框 → 任务 7
- ✅ 动画优化 → 任务 8

### 2. 占位符扫描
- ✅ 无 TODO 或待定
- ✅ 所有步骤都有完整代码

### 3. 类型一致性
- ✅ 仅样式修改，不涉及类型

---

计划已完成并保存到 `docs/superpowers/plans/2026-04-28-minimalist-white-redesign.md`。两种执行方式：

**1. 子代理驱动（推荐）** - 每个任务调度一个新的子代理，任务间进行审查，快速迭代

**2. 内联执行** - 在当前会话中使用 executing-plans 执行任务，批量执行并设有检查点

选哪种方式？
