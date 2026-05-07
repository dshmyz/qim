# 登录页重设计 & 启动优化 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 重新设计登录页视觉风格（柔和渐变背景 + 简约白色卡片），并添加 Electron 内联启动 HTML 页面消除白屏

**架构：** 修改 Login.vue 的 CSS 样式实现新视觉风格；创建 splash.html 并在 main.js 中实现启动页面逻辑

**技术栈：** Vue 3, Electron, HTML/CSS

---

## 文件结构

- **修改：** `qim-client/src/views/Login.vue` — 更新 `<style scoped>` 中的 CSS 样式
- **创建：** `qim-client/electron/splash.html` — 内联启动页面
- **修改：** `qim-client/electron/main.js` — 添加 splash 窗口逻辑，调整主窗口创建参数

---

### 任务 1：创建 Electron 启动页面 splash.html

**文件：**
- 创建：`qim-client/electron/splash.html`

- [ ] **步骤 1：创建 splash.html 文件**

```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>
    * {
      margin: 0;
      padding: 0;
      box-sizing: border-box;
    }

    body {
      width: 100vw;
      height: 100vh;
      background: linear-gradient(135deg, #e8ecf1 0%, #d5dde5 100%);
      display: flex;
      align-items: center;
      justify-content: center;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      overflow: hidden;
      -webkit-app-region: drag;
    }

    .splash-container {
      text-align: center;
      animation: fadeIn 0.3s ease-out;
    }

    @keyframes fadeIn {
      from { opacity: 0; transform: scale(0.95); }
      to { opacity: 1; transform: scale(1); }
    }

    .splash-logo {
      width: 64px;
      height: 64px;
      background: linear-gradient(135deg, #64b5f6 0%, #4fc3f7 100%);
      border-radius: 16px;
      margin: 0 auto 20px;
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-size: 28px;
      font-weight: bold;
      box-shadow: 0 4px 12px rgba(100, 181, 246, 0.3);
    }

    .splash-title {
      font-size: 22px;
      font-weight: 600;
      color: #333;
      letter-spacing: 1px;
      margin-bottom: 6px;
    }

    .splash-subtitle {
      font-size: 13px;
      color: #888;
      margin-bottom: 28px;
    }

    .splash-loader {
      width: 120px;
      height: 3px;
      background: rgba(0, 0, 0, 0.08);
      border-radius: 2px;
      margin: 0 auto;
      overflow: hidden;
    }

    .splash-loader-bar {
      width: 40%;
      height: 100%;
      background: linear-gradient(90deg, #64b5f6, #4fc3f7);
      border-radius: 2px;
      animation: loading 1.5s ease-in-out infinite;
    }

    @keyframes loading {
      0% { transform: translateX(-100%); }
      100% { transform: translateX(350%); }
    }

    .splash-version {
      position: absolute;
      bottom: 20px;
      left: 0;
      right: 0;
      text-align: center;
      font-size: 11px;
      color: #aaa;
    }
  </style>
</head>
<body>
  <div class="splash-container">
    <div class="splash-logo">Q</div>
    <div class="splash-title">QIM 青雀</div>
    <div class="splash-subtitle">即时通讯应用</div>
    <div class="splash-loader">
      <div class="splash-loader-bar"></div>
    </div>
  </div>
  <div class="splash-version">v1.0.0</div>
</body>
</html>
```

- [ ] **步骤 2：验证文件创建成功**

确认文件路径为 `qim-client/electron/splash.html`，内容完整无截断。

---

### 任务 2：修改 main.js 添加启动页面逻辑

**文件：**
- 修改：`qim-client/electron/main.js` — `createWindow()` 函数

- [ ] **步骤 1：修改 createWindow 函数，添加 splash 窗口**

在 `createWindow()` 函数中，在创建 `mainWindow` 之前添加 splash 窗口创建逻辑。找到 `createWindow()` 函数中的这段代码：

```js
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    icon: icon,
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: false
    },
    frame: false,
    titleBarStyle: 'customButtonsOnHover',
    titleBarOverlay: {
      visible: false,
      height: 0
    },
    trafficLightPosition: { x: -100, y: -100 }
  })
```

替换为：

```js
  // 创建启动页面窗口
  const splashWindow = new BrowserWindow({
    width: 360,
    height: 320,
    frame: false,
    transparent: true,
    alwaysOnTop: true,
    resizable: false,
    skipTaskbar: true,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true
    }
  })

  const splashPath = `file://${path.join(__dirname, 'splash.html')}`
  splashWindow.loadURL(splashPath)
  console.log(`Loading splash: ${splashPath}`)

  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    icon: icon,
    show: false,
    backgroundColor: '#e8ecf1',
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: false
    },
    frame: false,
    titleBarStyle: 'customButtonsOnHover',
    titleBarOverlay: {
      visible: false,
      height: 0
    },
    trafficLightPosition: { x: -100, y: -100 }
  })
```

- [ ] **步骤 2：添加 ready-to-show 事件监听，切换 splash 和主窗口**

找到这段代码：

```js
  mainWindow.webContents.on('did-finish-load', () => {
    console.log('Render process loaded')
    if (process.env.NODE_ENV !== 'production') {
      console.log('Opening DevTools in development mode')
      mainWindow.webContents.openDevTools()
    }
  })
```

替换为：

```js
  mainWindow.webContents.on('did-finish-load', () => {
    console.log('Render process loaded')
    if (process.env.NODE_ENV !== 'production') {
      console.log('Opening DevTools in development mode')
      mainWindow.webContents.openDevTools()
    }
  })

  mainWindow.once('ready-to-show', () => {
    console.log('Main window ready to show, closing splash')
    splashWindow.close()
    mainWindow.show()
  })
```

- [ ] **步骤 3：处理窗口销毁时清理 splash 窗口引用**

找到这段代码：

```js
  mainWindow.on('close', function () {
    mainWindow = null
  })
```

替换为：

```js
  mainWindow.on('close', function () {
    if (splashWindow && !splashWindow.isDestroyed()) {
      splashWindow.close()
    }
    mainWindow = null
  })
```

---

### 任务 3：更新 Login.vue 样式

**文件：**
- 修改：`qim-client/src/views/Login.vue` — `<style scoped>` 部分

- [ ] **步骤 1：修改背景渐变和装饰圆形**

找到：

```css
.login-container {
  width: 100vw;
  height: 100vh;
  background: linear-gradient(135deg, #64b5f6 0%, #4fc3f7 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  -webkit-app-region: drag;
}
```

替换为：

```css
.login-container {
  width: 100vw;
  height: 100vh;
  background: linear-gradient(135deg, #e8ecf1 0%, #d5dde5 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  -webkit-app-region: drag;
}
```

找到：

```css
.circle {
  position: absolute;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  animation: float 6s ease-in-out infinite;
}
```

替换为：

```css
.circle {
  position: absolute;
  border-radius: 50%;
  background: rgba(100, 181, 246, 0.08);
  animation: float 6s ease-in-out infinite;
}
```

- [ ] **步骤 2：修改登录卡片样式**

找到：

```css
.login-form {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  padding: 56px 48px;
  width: 560px;
  min-height: 500px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.3);
  z-index: 1;
  animation: slideIn 0.5s ease-out;
  -webkit-app-region: no-drag;
}
```

替换为：

```css
.login-form {
  background: #ffffff;
  border-radius: 16px;
  padding: 48px 40px;
  width: 360px;
  min-height: 460px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
  z-index: 1;
  animation: slideIn 0.5s ease-out;
  -webkit-app-region: no-drag;
}
```

- [ ] **步骤 3：修改 Logo 区域样式**

找到：

```css
.app-logo {
  width: 80px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 24px;
  animation: pulse 2s ease-in-out infinite;
}
```

替换为：

```css
.app-logo {
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 16px;
  background: linear-gradient(135deg, #64b5f6 0%, #4fc3f7 100%);
  border-radius: 14px;
  box-shadow: 0 4px 12px rgba(100, 181, 246, 0.3);
}
```

删除 `@keyframes pulse` 整个动画定义：

```css
@keyframes pulse {
  0%, 100% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.05);
  }
}
```

- [ ] **步骤 4：修改标题文字样式**

找到：

```css
.login-header h2 {
  margin: 0 0 8px 0;
  font-size: 28px;
  font-weight: 700;
  color: #333;
  letter-spacing: 2px;
}

.login-header p {
  margin: 0;
  font-size: 16px;
  color: #666;
}
```

替换为：

```css
.login-header h2 {
  margin: 0 0 6px 0;
  font-size: 22px;
  font-weight: 600;
  color: #333;
  letter-spacing: 1px;
}

.login-header p {
  margin: 0;
  font-size: 13px;
  color: #888;
}
```

- [ ] **步骤 5：修改输入框样式**

找到：

```css
.input-wrapper {
  position: relative;
  border-radius: 8px;
  background: rgba(240, 242, 245, 0.8);
  transition: all 0.3s ease;
  width: 100%;
  box-sizing: border-box;
  border: 1px solid transparent;
}

.input-wrapper:focus-within {
  background: white;
  box-shadow: 0 0 0 3px rgba(100, 181, 246, 0.2);
  transform: translateY(-2px);
}
```

替换为：

```css
.input-wrapper {
  position: relative;
  border-radius: 8px;
  background: #f5f7fa;
  transition: all 0.3s ease;
  width: 100%;
  box-sizing: border-box;
  border: 1px solid #e8ecf1;
}

.input-wrapper:focus-within {
  background: #ffffff;
  border-color: #64b5f6;
  box-shadow: 0 0 0 3px rgba(100, 181, 246, 0.1);
}
```

找到：

```css
.login-input {
  width: 100%;
  padding: 16px 16px 16px 48px;
  border: none;
  background: transparent;
  border-radius: 8px;
  font-size: 16px;
  transition: all 0.3s ease;
  outline: none;
  box-sizing: border-box;
}
```

替换为：

```css
.login-input {
  width: 100%;
  padding: 14px 14px 14px 48px;
  border: none;
  background: transparent;
  border-radius: 8px;
  font-size: 15px;
  transition: all 0.3s ease;
  outline: none;
  box-sizing: border-box;
}
```

- [ ] **步骤 6：修改登录按钮样式**

找到：

```css
.login-button {
  width: 100%;
  height: 56px;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  background: linear-gradient(135deg, #64b5f6 0%, #4fc3f7 100%);
  border: none;
  cursor: pointer;
  transition: all 0.3s ease;
  color: white;
}

.login-button:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(100, 181, 246, 0.4);
  background: linear-gradient(135deg, #42a5f5 0%, #29b6f6 100%);
}
```

替换为：

```css
.login-button {
  width: 100%;
  height: 48px;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 500;
  background: linear-gradient(135deg, #64b5f6 0%, #4fc3f7 100%);
  border: none;
  cursor: pointer;
  transition: all 0.3s ease;
  color: white;
}

.login-button:hover:not(:disabled) {
  box-shadow: 0 4px 12px rgba(100, 181, 246, 0.3);
  background: linear-gradient(135deg, #42a5f5 0%, #29b6f6 100%);
}
```

- [ ] **步骤 7：修改表单间距**

找到：

```css
.form-item {
  margin-bottom: 24px;
}
```

替换为：

```css
.form-item {
  margin-bottom: 18px;
}
```

找到：

```css
.form-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
```

替换为：

```css
.form-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 18px;
}
```

找到：

```css
.login-header {
  text-align: center;
  margin-bottom: 32px;
}
```

替换为：

```css
.login-header {
  text-align: center;
  margin-bottom: 28px;
}
```

- [ ] **步骤 8：修改版本号文字颜色**

找到：

```css
.info-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 10px;
  color: rgba(102, 102, 102, 0.6);
}
```

替换为：

```css
.info-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 10px;
  color: rgba(102, 102, 102, 0.5);
}
```

---

### 任务 4：验证和测试

**文件：** 无

- [ ] **步骤 1：启动开发服务器验证登录页**

运行：`cd qim-client && npm run dev`

预期：Vite 开发服务器启动，打开 `http://localhost:3000` 看到新的登录页样式

- [ ] **步骤 2：验证 Electron 启动页面**

运行：`cd qim-client && npm run electron:dev`（需要先启动 Vite dev server）

预期：
1. 先看到 splash 启动页面（Logo + 加载动画）
2. 主窗口加载完成后，splash 关闭，主窗口显示
3. 无白屏闪烁

- [ ] **步骤 3：验证登录功能正常**

预期：用户名/密码输入、登录按钮、记住密码、服务器设置、双因素认证等功能正常工作

---

### 任务 5：Commit

- [ ] **步骤 1：提交所有变更**

```bash
cd /Users/gracegaoya/work/project/qim
git add qim-client/electron/splash.html qim-client/electron/main.js qim-client/src/views/Login.vue
git commit -m "feat: 重设计登录页并添加启动页面消除白屏"
```
