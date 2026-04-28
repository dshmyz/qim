import { Type } from "@sinclair/typebox";

interface QIMWSMessage {
  type: string;
  data: any;
  request_id?: string;
}

interface QIMNewMessage {
  ID: number;
  ConversationID: number;
  SenderID: number;
  Type: string;
  Content: string;
  IsRead: boolean;
  QuotedMessageID?: number;
  CreatedAt: string;
  Sender?: {
    ID: number;
    Username: string;
    Nickname: string;
    Avatar: string;
  };
  QuotedMessage?: any;
}

type MessageListener = (message: QIMNewMessage) => void;

let globalLogger: any = null;

class QIMWebSocketClient {
  private ws: WebSocket | null = null;
  private apiUrl: string;
  private token: string = "";
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 5;
  private reconnectDelay: number = 3000;
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;
  private listeners: Set<MessageListener> = new Set();
  private isConnecting: boolean = false;
  private isManualClose: boolean = false;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private messageQueue: QIMWSMessage[] = [];

  constructor(apiUrl: string, token: string) {
    this.apiUrl = apiUrl;
    this.token = token;
  }

  setToken(token: string) {
    this.token = token;
  }

  async connect(): Promise<void> {
    if (this.isConnecting) {
      return;
    }

    if (!this.token) {
      throw new Error("No token available. Please login first.");
    }

    this.isConnecting = true;
    this.isManualClose = false;

    return new Promise((resolve, reject) => {
      const wsUrl = `${this.apiUrl.replace(/^http/, "ws")}/api/v1/ws?token=${this.token}`;

      try {
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          this.isConnecting = false;
          this.reconnectAttempts = 0;
          this.startHeartbeat();
          this.flushMessageQueue();
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message: QIMWSMessage = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
            globalLogger?.error(`Failed to parse WebSocket message: ${error}`);
          }
        };

        this.ws.onerror = (error) => {
          this.isConnecting = false;
          globalLogger?.error(`WebSocket error: ${error}`);
        };

        this.ws.onclose = () => {
          this.isConnecting = false;
          this.stopHeartbeat();

          if (!this.isManualClose) {
            this.handleReconnect();
          }
        };
      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    });
  }

  private handleMessage(message: QIMWSMessage): void {
    switch (message.type) {
      case "new_message":
        globalLogger?.info(`[QIM] 收到消息 from ${message.data?.Sender?.username || message.data?.SenderID}: ${message.data?.Content}`);
        this.notifyListeners(message.data as QIMNewMessage);
        break;
      case "message_read":
        globalLogger?.info(`[QIM] 消息已读: ${JSON.stringify(message.data)}`);
        break;
      case "heartbeat":
        break;
      default:
        globalLogger?.debug(`[QIM] 未知消息类型: ${message.type}`);
    }
  }

  private notifyListeners(message: QIMNewMessage): void {
    this.listeners.forEach((listener) => {
      try {
        listener(message);
      } catch (error) {
        globalLogger?.error(`Error in message listener: ${error}`);
      }
    });
  }

  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0) {
      const msg = this.messageQueue.shift();
      if (msg) {
        this.sendRaw(msg.type, msg.data);
      }
    }
  }

  addMessageListener(listener: MessageListener): void {
    this.listeners.add(listener);
  }

  removeMessageListener(listener: MessageListener): void {
    this.listeners.delete(listener);
  }

  private startHeartbeat(): void {
    this.heartbeatInterval = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: "heartbeat" }));
      }
    }, 30000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private handleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      globalLogger?.warn("[QIM] WebSocket 重连次数已达上限");
      return;
    }

    this.reconnectAttempts++;
    globalLogger?.info(`[QIM] 尝试重连 (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);

    this.reconnectTimer = setTimeout(() => {
      this.connect().catch((error) => {
        globalLogger?.error(`[QIM] 重连失败: ${error.message}`);
      });
    }, this.reconnectDelay);
  }

  private sendRaw(type: string, data: any): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      const message: QIMWSMessage = {
        type,
        data,
      };
      this.ws.send(JSON.stringify(message));
    } else {
      this.messageQueue.push({ type, data });
    }
  }

  send(type: string, data: any): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.sendRaw(type, data);
    } else {
      this.messageQueue.push({ type, data });
    }
  }

  sendMessage(conversationId: number, content: string, type: string = "text"): void {
    this.send("send_message", {
      conversation_id: conversationId,
      content,
      type,
    });
  }

  readMessage(conversationId: number, messageId: number): void {
    this.send("read_message", {
      conversation_id: conversationId,
      message_id: messageId,
    });
  }

  disconnect(): void {
    this.isManualClose = true;
    this.stopHeartbeat();

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}

class QIMClient {
  private apiUrl: string;
  private token: string = "";

  constructor(apiUrl: string) {
    this.apiUrl = apiUrl;
  }

  setToken(token: string) {
    this.token = token;
  }

  async login(username: string, password: string): Promise<{ token: string; user: any }> {
    const response = await fetch(`${this.apiUrl}/api/v1/auth/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
      throw new Error(`Login failed: ${response.status} ${response.statusText}`);
    }

    const data = await response.json();
    this.token = data.token;
    return { token: data.token, user: data.user };
  }

  async request(endpoint: string, options: RequestInit = {}) {
    if (!this.token) {
      throw new Error("Not authenticated. Please login first.");
    }

    const url = `${this.apiUrl}/api/v1${endpoint}`;
    const headers = {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${this.token}`,
      ...options.headers,
    };

    try {
      const response = await fetch(url, {
        ...options,
        headers,
      });

      if (!response.ok) {
        throw new Error(`API request failed: ${response.status} ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      throw new Error(`QIM API request failed: ${error.message}`);
    }
  }

  async sendMessage(conversationId: string, content: string, type: string = "text") {
    return this.request(`/conversations/${conversationId}/messages`, {
      method: "POST",
      body: JSON.stringify({ content, type }),
    });
  }

  async getMessages(conversationId: string, limit: number = 50, offset: number = 0) {
    return this.request(`/conversations/${conversationId}/messages?limit=${limit}&offset=${offset}`);
  }

  async getConversations() {
    return this.request("/conversations");
  }

  async createGroup(name: string, memberIds: number[]) {
    return this.request("/conversations/group", {
      method: "POST",
      body: JSON.stringify({ name, member_ids: memberIds }),
    });
  }

  async getCurrentUser() {
    return this.request("/users/me");
  }

  async searchUsers(keyword: string) {
    return this.request(`/users/search?keyword=${encodeURIComponent(keyword)}`);
  }

  async getConversation(conversationId: string) {
    return this.request(`/conversations/${conversationId}`);
  }

  async markAsRead(conversationId: string) {
    return this.request(`/conversations/${conversationId}/read`, {
      method: "POST",
    });
  }
}

let qimClient: QIMClient | null = null;
let wsClient: QIMWebSocketClient | null = null;

export default function register(api: any) {
  globalLogger = api.logger;

  const getConfig = () => {
    return api.config.plugins?.entries?.["qim-integration"]?.config || {};
  };

  const getQIMClient = (): QIMClient => {
    if (!qimClient) {
      const config = getConfig();
      const apiUrl = config.apiUrl || "http://localhost:8080";
      qimClient = new QIMClient(apiUrl);
    }
    return qimClient;
  };

  const getWSClient = (): QIMWebSocketClient => {
    if (!wsClient) {
      const config = getConfig();
      const apiUrl = config.apiUrl || "http://localhost:8080";
      wsClient = new QIMWebSocketClient(apiUrl, "");
    }
    return wsClient;
  };

  const ensureAuthenticated = async () => {
    const config = getConfig();
    if (!config.username || !config.password) {
      throw new Error("请先配置 QIM 用户名和密码");
    }

    const client = getQIMClient();
    if (!client) {
      throw new Error("QIM 客户端未初始化");
    }

    try {
      const { token } = await client.login(config.username, config.password);
      const ws = getWSClient();
      ws.setToken(token);
      api.logger.info("[QIM] 登录成功");
      return token;
    } catch (error) {
      api.logger.error(`[QIM] 登录失败: ${error.message}`);
      throw error;
    }
  };

  api.registerTool({
    name: "qim_login",
    description: "登录 QIM 并获取认证令牌",
    parameters: Type.Object({}),
    async execute(_id: string, _params: any) {
      try {
        const token = await ensureAuthenticated();
        return {
          content: [{
            type: "text",
            text: `登录成功！Token 已获取。`,
          }],
        };
      } catch (error) {
        return {
          content: [{
            type: "text",
            text: `登录失败: ${error.message}`,
          }],
        };
      }
    },
  });

  api.registerTool({
    name: "qim_send_message",
    description: "向 QIM 会话发送消息",
    parameters: Type.Object({
      conversationId: Type.String({ description: "会话 ID" }),
      content: Type.String({ description: "消息内容" }),
      type: Type.Optional(Type.String({ description: "消息类型，默认为 text" })),
    }),
    async execute(_id: string, params: any) {
      try {
        await ensureAuthenticated();
        const client = getQIMClient();
        const result = await client.sendMessage(params.conversationId, params.content, params.type);
        return {
          content: [{
            type: "text",
            text: `消息发送成功！`,
          }],
        };
      } catch (error) {
        return {
          content: [{
            type: "text",
            text: `消息发送失败: ${error.message}`,
          }],
        };
      }
    },
  });

  api.registerTool({
    name: "qim_get_messages",
    description: "获取 QIM 会话消息",
    parameters: Type.Object({
      conversationId: Type.String({ description: "会话 ID" }),
      limit: Type.Optional(Type.Number({ description: "消息数量限制，默认为 50" })),
      offset: Type.Optional(Type.Number({ description: "偏移量，默认为 0" })),
    }),
    async execute(_id: string, params: any) {
      try {
        await ensureAuthenticated();
        const client = getQIMClient();
        const messages = await client.getMessages(
          params.conversationId,
          params.limit || 50,
          params.offset || 0
        );
        return {
          content: [{
            type: "text",
            text: `获取到 ${messages.length || 0} 条消息:\n${JSON.stringify(messages, null, 2)}`,
          }],
        };
      } catch (error) {
        return {
          content: [{
            type: "text",
            text: `获取消息失败: ${error.message}`,
          }],
        };
      }
    },
  });

  api.registerTool({
    name: "qim_get_conversations",
    description: "获取 QIM 会话列表",
    parameters: Type.Object({}),
    async execute(_id: string, _params: any) {
      try {
        await ensureAuthenticated();
        const client = getQIMClient();
        const conversations = await client.getConversations();
        return {
          content: [{
            type: "text",
            text: `获取到 ${conversations.length || 0} 个会话:\n${JSON.stringify(conversations, null, 2)}`,
          }],
        };
      } catch (error) {
        return {
          content: [{
            type: "text",
            text: `获取会话列表失败: ${error.message}`,
          }],
        };
      }
    },
  });

  api.registerTool({
    name: "qim_create_group",
    description: "在 QIM 中创建群组",
    parameters: Type.Object({
      name: Type.String({ description: "群组名称" }),
      memberIds: Type.Array(Type.Number({ description: "成员 ID 列表" })),
    }),
    async execute(_id: string, params: any) {
      try {
        await ensureAuthenticated();
        const client = getQIMClient();
        const group = await client.createGroup(params.name, params.memberIds);
        return {
          content: [{
            type: "text",
            text: `群组创建成功！群组 ID: ${group.id}`,
          }],
        };
      } catch (error) {
        return {
          content: [{
            type: "text",
            text: `群组创建失败: ${error.message}`,
          }],
        };
      }
    },
  });

  api.registerTool({
    name: "qim_get_current_user",
    description: "获取当前用户信息",
    parameters: Type.Object({}),
    async execute(_id: string, _params: any) {
      try {
        await ensureAuthenticated();
        const client = getQIMClient();
        const user = await client.getCurrentUser();
        return {
          content: [{
            type: "text",
            text: `当前用户信息:\n${JSON.stringify(user, null, 2)}`,
          }],
        };
      } catch (error) {
        return {
          content: [{
            type: "text",
            text: `获取用户信息失败: ${error.message}`,
          }],
        };
      }
    },
  });

  api.registerTool({
    name: "qim_search_users",
    description: "搜索 QIM 用户",
    parameters: Type.Object({
      keyword: Type.String({ description: "搜索关键词" }),
    }),
    async execute(_id: string, params: any) {
      try {
        await ensureAuthenticated();
        const client = getQIMClient();
        const users = await client.searchUsers(params.keyword);
        return {
          content: [{
            type: "text",
            text: `找到 ${users.length || 0} 个用户:\n${JSON.stringify(users, null, 2)}`,
          }],
        };
      } catch (error) {
        return {
          content: [{
            type: "text",
            text: `搜索用户失败: ${error.message}`,
          }],
        };
      }
    },
  });

  api.registerTool({
    name: "qim_connect_websocket",
    description: "连接 QIM WebSocket 获取实时消息",
    parameters: Type.Object({}),
    async execute(_id: string, _params: any) {
      try {
        await ensureAuthenticated();
        const ws = getWSClient();
        await ws.connect();

        ws.addMessageListener((message) => {
          api.logger.info(`[QIM] 新消息 from ${message.Sender?.username || message.SenderID}: ${message.Content}`);
        });

        api.logger.info("[QIM] WebSocket 连接成功，已开始监听消息");
        return {
          content: [{
            type: "text",
            text: `WebSocket 连接成功！正在监听新消息...\n收到消息时会显示在这里。`,
          }],
        };
      } catch (error) {
        api.logger.error(`[QIM] WebSocket 连接失败: ${error.message}`);
        return {
          content: [{
            type: "text",
            text: `WebSocket 连接失败: ${error.message}`,
          }],
        };
      }
    },
  });

  api.registerTool({
    name: "qim_disconnect_websocket",
    description: "断开 QIM WebSocket 连接",
    parameters: Type.Object({}),
    async execute(_id: string, _params: any) {
      try {
        const ws = getWSClient();
        ws.disconnect();
        api.logger.info("[QIM] WebSocket 连接已断开");
        return {
          content: [{
            type: "text",
            text: `WebSocket 连接已断开`,
          }],
        };
      } catch (error) {
        return {
          content: [{
            type: "text",
            text: `断开连接失败: ${error.message}`,
          }],
        };
      }
    },
  });

  api.registerTool({
    name: "qim_get_websocket_status",
    description: "获取 QIM WebSocket 连接状态",
    parameters: Type.Object({}),
    async execute(_id: string, _params: any) {
      try {
        const ws = getWSClient();
        const isConnected = ws.isConnected();
        return {
          content: [{
            type: "text",
            text: `WebSocket 连接状态: ${isConnected ? "已连接" : "未连接"}`,
          }],
        };
      } catch (error) {
        return {
          content: [{
            type: "text",
            text: `获取连接状态失败: ${error.message}`,
          }],
        };
      }
    },
  });

  api.registerCli(({ program }: any) => {
    const qimCommand = program.command("qim").description("QIM 集成命令");

    qimCommand
      .command("login")
      .description("登录 QIM")
      .action(async () => {
        try {
          await ensureAuthenticated();
          console.log("登录成功！");
        } catch (error) {
          console.error(`登录失败: ${error.message}`);
        }
      });

    qimCommand
      .command("conversations")
      .description("获取会话列表")
      .action(async () => {
        try {
          await ensureAuthenticated();
          const client = getQIMClient();
          const conversations = await client.getConversations();
          console.log(`获取到 ${conversations.length} 个会话:`);
          console.log(JSON.stringify(conversations, null, 2));
        } catch (error) {
          console.error(`获取会话列表失败: ${error.message}`);
        }
      });

    qimCommand
      .command("send")
      .description("发送消息")
      .requiredOption("--to <conversationId>", "会话 ID")
      .requiredOption("--content <content>", "消息内容")
      .action(async (options: any) => {
        try {
          await ensureAuthenticated();
          const client = getQIMClient();
          await client.sendMessage(options.to, options.content);
          console.log("消息发送成功！");
        } catch (error) {
          console.error(`消息发送失败: ${error.message}`);
        }
      });

    qimCommand
      .command("ws-connect")
      .description("连接 QIM WebSocket")
      .action(async () => {
        try {
          await ensureAuthenticated();
          const ws = getWSClient();
          await ws.connect();

          ws.addMessageListener((message) => {
            console.log(`[收到消息] from ${message.Sender?.username || message.SenderID}: ${message.Content}`);
          });

          console.log("WebSocket 连接成功！正在监听新消息...");
        } catch (error) {
          console.error(`WebSocket 连接失败: ${error.message}`);
        }
      });

    qimCommand
      .command("ws-status")
      .description("查看 WebSocket 连接状态")
      .action(async () => {
        try {
          const ws = getWSClient();
          const isConnected = ws.isConnected();
          console.log(`WebSocket 连接状态: ${isConnected ? "已连接" : "未连接"}`);
        } catch (error) {
          console.error(`获取连接状态失败: ${error.message}`);
        }
      });

    qimCommand
      .command("user")
      .description("获取当前用户信息")
      .action(async () => {
        try {
          await ensureAuthenticated();
          const client = getQIMClient();
          const user = await client.getCurrentUser();
          console.log("当前用户信息:");
          console.log(JSON.stringify(user, null, 2));
        } catch (error) {
          console.error(`获取用户信息失败: ${error.message}`);
        }
      });
  }, {
    commands: ["qim"],
  });

  api.registerHttpRoute({
    path: "/qim/webhook",
    auth: "plugin",
    match: "exact",
    handler: async (req: any, res: any) => {
      try {
        const body = await req.json();
        api.logger.info("[QIM] Webhook received:", body);
        res.statusCode = 200;
        res.end(JSON.stringify({ success: true }));
        return true;
      } catch (error) {
        res.statusCode = 500;
        res.end(JSON.stringify({ success: false, error: error.message }));
        return true;
      }
    },
  });

  api.logger.info("[QIM] 插件已加载");
}