# 退出群聊「保留只读历史」实现方案

> 状态：待实施（仅方案，未动代码）
> 目标：用户退出群聊后，最近会话和历史消息仍保留、可查看，但标记「已退出」、不能再发言；群成员列表/数量不再包含已退出用户。
> 前置背景：当前退出群聊会物理删除 `conversation_members` 记录，导致刷新后该会话从最近列表彻底消失，与「保留只读历史」诉求冲突。

---

## 一、问题根因

`conversation_members` 这一张表目前同时承担三种语义：

1. **会话归属**：`GetConversations` 靠 `WHERE cm.user_id = ?` 决定「这个会话在我的最近列表里」。
2. **发言/操作权限**：10+ 处 `GetMember` / `IsConversationMember` 校验靠它判断「能不能发消息、能不能操作群」。
3. **群成员构成**：群成员列表、`member_count` 靠它统计「群里有谁」。

退出时执行 `RemoveMember`（物理删除）会一次性抹掉三种语义。「保留只读历史」要求把语义 1 与语义 2/3 拆开：**退出后仍属于我的会话列表（1=是），但不能发言（2=否）、不计入群成员（3=否）。**

---

## 二、核心设计：成员加「退出时间」标记

给 `ConversationMember` 增加一个软退出字段，用「是否为空」区分在群 / 已退出。

### 1. 数据模型变更（不可逆，需迁移）

文件：`qim-server/model/model.go` — `ConversationMember` 结构体

```go
type ConversationMember struct {
    // ... 现有字段 ...
    LeftAt *time.Time `json:"left_at" gorm:"index"` // 非空 = 已退出，仅保留只读历史
}
```

- 用 `*time.Time` 而非 bool：既能判断状态，又能保留退出时间（便于「你于 X 退出群聊」提示、审计）。
- GORM `AutoMigrate` 会自动加列；存量数据 `left_at` 默认 NULL（视为在群），无需数据回填。
- ⚠️ 这是 schema 变更，影响共享数据库，部署前需确认迁移在测试库验证过。

### 2. 退出接口：删除 → 打标记

文件：`qim-server/handler/group_handler.go` `ExitGroup`（约 327 行）

- 现状：`convSvc.RemoveMember(convIDUint, userID)`（物理删除）
- 改为：调用新方法 `convSvc.MarkMemberLeft(convIDUint, userID)`，设置 `left_at = now()`。

新增 service 方法（`conversation_service.go`）：

```go
func (s *ConversationService) MarkMemberLeft(convID, userID uint) error {
    now := time.Now()
    return s.db.Model(&model.ConversationMember{}).
        Where("conversation_id = ? AND user_id = ? AND left_at IS NULL", convID, userID).
        Update("left_at", now).Error
}
```

退出后仍要广播 `group_member_left`（现有逻辑保留），让群里其他人移除该成员。

---

## 三、查询语义改造（关键，逐处梳理）

成员表现在有「在群」和「已退出」两类记录，所有读路径都要明确选哪一类。

### A. 会话列表 —— 要包含已退出（保留只读历史）

文件：`qim-server/handler/conversation_handler.go` `GetConversations`（原生 SQL，约 49–72 行）

- `WHERE cm.user_id = ?` **不加** `left_at IS NULL`，这样已退出会话仍出现在最近列表。
- 但 SQL 的 `SELECT cm.*` 要带出 `left_at`，并在返回的 `ConversationWithPin` 里加 `IsExited bool` 字段（`left_at != nil`），供前端渲染只读态。
- ⚠️ 这是原生 SQL，不是 GORM 链式，改 `SELECT` 列时注意 `Scan` 目标结构体字段对齐，避免影响置顶/未读/隐藏等既有逻辑。

### B. 群成员列表 / 数量 —— 要排除已退出

需要加 `left_at IS NULL` 过滤的点：

1. `conversation_handler.go` `GetConversations` 内批量查群成员（约 171–172 行 `db.Where("conversation_id IN ?", groupConvIDs).Find(&groupMembers)`）→ 加 `AND left_at IS NULL`。
2. `conversation_handler.go` `GetConversation` 详情接口的 `Preload("Members")`（约 352 行）→ Preload 加条件：
   ```go
   db.Preload("Members", "left_at IS NULL").Preload("Members.User")
   ```
3. `member_count` 统计处（`group_handler.go` 约 1338 行附近）→ count 加 `left_at IS NULL`。
4. `service/conversation_service.go` `GetConversationMembers` / `GetMembersWithUser` / `GetMembersExcept` / `GetMembersByRoles` → 默认都应只返回在群成员，加 `left_at IS NULL`。
   - ⚠️ 注意这些方法有缓存（`ConversationMemberCache`），改查询语义后要确认缓存 key 不会串味，必要时退出时清缓存。

### C. 发言 / 操作权限 —— 必须排除已退出（安全边界）

这是**最容易漏、漏了最严重**的一类：漏改 = 退群后还能发消息。

- `service/conversation_service.go` `IsConversationMember` → 底层 `convRepo.IsMember` 加 `left_at IS NULL`。
- `GetMember`（约 426 行）→ 加 `left_at IS NULL`，让退出后所有 `GetMember` 校验自动失败（发消息、@提醒、群管理操作等 10+ 调用点统一生效）。
- 影响的调用点（确认全部走 `left_at IS NULL`）：
  - `message_handler.go` 232 / 305 / 376 / 512 / 767：发消息、发提醒的成员校验。
  - `group_handler.go` 64 / 247 / 321 / 391 / 509 / 602 / 811 / 890 / 977 / 1170：群管理类操作校验。
- 做法建议：**只改 `IsMember` 和 `GetMember` 两个底层方法加 `left_at IS NULL`**，上层调用点无需逐个改，天然全部收紧。这样安全边界集中、不易漏。

### D. 重新加入群 —— 复用旧记录而非新建

文件：`qim-server/handler/group_handler.go` `AddMemberToGroup` / 申请加入

- 现状大概率是 `Create(&ConversationMember{...})`。
- 问题：`conversation_members` 上若有 `(conversation_id, user_id)` 唯一约束，已退出用户重新加入时旧记录还在（只是 `left_at` 非空），直接 Create 会唯一键冲突。
- 改为 upsert 语义：若存在 `left_at` 非空的旧记录，则 `UPDATE left_at = NULL`（并重置 role/joined_at 视产品需要）；否则才 Create。

---

## 三-补、退群后消息可见范围 = 截止退群时刻（方案 B）

「保留只读历史」必须定义清楚：退群后能看到哪些消息？

### 现状

- 消息查询 `message_handler.go` `GetMessages`（约 246 行）只按 `ConvID + 分页 + before/after` 拉取，**返回该会话全部历史消息，不做任何时间裁剪**。
- 权限校验（约 232 行）已有兜底：非成员时若「曾收到过至少 1 条消息」（`Limit:1` 探测 `Total>0`）即放行查看——这其实已为「退群后看历史」预留了入口。

### 决策：方案 B —— 历史定格在退出时刻

| 方案 | 能看到的消息 | 取舍 |
|---|---|---|
| A. 全部历史 | 含退群后群里的新消息 | ❌ 退群了还看到后续新消息，不合理 |
| **B. 截止退群时刻（采用）** | 最早 → `left_at` 为止 | ✅ 符合微信/钉钉直觉，退群后定格 |
| C. 入群到退群区间 | `joined_at ≤ created_at ≤ left_at` | 入群前消息看不到，与「群消息全员可见」假设冲突 |

退群后历史定格在退出那一刻，之后群里的新消息看不到；在群成员不受限。

### 实现要点

1. `service` 层 `MessageQuery` 增加上界字段：
   ```go
   type MessageQuery struct {
       // ... 现有字段 ...
       MaxCreatedAt *time.Time // 仅对已退出成员注入：created_at <= MaxCreatedAt
   }
   ```
   底层查询在 `MaxCreatedAt != nil` 时追加 `AND created_at <= ?`。

2. `message_handler.go` `GetMessages`（约 246 行构造 query 处）：
   - 先查请求者的成员记录（含 `left_at`）。
   - 若 `left_at != nil`（已退出）→ `query.MaxCreatedAt = member.LeftAt`。
   - 若在群（`left_at == nil`）→ 不设上界，正常拉全部。
   - 注意：约 232 行那段「非成员探测放行」逻辑在引入 `left_at` 后可简化——已退出成员现在仍是「有记录但 left_at 非空」，直接用成员记录判定即可，不必再靠探测 `Total>0`。

3. ⚠️ 时间戳精度边界：同一秒多条消息用 `created_at <= left_at` 可能切不准。若要精确，可在退群时记录该会话当时的 `last_message_id`，按 `message.id <= left_message_id` 裁剪——但需在 `conversation_members` 再加一字段（如 `LeftMessageID *uint`）。**先按时间戳实现，够用；有精度问题再升级到按消息 ID。**

---

## 四、前端配套（后端就绪后）

1. **只读态渲染**：会话/聊天窗口根据后端返回的 `is_exited`（来自会话列表 B 项）显示「你已退出该群聊」横幅。
2. **禁用输入**：`MessageInput` 在 `is_exited` 时禁用输入框与发送按钮（不依赖后端报错才拦截，提升体验）。
3. **本地状态**：保留之前已改的 `patchConversation(id, { isExited: true })`；不要 `removeConversation`。退出成功后刷新会话列表即可（后端已会返回该会话）。
4. **成员列表**：无需额外处理——后端 B 项已不返回已退出成员。
5. （可选）支持用户主动「移除会话」：走现有 `handleRemove` → DELETE 会话隐藏接口（`conversation_sessions.is_hidden`），与退出语义独立。

---

## 五、测试计划（TDD）

后端（`qim-server`，与源码同目录 `_test.go`）：

1. **退出后会话仍可查**：退出群聊 → `GetConversations` 仍返回该会话且 `is_exited = true`。
2. **退出后不能发言**：退出群聊 → `IsConversationMember` / `GetMember` 返回 false/err → 发消息接口 403。
3. **退出后不计入群成员**：退出群聊 → `GetConversation` 详情的 `members` 不含该用户，`member_count` 减一。
4. **历史消息保留 + 可见范围定格**：退出群聊 → 拉取该会话历史消息接口仍正常返回；且只返回 `created_at <= left_at` 的消息，退群后群里产生的新消息不可见（方案 B）。
5. **重新加入**：已退出用户再加入 → 不报唯一键冲突，`left_at` 清空，恢复可发言。
6. 复用现有 `repository/conversation_repository_test.go` 的建表/建数据模式。

前端（`qim-client`，Vitest）：

7. 会话 `is_exited` 时 `MessageInput` 禁用、显示只读横幅。
8. 退出成功后 `conversations` 仍含该会话且标记 `isExited`（已有类似用例可扩展）。

---

## 六、改动范围与风险小结

| 模块 | 文件（约） | 难度 | 风险 |
|---|---|---|---|
| schema + 迁移 | model.go | 低 | ⚠️ 不可逆、影响共享 DB |
| 退出标记 | group_handler.go, conversation_service.go | 低 | 低 |
| 会话列表保留 | conversation_handler.go（原生 SQL） | 中 | SQL 改动易回归 |
| 成员列表/数量排除 | conversation_handler.go, conversation_service.go | 中 | 缓存语义需同步 |
| 发言权限收紧 | conversation_repository.go（IsMember/GetMember） | 中 | ⚠️ 安全边界，集中改两处 |
| 消息可见范围定格 | message_handler.go, service MessageQuery | 中 | 时间戳精度边界 |
| 重新加入 upsert | group_handler.go | 低 | 唯一键冲突 |
| 前端只读态 | ChatWindow, MessageInput | 中 | — |
| 测试 | 多个 _test.go | 中 | — |

预估：后端 8–12 文件、前端 2–3 文件 + 测试。

**实施顺序建议**（分步可验证，每步先红灯测试）：
1. schema + 退出标记 → 测「退出后会话仍可查」。
2. 收紧 `IsMember`/`GetMember` → 测「退出后不能发言」（安全优先）。
3. 成员列表/数量排除 → 测「不计入群成员」。
4. 消息可见范围定格（`MaxCreatedAt`）→ 测「历史保留且退群后新消息不可见」。
5. 重新加入 upsert → 测「重新加入」。
6. 前端只读态 → 前端用例。

**关键提醒**：第 2 步是安全边界，务必先于「放开会话列表」之外的任何上线；只改 `IsMember` + `GetMember` 两个底层方法即可让所有上层校验收紧，避免逐点遗漏。
