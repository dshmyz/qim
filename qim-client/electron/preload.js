const { contextBridge, ipcRenderer, shell } = require('electron')

// 暴露 electron API 到渲染进程
contextBridge.exposeInMainWorld('electron', {
  ipcRenderer: {
    send: (channel, data) => {
      ipcRenderer.send(channel, data)
    },
    on: (channel, callback) => {
      ipcRenderer.on(channel, (event, ...args) => callback(event, ...args))
    }
  },
  shell: {
    openExternal: (url) => {
      shell.openExternal(url)
    }
  }
})

window.addEventListener('DOMContentLoaded', () => {
  const replaceText = (selector, text) => {
    const element = document.getElementById(selector)
    if (element) element.innerText = text
  }

  for (const type of ['chrome', 'node', 'electron']) {
    replaceText(`${type}-version`, process.versions[type])
  }
})
