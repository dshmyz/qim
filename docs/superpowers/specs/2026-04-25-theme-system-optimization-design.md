# 主题系统优化设计文档

**日期:** 2026-04-25  
**作者:** AI Assistant  
**状态:** 待实施

## 概述

将分散在 themes.css 和 Main.vue 中的主题定义统一到 themes.css 中，消除重复定义，统一主题结构，减少硬编码颜色值。

## 当前问题

### 1. 重复定义

themes.css 和 Main.vue 中存在重复的主题 CSS 变量定义：
- `elegant-dark`: 两处都定义了完整的 CSS 变量
- `ocean-blue`: 两处都定义了完整的 CSS 变量  
- `warm-amber`: 两处都定义了完整的 CSS 变量

### 2. 不一致性

- Main.vue 中有 `sacredyellow`、`urban-jungle`、`mediterranean-dream`、`monochrome-elegance`、`spring-blossom` 等主题在 themes.css 中不存在
- 主题命名不一致（如 `sacredyellow` 可能应为 `warm-amber`）

### 3. 硬编码颜色

Main.vue 中大量组件样式使用硬编码颜色值而非 CSS 变量，导致主题切换时部分样式不跟随变化。

## 目标结构

```
themes.css (目标 1500-2000行)     ← 所有主题的唯一数据源
Main.vue (目标 6000-6500行)       ← 只保留布局组件逻辑，无主题样式
```

## 执行方案

### 阶段一：分析

扫描 Main.vue 中所有 `[data-theme="..."]` 选择器，分类为：
- CSS变量定义（重复的，应删除）
- 组件特定样式（应迁移到 themes.css）
- 注释掉的样式（应清理）

### 阶段二：迁移

将 Main.vue 中的主题样式追加到 themes.css，确保：
- 不重复已有的定义
- 统一命名
- 保持注释一致
- 使用 CSS 变量替代硬编码颜色

### 阶段三：清理

从 Main.vue 中删除所有已迁移的主题样式。

### 阶段四：验证

确保所有主题功能正常，无样式丢失。

## 子代理分工

- **子代理1:** 分析 Main.vue 主题样式并生成迁移清单
- **子代理2:** 迁移优雅深色/海洋蓝主题
- **子代理3:** 迁移高雅紫/琥珀黄/中国红/青草绿主题
- **子代理4:** 迁移其他主题（都市丛林、地中海等）
- **子代理5:** 清理 Main.vue 并验证

## 风险控制

- 每次迁移一个主题，迁移后立即验证
- 保留 Main.vue 的 git 历史作为备份
- 使用 Git 分支进行开发

## 验收标准

1. themes.css 包含所有主题的完整定义
2. Main.vue 中无 `[data-theme="..."]` 选择器
3. 所有主题切换功能正常
4. 无样式丢失或错位
