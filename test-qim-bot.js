const WebSocket = require('ws');

const config = {
  apiUrl: 'http://localhost:8080',
  username: 'user1',
  password: '123456',
  autoReply: true,
  replyDelay: 1000
};

let token = null;

async function login() {
  console.log('正在登录 QIM...');
  const response = await fetch(`${config.apiUrl}/api/v1/auth/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      username: config.username,
      password: config.password
    })
  });
  
  if (!response.ok) {
    throw new Error(`登录失败: ${response.statusText}`);
  }
  
  const data = await response.json();
  token = data.data.token;
  console.log('✅ 登录成功');
  return token;
}

async function connectWebSocket() {
  console.log('正在连接 WebSocket...');
  const ws = new WebSocket(`${config.apiUrl.replace('http', 'ws')}/api/v1/ws?token=${token}`);
  
  ws.on('open', () => {
    console.log('✅ WebSocket 连接成功！');
    console.log('正在监听新消息...');
    console.log('提示：按 Ctrl+C 退出');
  });
  
  ws.on('message', async (message) => {
    try {
      const data = JSON.parse(message.toString());
      
      if (data.type === 'message' && data.data) {
        const msg = data.data;
        console.log(`\n[收到消息] from ${msg.Sender?.Username || msg.SenderID}: ${msg.Content}`);
        
        if (config.autoReply && msg.SenderID !== 'bot') {
          await new Promise(resolve => setTimeout(resolve, config.replyDelay));
          await sendReply(msg.ConversationID, msg.Content);
        }
      }
    } catch (error) {
      console.error('消息解析错误:', error);
    }
  });
  
  ws.on('error', (error) => {
    console.error('WebSocket 错误:', error);
  });
  
  ws.on('close', () => {
    console.log('WebSocket 连接已关闭');
  });
  
  return ws;
}

async function sendReply(conversationId, content) {
  console.log('正在生成回复...');
  
  try {
    const response = await fetch(`${config.apiUrl}/api/v1/ai/completion`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify({
        prompt: content,
        model: 'openai/qwen3.6-plus',
        max_tokens: 500
      })
    });
    
    if (!response.ok) {
      throw new Error(`AI 调用失败: ${response.statusText}`);
    }
    
    const data = await response.json();
    const reply = data.data?.content || '抱歉，我无法回复。';
    
    await sendMessage(conversationId, reply);
    console.log(`[发送回复]: ${reply}`);
  } catch (error) {
    console.error('回复失败:', error);
  }
}

async function sendMessage(conversationId, content) {
  await fetch(`${config.apiUrl}/api/v1/conversations/${conversationId}/messages`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({
      content: content,
      type: 'text'
    })
  });
}

async function main() {
  try {
    await login();
    await connectWebSocket();
    
    process.on('SIGINT', () => {
      console.log('\n正在退出...');
      process.exit(0);
    });
    
    await new Promise(() => {});
  } catch (error) {
    console.error('❌ 启动失败:', error.message);
    process.exit(1);
  }
}

main();
