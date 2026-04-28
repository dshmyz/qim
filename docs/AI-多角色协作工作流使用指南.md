# AI 多角色协作工作流使用指南

> 基于 agency-orchestrator 的 YAML 工作流，在 Claude Code / Trae IDE 中实现多角色协作开发

***

## 📋 目录

- [什么是工作流](#什么是工作流)
- [本次实践概述](#本次实践概述)
- [核心概念](#核心概念)
- [环境搭建](#环境搭建)
- [角色创建规范](#角色创建规范)
- [工作流定义](#工作流定义)
- [使用方式](#使用方式)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)
- [实战案例](#实战案例)

***

## 什么是工作流

工作流是一套基于 YAML 定义的多角色协作流程，通过让 AI 依次扮演不同专业角色（产品经理、技术架构师、UI/UX 设计师、开发工程师等），实现从需求分析到开发实现的端到端协作。

**核心价值：**

- 🎭 **角色专业化**：每个角色有独立的专业知识和沟通风格
- 🔄 **流程标准化**：通过 YAML 定义可复用的协作流程
- 📝 **输出结构化**：每个步骤输出标准化的文档和交付物
- 🤝 **多人协作模拟**：模拟真实团队的产品、设计、开发协作

***

## 本次实践概述

### 背景

我们开发了一个企业级即时通讯系统（QIM），技术栈为：

- **前端**：Vue 3 + TypeScript + Electron + Element Plus
- **后端**：Go + Gin + GORM + SQLite

### 目标

创建一套适配我们团队的多角色协作工作流，提升需求评审、架构设计和开发实现的质量。

### 产出

1. **4 个定制化角色**：
   - 产品经理（已存在）
   - 技术架构师（新建）
   - UI/UX 设计师（新建）
   - 全栈开发工程师（新建）
2. **3 个预设工作流**：
   - `product-tech-review.yaml` - 产品+技术全链路评审
   - `feature-to-implementation.yaml` - 功能需求到开发实现
   - `code-review.yaml` - 代码审查与优化

***

## 核心概念

### 工作流结构

```yaml
name: "工作流名称"
description: "工作流描述"
agents_dir: "agency-agents-zh"    # 角色定义目录
inputs:                            # 输入变量
  - name: feature_description
    required: true
    description: "功能需求描述"
  - name: target_users
    required: false
    default: "企业用户"
steps:                             # 执行步骤
  - id: step_id
    role: "category/agent-name"    # 角色路径
    task: "任务描述 {{变量}}"       # 支持模板变量
    output: variable_name          # 输出变量名
    depends_on: [other_step_id]    # 依赖关系
```

### 执行流程

```
第 1 步: 解析工作流 → 读取 YAML，提取角色和步骤
第 2 步: 收集输入   → 收集用户提供的功能需求
第 3 步: 构建顺序   → 根据 depends_on 进行拓扑排序
第 4 步: 逐层执行   → 读取角色文件，扮演角色执行任务
第 5 步: 保存结果   → 输出到 .ao-output/ 目录
```

***

## 环境搭建

### 1. 安装角色库

```bash
cd /your/project/path
git clone --depth 1 https://github.com/jnMetaCode/agency-agents-zh.git
```

### 2. 目录结构

```
project/
├── .trae/
│   ├── skills/
│   │   └── workflow-runner/    # 工作流执行器 skill
│   └── workflows/              # 自定义工作流目录
│       ├── product-tech-review.yaml
│       ├── feature-to-implementation.yaml
│       └── code-review.yaml
├── agency-agents-zh/           # 角色定义库
│   ├── product/
│   │   └── product-manager.md
│   ├── engineering/
│   │   ├── engineering-technical-architect.md
│   │   └── engineering-fullstack-developer.md
│   └── design/
│       └── design-im-ui-ux-designer.md
└── .ao-output/                 # 工作流输出目录
    └── 功能需求到开发实现-2026-04-27/
        ├── steps/
        │   ├── 1-analyze_requirement.md
        │   ├── 2-design_architecture.md
        │   ├── 3-design_ui_ux.md
        │   └── 4-plan_development.md
        ├── summary.md
        └── metadata.json
```

***

## 角色创建规范

### 角色文件格式

```markdown
---
name: 角色名称
description: "角色描述"
color: 颜色标识
tools: 可用工具列表
---

# 🎭 角色名称智能体

## 🧠 身份与记忆
- 角色背景和经验
- 核心原则和价值观

## 🎯 核心使命
- 角色的主要职责

## 🚨 关键规则
1. 规则 1
2. 规则 2
...

## 🛠️ 技术交付物
- 模板 1
- 模板 2
...

## 📋 工作流程
- 阶段 1
- 阶段 2
...

## 💬 沟通风格
- 沟通特点

## 📊 成功指标
- 衡量标准

## 🎭 个性特征
- 代表性语录
```

### 创建角色的关键要点

1. **角色定位清晰**：明确角色的专业领域和职责边界
2. **原则具体可执行**：不要泛泛而谈，要有具体行为准则
3. **交付物模板化**：提供标准化的输出模板
4. **沟通风格鲜明**：每个角色有独特的表达方式
5. **适配团队技术栈**：针对团队实际使用的技术定制

### 示例：技术架构师角色核心规则

```markdown
1. 先找约束，不要先跳到方案
2. 先画上下文图，再画架构图
3. 每一项技术选型都必须有理由、替代方案对比和迁移成本评估
4. 说不——清晰地、技术性地、经常地
5. 设计之前先度量，上线之后必验证
```

***

## 工作流定义

### 工作流 1：功能需求到开发实现

**用途**：从产品需求分析到技术架构设计、UI 设计、最终输出开发计划的端到端流程

**流程**：

```
产品经理需求分析
       ↓
    ┌──┴──┐
技术架构  UI设计  (并行)
    └──┬──┘
       ↓
全栈开发实现计划
```

**YAML 定义**：

```yaml
name: "功能需求到开发实现"
agents_dir: "agency-agents-zh"
inputs:
  - name: feature_request
    required: true
    description: "功能需求描述"
  - name: context
    required: false
    default: "QIM 企业即时通讯系统"
    description: "项目上下文"

steps:
  - id: analyze_requirement
    role: "product/product-manager"
    task: "作为产品经理，分析以下功能需求：\n{{feature_request}}\n\n项目上下文：{{context}}"
    output: requirement_doc

  - id: design_architecture
    role: "engineering/engineering-technical-architect"
    task: "作为技术架构师，基于需求文档设计技术方案：\n\n需求文档：\n{{requirement_doc}}"
    output: architecture_doc
    depends_on: [analyze_requirement]

  - id: design_ui_ux
    role: "design/design-im-ui-ux-designer"
    task: "作为 UI/UX 设计师，基于需求文档进行界面和交互设计：\n\n需求文档：\n{{requirement_doc}}"
    output: design_doc
    depends_on: [analyze_requirement]

  - id: plan_development
    role: "engineering/engineering-fullstack-developer"
    task: "作为全栈开发工程师，基于架构设计和设计规范制定实现计划：\n需求：{{requirement_doc}}\n架构：{{architecture_doc}}\n设计：{{design_doc}}"
    output: development_plan
    depends_on: [design_architecture, design_ui_ux]
```

### 工作流 2：产品+技术全链路评审

**用途**：多维度评审功能需求（产品价值、架构合理性、用户体验、开发可行性）

### 工作流 3：代码审查与优化

**用途**：从产品质量、架构合理性、用户体验和性能优化四个角度审查代码

***

## 使用方式

### 方式 1：直接调用工作流（推荐）

在 Trae IDE 中输入：

```
运行 workflows/feature-to-implementation.yaml
功能需求：我想添加一个消息已读回执功能
```

### 方式 2：单独调用角色

```
作为技术架构师，帮我设计一个文件上传功能的架构方案
```

### 方式 3：多角色协作评审

```
让产品经理、架构师和开发工程师一起评审这个 PRD：
[粘贴你的 PRD 内容]
```

### 方式 4：代码审查

```
运行 workflows/code-review.yaml
代码上下文：/path/to/your/file.vue
审查重点：错误处理
```

***

## 最佳实践

### 1. 角色设计

| 实践     | 说明                   |
| ------ | -------------------- |
| 角色数量适中 | 3-5 个核心角色即可，过多会增加复杂度 |
| 职责边界清晰 | 每个角色有明确的输入输出，不重叠     |
| 技术栈定制  | 针对团队实际使用的技术定制角色      |
| 交付物模板化 | 提供标准化的输出模板，保证质量      |

### 2. 工作流设计

| 实践     | 说明                         |
| ------ | -------------------------- |
| 步骤精简   | 每个步骤都有明确的价值，不要为了多而多        |
| 依赖合理   | 利用 depends\_on 实现并行，缩短执行时间 |
| 变量传递清晰 | output 变量名要有意义，便于后续步骤引用    |
| 模板变量丰富 | task 中使用 {{变量}} 传递上下文      |

### 3. 使用流程

```
1. 明确需求 → 想清楚要让 AI 帮你做什么
2. 选择工作流 → 选择最匹配的工作流或角色
3. 提供上下文 → 尽可能详细地描述背景和需求
4. 审查输出 → 人工审查 AI 的输出，不要直接使用
5. 迭代优化 → 根据实际效果调整角色和工作流
```

### 4. 常见场景

| 场景    | 推荐工作流                     | 预期输出                     |
| ----- | ------------------------- | ------------------------ |
| 新功能开发 | feature-to-implementation | PRD + 架构设计 + 设计规范 + 开发计划 |
| 需求评审  | product-tech-review       | 多维度评审报告                  |
| 代码审查  | code-review               | 问题清单 + 修复建议              |
| 技术选型  | 单独调用技术架构师                 | 技术选型评估报告                 |
| UI 设计 | 单独调用 UI/UX 设计师            | 组件设计规范                   |

***

## 常见问题

### Q1: 工作流执行失败怎么办？

**A**: 检查以下几点：

1. 角色文件路径是否正确（`agents_dir` 配置）
2. 角色文件是否存在（`{agents_dir}/{role}.md`）
3. 必填输入是否提供（`required: true` 的 inputs）
4. YAML 语法是否正确

### Q2: 角色输出不符合预期怎么办？

**A**: 调整角色的 task 描述：

- 更具体地描述期望输出
- 提供更多上下文信息
- 调整角色文件中的关键规则和交付物模板

### Q3: 如何让角色更专业？

**A**:

- 在角色文件中添加更多行业知识和实践经验
- 提供具体的案例和示例
- 明确沟通风格和成功指标

### Q4: 工作流可以嵌套吗？

**A**: 目前不支持工作流嵌套，但可以在一个工作流的 task 中引用另一个工作流的输出。

### Q5: 执行时间过长怎么办？

**A**:

- 优化 depends\_on，让可并行的步骤并行执行
- 精简步骤，合并相似的任务
- 减少角色文件的长度，聚焦核心内容

***

## 实战案例

### 案例 1：消息已读回执功能开发

**需求**：为 IM 系统添加消息已读回执功能

**执行过程**：

1. **产品经理**（Step 1）
   - 输出 PRD 文档
   - 定义用户故事和验收标准
   - RICE 评分：381（高优先级）
2. **技术架构师**（Step 2，并行）
   - 系统上下文图
   - 组件架构设计
   - API 接口设计
   - 数据库表设计
3. **UI/UX 设计师**（Step 3，并行）
   - 用户旅程图
   - 组件设计规范（已读状态图标、已读用户列表、模态框）
   - 交互规则和无障碍要求
4. **全栈开发工程师**（Step 4）
   - 文件结构拆分方案
   - TypeScript 类型定义
   - composable 实现
   - 测试计划和性能优化清单

**输出文件**：

```
.ao-output/功能需求到开发实现-2026-04-27/
├── steps/
│   ├── 1-analyze_requirement.md      # 产品经理 PRD
│   ├── 2-design_architecture.md      # 技术架构设计
│   ├── 3-design_ui_ux.md             # UI/UX 设计规范
│   └── 4-plan_development.md         # 开发实现计划
├── summary.md                        # 最终总结
└── metadata.json                     # 元数据
```

### 案例 2：代码审查（错误处理专项）

**需求**：审查 Main.vue 文件的错误处理

**发现问题**：

1. try-catch 只 console.error，没有用户反馈
2. 错误被掩盖，后续开发者不知道问题所在
3. 没有错误上报机制

**修复建议**：

```typescript
// ❌ 错误做法：掩盖错误
try {
  await markMessagesAsRead(conversationId)
} catch (error) {
  console.error('标记消息已读失败:', error)
}

// ✅ 正确做法：明确处理
try {
  await markMessagesAsRead(conversationId)
} catch (error) {
  console.error('标记消息已读失败:', error)
  showToast('标记已读失败，请重试', 'error')
  reportError(error)  // 上报监控
}
```

***

## 总结

### 价值

1. **标准化流程**：从需求到开发的完整流程，减少遗漏
2. **多视角评审**：产品、技术、设计多角度审查，降低风险
3. **知识沉淀**：角色文件是团队知识的载体
4. **效率提升**：并行执行，缩短评审和规划时间

### 适用场景

✅ 适合：

- 新功能开发前的需求评审和技术设计
- 重要代码的多维度审查
- 技术选型评估
- UI/UX 设计规范制定

❌ 不适合：

- 简单的 bug 修复
- 紧急的线上问题处理
- 无需多角色协作的单点任务

### 下一步

1. **收集团队反馈**：根据实际使用体验调整角色和工作流
2. **扩展角色库**：添加测试工程师、DevOps 等角色
3. **创建工作流变体**：如 Bug 修复流程、性能优化流程等
4. **集成到 CI/CD**：在 PR 流程中自动触发代码审查工作流

***

## 附录

### A. 角色文件清单

| 角色        | 文件路径                                                              | 专长                     |
| --------- | ----------------------------------------------------------------- | ---------------------- |
| 产品经理      | `agency-agents-zh/product/product-manager.md`                     | 需求分析、PRD、路线图           |
| 技术架构师     | `agency-agents-zh/engineering/engineering-technical-architect.md` | 系统架构、技术选型、性能方案         |
| UI/UX 设计师 | `agency-agents-zh/design/design-im-ui-ux-designer.md`             | IM 界面设计、交互体验、设计系统      |
| 全栈开发工程师   | `agency-agents-zh/engineering/engineering-fullstack-developer.md` | Vue 3 + Go 开发、组件化、错误处理 |

### B. 工作流清单

| 工作流        | 文件路径                                             | 用途         |
| ---------- | ------------------------------------------------ | ---------- |
| 产品+技术全链路评审 | `.trae/workflows/product-tech-review.yaml`       | 多维度评审功能需求  |
| 功能需求到开发实现  | `.trae/workflows/feature-to-implementation.yaml` | 从需求分析到开发计划 |
| 代码审查与优化    | `.trae/workflows/code-review.yaml`               | 多维度代码审查    |

### C. 参考资源

- [agency-agents-zh](https://github.com/jnMetaCode/agency-agents-zh) - 角色定义库
- [workflow-runner Skill](.trae/skills/workflow-runner/SKILL.md) - 工作流执行器说明

***

*文档版本：v1.0*
*创建日期：2026-04-27*
*维护者：QIM 团队*
