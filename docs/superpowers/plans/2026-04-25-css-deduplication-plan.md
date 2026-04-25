# Main.vue 样式重复清理和迁移计划

## 背景

分析发现存在大量样式重复定义：
- **Main.vue** 与各 CSS 文件重复：149 个类
- **多个 CSS 文件之间**重复：15 个类

## 原则

**用户指定：Main.vue 里的样式为准**

## 任务分解

### 任务 1：清理 CSS 文件之间的重复定义
处理在多个 CSS 文件中定义的 15 个类的冲突：

| 类名 | 冲突文件 | 决策 |
|------|---------|------|
| .active | main.css, components.css, layout.css | 保留 components.css |
| .badge, .badge-* | main.css, components.css | 保留 components.css |
| .btn, .btn-primary, .btn-secondary | main.css, components.css | 保留 components.css |
| .close-btn | layout.css, dialogs.css | 保留 dialogs.css |
| .divider | components.css, menus.css | 保留 components.css |
| .input | main.css, components.css | 保留 components.css |
| .loading, .loading-spinner | main.css, components.css, dialogs.css | 保留 components.css |
| .progress-bar | components.css, dialogs.css | 保留 dialogs.css |

### 任务 2：处理 Main.vue 与 main.css 的重复（13个类）
- .active, .bot-badge, .btn, .btn-primary, .btn-secondary
- .conversation-item, .file-item, .loading-spinner, .muted-icon
- .org-content, .theme-icon, .tree-node-content, .unread-badge
- **决策**：保留 Main.vue，清理 main.css

### 任务 3：处理 Main.vue 与 layout.css 的重复（23个类）
- .active, .all-apps-section, .apps-content, .close-btn
- .content-area, .content-section, .empty-content, .empty-icon
- .empty-state, .im-container, .main-content, .main-content-area
- .maximize-btn, .minimize-btn, .option-icon, .right-content
- .right-content-body, .right-content-header, .side-options
- .window-control-btn, .window-controls, .window-controls-right
- .window-controls-spacer
- **决策**：保留 Main.vue，清理 layout.css

### 任务 4：处理 Main.vue 与 dialogs.css 的重复（70个类）
- 对话框、用户资料、更新、语音通话等相关样式
- **决策**：保留 Main.vue，清理 dialogs.css

### 任务 5：处理 Main.vue 与 components.css 的重复（11个类）
- .active, .btn, .btn-primary, .btn-secondary, .card
- .checkbox, .divider, .icon-btn, .loading-spinner
- .progress-bar, .switch
- **决策**：保留 Main.vue，清理 components.css

### 任务 6：处理 Main.vue 与 menus.css 的重复（11个类）
- .action-menu, .action-menu-icon, .action-menu-item
- .context-menu, .context-menu-divider, .context-menu-icon
- .context-menu-item, .divider, .user-context-menu
- .user-context-menu-icon, .user-context-menu-item
- **决策**：保留 Main.vue，清理 menus.css

### 任务 7：处理 Main.vue 与 markdown.css 的重复（10个类）
- .markdown-*, 保留 Main.vue（样式更完整）

## 执行顺序

1. 先处理 CSS 文件之间的冲突
2. 逐个清理其他 CSS 文件与 Main.vue 的重复
3. 最后确保 Main.vue 中的样式完整

## 工作目录

`/Users/gracegaoya/work/project/qim/qim-client`
