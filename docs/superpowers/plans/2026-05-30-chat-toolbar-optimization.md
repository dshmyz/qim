# ChatToolbar 优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 ChatToolbar 从 10 个按钮优化为 8 个，合并通话相关按钮为下拉菜单，消息管理器移到最右侧。

**架构：** 在 ChatToolbar.vue 内部，复用现有 screenshot-dropdown 模式创建 call-dropdown，将语音/视频/屏幕共享三个按钮合并为一个下拉按钮；消息管理按钮添加 margin-left: auto 推到工具栏最右侧。外部事件接口不变，无需修改上游组件的事件监听。

**技术栈：** Vue 3 + TypeScript

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `qim-client/src/components/chat/ChatToolbar.vue` | 修改 | 主要修改文件：合并通话按钮、消息管理器移到最右侧 |

ChatToolbar.vue 是唯一需要修改的文件。因为事件接口（emit）保持不变，上游组件 MessageInput.vue、ChatInputArea.vue、ChatWindow.vue 无需任何修改。

---

### 任务 1：合并通话按钮为下拉菜单

**文件：**
- 修改：`qim-client/src/components/chat/ChatToolbar.vue`

- [ ] **步骤 1：添加 call-dropdown 状态变量**

在 `<script setup>` 中，在 `showScreenshotMenu` 下方添加：

```typescript
const showCallMenu = ref(false)

const toggleCallMenu = () => {
  showCallMenu.value = !showCallMenu.value
}

const selectCallType = (type: 'voice' | 'video' | 'screen') => {
  showCallMenu.value = false
  if (type === 'voice') {
    emit('start-voice-call')
  } else if (type === 'video') {
    emit('start-video-call')
  } else {
    emit('start-screen-share')
  }
}
```

- [ ] **步骤 2：替换模板中的三个通话按钮**

将模板中这三个独立按钮：

```vue
<ChatToolbarButton
  icon="fas fa-phone-alt"
  title="语音通话"
  @click="$emit('start-voice-call')"
/>
<ChatToolbarButton
  icon="fas fa-video"
  title="视频通话"
  @click="$emit('start-video-call')"
/>
<ChatToolbarButton
  icon="fas fa-desktop"
  title="屏幕共享"
  @click="$emit('start-screen-share')"
/>
```

替换为：

```vue
<div class="call-dropdown">
  <ChatToolbarButton
    class="call-btn"
    icon="fas fa-phone-alt"
    title="通话"
    @click="$emit('start-voice-call')"
  />
  <button class="call-dropdown-trigger" @click="toggleCallMenu" title="更多通话选项">
    <i class="fas fa-caret-down"></i>
  </button>
  <div v-show="showCallMenu" class="call-menu" @click.stop>
    <div class="call-menu-item" @click="selectCallType('voice')">
      <i class="fas fa-phone-alt"></i>
      <span>语音通话</span>
    </div>
    <div class="call-menu-item" @click="selectCallType('video')">
      <i class="fas fa-video"></i>
      <span>视频通话</span>
    </div>
    <div class="call-menu-item" @click="selectCallType('screen')">
      <i class="fas fa-desktop"></i>
      <span>屏幕共享</span>
    </div>
  </div>
</div>
```

- [ ] **步骤 3：更新 onDocumentClick 以同时关闭通话下拉菜单**

将现有的 onDocumentClick 函数：

```typescript
const onDocumentClick = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (!target.closest('.screenshot-dropdown')) {
    showScreenshotMenu.value = false
  }
}
```

替换为：

```typescript
const onDocumentClick = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (!target.closest('.screenshot-dropdown')) {
    showScreenshotMenu.value = false
  }
  if (!target.closest('.call-dropdown')) {
    showCallMenu.value = false
  }
}
```

- [ ] **步骤 4：添加 call-dropdown 样式**

在 `<style scoped>` 中，在 `.screenshot-menu-item i` 样式块之后添加：

```css
.call-dropdown {
  position: relative;
  display: inline-flex;
  align-items: center;
}

.call-btn {
  border-radius: 4px 0 0 4px !important;
}

.call-dropdown-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 32px;
  background: transparent;
  border: none;
  border-radius: 0 4px 4px 0;
  cursor: pointer;
  color: var(--text-secondary, #666);
  font-size: 10px;
  padding: 0;
  margin-left: -6px;
  transition: all 0.2s ease;
}

.call-dropdown-trigger:hover {
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
  color: var(--text-primary, #333);
}

.call-menu {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: 1000;
  min-width: 140px;
  background: var(--bg-primary, #fff);
  border: 1px solid var(--border-color, #E5E5E5);
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  padding: 4px 0;
  margin-top: 4px;
}

.call-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-primary, #333);
  white-space: nowrap;
  transition: background 0.15s ease;
}

.call-menu-item:hover {
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
}

.call-menu-item i {
  width: 16px;
  font-size: 13px;
  color: var(--text-secondary, #666);
}
```

- [ ] **步骤 5：验证通话下拉功能正常**

在浏览器中测试：
1. 点击 📞 通话主按钮 → 应触发语音通话（默认行为）
2. 点击 ▾ 下拉箭头 → 应展开菜单，显示语音通话/视频通话/屏幕共享
3. 点击菜单项 → 应触发对应事件，菜单关闭
4. 点击页面其他区域 → 菜单应自动关闭

- [ ] **步骤 6：Commit**

```bash
git add qim-client/src/components/chat/ChatToolbar.vue
git commit -m "feat: merge voice/video/screen-share into call dropdown button"
```

---

### 任务 2：消息管理器移到工具栏最右侧

**文件：**
- 修改：`qim-client/src/components/chat/ChatToolbar.vue`

- [ ] **步骤 1：将消息管理按钮移到模板末尾并添加样式类**

将消息管理按钮从当前位置（截图按钮之后）移到模板最末尾（AI 按钮之后），并添加 `message-manager-btn` 类：

原位置（截图按钮之后）：
```vue
<ChatToolbarButton
  icon="fas fa-history"
  title="消息管理"
  @click="$emit('open-message-manager')"
/>
```

移到 AI 按钮之后，改为：
```vue
<ChatToolbarButton
  class="message-manager-btn"
  icon="fas fa-history"
  title="消息管理"
  @click="$emit('open-message-manager')"
/>
```

- [ ] **步骤 2：添加消息管理器间距样式**

在 `<style scoped>` 中添加：

```css
.message-manager-btn {
  margin-left: auto;
}
```

- [ ] **步骤 3：验证消息管理器位置**

在浏览器中测试：
1. 消息管理按钮应出现在工具栏最右侧
2. 与左侧按钮之间有自然间距
3. 点击消息管理按钮 → 应正常触发事件
4. 调整窗口宽度 → 间距应自适应

- [ ] **步骤 4：Commit**

```bash
git add qim-client/src/components/chat/ChatToolbar.vue
git commit -m "feat: move message manager button to toolbar rightmost position"
```

---

### 任务 3：整体验证

**文件：**
- 无文件修改，仅验证

- [ ] **步骤 1：验证完整工具栏布局**

确认工具栏按钮顺序为：
```
📞 通话 ▾ | 😊 表情 | 🖼️ 图片 | 📎 文件 | ✂️ 截图 ▾ | 🧩 小程序 | 🤖 AI | ···· | 📋 消息管理
```

- [ ] **步骤 2：验证所有功能正常**

逐个测试每个按钮：
1. 📞 通话 ▾ → 下拉菜单正常，语音/视频/屏幕共享均触发
2. 😊 表情 → 正常切换表情面板
3. 🖼️ 图片 → 正常打开图片选择器
4. 📎 文件 → 正常打开文件选择器
5. ✂️ 截图 ▾ → 下拉菜单正常（Electron 环境）
6. 🧩 小程序 → 正常打开小程序列表
7. 🤖 AI → 正常切换 AI 面板
8. 📋 消息管理 → 正常打开消息管理器，位于最右侧

- [ ] **步骤 3：验证下拉菜单互不干扰**

1. 打开通话下拉 → 点击截图下拉 → 通话下拉应关闭
2. 打开截图下拉 → 点击通话下拉 → 截图下拉应关闭
3. 打开任一下拉 → 点击页面空白 → 下拉应关闭

- [ ] **步骤 4：最终 Commit**

```bash
git commit --allow-empty -m "chore: chat toolbar optimization verified"
```
