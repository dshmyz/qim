# CLAUDE.md — QIM 项目开发指南

## 项目概览

**QIM (Quick Instant Messaging)** 是一个面向企业的即时通讯系统，提供完整的 IM 功能（单聊、群聊、讨论组、频道）以及办公应用集成（日历、笔记、任务、文件管理、AI 助手）。包含三个子项目：

| 子项目 | 说明 | 默认端口 |
|-------|------|---------|
| `qim-client/` | Electron 桌面客户端（主应用） | 3000 |
| `qim-admin/` | 管理后台（Web） | 3008 |
| `qim-server/` | 后端 API + WebSocket 服务 | 8080 |

**v1.0.0 已发布**，当前处于 v2.0 迭代阶段。

## 技术栈

### 前端（qim-client / qim-admin）
- **Vue 3** (Composition API) + **TypeScript**
- **Vite 5** 构建
- **Element Plus** UI 组件库（admin 使用，client 也有依赖）
- **Pinia** 状态管理
- **Vue Router** 路由
- **Electron 33** 桌面框架（仅 client）
- 测试：**Vitest 4** + **Playwright**

### 后端（qim-server）
- **Go 1.25** + **Gin** HTTP 框架
- **GORM** ORM（支持 SQLite / MySQL 双数据库）
- **Gorilla WebSocket** 实时通信
- **JWT** 认证（golang-jwt v5）
- **AWS S3 SDK v2** 文件存储
- **Cloudwego Eino** + **CortexDB** AI 能力（多模型支持）
- **Swaggo** API 文档生成
- 数据库默认 SQLite（`qim.db`），可选 MySQL

## 目录结构

```
qim/
├── qim-client/              # Electron 桌面客户端
│   ├── src/
│   │   ├── components/      # Vue 组件（chat/, apps/, layout/, shared/）
│   │   ├── composables/     # 组合式函数
│   │   ├── stores/          # Pinia 状态管理
│   │   ├── types/           # TS 类型定义
│   │   └── utils/           # 工具函数
│   ├── electron/            # Electron 主进程、图标、构建脚本
│   └── vite.config.ts       # Vite 配置 + manualChunks 分包
│
├── qim-admin/               # 管理后台
│   ├── src/
│   │   ├── api/             # API 接口封装
│   │   ├── views/           # 页面视图
│   │   ├── layouts/         # 布局组件
│   │   └── stores/          # 状态管理
│   └── vite.config.ts       # 含 /api 代理到 localhost:8080
│
├── qim-server/              # Go 后端
│   ├── ai/                  # AI 服务（MCP、output filter）
│   ├── app/                 # 应用初始化（路由、DI 容器）
│   ├── cache/               # 本地缓存
│   ├── config/              # 配置管理
│   ├── database/            # 数据库连接
│   ├── di/                  # 依赖注入容器
│   ├── handler/             # HTTP/WebSocket 处理器
│   ├── middleware/           # 中间件（auth、rate limit）
│   ├── model/               # GORM 数据模型
│   ├── pkg/                 # 公共包（errors、pagination、response）
│   ├── repository/          # 数据访问层
│   ├── service/             # 业务逻辑层
│   ├── utils/               # 工具函数
│   ├── ws/                  # WebSocket Hub
│   └── main.go              # 入口
│
├── docs/                    # 项目文档
└── schema.sql               # 旧版数据库结构参考
```

## 关键命令

```bash
# 后端
cd qim-server
go mod download           # 安装依赖
go run main.go            # 启动服务（默认 8080）

# 客户端开发
cd qim-client
npm run dev               # 启动 Vite 开发服务器（3000）
npm run electron:dev      # 启动 Electron + Vite
npm run build             # 构建 Web 资源
npm run electron:build    # 打包 Electron 应用
npm run test              # 运行单元测试
npm run test:e2e          # 运行 E2E 测试

# 管理后台
cd qim-admin
npm run dev               # 启动开发服务器（3008）
npm run build             # 构建
npm run test              # 运行测试
npm run e2e               # E2E 测试

# 后端 Swagger 文档
cd qim-server
swag init                  # 重新生成 API 文档
```

## 编码约定

### 前端
- Vue 3 Composition API + `<script setup>` 语法
- TypeScript 类型必须完整
- 路径别名：`@` → `src/`
- Pinia stores 使用 `defineStore`
- 组件命名 PascalCase，文件名 kebab-case
- Vite manualChunks 按功能模块分包（chat、ai、sticky、notes 等）

### 后端
- Go 标准分层架构：handler → service → repository → model
- DI 容器在 `app/` 中初始化，通过 `app.Get*Service()` 获取
- 统一响应格式：`pkg/response` 封装
- 错误处理：`pkg/errors` 自定义错误类型
- 路由在 `app/routes.go`（或类似文件）集中注册
- Swaggo 注释写在 handler 函数上
- 测试文件与源码同目录 `_test.go` 后缀

## 重要注意事项

1. **数据库**：默认 SQLite（`qim.db` 在项目根目录），改用 MySQL 需修改 `qim-server/config.yaml`
2. **前端服务器地址**：client 默认连接 `http://localhost:8080`，可在登录界面修改
3. **Electron 版本锁定**：electronVersion 固定为 33.0.0
4. **Go 版本**：go.mod 声明 go 1.25.0，请使用匹配版本
5. **AI 多模型**：后端通过 Cloudwego Eino 框架支持 OpenAI/Claude/通义千问/文心一言等多模型
6. **WebSocket**：连接路径 `/ws?token={jwt_token}`，消息格式 `{type, data, request_id}`
7. **双因素认证**：后端验证为模拟实现，非生产级
