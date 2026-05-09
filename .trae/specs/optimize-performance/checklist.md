# Checklist

- [x] 会话列表接口数据库查询次数从 5 次降低到 1-2 次 — 已修改 `GetConversations` 使用批量 IN 查询，移除 Preload 导致的 N+1
- [x] `GetUsersByIDs` 方法使用单次 IN 查询替代循环查询 — 已在 `user_repository.go` 新增 `FindByIDs` 方法
- [x] 消息搜索使用全文索引而非 `LIKE '%keyword%'` — 已添加 MySQL FULLTEXT 和 SQLite FTS5 索引
- [x] WebSocket Hub 广播使用并发方式，不阻塞其他事件 — 已重构 `Hub.Run`，广播异步化 + 并发发送
- [x] AI Bot 消息处理无 goroutine 泄漏 — 已移除未使用的 `responseChan`
- [x] 消息已读标记使用高效的批量写入方式 — 已改为 `INSERT SELECT` 单条语句
- [x] 会话成员缓存支持主动失效机制 — 已有 `UpdateConversationMembers` 调用在 Add/Remove/Exit 中
- [x] 缓存具有 TTL 过期功能 — 已为 `local_cache.go` 添加 TTL 机制
- [x] 启动迁移逻辑不重复执行 — 已添加迁移版本号标记机制
- [x] `randomString` 使用安全的随机数生成方式 — 已改为 `crypto/rand`
- [x] 所有代码编译通过，无错误 — `go build ./...` 通过，exit code 0
- [ ] 单元测试通过（如有）— 当前项目无单元测试，跳过
