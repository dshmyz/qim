# 频道页面布局重新设计文档

**日期**: 2026-05-03  
**状态**: 待审查  
**设计风格**: 内容优先 · 统一风格

## 概述

本次重新设计旨在优化频道页面的整体布局，使其与其他tab页保持风格统一，同时保留频道功能的特色，提升用户体验和视觉效果。

## 设计目标

1. **统一风格**: 与其他tab页的侧边栏样式保持一致
2. **优化布局**: 调整空间分配，提升内容展示效果
3. **改善体验**: 优化信息层次和交互流程
4. **保留特色**: 保持频道功能的独特性

## 设计决策

### 1. 侧边栏统一规格

**与其他tab页保持一致：**
- 侧边栏宽度：320px
- Header高度：72px
- Header padding：16px 20px
- 搜索框padding：12px 8px
- 列表项padding：10px 12px
- 头像尺寸：40px

**选择理由：**
- 保持整体应用风格的统一性
- 用户在不同tab页之间切换时体验一致
- 降低学习成本，提升用户熟悉度

### 2. 频道特色保留

**差异化设计：**
- 标题样式：16px 加粗（vs 用户名14px）
- 头像形状：圆角8px（vs 圆形）
- 标签切换：订阅/广场切换
- 列表项信息：消息数量（vs 最近消息）
- 创建按钮：蓝色实心（vs 透明）

**选择理由：**
- 突出频道功能的独特性
- 区分频道与聊天列表的不同用途
- 保持功能的清晰性和辨识度

### 3. 右侧内容区优化

**布局规格：**
- Header高度：72px
- Header padding：16px 20px
- 工具栏高度：53px
- 工具栏padding：12px 16px
- 内容区padding：20px
- 内容区背景：#f8f9fa

**选择理由：**
- 与侧边栏header高度一致
- 提供舒适的阅读体验
- 清晰的信息层次

## 组件设计细节

### 侧边栏组件 (ChannelSidebar.vue)

**Header区域（72px）：**
```
┌─────────────────────────────────┐
│ 频道                    [+]     │
│ padding: 16px 20px              │
└─────────────────────────────────┘
```

**搜索和切换区域：**
```
┌─────────────────────────────────┐
│ [搜索框]                        │
│ padding: 12px 8px               │
│ [订阅] [广场]                   │
└─────────────────────────────────┘
```

**列表项：**
```
┌─────────────────────────────────┐
│ [头像] 标题                     │
│        消息数量                 │
│ padding: 10px 12px              │
└─────────────────────────────────┘
```

**样式要点：**
- 头像：40x40px，圆角8px
- 标题：14px，font-weight: 500
- 消息数量：12px，color: #6c757d
- 列表项间距：4px
- 选中状态：background: #f8f9fa

### 内容区Header组件 (ChannelHeader.vue)

**布局结构（72px）：**
```
┌─────────────────────────────────────────┐
│ [头像] 标题              [订阅按钮]    │
│        描述                              │
│ padding: 16px 20px                      │
└─────────────────────────────────────────┘
```

**样式要点：**
- 头像：44x44px，圆角10px
- 标题：18px，font-weight: 600
- 描述：13px，color: #6c757d
- 按钮：padding: 10px 20px，background: #3385ff

### 工具栏组件 (MessageList.vue)

**布局结构（53px）：**
```
┌─────────────────────────────────────────┐
│ 标题          [模式] [排序]             │
│ padding: 12px 16px                      │
└─────────────────────────────────────────┘
```

**样式要点：**
- 标题：16px，font-weight: 600
- 按钮：padding: 6px 12px
- 背景：white
- 边框：1px solid #e9ecef

### 消息卡片组件 (MessageCard.vue)

**布局结构：**
```
┌─────────────────────────────────────────┐
│ [头像] 发送者 · 时间                    │
│        消息内容                          │
│ padding: 16px                           │
│ border-radius: 10px                     │
└─────────────────────────────────────────┘
```

**样式要点：**
- 头像：36x36px，圆角8px
- 发送者：14px，font-weight: 600
- 时间：12px，color: #6c757d
- 内容：13px，line-height: 1.6
- 卡片间距：12px
- 阴影：0 1px 3px rgba(0,0,0,0.08)

## 颜色规范

### 主题色
- **主色**: #3385ff (蓝色)
- **主色深**: #2968cc (悬停状态)

### 文字颜色
- **主文字**: #2c3e50
- **次要文字**: #6c757d
- **禁用文字**: #adb5bd

### 背景颜色
- **页面背景**: #f8f9fa
- **卡片背景**: #ffffff
- **边框颜色**: #e9ecef
- **hover背景**: var(--hover-color)

## 间距规范

### 侧边栏间距
- Header padding: 16px 20px
- 搜索框 padding: 12px 8px
- 列表项 padding: 10px 12px
- 列表项间距: 4px

### 内容区间距
- Header padding: 16px 20px
- 工具栏 padding: 12px 16px
- 内容区 padding: 20px
- 卡片 padding: 16px
- 卡片间距: 12px

## 实现范围

### 需要修改的文件

1. **`src/assets/styles/modules/channel.css`**
   - 职责：全局频道样式
   - 修改：调整侧边栏宽度为320px，统一间距和样式

2. **`src/components/channel/ChannelSidebar.vue`**
   - 职责：频道侧边栏组件
   - 修改：调整header高度和padding，优化列表项样式

3. **`src/components/channel/ChannelHeader.vue`**
   - 职责：频道头部组件
   - 修改：调整header高度和padding，优化布局

4. **`src/components/channel/MessageList.vue`**
   - 职责：消息列表组件
   - 修改：调整工具栏高度和padding，优化内容区

5. **`src/components/channel/MessageCard.vue`**
   - 职责：消息卡片组件
   - 修改：调整卡片样式和间距

6. **`src/components/channel/ChannelCard.vue`**
   - 职责：频道卡片组件
   - 修改：调整卡片样式和头像形状

### 实现要点

1. **统一侧边栏规格**：
   - 宽度：320px
   - Header高度：72px
   - Header padding：16px 20px
   - 搜索框padding：12px 8px

2. **优化列表项**：
   - padding：10px 12px
   - 头像：40x40px，圆角8px
   - 标题：14px
   - 描述：12px

3. **统一header高度**：
   - ChannelHeader：72px
   - MessageList toolbar：53px
   - 与侧边栏header对齐

4. **优化内容区**：
   - 内容区padding：20px
   - 卡片padding：16px
   - 卡片圆角：10px
   - 卡片阴影：轻量

## 验收标准

1. ✅ 侧边栏宽度为320px，与其他tab页一致
2. ✅ Header高度为72px，与其他tab页一致
3. ✅ 列表项样式与其他tab页一致
4. ✅ 右侧内容区header高度统一
5. ✅ 所有间距符合设计规范
6. ✅ 颜色使用符合设计系统
7. ✅ 响应式布局正常工作
8. ✅ 所有主题下显示正常

## 后续优化建议

1. **响应式优化**: 移动端侧边栏可折叠
2. **动画优化**: 添加必要的过渡动画
3. **性能优化**: 虚拟滚动优化长列表
4. **无障碍**: 添加ARIA标签和键盘导航

## 参考资料

- 设计原型: `.superpowers/brainstorm/33728-1777756127/design-spec-unified.html`
- 现有代码: `src/components/channel/`
- 设计系统: `src/assets/styles/design-tokens.css`
- 其他tab页: `src/components/layout/Sidebar.vue`
