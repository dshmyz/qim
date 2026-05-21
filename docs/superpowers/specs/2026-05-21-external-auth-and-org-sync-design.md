# 外部认证与组织架构同步功能设计

## 1. 概述

### 1.1 背景
QIM需要支持对接外部认证系统（LDAP、OAuth、CAS等），并实现组织架构的自动同步，以满足企业统一身份管理和组织架构管理的需求。

### 1.2 目标
- 支持多种认证方式：本地认证、LDAP、OAuth2.0、CAS
- 认证方式可配置，支持按优先级自动尝试
- 支持组织架构自动同步（定时、手动、实时）
- 外部用户首次登录自动创建本地账号
- 提供管理界面进行配置和监控

### 1.3 适用场景
- 企业已有统一认证系统（LDAP、AD、CAS）
- 需要对接企业微信、钉钉等平台
- 需要统一管理组织架构
- 桌面客户端应用

## 2. 整体架构

### 2.1 架构设计

采用插件化认证架构，核心组件如下：

```
qim-server/
├── auth/                    # 认证模块
│   ├── provider/           # 认证提供者
│   │   ├── interface.go    # AuthProvider接口
│   │   ├── local.go        # 本地认证
│   │   ├── ldap.go         # LDAP认证
│   │   ├── oauth.go        # OAuth2.0认证
│   │   └── cas.go          # CAS认证
│   ├── chain.go            # 认证链管理
│   ├── mapper.go           # 用户信息映射
│   └── config.go           # 认证配置
├── sync/                   # 同步模块
│   ├── syncer/             # 同步器
│   │   ├── interface.go    # OrgSyncer接口
│   │   ├── ldap.go         # LDAP同步
│   │   └── api.go          # HTTP API同步
│   ├── scheduler.go        # 定时调度器
│   └── webhook.go          # Webhook处理器
└── model/
    └── auth_config.go      # 认证配置模型
```

### 2.2 核心接口

```go
// AuthProvider 认证提供者接口
type AuthProvider interface {
    Name() string
    Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error)
    IsEnabled() bool
    Priority() int
}

// OrgSyncer 组织架构同步器接口
type OrgSyncer interface {
    Name() string
    Sync(ctx context.Context) (*SyncResult, error)
    Validate() error
}
```

## 3. 认证流程设计

### 3.1 认证方式分类

**直接认证方式**（可在后端依次尝试）：
- 本地用户名密码认证
- LDAP认证
- 外部HTTP服务认证

**重定向认证方式**（需要用户主动选择）：
- OAuth2.0（企业微信、钉钉等）
- CAS

### 3.2 用户名密码登录流程

```
用户输入用户名密码
    ↓
前端 POST /api/auth/login
    ↓
后端认证链管理器
    ↓
按优先级遍历直接认证Provider
    ↓
Provider 1 (LDAP) → 失败 → Provider 2 (外部HTTP) → 失败 → Provider 3 (本地)
    ↓                                           ↓                    ↓
  返回成功                                  返回成功             返回成功/失败
    ↓
用户信息映射 → 创建/更新本地用户 → 生成JWT Token → 返回客户端
```

### 3.3 OAuth/CAS登录流程（桌面应用）

```
1. 用户点击"企业微信登录"
   ↓
2. 桌面应用启动临时HTTP服务器（随机端口，如12345）
   ↓
3. POST /api/auth/oauth/wechat/authorize
   Request: { callback_port: 12345 }
   Response: { redirect_url: "https://open.weixin.qq.com/...&redirect_uri=http://localhost:12345/callback" }
   ↓
4. 打开系统浏览器，跳转到OAuth授权页面
   ↓
5. 用户在浏览器中完成认证
   ↓
6. OAuth服务重定向到 http://localhost:12345/callback?code=xxx
   ↓
7. 桌面应用本地服务器接收callback
   ↓
8. POST /api/auth/oauth/wechat/callback
   Request: { code: "xxx", state: "xxx" }
   Response: { token: "jwt_token", user: {...} }
   ↓
9. 桌面应用保存token，完成登录
```

### 3.4 用户信息映射

外部系统用户信息映射到本地用户：

```yaml
attribute_mapping:
  username: uid          # LDAP的uid字段映射为username
  nickname: cn           # LDAP的cn字段映射为nickname
  email: mail            # LDAP的mail字段映射为email
  phone: telephoneNumber # LDAP的telephoneNumber字段映射为phone
```

### 3.5 用户创建策略

外部用户首次登录时：
1. 检查本地是否存在该用户（通过external_user_mappings表）
2. 如果不存在，根据外部用户信息创建本地用户
3. 创建用户映射关系
4. 后续登录时更新用户信息

## 4. 组织架构同步设计

### 4.1 同步方式

**定时同步**：
- 通过cron表达式配置同步时间
- 如：`0 2 * * *` 表示每天凌晨2点同步

**手动同步**：
- 管理员在后台手动触发同步
- 适用于配置变更后立即生效

**实时同步**：
- 通过Webhook接收外部系统的变更通知
- 支持事件：user_created、user_updated、user_deleted、department_created等

### 4.2 同步流程

```
同步任务启动
    ↓
连接外部系统（LDAP/API）
    ↓
获取外部组织架构数据
    ↓
获取本地组织架构数据
    ↓
对比差异
    ↓
执行同步操作：
  - 创建新部门/用户
  - 更新现有部门/用户
  - 删除孤立数据（可配置）
    ↓
记录同步日志
    ↓
更新同步状态
```

### 4.3 同步策略配置

```yaml
sync_strategy:
  create_if_not_exists: true   # 不存在则创建
  update_if_exists: true       # 存在则更新
  delete_orphaned: false       # 是否删除孤立数据
  batch_size: 100              # 批量处理大小
```

## 5. 数据模型

### 5.1 认证配置表

```sql
CREATE TABLE auth_providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(50) NOT NULL UNIQUE,
    type VARCHAR(20) NOT NULL,              -- direct/redirect
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 100,
    config TEXT,                            -- JSON配置
    display_name VARCHAR(100),
    icon VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 5.2 外部用户映射表

```sql
CREATE TABLE external_user_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    provider_name VARCHAR(50) NOT NULL,
    external_user_id VARCHAR(200) NOT NULL,
    external_username VARCHAR(200),
    external_data TEXT,
    last_sync_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider_name, external_user_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### 5.3 组织架构同步配置表

```sql
CREATE TABLE org_sync_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(50) NOT NULL UNIQUE,
    enabled BOOLEAN DEFAULT true,
    sync_type VARCHAR(20) NOT NULL,         -- ldap/api
    schedule VARCHAR(100),                  -- cron表达式
    config TEXT,
    last_sync_at TIMESTAMP,
    last_sync_status VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 5.4 组织架构同步日志表

```sql
CREATE TABLE org_sync_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    config_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,            -- running/success/failed
    started_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP,
    stats TEXT,                             -- JSON统计信息
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (config_id) REFERENCES org_sync_configs(id)
);
```

## 6. API接口

### 6.1 认证接口

```go
// 用户名密码登录
POST /api/auth/login
Request: { username, password, version }
Response: { token, user, provider }

// 获取重定向认证方式列表
GET /api/auth/providers
Response: [{ name, display_name, icon, enabled }]

// 发起OAuth认证
POST /api/auth/oauth/:provider/authorize
Request: { callback_port }
Response: { redirect_url, state }

// OAuth回调处理
POST /api/auth/oauth/:provider/callback
Request: { code, state }
Response: { token, user }

// CAS回调处理
GET /api/auth/cas/callback?ticket=xxx&state=xxx
```

### 6.2 组织架构同步接口

```go
// 获取同步配置列表
GET /api/org/sync/configs

// 创建/更新同步配置
POST /api/org/sync/configs
PUT /api/org/sync/configs/:id

// 手动触发同步
POST /api/org/sync/trigger/:configId

// 获取同步日志
GET /api/org/sync/logs?config_id=1&page=1&page_size=20

// Webhook接收
POST /api/org/sync/webhook/:provider
```

### 6.3 管理接口

```go
// 认证提供者管理
GET /api/admin/auth/providers
POST /api/admin/auth/providers
PUT /api/admin/auth/providers/:id
POST /api/admin/auth/providers/:id/test
PUT /api/admin/auth/providers/priority
```

## 7. 前端界面

### 7.1 登录页面

- 用户名密码登录表单
- 第三方登录按钮区域（动态显示启用的重定向认证方式）

### 7.2 管理后台

**认证配置管理页面**：
- 直接认证方式列表（可拖拽调整优先级）
- 重定向认证方式列表
- 配置对话框（LDAP、OAuth、CAS等）
- 测试连接功能

**组织架构同步管理页面**：
- 同步配置列表
- 同步日志查看
- 手动触发同步
- 配置编辑

## 8. 安全考虑

### 8.1 敏感信息保护
- LDAP Bind密码、OAuth Client Secret等加密存储
- 使用环境变量或密钥管理服务
- 日志中不输出敏感信息

### 8.2 防CSRF
- OAuth/CAS使用state参数防CSRF攻击
- State存储在Redis中，设置短过期时间

### 8.3 Token安全
- JWT Token设置合理过期时间
- 使用Refresh Token机制
- 桌面应用使用安全存储

### 8.4 权限控制
- 认证配置管理需要管理员权限
- 组织架构同步需要管理员权限
- Webhook接口验证签名

## 9. 性能优化

### 9.1 连接池
- LDAP连接池复用连接
- HTTP客户端连接池

### 9.2 批量处理
- 组织架构同步使用批量插入/更新
- 大量数据时分批处理

### 9.3 缓存
- 缓存认证提供者配置
- 缓存用户信息映射关系

## 10. 监控和日志

### 10.1 认证日志
- 记录认证尝试（成功/失败）
- 包含：用户名、认证方式、IP、耗时等

### 10.2 同步日志
- 记录每次同步的详细信息
- 包含：开始时间、结束时间、统计信息、错误信息等

### 10.3 性能指标
- 认证响应时间
- 同步耗时统计
- 错误率监控

## 11. 实施计划

### 11.1 第一阶段：认证框架
1. 实现AuthProvider接口和认证链管理
2. 实现本地认证Provider
3. 实现LDAP认证Provider
4. 实现用户信息映射
5. 数据库表创建
6. 基础API接口

### 11.2 第二阶段：OAuth/CAS支持
1. 实现OAuth Provider
2. 实现CAS Provider
3. 桌面应用OAuth处理
4. 前端登录页面改造

### 11.3 第三阶段：组织架构同步
1. 实现OrgSyncer接口
2. 实现LDAP同步器
3. 实现定时调度器
4. 实现Webhook处理器
5. 同步日志记录

### 11.4 第四阶段：管理界面
1. 认证配置管理页面
2. 组织架构同步管理页面
3. 配置测试功能
4. 日志查看功能

## 12. 配置示例

### 12.1 LDAP认证配置

```yaml
name: ldap
type: direct
enabled: true
priority: 10
config:
  host: ldap.company.com
  port: 389
  use_ssl: true
  base_dn: "dc=company,dc=com"
  bind_dn: "cn=admin,dc=company,dc=com"
  bind_password: "${LDAP_BIND_PASSWORD}"
  user_filter: "(uid={username})"
  attribute_mapping:
    username: uid
    nickname: cn
    email: mail
    phone: telephoneNumber
```

### 12.2 OAuth配置

```yaml
name: oauth_wechat
type: redirect
enabled: true
display_name: "企业微信登录"
icon: "/icons/wechat.svg"
config:
  client_id: "${WECHAT_CLIENT_ID}"
  client_secret: "${WECHAT_CLIENT_SECRET}"
  authorize_url: "https://open.weixin.qq.com/connect/authorize"
  token_url: "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
  user_info_url: "https://qyapi.weixin.qq.com/cgi-bin/user/get"
  scope: "snsapi_base"
```

### 12.3 组织架构同步配置

```yaml
name: "LDAP组织架构同步"
enabled: true
sync_type: ldap
schedule: "0 2 * * *"
config:
  host: ldap.company.com
  port: 389
  base_dn: "ou=departments,dc=company,dc=com"
  department_filter: "(objectClass=organizationalUnit)"
  user_filter: "(objectClass=inetOrgPerson)"
  attribute_mapping:
    department_name: ou
    department_code: departmentNumber
    user_username: uid
    user_nickname: cn
  sync_strategy:
    create_if_not_exists: true
    update_if_exists: true
    delete_orphaned: false
```

## 13. 总结

本设计采用插件化架构，具有良好的扩展性和可维护性。通过认证链机制支持多种认证方式的灵活组合，通过组织架构同步机制实现与外部系统的数据同步。设计充分考虑了安全性、性能和可运维性，能够满足企业级应用的需求。
