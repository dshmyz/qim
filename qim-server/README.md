# QIM Server - 后端服务

基于 Go 语言开发的 QIM 即时通讯后端服务，支持 SQLite 和 MySQL 双数据库。

## 技术栈

- Web 框架: Gin
- WebSocket: Gorilla WebSocket
- ORM: GORM
- 数据库: SQLite（默认）/ MySQL
- 认证: JWT (golang-jwt v5)
- 文件存储: AWS S3 SDK v2
- AI: Cloudwego Eino + CortexDB（多模型支持）
- API 文档: Swaggo

## 功能特性

- 用户注册/登录（OAuth/OIDC/CAS 多Provider）
- 组织架构管理（部门树、员工关联）
- 单聊/群聊/讨论组/频道
- 消息发送/接收/撤回/已读
- WebSocket 实时通信 + 在线状态
- 文件上传（分片/断点续传）/文件夹管理
- AI 助手（多模型、语音回复、智能搜索）
- 通知系统（提醒/事件/任务）
- 审批工作流
- 小程序支持（iframe 沙箱 + postMessage Bridge）
- 多因素认证（TOTP）
- 操作日志审计

## 快速开始

### 安装依赖

```bash
go mod download
```

### 运行服务

```bash
go run main.go
```

服务将在 `http://localhost:8080` 启动

## API 接口

### 认证

#### 注册
```
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "test",
  "password": "123456",
  "nickname": "测试用户"
}
```

#### 登录
```
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "test",
  "password": "123456"
}
```

### 用户

#### 获取当前用户
```
GET /api/v1/users/me
Authorization: Bearer {token}
```

#### 更新用户
```
PUT /api/v1/users/me
Authorization: Bearer {token}
Content-Type: application/json

{
  "nickname": "新昵称",
  "signature": "个性签名"
}
```

### 组织架构

#### 获取组织架构树
```
GET /api/v1/organization/tree
Authorization: Bearer {token}
```

### 会话

#### 获取会话列表
```
GET /api/v1/conversations
Authorization: Bearer {token}
```

#### 创建单聊
```
POST /api/v1/conversations/single
Authorization: Bearer {token}
Content-Type: application/json

{
  "user_id": 2
}
```

#### 创建群聊
```
POST /api/v1/conversations/group
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "群聊名称",
  "member_ids": [2, 3]
}
```

#### 获取会话详情
```
GET /api/v1/conversations/{id}
Authorization: Bearer {token}
```

### 消息

#### 获取历史消息
```
GET /api/v1/conversations/{id}/messages
Authorization: Bearer {token}
```

#### 发送消息
```
POST /api/v1/conversations/{id}/messages
Authorization: Bearer {token}
Content-Type: application/json

{
  "type": "text",
  "content": "你好！"
}
```

## WebSocket

### 连接
```
GET /ws?token={token}
Upgrade: websocket
Connection: Upgrade
```

### 消息格式
```json
{
  "type": "message_type",
  "data": {},
  "request_id": "optional"
}
```

### 客户端发送消息

| 类型 | 说明 |
|------|------|
| heartbeat | 心跳 |
| send_message | 发送消息 |
| read_message | 标记已读 |

### 服务端推送消息

| 类型 | 说明 |
|------|------|
| new_message | 新消息 |

## 初始化数据

首次运行后，可以手动添加一些测试数据：

```sql
-- 添加测试部门
INSERT INTO departments (name, level, path, sort_order) VALUES 
('总公司', 1, '1', 0),
('技术部', 2, '1/2', 1),
('产品部', 2, '1/3', 2);

-- 添加测试用户（密码都是123456）
INSERT INTO users (username, password_hash, nickname, avatar, status, created_at, updated_at) VALUES 
('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhW', '管理员', 'https://api.dicebear.com/7.x/avataaars/svg?seed=admin', 'offline', datetime('now'), datetime('now')),
('zhangsan', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhW', '张三', 'https://api.dicebear.com/7.x/avataaars/svg?seed=zhangsan', 'offline', datetime('now'), datetime('now')),
('lisi', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhW', '李四', 'https://api.dicebear.com/7.x/avataaars/svg?seed=lisi', 'offline', datetime('now'), datetime('now'));

-- 关联用户到部门
INSERT INTO department_employees (user_id, department_id, position, is_primary, created_at) VALUES 
(1, 2, '技术总监', 1, datetime('now')),
(2, 2, '前端工程师', 1, datetime('now')),
(3, 3, '产品经理', 1, datetime('now'));
```

## 目录结构

```
qim-server/
├── main.go              # 入口文件
├── go.mod               # Go 模块
├── go.sum               # 依赖锁定
├── config.example.yaml  # 配置示例
├── config/              # 配置管理
│   └── config.go
├── database/            # 数据库连接初始化
│   └── database.go
├── model/               # GORM 数据模型
│   └── model.go
├── handler/             # HTTP/WebSocket 处理器（~40文件）
│   ├── auth_handler.go
│   ├── user_handler.go
│   ├── message_handler.go
│   ├── conversation_handler.go
│   ├── group_handler.go
│   ├── channel_handler.go
│   ├── file_handler.go
│   ├── ai_handler.go
│   └── ...
├── service/             # 业务逻辑层
│   ├── user_service.go
│   ├── conversation_service.go
│   ├── message_service.go
│   ├── auth_service.go
│   ├── avatar_service.go
│   └── ...
├── repository/          # 数据访问层（通用泛型基类 + 具体repo）
│   ├── base_repository.go
│   ├── interfaces.go
│   ├── user_repository.go
│   └── ...
├── middleware/           # 中间件（auth、rate limit、operation log）
│   ├── auth.go
│   ├── rate_limit.go
│   └── operation_log.go
├── ws/                  # WebSocket Hub + 实时通信
│   ├── ws.go
│   └── realtime.go
├── ai/                  # AI 服务（MCP、output filter、provider factory）
├── app/                 # 应用初始化（路由注册、DI 容器、数据库迁移）
│   ├── routes.go
│   └── init.go
├── di/                  # 依赖注入容器
│   └── container.go
├── cache/               # 本地缓存
├── pkg/                 # 公共包（errors、response、params、pagination、logger）
│   ├── errors/
│   ├── response/
│   ├── params/
│   ├── pagination/
│   └── logger/
├── utils/               # 工具函数（加密等）
├── auth/                # OAuth/OIDC/CAS 认证提供者
├── sync/                # 定时同步任务
├── docs/swagger/        # Swaggo 生成的 API 文档
├── migrations/          # 数据库迁移脚本
└── test/                # 测试辅助数据
```

## 注意事项

这是一个快速实现版本，适合开发和测试使用。生产环境建议：

- 使用PostgreSQL/MySQL代替SQLite
- 添加更多安全措施
- 实现完整的错误处理
- 添加日志和监控
