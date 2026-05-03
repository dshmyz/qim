# 屏幕共享弹窗优化设计

**日期**: 2026-05-03  
**状态**: 设计完成，待实现  
**影响范围**: `qim-client/src/components/shared/ScreenShare.vue`

## 背景

当前屏幕共享弹窗存在以下问题：

### 拖拽问题
- 拖拽时弹窗不跟随鼠标
- 拖拽延迟明显
- 拖拽位置不准确
- 拖拽启动困难

### 最小化问题
- 最小化样式不够美观
- 信息显示不全（缺少共享状态、时长、窗口名称）
- 恢复操作不便

## 目标

1. 实现流畅、精确的拖拽体验，弹窗即时响应鼠标移动
2. 优化最小化样式，显示完整信息和视频预览
3. 提供便捷的恢复操作方式

## 设计方案

### 一、拖拽优化

#### 1.1 问题分析

当前拖拽逻辑的核心问题：

**代码位置**: `ScreenShare.vue:582-625`

```typescript
// 当前实现的问题
const startDrag = (e: MouseEvent) => {
  isDragging.value = true
  dragOffset.value = { x: e.clientX, y: e.clientY }  // 问题：记录的是鼠标绝对位置
  document.addEventListener('mousemove', onWindowDrag)
  document.addEventListener('mouseup', stopWindowDrag)
}

const onWindowDrag = (e: MouseEvent) => {
  if (isDragging.value && screenShareOverlayRef.value) {
    const element = screenShareOverlayRef.value
    const rect = element.getBoundingClientRect()
    
    const newX = rect.left + e.clientX - dragOffset.value.x  // 问题：每次都累加
    const newY = rect.top + e.clientY - dragOffset.value.y
    
    element.style.left = `${newX}px`
    element.style.top = `${newY}px`
    
    dragOffset.value = { x: e.clientX, y: e.clientY }  // 问题：每次都更新
  }
}
```

**问题根源**:
1. 每次移动都更新 `dragOffset`，导致位置累积错误
2. 没有记录鼠标相对于元素的初始偏移
3. 缺少边界检测

#### 1.2 优化方案

**新的拖拽状态管理**:

```typescript
const dragState = ref({
  isDragging: false,
  startX: 0,        // 鼠标按下时的 X 坐标
  startY: 0,        // 鼠标按下时的 Y 坐标
  elementX: 0,      // 元素初始位置 X
  elementY: 0       // 元素初始位置 Y
})

let rafId: number | null = null
```

**改进后的拖拽逻辑**:

```typescript
const startDrag = (e: MouseEvent) => {
  e.preventDefault()
  const element = screenShareOverlayRef.value
  if (!element) return
  
  const rect = element.getBoundingClientRect()
  
  // 记录初始状态
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

const onDrag = (e: MouseEvent) => {
  if (!dragState.value.isDragging) return
  
  // 计算移动距离
  const deltaX = e.clientX - dragState.value.startX
  const deltaY = e.clientY - dragState.value.startY
  
  // 计算新位置
  let newX = dragState.value.elementX + deltaX
  let newY = dragState.value.elementY + deltaY
  
  // 边界检测
  const element = screenShareOverlayRef.value
  if (element) {
    const rect = element.getBoundingClientRect()
    newX = Math.max(0, Math.min(newX, window.innerWidth - rect.width))
    newY = Math.max(0, Math.min(newY, window.innerHeight - rect.height))
  }
  
  // 使用 requestAnimationFrame 优化性能
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

**浮窗拖拽的类似修复**:

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

const onFloatingDrag = (e: MouseEvent) => {
  if (!dragState.value.isDragging) return
  
  const deltaX = e.clientX - dragState.value.startX
  const deltaY = e.clientY - dragState.value.startY
  
  let newX = dragState.value.elementX - deltaX  // 注意：浮窗使用 right 定位，所以是减法
  let newY = dragState.value.elementY - deltaY  // 注意：浮窗使用 bottom 定位，所以是减法
  
  // 边界检测
  newX = Math.max(0, Math.min(newX, window.innerWidth - 280))
  newY = Math.max(0, Math.min(newY, window.innerHeight - 200))
  
  floatingPosition.value = { x: newX, y: newY }
}

const stopFloatingDrag = () => {
  dragState.value.isDragging = false
  document.removeEventListener('mousemove', onFloatingDrag)
  document.removeEventListener('mouseup', stopFloatingDrag)
}
```

#### 1.3 视觉反馈

**添加拖拽状态样式**:

```css
.screen-share-overlay.dragging {
  cursor: grabbing;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.5);
  transition: box-shadow 0.2s;
}

.screen-share-header {
  cursor: grab;
  user-select: none;
}

.screen-share-header:active {
  cursor: grabbing;
}
```

**动态添加 dragging 类**:

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

### 二、最小化优化

#### 2.1 当前问题

**代码位置**: `ScreenShare.vue:806-815`

```css
.screen-share-overlay.minimized {
  width: 320px;
  height: 48px;  /* 问题：高度太小，无法显示内容 */
}

.screen-share-overlay.minimized .screen-share-body,
.screen-share-overlay.minimized .screen-share-controls,
.screen-share-overlay.minimized .initiator-controls {
  display: none;  /* 问题：隐藏了所有内容 */
}
```

#### 2.2 优化方案

**新的最小化布局**:

```
┌─────────────────────────────────────┐
│ 🟢 正在共享屏幕 - Chrome            │  ← 标题栏（可拖拽）
│ ┌───────────┬──────────────────┐   │
│ │           │  共享状态: 共享中  │   │
│ │  视频预览  │  时长: 00:05:32   │   │  ← 内容区
│ │           │  [展开] [停止]    │   │
│ └───────────┴──────────────────┘   │
└─────────────────────────────────────┘
```

**模板结构**:

```vue
<div v-if="isMinimized" class="minimized-content" @click="expandFromMinimized">
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

**样式设计**:

```css
.screen-share-overlay.minimized {
  width: 320px;
  height: auto;
  min-height: 140px;
}

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

#### 2.3 视频流同步

**新增 ref**:

```typescript
const minimizedVideoRef = ref<HTMLVideoElement | null>(null)
```

**同步逻辑**:

```typescript
const toggleMinimize = () => {
  isMinimized.value = !isMinimized.value
  
  if (isMinimized.value) {
    // 同步视频流到最小化预览
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

const expandFromMinimized = () => {
  isMinimized.value = false
  // 视频流已经在 remoteVideoRef 中，无需额外处理
}
```

#### 2.4 交互优化

**双击展开**:

```typescript
const handleHeaderDblClick = () => {
  if (isMinimized.value) {
    expandFromMinimized()
  }
}
```

```vue
<div class="screen-share-header" 
     @mousedown="startDrag"
     @dblclick="handleHeaderDblClick">
```

**点击预览区域展开**:

```vue
<div class="minimized-preview" @click="expandFromMinimized">
```

## 实现要点

### 1. 代码改动范围

**文件**: `qim-client/src/components/shared/ScreenShare.vue`

**改动内容**:
1. 修复拖拽逻辑（约 50 行代码修改）
2. 新增最小化内容模板（约 30 行）
3. 新增最小化样式（约 80 行）
4. 新增视频流同步逻辑（约 20 行）
5. 优化事件处理（约 15 行）

**总计**: 约 195 行代码改动

### 2. 需要移除的代码

```typescript
// 删除旧的拖拽状态
const isDragging = ref(false)
const dragOffset = ref({ x: 0, y: 0 })

// 删除旧的拖拽函数
const onDrag = (e: MouseEvent) => { ... }
const stopDrag = () => { ... }
```

### 3. 需要新增的代码

```typescript
// 新增拖拽状态
const dragState = ref({
  isDragging: false,
  startX: 0,
  startY: 0,
  elementX: 0,
  elementY: 0
})

let rafId: number | null = null

// 新增最小化视频引用
const minimizedVideoRef = ref<HTMLVideoElement | null>(null)

// 新增展开函数
const expandFromMinimized = () => { ... }

// 新增双击处理
const handleHeaderDblClick = () => { ... }
```

### 4. 需要修改的代码

```typescript
// 修改 toggleMinimize 函数，添加视频流同步
const toggleMinimize = () => {
  isMinimized.value = !isMinimized.value
  if (isMinimized.value) {
    nextTick(() => {
      if (minimizedVideoRef.value && remoteVideoRef.value?.srcObject) {
        minimizedVideoRef.value.srcObject = remoteVideoRef.value.srcObject
      }
    })
  }
}
```

## 测试计划

### 1. 拖拽功能测试

**测试场景**:
- ✅ 点击标题栏开始拖拽，弹窗应立即跟随鼠标
- ✅ 拖拽过程中弹窗位置精确，无延迟
- ✅ 拖拽到屏幕边缘时，弹窗不会超出屏幕
- ✅ 松开鼠标后，弹窗停留在当前位置
- ✅ 快速拖拽时，弹窗依然流畅跟随
- ✅ 拖拽时显示 grabbing 光标
- ✅ 浮窗拖拽同样流畅

### 2. 最小化功能测试

**测试场景**:
- ✅ 点击最小化按钮，弹窗切换到最小化状态
- ✅ 最小化后显示视频预览缩略图
- ✅ 显示共享状态、时长、窗口名称
- ✅ 点击展开按钮恢复到正常大小
- ✅ 双击标题栏快速展开
- ✅ 点击预览区域展开
- ✅ 最小化状态下可以拖拽移动位置

### 3. 视频流测试

**测试场景**:
- ✅ 最小化时视频流同步到预览窗口
- ✅ 展开时视频流正常显示
- ✅ 视频预览清晰，无卡顿
- ✅ 切换最小化/展开状态时视频不中断

### 4. 边界情况测试

**测试场景**:
- ✅ 窗口大小改变时，弹窗位置自动调整
- ✅ 拖拽到屏幕角落时的边界处理
- ✅ 快速连续点击最小化/展开按钮
- ✅ 拖拽过程中切换最小化状态

### 5. 性能测试

**测试场景**:
- ✅ 拖拽时 CPU 占用正常
- ✅ 视频预览不影响主视频性能
- ✅ 内存使用稳定，无泄漏

## 风险评估

### 低风险
- 拖拽逻辑修复：改动明确，风险可控
- 样式优化：纯 CSS 改动，不影响功能

### 中风险
- 视频流同步：需要确保流的状态管理正确
- 边界检测：需要考虑各种屏幕尺寸

### 缓解措施
1. 充分测试各种边界情况
2. 保留原有功能作为降级方案
3. 添加错误处理和日志记录

## 预期效果

### 拖拽体验
- 即时响应，流畅跟随鼠标
- 位置精确，无延迟和跳动
- 边界检测，不会拖出屏幕
- 视觉反馈清晰

### 最小化体验
- 显示视频预览缩略图
- 显示共享状态、时长、窗口名称
- 操作按钮清晰易用
- 支持多种展开方式（按钮、双击、点击预览）

## 后续优化

如果本次优化效果良好，可以考虑：
1. 添加拖拽位置记忆功能
2. 支持触摸设备拖拽
3. 添加最小化窗口大小调整功能
4. 支持多个最小化窗口并排显示
