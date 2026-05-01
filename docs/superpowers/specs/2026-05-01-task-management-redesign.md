# 任务管理模块重新设计

## 背景

当前 `TaskManagementApp.vue` 是一个 915 行的单体组件，存在以下问题：

- **视觉平淡**：三列看板千篇一律，卡片没有视觉层次，优先级只是文字标签
- **信息密度低**：卡片只显示标题+描述+日期+优先级，缺少指派人、标签、进度等
- **交互原始**：只能点按钮改状态，没有拖拽、没有快捷操作、没有视图切换
- **无协作能力**：没有指派人、没有评论、不能从聊天创建任务
- **技术债**：使用 `any[]` 类型、API 内联在组件中、无状态管理

## 设计决策

| 维度 | 决策 |
|------|------|
| 改进方向 | 全面重新设计 |
| 用户场景 | 个人为主 + 轻协作 |
| 视觉风格 | 混合方案：极简骨架 + 丰富信息 + 柔和配色 |
| 架构方案 | 渐进式重构 |
| 视图关系 | 四种视图平等切换（看板/列表/日历/我的工作台） |
| 协作深度 | 轻量联动（从消息创建任务、指派、通知） |

## 视觉风格

混合风格三要素：

1. **极简骨架**：干净的线条、克制的间距、无多余装饰。stone 色系为基底，紫色（#8b5cf6）作为品牌强调色
2. **丰富信息**：卡片上展示优先级色条（左侧 3px border）、标签（彩色小标签）、指派人头像、子任务进度条、评论数
3. **柔和配色**：使用项目设计系统中的 stone 中性色系，避免高饱和度色彩，注重长时间使用的舒适度

### 任务卡片设计

- 左侧 3px 色条标识优先级：红色=高、黄色=中、蓝色=低
- 标题下方展示标签（彩色小标签）
- 底部左：指派人头像 + 截止日期 / 进度百分比
- 底部右：子任务数（📋 2/3）、评论数（💬 3）
- 进行中的任务显示进度条

### 已完成任务

- 整体透明度降低至 0.6
- 标题加删除线
- 显示完成日期

## 整体布局

```
+--------+------------------+------------------------------+
| 侧边栏  | 工具栏            | 主内容区                      |
| 200px  | 搜索+新建+批量操作  | 根据视图切换                   |
|        |                  |                              |
| 视图切换 |                  | 看板：三列拖拽卡片              |
| ├ 看板  |                  | 列表：可排序表格               |
| ├ 列表  |                  | 日历：按截止日期展示            |
| ├ 日历  |                  | 工作台：今日聚焦+快捷入口       |
| └ 工作台|                  |                              |
|        |                  |                              |
| 筛选    |                  |                              |
| ├ 高优先 |                  |                              |
| ├ 即将到期|                  |                              |
| └ 指派给我|                  |                              |
|        |                  |                              |
| 进度概览 |                  |                              |
| 本周68% |                  |                              |
+--------+------------------+------------------------------+
```

点击任务卡片时，右侧滑出 `TaskDetailPanel` 显示详情和编辑表单。

## 四种视图

### 看板视图（KanbanView）

- 三列：待办 / 进行中 / 已完成
- 支持拖拽卡片在列间移动（改变状态）
- 支持拖拽卡片在列内排序
- 列头显示任务计数
- 空列显示拖入提示

### 列表视图（ListView）

- 表格式展示所有任务
- 支持按优先级/状态/日期排序
- 支持按标签/指派人分组
- 支持批量选择和批量操作
- 行内快捷操作：勾选完成、改优先级

### 日历视图（CalendarView）

- 按截止日期在日历上展示任务
- 月视图为主，可切换周视图
- 拖拽任务可调整截止日期
- 今日/过期任务高亮

### 我的工作台（MyWorkspace）

- 今日待办：今天截止的任务
- 进行中：我正在做的任务
- 已指派给我：别人指派给我的任务
- 快捷入口：最近编辑的任务

## 任务组织能力

### 标签系统

- 每个任务可添加多个标签
- 标签有名称和颜色
- 预设标签：设计、后端、前端、重构、文档、Bug
- 支持自定义标签
- 可按标签筛选

### 子任务

- 任务下可创建子任务
- 子任务有标题和完成状态
- 子任务完成进度显示在父任务卡片上（📋 2/3）
- 子任务进度影响父任务进度条

### 任务模板

- 预设常用任务模板：Bug 修复、功能开发、代码评审、文档编写
- 模板预填标题、描述、标签、子任务
- 从模板创建时自动填充表单

### 重复任务

- 支持设置重复频率：每天/每周/每月
- 完成后自动创建下一期的任务
- 可设置结束条件：指定日期或次数

## 拖拽与快捷交互

### 拖拽

- 使用原生 HTML5 Drag and Drop API（无需引入额外库）
- 看板视图：拖拽卡片在列间移动改变状态
- 看板视图：拖拽卡片在列内改变排序
- 日历视图：拖拽任务调整截止日期
- 拖拽时显示半透明预览和放置指示器

### 右键菜单

- 任务卡片右键菜单：编辑、删除、改状态、改优先级、指派、复制
- 列头右键菜单：重命名、清空已完成
- 看板空白区域右键：新建任务

### 快捷键

- `⌘K`：聚焦搜索框
- `N`：新建任务
- `E`：编辑选中任务
- `Delete`：删除选中任务
- `1/2/3`：切换到看板/列表/日历视图
- `Space`：切换任务完成状态

### 批量操作

- 列表视图支持多选（Shift/Cmd 点击）
- 批量操作：改状态、改优先级、删除、添加标签

## 聊天协作联动

### 从消息创建任务

- 聊天消息右键菜单新增"创建为任务"选项
- 自动填充：标题=消息内容摘要、描述=完整消息内容、指派=消息发送者
- 创建后发送一条系统消息："已创建任务「xxx」"

### 任务指派

- 创建/编辑任务时可选择指派人（从联系人列表选择）
- 被指派人收到通知

### 任务通知

- 任务指派通知：`todo_assigned` 类型（已在 notificationMapper 中定义）
- 任务状态变更通知
- 任务即将到期提醒（提前 1 天）
- 通知点击跳转到任务详情

## 组件架构

### 组件树

```
TaskManagementApp
├── AppHeader（已有，复用）
├── TaskSidebar
│   ├── ViewSwitcher
│   ├── FilterGroup
│   └── ProgressSummary
├── TaskToolbar
│   ├── SearchBox
│   └── CreateTaskButton
├── 视图容器（动态切换）
│   ├── KanbanView
│   │   ├── KanbanColumn
│   │   │   └── TaskCard
│   │   └── ...
│   ├── ListView
│   │   ├── ListHeader
│   │   └── TaskRow
│   ├── CalendarView
│   └── MyWorkspace
├── TaskDetailPanel
│   ├── TaskForm
│   ├── TagSelector
│   ├── AssigneeSelector
│   └── SubTaskList
└── TaskCreateModal
```

### 文件组织

```
src/components/apps/task/
├── TaskManagementApp.vue
├── TaskSidebar.vue
├── TaskToolbar.vue
├── views/
│   ├── KanbanView.vue
│   ├── KanbanColumn.vue
│   ├── ListView.vue
│   ├── CalendarView.vue
│   └── MyWorkspace.vue
├── components/
│   ├── TaskCard.vue
│   ├── TaskRow.vue
│   ├── TaskDetailPanel.vue
│   ├── TaskCreateModal.vue
│   ├── TagSelector.vue
│   ├── AssigneeSelector.vue
│   └── SubTaskList.vue
└── index.ts

src/composables/
└── useTaskDragDrop.ts

src/stores/
└── task.ts

src/api/
└── task.ts

src/types/
└── task.ts
```

## 数据模型

```typescript
interface Task {
  id: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  dueDate: string | null
  tags: Tag[]
  assignee: User | null
  creator: User
  subTasks: SubTask[]
  commentCount: number
  position: number
  createdAt: string
  updatedAt: string
}

interface SubTask {
  id: string
  title: string
  completed: boolean
  position: number
}

interface Tag {
  id: string
  name: string
  color: string
}

type TaskStatus = 'todo' | 'in_progress' | 'completed'
type TaskPriority = 'low' | 'medium' | 'high'
type TaskView = 'kanban' | 'list' | 'calendar' | 'workspace'

interface TaskFilters {
  search: string
  priority: TaskPriority | null
  assigneeId: string | null
  tagId: string | null
  dueDateRange: { start: string; end: string } | null
}
```

## 状态管理

### Pinia Store: useTaskStore

**State:**
- `tasks: Task[]` — 所有任务
- `currentView: TaskView` — 当前视图
- `filters: TaskFilters` — 筛选条件
- `selectedTaskId: string | null` — 选中的任务
- `loading: boolean` — 加载状态

**Actions:**
- `fetchTasks()` — 获取任务列表
- `createTask(data)` — 创建任务
- `updateTask(id, data)` — 更新任务
- `deleteTask(id)` — 删除任务
- `updateStatus(id, status)` — 更新任务状态
- `reorderTask(id, position, status?)` — 拖拽排序

**Getters:**
- `filteredTasks` — 经过筛选的任务
- `todoTasks` — 待办任务
- `inProgressTasks` — 进行中任务
- `completedTasks` — 已完成任务
- `tasksByDate` — 按日期分组的任务
- `myTasks` — 指派给我的任务

## API 接口

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/v1/tasks` | 获取任务列表 |
| GET | `/api/v1/tasks/:id` | 获取任务详情 |
| POST | `/api/v1/tasks` | 创建任务 |
| PUT | `/api/v1/tasks/:id` | 更新任务 |
| PUT | `/api/v1/tasks/:id/reorder` | 拖拽排序 |
| PATCH | `/api/v1/tasks/:id/status` | 更新状态 |
| DELETE | `/api/v1/tasks/:id` | 删除任务 |

## 渐进式重构步骤

### 第一步：基础重构（解决技术债）

- 创建 `src/types/task.ts` 类型定义
- 创建 `src/api/task.ts` API 封装
- 创建 `src/stores/task.ts` Pinia Store
- 将 `TaskManagementApp.vue` 拆分为子组件
- 保持现有功能不变，仅重构内部结构

### 第二步：视觉升级 + 视图切换

- 重新设计 TaskCard 视觉（优先级色条、标签、头像、进度）
- 实现 TaskSidebar（视图切换 + 筛选）
- 实现 KanbanView / ListView / CalendarView / MyWorkspace
- 实现视图切换逻辑

### 第三步：交互增强

- 实现拖拽排序（useTaskDragDrop composable）
- 实现右键菜单
- 实现快捷键
- 实现批量操作
- 实现 TaskDetailPanel 侧滑面板

### 第四步：任务组织 + 协作联动

- 实现标签系统（TagSelector）
- 实现子任务（SubTaskList）
- 实现指派人选择（AssigneeSelector）
- 实现从消息创建任务
- 实现任务通知

## 错误处理

- API 调用失败时使用 QMessage 显示错误提示
- 拖拽操作失败时回滚到原始位置
- 网络断开时禁用拖拽和编辑操作，显示离线提示
- 表单验证：标题必填、截止日期不能早于今天

## 测试策略

- 单元测试：Pinia Store 的 actions 和 getters
- 组件测试：各视图组件的渲染和交互
- E2E 测试：完整的任务 CRUD 流程、拖拽操作、视图切换
