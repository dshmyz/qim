# AI 分身能力增强实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为 AI 分身增加 RAG 知识检索、智能触发、长期记忆和主动发言能力，使其更智能、更自然地代表用户回复消息。

**架构：** 
- 后端引入 sqvect 向量数据库，为知识库文档和笔记内容提供语义检索
- 分身回复流程中集成 RAG：检索相关知识 → 注入 prompt → LLM 生成回复
- 智能触发：用轻量级 LLM 判断群消息是否需要分身回复
- 长期记忆：重要信息向量化存储，回复时检索相关记忆

**技术栈：** Go, sqvect, OpenAI/兼容 embedding 接口, Vue 3, Pinia

---

## 文件结构

### 后端（qim-server）

| 文件 | 操作 | 职责 |
|------|------|------|
| `go.mod` | 修改 | 添加 sqvect 依赖 |
| `ai/provider.go` | 修改 | Provider 接口添加 Embedding 方法 |
| `ai/provider_openai.go` | 修改 | 实现 OpenAI embedding |
| `ai/provider_baidu.go` | 修改 | 实现百度 embedding |
| `ai/provider_alibaba.go` | 修改 | 实现阿里 embedding |
| `ai/provider_tencent.go` | 修改 | 实现腾讯 embedding |
| `ai/provider_bytedance.go` | 修改 | 实现字节 embedding |
| `ai/provider_anthropic.go` | 修改 | 实现 Anthropic embedding |
| `service/vector_service.go` | 创建 | sqvect 向量存储服务封装 |
| `service/document_parser.go` | 创建 | 文档内容解析（PDF/TXT/DOCX） |
| `service/chunker.go` | 创建 | 文本切片服务 |
| `service/group_document_service.go` | 修改 | 添加文档时触向量化 |
| `service/note_vector_service.go` | 创建 | 笔记向量化和检索服务 |
| `service/note_service.go` | 修改 | 保存/更新笔记时触发向量化 |
| `service/avatar_memory_service.go` | 创建 | 分身长期记忆存储和检索 |
| `service/avatar_trigger_service.go` | 创建 | 智能触发判断服务 |
| `service/avatar_service.go` | 修改 | 回复流程集成 RAG 和记忆 |
| `handler/group_document_handler.go` | 修改 | 添加文档返回处理状态 |
| `handler/avatar_handler.go` | 修改 | 添加记忆管理和触发配置接口 |
| `model/avatar.go` | 修改 | 添加记忆和触发相关模型 |
| `model/document.go` | 创建 | 文档处理状态模型 |
| `app/init.go` | 修改 | 初始化向量服务 |
| `app/routes.go` | 修改 | 添加新 API 路由 |
| `di/container.go` | 修改 | 注册新服务 |
| `ddl_mysql.sql` | 修改 | 添加新表 |
| `ddl_sqlite.sql` | 修改 | 添加新表 |

### 前端（qim-client）

| 文件 | 操作 | 职责 |
|------|------|------|
| `src/components/avatar/AvatarKnowledgeSettings.vue` | 修改 | 添加文档处理状态显示 |
| `src/components/avatar/AvatarTriggerSettings.vue` | 修改 | 添加智能触发配置 |
| `src/components/avatar/AvatarMemoryPanel.vue` | 创建 | 分身记忆管理面板 |
| `src/components/avatar/AvatarSettingsPanel.vue` | 修改 | 集成记忆面板入口 |
| `src/composables/useAvatar.ts` | 修改 | 添加记忆相关方法 |
| `src/api/ai.ts` | 修改 | 添加新 API 调用 |
| `src/types/index.ts` | 修改 | 添加新类型定义 |

---

## 后端实现

### 任务 1：引入 sqvect 和添加 Embedding 接口

#### 步骤 1：安装 sqvect 依赖

```bash
cd qim-server && go get github.com/liliang-cn/sqvect/v2
```

#### 步骤 2：扩展 Provider 接口

修改 `ai/provider.go`，添加 Embedding 方法：

```go
// ai/provider.go
type Provider interface {
    Chat(messages []Message) (string, error)
    ChatStream(messages []Message, onChunk func(chunk StreamChunk) error) error
    Embedding(text string) ([]float32, error)  // 新增
    IsConfigured() bool
}
```

#### 步骤 3：为每个 Provider 实现 Embedding

以 OpenAI 为例：

```go
// ai/provider_openai.go
func (p *OpenAIProvider) Embedding(text string) ([]float32, error) {
    type embeddingRequest struct {
        Model string `json:"model"`
        Input string `json:"input"`
    }
    
    req := embeddingRequest{
        Model: p.model,
        Input: text,
    }
    
    // 调用 OpenAI /v1/embeddings 接口
    // 解析响应，返回 []float32
}
```

每个 Provider 的 embedding 端点不同，但流程一致：构造请求 → HTTP 调用 → 解析响应。

#### 步骤 4：创建 VectorService

```go
// service/vector_service.go
package service

import (
    "context"
    "fmt"
    "github.com/liliang-cn/sqvect/v2"
)

type VectorService struct {
    db *sqvect.DB
}

func NewVectorService(path string) (*VectorService, error) {
    db, err := sqvect.Open(sqvect.DefaultConfig(path))
    if err != nil {
        return nil, err
    }
    return &VectorService{db: db}, nil
}

func (s *VectorService) EnsureCollection(name string) error {
    _, err := s.db.CreateCollection(context.Background(), name)
    return err
}

func (s *VectorService) AddVector(collection string, embedding []float32, content string, metadata map[string]any) error {
    _, err := s.db.Quick().AddWithMetadata(context.Background(), embedding, content, metadata)
    return err
}

func (s *VectorService) Search(collection string, queryVector []float32, topK int) ([]sqvect.ScoredEmbedding, error) {
    return s.db.Quick().Search(context.Background(), queryVector, topK)
}

func (s *VectorService) DeleteByMetadata(collection string, key string, value any) error {
    // 按元数据删除
    return nil
}
```

#### 步骤 5：在 DI 容器中注册 VectorService

修改 `di/container.go`，添加 VectorService 字段和初始化。

#### 步骤 6：在 app/init.go 中初始化

启动时创建向量数据库文件目录。

---

### 任务 2：实现文档解析和文本切片

#### 步骤 1：创建 DocumentParser

```go
// service/document_parser.go
package service

import (
    "os"
    "strings"
)

type DocumentParser struct{}

func (p *DocumentParser) Parse(filePath string) (string, error) {
    ext := strings.ToLower(filePath[strings.LastIndex(filePath, ".")+1:])
    
    switch ext {
    case "txt", "md", "markdown":
        return p.parseText(filePath)
    case "pdf":
        return p.parsePDF(filePath)
    case "docx":
        return p.parseDocx(filePath)
    default:
        return p.parseText(filePath)
    }
}

func (p *DocumentParser) parseText(filePath string) (string, error) {
    data, err := os.ReadFile(filePath)
    return string(data), err
}

func (p *DocumentParser) parsePDF(filePath string) (string, error) {
    // 使用 ledongthuc/pdf 库提取文本
    // 简化实现：先返回空字符串，后续完善
    return "", nil
}

func (p *DocumentParser) parseDocx(filePath string) (string, error) {
    // 使用 docx 库提取文本
    return "", nil
}
```

#### 步骤 2：创建 Chunker

```go
// service/chunker.go
package service

import (
    "regexp"
    "strings"
)

type Chunk struct {
    Content string
    Title   string
}

func SplitMarkdownByHeading(text string) []Chunk {
    re := regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
    
    var chunks []Chunk
    matches := re.FindAllStringIndex(text, -1)
    
    if len(matches) == 0 {
        return []Chunk{{Content: text, Title: ""}}
    }
    
    for i, match := range matches {
        var content string
        title := text[match[0]:match[1]]
        
        if i+1 < len(matches) {
            content = text[match[0]:matches[i+1][0]]
        } else {
            content = text[match[0]:]
        }
        
        chunks = append(chunks, Chunk{
            Content: content,
            Title:   strings.TrimLeft(title, "# "),
        })
    }
    
    return chunks
}

func SplitBySize(text string, maxSize int) []string {
    if len(text) <= maxSize {
        return []string{text}
    }
    
    paragraphs := strings.Split(text, "\n\n")
    var chunks []string
    current := ""
    
    for _, p := range paragraphs {
        if len(current)+len(p) > maxSize && current != "" {
            chunks = append(chunks, current)
            current = p
        } else {
            if current != "" {
                current += "\n\n"
            }
            current += p
        }
    }
    
    if current != "" {
        chunks = append(chunks, current)
    }
    
    return chunks
}
```

---

### 任务 3：实现知识库 RAG 检索

#### 步骤 1：创建 DocumentProcess 模型

```go
// model/document.go
package model

import "time"

type DocumentProcessStatus struct {
    ID         uint      `json:"id" gorm:"primarykey"`
    GroupDocID uint      `json:"group_doc_id" gorm:"not null;index"`
    Status     string    `json:"status" gorm:"size:20;default:'pending'"` // pending, processing, completed, failed
    Error      string    `json:"error" gorm:"type:text"`
    ChunkCount int       `json:"chunk_count"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}
```

#### 步骤 2：改造 AddGroupDocument handler

修改 `handler/group_document_handler.go` 的 AddGroupDocument 函数，添加后创建 DocumentProcessStatus 记录并触发异步向量化。

#### 步骤 3：实现向量化流程

在 `service/group_document_service.go` 中添加：

```go
func (s *GroupDocumentService) ProcessDocument(groupDocID uint) error {
    // 1. 查询文档记录获取文件路径
    // 2. 更新状态为 processing
    // 3. 解析文档内容
    // 4. 切片
    // 5. 生成向量并存储
    // 6. 更新状态为 completed
}
```

#### 步骤 4：创建 RAG 检索方法

```go
func (s *GroupDocumentService) SearchKnowledge(groupID uint, query string, topK int) ([]KnowledgeResult, error) {
    // 1. query → embedding
    // 2. 在 group_{groupID} collection 中搜索
    // 3. 返回 Top-K 结果
}
```

---

### 任务 4：实现笔记向量化服务

#### 步骤 1：创建 NoteVectorService

```go
// service/note_vector_service.go
package service

type NoteVectorService struct {
    vectorSvc    *VectorService
    aiService    *ai.AIService
}

func (s *NoteVectorService) VectorizeNote(userID, noteID uint, title, content string) error {
    collectionName := fmt.Sprintf("user_notes_%d", userID)
    s.vectorSvc.EnsureCollection(collectionName)
    
    // 删除旧向量
    s.vectorSvc.DeleteByMetadata(collectionName, "note_id", noteID)
    
    // 按标题切片
    chunks := SplitMarkdownByHeading(content)
    if len(chunks) == 0 {
        chunks = []Chunk{{Content: content, Title: title}}
    }
    
    for i, chunk := range chunks {
        embedding, err := s.aiService.Embed(chunk)
        if err != nil {
            return err
        }
        
        s.vectorSvc.AddVector(collectionName, embedding, chunk.Content, map[string]any{
            "note_id":  noteID,
            "chunk_id": i,
            "title":    title,
        })
    }
    
    return nil
}

func (s *NoteVectorService) SearchNotes(userID uint, query string, topK int) ([]KnowledgeResult, error) {
    collectionName := fmt.Sprintf("user_notes_%d", userID)
    queryVector, err := s.aiService.Embed(query)
    if err != nil {
        return nil, err
    }
    
    return s.vectorSvc.Search(collectionName, queryVector, topK)
}
```

#### 步骤 2：修改 NoteService

在 `service/note_service.go` 的 CreateNote 和 UpdateNote 方法中调用向量化。

---

### 任务 5：实现分身长期记忆

#### 步骤 1：创建 AvatarMemoryService

```go
// service/avatar_memory_service.go
package service

type AvatarMemoryService struct {
    vectorSvc *VectorService
    aiService *ai.AIService
    db        *gorm.DB
}

// 记忆重要信息
func (s *AvatarMemoryService) Remember(userID uint, conversationID uint, content string) error {
    collectionName := fmt.Sprintf("avatar_memory_%d", userID)
    s.vectorSvc.EnsureCollection(collectionName)
    
    embedding, err := s.aiService.Embed(content)
    if err != nil {
        return err
    }
    
    return s.vectorSvc.AddVector(collectionName, embedding, content, map[string]any{
        "conversation_id": conversationID,
        "remembered_at":   time.Now().Unix(),
    })
}

// 检索相关记忆
func (s *AvatarMemoryService) Recall(userID uint, query string, topK int) ([]KnowledgeResult, error) {
    collectionName := fmt.Sprintf("avatar_memory_%d", userID)
    queryVector, err := s.aiService.Embed(query)
    if err != nil {
        return nil, err
    }
    
    return s.vectorSvc.Search(collectionName, queryVector, topK)
}

// 判断信息是否值得记忆（用 LLM）
func (s *AvatarMemoryService) ShouldRemember(message string) (bool, error) {
    prompt := `判断以下对话内容是否包含值得记忆的长期信息。
值得记忆的信息包括：个人偏好、重要决定、项目关键信息、约定事项。
普通闲聊、简短回复不需要记忆。
只返回 true 或 false。

内容：` + message
    
    result, err := s.aiService.GetCompletion([]ai.Message{{Role: "user", Content: prompt}})
    return strings.Contains(strings.ToLower(result), "true"), err
}
```

---

### 任务 6：实现智能触发

#### 步骤 1：创建 AvatarTriggerService

```go
// service/avatar_trigger_service.go
package service

type AvatarTriggerService struct {
    aiService *ai.AIService
    db        *gorm.DB
}

// ShouldReply 判断分身是否应该回复当前消息
func (s *AvatarTriggerService) ShouldReply(userID uint, conversationID uint, message string, senderName string) (bool, string, error) {
    // 1. 检查关键词规则
    rules := s.getTriggerRules(userID)
    if s.matchKeywords(rules, message) {
        return true, "关键词触发", nil
    }
    
    // 2. 检查时间范围
    if !s.inTimeRange(rules) {
        return false, "", nil
    }
    
    // 3. LLM 智能判断
    return s.llmShouldReply(userID, message, senderName)
}

func (s *AvatarTriggerService) llmShouldReply(userID uint, message, senderName string) (bool, string, error) {
    // 获取用户人设
    var config model.AvatarConfig
    s.db.Where("user_id = ?", userID).First(&config)
    
    prompt := fmt.Sprintf(`你是%s的AI分身。判断以下群消息是否需要你代表%s回复。

考虑因素：
1. 消息是否向你（或你代表的用户）提问？
2. 消息内容是否在你的专业领域内？
3. 是否是重要的讨论需要你参与？
4. 是否只是普通闲聊不需要回复？

只返回 JSON：{"should_reply": true/false, "reason": "原因"}

消息：%s
发送者：%s`, config.Name, senderName, message, senderName)
    
    // 轻量级判断，可用小模型
    result, err := s.aiService.GetCompletion([]ai.Message{{Role: "user", Content: prompt}})
    // 解析 JSON 返回结果
    return true, result, err
}
```

---

### 任务 7：集成 RAG 到分身回复流程

#### 步骤 1：改造 GenerateReply

修改 `service/avatar_service.go` 的 GenerateReply 方法：

```go
func (s *AvatarService) GenerateReply(userID uint, conversationID uint, triggerMessage string) (string, error) {
    // ... 现有代码：获取配置、构建 systemPrompt ...
    
    // ===== 新增：RAG 知识检索 =====
    knowledgeContext := s.buildKnowledgeContext(userID, conversationID, triggerMessage)
    
    // ===== 新增：记忆检索 =====
    memoryContext := s.buildMemoryContext(userID, triggerMessage)
    
    // 构建完整 prompt
    contextParts := []string{}
    if knowledgeContext != "" {
        contextParts = append(contextParts, "【相关知识】\n"+knowledgeContext)
    }
    if memoryContext != "" {
        contextParts = append(contextParts, "【相关记忆】\n"+memoryContext)
    }
    
    // ... 调用 LLM 生成回复 ...
}

func (s *AvatarService) buildKnowledgeContext(userID uint, conversationID uint, query string) string {
    var parts []string
    
    // 1. 检索群知识库
    var group model.Group
    if err := s.db.Where("conversation_id = ?", conversationID).First(&group); err == nil {
        docs, _ := s.groupDocService.SearchKnowledge(group.ID, query, 3)
        for _, doc := range docs {
            parts = append(parts, doc.Content)
        }
    }
    
    // 2. 检索用户笔记
    notes, _ := s.noteVectorService.SearchNotes(userID, query, 2)
    for _, note := range notes {
        parts = append(parts, fmt.Sprintf("【笔记】%s: %s", note.Metadata["title"], note.Content))
    }
    
    return strings.Join(parts, "\n\n")
}

func (s *AvatarService) buildMemoryContext(userID uint, query string) string {
    memories, _ := s.avatarMemoryService.Recall(userID, query, 3)
    var parts []string
    for _, m := range memories {
        parts = append(parts, m.Content)
    }
    return strings.Join(parts, "\n\n")
}
```

#### 步骤 2：在消息处理中集成

修改 `service/message_service.go`，在收到群消息时：
1. 调用 AvatarTriggerService.ShouldReply 判断是否需要回复
2. 如果需要，调用 AvatarService.GenerateReply
3. 发送回复消息
4. 判断是否需要记忆（AvatarMemoryService.ShouldRemember）

---

## 前端实现

### 任务 8：知识库文档状态反馈

#### 步骤 1：修改 AIKnowledgeSettings.vue

在文档列表中添加状态列：

```vue
<div v-for="doc in documents" :key="doc.id" class="document-item">
  <div class="doc-info">
    <i :class="getFileIcon(doc.file?.type)"></i>
    <span>{{ doc.file?.name }}</span>
  </div>
  <div class="doc-status">
    <span v-if="!doc.process_status || doc.process_status === 'pending'" class="status-pending">
      等待处理
    </span>
    <span v-else-if="doc.process_status === 'processing'" class="status-loading">
      正在处理中...
    </span>
    <span v-else-if="doc.process_status === 'completed'" class="status-done">
      已就绪
    </span>
    <span v-else-if="doc.process_status === 'failed'" class="status-error">
      处理失败：{{ doc.process_error }}
      <button @click="retryDocument(doc)">重试</button>
    </span>
  </div>
</div>
```

---

### 任务 9：智能触发配置 UI

#### 步骤 1：修改 AvatarTriggerSettings.vue

添加智能触发选项：

```vue
<div class="trigger-section">
  <h4>触发模式</h4>
  <div class="trigger-modes">
    <label v-for="mode in triggerModes" :key="mode.value">
      <input type="radio" v-model="rules.mode" :value="mode.value" />
      <span>{{ mode.label }}</span>
      <small>{{ mode.description }}</small>
    </label>
  </div>
</div>

<div v-if="rules.mode === 'smart'" class="smart-config">
  <div class="sensitivity-slider">
    <label>触发灵敏度</label>
    <input type="range" v-model="rules.sensitivity" min="0" max="100" />
    <span>{{ rules.sensitivity }}%</span>
  </div>
</div>
```

---

### 任务 10：分身记忆管理 UI

#### 步骤 1：创建 AvatarMemoryPanel.vue

```vue
<template>
  <div class="avatar-memory-panel">
    <h4>分身记忆</h4>
    <div class="memory-stats">
      <div class="stat-item">
        <span class="stat-value">{{ memoryCount }}</span>
        <span class="stat-label">记忆条数</span>
      </div>
    </div>
    
    <div class="memory-list">
      <div v-for="memory in memories" :key="memory.id" class="memory-item">
        <p>{{ memory.content }}</p>
        <div class="memory-meta">
          <span>{{ formatTime(memory.remembered_at) }}</span>
          <button @click="forgetMemory(memory)">忘记</button>
        </div>
      </div>
    </div>
  </div>
</template>
```

#### 步骤 2：集成到 AvatarSettingsPanel.vue

在设置面板中添加"记忆管理"标签页。

---

## 测试计划

### 后端测试

1. **VectorService 测试**
   - 创建 Collection
   - 添加向量和搜索
   - 按元数据过滤

2. **Chunker 测试**
   - Markdown 按标题切片
   - 长文本按段落切片

3. **AvatarTriggerService 测试**
   - 关键词匹配
   - LLM 判断结果解析

### 前端测试

1. 知识库文档状态展示
2. 触发模式切换
3. 记忆列表渲染

---

## 实施顺序

```
1. 基础设施（sqvect + Embedding）→ 后端可用向量能力
2. 文档处理（解析 + 切片 + 向量化）→ 知识库可检索
3. 笔记向量化 → 笔记知识可用
4. 长期记忆 → 分身能记住重要信息
5. 智能触发 → 分身能判断何时回复
6. RAG 集成 → 回复流程串联所有能力
7. 前端反馈 → 用户可见可控
```
