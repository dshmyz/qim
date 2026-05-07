# 数据迁移方案

## 概述

本文档描述从 `schema.sql`（旧版 OIM 系统）迁移数据到当前 QIM Server 数据库的方法。

## 数据库结构差异分析

### 旧版数据库（schema.sql）vs 新版数据库（qim-server）

| 旧版表名 | 新版对应 | 说明 |
|---------|---------|------|
| `w_user` | `users` | 用户表，字段差异较大 |
| `w_user_number` | - | 用户号生成器，新版不需要 |
| `w_user_head` | `users.avatar` | 头像合并到 users 表 |
| `w_user_security_question` | - | 密保问题，新版暂不支持 |
| `m_role` | `user_roles` | 角色表，结构简化 |
| `m_function` | - | 功能菜单，新版暂不支持 |
| `m_user_role` | `user_roles` | 用户角色关联 |
| `m_role_function` | - | 角色功能关联，新版暂不支持 |
| `w_group` | `groups` + `conversations` | 群组拆分为两组表 |
| `w_group_member` | `conversation_members` | 群成员 |
| `w_group_join_setting` | - | 入群设置，新版简化 |
| `w_group_join_apply` | - | 入群申请，新版简化 |
| `w_group_category` | - | 群分组，新版暂不支持 |
| `w_group_relation` | `conversation_members` | 群关系合并 |
| `w_group_notice` | `groups.announcement` | 群公告 |
| `im_user_chat_content` | `messages` | 单聊消息 |
| `im_group_chat_content` | `messages` | 群聊消息 |
| `im_recent_chat` | `conversation_sessions` | 最近会话 |
| `im_user_chat_unread` | `conversation_members.unread_count` | 未读数 |
| `im_group_chat_unread` | `conversation_members.unread_count` | 群未读数 |
| `im_user_chat_item` | `messages.content` | 消息内容项 |
| `im_group_chat_item` | `messages.content` | 群消息内容项 |
| `w_contact_relation` | - | 联系人关系，新版暂不支持 |
| `w_contact_category` | - | 联系人分组，新版暂不支持 |
| `w_contact_add_apply` | - | 好友申请，新版暂不支持 |
| `base_file_data` | `files` | 通用文件 |
| `base_image_data` | `files` | 图片合并到 files |
| `base_user_head_data` | `users.avatar` | 用户头像合并 |
| `base_group_head_data` | `groups.avatar` | 群头像合并 |
| `setting_app_client` | `client_versions` | 客户端版本 |
| `setting_multiple_online_strategy` | - | 多端在线策略，新版暂不支持 |
| `w_text_notice` | `notifications` | 通知 |
| `w_user_text_notice` | `notifications` | 用户通知 |
| `im_words_filter` | `sensitive_words` | 敏感词 |
| `server_action_info` | `operation_logs` | 服务端动作日志 |
| `server_type` | - | 服务器类型，新版暂不支持 |
| `server_address` | - | 服务器地址，新版暂不支持 |

## 迁移步骤

### 准备工作

1. **备份数据**
```bash
# 备份旧数据库
mysqldump -u root -p old_database > backup_old.sql

# 备份新版数据库（如果已存在）
mysqldump -u root -p new_database > backup_new.sql
```

2. **创建新版数据库**
```bash
mysql -u root -p
CREATE DATABASE qim_server CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE qim_server;
SOURCE ddl_mysql.sql;
```

### 分阶段迁移

#### 阶段 1：用户数据迁移

```sql
-- 1.1 迁移用户基础数据
INSERT INTO users (id, username, password_hash, nickname, avatar, type, signature, phone, email, status, created_at, updated_at)
SELECT
    CAST(id AS UNSIGNED) as id,
    account as username,
    password as password_hash,
    nickname,
    CASE WHEN avatar != '' THEN avatar ELSE NULL END as avatar,
    CASE type
        WHEN '0' THEN 'user'
        WHEN '1' THEN 'admin'
        WHEN '2' THEN 'system'
        ELSE 'user'
    END as type,
    signature,
    mobile as phone,
    email,
    CASE WHEN isDisable = 1 THEN 'disabled' ELSE 'offline' END as status,
    FROM_UNIXTIME(canceledTimestamp/1000) as created_at,
    NOW() as updated_at
FROM w_user
WHERE canceledTimestamp = 0;  -- 只迁移未注销用户

-- 1.2 迁移用户头像（如果新版 avatar 字段为空）
UPDATE users u
JOIN w_user_head uh ON u.id = CAST(uh.userId AS UNSIGNED)
SET u.avatar = uh.url
WHERE u.avatar IS NULL OR u.avatar = '';
```

#### 阶段 2：部门数据迁移

```sql
-- 2.1 迁移部门
INSERT INTO departments (id, name, parent_id, level, path, sort_order, created_at, updated_at)
SELECT
    CAST(SUBSTRING_INDEX(id, '-', -1) AS UNSIGNED) as id,
    name,
    NULLIF(CAST(SUBSTRING_INDEX(parentId, '-', -1) AS UNSIGNED), 0) as parent_id,
    level,
    path,
    sort_order,
    NOW() as created_at,
    NOW() as updated_at
FROM old_department;

-- 2.2 迁移部门员工关联
INSERT INTO department_employees (user_id, department_id, position, is_primary, created_at)
SELECT
    CAST(userId AS UNSIGNED),
    CAST(SUBSTRING_INDEX(departmentId, '-', -1) AS UNSIGNED),
    position,
    TRUE,
    NOW()
FROM w_user_department;
```

#### 阶段 3：群组数据迁移

```sql
-- 3.1 迁移群组创建会话
INSERT INTO conversations (id, type, name, avatar, creator_id, created_at, updated_at)
SELECT
    CAST(SUBSTRING_INDEX(id, '-', -1) AS UNSIGNED) as id,
    'group',
    name,
    CASE WHEN avatar != '' THEN avatar ELSE NULL END,
    CAST(creator_id AS UNSIGNED),
    NOW(),
    NOW()
FROM w_group;

-- 3.2 迁移群组详情
INSERT INTO groups (id, conversation_id, group_type, name, avatar, creator_id, announcement, created_at, updated_at)
SELECT
    CAST(SUBSTRING_INDEX(id, '-', -1) AS UNSIGNED),
    CAST(SUBSTRING_INDEX(id, '-', -1) AS UNSIGNED),
    'group',
    name,
    CASE WHEN avatar != '' THEN avatar ELSE NULL END,
    CAST(creator_id AS UNSIGNED),
    introduce,
    NOW(),
    NOW()
FROM w_group;

-- 3.3 迁移群成员
INSERT INTO conversation_members (conversation_id, user_id, role, joined_at)
SELECT
    CAST(SUBSTRING_INDEX(groupId, '-', -1) AS UNSIGNED),
    CAST(SUBSTRING_INDEX(userId, '-', -1) AS UNSIGNED),
    CASE position
        WHEN '1' THEN 'owner'
        WHEN '2' THEN 'admin'
        ELSE 'member'
    END,
    NOW()
FROM w_group_member;
```

#### 阶段 4：消息数据迁移

```sql
-- 4.1 迁移单聊消息
INSERT INTO messages (id, conversation_id, sender_id, type, content, created_at, updated_at)
SELECT
    CAST(SUBSTRING_INDEX(id, '-', -1) AS UNSIGNED),
    -- 需要根据 ownKey 计算 conversation_id
    (SELECT id FROM conversations c
     WHERE c.type = 'single'
     AND EXISTS (SELECT 1 FROM conversation_members cm1
                 WHERE cm1.conversation_id = c.id
                 AND cm1.user_id = CAST(SUBSTRING_INDEX(sendUserId, '-', -1) AS UNSIGNED))
     AND EXISTS (SELECT 1 FROM conversation_members cm2
                 WHERE cm2.conversation_id = c.id
                 AND cm2.user_id = CAST(SUBSTRING_INDEX(receiveUserId, '-', -1) AS UNSIGNED))
    ) as conversation_id,
    CAST(SUBSTRING_INDEX(sendUserId, '-', -1) AS UNSIGNED),
    'text',
    content,
    dateTime,
    dateTime
FROM im_user_chat_content
WHERE isDeleted = '0';

-- 4.2 迁移群聊消息
INSERT INTO messages (id, conversation_id, sender_id, type, content, created_at, updated_at)
SELECT
    CAST(SUBSTRING_INDEX(id, '-', -1) AS UNSIGNED),
    CAST(SUBSTRING_INDEX(groupId, '-', -1) AS UNSIGNED),
    CAST(SUBSTRING_INDEX(userId, '-', -1) AS UNSIGNED),
    'text',
    content,
    dateTime,
    dateTime
FROM im_group_chat_content
WHERE isDeleted = '0';
```

#### 阶段 5：文件数据迁移

```sql
-- 迁移通用文件
INSERT INTO files (id, user_id, name, original_name, size, mime_type, storage_path, checksum, created_at, updated_at)
SELECT
    CAST(SUBSTRING_INDEX(id, '-', -1) AS UNSIGNED),
    CAST(SUBSTRING_INDEX(userId, '-', -1) AS UNSIGNED),
    saveName,
    originalName,
    size,
    type,
    fullPathName,
    md5,
    NOW(),
    NOW()
FROM base_file_data;
```

#### 阶段 6：通知数据迁移

```sql
-- 迁移通知
INSERT INTO notifications (id, user_id, type, title, content, created_at)
SELECT
    CAST(SUBSTRING_INDEX(id, '-', -1) AS UNSIGNED),
    CAST(SUBSTRING_INDEX(userId, '-', -1) AS UNSIGNED),
    'system',
    title,
    content,
    FROM_UNIXTIME(timestamp/1000)
FROM w_text_notice;
```

#### 阶段 7：敏感词迁移

```sql
-- 迁移敏感词
INSERT INTO sensitive_words (id, word, level, created_at)
SELECT
    CAST(SUBSTRING_INDEX(id, '-', -1) AS UNSIGNED),
    words,
    CASE level
        WHEN 1 THEN 'low'
        WHEN 2 THEN 'medium'
        WHEN 3 THEN 'high'
        ELSE 'medium'
    END,
    NOW()
FROM im_words_filter;
```

### ID 映射策略

由于新旧系统使用不同的 ID 格式（旧版使用 VARCHAR 36 位 UUID，新版使用 INT 自增），需要进行 ID 映射。

**推荐方案：**

1. 创建 ID 映射表
```sql
CREATE TABLE id_mapping (
    old_id VARCHAR(40) NOT NULL,
    new_id INT UNSIGNED NOT NULL,
    table_name VARCHAR(50) NOT NULL,
    PRIMARY KEY (old_id, table_name)
);
```

2. 在迁移过程中填充映射表
3. 迁移完成后删除映射表

### 数据验证

```sql
-- 验证用户数量
SELECT 'w_user' as source, COUNT(*) as count FROM w_user
UNION ALL
SELECT 'users', COUNT(*) FROM users;

-- 验证群组数量
SELECT 'w_group' as source, COUNT(*) as count FROM w_group
UNION ALL
SELECT 'groups', COUNT(*) FROM groups;

-- 验证消息数量
SELECT 'im_user_chat_content' as source, COUNT(*) as count FROM im_user_chat_content
UNION ALL
SELECT 'messages', COUNT(*) FROM messages WHERE conversation_id IN (SELECT id FROM conversations WHERE type = 'single');

-- 验证文件数量
SELECT 'base_file_data' as source, COUNT(*) as count FROM base_file_data
UNION ALL
SELECT 'files', COUNT(*) FROM files;
```

## Python 迁移脚本示例

```python
#!/usr/bin/env python3
"""
数据迁移脚本 - 从旧版 OIM 迁移到 QIM Server
"""

import pymysql
import json
from datetime import datetime

class MigrationRunner:
    def __init__(self, old_db_config, new_db_config):
        self.old_db = pymysql.connect(**old_db_config)
        self.new_db = pymysql.connect(**new_db_config)

    def migrate_users(self):
        """迁移用户数据"""
        cursor = self.old_db.cursor(pymysql.cursors.DictCursor)
        cursor.execute("""
            SELECT id, account, password, nickname, avatar, mobile, email,
                   signature, type, isDisable, canceledTimestamp
            FROM w_user WHERE canceledTimestamp = 0
        """)

        new_cursor = self.new_db.cursor()
        for row in cursor.fetchall():
            new_id = self.get_next_id('users')
            self.id_mapping['users'][row['id']] = new_id

            new_cursor.execute("""
                INSERT INTO users (id, username, password_hash, nickname, avatar,
                                   phone, email, type, signature, status, created_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """, (
                new_id,
                row['account'],
                row['password'],
                row.get('nickname', ''),
                row.get('avatar', ''),
                row.get('mobile', ''),
                row.get('email', ''),
                'user' if row.get('type', '0') == '0' else 'admin',
                row.get('signature', ''),
                'disabled' if row.get('isDisable') == 1 else 'offline',
                datetime.fromtimestamp(row['canceledTimestamp']/1000) if row['canceledTimestamp'] > 0 else datetime.now()
            ))

        self.new_db.commit()

    def migrate_groups(self):
        """迁移群组数据"""
        cursor = self.old_db.cursor(pymysql.cursors.DictCursor)
        cursor.execute("SELECT * FROM w_group")

        new_cursor = self.new_db.cursor()
        for row in cursor.fetchall():
            new_id = self.get_next_id('groups')
            old_id = row['id']
            self.id_mapping['groups'][old_id] = new_id

            new_cursor.execute("""
                INSERT INTO conversations (id, type, name, avatar, creator_id, created_at)
                VALUES (%s, 'group', %s, %s, %s, %s)
            """, (new_id, row['name'], row.get('avatar', ''),
                  self.id_mapping['users'].get(row.get('creator_id', ''), 0),
                  datetime.now()))

            new_cursor.execute("""
                INSERT INTO groups (id, conversation_id, name, avatar, creator_id, announcement)
                VALUES (%s, %s, %s, %s, %s, %s)
            """, (new_id, new_id, row['name'], row.get('avatar', ''),
                  self.id_mapping['users'].get(row.get('creator_id', ''), 0),
                  row.get('introduce', '')))

        self.new_db.commit()

    def migrate_group_members(self):
        """迁移群组成员"""
        cursor = self.old_db.cursor(pymysql.cursors.DictCursor)
        cursor.execute("SELECT * FROM w_group_member")

        new_cursor = self.new_db.cursor()
        for row in cursor.fetchall():
            new_group_id = self.id_mapping['groups'].get(row['groupId'])
            new_user_id = self.id_mapping['users'].get(row['userId'])

            if new_group_id and new_user_id:
                role = 'owner' if row.get('position') == '1' else 'admin' if row.get('position') == '2' else 'member'
                new_cursor.execute("""
                    INSERT INTO conversation_members (conversation_id, user_id, role, joined_at)
                    VALUES (%s, %s, %s, %s)
                """, (new_group_id, new_user_id, role, datetime.now()))

        self.new_db.commit()

    def get_next_id(self, table):
        """获取下一个自增 ID"""
        cursor = self.new_db.cursor()
        cursor.execute(f"SELECT MAX(id) FROM {table}")
        result = cursor.fetchone()[0]
        return (result or 0) + 1

    def run(self):
        """执行完整迁移"""
        self.id_mapping = {
            'users': {},
            'groups': {},
        }

        print("开始迁移用户数据...")
        self.migrate_users()

        print("开始迁移群组数据...")
        self.migrate_groups()

        print("开始迁移群组成员...")
        self.migrate_group_members()

        print("迁移完成！")

if __name__ == '__main__':
    runner = MigrationRunner(
        old_db_config={
            'host': 'localhost',
            'user': 'root',
            'password': 'password',
            'database': 'old_oim'
        },
        new_db_config={
            'host': 'localhost',
            'user': 'root',
            'password': 'password',
            'database': 'qim_server'
        }
    )
    runner.run()
```

## 注意事项

1. **UUID 到 INT 的转换**：旧版使用 36 位 UUID，新版使用自增 INT。迁移时需要建立映射关系。

2. **消息 Key 计算**：旧版使用复杂的 `ownKey` 和 `messageKey`，新版使用简单的 conversation_id。需要根据发送者和接收者关系计算。

3. **时间戳处理**：旧版使用毫秒级时间戳，新版使用秒级。需要进行转换。

4. **头像处理**：旧版头像存储在多个表（w_user_head、base_user_head_data 等），新版统一存储在 users.avatar。

5. **未迁移的功能**：密保问题、功能菜单、入群设置等新版暂不支持的功能不会迁移。

6. **密码兼容性**：如果旧版密码使用 MD5 加密，而新版使用 bcrypt，需要考虑密码重置或重新加密。

## 回滚方案

如果迁移失败，执行以下步骤回滚：

```sql
-- 恢复到迁移前的备份
mysqldump -u root -p backup_old.sql | mysql -u root -p qim_server
```

建议在测试环境中先完成完整迁移测试，确认无误后再应用到生产环境。
