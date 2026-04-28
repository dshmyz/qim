# QIM AI 深度升级测试指南

## 前置条件

### 1. 配置 AI 服务

在 `qim-server/config.yaml` 中添加 AI 配置，或通过环境变量设置：

```yaml
ai:
  provider: "openai"  # 或 "baidu", "alibaba", "tencent", "bytedance", "anthropic"
  openai:
    api_key: "your-api-key-here"
    model: "gpt-3.5-turbo"
    base_url: "https://api.openai.com/v1"
```

**环境变量方式（推荐）：**
```bash
export AI_OPENAI_API_KEY="your-api-key-here"
export AI_OPENAI_MODEL="gpt-3.5-turbo"
```

## 测试步骤

### 1. 启动服务器

```bash
cd qim-server
go run main.go
```

启动后你应该看到类似输出：
```
定时任务已启动：群聊总结 (每天 22:00)
服务器启动在端口 8080
```

### 2. 测试意图识别

登录系统后，在普通群聊中发送以下类型的消息，观察日志输出：

| 消息类型 | 示例消息 | 预期意图 |
|---------|---------|---------|
| 普通聊天 | "今天天气真好" | `chat` |
| 问题咨询 | "怎么创建新项目？" | `query` |
| 告警 | "服务器挂了，无法访问" | `alert` |
| 待办 | "明天记得提交代码" | `todo` |

**查看日志验证：**
```
[SmartReply] 检测到意图: type=query, confidence=0.70, content=怎么创建新项目？
[SmartReply] 已发送智能回复到会话 1
```

### 3. 测试智能回复

在群聊中发送一个问问题的消息（如"如何设置管理员？"），等待几秒后：

- **预期结果**：AI 会自动回复一条消息到群聊
- **消息发送者**：ID 为 0 的系统/AI 用户
- **消息类型**：`text`

**注意事项：**
- 如果 AI 服务未配置，不会有智能回复
- 只有普通群聊会触发智能回复，机器人会话由现有逻辑处理
- 回复是异步的，不会阻塞正常消息发送

### 4. 测试运维面板接口

```bash
# 获取 token（先登录获取 JWT）
TOKEN="your-jwt-token-here"

# 调用运维面板接口
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/v1/ai/ops/dashboard
```

**预期响应：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "ai_configured": true,
    "provider": "openai",
    "tools": [
      {"name": "server_monitor", "description": "...", "parameters": {...}},
      {"name": "log_analyzer", "description": "...", "parameters": {...}},
      ...
    ],
    "stats": {
      "total_bots": 0,
      "active_bots": 0,
      "total_messages": 0,
      "ai_messages": 0
    }
  }
}
```

### 5. 测试群聊总结功能

**手动触发测试**（修改定时任务时间用于测试）：

在 `main.go` 中临时改为每分钟执行：
```go
go runDailyJob("22:00", func() {
    summaryJob.GenerateDailySummaries()
})
```

改为：
```go
go func() {
    time.Sleep(30 * time.Second) // 30秒后执行一次
    summaryJob.GenerateDailySummaries()
}()
```

**查看日志：**
```
[GroupSummary] 开始为 3 个群生成每日总结
[GroupSummary] 群 1 (技术交流群) 总结已生成
```

**验证系统消息**：
在数据库中检查 `system_messages` 表，应该有新的总结消息记录。

## 常见问题

### Q1: 日志中没有看到意图检测信息
**原因**：AI 服务未配置
**解决**：确认环境变量 `AI_OPENAI_API_KEY` 已设置

### Q2: 消息发送后没有智能回复
**原因**：
1. AI 服务未配置
2. 消息意图被识别为 `chat`（普通聊天不会自动回复）
3. 置信度未达到阈值（query 需要 >= 0.7）

**解决**：发送一个明确的问句类型消息

### Q3: 群聊总结没有生成
**原因**：
1. 群消息少于 5 条
2. AI 服务调用失败

**解决**：在群中多发几条消息，查看日志中的错误信息

## 调试技巧

### 启用详细日志
```bash
export GIN_MODE=debug
go run main.go
```

### 查看数据库中的 AI 消息
```sql
SELECT * FROM messages WHERE sender_id = 0 ORDER BY created_at DESC LIMIT 10;
```

### 查看系统消息（群聊总结）
```sql
SELECT * FROM system_messages WHERE target_type = 'group' ORDER BY created_at DESC LIMIT 5;
```
