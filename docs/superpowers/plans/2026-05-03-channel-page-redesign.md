# 频道页面样式优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 优化频道页面样式，采用现代简约风格，提升视觉吸引力和用户体验

**架构：** 通过修改现有 Vue 组件的样式，统一使用轻量阴影、实心按钮、紧凑布局，移除所有动画效果，保持简洁直接的交互体验

**技术栈：** Vue 3 + TypeScript + CSS Variables

---

## 文件结构

### 需要修改的文件

1. **`src/assets/styles/modules/channel.css`**
   - 职责：全局频道样式，定义卡片、按钮、间距等基础样式
   - 修改：优化阴影、间距、颜色变量

2. **`src/components/channel/ChannelCard.vue`**
   - 职责：频道卡片组件
   - 修改：调整卡片样式、按钮样式、间距

3. **`src/components/channel/ChannelListItem.vue`**
   - 职责：频道列表项组件
   - 修改：优化列表项样式、间距

4. **`src/components/channel/ChannelHeader.vue`**
   - 职责：频道头部组件
   - 修改：调整头部布局、按钮样式

5. **`src/components/channel/MessageCard.vue`**
   - 职责：消息卡片组件
   - 修改：优化消息卡片样式、操作按钮

6. **`src/components/channel/ChannelSidebar.vue`**
   - 职责：频道侧边栏组件
   - 修改：调整侧边栏布局、间距

7. **`src/components/channel/ChannelDetailNew.vue`**
   - 职责：频道详情组件
   - 修改：优化消息输入区域样式

---

## 任务 1：优化全局频道样式文件

**文件：**
- 修改：`src/assets/styles/modules/channel.css`

- [ ] **步骤 1：更新卡片基础样式**

在 `channel.css` 中找到 `.channel-card` 样式块（约第 263-277 行），替换为：

```css
.channel-card {
  display: flex;
  flex-direction: column;
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  padding: var(--spacing-3);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  border: 1px solid var(--border-color);
}

.channel-card.selected {
  border: 2px solid var(--primary-color);
  box-shadow: 0 2px 12px rgba(51, 133, 255, 0.15);
}
```

- [ ] **步骤 2：更新列表项样式**

在 `channel.css` 中找到 `.channel-list-item` 样式块（约第 183-200 行），替换为：

```css
.channel-list-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-3);
  border-radius: var(--radius-md);
  cursor: pointer;
  border: 1px solid transparent;
}

.channel-list-item:hover {
  background: var(--color-gray-100);
}

.channel-list-item.active {
  background: var(--color-gray-100);
  border-color: var(--primary-color);
}
```

- [ ] **步骤 3：更新消息卡片样式**

在 `channel.css` 中找到 `.message-card` 样式块（约第 354-366 行），替换为：

```css
.message-card {
  display: flex;
  gap: var(--spacing-3);
  padding: var(--spacing-4);
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid var(--border-color);
}
```

- [ ] **步骤 4：移除所有过渡动画**

在 `channel.css` 中搜索所有 `transition:` 属性，删除或注释掉：

```css
/* 移除所有 transition 属性 */
/* transition: all var(--transition-fast); */
```

- [ ] **步骤 5：Commit 全局样式修改**

```bash
git add src/assets/styles/modules/channel.css
git commit -m "style: 优化频道全局样式

- 使用轻量阴影效果
- 统一卡片和列表项样式
- 移除所有过渡动画
- 优化选中状态样式"
```

---

## 任务 2：优化频道卡片组件

**文件：**
- 修改：`src/components/channel/ChannelCard.vue`

- [ ] **步骤 1：更新卡片容器样式**

在 `ChannelCard.vue` 的 `<style scoped>` 部分，找到 `.channel-card` 样式（约第 95-114 行），替换为：

```css
.channel-card {
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  padding: var(--spacing-3);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  border: 1px solid var(--border-color);
}

.channel-card:hover {
  border-color: var(--primary-color);
}

.channel-card.active {
  border: 2px solid var(--primary-color);
  box-shadow: 0 2px 12px rgba(51, 133, 255, 0.15);
}
```

- [ ] **步骤 2：更新订阅按钮样式**

找到 `.card-subscribe-btn` 样式（约第 135-165 行），替换为：

```css
.card-subscribe-btn {
  padding: var(--spacing-1) var(--spacing-3);
  border: 1px solid var(--primary-color);
  background: white;
  color: var(--primary-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 12px;
  font-weight: var(--font-weight-medium);
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
}

.card-subscribe-btn:hover {
  background: var(--primary-color);
  color: white;
}

.card-subscribe-btn.subscribed {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}
```

- [ ] **步骤 3：更新卡片内容样式**

找到 `.card-body` 和 `.card-title` 样式，调整为紧凑布局：

```css
.card-body {
  margin-bottom: var(--spacing-2);
}

.card-title {
  margin: 0 0 var(--spacing-1) 0;
  font-size: 14px;
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-description {
  margin: 0;
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.card-footer {
  font-size: 11px;
  color: var(--text-secondary);
  border-top: 1px solid var(--border-color);
  padding-top: var(--spacing-2);
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
}
```

- [ ] **步骤 4：移除过渡动画**

删除所有 `transition` 属性：

```css
/* 删除这行 */
/* transition: all var(--transition-fast); */
```

- [ ] **步骤 5：Commit 频道卡片修改**

```bash
git add src/components/channel/ChannelCard.vue
git commit -m "style: 优化频道卡片组件样式

- 使用轻量阴影和实心按钮
- 调整间距为紧凑布局
- 移除过渡动画
- 优化选中状态视觉效果"
```

---

## 任务 3：优化频道列表项组件

**文件：**
- 修改：`src/components/channel/ChannelListItem.vue`

- [ ] **步骤 1：更新列表项容器样式**

在 `ChannelListItem.vue` 的 `<style scoped>` 部分，找到 `.channel-list-item` 样式（约第 89-111 行），替换为：

```css
.channel-list-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-3);
  border-radius: var(--radius-md);
  cursor: pointer;
  border: 1px solid transparent;
}

.channel-list-item:hover {
  background: var(--color-gray-100);
}

.channel-list-item.active {
  background: var(--color-gray-100);
  border-color: var(--primary-color);
}
```

- [ ] **步骤 2：更新订阅按钮样式**

找到 `.subscribe-btn` 样式（约第 147-176 行），替换为：

```css
.subscribe-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid var(--primary-color);
  background: white;
  color: var(--primary-color);
  border-radius: var(--radius-md);
  cursor: pointer;
}

.subscribe-btn:hover {
  background: var(--primary-color);
  color: white;
}

.subscribe-btn.subscribed {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}
```

- [ ] **步骤 3：移除过渡动画**

删除所有 `transition` 属性。

- [ ] **步骤 4：Commit 列表项修改**

```bash
git add src/components/channel/ChannelListItem.vue
git commit -m "style: 优化频道列表项组件样式

- 简化样式，移除动画
- 优化选中状态
- 统一按钮样式"
```

---

## 任务 4：优化频道头部组件

**文件：**
- 修改：`src/components/channel/ChannelHeader.vue`

- [ ] **步骤 1：更新头部容器样式**

在 `ChannelHeader.vue` 的 `<style scoped>` 部分，找到 `.channel-header` 样式（约第 86-93 行），替换为：

```css
.channel-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
}
```

- [ ] **步骤 2：更新订阅按钮样式**

找到 `.subscribe-btn` 样式（约第 160-195 行），替换为：

```css
.subscribe-btn {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2) var(--spacing-4);
  border: 1px solid var(--primary-color);
  border-radius: var(--radius-md);
  background: white;
  color: var(--primary-color);
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}

.subscribe-btn:hover {
  background: var(--primary-color);
  color: white;
}

.subscribe-btn.subscribed {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}
```

- [ ] **步骤 3：移除过渡动画**

删除所有 `transition` 属性。

- [ ] **步骤 4：Commit 头部组件修改**

```bash
git add src/components/channel/ChannelHeader.vue
git commit -m "style: 优化频道头部组件样式

- 调整布局间距
- 统一按钮样式
- 移除过渡动画"
```

---

## 任务 5：优化消息卡片组件

**文件：**
- 修改：`src/components/channel/MessageCard.vue`

- [ ] **步骤 1：更新消息卡片容器样式**

在 `MessageCard.vue` 的 `<style scoped>` 部分，找到 `.message-card` 样式（约第 135-147 行），替换为：

```css
.message-card {
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  padding: var(--spacing-4);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid var(--border-color);
}
```

- [ ] **步骤 2：更新操作按钮样式**

找到 `.action-btn` 样式（约第 216-251 行），替换为：

```css
.action-btn {
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
  padding: var(--spacing-1) var(--spacing-3);
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: var(--font-size-xs);
  border-radius: var(--radius-sm);
}

.action-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.action-btn.active {
  color: var(--danger-color);
}
```

- [ ] **步骤 3：移除过渡动画**

删除所有 `transition` 属性。

- [ ] **步骤 4：Commit 消息卡片修改**

```bash
git add src/components/channel/MessageCard.vue
git commit -m "style: 优化消息卡片组件样式

- 使用轻量阴影
- 简化操作按钮样式
- 移除过渡动画"
```

---

## 任务 6：优化频道侧边栏组件

**文件：**
- 修改：`src/components/channel/ChannelSidebar.vue`

- [ ] **步骤 1：更新侧边栏头部样式**

在 `ChannelSidebar.vue` 的 `<style scoped>` 部分，找到 `.channel-sidebar-header` 样式（约第 253-260 行），替换为：

```css
.channel-sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
  min-height: 56px;
}
```

- [ ] **步骤 2：更新标签按钮样式**

找到 `.tab-btn` 样式（约第 308-335 行），替换为：

```css
.tab-btn {
  flex: 1;
  padding: var(--spacing-2) var(--spacing-3);
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: var(--font-weight-medium);
  color: var(--text-secondary);
  cursor: pointer;
}

.tab-btn:hover {
  background: var(--color-gray-100);
  color: var(--text-color);
}

.tab-btn.active {
  background: var(--primary-color);
  color: white;
}
```

- [ ] **步骤 3：更新创建按钮样式**

找到 `.create-btn` 样式（约第 275-299 行），替换为：

```css
.create-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background: var(--primary-color);
  color: white;
  border-radius: var(--radius-md);
  cursor: pointer;
}

.create-btn:hover {
  background: var(--primary-dark);
}
```

- [ ] **步骤 4：移除过渡动画**

删除所有 `transition` 属性。

- [ ] **步骤 5：Commit 侧边栏修改**

```bash
git add src/components/channel/ChannelSidebar.vue
git commit -m "style: 优化频道侧边栏组件样式

- 调整布局间距
- 统一按钮样式
- 移除过渡动画"
```

---

## 任务 7：优化频道详情组件

**文件：**
- 修改：`src/components/channel/ChannelDetailNew.vue`

- [ ] **步骤 1：更新消息输入区域样式**

在 `ChannelDetailNew.vue` 的 `<style scoped>` 部分，找到 `.message-input-area` 样式（约第 178-182 行），替换为：

```css
.message-input-area {
  padding: var(--spacing-3);
  border-top: 1px solid var(--border-color);
  background: var(--card-bg);
}
```

- [ ] **步骤 2：更新发送按钮样式**

找到 `.send-btn` 样式（约第 220-249 行），替换为：

```css
.send-btn {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2) var(--spacing-4);
  border: none;
  border-radius: var(--radius-md);
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
}

.send-btn:hover:not(:disabled) {
  background: var(--primary-dark);
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
```

- [ ] **步骤 3：移除过渡动画**

删除所有 `transition` 属性。

- [ ] **步骤 4：Commit 详情组件修改**

```bash
git add src/components/channel/ChannelDetailNew.vue
git commit -m "style: 优化频道详情组件样式

- 调整输入区域间距
- 统一按钮样式
- 移除过渡动画"
```

---

## 任务 8：验证和测试

**文件：**
- 无文件修改，仅测试

- [ ] **步骤 1：启动开发服务器**

运行：`cd qim-client && npm run dev`

预期：开发服务器正常启动，无编译错误

- [ ] **步骤 2：检查频道页面**

在浏览器中打开频道页面，检查以下内容：

1. 卡片样式是否使用轻量阴影
2. 按钮样式是否统一（实心主按钮，描边次要按钮）
3. 间距是否紧凑
4. 是否没有动画效果
5. 选中状态是否明显

- [ ] **步骤 3：测试交互功能**

测试以下交互：

1. 点击频道卡片，检查选中状态
2. 点击订阅按钮，检查样式变化
3. 切换列表/卡片视图
4. 测试搜索功能

- [ ] **步骤 4：检查响应式布局**

调整浏览器窗口大小，检查移动端适配是否正常。

- [ ] **步骤 5：最终 Commit**

如果一切正常，创建最终 commit：

```bash
git add -A
git commit -m "feat: 完成频道页面样式优化

- 采用现代简约风格
- 轻量阴影 + 实心按钮 + 紧凑布局
- 移除所有动画效果
- 统一颜色和间距规范

Closes #频道页面样式优化"
```

---

## 验收标准

- [ ] 所有卡片使用轻量阴影效果 (0 2px 8px rgba(0,0,0,0.06))
- [ ] 按钮样式统一，主次分明
- [ ] 间距紧凑，信息密度适中
- [ ] 无任何过渡动画效果
- [ ] 选中状态明显区分（主题色边框）
- [ ] 颜色对比度符合可访问性标准
- [ ] 响应式布局正常工作
- [ ] 所有交互功能正常
