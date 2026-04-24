const { contextBridge, ipcRenderer, shell } = require('electron')

contextBridge.exposeInMainWorld('electron', {
  ipcRenderer: {
    send: (channel, data) => {
      ipcRenderer.send(channel, data)
    },
    on: (channel, callback) => {
      ipcRenderer.on(channel, (event, ...args) => callback(event, ...args))
    },
    once: (channel, callback) => {
      ipcRenderer.once(channel, (event, ...args) => callback(event, ...args))
    }
  },
  shell: {
    openExternal: (url) => {
      shell.openExternal(url)
    }
  },
  screenshot: {
    take: () => {
      ipcRenderer.send('take-screenshot')
    },
    onTaken: (callback) => {
      ipcRenderer.on('screenshot-taken', (event, data) => callback(data))
    }
  },
  websocket: {
    send: (message) => {
      ipcRenderer.send('send-websocket-message', message)
    },
    onMessage: (callback) => {
      ipcRenderer.on('websocket-message', (event, message) => callback(message))
    }
  },
  webrtc: {
    send: (message) => {
      ipcRenderer.send('webrtc-message', message)
    },
    onMessage: (callback) => {
      ipcRenderer.on('webrtc-message', (event, message) => callback(message))
    }
  },
  tray: {
    flash: () => {
      ipcRenderer.send('flash-tray')
    },
    stopFlash: () => {
      ipcRenderer.send('stop-tray-flash')
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
