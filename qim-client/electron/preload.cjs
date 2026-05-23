const { contextBridge, ipcRenderer, shell } = require('electron')

// Track listeners for cleanup
const listenerMap = new Map()

contextBridge.exposeInMainWorld('electron', {
  ipcRenderer: {
    send: (channel, data) => {
      ipcRenderer.send(channel, data)
    },
    invoke: (channel, data) => {
      return ipcRenderer.invoke(channel, data)
    },
    on: (channel, callback) => {
      const listener = (event, ...args) => callback(event, ...args)
      // Store the mapping so it can be removed later
      if (!listenerMap.has(channel)) {
        listenerMap.set(channel, new Set())
      }
      listenerMap.get(channel).add({ callback, listener })
      ipcRenderer.on(channel, listener)
    },
    once: (channel, callback) => {
      const listener = (event, ...args) => {
        callback(event, ...args)
        // Clean up after once fires
        const listeners = listenerMap.get(channel)
        if (listeners) {
          const entry = [...listeners].find(e => e.callback === callback)
          if (entry) {
            listeners.delete(entry)
          }
        }
      }
      if (!listenerMap.has(channel)) {
        listenerMap.set(channel, new Set())
      }
      listenerMap.get(channel).add({ callback, listener })
      ipcRenderer.once(channel, listener)
    },
    removeListener: (channel, callback) => {
      const listeners = listenerMap.get(channel)
      if (listeners) {
        const entry = [...listeners].find(e => e.callback === callback)
        if (entry) {
          listeners.delete(entry)
          ipcRenderer.removeListener(channel, entry.listener)
        }
      }
    },
    removeAllListeners: (channel) => {
      const listeners = listenerMap.get(channel)
      if (listeners) {
        listeners.forEach(({ listener }) => {
          ipcRenderer.removeListener(channel, listener)
        })
        listenerMap.delete(channel)
      }
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
      const listener = (event, data) => callback(data)
      if (!listenerMap.has('screenshot-taken')) {
        listenerMap.set('screenshot-taken', new Set())
      }
      listenerMap.get('screenshot-taken').add({ callback, listener })
      ipcRenderer.on('screenshot-taken', listener)
    },
    removeOnTaken: (callback) => {
      const listeners = listenerMap.get('screenshot-taken')
      if (listeners) {
        const entry = [...listeners].find(e => e.callback === callback)
        if (entry) {
          listeners.delete(entry)
          ipcRenderer.removeListener('screenshot-taken', entry.listener)
        }
      }
    }
  },
  websocket: {
    send: (message) => {
      ipcRenderer.send('send-websocket-message', message)
    },
    onMessage: (callback) => {
      const listener = (event, message) => callback(message)
      if (!listenerMap.has('websocket-message')) {
        listenerMap.set('websocket-message', new Set())
      }
      listenerMap.get('websocket-message').add({ callback, listener })
      ipcRenderer.on('websocket-message', listener)
    },
    removeOnMessage: (callback) => {
      const listeners = listenerMap.get('websocket-message')
      if (listeners) {
        const entry = [...listeners].find(e => e.callback === callback)
        if (entry) {
          listeners.delete(entry)
          ipcRenderer.removeListener('websocket-message', entry.listener)
        }
      }
    }
  },
  webrtc: {
    send: (message) => {
      ipcRenderer.send('webrtc-message', message)
    },
    onMessage: (callback) => {
      const listener = (event, message) => callback(message)
      if (!listenerMap.has('webrtc-message')) {
        listenerMap.set('webrtc-message', new Set())
      }
      listenerMap.get('webrtc-message').add({ callback, listener })
      ipcRenderer.on('webrtc-message', listener)
    },
    removeOnMessage: (callback) => {
      const listeners = listenerMap.get('webrtc-message')
      if (listeners) {
        const entry = [...listeners].find(e => e.callback === callback)
        if (entry) {
          listeners.delete(entry)
          ipcRenderer.removeListener('webrtc-message', entry.listener)
        }
      }
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
