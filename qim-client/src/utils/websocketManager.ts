// WebSocket 管理器 - 简洁版

// 定义消息类型
export interface WebSocketMessage {
  type: string;
  data: any;
}

// 消息处理器类型
export type MessageHandler = (message: WebSocketMessage) => boolean;

// WebSocket 连接实例
let ws: WebSocket | null = null;

// 消息处理器映射，按消息类型分类
const messageHandlers: Map<string, MessageHandler[]> = new Map();

// 通用消息处理器（处理所有未分类的消息）
const generalHandlers: MessageHandler[] = [];

/**
 * 连接 WebSocket
 * @param serverUrl 服务器地址
 * @param token 用户令牌
 */
export const connectWebSocket = (serverUrl: string, token: string): WebSocket => {
  if (ws && ws.readyState === WebSocket.OPEN) {
    return ws;
  }
  
  // 连接 WebSocket
  const wsUrl = `ws://${serverUrl.replace('http://', '')}/api/v1/ws?token=${token}`;
  ws = new WebSocket(wsUrl);
  
  // 暴露到全局，供其他模块使用
  if (ws) {
    (window as any).ws = ws;
  }
  
  ws.onopen = () => {
    console.log('WebSocket 连接成功');
  };
  
  ws.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data) as WebSocketMessage;
      handleWebSocketMessage(message);
    } catch (error) {
      console.error('解析 WebSocket 消息失败:', error);
    }
  };
  
  ws.onclose = () => {
    console.log('WebSocket 连接关闭');
  };
  
  ws.onerror = (error) => {
    console.error('WebSocket 错误:', error);
  };
  
  return ws;
};

/**
 * 处理 WebSocket 消息
 * @param message 消息对象
 */
const handleWebSocketMessage = (message: WebSocketMessage) => {
  const { type } = message;
  
  // 首先处理特定类型的消息
  if (messageHandlers.has(type)) {
    const handlers = messageHandlers.get(type)!;
    for (const handler of handlers) {
      if (handler(message)) {
        return; // 消息被处理，停止继续处理
      }
    }
  }
  
  // 然后处理通用消息
  for (const handler of generalHandlers) {
    if (handler(message)) {
      return; // 消息被处理，停止继续处理
    }
  }
};

/**
 * 添加消息处理器
 * @param handler 消息处理器函数
 * @param messageType 消息类型（可选，不指定则处理所有消息）
 */
export const addMessageHandler = (handler: MessageHandler, messageType?: string) => {
  if (messageType) {
    if (!messageHandlers.has(messageType)) {
      messageHandlers.set(messageType, []);
    }
    messageHandlers.get(messageType)!.push(handler);
  } else {
    generalHandlers.push(handler);
  }
};

/**
 * 移除消息处理器
 * @param handler 消息处理器函数
 * @param messageType 消息类型（可选，不指定则从通用处理器中移除）
 */
export const removeMessageHandler = (handler: MessageHandler, messageType?: string) => {
  if (messageType) {
    if (messageHandlers.has(messageType)) {
      const handlers = messageHandlers.get(messageType)!;
      const index = handlers.indexOf(handler);
      if (index !== -1) {
        handlers.splice(index, 1);
      }
    }
  } else {
    const index = generalHandlers.indexOf(handler);
    if (index !== -1) {
      generalHandlers.splice(index, 1);
    }
  }
};

/**
 * 获取 WebSocket 实例
 * @returns WebSocket 实例
 */
export const getWebSocket = (): WebSocket | null => {
  return ws;
};

/**
 * 发送 WebSocket 消息
 * @param message 消息对象
 */
export const sendWebSocketMessage = (message: any): void => {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(message));
  } else {
    console.error('WebSocket 连接未就绪，无法发送消息');
  }
};

/**
 * 关闭 WebSocket 连接
 */
export const closeWebSocket = (): void => {
  if (ws) {
    ws.close();
    ws = null;
  }
};
