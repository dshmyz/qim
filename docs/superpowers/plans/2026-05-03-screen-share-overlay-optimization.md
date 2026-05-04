# 屏幕共享弹窗优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 优化屏幕共享弹窗的拖拽体验和最小化显示效果

**架构：** 修复拖拽逻辑的偏移量计算问题，重新设计最小化样式以显示视频预览和完整信息，使用 requestAnimationFrame 优化拖拽性能

**技术栈：** Vue 3 Composition API, TypeScript, CSS

---

## 文件结构

**修改文件：**
- `qim-client/src/components/shared/ScreenShare.vue` - 屏幕共享弹窗组件（主要修改）
  - 修复拖拽逻辑（script 部分）
  - 新增最小化内容模板（template 部分）
  - 新增最小化样式（style 部分）

**改动范围：**
- Script 部分：约 70 行修改/新增
- Template 部分：约 30 行新增
- Style 部分：约 80 行新增
- 总计：约 180 行代码改动

---

## 任务 1：修复主窗口拖拽逻辑

**文件：**
- 修改：`qim-client/src/components/shared/ScreenShare.vue:201-203, 580-625`

- [ ] **步骤 1：移除旧的拖拽状态变量**

在 script setup 部分，找到并删除以下代码（约第 201-203 行）：

```typescript
const isDragging = ref(false)
const dragOffset = ref({ x: 0, y: 0 })
```

- [ ] **步骤 2：添加新的拖拽状态管理**

在删除的位置添加新的拖拽状态：

```typescript
const dragState = ref({
  isDragging: false,
  startX: 0,
  startY: 0,
  elementX: 0,
  elementY: 0
})

let rafId: number | null = null
```

- [ ] **步骤 3：重写 startDrag 函数**

找到 `startDrag` 函数（约第 582-587 行），替换为：

```typescript
const startDrag = (e: MouseEvent) => {
  e.preventDefault()
  const element = screenShareOverlayRef.value
  if (!element) return
  
  const rect = element.getBoundingClientRect()
  
  dragState.value = {
    isDragging: true,
    startX: e.clientX,
    startY: e.clientY,
    elementX: rect.left,
    elementY: rect.top
  }
  
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}
```

- [ ] **步骤 4：重写 onDrag 函数**

找到 `onWindowDrag` 函数（约第 599-613 行），替换为：

```typescript
const onDrag = (e: MouseEvent) => {
  if (!dragState.value.isDragging) return
  
  const deltaX = e.clientX - dragState.value.startX
  const deltaY = e.clientY - dragState.value.startY
  
  let newX = dragState.value.elementX + deltaX
  let newY = dragState.value.elementY + deltaY
  
  const element = screenShareOverlayRef.value
  if (element) {
    const rect = element.getBoundingClientRect()
    newX = Math.max(0, Math.min(newX, window.innerWidth - rect.width))
    newY = Math.max(0, Math.min(newY, window.innerHeight - rect.height))
  }
  
  if (rafId) {
    cancelAnimationFrame(rafId)
  }
  
  rafId = requestAnimationFrame(() => {
    if (element) {
      element.style.left = `${newX}px`
      element.style.top = `${newY}px`
      element.style.transform = 'none'
    }
  })
}
```

- [ ] **步骤 5：重写 stopDrag 函数**

找到 `stopWindowDrag` 函数（约第 621-625 行），替换为：

```typescript
const stopDrag = () => {
  if (rafId) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
  dragState.value.isDragging = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
}
```

- [ ] **步骤 6：删除旧的 onDrag 和 stopDrag 函数**

删除以下旧函数（如果存在）：
- `onDrag` 函数（约第 589-597 行）
- `stopDrag` 函数（约第 615-619 行）

- [ ] **步骤 7：添加拖拽状态监听**

在 script setup 部分添加 watch：

```typescript
watch(() => dragState.value.isDragging, (isDragging) => {
  if (screenShareOverlayRef.value) {
    if (isDragging) {
      screenShareOverlayRef.value.classList.add('dragging')
    } else {
      screenShareOverlayRef.value.classList.remove('dragging')
    }
  }
})
```

- [ ] **步骤 8：测试主窗口拖拽**

运行开发服务器并测试：
```bash
cd qim-client && npm run dev
```

测试要点：
- 点击标题栏开始拖拽，弹窗应立即跟随鼠标
- 拖拽过程中弹窗位置精确，无延迟
- 拖拽到屏幕边缘时，弹窗不会超出屏幕
- 松开鼠标后，弹窗停留在当前位置

- [ ] **步骤 9：Commit 拖拽逻辑修复**

```bash
git add qim-client/src/components/shared/ScreenShare.vue
git commit -m "fix: 修复屏幕共享弹窗主窗口拖拽逻辑

- 修复偏移量计算错误，改为记录初始位置
- 添加边界检测，防止弹窗拖出屏幕
- 使用 requestAnimationFrame 优化性能
- 添加拖拽状态视觉反馈"
```

---

## 任务 2：修复浮窗拖拽逻辑

**文件：**
- 修改：`qim-client/src/components/shared/ScreenShare.vue:627-650`

- [ ] **步骤 1：重写 startFloatingDrag 函数**

找到 `startFloatingDrag` 函数（约第 627-634 行），替换为：

```typescript
const startFloatingDrag = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (target.closest('.floating-actions')) return
  
  e.preventDefault()
  
  dragState.value = {
    isDragging: true,
    startX: e.clientX,
    startY: e.clientY,
    elementX: floatingPosition.value.x,
    elementY: floatingPosition.value.y
  }
  
  document.addEventListener('mousemove', onFloatingDrag)
  document.addEventListener('mouseup', stopFloatingDrag)
}
```

- [ ] **步骤 2：重写 onFloatingDrag 函数**

找到 `onFloatingDrag` 函数（约第 636-644 行），替换为：

```typescript
const onFloatingDrag = (e: MouseEvent) => {
  if (!dragState.value.isDragging) return
  
  const deltaX = e.clientX - dragState.value.startX
  const deltaY = e.clientY - dragState.value.startY
  
  let newX = dragState.value.elementX - deltaX
  let newY = dragState.value.elementY - deltaY
  
  newX = Math.max(0, Math.min(newX, window.innerWidth - 280))
  newY = Math.max(0, Math.min(newY, window.innerHeight - 200))
  
  floatingPosition.value = { x: newX, y: newY }
}
```

- [ ] **步骤 3：重写 stopFloatingDrag 函数**

找到 `stopFloatingDrag` 函数（约第 646-650 行），替换为：

```typescript
const stopFloatingDrag = () => {
  dragState.value.isDragging = false
  document.removeEventListener('mousemove', onFloatingDrag)
  document.removeEventListener('mouseup', stopFloatingDrag)
}
```

- [ ] **步骤 4：测试浮窗拖拽**

测试要点：
- 浮窗可以正常拖拽
- 拖拽流畅，无延迟
- 边界检测正常

- [ ] **步骤 5：Commit 浮窗拖拽修复**

```bash
git add qim-client/src/components/shared/ScreenShare.vue
git commit -m "fix: 修复屏幕共享浮窗拖拽逻辑

- 统一使用 dragState 管理拖拽状态
- 修复浮窗位置计算（使用 right/bottom 定位）
- 添加边界检测"
```

---

## 任务 3：添加拖拽视觉反馈样式

**文件：**
- 修改：`qim-client/src/components/shared/ScreenShare.vue:817-826`

- [ ] **步骤 1：修改标题栏样式**

找到 `.screen-share-header` 样式（约第 817-826 行），确保包含：

```css
.screen-share-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, rgba(255, 255, 255, 0.05) 100%);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  cursor: grab;
  user-select: none;
}

.screen-share-header:active {
  cursor: grabbing;
}
```

- [ ] **步骤 2：添加拖拽状态样式**

在 style 部分添加新样式：

```css
.screen-share-overlay.dragging {
  cursor: grabbing;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
  transition: box-shadow 0.2s;
}
```

- [ ] **步骤 3：测试视觉反馈**

测试要点：
- 鼠标悬停在标题栏时显示 grab 光标
- 拖拽时显示 grabbing 光标
- 拖拽时弹窗阴影加深

- [ ] **步骤 4：Commit 视觉反馈样式**

```bash
git add qim-client/src/components/shared/ScreenShare.vue
git commit -m "style: 添加屏幕共享弹窗拖拽视觉反馈

- 标题栏显示 grab 光标
- 拖拽时显示 grabbing 光标和加深阴影"
```

---

## 任务 4：添加最小化视频预览

**文件：**
- 修改：`qim-client/src/components/shared/ScreenShare.vue:192-194`

- [ ] **步骤 1：添加最小化视频引用**

在 script setup 部分找到 `remoteVideoRef` 和 `floatingVideoRef` 定义（约第 192-194 行），在其后添加：

```typescript
const remoteVideoRef = ref<HTMLVideoElement | null>(null)
const floatingVideoRef = ref<HTMLVideoElement | null>(null)
const minimizedVideoRef = ref<HTMLVideoElement | null>(null)
```

- [ ] **步骤 2：修改 toggleMinimize 函数**

找到 `toggleMinimize` 函数（约第 531-537 行），替换为：

```typescript
const toggleMinimize = () => {
  isMinimized.value = !isMinimized.value
  
  if (isMinimized.value) {
    nextTick(() => {
      if (minimizedVideoRef.value && remoteVideoRef.value?.srcObject) {
        minimizedVideoRef.value.srcObject = remoteVideoRef.value.srcObject
        minimizedVideoRef.value.play().catch(err => {
          console.error('最小化视频播放失败:', err)
        })
      }
    })
  }
}
```

- [ ] **步骤 3：添加 expandFromMinimized 函数**

在 `toggleMinimize` 函数后添加：

```typescript
const expandFromMinimized = () => {
  isMinimized.value = false
}
```

- [ ] **步骤 4：添加双击展开处理函数**

添加双击展开函数：

```typescript
const handleHeaderDblClick = () => {
  if (isMinimized.value) {
    expandFromMinimized()
  }
}
```

- [ ] **步骤 5：测试视频流同步**

测试要点：
- 最小化时视频流同步到预览窗口
- 展开时视频流正常显示
- 视频预览清晰，无卡顿

- [ ] **步骤 6：Commit 视频流同步**

```bash
git add qim-client/src/components/shared/ScreenShare.vue
git commit -m "feat: 添加屏幕共享最小化视频预览

- 添加最小化视频引用
- 最小化时同步视频流到预览窗口
- 添加展开和双击展开功能"
```

---

## 任务 5：添加最小化内容模板

**文件：**
- 修改：`qim-client/src/components/shared/ScreenShare.vue:4-23`

- [ ] **步骤 1：修改标题栏添加双击事件**

找到标题栏 div（约第 4 行），修改为：

```vue
<div class="screen-share-header" @mousedown="startDrag" @dblclick="handleHeaderDblClick">
```

- [ ] **步骤 2：添加最小化内容模板**

在标题栏后（约第 23 行后），添加最小化内容：

```vue
<div v-if="isMinimized" class="minimized-content">
  <div class="minimized-preview">
    <video ref="minimizedVideoRef" autoplay playsinline muted></video>
  </div>
  <div class="minimized-info">
    <div class="info-top">
      <div class="minimized-status">
        <span class="pulse-dot" :class="{ active: isSharing }"></span>
        <span>{{ isInitiator ? '共享中' : '观看中' }}</span>
      </div>
      <div class="minimized-duration">{{ formattedDuration }}</div>
      <div class="minimized-name">{{ screenShareName || senderName || '屏幕共享' }}</div>
    </div>
    <div class="minimized-actions" @click.stop>
      <button class="action-btn expand-btn" @click="expandFromMinimized">
        <i class="fas fa-expand"></i>
        <span>展开</span>
      </button>
      <button class="action-btn close-btn" @click="stopShare">
        <i class="fas fa-stop"></i>
        <span>停止</span>
      </button>
    </div>
  </div>
</div>
```

- [ ] **步骤 3：修改最小化样式**

找到 `.screen-share-overlay.minimized` 样式（约第 806-815 行），替换为：

```css
.screen-share-overlay.minimized {
  width: 320px;
  height: auto;
  min-height: 140px;
}

.screen-share-overlay.minimized .screen-share-body,
.screen-share-overlay.minimized .screen-share-controls,
.screen-share-overlay.minimized .initiator-controls {
  display: none;
}
```

- [ ] **步骤 4：测试最小化模板**

测试要点：
- 点击最小化按钮，弹窗切换到最小化状态
- 显示视频预览缩略图
- 显示共享状态、时长、窗口名称
- 点击展开按钮恢复到正常大小

- [ ] **步骤 5：Commit 最小化模板**

```bash
git add qim-client/src/components/shared/ScreenShare.vue
git commit -m "feat: 添加屏幕共享最小化内容模板

- 显示视频预览缩略图
- 显示共享状态、时长、窗口名称
- 添加展开和停止按钮
- 支持双击标题栏展开"
```

---

## 任务 6：添加最小化样式

**文件：**
- 修改：`qim-client/src/components/shared/ScreenShare.vue:1332+`

- [ ] **步骤 1：添加最小化内容样式**

在 style 部分末尾添加：

```css
.minimized-content {
  display: flex;
  padding: 12px;
  gap: 12px;
  cursor: pointer;
  transition: background 0.2s;
}

.minimized-content:hover {
  background: rgba(255, 255, 255, 0.05);
}

.minimized-preview {
  width: 120px;
  height: 68px;
  border-radius: 8px;
  overflow: hidden;
  background: rgba(0, 0, 0, 0.4);
  flex-shrink: 0;
}

.minimized-preview video {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.minimized-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  min-width: 0;
}

.info-top {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.minimized-status {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #fff;
  font-size: 13px;
  font-weight: 500;
}

.minimized-duration {
  color: rgba(255, 255, 255, 0.7);
  font-size: 12px;
  font-family: 'SF Mono', Monaco, monospace;
}

.minimized-name {
  color: rgba(255, 255, 255, 0.6);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.minimized-actions {
  display: flex;
  gap: 8px;
}

.minimized-actions .action-btn {
  flex: 1;
  padding: 6px 10px;
  font-size: 11px;
}

.expand-btn {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
}

.expand-btn:hover {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
}
```

- [ ] **步骤 2：测试最小化样式**

测试要点：
- 最小化样式美观
- 视频预览清晰
- 信息显示完整
- 按钮交互正常

- [ ] **步骤 3：Commit 最小化样式**

```bash
git add qim-client/src/components/shared/ScreenShare.vue
git commit -m "style: 添加屏幕共享最小化样式

- 视频预览区域样式
- 信息显示区域样式
- 操作按钮样式
- 悬停效果"
```

---

## 任务 7：综合测试和优化

**文件：**
- 测试：`qim-client/src/components/shared/ScreenShare.vue`

- [ ] **步骤 1：运行开发服务器**

```bash
cd qim-client && npm run dev
```

- [ ] **步骤 2：测试拖拽功能**

测试清单：
- [ ] 点击标题栏开始拖拽，弹窗应立即跟随鼠标
- [ ] 拖拽过程中弹窗位置精确，无延迟
- [ ] 拖拽到屏幕边缘时，弹窗不会超出屏幕
- [ ] 松开鼠标后，弹窗停留在当前位置
- [ ] 快速拖拽时，弹窗依然流畅跟随
- [ ] 拖拽时显示 grabbing 光标
- [ ] 浮窗拖拽同样流畅

- [ ] **步骤 3：测试最小化功能**

测试清单：
- [ ] 点击最小化按钮，弹窗切换到最小化状态
- [ ] 最小化后显示视频预览缩略图
- [ ] 显示共享状态、时长、窗口名称
- [ ] 点击展开按钮恢复到正常大小
- [ ] 双击标题栏快速展开
- [ ] 最小化状态下可以拖拽移动位置

- [ ] **步骤 4：测试视频流**

测试清单：
- [ ] 最小化时视频流同步到预览窗口
- [ ] 展开时视频流正常显示
- [ ] 视频预览清晰，无卡顿
- [ ] 切换最小化/展开状态时视频不中断

- [ ] **步骤 5：测试边界情况**

测试清单：
- [ ] 窗口大小改变时，弹窗位置自动调整
- [ ] 拖拽到屏幕角落时的边界处理
- [ ] 快速连续点击最小化/展开按钮
- [ ] 拖拽过程中切换最小化状态

- [ ] **步骤 6：性能测试**

测试清单：
- [ ] 拖拽时 CPU 占用正常
- [ ] 视频预览不影响主视频性能
- [ ] 内存使用稳定，无泄漏

- [ ] **步骤 7：修复发现的问题**

如果测试中发现问题，记录并修复：

```
问题：
修复：
```

- [ ] **步骤 8：最终 Commit**

```bash
git add qim-client/src/components/shared/ScreenShare.vue
git commit -m "test: 完成屏幕共享弹窗优化综合测试

- 拖拽功能测试通过
- 最小化功能测试通过
- 视频流同步测试通过
- 边界情况测试通过
- 性能测试通过"
```

---

## 实现要点

### 关键技术点

1. **拖拽偏移量计算**
   - 记录鼠标按下时的初始位置和元素初始位置
   - 计算鼠标移动距离（deltaX, deltaY）
   - 新位置 = 元素初始位置 + 移动距离

2. **边界检测**
   - 使用 `Math.max` 和 `Math.min` 限制位置范围
   - 考虑元素宽度和高度
   - 防止拖出屏幕

3. **性能优化**
   - 使用 `requestAnimationFrame` 优化渲染
   - 避免频繁的 DOM 操作
   - 及时清理动画帧

4. **视频流同步**
   - 使用 `nextTick` 确保 DOM 更新完成
   - 同步 `srcObject` 属性
   - 处理播放错误

### 注意事项

1. **浮窗定位差异**
   - 浮窗使用 `right` 和 `bottom` 定位
   - 拖拽计算时使用减法而非加法

2. **事件清理**
   - 确保在组件卸载时清理事件监听器
   - 取消未完成的动画帧

3. **响应式更新**
   - 使用 `ref` 管理拖拽状态
   - 使用 `watch` 监听状态变化

### 可能的问题

1. **视频流同步失败**
   - 检查 `remoteVideoRef.value?.srcObject` 是否存在
   - 添加错误处理

2. **拖拽卡顿**
   - 检查是否有其他性能问题
   - 确保使用 `requestAnimationFrame`

3. **边界检测不准确**
   - 检查元素尺寸获取是否正确
   - 考虑窗口大小变化

---

## 完成标准

- [ ] 所有测试通过
- [ ] 代码无 TypeScript 错误
- [ ] 代码无 ESLint 警告
- [ ] 所有 commit 信息清晰
- [ ] 功能符合设计文档要求
