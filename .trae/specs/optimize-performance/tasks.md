# Tasks

## Task 1: 修复会话列表查询 N+1 问题
- [x] Task 1.1: 分析 `GetConversations` 中的 5 轮数据库查询，确定可以合并的查询
- [x] Task 1.2: 使用批量 IN 查询或 JOIN 重写会话列表查询逻辑
- [x] Task 1.3: 验证查询结果与原逻辑一致
- [x] Task 1.4: 测试会话列表接口性能

## Task 2: 修复 GetUsersByIDs N+1 查询问题
- [x] Task 2.1: 修改 `user_service.go` 中的 `GetUsersByIDs` 方法
- [x] Task 2.2: 使用 `WHERE id IN ?` 批量查询替代循环查询
- [x] Task 2.3: 确保缓存逻辑正常工作

## Task 3: 添加数据库索引优化
- [x] Task 3.1: 创建索引迁移脚本
- [x] Task 3.2: 添加 `messages(conversation_id, created_at)` 复合索引
- [x] Task 3.3: 添加消息全文搜索索引（MySQL FULLTEXT / SQLite FTS5）
- [x] Task 3.4: 添加 `groups(name)` 索引
- [x] Task 3.5: 添加 `notifications(user_id, read, created_at)` 复合索引
- [x] Task 3.6: 更新 DDL SQL 文件

## Task 4: 优化 WebSocket Hub 广播并发
- [x] Task 4.1: 重构 `Hub.Run` 方法，将广播逻辑异步化
- [x] Task 4.2: 实现 worker pool 并行发送消息
- [x] Task 4.3: 确保广播不阻塞注册/注销等其他事件
- [x] Task 4.4: 测试高并发场景下的广播性能

## Task 5: 修复 AI Bot goroutine 泄漏
- [x] Task 5.1: 分析 `handleBotMessage` 中的 channel 使用问题
- [x] Task 5.2: 移除未使用的 `responseChan` 或改为带缓冲 channel
- [x] Task 5.3: 确保 AI 流式响应正常处理

## Task 6: 优化消息已读标记批量写入
- [x] Task 6.1: 分析 `MarkAsRead` 和 `handleReadMessage` 中的写入逻辑
- [x] Task 6.2: 使用原生 SQL 优化批量写入
- [x] Task 6.3: 测试大规模已读标记性能

## Task 7: 实现会话成员缓存主动失效
- [x] Task 7.1: 在 `AddMemberToGroup` 中添加缓存失效逻辑
- [x] Task 7.2: 在 `RemoveMemberFromGroup` 中添加缓存失效逻辑
- [x] Task 7.3: 在 `ExitGroup` 中添加缓存失效逻辑
- [x] Task 7.4: 验证新成员立即可收到消息

## Task 8: 完善缓存策略
- [x] Task 8.1: 为 `local_cache.go` 添加 TTL 过期机制
- [x] Task 8.2: 在 `UpdateUser` 中更新用户缓存
- [x] Task 8.3: 添加 Bot 配置缓存
- [x] Task 8.4: 添加 SystemConfig 缓存

## Task 9: 优化启动迁移逻辑
- [x] Task 9.1: 创建迁移版本号标记机制
- [x] Task 9.2: 修改 `migrateAIConfigs` 等函数，避免重复执行
- [x] Task 9.3: 测试多次启动不重复执行迁移

## Task 10: 优化 randomString 函数
- [x] Task 10.1: 使用 `crypto/rand` 或 `math/rand` 替代 `time.Now().UnixNano()` 取模
- [x] Task 10.2: 验证随机字符串质量

# Task Dependencies
- Task 1 依赖 Task 3（索引优化）
- Task 4 可独立并行执行
- Task 5 可独立并行执行
- Task 6 可独立并行执行
- Task 7 依赖 Task 4（需要先理解 Hub 的缓存机制）
- Task 8 可独立并行执行
- Task 9 可独立并行执行
- Task 10 可独立并行执行
