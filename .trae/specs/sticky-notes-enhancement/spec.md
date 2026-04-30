# 便签模块优化设计

## 概述

对 QIM 便签模块进行渐进式优化，包括界面改进、AI 能力增强和跨模块转化功能。

## 目标

- 优化便签卡片样式和交互体验
- 增加标签筛选和全屏编辑功能
- 添加 AI 能力（自动提取标签、智能分类、内容摘要）
- 支持便签转化为笔记或任务

## 非目标

- 不重构便签核心数据结构
- 不改变便签与聊天的关联方式

## 功能需求

### FR-1：界面优化

- 卡片样式优化，使用设计系统令牌
- 增加标签筛选器
- 增加全屏编辑模式
- 组件拆分，复用笔记模块的子组件模式

### FR-2：AI 能力

- 自动提取标签：根据便签内容推荐标签
- 智能分类：根据内容自动分类（工作、个人、想法等）
- 内容摘要：生成简短摘要
- 提取行动项：识别便签中的行动项

### FR-3：跨模块转化

- 便签 → 笔记：一键将便签转为笔记
- 便签 → 任务：提取行动项转为任务

## 数据模型改动

StickyNote 复用 Note 模型，已有字段：
- Tags：标签数组
- Summary：AI 生成的摘要

新增字段：
```typescript
interface StickyNote extends Note {
  category: string  // 智能分类
  color: string     // 颜色
  paperStyle: string // 纸张样式
  fontFamily: string // 字体
  reminder: string   // 提醒时间
}
```

## 后端 API 改动

| API | 方法 | 说明 |
|-----|------|------|
| `/api/v1/notes/:id/analyze` | POST | 复用笔记 AI 分析（已有） |
| `/api/v1/notes/:id/convert` | POST | 转化为笔记或任务 |

## 前端组件改动

### 主要改动文件

- `qim-client/src/components/apps/StickyNotesApp.vue` - 主组件重构

### 新建子组件

- `qim-client/src/components/apps/sticky/StickyNoteCard.vue` - 便签卡片组件
- `qim-client/src/components/apps/sticky/StickyNoteModal.vue` - 编辑弹窗组件
- `qim-client/src/components/apps/sticky/StickyTagFilter.vue` - 标签筛选器组件
- `qim-client/src/components/apps/sticky/StickyAIAnalysisModal.vue` - AI 分析弹窗

## 验收标准

### AC-1：标签筛选

- **给定**：便签列表中有多篇带标签的便签
- **当**：用户点击某个标签
- **则**：列表只显示包含该标签的便签

### AC-2：AI 分析

- **给定**：用户创建或编辑一个便签
- **当**：用户点击"AI 分析"按钮
- **则**：显示 AI 分析结果，包含标签、分类、摘要

### AC-3：全屏编辑

- **给定**：用户正在编辑便签
- **当**：用户点击全屏按钮
- **则**：进入全屏编辑模式

### AC-4：转化为笔记

- **给定**：用户有一个便签
- **当**：用户点击"转为笔记"按钮
- **则**：便签转化为笔记，出现在笔记列表中

## 改动量评估

| 改动项 | 前端 | 后端 | 复杂度 |
|--------|------|------|--------|
| 卡片样式优化 | CSS 调整 | 无 | 低 |
| 标签筛选 | 新增组件 | 无 | 低 |
| 全屏编辑 | 新增功能 | 无 | 低 |
| AI 分析 | 复用笔记组件 | 复用已有 API | 低 |
| 转化功能 | 新增按钮和逻辑 | 新增 API | 中 |

**总体评估**：增量式改动，复用笔记模块已有能力，预计 1-2 周完成。
