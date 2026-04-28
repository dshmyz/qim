# QIM 管理后台功能完善设计文档

**日期**: 2026-04-28  
**版本**: 1.0  
**作者**: AI Assistant  

## 一、背景与目标

### 1.1 项目背景
QIM 是一个企业内部办公即时通讯系统，管理后台需要补充核心功能以满足日常管理需求。

### 1.2 当前现状
管理后台已具备 19 个功能模块，覆盖用户管理、组织架构、消息管理、AI 功能、系统运维等方面。

### 1.3 核心问题
- 消息管理能力不足
- 文件管理缺失
- 系统监控不足
- 客户端管理缺失

### 1.4 设计目标
补充 IM 管理后台缺失的核心功能，提升管理效率和运营能力。

---

## 二、功能设计

### 2.1 消息管理增强

#### 2.1.1 消息搜索
**功能描述**: 支持按关键词、用户、时间范围、消息类型搜索消息

**核心功能**:
- 全文搜索（支持模糊匹配）
- 按发送者/接收者筛选
- 按时间范围筛选
- 按消息类型筛选（文本、图片、文件等）
- 按会话类型筛选（单聊、群聊、频道）
- 搜索结果高亮显示
- 导出搜索结果

**界面设计**:
- 搜索表单：关键词输入框、用户选择器、时间范围选择器、消息类型下拉框、会话类型下拉框
- 搜索结果列表：显示消息内容、发送者、接收者、时间、消息类型
- 操作按钮：导出结果、查看详情

#### 2.1.2 消息审计
**功能描述**: 查看和管理消息记录，支持审计需求

**核心功能**:
- 消息详情查看（发送时间、发送者、接收者、内容）
- 消息撤回记录
- 消息删除记录
- 消息转发记录
- 敏感消息标记
- 消息审计报告导出

**界面设计**:
- 消息列表：显示消息基本信息
- 详情面板：显示消息完整信息
- 操作记录：显示撤回、删除、转发记录
- 导出按钮：导出审计报告

#### 2.1.3 消息导出
**功能描述**: 批量导出消息记录

**核心功能**:
- 按时间范围导出
- 按用户导出
- 按会话导出
- 支持多种格式（CSV、Excel、JSON）
- 导出任务管理（异步导出）

**界面设计**:
- 导出表单：时间范围选择器、用户选择器、会话选择器、格式选择器
- 导出任务列表：显示任务状态、进度、下载链接

---

### 2.2 文件管理

#### 2.2.1 文件存储管理
**功能描述**: 管理 IM 系统中的文件存储

**核心功能**:
- 文件存储统计（总容量、已用容量、文件数量）
- 文件类型分布统计
- 存储空间趋势图
- 大文件排行
- 文件清理策略配置（按时间、大小、类型）

**界面设计**:
- 统计卡片：显示总容量、已用容量、文件数量
- 图表区域：显示存储空间趋势、文件类型分布
- 大文件列表：显示占用空间最大的文件
- 清理策略配置：设置自动清理规则

#### 2.2.2 文件访问日志
**功能描述**: 记录和查询文件访问记录

**核心功能**:
- 文件上传日志
- 文件下载日志
- 文件删除日志
- 按用户查询访问记录
- 按文件查询访问记录
- 异常访问告警（频繁下载、大量下载）

**界面设计**:
- 日志列表：显示文件名、操作类型、操作者、时间
- 筛选器：按用户、文件、操作类型筛选
- 告警列表：显示异常访问记录

#### 2.2.3 文件清理
**功能描述**: 自动清理过期或无用文件

**核心功能**:
- 清理规则配置（按时间、大小、类型）
- 手动清理
- 清理预览（查看将被清理的文件）
- 清理日志
- 清理统计

**界面设计**:
- 规则列表：显示已配置的清理规则
- 规则配置表单：设置清理条件
- 清理预览：显示将被清理的文件列表
- 清理日志：显示清理记录

---

### 2.3 系统监控

#### 2.3.1 服务器监控
**功能描述**: 实时监控服务器性能指标

**核心功能**:
- CPU 使用率监控
- 内存使用率监控
- 磁盘使用率监控
- 网络流量监控
- 实时图表展示
- 历史数据查询

**界面设计**:
- 实时监控面板：显示 CPU、内存、磁盘、网络的实时数据
- 图表区域：显示历史趋势图
- 时间选择器：选择查询时间范围

#### 2.3.2 服务状态监控
**功能描述**: 监控各服务组件的运行状态

**核心功能**:
- 消息服务状态
- 推送服务状态
- 文件服务状态
- 数据库服务状态
- 缓存服务状态
- 服务健康检查

**界面设计**:
- 服务列表：显示各服务的状态（正常、异常、警告）
- 详情面板：显示服务的详细信息
- 健康检查按钮：手动触发健康检查

#### 2.3.3 告警管理
**功能描述**: 配置和管理系统告警

**核心功能**:
- 告警规则配置（阈值、频率）
- 告警通知方式（邮件、短信、IM）
- 告警历史记录
- 告警处理状态跟踪
- 告警统计分析

**界面设计**:
- 告警规则列表：显示已配置的告警规则
- 规则配置表单：设置告警条件和通知方式
- 告警历史：显示历史告警记录
- 告警统计：显示告警趋势和分布

---

### 2.4 客户端管理

#### 2.4.1 客户端版本管理
**功能描述**: 管理客户端版本发布和更新

**核心功能**:
- 版本发布管理
- 版本更新日志
- 强制更新配置
- 灰度发布配置
- 版本分布统计
- 版本兼容性管理

**界面设计**:
- 版本列表：显示所有版本信息
- 发布表单：填写版本号、更新日志、下载链接
- 配置面板：设置强制更新、灰度发布
- 统计图表：显示版本分布

#### 2.4.2 崩溃日志管理
**功能描述**: 收集和分析客户端崩溃日志

**核心功能**:
- 崩溃日志收集
- 崩溃日志列表
- 崩溃趋势分析
- 崩溃原因分类
- 崩溃详情查看
- 崩溃统计报表

**界面设计**:
- 崩溃列表：显示崩溃时间、设备、版本、原因
- 详情面板：显示崩溃堆栈信息
- 趋势图表：显示崩溃趋势
- 统计报表：显示崩溃原因分布

#### 2.4.3 用户反馈管理
**功能描述**: 收集和管理用户反馈

**核心功能**:
- 反馈列表
- 反馈分类（Bug、建议、投诉）
- 反馈处理状态
- 反馈回复
- 反馈统计
- 反馈导出

**界面设计**:
- 反馈列表：显示反馈内容、类型、状态、时间
- 详情面板：显示反馈详情和处理记录
- 回复表单：填写回复内容
- 统计图表：显示反馈类型分布

---

## 三、技术架构

### 3.1 前端架构
- **框架**: Vue 3 + TypeScript
- **UI 组件库**: Element Plus
- **状态管理**: Pinia
- **路由**: Vue Router
- **HTTP 客户端**: Axios
- **图表库**: ECharts

### 3.2 后端 API 设计
需要新增以下 API 模块：

#### 消息管理 API
- `GET /api/messages/search` - 搜索消息
- `GET /api/messages/:id` - 获取消息详情
- `GET /api/messages/:id/audit` - 获取消息审计记录
- `POST /api/messages/export` - 导出消息
- `GET /api/messages/export/:taskId` - 获取导出任务状态

#### 文件管理 API
- `GET /api/files/statistics` - 获取文件统计
- `GET /api/files/access-logs` - 获取文件访问日志
- `POST /api/files/cleanup` - 清理文件
- `GET /api/files/cleanup/preview` - 预览清理文件

#### 系统监控 API
- `GET /api/monitor/server` - 获取服务器监控数据
- `GET /api/monitor/services` - 获取服务状态
- `GET /api/monitor/alerts` - 获取告警列表
- `POST /api/monitor/alerts` - 创建告警规则
- `PUT /api/monitor/alerts/:id` - 更新告警规则

#### 客户端管理 API
- `GET /api/versions` - 获取版本列表
- `POST /api/versions` - 发布新版本
- `PUT /api/versions/:id` - 更新版本配置
- `GET /api/crashes` - 获取崩溃日志
- `GET /api/feedbacks` - 获取用户反馈
- `PUT /api/feedbacks/:id` - 处理反馈

### 3.3 数据库设计
需要新增以下数据表：

#### 消息审计表
```sql
CREATE TABLE message_audit (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  message_id BIGINT NOT NULL,
  action VARCHAR(20) NOT NULL COMMENT 'recall, delete, forward',
  operator_id BIGINT NOT NULL,
  operator_name VARCHAR(100),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_message_id (message_id),
  INDEX idx_created_at (created_at)
);
```

#### 文件访问日志表
```sql
CREATE TABLE file_access_log (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  file_id BIGINT NOT NULL,
  file_name VARCHAR(255),
  action VARCHAR(20) NOT NULL COMMENT 'upload, download, delete',
  user_id BIGINT NOT NULL,
  user_name VARCHAR(100),
  ip_address VARCHAR(50),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_file_id (file_id),
  INDEX idx_user_id (user_id),
  INDEX idx_created_at (created_at)
);
```

#### 告警规则表
```sql
CREATE TABLE alert_rule (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(100) NOT NULL,
  metric VARCHAR(50) NOT NULL COMMENT 'cpu, memory, disk, network',
  condition VARCHAR(20) NOT NULL COMMENT 'gt, lt, eq',
  threshold DECIMAL(10, 2) NOT NULL,
  duration INT DEFAULT 60 COMMENT '持续时间（秒）',
  notify_methods JSON COMMENT '["email", "sms", "im"]',
  notify_targets JSON COMMENT '通知对象列表',
  enabled BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

#### 告警历史表
```sql
CREATE TABLE alert_history (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  rule_id BIGINT NOT NULL,
  metric VARCHAR(50) NOT NULL,
  value DECIMAL(10, 2) NOT NULL,
  status VARCHAR(20) NOT NULL COMMENT 'firing, resolved',
  handled_at TIMESTAMP NULL,
  handler_id BIGINT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_rule_id (rule_id),
  INDEX idx_created_at (created_at)
);
```

#### 客户端版本表
```sql
CREATE TABLE client_version (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  version VARCHAR(20) NOT NULL,
  platform VARCHAR(20) NOT NULL COMMENT 'windows, mac, linux, ios, android',
  changelog TEXT,
  download_url VARCHAR(500),
  force_update BOOLEAN DEFAULT FALSE,
  gray_release BOOLEAN DEFAULT FALSE,
  gray_ratio INT DEFAULT 0 COMMENT '灰度比例（0-100）',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_version (version),
  INDEX idx_platform (platform)
);
```

#### 崩溃日志表
```sql
CREATE TABLE crash_log (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  device_id VARCHAR(100),
  device_model VARCHAR(100),
  os_version VARCHAR(50),
  app_version VARCHAR(20),
  crash_reason TEXT,
  stack_trace TEXT,
  user_id BIGINT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_app_version (app_version),
  INDEX idx_created_at (created_at)
);
```

#### 用户反馈表
```sql
CREATE TABLE user_feedback (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  type VARCHAR(20) NOT NULL COMMENT 'bug, suggestion, complaint',
  title VARCHAR(200),
  content TEXT,
  screenshots JSON COMMENT '截图URL列表',
  status VARCHAR(20) DEFAULT 'pending' COMMENT 'pending, processing, resolved, closed',
  handler_id BIGINT,
  reply TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_user_id (user_id),
  INDEX idx_status (status),
  INDEX idx_created_at (created_at)
);
```

---

## 四、实施计划

### 4.1 开发优先级
1. **P0 - 核心功能**
   - 消息搜索
   - 服务器监控
   - 客户端版本管理

2. **P1 - 重要功能**
   - 消息审计
   - 文件存储管理
   - 服务状态监控
   - 崩溃日志管理

3. **P2 - 增强功能**
   - 消息导出
   - 文件访问日志
   - 文件清理
   - 告警管理
   - 用户反馈管理

### 4.2 开发周期估算
- **P0 功能**: 2-3 周
- **P1 功能**: 3-4 周
- **P2 功能**: 2-3 周
- **总计**: 7-10 周

---

## 五、风险与挑战

### 5.1 技术风险
- **消息搜索性能**: 大量消息的全文搜索可能影响性能，需要优化索引和分页
- **实时监控**: 需要考虑监控数据的采集频率和存储策略
- **文件清理**: 需要确保清理逻辑不会误删重要文件

### 5.2 业务风险
- **权限控制**: 新增功能需要合理的权限控制，避免数据泄露
- **用户体验**: 需要确保新功能的操作流程简洁易用
- **兼容性**: 需要考虑与现有功能的兼容性

---

## 六、后续规划

### 6.1 短期优化
- 优化搜索性能，支持更复杂的搜索条件
- 完善监控告警机制，支持更多告警渠道
- 增强数据分析能力，提供更多维度的统计报表

### 6.2 长期规划
- 引入 AI 辅助分析，自动识别异常模式
- 支持自定义仪表盘，用户可配置关注的指标
- 集成第三方监控工具，提供更全面的监控能力
