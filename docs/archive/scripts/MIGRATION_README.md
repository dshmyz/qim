# OIM 到 QIM 数据迁移脚本

这个Python脚本用于将数据从OIM数据库迁移到QIM系统。

## 功能特性

- ✅ 迁移用户数据（包括用户信息、密码、头像等）
- ✅ 迁移群组数据（包括群组信息、成员、权限等）
- ✅ 迁移联系人关系
- ✅ 迁移消息数据（单聊和群聊消息）
- ✅ 自动处理数据格式转换
- ✅ 完善的错误处理和日志记录
- ✅ 支持断点续传（通过ON DUPLICATE KEY UPDATE）

## 环境要求

- Python 3.7+
- MySQL 5.7+
- PyMySQL 1.1.0+

## 安装依赖

```bash
pip install -r migration_requirements.txt
```

## 配置说明

在运行脚本之前，需要修改 `migrate_oim_to_qim.py` 中的数据库配置：

```python
# OIM 数据库配置
oim_config = {
    'host': 'localhost',      # OIM 数据库主机
    'port': 3306,            # OIM 数据库端口
    'user': 'root',          # OIM 数据库用户名
    'password': 'your_password',  # OIM 数据库密码
    'database': 'oim_database',   # OIM 数据库名称
    'charset': 'utf8mb4'
}

# QIM 数据库配置
qim_config = {
    'host': 'localhost',      # QIM 数据库主机
    'port': 3306,            # QIM 数据库端口
    'user': 'root',          # QIM 数据库用户名
    'password': 'your_password',  # QIM 数据库密码
    'database': 'qim_server',     # QIM 数据库名称
    'charset': 'utf8mb4'
}
```

## 使用方法

### 1. 备份数据库

在执行迁移之前，强烈建议备份两个数据库：

```bash
# 备份 OIM 数据库
mysqldump -u root -p oim_database > oim_backup_$(date +%Y%m%d_%H%M%S).sql

# 备份 QIM 数据库
mysqldump -u root -p qim_server > qim_backup_$(date +%Y%m%d_%H%M%S).sql
```

### 2. 执行迁移

```bash
python migrate_oim_to_qim.py
```

### 3. 查看日志

脚本会输出详细的日志信息，包括：
- 数据库连接状态
- 找到的数据量
- 迁移进度
- 错误信息

## 迁移流程

脚本按照以下顺序执行迁移：

1. **用户数据迁移**
   - 从 `w_user` 表读取用户信息
   - 转换为 QIM 系统的用户格式
   - 插入到 `users` 表

2. **群组数据迁移**
   - 从 `w_group` 表读取群组信息
   - 创建对应的会话记录
   - 插入到 `groups` 表
   - 迁移群成员到 `conversation_members` 表

3. **联系人关系迁移**
   - 从 `w_contact_relation` 表读取联系人关系
   - 创建对应的会话记录
   - 更新会话成员信息

4. **消息数据迁移**
   - 从 `im_user_chat_content` 表读取单聊消息
   - 从 `im_group_chat_content` 表读取群聊消息
   - 转换消息内容
   - 插入到 `messages` 表

## 数据映射说明

### 用户类型映射

| OIM 类型 | QIM 类型 |
|---------|---------|
| 0 (普通) | user |
| 1 (管理员) | admin |
| 2 (超级管理员) | super_admin |

### 群成员角色映射

| OIM 角色 | QIM 角色 |
|---------|---------|
| 1 (群主) | owner |
| 2 (管理员) | admin |
| 3 (普通成员) | member |

### 用户状态映射

| OIM 状态 | QIM 状态 |
|---------|---------|
| isDisable = 0 | online |
| isDisable = 1 | offline |

## 注意事项

1. **数据完整性**
   - 迁移前请确保 OIM 数据库的数据完整性
   - 建议在测试环境先进行迁移测试

2. **用户ID映射**
   - 脚本通过用户名（account/email）来映射用户ID
   - 如果 OIM 和 QIM 系统的用户名不一致，需要修改 `_map_user_id` 方法

3. **消息内容**
   - 消息内容从 `im_user_chat_item` 和 `im_group_chat_item` 表中获取
   - 多个消息项会被合并为一个消息

4. **错误处理**
   - 单条数据迁移失败不会中断整个迁移过程
   - 所有错误都会记录到日志中
   - 迁移失败时会自动回滚

5. **性能考虑**
   - 对于大量数据，建议分批迁移
   - 可以在脚本中添加分页查询逻辑

## 自定义扩展

如果需要自定义迁移逻辑，可以修改以下方法：

- `_convert_user()`: 自定义用户数据转换
- `_convert_group()`: 自定义群组数据转换
- `_convert_user_message()`: 自定义单聊消息转换
- `_convert_group_message()`: 自定义群聊消息转换
- `_map_user_id()`: 自定义用户ID映射逻辑

## 故障排除

### 连接数据库失败

检查：
- 数据库服务是否启动
- 主机、端口、用户名、密码是否正确
- 数据库是否存在
- 用户是否有足够的权限

### 用户ID映射失败

检查：
- OIM 和 QIM 系统的用户名是否一致
- `_map_user_id` 方法的映射逻辑是否正确

### 消息迁移失败

检查：
- 消息内容表（`im_user_chat_item`、`im_group_chat_item`）是否存在
- 消息ID是否正确关联

## 技术支持

如果遇到问题，请检查：
1. 日志文件中的错误信息
2. 数据库配置是否正确
3. 数据表结构是否匹配

## 许可证

根据项目许可证使用。
