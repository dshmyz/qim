# AI 总结保存到笔记功能实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 在 AI 总结面板中添加"保存到笔记"按钮，允许用户将 AI 生成的会话摘要一键保存到笔记系统

**架构：** 在 AISummaryPanel.vue 组件中添加保存按钮，调用现有的 useNotes composable 创建笔记，使用时间范围作为标题，摘要内容作为笔记内容

**技术栈：** Vue 3 + TypeScript

---

## 文件结构

| 文件 | 职责 | 操作 |
|------|------|------|
| `qim-client/src/components/ai/AISummaryPanel.vue` | AI 总结面板组件 | 修改：添加保存到笔记功能 |

---

## 任务 1：添加保存到笔记功能

**文件：**
- 修改：`qim-client/src/components/ai/AISummaryPanel.vue`

- [ ] **步骤 1：导入必要的依赖**

在 `<script setup>` 部分添加导入：

```typescript
import { useNotes } from '../../composables/useNotes'
import { QMessage } from '../../utils/message'
```

- [ ] **步骤 2：添加保存状态和方法**

在 `<script setup>` 部分添加：

```typescript
const { createNote } = useNotes()
const saving = ref(false)

const saveToNote = async () => {
  if (!summaryData.value || saving.value) return
  
  saving.value = true
  try {
    const result = await createNote({
      title: summaryData.value.time_range || '会话摘要',
      content: summaryData.value.summary,
      type: 'note'
    })
    
    if (result) {
      QMessage.success('已保存到笔记')
    } else {
      QMessage.error('保存失败，请稍后重试')
    }
  } catch (error) {
    QMessage.error('保存失败，请稍后重试')
  } finally {
    saving.value = false
  }
}
```

- [ ] **步骤 3：添加保存按钮到模板**

在 `<template>` 的 `.summary-actions` div 中添加按钮：

```html
<div class="summary-actions">
  <button @click="copySummary">复制摘要</button>
  <button @click="exportSummary">导出 Markdown</button>
  <button @click="saveToNote" :disabled="saving">
    {{ saving ? '保存中...' : '保存到笔记' }}
  </button>
</div>
```

- [ ] **步骤 4：验证编译**

运行：`cd qim-client && npm run typecheck`
预期：无类型错误

- [ ] **步骤 5：手动测试功能**

测试清单：
- [ ] 打开 AI 总结面板
- [ ] 等待总结生成完成
- [ ] 点击"保存到笔记"按钮
- [ ] 验证按钮显示"保存中..."状态
- [ ] 验证成功提示"已保存到笔记"出现
- [ ] 打开笔记应用，验证笔记已创建
- [ ] 验证笔记标题为时间范围
- [ ] 验证笔记内容为摘要内容

- [ ] **步骤 6：Commit**

```bash
git add qim-client/src/components/ai/AISummaryPanel.vue
git commit -m "feat(ai): 添加保存总结到笔记功能

- 在 AI 总结面板添加\"保存到笔记\"按钮
- 使用时间范围作为笔记标题
- 添加保存状态和错误处理
- 显示成功/失败提示消息"
```

---

## 完成标准

- [ ] 用户可以在 AI 总结面板看到"保存到笔记"按钮
- [ ] 点击按钮可以将摘要保存为笔记
- [ ] 笔记标题正确显示时间范围
- [ ] 笔记内容正确显示摘要内容
- [ ] 保存过程中按钮显示加载状态
- [ ] 保存成功显示成功提示
- [ ] 保存失败显示错误提示
- [ ] 代码通过类型检查
- [ ] 功能通过手动测试
