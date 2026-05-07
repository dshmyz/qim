# QIM 登录页重设计 & 启动优化

## 概述

重新设计登录页视觉风格，并优化 Electron 应用启动时的白屏问题。

## 登录页设计

### 视觉风格

- **背景**：柔和的灰蓝色渐变 `linear-gradient(135deg, #e8ecf1 0%, #d5dde5 100%)`，搭配 2-3 个半透明圆形装饰元素
- **登录卡片**：纯白色背景，圆角 16px，柔和阴影 `0 4px 24px rgba(0,0,0,0.08)`
- **Logo 区域**：渐变色圆角方块 `linear-gradient(135deg, #64b5f6 0%, #4fc3f7 100%)`，尺寸 56x56px
- **标题**：「QIM 青雀」，22px，字重 600，字间距 1px
- **输入框**：浅灰背景 `#f5f7fa`，圆角 8px，边框 `1px solid #e8ecf1`
- **登录按钮**：渐变色 `linear-gradient(135deg, #64b5f6 0%, #4fc3f7 100%)`，圆角 8px，高度 48px
- **整体卡片宽度**：360px，内边距 40px

### 功能保持不变

- 用户名/密码登录
- 双因素认证表单
- 记住密码
- 服务器设置
- 窗口控制按钮（最小化/最大化/关闭）
- Electron 无边框窗口支持

### 修改文件

- `qim-client/src/views/Login.vue` — 更新 `<style scoped>` 中的 CSS

## 启动优化

### 方案：内联启动 HTML

在 Electron 主进程中，先加载一个本地的小型 HTML 启动页面，等主页面（Vite dev server 或打包后的 dist）加载完成后再切换。

### 实现步骤

1. **创建启动页面** `qim-client/electron/splash.html`
   - 全屏居中显示 Logo + 「QIM 青雀」+ 加载进度条动画
   - 背景色与登录页一致 `#e8ecf1`
   - 使用内联 CSS，无外部依赖

2. **修改 `electron/main.js`**
   - `createWindow()` 中先加载 `splash.html`
   - 创建主窗口但设置 `show: false`
   - 监听主窗口 `ready-to-show` 事件
   - 主窗口准备好后关闭 splash 窗口，显示主窗口

3. **窗口配置调整**
   ```js
   mainWindow = new BrowserWindow({
     width: 1200,
     height: 800,
     show: false, // 先不显示
     backgroundColor: '#e8ecf1',
     // ... 其他配置
   })
   
   mainWindow.once('ready-to-show', () => {
     splashWindow.close()
     mainWindow.show()
   })
   ```

### 修改文件

- `qim-client/electron/splash.html` — 新建
- `qim-client/electron/main.js` — 修改 `createWindow()` 函数

## 技术考虑

- 启动页面使用内联样式，不依赖任何外部资源，确保即时显示
- `ready-to-show` 事件比 `did-finish-load` 更准确，表示页面已渲染完成可以显示
- 开发模式下 Vite 启动可能需要几秒，splash 页面能有效掩盖这段时间
- 生产模式下打包后的 HTML 加载较快，splash 页面会短暂闪现
