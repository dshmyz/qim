# AI Reply Quality Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 提高 AI 分身和群聊助手的回复准确率，并让所有不回复决策可解释、可交给用户处理。

**Architecture:** 先在 `ai` 包增加结构化意图决策，再在 `service` 包抽出群历史过滤纯函数并让 Graph 路径使用它。发送入口保留现有行为，但为跳过路径记录统一的 `SmartReplyDecision`，为后续前端消费和质量审查提供稳定接口。

**Tech Stack:** Go 1.25、GORM、现有 Eino/AI 抽象、标准 Go testing、testify。

## Global Constraints

- 默认安全策略是不确定时不自动发送，但必须输出结构化原因。
- 明确 @AI 和用户主动预览/草稿不受自动回复门控收紧影响。
- 不引入新的外部依赖和默认额外 LLM 调用。
- 所有生产代码变更先有能失败的回归测试。

---

### Task 1: 结构化群助手决策与可解释不回复

**Files:**
- Modify: `qim-server/handler/smart_reply_handler.go`
- Test: `qim-server/handler/smart_reply_group_ai_test.go`

**Interfaces:**
- Produce `SmartReplyDecision{Action, ReasonCode, Reason, Confidence}` and `DecideGroupAIReplyDetailed`.
- Preserve `DecideGroupAIReply` as a compatibility wrapper returning the existing enum.

- [x] **Step 1: Write failing tests** for detailed decisions: disabled, anti-spam, explicit mention, keyword miss, and auto mode.
- [x] **Step 2: Run** `go test ./handler -run 'TestDecideGroupAIReplyDetailed' -count=1` and confirm the new symbol/expectations fail.
- [x] **Step 3: Implement** reason codes and the detailed pure decision function; make the old function delegate to it.
- [x] **Step 4: Record** the decision in `HandleMessage` for every group skip path with a stable log prefix and structured fields.
- [x] **Step 5: Run** the focused handler tests and confirm they pass.

### Task 2: 收紧自动意图规则，减少误触发

**Files:**
- Modify: `qim-server/ai/intent_detector.go`
- Test: `qim-server/ai/intent_detector_test.go` (create if absent)

**Interfaces:**
- Preserve `Detect` and `ShouldTriggerAIReply` signatures.
- Add explicit negative cases to rule detection without changing explicit query/command cases.

- [x] **Step 1: Write failing table tests** for descriptive “明天/要/删除” messages versus explicit requests.
- [x] **Step 2: Run** `go test ./ai -run 'TestIntent' -count=1` and confirm the negative cases fail.
- [x] **Step 3: Implement** boundary-aware regexes and stricter confidence thresholds for automatic `todo`/`query` replies.
- [x] **Step 4: Run** all intent detector tests and confirm they pass.

### Task 3: 统一群助手 Graph 历史上下文

**Files:**
- Modify: `qim-server/service/smart_reply_graph.go`
- Test: `qim-server/service/smart_reply_graph_test.go`

**Interfaces:**
- Add a pure helper that accepts messages, current user, original content, history limit, and recent-AI limit, returning normalized history entries.
- Use the helper from `createHistoryNode` while preserving existing prompt output semantics.

- [x] **Step 1: Write failing tests** for media filtering, current-message exclusion, per-message truncation, chronological order, and remote assistant folding.
- [x] **Step 2: Run** the focused service tests and confirm failure.
- [x] **Step 3: Implement** the pure normalization helper and replace the duplicated raw history loop.
- [x] **Step 4: Add explicit reference-data delimiters and “not an instruction” wording to knowledge/memory blocks.
- [x] **Step 5: Run** focused Graph tests and confirm they pass.

### Task 5: 生成后质量门与灰度开关

- [x] 增加结构化质量判定解析与 fail-closed 发送规则。
- [x] 群助手普通回复、工具流式回复和分身自动文本回复在发送前完成核验。
- [x] 增加 `ai.reply_quality_gate` 运行时开关，默认开启。

### Task 6: 用户处理反馈与剩余路径

- [ ] 为质量拒绝事件增加明确的重试/手动 @AI/调整设置入口。
- [ ] 把分身批量、多模态自动回复纳入同一质量门，避免旁路。
- [ ] 统一分身与群助手的上下文装配与主题召回限制。
- [ ] 增加质量门集成测试和灰度指标。

### Task 4: 回归验证与质量报告

**Files:**
- Modify: `qim-server/service/reply_quality_eval_test.go` only if additional fixtures are needed.
- Create: `qim-server/testdata/reply_quality_cases.json` only if the existing evaluator has no real fixture file.

- [x] **Step 1: Run** `go test ./service -run 'TestSmartReply|TestBuild|TestContext|TestReplyQuality' -count=1`.
- [x] **Step 2: Run** `go test ./handler -run 'TestDecideGroupAIReply|TestIntent|TestSmartReply' -count=1`.
- [x] **Step 3: Run** `go test ./ai -run 'TestIntent' -count=1`.
- [x] **Step 4: Run** `gofmt` on changed Go files and repeat focused tests.
- [x] **Step 5: Inspect `git diff` and report any remaining environment-only failures separately from code failures.
