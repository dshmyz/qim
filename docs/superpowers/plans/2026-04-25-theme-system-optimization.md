# 主题系统优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将所有主题定义从 Main.vue 统一到 themes.css，消除重复，统一结构，减少硬编码颜色值。

**架构：** 将 themes.css 作为所有主题的唯一数据源，包含 CSS 变量定义和组件特定样式。从 Main.vue 中删除所有 `[data-theme="..."]` 选择器。

**技术栈：** Vue 3, CSS, CSS 变量（Custom Properties）

**当前状态分析：**
- themes.css: 480行，包含 6 个主题（elegant-dark, ocean-blue, elegantpurple, warm-amber, chinesered, grassgreen）
- Main.vue: ~11000行，包含 12 个主题的相关样式，其中 6 个与 themes.css 重复
- 重复主题：elegant-dark, ocean-blue, elegantpurple, chinesered, grassgreen 的 CSS 变量定义
- 命名冲突：Main.vue 中有 sacredyellow（注释为"月牙黄"），但 themes.css 中用 warm-amber 表示琥珀黄
- Main.vue 独有主题：modern-light, sacredyellow, urban-jungle, mediterranean-dream, monochrome-elegance, spring-blossom

**验收标准：**
1. themes.css 包含所有主题的完整定义
2. Main.vue 中无 `[data-theme="..."]` 选择器
3. 所有主题切换功能正常
4. 无样式丢失或错位

---

### 任务 1：迁移 modern-light 主题到 themes.css

**文件：**
- 修改：`src/assets/styles/themes.css` - 追加 modern-light 主题定义
- 修改：`src/views/Main.vue:6666-6688` - 删除重复的 modern-light CSS 变量定义

- [ ] **步骤 1：读取 Main.vue 中 modern-light 的完整定义**

读取 `/Users/gracegaoya/work/project/qim/qim-client/src/views/Main.vue` 第 6667-6688 行，获取 modern-light 主题的 CSS 变量定义。

- [ ] **步骤 2：追加到 themes.css**

在 themes.css 末尾追加 modern-light 主题定义：

```css
/* 现代浅色主题 */
[data-theme="modern-light"] {
  --primary-color: #3b82f6;
  --primary-light: #eff6ff;
  --secondary-color: #f9fafb;
  --text-color: #1f2937;
  --border-color: #e5e7eb;
  --hover-color: #eff6ff;
  --active-color: #2563eb;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #f3f4f6;
  --accent-color: #60a5fa;
  --text-secondary: #6b7280;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  --panel-bg: #ffffff;
  --list-bg: #fafafa;
  --card-bg: #ffffff;
  --header-panel-bg: #f8fafc;
  --content-bg: #f9fafb;
}
```

- [ ] **步骤 3：从 Main.vue 中删除**

删除 Main.vue 第 6666-6688 行：
```css
/* 现代浅色主题 */
[data-theme="modern-light"] {
  --primary-color: #3b82f6;
  /* ... 删除整个主题块 ... */
}
```

- [ ] **步骤 4：Commit**

```bash
git add src/assets/styles/themes.css src/views/Main.vue
git commit -m "refactor: 迁移 modern-light 主题到 themes.css"
```

---

### 任务 2：迁移 sacredyellow 主题到 themes.css

**文件：**
- 修改：`src/assets/styles/themes.css` - 追加 sacredyellow 主题定义（注意：Main.vue 中注释为"月牙黄"，但使用 sacredyellow 作为主题名）
- 修改：`src/views/Main.vue:7447-7493` - 删除 sacredyellow 主题定义

**注意：** sacredyellow 和 warm-amber 都指向"琥珀黄"主题。保留 sacredyellow 作为独立主题名，以保持与 UI 中主题选择器的一致性。

- [ ] **步骤 1：追加 sacredyellow 到 themes.css**

```css
/* 月牙黄主题（琥珀黄） */
[data-theme="sacredyellow"] {
  --primary-color: #d4b85f;
  --primary-light: #fffef8;
  --secondary-color: #fffef8;
  --text-color: #6b5a2f;
  --border-color: #f0e6c8;
  --hover-color: #fffef8;
  --active-color: #c9a85a;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #fffef8;
  --accent-color: #e8d4a0;
  --text-secondary: #8b7a50;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.03);
  --shadow-md: 0 2px 4px -1px rgba(0, 0, 0, 0.05), 0 1px 2px -1px rgba(0, 0, 0, 0.03);
  --panel-bg: #ffffff;
  --list-bg: #fffef8;
  --card-bg: #ffffff;
  --header-panel-bg: #fffef8;
}

/* 月牙黄主题（琥珀黄） - 左边侧边栏 */
[data-theme="sacredyellow"] .side-options {
  background: linear-gradient(135deg, #e8d4a0 0%, #f0e2b8 100%);
}

/* 月牙黄主题（琥珀黄） - 文本颜色 */
[data-theme="sacredyellow"] .window-title,
[data-theme="sacredyellow"] .option-item,
[data-theme="sacredyellow"] .option-item.active {
  color: #5a4a25;
  text-shadow: 0 1px 1px rgba(255, 255, 255, 0.6);
}

/* 月牙黄主题（琥珀黄） - 侧边栏头部 */
[data-theme="sacredyellow"] .sidebar-header .user-name {
  color: var(--text-color);
  text-shadow: none;
}

/* 月牙黄主题（琥珀黄） - 窗口控制栏左侧 */
[data-theme="sacredyellow"] .window-controls-left {
  background: linear-gradient(135deg, #e8d4a0 0%, #f0e2b8 100%);
}
```

- [ ] **步骤 2：从 Main.vue 中删除 sacredyellow**

删除 Main.vue 第 7447-7493 行：
```css
[data-theme="sacredyellow"] {
  /* ... 删除整个主题块 ... */
}

/* 月牙黄主题 - 左边侧边栏 */
/* ... 删除所有 sacredyellow 相关样式 ... */
```

- [ ] **步骤 3：Commit**

```bash
git add src/assets/styles/themes.css src/views/Main.vue
git commit -m "refactor: 迁移 sacredyellow 主题到 themes.css"
```

---

### 任务 3：迁移 elegant-dark 的组件样式到 themes.css

**文件：**
- 修改：`src/assets/styles/themes.css` - 追加 elegant-dark 独有组件样式
- 修改：`src/views/Main.vue:7220-11854` - 删除 elegant-dark 组件样式（跨多个行范围）

**注意：** elegant-dark 的 CSS 变量已存在于 themes.css，只需迁移组件样式。

- [ ] **步骤 1：收集 elegant-dark 在 Main.vue 中的所有样式**

需要迁移的选择器（从 Main.vue）：
- `.search-popup-item:hover` (L7227)
- `.search-popup-status.online` (L7231)
- `.search-popup-status.offline` (L7236)
- `.markdown-link:hover` (L7331)
- `.preview-content` (L7335)
- `.app-item:hover` (L7339)
- `.category-app-item:hover` (L7353)
- `.recent-app-grid-item:hover` (L7359)
- `.main-content` (L7364)
- `.no-conversation` (L7369)
- `.apps-content` (L7375)
- `.settings-content` (L11296)
- `.settings-header` (L11302)
- `.settings-header h3` (L11308)
- `.settings-sidebar` (L11312)
- `.settings-sidebar-item:hover` (L11317)
- `.settings-sidebar-item.active` (L11321)
- `.save-btn` (L11327)
- `.settings-main` (L11332)
- `.settings-section-header h4` (L11336)
- `icon-btn:hover` (L11340)
- `.settings-item label` (L11344)
- `.settings-input, .settings-textarea, .settings-select` (L11348-11350)
- `.settings-input:focus, .settings-textarea:focus, .settings-select:focus` (L11356-11358)
- `.theme-option:hover` (L11363)
- `.theme-option.active` (L11367)
- `.clear-cache-btn, .security-btn` (L11372-11373)
- `.message-list` (L11565)
- `.clear-cache-btn:hover, .security-btn:hover` (L11845-11846)
- `.about-info` (L11850)
- `.settings-footer` (L11854)

- [ ] **步骤 2：追加到 themes.css**

在 themes.css 的 elegant-dark 部分追加上述所有样式。

- [ ] **步骤 3：从 Main.vue 中删除**

删除所有 `[data-theme="elegant-dark"]` 的选择器定义。

- [ ] **步骤 4：Commit**

```bash
git add src/assets/styles/themes.css src/views/Main.vue
git commit -m "refactor: 迁移 elegant-dark 组件样式到 themes.css"
```

---

### 任务 4：迁移 ocean-blue 组件样式到 themes.css

**文件：**
- 修改：`src/assets/styles/themes.css` - 追加 ocean-blue 独有组件样式
- 修改：`src/views/Main.vue:6772-6777` - 删除 ocean-blue 组件样式

- [ ] **步骤 1：追加到 themes.css**

```css
/* 海洋蓝主题 - 侧边栏头部 */
[data-theme="ocean-blue"] .sidebar-header .user-name {
  color: var(--text-color);
  text-shadow: none;
}
```

- [ ] **步骤 2：从 Main.vue 中删除**

删除 Main.vue 第 6772-6777 行。

- [ ] **步骤 3：Commit**

```bash
git add src/assets/styles/themes.css src/views/Main.vue
git commit -m "refactor: 迁移 ocean-blue 组件样式到 themes.css"
```

---

### 任务 5：迁移优雅深色主题重复样式到 themes.css

**文件：**
- 修改：`src/assets/styles/themes.css` - 检查并统一 elegant-dark 定义
- 修改：`src/views/Main.vue:6791-6899` - 删除重复的 elegant-dark 组件样式

- [ ] **步骤 1：对比 themes.css 和 Main.vue 中的 elegant-dark 样式**

themes.css 已有：
- `.sidebar-header` (L123-127)
- `.sidebar-header .user-name` (L129-131)
- `.org-content` (L134-136)
- `.employee-node .tree-node-content` (L138-141)
- `.employee-node .tree-node-content:hover` (L143-146)
- `.option-icon` (L149-151)
- `.option-item:hover .option-icon, .option-item.active .option-icon` (L153-156)
- 图标颜色列表 (L158-173)
- `.app-item:hover .app-icon` 等 (L175-181)
- `.option-item` (L184-186)
- `.option-item:hover` (L188-192)
- `.option-item.active` (L194-198)
- `.app-tab-item` (L201-204)
- `.app-tab-item:hover` (L206-209)
- `.app-tab-item.active` (L211-214)
- `.group-badge` (L217-221)
- `.right-content` (L224-226)
- `.right-content-header` (L228-232)
- `.right-content-body` (L234-237)
- `.apps-content` (L240-242)
- `.recent-apps-section, .all-apps-section` (L244-247)
- `.app-category-item` (L250-253)
- `.category-header` (L255-257)

Main.vue 中重复的需要删除：
- `.sidebar-header .user-name` (L6781-6783) - 重复
- `.employee-node .tree-node-content` (L6793-6796) - 重复
- `.employee-node .tree-node-content:hover` (L6798-6801) - 重复
- `.option-icon` (L6804-6806) - 重复
- `.option-item:hover .option-icon, .option-item.active .option-icon` (L6808-6811) - 重复
- 图标颜色列表 (L6813-6835) - 重复
- `.option-item` (L6838-6840) - 重复
- `.option-item:hover` (L6842-6846) - 重复
- `.option-item.active` (L6848-6852) - 重复
- `.app-tab-item` (L6855-6858) - 重复
- `.app-tab-item:hover` (L6860-6863) - 重复
- `.app-tab-item.active` (L6865-6868) - 重复
- `.right-content` (L6873-6875) - 重复
- `.right-content-header` (L6877-6881) - 重复
- `.right-content-body` (L6883-6886) - 重复
- `.apps-content` (L6889-6891) - 重复
- `.recent-apps-section, .all-apps-section` (L6893-6896) - 重复

- [ ] **步骤 2：从 Main.vue 中删除重复样式**

删除 L6781-6896 范围内的所有重复样式。

- [ ] **步骤 3：Commit**

```bash
git add src/views/Main.vue
git commit -m "refactor: 删除 Main.vue 中重复的 elegant-dark 样式"
```

---

### 任务 6：迁移 urban-jungle 主题到 themes.css

**文件：**
- 修改：`src/assets/styles/themes.css` - 追加完整的 urban-jungle 主题定义
- 修改：`src/views/Main.vue:7591-7608` - 删除 urban-jungle 样式

- [ ] **步骤 1：在 themes.css 中追加 urban-jungle 主题**

```css
/* 都市丛林主题 */
[data-theme="urban-jungle"] {
  --primary-color: #27ae60;
  --primary-light: #e8f8f0;
  --secondary-color: #f0fff4;
  --text-color: #1a4a2e;
  --border-color: #c6e6d5;
  --hover-color: #dcf5e8;
  --active-color: #219150;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #e8f8f0;
  --accent-color: #2ecc71;
  --text-secondary: #2e6b4a;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.03);
  --shadow-md: 0 2px 4px -1px rgba(0, 0, 0, 0.05), 0 1px 2px -1px rgba(0, 0, 0, 0.03);
  --panel-bg: #ffffff;
  --list-bg: #f0fff4;
  --card-bg: #ffffff;
  --header-panel-bg: #e8f8f0;
}

/* 都市丛林主题 - 左边侧边栏 */
[data-theme="urban-jungle"] .side-options {
  background: linear-gradient(135deg, #27ae60 0%, #2ecc71 100%);
}

/* 都市丛林主题 - 文本颜色 */
[data-theme="urban-jungle"] .window-title,
[data-theme="urban-jungle"] .option-item,
[data-theme="urban-jungle"] .option-item.active {
  color: #ffffff;
}

/* 都市丛林主题 - 窗口控制栏左侧 */
[data-theme="urban-jungle"] .window-controls-left {
  background: linear-gradient(135deg, #27ae60 0%, #2ecc71 100%);
}
```

- [ ] **步骤 2：从 Main.vue 中删除**

删除 Main.vue 第 7591-7608 行。

- [ ] **步骤 3：Commit**

```bash
git add src/assets/styles/themes.css src/views/Main.vue
git commit -m "refactor: 迁移 urban-jungle 主题到 themes.css"
```

---

### 任务 7：迁移 mediterranean-dream 主题到 themes.css

**文件：**
- 修改：`src/assets/styles/themes.css` - 追加完整的 mediterranean-dream 主题定义
- 修改：`src/views/Main.vue:7630-7647` - 删除 mediterranean-dream 样式

- [ ] **步骤 1：在 themes.css 中追加 mediterranean-dream 主题**

```css
/* 地中海主题 */
[data-theme="mediterranean-dream"] {
  --primary-color: #c0392b;
  --primary-light: #fde8e8;
  --secondary-color: #f0f9ff;
  --text-color: #1e3a5f;
  --border-color: #bae6fd;
  --hover-color: #dbeafe;
  --active-color: #e74c3c;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #fde8e8;
  --accent-color: #3498db;
  --text-secondary: #475569;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  --panel-bg: #ffffff;
  --list-bg: #f0f9ff;
  --card-bg: #ffffff;
  --header-panel-bg: #fde8e8;
}

/* 地中海主题 - 左边侧边栏 */
[data-theme="mediterranean-dream"] .side-options {
  background: linear-gradient(135deg, #c0392b 0%, #3498db 100%);
}

/* 地中海主题 - 文本颜色 */
[data-theme="mediterranean-dream"] .window-title,
[data-theme="mediterranean-dream"] .option-item,
[data-theme="mediterranean-dream"] .option-item.active {
  color: #ffffff;
}

/* 地中海主题 - 窗口控制栏左侧 */
[data-theme="mediterranean-dream"] .window-controls-left {
  background: linear-gradient(135deg, #c0392b 0%, #3498db 100%);
}
```

- [ ] **步骤 2：从 Main.vue 中删除**

删除 Main.vue 第 7630-7647 行。

- [ ] **步骤 3：Commit**

```bash
git add src/assets/styles/themes.css src/views/Main.vue
git commit -m "refactor: 迁移 mediterranean-dream 主题到 themes.css"
```

---

### 任务 8：迁移 monochrome-elegance 主题到 themes.css

**文件：**
- 修改：`src/assets/styles/themes.css` - 追加完整的 monochrome-elegance 主题定义
- 修改：`src/views/Main.vue:7649-7666` - 删除 monochrome-elegance 样式

- [ ] **步骤 1：在 themes.css 中追加 monochrome-elegance 主题**

```css
/* 单色雅主题 */
[data-theme="monochrome-elegance"] {
  --primary-color: #333333;
  --primary-light: #f5f5f5;
  --secondary-color: #fafafa;
  --text-color: #1f2937;
  --border-color: #e5e7eb;
  --hover-color: #f3f4f6;
  --active-color: #4b5563;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #f5f5f5;
  --accent-color: #666666;
  --text-secondary: #6b7280;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  --panel-bg: #ffffff;
  --list-bg: #fafafa;
  --card-bg: #ffffff;
  --header-panel-bg: #f5f5f5;
}

/* 单色雅主题 - 左边侧边栏 */
[data-theme="monochrome-elegance"] .side-options {
  background: linear-gradient(135deg, #333333 0%, #666666 100%);
}

/* 单色雅主题 - 文本颜色 */
[data-theme="monochrome-elegance"] .window-title,
[data-theme="monochrome-elegance"] .option-item,
[data-theme="monochrome-elegance"] .option-item.active {
  color: #ffffff;
}

/* 单色雅主题 - 窗口控制栏左侧 */
[data-theme="monochrome-elegance"] .window-controls-left {
  background: linear-gradient(135deg, #333333 0%, #666666 100%);
}
```

- [ ] **步骤 2：从 Main.vue 中删除**

删除 Main.vue 第 7649-7666 行。

- [ ] **步骤 3：Commit**

```bash
git add src/assets/styles/themes.css src/views/Main.vue
git commit -m "refactor: 迁移 monochrome-elegance 主题到 themes.css"
```

---

### 任务 9：迁移 spring-blossom 主题到 themes.css

**文件：**
- 修改：`src/assets/styles/themes.css` - 追加完整的 spring-blossom 主题定义
- 修改：`src/views/Main.vue:7668-7685` - 删除 spring-blossom 样式

- [ ] **步骤 1：在 themes.css 中追加 spring-blossom 主题**

```css
/* 春日花主题 */
[data-theme="spring-blossom"] {
  --primary-color: #f8bbd9;
  --primary-light: #fce4ec;
  --secondary-color: #f3e5f5;
  --text-color: #4a148c;
  --border-color: #f8bbd9;
  --hover-color: #fce4ec;
  --active-color: #e1bee7;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #fce4ec;
  --accent-color: #e1bee7;
  --text-secondary: #7b1fa2;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  --panel-bg: #ffffff;
  --list-bg: #fce4ec;
  --card-bg: #ffffff;
  --header-panel-bg: #fce4ec;
}

/* 春日花主题 - 左边侧边栏 */
[data-theme="spring-blossom"] .side-options {
  background: linear-gradient(135deg, #f8bbd9 0%, #e1bee7 100%);
}

/* 春日花主题 - 文本颜色 */
[data-theme="spring-blossom"] .window-title,
[data-theme="spring-blossom"] .option-item,
[data-theme="spring-blossom"] .option-item.active {
  color: #4a148c;
}

/* 春日花主题 - 窗口控制栏左侧 */
[data-theme="spring-blossom"] .window-controls-left {
  background: linear-gradient(135deg, #f8bbd9 0%, #e1bee7 100%);
}
```

- [ ] **步骤 2：从 Main.vue 中删除**

删除 Main.vue 第 7668-7685 行。

- [ ] **步骤 3：Commit**

```bash
git add src/assets/styles/themes.css src/views/Main.vue
git commit -m "refactor: 迁移 spring-blossom 主题到 themes.css"
```

---

### 任务 10：清理 Main.vue 中剩余的主题样式

**文件：**
- 修改：`src/views/Main.vue` - 删除所有剩余的 `[data-theme="..."]` 选择器

- [ ] **步骤 1：搜索 Main.vue 中所有剩余的 `[data-theme="..."]`**

运行以下命令确认是否还有残留：

```bash
grep -n '\[data-theme=' src/views/Main.vue
```

预期输出：无匹配行（或仅有被注释掉的代码）

- [ ] **步骤 2：删除任何残留的主题样式**

如果步骤 1 发现残留，使用 SearchReplace 工具逐个删除。

- [ ] **步骤 3：验证 Main.vue 的 CSS 部分完整性**

确保删除主题样式时没有破坏其他 CSS 选择器。检查相邻的 CSS 规则是否仍有正确的开闭括号。

- [ ] **步骤 4：Commit**

```bash
git add src/views/Main.vue
git commit -m "refactor: 清理 Main.vue 中所有主题相关样式"
```

---

### 任务 11：清理 themes.css 中 warm-amber 和 sacredyellow 的冲突

**文件：**
- 修改：`src/assets/styles/themes.css` - 统一 warm-amber 和 sacredyellow 的命名和样式

- [ ] **步骤 1：确认 warm-amber 和 sacredyellow 的区别**

- warm-amber (琥珀黄): 橙色到红色渐变 `#f4a900 → #c1666b`
- sacredyellow (月牙黄): 浅黄色渐变 `#e8d4a0 → #f0e2b8`

这两个是不同的主题，需要保留两者，但确保注释和命名一致。

- [ ] **步骤 2：统一主题注释格式**

确保所有主题注释格式一致：`/* 主题名主题 */`

- [ ] **步骤 3：Commit**

```bash
git add src/assets/styles/themes.css
git commit -m "refactor: 统一主题注释格式和命名"
```

---

### 任务 12：修复 themes.css 中悬停颜色与背景色相同的问题

**文件：**
- 修改：`src/assets/styles/themes.css` - 调整部分主题的 `--hover-color` 值

- [ ] **步骤 1：识别悬停颜色问题**

需要修改的主题：
- sacredyellow: `--hover-color: #fffef8` 与 `--list-bg: #fffef8` 相同
- warm-amber: 已修改为 `#f5ecd5` (之前任务已完成)
- chinesered: 已修改为 `#ffe8e8` (之前任务已完成)
- grassgreen: 已修改为 `#dcf5e8` (之前任务已完成)
- elegantpurple: 已修改为 `#ebd6fa` (之前任务已完成)

- [ ] **步骤 2：修复 sacredyellow 的 hover-color**

```css
/* 修改前 */
--hover-color: #fffef8;
/* 修改后 */
--hover-color: #f5ecd5;
```

- [ ] **步骤 3：Commit**

```bash
git add src/assets/styles/themes.css
git commit -m "fix: 修复 sacredyellow 主题悬停颜色与背景色相同的问题"
```

---

### 任务 13：最终验证和测试

**文件：**
- 读取：`src/assets/styles/themes.css` - 验证完整性
- 读取：`src/views/Main.vue` - 验证清理完成

- [ ] **步骤 1：验证 themes.css 结构**

检查 themes.css 是否包含以下所有主题：
1. modern-light
2. elegant-dark
3. ocean-blue
4. elegantpurple
5. sacredyellow
6. warm-amber
7. chinesered
8. grassgreen
9. urban-jungle
10. mediterranean-dream
11. monochrome-elegance
12. spring-blossom

- [ ] **步骤 2：验证 Main.vue 无主题选择器**

运行：
```bash
grep -c '\[data-theme=' src/views/Main.vue
```
预期输出：0

- [ ] **步骤 3：检查 CSS 语法**

确保 themes.css 和 Main.vue 中的 CSS 语法正确，无未闭合的大括号。

- [ ] **步骤 4：启动开发服务器验证**

```bash
cd /Users/gracegaoya/work/project/qim/qim-client && npm run dev
```

检查应用是否正常启动，主题切换功能是否工作。

- [ ] **步骤 5：Commit**

```bash
git add .
git commit -m "chore: 主题系统优化完成 - 统一主题定义到 themes.css"
```
