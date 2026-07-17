# 消息提醒目标系统名称 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让消息提醒成功提示展示管理员配置的目标系统名称，并在旧配置中安全回退为“外部系统”。

**Architecture:** Webhook 配置新增可选 `system_name`，服务端从配置读取后写入成功的 `remind_result` WebSocket 回执。客户端仅消费该回执字段生成提示文本；管理后台负责保存和恢复名称。空名称全链路允许，保持原有提示作为回退。

**Tech Stack:** Go、Gin、GORM、Vue 3、TypeScript、Vitest、Element Plus。

## Global Constraints

- 系统名称由管理员手动填写；不得根据 Webhook URL 推断系统类型。
- 只改变提醒成功回执和配置界面；不得改变 Webhook 请求、HMAC 签名或失败提示。
- 已保存的旧 Webhook JSON 没有 `system_name` 时必须继续可用，客户端显示“提醒已送达外部系统”。

---

## File Structure

- `qim-server/service/webhook_sender.go`：声明并反序列化 `WebhookConfig.SystemName`。
- `qim-server/handler/message_handler.go`：在成功 `remind_result` 中发送目标系统名称。
- `qim-server/handler/message_handler_test.go`：覆盖成功回执的系统名称字段。
- `qim-server/app/init.go`：在新安装的默认 JSON 配置中提供空 `system_name`。
- `qim-admin/src/views/components/MessageRemindWebhookConfig.vue`：让管理员填写、加载和保存系统名称。
- `qim-client/src/views/Main.vue`：在成功提示中使用回执的系统名称并提供回退。
- `qim-client/tests/unit/message-remind-result.test.ts`：静态回归验证客户端的具体名称与空值回退分支。

### Task 1: 服务端配置与成功回执

**Files:**
- Modify: `qim-server/service/webhook_sender.go:50-58`
- Modify: `qim-server/handler/message_handler.go:1023-1030`
- Modify: `qim-server/app/init.go:628-638`
- Modify: `qim-server/handler/message_handler_test.go`

**Interfaces:**
- Produces: `service.WebhookConfig{SystemName string \`json:"system_name"\`}`。
- Produces: 成功 `remind_result.data.system_name`，值为已加载的 `webhookCfg.SystemName`。

- [ ] **Step 1: 写入会失败的成功回执测试**

在 `qim-server/handler/message_handler_test.go` 增加一个覆盖异步成功分支的测试：将 Webhook 配置设为 `SystemName: "企业微信"`，调用提醒接口，等待发送者 WebSocket 收到 `remind_result`，并断言 JSON 的 `data.success` 为 `true`、`data.system_name` 为 `"企业微信"`。

- [ ] **Step 2: 运行测试并确认失败原因正确**

运行：`cd qim-server && go test ./handler -run TestRemindMessage.*SystemName -v`

预期：FAIL，因为成功回执尚未包含 `system_name`。

- [ ] **Step 3: 以最小改动实现配置字段和回执字段**

在 `WebhookConfig` 添加：

```go
SystemName string `json:"system_name"` // 管理员配置的目标系统展示名称
```

在 `seedMessageRemindWebhook` 的默认 JSON 增加：

```json
"system_name":""
```

在成功 WebSocket payload 的 `data` 对象增加：

```go
"system_name": webhookCfg.SystemName,
```

- [ ] **Step 4: 运行服务端回归测试**

运行：`cd qim-server && go test ./handler -run TestRemindMessage.*SystemName -v && go test ./service -run TestSendRemind -v`

预期：PASS，现有 Webhook 请求行为不变。

- [ ] **Step 5: 提交服务端改动**

```bash
git add qim-server/service/webhook_sender.go qim-server/handler/message_handler.go qim-server/handler/message_handler_test.go qim-server/app/init.go
git commit -m "feat(remind): include target system name in receipt"
```

### Task 2: 管理后台配置字段

**Files:**
- Modify: `qim-admin/src/views/components/MessageRemindWebhookConfig.vue:10-29,79-103`

**Interfaces:**
- Consumes: Webhook JSON 的可选字段 `system_name: string`。
- Produces: `message_remind_webhook` JSON 中的 `system_name`，保留旧配置加载行为。

- [ ] **Step 1: 写入会失败的配置字段测试**

在现有管理后台的 Playwright 系统配置测试中增加：进入“消息提醒 Webhook”标签，断言出现标签“系统名称”和占位提示“例如：企业微信、飞书、Slack”，填写“企业微信”并保存后，拦截的 `PUT /v1/system/config` 请求体中 `message_remind_webhook` 解析后含有 `system_name: "企业微信"`。

- [ ] **Step 2: 运行测试并确认失败原因正确**

运行：`cd qim-admin && npm run e2e -- --grep "消息提醒系统名称"`

预期：FAIL，因为配置表单尚无“系统名称”输入框。

- [ ] **Step 3: 以最小改动实现表单字段**

扩展前端接口和默认值：

```ts
interface WebhookConfig {
  system_name: string
  // 保留其余既有字段
}

const config = reactive<WebhookConfig>({
  system_name: '',
  // 保留其余既有默认值
})
```

在“请求地址”前增加：

```vue
<el-form-item label="系统名称">
  <el-input v-model="config.system_name" placeholder="例如：企业微信、飞书、Slack" clearable />
  <span class="desc">（用于提醒成功后的客户端提示）</span>
</el-form-item>
```

- [ ] **Step 4: 运行管理后台回归测试**

运行：`cd qim-admin && npm run e2e -- --grep "消息提醒系统名称" && npm run build`

预期：PASS，构建无 TypeScript 错误。

- [ ] **Step 5: 提交管理后台改动**

```bash
git add qim-admin/src/views/components/MessageRemindWebhookConfig.vue qim-admin/e2e/features.spec.ts
git commit -m "feat(admin): configure reminder target system name"
```

### Task 3: 客户端成功提示与回退

**Files:**
- Modify: `qim-client/src/views/Main.vue:1666-1674`
- Create: `qim-client/tests/unit/message-remind-result.test.ts`

**Interfaces:**
- Consumes: `remind_result.data.system_name?: string`。
- Produces: 有名称时“提醒已送达{系统名称}”；无名称时“提醒已送达外部系统”。

- [ ] **Step 1: 写入会失败的客户端回归测试**

创建 `qim-client/tests/unit/message-remind-result.test.ts`，读取 `Main.vue` 源码并断言成功回执分支同时包含下列逻辑：

```ts
const systemName = data.system_name?.trim() || '外部系统'
showMessage({ message: `提醒已送达${systemName}`, type: 'success' })
```

- [ ] **Step 2: 运行测试并确认失败原因正确**

运行：`cd qim-client && npm run test:unit -- tests/unit/message-remind-result.test.ts`

预期：FAIL，因为当前文本固定为“提醒已送达外部系统”。

- [ ] **Step 3: 以最小改动实现名称回退**

将成功分支改为：

```ts
const systemName = data.system_name?.trim() || '外部系统'
showMessage({ message: `提醒已送达${systemName}`, type: 'success' })
```

- [ ] **Step 4: 运行客户端验证**

运行：`cd qim-client && npm run test:unit -- tests/unit/message-remind-result.test.ts && npm run typecheck`

预期：PASS，具体名称与空值回退均由测试覆盖。

- [ ] **Step 5: 提交客户端改动**

```bash
git add qim-client/src/views/Main.vue qim-client/tests/unit/message-remind-result.test.ts
git commit -m "feat(client): show reminder target system name"
```

## Final Verification

- [ ] 运行 `cd qim-server && go test ./handler ./service`。
- [ ] 运行 `cd qim-admin && npm run build`。
- [ ] 运行 `cd qim-client && npm run test:unit -- tests/unit/message-remind-result.test.ts && npm run typecheck`。
- [ ] 验证已保存的旧 JSON 没有 `system_name` 时，客户端回退为“提醒已送达外部系统”。
