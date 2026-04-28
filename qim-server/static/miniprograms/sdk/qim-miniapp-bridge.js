/**
 * 小程序桥接 SDK
 * 小程序通过此 SDK 与主程序通信，获取用户信息、调用后端 API 等
 *
 * 使用方式：在小程序 HTML 中引入此脚本：
 * <script src="/miniprograms/sdk/qim-miniapp-bridge.js"></script>
 */
(function() {
  'use strict';

  var QIMBridge = {
    appId: null,
    ready: false,
    listeners: {},

    init: function(callback) {
      var self = this;
      window.addEventListener('message', function(event) {
        var data = event.data;
        if (!data || typeof data !== 'object') return;

        switch (data.type) {
          case 'bridge-ready':
            self.appId = data.payload && data.payload.appId;
            self.ready = true;
            if (callback) callback();
            break;
          case 'user-info-response':
            self._emit('userInfo', data.payload);
            break;
          case 'token-response':
            self._emit('token', data.payload);
            break;
          case 'api-response':
            self._emit('apiResponse', data.payload);
            break;
        }
      });
    },

    getUserInfo: function() {
      var self = this;
      return new Promise(function(resolve, reject) {
        if (self.ready) {
          self._once('userInfo', function(data) {
            if (data && data.error) {
              reject(new Error(data.error));
            } else {
              resolve(data);
            }
          });
          parent.postMessage({ type: 'get-user-info' }, '*');
        } else {
          self.init(function() {
            self._once('userInfo', function(data) {
              if (data && data.error) {
                reject(new Error(data.error));
              } else {
                resolve(data);
              }
            });
            parent.postMessage({ type: 'get-user-info' }, '*');
          });
        }
      });
    },

    getToken: function() {
      var self = this;
      return new Promise(function(resolve, reject) {
        if (self.ready) {
          self._once('token', function(data) {
            if (data && data.error) {
              reject(new Error(data.error));
            } else {
              resolve(data);
            }
          });
          parent.postMessage({ type: 'get-token' }, '*');
        } else {
          self.init(function() {
            self._once('token', function(data) {
              if (data && data.error) {
                reject(new Error(data.error));
              } else {
                resolve(data);
              }
            });
            parent.postMessage({ type: 'get-token' }, '*');
          });
        }
      });
    },

    api: function(method, url, body) {
      var self = this;
      return new Promise(function(resolve, reject) {
        var payload = { method: method || 'GET', url: url };
        if (body) payload.body = body;

        if (self.ready) {
          self._once('apiResponse', function(data) {
            if (data && data.code === 403) {
              reject(new Error(data.message || '权限不足'));
            } else {
              resolve(data);
            }
          });
          parent.postMessage({ type: 'api-request', payload: payload }, '*');
        } else {
          self.init(function() {
            self._once('apiResponse', function(data) {
              if (data && data.code === 403) {
                reject(new Error(data.message || '权限不足'));
              } else {
                resolve(data);
              }
            });
            parent.postMessage({ type: 'api-request', payload: payload }, '*');
          });
        }
      });
    },

    showToast: function(message) {
      parent.postMessage({ type: 'miniapp-toast', payload: { message: message } }, '*');
    },

    readClipboard: function() {
      var self = this;
      return new Promise(function(resolve, reject) {
        if (self.ready) {
          self._once('clipboard-read-response', function(data) {
            if (data && data.error) {
              reject(new Error(data.error));
            } else {
              resolve(data && data.text);
            }
          });
          parent.postMessage({ type: 'clipboard-read' }, '*');
        } else {
          self.init(function() {
            self._once('clipboard-read-response', function(data) {
              if (data && data.error) {
                reject(new Error(data.error));
              } else {
                resolve(data && data.text);
              }
            });
            parent.postMessage({ type: 'clipboard-read' }, '*');
          });
        }
      });
    },

    writeClipboard: function(text) {
      var self = this;
      return new Promise(function(resolve, reject) {
        if (self.ready) {
          self._once('clipboard-write-response', function(data) {
            if (data && data.error) {
              reject(new Error(data.error));
            } else {
              resolve(data);
            }
          });
          parent.postMessage({ type: 'clipboard-write', payload: { text: text } }, '*');
        } else {
          self.init(function() {
            self._once('clipboard-write-response', function(data) {
              if (data && data.error) {
                reject(new Error(data.error));
              } else {
                resolve(data);
              }
            });
            parent.postMessage({ type: 'clipboard-write', payload: { text: text } }, '*');
          });
        }
      });
    },

    on: function(event, handler) {
      if (!this.listeners[event]) this.listeners[event] = [];
      this.listeners[event].push(handler);
    },

    off: function(event, handler) {
      if (!this.listeners[event]) return;
      var idx = this.listeners[event].indexOf(handler);
      if (idx !== -1) this.listeners[event].splice(idx, 1);
    },

    _emit: function(event, data) {
      var handlers = this.listeners[event];
      if (handlers) handlers.forEach(function(h) { h(data); });
    },

    _once: function(event, resolve) {
      var self = this;
      var handler = function(data) {
        resolve(data);
        self.off(event, handler);
      };
      this.on(event, handler);
    }
  };

  if (typeof window.QIMBridge === 'undefined') {
    window.QIMBridge = QIMBridge;
  }

  if (typeof define === 'function' && define.amd) {
    define(function() { return QIMBridge; });
  } else if (typeof module !== 'undefined' && module.exports) {
    module.exports = QIMBridge;
  }
})();
