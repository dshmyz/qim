# QIM 聊天程序后端技术方案设计

## 1. 项目概述

### 1.1 项目背景

基于前端Vue3 + Electron的QIM即时通讯应用，需要配套的Go语言后端服务，支持用户认证、即时消息、群组管理、文件存储等功能。

### 1.2 前端功能点分析

根据前端代码分析，需支持以下核心功能：

- 用户认证（登录、密码加密）
- 单聊/群聊
- 消息类型：文本、图片、文件
- 消息撤回、删除、复制
- 组织架构管理
- 群聊创建、成员管理、管理员设置
- 文件上传/下载/管理
- 笔记管理（Markdown支持）
- 消息搜索、历史消息管理
- 用户资料管理、在线状态
- 免打扰设置

***

## 2. 技术选型

### 2.1 核心技术栈

| 组件        | 选型                | 说明                 |
| --------- | ----------------- | ------------------ |
| Web框架     | Gin               | 高性能HTTP框架          |
| WebSocket | Gorilla WebSocket | 成熟稳定的WebSocket库    |
| ORM       | GORM              | 易用的Go ORM库         |
| 数据库       | PostgreSQL        | 关系型数据库，支持JSON、全文搜索 |
| 缓存        | Redis             | 会话管理、消息队列、在线状态     |
| 对象存储      | MinIO             | 自建对象存储，兼容S3 API    |
| 消息队列      | Redis Stream      | 消息持久化、离线消息同步       |
| 认证        | JWT               | 无状态认证              |
| 配置管理      | Viper             | 多格式配置支持            |
| 日志        | Zap               | 高性能日志库             |

### 2.2 技术架构图

```
┌─────────────┐
│   前端客户端 │
└──────┬──────┘
       │
   HTTP/WS
       │
┌──────▼─────────────────┐
│     API Gateway (Gin)   │
├─────────────────────────┤
│  认证中间件  │ 限流中间件 │
└──────┬─────────────────┘
       │
   ┌───┴──────────┐
   │              │
┌──▼───┐     ┌───▼────┐
│HTTP  │     │WebSocket│
│API   │     │服务    │
└──┬───┘     └───┬────┘
   │             │
   └──────┬──────┘
          │
   ┌──────▼───────┐
   │  业务服务层   │
   ├───────────────┤
   │ • 用户服务    │
   │ • 消息服务    │
   │ • 群组服务    │
   │ • 文件服务    │
   │ • 组织服务    │
   │ • 笔记服务    │
   └──────┬───────┘
          │
   ┌──────▼───────┐
   │  数据访问层   │ (GORM)
   └──────┬───────┘
          │
   ┌──────┴───────┐
   │              │
┌──▼───┐      ┌───▼────┐
│PostgreSQL│  │  Redis  │
└───────┘      └────────┘
       │              │
   ┌───▼──────────┐   │
   │   MinIO     │◄──┘
   │ (对象存储)  │
   └─────────────┘
```

***

## 3. 数据库设计

### 3.1 表结构设计

#### 3.1.1 用户表 (users)

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nickname VARCHAR(100),
    avatar VARCHAR(500),
    signature TEXT,
    phone VARCHAR(20),
    email VARCHAR(100),
    status VARCHAR(20) DEFAULT 'offline', -- online/offline/busy
    last_active_at TIMESTAMP,
    ip VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_email ON users(email);
```

#### 3.1.2 组织架构表 (departments)

```sql
CREATE TABLE departments (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    parent_id BIGINT REFERENCES departments(id),
    level INT NOT NULL,
    path VARCHAR(500), -- 完整路径，如 1/2/3
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_departments_parent ON departments(parent_id);
CREATE INDEX idx_departments_path ON departments(path);
```

#### 3.1.3 员工-部门关联表 (department\_employees)

```sql
CREATE TABLE department_employees (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    department_id BIGINT NOT NULL REFERENCES departments(id),
    position VARCHAR(100), -- 职位
    is_primary BOOLEAN DEFAULT true, -- 是否主部门
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, department_id)
);

CREATE INDEX idx_dept_emp_user ON department_employees(user_id);
CREATE INDEX idx_dept_emp_dept ON department_employees(department_id);
```

#### 3.1.4 会话表 (conversations)

```sql
CREATE TABLE conversations (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(20) NOT NULL, -- single/group
    name VARCHAR(200), -- 群聊名称
    avatar VARCHAR(500),
    creator_id BIGINT REFERENCES users(id),
    last_message_id BIGINT,
    last_message_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_conv_type ON conversations(type);
CREATE INDEX idx_conv_last_msg ON conversations(last_message_at DESC);
```

#### 3.1.5 会话成员表 (conversation\_members)

```sql
CREATE TABLE conversation_members (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    role VARCHAR(20) DEFAULT 'member', -- owner/admin/member
    unread_count INT DEFAULT 0,
    muted BOOLEAN DEFAULT false,
    last_read_at TIMESTAMP,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(conversation_id, user_id)
);

CREATE INDEX idx_conv_member_conv ON conversation_members(conversation_id);
CREATE INDEX idx_conv_member_user ON conversation_members(user_id);
```

#### 3.1.6 消息表 (messages)

```sql
CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id BIGINT NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL, -- text/image/file
    content TEXT NOT NULL,
    file_id BIGINT REFERENCES files(id),
    is_recalled BOOLEAN DEFAULT false,
    recalled_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_msg_conv ON messages(conversation_id);
CREATE INDEX idx_msg_sender ON messages(sender_id);
CREATE INDEX idx_msg_created ON messages(created_at DESC);
CREATE INDEX idx_msg_content_gin ON messages USING GIN (to_tsvector('chinese', content)); -- 全文搜索
```

#### 3.1.7 文件表 (files)

```sql
CREATE TABLE files (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255),
    size BIGINT NOT NULL,
    mime_type VARCHAR(100),
    storage_path VARCHAR(500) NOT NULL, -- MinIO路径
    checksum VARCHAR(64), -- SHA256
    folder_id BIGINT REFERENCES folders(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_files_user ON files(user_id);
CREATE INDEX idx_files_folder ON files(folder_id);
```

#### 3.1.8 文件夹表 (folders)

```sql
CREATE TABLE folders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    parent_id BIGINT REFERENCES folders(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_folders_user ON folders(user_id);
```

#### 3.1.9 笔记表 (notes)

```sql
CREATE TABLE notes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL, -- Markdown内容
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_notes_user ON notes(user_id);
CREATE INDEX idx_notes_updated ON notes(updated_at DESC);
```

#### 3.1.10 应用表 (apps)

```sql
CREATE TABLE apps (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    icon VARCHAR(200),
    url VARCHAR(500),
    category VARCHAR(50),
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### 3.1.11 会话记录表 (conversation\_sessions)

```sql
-- 用于记录用户的会话列表排序、置顶等
CREATE TABLE conversation_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    conversation_id BIGINT NOT NULL REFERENCES conversations(id),
    is_pinned BOOLEAN DEFAULT false,
    pinned_at TIMESTAMP,
    last_visited_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, conversation_id)
);

CREATE INDEX idx_session_user ON conversation_sessions(user_id);
```

### 3.2 Redis数据结构设计

#### 3.2.1 用户在线状态

```
Key: user:online:{user_id}
Type: String
Value: timestamp (最后活跃时间)
TTL: 5分钟 (心跳更新)
```

#### 3.2.2 用户WebSocket连接

```
Key: user:ws:{user_id}
Type: Set
Value: WebSocket连接ID列表
```

#### 3.2.3 JWT黑名单

```
Key: jwt:blacklist:{token_id}
Type: String
Value: 1
TTL: JWT过期时间
```

#### 3.2.4 消息队列 (Redis Stream)

```
Key: stream:messages
Type: Stream
用于消息持久化和离线消息同步
```

#### 3.2.5 会话消息缓存

```
Key: conv:msgs:{conv_id}:{page}
Type: List
Value: 消息ID列表
TTL: 1小时
```

***

## 4. 目录结构设计

```
qim-server/
├── cmd/
│   └── server/
│       └── main.go              # 入口文件
├── configs/
│   └── config.yaml              # 配置文件
├── internal/
│   ├── api/
│   │   ├── handler/             # HTTP处理器
│   │   │   ├── auth.go
│   │   │   ├── user.go
│   │   │   ├── message.go
│   │   │   ├── conversation.go
│   │   │   ├── group.go
│   │   │   ├── file.go
│   │   │   ├── organization.go
│   │   │   ├── note.go
│   │   │   └── app.go
│   │   ├── middleware/          # 中间件
│   │   │   ├── auth.go
│   │   │   ├── cors.go
│   │   │   ├── rate_limit.go
│   │   │   ├── logger.go
│   │   │   └── recovery.go
│   │   └── router/              # 路由
│   │       └── router.go
│   ├── ws/                      # WebSocket服务
│   │   ├── server.go
│   │   ├── client.go
│   │   ├── hub.go
│   │   └── handler.go
│   ├── service/                 # 业务逻辑层
│   │   ├── auth_service.go
│   │   ├── user_service.go
│   │   ├── message_service.go
│   │   ├── conversation_service.go
│   │   ├── group_service.go
│   │   ├── file_service.go
│   │   ├── organization_service.go
│   │   ├── note_service.go
│   │   └── app_service.go
│   ├── repository/              # 数据访问层
│   │   ├── user_repo.go
│   │   ├── message_repo.go
│   │   ├── conversation_repo.go
│   │   ├── file_repo.go
│   │   ├── org_repo.go
│   │   └── note_repo.go
│   ├── model/                   # 数据模型
│   │   ├── user.go
│   │   ├── message.go
│   │   ├── conversation.go
│   │   ├── file.go
│   │   ├── organization.go
│   │   └── note.go
│   └── pkg/
│       ├── cache/                # 缓存
│       │   └── redis.go
│       ├── config/               # 配置
│       │   └── config.go
│       ├── database/             # 数据库
│       │   └── postgres.go
│       ├── storage/              # 对象存储
│       │   └── minio.go
│       ├── logger/               # 日志
│       │   └── logger.go
│       ├── jwt/                  # JWT
│       │   └── jwt.go
│       ├── queue/                # 消息队列
│       │   └── stream.go
│       └── utils/                # 工具
│           ├── password.go
│           ├── snowflake.go      # ID生成
│           └── validator.go
├── pkg/                          # 公共包
│   ├── errors/
│   │   └── errors.go
│   └── response/
│       └── response.go
├── scripts/                      # 脚本
│   ├── init.sql
│   └── migrate.go
├── go.mod
├── go.sum
└── README.md
```

***

## 5. API接口设计

### 5.1 认证相关

#### 5.1.1 登录

```
POST /api/v1/auth/login
Request:
{
  "username": "string",
  "password": "string"
}
Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "jwt_token",
    "user": {
      "id": 1,
      "username": "xxx",
      "nickname": "xxx",
      "avatar": "xxx"
    }
  }
}
```

#### 5.1.2 登出

```
POST /api/v1/auth/logout
Headers: Authorization: Bearer {token}
```

#### 5.1.3 刷新Token

```
POST /api/v1/auth/refresh
Headers: Authorization: Bearer {token}
Response:
{
  "code": 0,
  "data": {
    "token": "new_jwt_token"
  }
}
```

### 5.2 用户相关

#### 5.2.1 获取当前用户信息

```
GET /api/v1/users/me
Headers: Authorization: Bearer {token}
```

#### 5.2.2 更新用户资料

```
PUT /api/v1/users/me
Headers: Authorization: Bearer {token}
Request:
{
  "nickname": "string",
  "signature": "string",
  "avatar": "string"
}
```

#### 5.2.3 获取用户信息

```
GET /api/v1/users/{user_id}
Headers: Authorization: Bearer {token}
```

#### 5.2.4 修改密码

```
PUT /api/v1/users/password
Headers: Authorization: Bearer {token}
Request:
{
  "old_password": "string",
  "new_password": "string"
}
```

### 5.3 组织架构相关

#### 5.3.1 获取组织架构树

```
GET /api/v1/organization/tree
Headers: Authorization: Bearer {token}
Response:
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "name": "总公司",
      "subDepartments": [
        {
          "id": 2,
          "name": "技术部",
          "subDepartments": [
            {
              "id": 3,
              "name": "前端开发",
              "subDepartments": [],
              "employees": [
                {
                  "id": 101,
                  "name": "张三",
                  "nickname": "小张",
                  "position": "前端工程师",
                  "avatar": "https://...",
                  "phone": "13800138000",
                  "email": "zhangsan@company.com",
                  "signature": "代码改变世界",
                  "status": "online"
                }
              ]
            }
          ],
          "employees": []
        }
      ],
      "employees": []
    }
  ]
}
```

### 5.4 会话相关

#### 5.4.1 获取会话列表

```
GET /api/v1/conversations
Headers: Authorization: Bearer {token}
Query:
  - page: 页码
  - page_size: 每页数量
```

#### 5.4.2 创建单聊会话

```
POST /api/v1/conversations/single
Headers: Authorization: Bearer {token}
Request:
{
  "user_id": 123
}
```

#### 5.4.3 创建群聊

```
POST /api/v1/conversations/group
Headers: Authorization: Bearer {token}
Request:
{
  "name": "群聊名称",
  "avatar": "头像URL",
  "member_ids": [1, 2, 3]
}
```

#### 5.4.4 获取会话详情

```
GET /api/v1/conversations/{conv_id}
Headers: Authorization: Bearer {token}
```

#### 5.4.5 会话置顶/取消置顶

```
PUT /api/v1/conversations/{conv_id}/pin
Headers: Authorization: Bearer {token}
Request:
{
  "is_pinned": true
}
```

#### 5.4.6 设置免打扰

```
PUT /api/v1/conversations/{conv_id}/mute
Headers: Authorization: Bearer {token}
Request:
{
  "muted": true
}
```

### 5.5 消息相关

#### 5.5.1 获取历史消息

```
GET /api/v1/conversations/{conv_id}/messages
Headers: Authorization: Bearer {token}
Query:
  - before_msg_id: 起始消息ID（用于分页）
  - limit: 数量
  - type: 消息类型过滤
```

#### 5.5.2 搜索消息

```
GET /api/v1/messages/search
Headers: Authorization: Bearer {token}
Query:
  - keyword: 关键词
  - conv_id: 会话ID（可选）
  - start_date: 开始日期
  - end_date: 结束日期
  - type: 消息类型
  - page: 页码
  - page_size: 每页数量
```

#### 5.5.3 撤回消息

```
PUT /api/v1/messages/{msg_id}/recall
Headers: Authorization: Bearer {token}
```

#### 5.5.4 删除消息

```
DELETE /api/v1/messages/{msg_id}
Headers: Authorization: Bearer {token}
```

### 5.6 群组相关

#### 5.6.1 获取群成员

```
GET /api/v1/groups/{conv_id}/members
Headers: Authorization: Bearer {token}
```

#### 5.6.2 添加群成员

```
POST /api/v1/groups/{conv_id}/members
Headers: Authorization: Bearer {token}
Request:
{
  "member_ids": [1, 2, 3]
}
```

#### 5.6.3 移除群成员

```
DELETE /api/v1/groups/{conv_id}/members/{user_id}
Headers: Authorization: Bearer {token}
```

#### 5.6.4 设置管理员

```
PUT /api/v1/groups/{conv_id}/members/{user_id}/role
Headers: Authorization: Bearer {token}
Request:
{
  "role": "admin"
}
```

#### 5.6.5 退出群聊

```
POST /api/v1/groups/{conv_id}/leave
Headers: Authorization: Bearer {token}
```

#### 5.6.6 更新群信息

```
PUT /api/v1/groups/{conv_id}
Headers: Authorization: Bearer {token}
Request:
{
  "name": "新名称",
  "avatar": "新头像"
}
```

### 5.7 文件相关

#### 5.7.1 上传文件

```
POST /api/v1/files/upload
Headers: Authorization: Bearer {token}
Content-Type: multipart/form-data
Form:
  - file: 文件
  - folder_id: 文件夹ID（可选）
Response:
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "filename.txt",
    "url": "https://..."
  }
}
```

#### 5.7.2 获取文件列表

```
GET /api/v1/files
Headers: Authorization: Bearer {token}
Query:
  - folder_id: 文件夹ID
  - page: 页码
  - page_size: 每页数量
```

#### 5.7.3 下载文件

```
GET /api/v1/files/{file_id}/download
Headers: Authorization: Bearer {token}
Response: 文件流
```

#### 5.7.4 删除文件

```
DELETE /api/v1/files/{file_id}
Headers: Authorization: Bearer {token}
```

#### 5.7.5 创建文件夹

```
POST /api/v1/folders
Headers: Authorization: Bearer {token}
Request:
{
  "name": "文件夹名称",
  "parent_id": 1
}
```

#### 5.7.6 获取文件夹树

```
GET /api/v1/folders/tree
Headers: Authorization: Bearer {token}
```

### 5.8 笔记相关

#### 5.8.1 获取笔记列表

```
GET /api/v1/notes
Headers: Authorization: Bearer {token}
Query:
  - page: 页码
  - page_size: 每页数量
```

#### 5.8.2 获取笔记详情

```
GET /api/v1/notes/{note_id}
Headers: Authorization: Bearer {token}
```

#### 5.8.3 创建笔记

```
POST /api/v1/notes
Headers: Authorization: Bearer {token}
Request:
{
  "title": "标题",
  "content": "Markdown内容"
}
```

#### 5.8.4 更新笔记

```
PUT /api/v1/notes/{note_id}
Headers: Authorization: Bearer {token}
Request:
{
  "title": "新标题",
  "content": "新内容"
}
```

#### 5.8.5 删除笔记

```
DELETE /api/v1/notes/{note_id}
Headers: Authorization: Bearer {token}
```

### 5.9 应用相关

#### 5.9.1 获取应用列表

```
GET /api/v1/apps
Headers: Authorization: Bearer {token}
```

***

## 6. WebSocket协议设计

### 6.1 连接建立

```
GET /ws?token={jwt_token}
Upgrade: websocket
Connection: Upgrade
```

### 6.2 消息格式

所有WebSocket消息采用JSON格式：

```json
{
  "type": "message_type",
  "data": {},
  "request_id": "uuid" // 可选，用于请求-响应模式
}
```

### 6.3 消息类型

#### 6.3.1 客户端发送消息

| 类型            | 说明   | data结构                                      |
| ------------- | ---- | ------------------------------------------- |
| heartbeat     | 心跳包  | {}                                          |
| send\_message | 发送消息 | {conversation\_id, type, content, file\_id} |
| typing        | 输入状态 | {conversation\_id, is\_typing}              |
| read\_message | 已读消息 | {conversation\_id, message\_id}             |

#### 6.3.2 服务端推送消息

| 类型                    | 说明     | data结构                                   |
| --------------------- | ------ | ---------------------------------------- |
| new\_message          | 新消息    | Message对象                                |
| message\_recalled     | 消息撤回   | {message\_id, conversation\_id}          |
| message\_deleted      | 消息删除   | {message\_id, conversation\_id}          |
| user\_status          | 用户在线状态 | {user\_id, status}                       |
| typing\_status        | 对方输入中  | {conversation\_id, user\_id, is\_typing} |
| conversation\_updated | 会话更新   | Conversation对象                           |
| group\_member\_joined | 群成员加入  | {conversation\_id, members}              |
| group\_member\_left   | 群成员退出  | {conversation\_id, user\_id}             |
| ack                   | 消息确认   | {request\_id, message\_id}               |
| error                 | 错误     | {code, message}                          |

### 6.4 心跳机制

- 客户端每30秒发送一次心跳
- 服务端5分钟未收到心跳则断开连接
- 心跳包同时更新用户在线状态

### 6.5 消息可靠性保证

1. 发送消息时服务端返回ack
2. 离线消息通过Redis Stream持久化
3. 用户上线时同步离线消息（基于last\_read\_at）
4. 消息ID使用Snowflake算法保证唯一性

***

## 7. 核心业务逻辑设计

### 7.1 消息发送流程

```
1. 客户端通过WebSocket发送send_message
2. 服务端验证权限
3. 保存消息到数据库
4. 写入Redis Stream
5. 推送消息给在线用户
6. 更新会话last_message
7. 增加未读数
8. 返回ack给发送者
```

### 7.2 群聊消息流程

```
1. 发送者发送消息
2. 验证是否为群成员
3. 保存消息
4. 获取所有群成员
5. 通过WebSocket推送给在线成员
6. 离线成员消息存入Stream
7. 更新所有成员的未读数
```

### 7.3 离线消息同步

```
1. 用户建立WebSocket连接
2. 查询用户所有会话的last_read_at
3. 从Redis Stream读取该时间之后的消息
4. 批量推送给用户
5. 用户确认后更新last_read_at
```

### 7.4 文件上传流程

```
1. 客户端请求上传凭证
2. 服务端生成预签名URL
3. 客户端直接上传到MinIO
4. 上传完成后回调服务端
5. 服务端保存文件记录
6. 返回文件信息
```

***

## 8. 安全性设计

### 8.1 认证与授权

- JWT Token认证，有效期2小时
- Refresh Token机制，有效期7天
- Token黑名单机制
- 接口权限控制（群管理仅群主/管理员）

### 8.2 密码安全

- bcrypt加密存储（cost=12）
- 密码强度校验
- 登录失败次数限制（5次锁定15分钟）

### 8.3 接口安全

- HTTPS加密传输
- 请求签名验证（可选）
- 接口限流（基于Redis）
- CORS配置
- SQL注入防护（GORM参数化查询）
- XSS防护（消息内容转义）

### 8.4 文件安全

- 文件类型白名单
- 文件大小限制
- 病毒扫描（可选，集成ClamAV）
- 私有bucket，签名URL访问

### 8.5 数据隐私

- 消息端到端加密（可选，高级功能）
- 敏感数据脱敏
- 访问日志记录

***

## 9. 性能与扩展性设计

### 9.1 缓存策略

- 用户信息缓存（Redis，TTL 30分钟）
- 会话列表缓存
- 热点消息缓存
- 在线状态使用Redis

### 9.2 数据库优化

- 合理索引设计
- 消息表分表（按时间或conversation\_id）
- 读写分离
- 连接池配置

### 9.3 水平扩展

- 无状态设计，支持多实例部署
- WebSocket Hub支持集群（Redis PubSub）
- 负载均衡（Nginx）

### 9.4 消息队列

- Redis Stream处理离线消息
- 异步处理耗时任务（如消息推送、通知）

***

## 10. 监控与运维

### 10.1 日志

- Zap结构化日志
- 日志分级（debug/info/warn/error）
- 访问日志、错误日志分离

### 10.2 监控

- Prometheus指标暴露
- 健康检查接口
- 业务指标（消息量、在线用户数等）

### 10.3 部署

- Docker容器化
- Docker Compose编排
- 配置文件外部化

***

## 11. 配置文件示例 (config.yaml)

```yaml
server:
  port: 8080
  mode: debug # debug/release

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: qim
  sslmode: disable
  max_open_conns: 100
  max_idle_conns: 10

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 100

minio:
  endpoint: localhost:9000
  access_key: minioadmin
  secret_key: minioadmin
  bucket: qim-files
  use_ssl: false

jwt:
  secret: your-secret-key-change-in-production
  expire_hours: 2
  refresh_expire_days: 7

log:
  level: info
  filename: logs/qim.log
  max_size: 100
  max_backups: 10
  max_age: 30
```

***

## 12. 补充功能详细设计

### 12.1 消息转发功能

#### 12.1.1 功能概述

支持将单条消息转发到一个或多个会话（单聊或群聊），可同时转发文字、图片、文件等多种类型消息。

#### 12.1.2 数据库表设计

**消息转发记录表 (message\_forwards)**

```sql
CREATE TABLE message_forwards (
    id BIGSERIAL PRIMARY KEY,
    original_message_id BIGINT NOT NULL REFERENCES messages(id),
    original_conversation_id BIGINT NOT NULL REFERENCES conversations(id),
    forwarder_id BIGINT NOT NULL REFERENCES users(id),
    target_conversation_id BIGINT NOT NULL REFERENCES conversations(id),
    new_message_id BIGINT REFERENCES messages(id),
    forward_comment TEXT, -- 转发时的附言
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_forward_original ON message_forwards(original_message_id);
CREATE INDEX idx_forward_target ON message_forwards(target_conversation_id);
CREATE INDEX idx_forward_forwarder ON message_forwards(forwarder_id);
```

#### 12.1.3 API接口设计

**转发消息**

```
POST /api/v1/messages/{msg_id}/forward
Headers: Authorization: Bearer {token}
Request:
{
  "target_conversation_ids": [1, 2, 3],
  "comment": "转发附言"
}
Response:
{
  "code": 0,
  "data": {
    "success_count": 3,
    "failed_count": 0,
    "results": [
      {
        "conversation_id": 1,
        "new_message_id": 1001,
        "status": "success"
      }
    ]
  }
}
```

#### 12.1.4 WebSocket消息类型

| 类型                 | 说明     | data结构                                                                     |
| ------------------ | ------ | -------------------------------------------------------------------------- |
| message\_forwarded | 消息转发通知 | {original\_message\_id, target\_conversation\_id, new\_message, forwarder} |

#### 12.1.5 业务流程

```
1. 用户选择要转发的消息
2. 选择目标会话（支持多选）
3. 可选添加转发附言
4. 服务端验证消息权限
5. 为每个目标会话创建新消息
6. 记录转发关系
7. 推送转发通知给相关用户
8. 返回转发结果
```

***

### 12.2 @提及功能

#### 12.2.1 功能概述

在群聊消息中@成员，被@的成员会收到特别通知，支持@所有人、@多个成员。

#### 12.2.2 数据库表设计

**消息提及记录表 (message\_mentions)**

```sql
CREATE TABLE message_mentions (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id),
    mentioned_user_id BIGINT NOT NULL REFERENCES users(id),
    mention_type VARCHAR(20) DEFAULT 'user', -- user/all
    is_read BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mention_message ON message_mentions(message_id);
CREATE INDEX idx_mention_user ON message_mentions(mentioned_user_id);
CREATE INDEX idx_mention_unread ON message_mentions(mentioned_user_id, is_read) WHERE is_read = false;
```

**消息表扩展**

```sql
ALTER TABLE messages ADD COLUMN has_mentions BOOLEAN DEFAULT false;
ALTER TABLE messages ADD COLUMN mention_user_ids BIGINT[]; -- 被@的用户ID数组
```

#### 12.2.3 API接口设计

**获取未读@消息**

```
GET /api/v1/mentions/unread
Headers: Authorization: Bearer {token}
Query:
  - page: 页码
  - page_size: 每页数量
```

**标记@消息已读**

```
PUT /api/v1/mentions/{mention_id}/read
Headers: Authorization: Bearer {token}
```

**批量标记已读**

```
PUT /api/v1/mentions/read-all
Headers: Authorization: Bearer {token}
Request:
{
  "conversation_id": 1  // 可选，指定会话
}
```

#### 12.2.4 WebSocket消息类型

| 类型        | 说明   | data结构                                                          |
| --------- | ---- | --------------------------------------------------------------- |
| mentioned | 被@通知 | {message\_id, conversation\_id, sender, content, mention\_type} |

#### 12.2.5 消息格式约定

```
@用户ID[用户名] 消息内容
示例: "Hello @1[张三] @2[李四]，大家好！"
```

#### 12.2.6 业务流程

```
1. 用户在输入框@成员
2. 前端解析@标记，提取用户ID
3. 发送消息时携带mention_user_ids
4. 服务端保存消息并创建mention记录
5. 推送@通知给被@用户
6. 被@用户收到特殊提醒（角标、通知栏）
7. 用户查看后标记已读
```

***

### 12.3 表情包/贴纸功能

#### 12.3.1 功能概述

支持系统表情包、自定义表情包、贴纸的管理和发送。

#### 12.3.2 数据库表设计

**表情包表 (sticker\_packs)**

```sql
CREATE TABLE sticker_packs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    cover_url VARCHAR(500),
    description TEXT,
    author VARCHAR(100),
    is_system BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sticker_pack_system ON sticker_packs(is_system) WHERE is_system = true;
```

**贴纸表 (stickers)**

```sql
CREATE TABLE stickers (
    id BIGSERIAL PRIMARY KEY,
    pack_id BIGINT NOT NULL REFERENCES sticker_packs(id) ON DELETE CASCADE,
    name VARCHAR(100),
    image_url VARCHAR(500) NOT NULL,
    thumbnail_url VARCHAR(500),
    emoji_code VARCHAR(20), -- 关联的emoji
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sticker_pack ON stickers(pack_id);
```

**用户表情包收藏表 (user\_sticker\_packs)**

```sql
CREATE TABLE user_sticker_packs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    pack_id BIGINT NOT NULL REFERENCES sticker_packs(id),
    sort_order INT DEFAULT 0,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, pack_id)
);

CREATE INDEX idx_user_sticker_user ON user_sticker_packs(user_id);
```

**自定义表情包表 (custom\_emojis)**

```sql
CREATE TABLE custom_emojis (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    name VARCHAR(50) NOT NULL,
    image_url VARCHAR(500) NOT NULL,
    is_public BOOLEAN DEFAULT false,
    usage_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_custom_emoji_user ON custom_emojis(user_id);
```

#### 12.3.3 API接口设计

**获取表情包列表**

```
GET /api/v1/sticker-packs
Headers: Authorization: Bearer {token}
Query:
  - type: all/system/user
```

**获取表情包贴纸**

```
GET /api/v1/sticker-packs/{pack_id}/stickers
Headers: Authorization: Bearer {token}
```

**添加/移除表情包**

```
POST /api/v1/sticker-packs/{pack_id}/toggle
Headers: Authorization: Bearer {token}
```

**获取我的表情包**

```
GET /api/v1/sticker-packs/my
Headers: Authorization: Bearer {token}
```

**上传自定义表情**

```
POST /api/v1/custom-emojis
Headers: Authorization: Bearer {token}
Content-Type: multipart/form-data
Form:
  - name: 表情名称
  - image: 图片文件
```

**获取自定义表情列表**

```
GET /api/v1/custom-emojis
Headers: Authorization: Bearer {token}
```

#### 12.3.4 WebSocket消息扩展

消息类型增加：

```
{
  "type": "sticker",
  "content": {
    "sticker_id": 1,
    "pack_id": 1,
    "image_url": "..."
  }
}
{
  "type": "custom_emoji",
  "content": {
    "emoji_id": 1,
    "name": "偷笑",
    "image_url": "..."
  }
}
```

#### 12.3.5 业务流程

```
1. 系统初始化默认表情包
2. 用户浏览表情包商店
3. 用户添加表情包到收藏
4. 用户选择表情包/贴纸发送
5. 服务端记录使用次数
6. 支持用户上传自定义表情
```

***

### 12.4 消息已读回执功能

#### 12.4.1 功能概述

显示消息的已读状态，单聊显示对方是否已读，群聊显示哪些成员已读。

#### 12.4.2 数据库表设计

**消息已读记录表 (message\_read\_receipts)**

```sql
CREATE TABLE message_read_receipts (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    read_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(message_id, user_id)
);

CREATE INDEX idx_read_receipt_message ON message_read_receipts(message_id);
CREATE INDEX idx_read_receipt_user ON message_read_receipts(user_id);
CREATE INDEX idx_read_receipt_conv ON message_read_receipts(conversation_id);
```

**会话成员表扩展**

```sql
ALTER TABLE conversation_members ADD COLUMN last_read_message_id BIGINT REFERENCES messages(id);
```

#### 12.4.3 API接口设计

**获取消息已读情况**

```
GET /api/v1/messages/{msg_id}/read-receipts
Headers: Authorization: Bearer {token}
Response:
{
  "code": 0,
  "data": {
    "total_members": 10,
    "read_count": 5,
    "read_users": [
      {
        "user_id": 1,
        "name": "张三",
        "avatar": "...",
        "read_at": "2026-04-12T10:00:00Z"
      }
    ],
    "unread_users": [...]
  }
}
```

#### 12.4.4 WebSocket消息类型

| 类型                 | 说明     | data结构                                                |
| ------------------ | ------ | ----------------------------------------------------- |
| message\_read      | 消息已读通知 | {message\_id, conversation\_id, user\_id, read\_at}   |
| conversation\_read | 会话已读同步 | {conversation\_id, user\_id, last\_read\_message\_id} |

#### 12.4.5 业务流程

**单聊已读回执**

```
1. 用户A发送消息给用户B
2. 用户B打开会话并查看消息
3. 前端发送read_message到WebSocket
4. 服务端记录read_receipt
5. 推送message_read给用户A
6. 用户A看到消息变为"已读"
```

**群聊已读回执**

```
1. 用户在群聊发送消息
2. 其他成员查看消息时触发read_message
3. 服务端批量记录已读状态
4. 推送已读通知给发送者
5. 发送者可点击查看具体已读成员列表
```

#### 12.4.6 批量同步优化

为避免频繁通知，采用以下策略：

- 单聊：实时推送
- 群聊：合并通知（每5秒推送一次，或累积10个已读后推送）
- 用户进入会话时：批量同步该会话所有未读消息的已读状态

***

### 12.5 消息引用功能

#### 12.5.1 功能概述

支持回复/引用历史消息，可以看到引用的消息上下文，支持引用文本、图片、文件等各类消息。

#### 12.5.2 数据库表设计

**消息表扩展**

```sql
ALTER TABLE messages ADD COLUMN quoted_message_id BIGINT REFERENCES messages(id);
ALTER TABLE messages ADD COLUMN is_quoted BOOLEAN DEFAULT false;
```

**消息引用关系表 (message\_quotes)**

```sql
CREATE TABLE message_quotes (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    quoted_message_id BIGINT NOT NULL REFERENCES messages(id),
    conversation_id BIGINT NOT NULL REFERENCES conversations(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_quote_message ON message_quotes(message_id);
CREATE INDEX idx_quote_quoted ON message_quotes(quoted_message_id);
CREATE INDEX idx_quote_conv ON message_quotes(conversation_id);
```

#### 12.5.3 消息对象扩展

引用消息在返回时会包含引用的消息信息：

```json
{
  "id": 1001,
  "conversation_id": 1,
  "sender_id": 2,
  "type": "text",
  "content": "说得对！",
  "is_self": false,
  "created_at": "2026-04-12T10:30:00Z",
  "quoted_message": {
    "id": 999,
    "sender_id": 1,
    "type": "text",
    "content": "今天天气真好",
    "is_deleted": false,
    "is_recalled": false,
    "sender": {
      "id": 1,
      "name": "张三",
      "avatar": "..."
    }
  }
}
```

#### 12.5.4 API接口设计

**发送引用消息**

```
POST /api/v1/conversations/{conv_id}/messages
Headers: Authorization: Bearer {token}
Request:
{
  "type": "text",
  "content": "引用回复内容",
  "quoted_message_id": 999
}
Response:
{
  "code": 0,
  "data": {
    "id": 1001,
    "content": "引用回复内容",
    "quoted_message": {...}
  }
}
```

**获取消息引用链**

```
GET /api/v1/messages/{msg_id}/quote-chain
Headers: Authorization: Bearer {token}
Response:
{
  "code": 0,
  "data": {
    "messages": [
      { "id": 1001, "content": "回复3", "quoted_message_id": 1000 },
      { "id": 1000, "content": "回复2", "quoted_message_id": 999 },
      { "id": 999, "content": "原始消息", "quoted_message_id": null }
    ]
  }
}
```

#### 12.5.5 WebSocket消息类型

引用消息通过标准的 `new_message` 类型推送，消息数据中包含 `quoted_message` 字段。

#### 12.5.6 业务流程

```
1. 用户长按/右键点击消息，选择"引用"
2. 前端显示引用消息预览
3. 用户输入回复内容
4. 发送消息时携带 quoted_message_id
5. 服务端验证引用消息存在且在同一会话中
6. 保存引用关系到 message_quotes 表
7. 标记原消息 is_quoted = true
8. 返回包含引用信息的完整消息
9. 推送给会话所有成员
```

#### 12.5.7 引用消息的特殊处理

- **引用消息被删除/撤回**：引用时仍显示标记"消息已删除"或"消息已撤回"
- **跨会话引用**：不支持，只能引用同一会话的消息
- **多层引用**：支持展示引用链，但最多显示3层（避免过深嵌套）
- **性能优化**：引用消息数据在消息列表查询时预加载（JOIN查询）

***

### 12.6 系统消息推送功能

#### 12.6.1 功能概述

服务端主动向用户或群组推送系统消息，包括但不限于：

- 系统公告
- 群组事件通知（成员加入/退出、管理员变更等）
- 安全通知（登录异常、密码修改等）
- 运营活动通知
- 系统维护通知

#### 12.6.2 数据库表设计

**系统消息表 (system\_messages)**

```sql
CREATE TABLE system_messages (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(50) NOT NULL, -- announcement/group_event/security/activity/maintenance
    category VARCHAR(50), -- 消息分类
    title VARCHAR(200),
    content TEXT NOT NULL,
    extra JSONB, -- 扩展字段，存储结构化数据
    priority INT DEFAULT 0, -- 优先级，数字越大优先级越高
    sender_type VARCHAR(20) DEFAULT 'system', -- system/admin
    sender_id BIGINT REFERENCES users(id),
    is_global BOOLEAN DEFAULT false, -- 是否全局广播
    target_type VARCHAR(20), -- user/conversation/department/all
    target_id BIGINT, -- 目标ID
    publish_time TIMESTAMP, -- 发布时间（可选，用于定时发布）
    expire_time TIMESTAMP, -- 过期时间
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sys_msg_type ON system_messages(type);
CREATE INDEX idx_sys_msg_target ON system_messages(target_type, target_id);
CREATE INDEX idx_sys_msg_global ON system_messages(is_global) WHERE is_global = true;
CREATE INDEX idx_sys_msg_publish ON system_messages(publish_time);
```

**系统消息接收表 (system\_message\_recipients)**

```sql
CREATE TABLE system_message_recipients (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES system_messages(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    is_read BOOLEAN DEFAULT false,
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(message_id, user_id)
);

CREATE INDEX idx_sys_recipient_msg ON system_message_recipients(message_id);
CREATE INDEX idx_sys_recipient_user ON system_message_recipients(user_id);
CREATE INDEX idx_sys_recipient_unread ON system_message_recipients(user_id, is_read) WHERE is_read = false;
```

**消息表扩展 - 支持系统消息类型**

```sql
ALTER TABLE messages ADD COLUMN is_system BOOLEAN DEFAULT false;
ALTER TABLE messages ADD COLUMN system_message_id BIGINT REFERENCES system_messages(id);
```

#### 12.6.3 系统消息类型枚举

| 类型               | 说明   | 场景                  |
| ---------------- | ---- | ------------------- |
| announcement     | 系统公告 | 全局通知、重要公告           |
| group\_event     | 群组事件 | 成员加入/退出、群信息变更、管理员变更 |
| security         | 安全通知 | 异地登录、密码修改、设备登录      |
| activity         | 活动通知 | 运营活动、节日活动           |
| maintenance      | 维护通知 | 系统维护、功能升级           |
| friend\_request  | 好友请求 | 收到好友申请              |
| friend\_accepted | 好友通过 | 好友申请已通过             |

#### 12.6.4 API接口设计

**管理端 - 创建系统消息**

```
POST /api/v1/admin/system-messages
Headers: Authorization: Bearer {admin_token}
Request:
{
  "type": "announcement",
  "title": "系统升级通知",
  "content": "系统将于今晚22:00进行升级维护",
  "extra": {
    "link": "https://..."
  },
  "priority": 10,
  "is_global": true,
  "target_type": "all",
  "publish_time": "2026-04-12T22:00:00Z",
  "expire_time": "2026-04-13T22:00:00Z"
}
Response:
{
  "code": 0,
  "data": {
    "id": 1,
    "status": "published"
  }
}
```

**获取系统消息列表**

```
GET /api/v1/system-messages
Headers: Authorization: Bearer {token}
Query:
  - type: 消息类型（可选）
  - is_read: 是否已读（可选）
  - page: 页码
  - page_size: 每页数量
```

**获取系统消息详情**

```
GET /api/v1/system-messages/{msg_id}
Headers: Authorization: Bearer {token}
```

**标记系统消息已读**

```
PUT /api/v1/system-messages/{msg_id}/read
Headers: Authorization: Bearer {token}
```

**批量标记已读**

```
PUT /api/v1/system-messages/read-all
Headers: Authorization: Bearer {token}
Request:
{
  "type": "announcement"  // 可选，按类型标记
}
```

**删除系统消息（仅管理端）**

```
DELETE /api/v1/admin/system-messages/{msg_id}
Headers: Authorization: Bearer {admin_token}
```

#### 12.6.5 WebSocket消息类型

| 类型                    | 说明       | data结构                  |
| --------------------- | -------- | ----------------------- |
| system\_message       | 系统消息推送   | SystemMessage对象         |
| system\_message\_read | 系统消息已读确认 | {message\_id, user\_id} |

系统消息对象示例：

```json
{
  "id": 1,
  "type": "announcement",
  "title": "系统升级通知",
  "content": "系统将于今晚22:00进行升级维护",
  "extra": {
    "link": "https://..."
  },
  "priority": 10,
  "is_global": true,
  "created_at": "2026-04-12T10:00:00Z"
}
```

#### 12.6.6 业务流程

**全局公告推送**

```
1. 管理员创建系统公告，设置 is_global=true
2. 系统定时任务检查 publish_time
3. 到达发布时间后，获取所有在线用户
4. 通过WebSocket推送消息给在线用户
5. 为所有用户创建 recipient 记录
6. 记录消息到 messages 表（可选，作为会话消息）
```

**群组事件推送**

```
1. 触发群组事件（如成员加入）
2. 创建系统消息，target_type=conversation
3. 获取群组成员列表
4. 推送给在线成员
5. 为离线成员创建 recipient 记录
6. 成员上线时同步未读系统消息
```

**定时消息发布**

```
1. 使用 cron 定时任务，每分钟检查一次
2. 查询 publish_time <= now 且未发布的消息
3. 执行推送逻辑
4. 标记消息为已发布
```

#### 12.6.7 特殊处理

- **消息优先级**：高优先级消息（如安全通知）会置顶显示
- **消息过期**：超过 expire\_time 的消息不再显示
- **未读数统计**：系统消息未读数独立统计，在UI上单独展示
- **全局消息**：is\_global=true 的消息，新注册用户也会收到
- **撤回机制**：管理员可撤回已发布的系统消息

***

## 13. 其他补充功能建议

根据前端设计，还可以考虑以下功能：

1. **语音/视频通话**
   - WebRTC集成
   - 信令服务
2. **朋友圈/动态**
   - 类似微信朋友圈功能
3. **机器人支持**
   - 群机器人
   - 自动回复
4. **多端同步**
   - 消息多端同步
   - 在线设备管理

***

## 15. 总结

本技术方案基于Go语言生态，采用Gin + GORM + PostgreSQL + Redis + MinIO的技术栈，完整覆盖了前端QIM应用的所有功能需求，并在安全性、性能、扩展性等方面做了充分考虑。

方案特点：

- 模块化设计，便于维护和扩展
- WebSocket实时通信，保证消息即时性
- Redis Stream保证消息可靠性
- 完善的权限控制和安全机制
- 支持水平扩展，适应高并发场景

