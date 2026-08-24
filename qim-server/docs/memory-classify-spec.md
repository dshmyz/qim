# 记忆模块：写入侧反例体系与实测留档

> 适用范围：分身（avatar）与群助手（group）记忆分类，以及反射（reflection）阶段。
> 相关代码：`service/memory_verdict.go`（共享判定提示）、`service/knowledge_km_reflector.go`（反射/分类）、`service/avatar_memory_service.go`、`service/group_memory_service.go`。

## 1. 目标

防止"记忆乱回"：噪音/临时/敏感信息若错误落库，会在未来召回时被当作长期事实注入回复，造成答非所问。本文档约束**什么不该被记**（写入侧），并记录已实测验证的行为。

## 2. 写入侧"不值得记"体系（W1~W4）

### 共享负面清单（`rememberVerdictNegativeClause`）
分身与群共用一段文字，避免两处改得不一致。当前覆盖：

寒暄问候、确认/感谢短句、情绪化表达、日常流水（吃饭、天气、出行等琐碎）、针对某人的临时答复、闲聊式问答（未形成可复用知识）、**一次性/过期即失效的信息**（"今天""这周"等相对时间）、**即时操作指令**（"把文件放到桌面"）、**敏感凭据**（密码、token、密钥、证件号、卡号）、**吐槽与主观宣泄**。

### 重要性联动（W1）
在 `evaluateRemember` 的 JSON 指令中加入联动规则：
> 若判定 importance ≤ 2（偶发或琐碎、记了没长期价值），通常也应 `remember=false`。

目的：堵住"判定记了、却只打 1-2 分"的矛盾，把低价值记忆挡在写入前。

### 正向清单（值得记）
个人偏好、重要决定、项目关键信息、约定事项、群内决定与共识、**答疑形成的可复用知识**（步骤/配置/口径/规范）。
反例兜底：答疑只记"可复用知识"，不记"针对某人的临时答复、闲聊式问答"。

## 3. 实测结论（真实 LLM，deepseek-v4-flash）

### 写入侧（before = 旧提示词，after = 新提示词）

| 类别 | 样本 | before | after | 结论 |
|---|---|---|---|---|
| W2 敏感凭据 | 微信密码 | 记=5（最危险） | 不记=1 | 有效拦截 |
| W3 一次性相对时间 | "今天周三，明天截止" | 记=4 | 不记=1 | 有效 |
| W1 联动 | "会议室订到明早" | 记=3 | 不记=2 | 有效 |
| W4 即时指令 | "把文件放桌面" | 不记 | 不记 | 无回归 |
| W4 吐槽 | "这系统卡死" | 不记 | 不记 | 无回归 |
| 反例（既有） | 临时答复 x2 | 记 3~4 | 不记 1 | 反例纠正误记 |
| 答疑知识 | 接入钉钉/部署流程 | 记 4 | 记 3~4，scope=global/fact | 仍记且可复用 |
| 正向 | 约定/决定/偏好 | 记 | 记 3~5 | 无回归 |

### scope 标注（反射阶段）
私约/承诺 → `conversation`；项目事实/偏好/答疑知识 → `global`。类型（event/fact/preference）标注正确，无泄漏。

## 4. 边界取舍（需产品确认）

- **"这周末想去 XX 山自驾"（一次性周末邀约）**：现判为**不记**（importance=2）。这基于"临时事件，过期即失效"原则，利于防乱回；但若希望"最近邀约/计划"在未来对话仍被记得，此收紧可能过严。
  - 保持现状：不改代码；想放宽：在 `rememberVerdictNegativeClause` 或锚点里调整"一次性"的口径。
- 一次性 event 类内容被判定更严格，是 W1/W3 的预期后果。

## 5. 读出侧现状与待办

- 已有门槛：记忆注入默认分 **0.5**（`memoryRecallThreshold`）；按对话 `scope` 过滤；范围外记忆静默（不旁路放行）。
- **R1 已实现（分身 + 群统一）**：近义/冲突合并判定 `inferMemoryMergeKind`（见 `service/memory_merge.go`）。把原"冲突二分类"升级为三方判定：
  - `conflict` → 用新的更新旧记忆（保留 memoryID）；
  - `duplicate`（近义复述/确认）→ 不新增，仅刷新旧记忆重要度为 max(旧,新)，防近义重复占位/挤占召回 TopK；
  - `new` → 新增一条。
  分身 `saveConsolidatedMemory` 与群 `saveConsolidatedGroupMemory` 共用同一套判定（各自保留可注入的 `SetMergeCheck` 测试缝）。判定失败/ aiService=nil 时退化为新增（安全方向，不误并）。
- 仍待办：**并发写入**还未做同键串行化——高活跃群多条近义确认并发执行时，理论上仍可能同时判定 duplicate 撞进同一时刻。风险窗口很小（首次写入后即可被召回命中），暂可接受，后续如需可在保存路径加 per-group/user 互斥。

## 6. 群记忆补充（P2/P3）

- **P2 已实现（记发言归属）**：群记忆新增 metadata `sender_id`/`sender_name`，由 handler 从 `HandleMessage(msg.SenderID, senderDisplayName(msg))` 经 `maybeRememberGroupMessage`、`ConsolidateGroupMessage` 一路传入 `saveConsolidatedGroupMemory` 落库。便于群助手回答"上次谁拍板/谁接手 X"类追问；冲突/复述更新时保留首提者（不覆写 metadata）。展示名优先级 Nickname→RealName→Username，Sender 未预载时查一次库。
- **P3 不做**：经确认，超短/纯表情/重复噪音已在 handler 的 `looksMemorable` 预筛（≤15 字、`@ai`、寒暄短语、去重字符集≤2 的 `isLowSignalNoise`）挡住，P3 描述的"轻量预过滤"本已存在。剩余"同话题短窗口批量折叠"属较大改动，需先有群消息量级/调用压力证据才值得做。

## 7. 群/分身回复侧的后续强化（P2-consume / 串行化 / R2 / 评测固化）

- **P2-consume 已实现（sender_name 消费进回复）**：群助手 3 条注入链路（handler legacy systemPrompt、`smart_reply_graph.go` prepareInput、recallGroupMemory）原先只拼 `r.Content`，现统一改用共享 `GroupMemoryCtxText(results)`，把每条群记忆按 `• [sender_name] 内容` 注入 prompt，让群助手能回答"上次谁拍板/谁说的"。sender_name 为空退化为纯内容行（兼容旧记忆）。
- **并发串行化已实现**：新增 `memwrite_locker.go` 的按 key 互斥（远端 `avatarWriteMutex`/`groupWriteMutex`），`saveConsolidated*` 与 `Remember` 在短 DB 写段内按 userID/groupID 互斥，防近义/冲突记忆并发读-改-写竞态。不跨 Recall/LLM 调用，避免长锁阻塞。
- **R2 已实现（低重要度不给徽章）**：分身 `avatar_reply_graph.go` buildContext 中 importance≤2 的命中记忆**仍注入正文**（信息不丢），但**不进 Sources（依据徽章）**；群记忆 `memoryResultsToSources` 同步压制 importance≤2 的「知识来源」徽章。importance 缺失/不可解析按"可展示"处理，不压制未知。
- **评测固化已实现**：新增 `memory_merge_test.go`（假 LLM mock，不真调 LLM）轮锁定：三分类 kind 字符串→枚举映射、安全回退（nil/不可解析退化为 new）、重要度合并取 max、`GroupMemoryCtxText` 发言人注入、R2 群徽章压制。与 `knowledge_km_reflector_test.go` 的 scope 传播回归测试一起构成离线护栏。真实 LLM 抽样仍保留在 `MEM_EVAL=1` 门控的 `TestMemoryPromptEval`/`TestMemoryMergeKindEval`。

## 8. 分身记忆注入的召回治理（评审后发现并修复，A/B）

智慧评审分身 `buildContext` 时发现记忆路径与笔记/群知识存在两处不对称，已修复：

- **A（收敛封顶）**：`recall` 内为补偿对话过滤把 TopK 翻倍（3→6），但注入前无收敛，导致 global 记忆多时「相关记忆」可注入 6 条而非标称 3 条。已用 `memoryInjectTopK=3` 常量 + `selectTopByScore` 在过滤/精排后按分数收敛回 3 条。
- **B（补齐相关性精排）**：记忆曾是三类知识来源里唯一无 LLM 相关性二次判定的路径，纯靠 embedding 分数，可能让"高分但与当前意图无关"的记忆直接进 prompt。已把过 0.5 门槛的候选转 `KnowledgeSnippet` 走 `filterSnippetsByReranker` 精排（与笔记/群同款），再收敛回 3。开关 `ai.knowledge_llm_rerank=0` 或 nil reranker 时跳过（纯阈值模式）。
- **成本提示**：B 每轮分身回复对 ≤6 条记忆候选最多做一次相关度判定（受 15s 总预算与开关约束），若线上分身回复延迟上升优先用 `ai.knowledge_llm_rerank=0` 关停。
- **残留说明**：`webhook_sender_test.go` 曾出现 `can't assign requested address` 端口绑定瞬时失败，与记忆无关，单独重跑即通过（环境抖动）。

> 当前版本记录（保留并发串行化尚未覆盖的边界）：`keyedMutex` 只锁写段不跨 Recall，极端窗口内两条并发 consolidation 仍可能对"各自 Recall 到的旧快照"各自判定。风险窗口小（首写后即可被召回命中），如需完全闭环，可在写段内用锁内重新校验目标记忆后再决定插入。

## 9. 方向2（长会话滚动摘要）搁置说明

需求：把"超出 `ai.context_history_limit` 注入窗口的更早消息"压缩成摘要保留，避免早期决定/约定滚出窗口后消失。

经评估**暂不实施**：
- 记忆模块已按相似度召回"跨窗口的要紧事实"，是"任意深度记忆"的对口机制；滚动摘要再做任意深度压损覆盖属重复劳动。
- 摘要是连续性锦上添花而非刚需，收益难量化；实现（无论持久化还是零存储）都增加每回复一次 LLM 折叠调用或一张表，属低置信度投入。
- 会话消息数分布难可靠评估，故不在此刻投入。

若日后确有"长会话早期关键信息丢失"的真实案例，可再评估零存储的"次级窗口折叠"（构造历史时把窗口外紧跟的固定 W 条当场所折，成本恒定有界、读时永远新鲜），仍不推荐持久化滚动摘要。

## 10. AI 回复质量评测闭环（方向1，已落地）

目标：让"回复质量"可持续度量、可回归，成为后续提示词/检索配置 A/B 的度量平台。代码在 `service/reply_quality_eval_test.go`，样例 `testdata/reply_quality_cases.json`，默认跳过（`MEM_EVAL=1` 开启）。

- **`judgeReplyQuality(svc, question, context, reply)`**：可复用打分器，LLM-as-judge 输出忠实度(1-5)/相关度(1-5)/幻觉(bool)/综合分(1-5)。
- **judge 自身稳定性测试** `TestReplyQualityJudgeEval`：5 条优质 vs 劣质对照，实测 5/5，好回复均值 4.5 vs 坏回复均值 1.0，能稳定分辨"忠实基于依据的回答"与"幻觉/跑题回答"。
- **基线/A-B 回路** `TestReplyQualityBaselineEval`：从 JSON 读"问题+依据+回复+tag"，按 tag 聚合各维度均值，并对同问题输出 baseline vs target 的 delta。后续改提示词/检索只需把改动前后真实回复填进 fixture 重跑即可量化对比。
- 样例当前为合成数据演示；真实回复日志按同 schema 填充即可投入日常回归。

## 11. 如何复跑评测

真实 LLM 评测脚本为 `service/memory_prompt_eval_test.go`（非正式测试，默认跳过）：

```bash
cd qim-server
MEM_EVAL=1 go test ./service/ -run TestMemoryPromptEval -count=1 -v
```

注意：会触发 24 条判定 + 7 条 scope 的真实调用（约 3~4 分钟）并产生费用；期间若连续跑多次可能命中供应商 **429 限流**（属正常配额问题，等恢复后重跑即可，与代码无关）。