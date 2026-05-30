# ChatToolbar 优化设计

## 背景

当前 ChatToolbar 有 10 个按钮，空间拥挤，功能分区不清晰。部分按钮语义上不属于「内容输入」范畴，低频操作和高频操作混在一起。

## 设计决策

### 1. 语音 + 视频 + 屏幕共享 → 合并为「通话」下拉按钮

**原布局：** 📞 语音通话 | 📹 视频通话 | 🖥️ 屏幕共享（3 个独立按钮）

**新布局：** 📞 通话 ▾（下拉菜单：语音通话 / 视频通话 / 屏幕共享）

**理由：**
- 三者同属「实时通信」，语义统一
- 使用频率均较低，多一次点击可接受
- 屏幕共享是通话场景的延伸（Zoom/Teams/飞书均如此处理）
- 节省 2 个按钮位

### 2. 图片 + 文件 → 保持独立

**理由：**
- 图片是 IM 中极高频操作，合并后多一次点击不可接受
- 图片选择器和文件选择器是两个不同的系统交互
- 所有主流 IM（微信、Telegram、Slack）均保持独立

### 3. 消息管理 → 移到工具栏最右侧

**原位置：** 和其他按钮紧凑排列

**新位置：** 工具栏最右侧，通过 `margin-left: auto` 与左侧按钮拉开间距

**理由：**
- 消息管理不属于「内容输入」，属于「管理操作」
- 间距分隔形成自然的功能分区：左侧=输入工具，右侧=管理操作
- 位置固定在最右侧，形成肌肉记忆
- 类似 VS Code 状态栏的布局思路

### 4. 小程序 → 保持不动

不调整小程序按钮的位置和交互方式。

## 最终布局

```
📞 通话 ▾  |  😊 表情  🖼️ 图片  📎 文件  ✂️ 截图 ▾  🤖 AI  🧩 小程序  ····  📋 消息管理
 ← 内容输入 →                                    ← 管理扩展 →     ← 间距 → ← 管理 →
```

**按钮数量：** 10 → 8（减少 20%）

**功能分区：**
- 左侧：内容输入相关（通话、表情、图片、文件、截图）
- 右侧：扩展和管理（AI、小程序）
- 最右侧（间距分隔）：消息管理

**下拉菜单内容：**
- 📞 通话 ▾：语音通话 | 视频通话 | 屏幕共享
- ✂️ 截图 ▾：区域截图 | 隐藏窗口截图（保持现有逻辑不变）

## 涉及文件

- `qim-client/src/components/chat/ChatToolbar.vue` — 主要修改：合并通话按钮、消息管理器移到最右侧
- `qim-client/src/components/chat/MessageInput.vue` — 调整事件传递（start-voice-call / start-video-call / start-screen-share 合并为通话相关事件）
- `qim-client/src/components/chat/ChatInputArea.vue` — 事件透传调整
- `qim-client/src/components/chat/ChatWindow.vue` — 事件处理调整

## 实现要点

### 通话下拉组件

复用现有截图下拉菜单的模式（screenshot-dropdown），创建 call-dropdown：

```vue
<div class="call-dropdown">
  <ChatToolbarButton icon="fas fa-phone-alt" title="通话" @click="startDefaultCall" />
  <button class="call-dropdown-trigger" @click="toggleCallMenu">
    <i class="fas fa-caret-down"></i>
  </button>
  <div v-show="showCallMenu" class="call-menu" @click.stop>
    <div class="call-menu-item" @click="selectCallType('voice')">
      <i class="fas fa-phone-alt"></i><span>语音通话</span>
    </div>
    <div class="call-menu-item" @click="selectCallType('video')">
      <i class="fas fa-video"></i><span>视频通话</span>
    </div>
    <div class="call-menu-item" @click="selectCallType('screen')">
      <i class="fas fa-desktop"></i><span>屏幕共享</span>
    </div>
  </div>
</div>
```

### 消息管理器间距

在消息管理按钮上添加 `margin-left: auto`，利用 flex 布局自动推到最右侧：

```css
.message-manager-btn {
  margin-left: auto;
}
```

### 事件变更

原事件：
- `start-voice-call`
- `start-video-call`
- `start-screen-share`

保持不变，由 ChatToolbar 内部的 call-dropdown 根据用户选择触发对应事件。外部组件无需修改事件监听逻辑。
