# 小程序抽屉式弹出设计规格

> **日期：** 2026-05-31
> **状态：** 待用户审查

## 目标

将小程序弹出方式从"全屏遮罩"改为"右侧覆盖式抽屉"，解决以下问题：
- 当前双层遮罩导致上下文丢失
- 小程序弹出后完全遮挡聊天内容
- 无法边看聊天边使用工具

## 设计方案

### 交互流程

```
用户点击小程序图标
    ↓
从右侧滑入抽屉面板（宽度 400px，可调整）
    ↓
抽屉覆盖在聊天区域右侧，带半透明遮罩
    ↓
用户可看到左侧聊天内容
    ↓
点击聊天区域或按 ESC 关闭抽屉
```

### 视觉规格

#### 1. 抽屉面板

| 属性 | 值 |
|------|-----|
| 位置 | `position: absolute; right: 0; top: 0; bottom: 0` |
| 宽度 | `400px`（默认），可拖拽调整 `300px - 600px` |
| 背景 | `var(--sidebar-bg)`（与侧边栏一致） |
| 阴影 | `box-shadow: -4px 0 24px rgba(0, 0, 0, 0.15)` |
| z-index | `100`（在聊天内容之上，在主面板之下） |
| 圆角 | 左上角 `12px`，其余 `0` |

#### 2. 遮罩层

| 属性 | 值 |
|------|-----|
| 位置 | 覆盖整个 `main-content-area` |
| 背景 | `rgba(0, 0, 0, 0.3)`（半透明，聊天内容隐约可见） |
| z-index | `99`（在抽屉之下） |
| 点击行为 | 关闭抽屉 |

#### 3. 动画

| 动画 | 持续时间 | 缓动函数 | 效果 |
|------|---------|---------|------|
| 滑入 | `0.3s` | `cubic-bezier(0.4, 0, 0.2, 1)` | `translateX(100%) → translateX(0)` |
| 滑出 | `0.2s` | `cubic-bezier(0.4, 0, 0.2, 1)` | `translateX(0) → translateX(100%)` |
| 遮罩淡入 | `0.2s` | `ease` | `opacity: 0 → 0.3` |

#### 4. 拖拽调整宽度

```
┌─────────────────────────────────────┐
│                                     │
│   聊天区域（隐约可见）                │ ║  ← 拖拽手柄
│                                     │    （宽度 8px）
│                                     │
└─────────────────────────────────────┘
```

- 拖拽手柄位置：抽屉左侧边缘
- 拖拽范围：`300px - 600px`
- 光标：`cursor: col-resize`
- 拖拽时显示宽度提示（可选）

### 组件结构

```
MiniAppDrawer.vue (新建)
├── 遮罩层 (.drawer-overlay)
│   └── 点击关闭
└── 抽屉面板 (.drawer-panel)
    ├── 标题栏 (.drawer-header)
    │   ├── 小程序图标 + 名称
    │   └── 关闭按钮
    ├── 拖拽手柄 (.drawer-resize-handle)
    └── 内容区 (.drawer-body)
        └── iframe (小程序内容)
```

### 文件变更

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `src/components/miniapp/MiniAppDrawer.vue` | 新建 | 抽屉组件 |
| `src/components/apps/MiniAppManager.vue` | 修改 | 替换 MiniAppLoader 为 MiniAppDrawer |
| `src/components/miniapp/MiniAppLoader.vue` | 保留 | 继续用于聊天消息中的小程序卡片点击 |

### 状态管理

```typescript
// MiniAppManager.vue 中的状态
const drawerOpen = ref(false)
const drawerWidth = ref(400)
const activeMiniApp = ref<MiniAppData | null>(null)

// 打开抽屉
function openDrawer(app: MiniAppData) {
  activeMiniApp.value = app
  drawerOpen.value = true
}

// 关闭抽屉
function closeDrawer() {
  drawerOpen.value = false
  setTimeout(() => { activeMiniApp.value = null }, 200) // 等待动画结束
}
```

### 键盘交互

| 按键 | 行为 |
|------|------|
| `ESC` | 关闭抽屉 |
| 无其他快捷键 | - |

### 边界情况处理

| 场景 | 处理方式 |
|------|---------|
| 屏幕宽度 < 768px | 抽屉宽度改为 `100%`，全屏覆盖 |
| 小程序加载失败 | 显示错误状态 + 重试按钮（与现有 MiniAppLoader 一致） |
| 快速切换小程序 | 先关闭当前抽屉，再打开新的（带过渡动画） |
| 拖拽超出范围 | 限制在 `300px - 600px` 范围内 |

### 与现有功能的兼容

| 功能 | 兼容性 | 说明 |
|------|-------|------|
| 聊天消息中的小程序卡片 | ✅ 兼容 | 点击仍使用 MiniAppLoader 全屏弹出 |
| 小程序列表 | ✅ 兼容 | 点击图标改为打开抽屉 |
| Bridge 通信 | ✅ 兼容 | iframe 通信机制不变 |
| 权限控制 | ✅ 兼容 | 权限检查逻辑不变 |

## 实现优先级

### 第一期（必须）
- [ ] 基础抽屉弹出/关闭
- [ ] 滑入/滑出动画
- [ ] 点击遮罩关闭
- [ ] ESC 键关闭

### 第二期（可选）
- [ ] 拖拽调整宽度
- [ ] 响应式适配（移动端全屏）
- [ ] 宽度记忆（localStorage）
