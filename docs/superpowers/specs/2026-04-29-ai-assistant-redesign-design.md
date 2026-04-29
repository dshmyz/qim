# AI助手功能重构设计文档

**日期**: 2026-04-29  
**状态**: 待审批

## 一、需求概述

### 1.1 背景
当前AI助手功能存在以下问题：
- AI大模型配置与用户级AI助手混在一起，职责不清
- AIAssistantApp.vue文件过大（1050行），包含多个功能模块
- 多个创建机器人入口重复
- 用户无法自定义自己的大模型配置

### 1.2 目标
1. 将系统配置迁移到管理后台
2. 重构AI助手前端架构，拆分组件
3. 新增用户级"我的模型配置"功能
4. 优化创建机器人流程

---

## 二、架构设计

### 2.1 功能分层

#### 用户端（qim-client）
```
AI助手中心
├── Tab 1: 对话中心 - 统一的对话界面
├── Tab 2: 我的机器人 - 管理个人创建的机器人
├── Tab 3: 创建机器人 - 创建新机器人向导
└── Tab 4: 我的模型配置 - 管理个人大模型配置（新增）
```

#### 管理后台（qim-admin）
```
AI管理
├── AI助手管理Tab - 系统机器人管理
├── Bot审批Tab - 用户机器人审批
└── 系统配置Tab - 全局大模型配置（从用户端迁移）
```

### 2.2 配置使用策略

| 配置类型 | 可见性 | 使用场景 | 审批要求 | 限制数量 |
|---------|--------|---------|---------|---------|
| 系统配置 | 管理员 | 所有用户默认使用 | N/A | 1个 |
| 用户自定义配置 | 用户个人 | 创建个人机器人 | 不需要审批 | 最多5个 |

**审批规则**：
- 使用系统模型创建机器人 → 需要管理员审批
- 使用用户自定义配置创建机器人 → 无需审批，直接可用

---

## 三、数据库设计

### 3.1 新增表：用户AI配置

```sql
CREATE TABLE user_ai_configs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  config_name VARCHAR(50) NOT NULL,           -- 配置名称（如"我的GPT-4"）
  provider VARCHAR(20) NOT NULL,              -- 提供商（openai/alibaba/tencent等）
  api_key_encrypted TEXT NOT NULL,            -- 加密后的API Key
  model_name VARCHAR(50) NOT NULL,            -- 模型名称
  base_url VARCHAR(255),                      -- Base URL（可选）
  temperature DECIMAL(3,2) DEFAULT 0.7,       -- 温度参数
  max_tokens INTEGER DEFAULT 1000,            -- 最大tokens
  is_verified BOOLEAN DEFAULT FALSE,          -- 是否已验证
  last_tested_at DATETIME,                    -- 最后测试时间
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id),
  UNIQUE(user_id, config_name)
);

-- 索引
CREATE INDEX idx_user_ai_configs_user_id ON user_ai_configs(user_id);
```

### 3.2 修改表：机器人表（ai_bots）

```sql
-- 新增字段
ALTER TABLE ai_bots ADD COLUMN user_config_id INTEGER REFERENCES user_ai_configs(id);
ALTER TABLE ai_bots ADD COLUMN use_system_config BOOLEAN DEFAULT TRUE;
```

**字段说明**：
- `user_config_id`: 关联的用户配置ID（如果使用自定义配置）
- `use_system_config`: 是否使用系统配置（TRUE=系统配置，FALSE=用户配置）

---

## 四、API设计

### 4.1 用户配置管理API

#### 获取我的配置列表
```
GET /api/v1/ai/configs/my
Authorization: Bearer <token>
Response:
{
  "code": 0,
  "data": [
    {
      "id": 1,
      "config_name": "我的GPT-4",
      "provider": "openai",
      "model_name": "gpt-4",
      "base_url": "https://api.openai.com/v1",
      "is_verified": true,
      "last_tested_at": "2026-04-29T10:00:00Z",
      "created_at": "2026-04-29T10:00:00Z"
    }
  ]
}
```

#### 添加配置
```
POST /api/v1/ai/configs/my
Authorization: Bearer <token>
Request:
{
  "config_name": "我的GPT-4",
  "provider": "openai",
  "api_key": "sk-xxx...",
  "model_name": "gpt-4",
  "base_url": "https://api.openai.com/v1"
}
Response:
{
  "code": 0,
  "data": {
    "id": 1,
    "is_verified": true
  }
}
```

**业务逻辑**：
1. 检查用户配置数量是否超过5个限制
2. 加密API Key后存储
3. 立即测试连接，返回验证状态
4. 测试失败则返回错误，不保存配置

#### 更新配置
```
PUT /api/v1/ai/configs/my/:id
Authorization: Bearer <token>
Request: 同添加配置
Response: 同添加配置
```

#### 删除配置
```
DELETE /api/v1/ai/configs/my/:id
Authorization: Bearer <token>
Response:
{
  "code": 0,
  "message": "删除成功"
}
```

**业务逻辑**：
- 检查是否有机器人正在使用该配置
- 如果有，拒绝删除并提示

#### 测试配置连接
```
POST /api/v1/ai/configs/my/:id/test
Authorization: Bearer <token>
Response:
{
  "code": 0,
  "data": {
    "success": true,
    "message": "连接成功",
    "model_info": {
      "name": "gpt-4",
      "max_tokens": 8192
    }
  }
}
```

### 4.2 机器人创建API修改

```
POST /api/v1/ai/bots
Authorization: Bearer <token>
Request:
{
  "name": "我的客服机器人",
  "description": "用于客服场景",
  "type": "ai",
  "use_system_config": false,           // 新增字段
  "user_config_id": 1,                  // 新增字段（使用自定义配置时必填）
  "system_prompt": "你是一个客服助手..."
}
```

**业务逻辑**：
- `use_system_config = true`: 使用系统配置，需要审批
- `use_system_config = false`: 使用用户自定义配置，无需审批，直接创建

---

## 五、前端组件设计

### 5.1 组件树结构

```
AIAssistantApp.vue（主容器）
├── TabNavigation.vue（Tab导航栏）
├── ChatCenter.vue（对话中心Tab）
│   ├── BotList.vue（机器人列表）
│   └── BotChatView.vue（对话界面）
├── MyBotsPanel.vue（我的机器人Tab）
├── CreateBotWizard.vue（创建机器人Tab）
│   ├── CreateMethodSelector.vue（选择创建方式）
│   ├── TemplateSelector.vue（模板选择）
│   └── CustomConfigForm.vue（自定义配置表单）
│       ├── ModelSourceSelector.vue（模型来源选择）
│       └── BotBasicForm.vue（基本信息）
└── MyModelConfigs.vue（我的模型配置Tab）
    ├── ModelConfigList.vue（配置列表）
    ├── ModelConfigCard.vue（配置卡片）
    ├── ModelConfigFormModal.vue（添加/编辑弹窗）
    └── ConnectionTester.vue（连接测试）
```

### 5.2 关键组件设计

#### AIAssistantApp.vue（主容器）
- 职责：管理Tab状态、提供共享数据
- 状态：`activeTab`、`sidebarCollapsed`
- 行数目标：~300行（从1050行减少）

#### MyModelConfigs.vue（我的模型配置）
- 职责：展示和管理用户的模型配置
- 功能：
  - 配置列表展示（卡片式）
  - 添加/编辑/删除配置
  - 配置验证状态显示
  - 数量限制提示（5个）

#### ChatCenter.vue（对话中心）
- 职责：统一的对话界面，移除模式选择
- 功能：
  - 直接进入机器人列表
  - 选择机器人后进入对话
  - 保留对话历史
- 与原设计的差异：
  - 移除模式选择器（聊天/运维）
  - 移除运维工具模态框
  - 简化为用户直接选择机器人开始对话

#### CreateBotWizard.vue（创建机器人向导）
- 职责：引导用户完成机器人创建
- 流程：
  1. 选择创建方式（模板/自定义）
  2. 模板路径：选择模板 → 填写名称 → 完成
  3. 自定义路径：填写基本信息 → 选择模型来源 → 完成

### 5.3 UI设计规范

#### Tab导航
- 样式：底部边框高亮
- 激活态：主题色文字 + 底部2px边框
- 图标：Font Awesome图标

#### 模型配置卡片
```
┌─────────────────────────────────────┐
│ 🔑 我的GPT-4                 [操作] │
│ OpenAI • gpt-4 • ✅ 已验证          │
│ 添加于 2026-04-29                   │
└─────────────────────────────────────┘
```

#### 验证状态标识
- ✅ 已验证：绿色
- ⚠️ 未验证：橙色
- ❌ 失效：红色

---

## 六、数据安全

### 6.1 API Key加密
- 使用AES-256-GCM加密
- 密钥从环境变量读取
- 数据库中仅存储加密后的密文

### 6.2 访问控制
- 用户只能查看/编辑自己的配置
- 管理员无法查看用户的API Key
- API响应中不返回完整的API Key（返回掩码版本）

### 6.3 删除保护
- 配置被机器人使用时禁止删除
- 提示用户先删除或修改关联的机器人

---

## 七、迁移计划

### 7.1 系统配置迁移
1. 将 `AIConfigApp.vue` 从 qim-client 移动到 qim-admin
2. 更新路由配置
3. 添加管理员权限检查

### 7.2 兼容性
- 现有机器人继续使用系统配置
- 新创建的机器人可以选择配置来源
- 提供配置迁移工具（可选）

---

## 八、测试计划

### 8.1 单元测试
- 用户配置CRUD逻辑
- 配置验证逻辑
- API Key加解密
- 数量限制检查

### 8.2 集成测试
- 创建机器人流程
- 使用自定义配置的对话
- 配置删除保护
- 对话中心简化后功能验证

### 8.3 用户验收测试
- 添加配置并验证连接
- 使用自定义配置创建机器人
- 创建后机器人对话测试
- 配置数量限制提示

---

## 九、后续优化方向

1. **配置分享**（可选）
   - 团队内共享配置
   - 配置模板市场

2. **用量统计**
   - 每个配置的Token消耗
   - API调用次数统计

3. **配置导入导出**
   - 导出为JSON
   - 从文件导入

---

## 十、规格自检

- [x] 无占位符或TODO
- [x] 数据库设计与API设计一致
- [x] 前端组件与功能需求匹配
- [x] 安全策略明确
- [x] 范围适当，可以按实施计划分阶段完成

---

**待用户审查后进入实施计划阶段**
