# 性能优化 Spec

## Why
qim-server 在会话列表查询、消息搜索、WebSocket 广播等核心场景存在严重性能瓶颈，包括 N+1 查询问题、goroutine 泄漏、消息搜索全表扫描等，导致用户体验差且高并发场景下系统吞吐量低。

## What Changes
- 修复会话列表查询的 N+1 数据库查询问题，将 5 轮查询合并为 1-2 轮
- 修复 `GetUsersByIDs` 方法的 N+1 查询问题
- 为消息搜索添加数据库全文索引
- 修复 WebSocket Hub 单 goroutine 事件循环导致的广播阻塞问题
- 修复 AI Bot 消息处理中的 goroutine 泄漏问题
- 优化消息已读标记批量写入逻辑
- 实现会话成员缓存主动失效机制
- 添加缺失的数据库索引
- 优化启动迁移逻辑，避免重复执行 DDL
- 完善缓存策略，添加 TTL 和主动失效机制

## Impact
- Affected specs: 性能优化、数据库查询优化、WebSocket 广播优化
- Affected code: 
  - `handler/conversation_handler.go`
  - `service/user_service.go`
  - `service/message_service.go`
  - `ws/ws.go`
  - `cache/local_cache.go`
  - `app/init.go`
  - DDL SQL 文件

## ADDED Requirements

### Requirement: 数据库索引优化
系统 SHALL 为高频查询字段添加合适的数据库索引以提升查询性能。

#### Scenario: 添加消息搜索全文索引
- **WHEN** 系统启动时执行迁移
- **THEN** `messages.content` 字段应有全文索引支持快速搜索

### Requirement: WebSocket 广播并发优化
系统 SHALL 使用并发方式广播消息，避免阻塞 WebSocket 事件循环。

#### Scenario: 广播消息不阻塞其他事件
- **WHEN** Hub 需要向 1000 个用户广播消息
- **THEN** 消息广播应在后台 goroutine 中进行，注册/注销等其他事件不被阻塞

### Requirement: 缓存主动失效机制
系统 SHALL 在数据变更时主动使相关缓存失效，确保数据一致性。

#### Scenario: 群成员变更后缓存失效
- **WHEN** 用户加入或退出群聊
- **THEN** 该会话的成员缓存应立即失效，下次查询时重新加载

## MODIFIED Requirements

### Requirement: 会话列表查询优化
系统 SHALL 使用批量查询替代多次独立查询，将数据库往返次数从 5 次降低到 1-2 次。

**修改前**: `GetConversations` 执行 5 次独立数据库查询
**修改后**: 使用 JOIN 或批量 IN 查询合并为 1-2 次查询

### Requirement: 用户批量查询优化
系统 SHALL 使用单次 IN 查询替代循环查询。

**修改前**: `GetUsersByIDs` 循环调用 `FindByID` 查询每个用户
**修改后**: 使用 `WHERE id IN ?` 单次查询所有用户

### Requirement: 消息已读标记写入优化
系统 SHALL 使用高效的批量写入方式处理已读回执。

**修改前**: 先 Pluck 所有未读消息 ID，再分 500 条/批写入
**修改后**: 使用原生 SQL 一次性批量写入

### Requirement: AI Bot 消息处理修复
系统 SHALL 正确处理 AI 流式响应，避免 goroutine 泄漏。

**修改前**: 创建无缓冲 channel 但无人读取，导致永久阻塞
**修改后**: 移除未使用的 channel 或使用带缓冲 channel

## REMOVED Requirements

### Requirement: 启动时重复迁移逻辑
**Reason**: 每次启动都执行 DDL 操作增加启动时间且有生产风险
**Migration**: 使用版本号标记已完成的迁移，避免重复执行
