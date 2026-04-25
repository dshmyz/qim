<template>
  <div class="window-controls">
    <div class="window-controls-spacer"></div>
    <div class="window-controls-right">
      <button class="window-control-btn minimize-btn" @click="minimizeWindow" title="最小化">—</button>
      <button class="window-control-btn maximize-btn" @click="maximizeWindow" title="最大化">☐</button>
      <button class="window-control-btn close-btn" @click="closeWindow" title="关闭">×</button>
    </div>
  </div>
</template>

<script setup lang="ts">
/**
 * WindowControls - 窗口控制栏组件
 * 提供最小化、最大化、关闭按钮
 */
const minimizeWindow = () => {
  console.log('Minimize window clicked')
  if (window.electron?.ipcRenderer) {
    window.electron.ipcRenderer.send('minimize-window')
  } else {
    console.log('Electron not available')
  }
}

const maximizeWindow = () => {
  console.log('Maximize window clicked')
  if (window.electron?.ipcRenderer) {
    window.electron.ipcRenderer.send('maximize-window')
  } else {
    console.log('Electron not available')
  }
}

const closeWindow = () => {
  console.log('Close window clicked')
  if (window.electron?.ipcRenderer) {
    window.electron.ipcRenderer.send('close-window')
  } else {
    console.log('Electron not available')
  }
}
</script>

<style scoped>
.window-controls {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  height: 100%;
  background: var(--window-controls-bg);
  padding: 0 20px;
  user-select: none;
  -webkit-app-region: drag;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.window-controls-spacer {
  flex: 1;
}

.window-controls-right {
  display: flex;
  align-items: center;
  gap: 12px;
  -webkit-app-region: no-drag;
}

.window-control-btn {
  width: 16px;
  height: 16px;
  border: none;
  border-radius: 50%;
  font-size: 11px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  -webkit-app-region: no-drag;
  line-height: 1;
  box-shadow: var(--shadow-sm);
}

.minimize-btn {
  background: #ffbd2e;
  color: white;
}

.maximize-btn {
  background: #27c93f;
  color: white;
}

.close-btn {
  background: #ff5f56;
  color: white;
}

.window-control-btn:hover {
  transform: scale(1.05);
  opacity: 0.9;
}
</style>
