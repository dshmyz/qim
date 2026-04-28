#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
OIM Server -> QIM Server 数据迁移脚本

架构差异说明:
1. 主键类型：老系统使用 VARCHAR(40) UUID，新系统使用 INT AUTO_INCREMENT
2. 消息内容：老系统内容在 im_user_chat_item/im_group_chat_item 表中，
   按 contentId 分组，每个 item 对应一条新消息（方案A: 拆分）
3. 会话模型：老系统分离单聊和群聊，新系统统一使用 conversations 表
   - 单聊: type='single'
   - 群聊: type='group'
4. 未读计数：老系统独立表存储，新系统放在 conversation_members.unread_count

依赖:
    pip install pymysql

使用方法:
    python migrate.py --source-host=localhost --source-db=oim_db \
                      --target-host=localhost --target-db=qim_server \
                      --source-user=root --source-pass=xxx \
                      --target-user=root --target-pass=xxx

    测试模式（不实际写入）:
    python migrate.py --source-host=localhost --source-db=oim_db \
                      --target-host=localhost --target-db=qim_server \
                      --source-user=root --source-pass=xxx \
                      --target-user=root --target-pass=xxx \
                      --dry-run
"""

import argparse
import pymysql
import json
import logging
from datetime import datetime
from typing import Dict, List, Optional

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger(__name__)

MESSAGE_TYPE_MAP = {
    'text': 'text',
    'face': 'emoji',
    'image': 'image',
    'file': 'file',
    'audio': 'audio',
    'video': 'video',
    'url': 'link',
    'position': 'location',
    'at': 'text',
}


class MigrationEngine:
    def __init__(self, source_config: Dict, target_config: Dict, dry_run: bool = False):
        self.dry_run = dry_run

        self.source_conn = pymysql.connect(
            host=source_config['host'],
            port=source_config.get('port', 3306),
            user=source_config['user'],
            password=source_config['password'],
            database=source_config['database'],
            charset='utf8mb4',
            cursorclass=pymysql.cursors.DictCursor
        )

        if not dry_run:
            self.target_conn = pymysql.connect(
                host=target_config['host'],
                port=target_config.get('port', 3306),
                user=target_config['user'],
                password=target_config['password'],
                database=target_config['database'],
                charset='utf8mb4',
                cursorclass=pymysql.cursors.DictCursor
            )
        else:
            self.target_conn = None

        self.user_id_map: Dict[str, int] = {}
        self.group_id_map: Dict[str, int] = {}
        self.conversation_id_map: Dict[str, int] = {}

        self.stats = {
            'users': 0,
            'groups': 0,
            'conversations': 0,
            'messages': 0,
            'members': 0,
            'sessions': 0,
        }

    def migrate_all(self):
        if self.dry_run:
            logger.warning("【测试模式】不会实际写入数据")

        logger.info("开始迁移数据...")
        logger.info("=" * 50)

        logger.info("\n[1/8] 迁移用户数据...")
        self.migrate_users()
        logger.info(f"  完成: {self.stats['users']} 个用户")

        logger.info("\n[2/8] 迁移群组数据...")
        self.migrate_groups()
        logger.info(f"  完成: {self.stats['groups']} 个群组")

        logger.info("\n[3/8] 创建单聊会话...")
        self.migrate_private_conversations()
        logger.info(f"  完成: {self.stats['conversations']} 个单聊会话")

        logger.info("\n[4/8] 迁移群成员...")
        self.migrate_members()
        logger.info(f"  完成: {self.stats['members']} 条成员关系")

        logger.info("\n[5/8] 迁移消息数据（方案A: 每个item拆分为独立消息）...")
        self.migrate_messages()
        logger.info(f"  完成: {self.stats['messages']} 条消息")

        logger.info("\n[6/8] 迁移未读计数...")
        self.migrate_unread_counts()
        logger.info(f"  完成: 更新未读计数")

        logger.info("\n[7/8] 迁移会话记录...")
        self.migrate_sessions()
        logger.info(f"  完成: {self.stats['sessions']} 条会话记录")

        logger.info("\n[8/8] 迁移系统通知...")
        self.migrate_notifications()
        logger.info(f"  完成: 迁移通知")

        logger.info("\n" + "=" * 50)
        logger.info("迁移完成！统计信息:")
        for key, value in self.stats.items():
            logger.info(f"  {key}: {value}")

    def migrate_users(self):
        with self.source_conn.cursor() as cursor:
            cursor.execute("SELECT * FROM w_user ORDER BY id")
            old_users = cursor.fetchall()

        logger.info(f"  找到 {len(old_users)} 个用户")

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将迁移 {len(old_users)} 个用户")
            for old_user in old_users[:3]:
                logger.info(f"    - {old_user.get('account') or old_user.get('number')} -> {old_user.get('nickname', '')}")
            if len(old_users) > 3:
                logger.info(f"    ... 及其他 {len(old_users) - 3} 个用户")
            self.stats['users'] = len(old_users)
            return

        with self.target_conn.cursor() as target_cursor:
            for idx, old_user in enumerate(old_users, 1):
                try:
                    new_user_data = {
                        'username': old_user['account'] if old_user.get('account') else f"user_{old_user.get('number', '')}",
                        'password_hash': old_user.get('password', ''),
                        'nickname': old_user.get('nickname', ''),
                        'avatar': old_user.get('avatar', ''),
                        'signature': old_user.get('signature', ''),
                        'phone': old_user.get('mobile', ''),
                        'email': old_user.get('email', ''),
                        'status': 'offline',
                        'ip': '',
                        'two_factor_enabled': False,
                        'created_at': self._timestamp_to_datetime(old_user.get('onlineTimestamp', 0)),
                        'updated_at': datetime.now(),
                        'deleted_at': self._timestamp_to_datetime(old_user.get('canceledTimestamp', 0)) if old_user.get('canceledTimestamp', 0) > 0 else None,
                    }

                    target_cursor.execute("""
                        INSERT INTO users (
                            username, password_hash, nickname, avatar, signature,
                            phone, email, status, ip, two_factor_enabled,
                            created_at, updated_at, deleted_at
                        ) VALUES (
                            %(username)s, %(password_hash)s, %(nickname)s, %(avatar)s, %(signature)s,
                            %(phone)s, %(email)s, %(status)s, %(ip)s, %(two_factor_enabled)s,
                            %(created_at)s, %(updated_at)s, %(deleted_at)s
                        )
                    """, new_user_data)

                    new_id = target_cursor.lastrowid
                    self.user_id_map[old_user['id']] = new_id
                    self.stats['users'] += 1

                    if idx % 100 == 0:
                        logger.info(f"  已迁移 {idx}/{len(old_users)} 用户")
                except Exception as e:
                    logger.error(f"  用户迁移失败: id={old_user['id']}, error={e}")

            self.target_conn.commit()

    def migrate_groups(self):
        with self.source_conn.cursor() as cursor:
            cursor.execute("SELECT * FROM w_group ORDER BY id")
            old_groups = cursor.fetchall()

        logger.info(f"  找到 {len(old_groups)} 个群组")

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将迁移 {len(old_groups)} 个群组")
            for old_group in old_groups[:3]:
                logger.info(f"    - {old_group.get('name')} (id={old_group['id']})")
            if len(old_groups) > 3:
                logger.info(f"    ... 及其他 {len(old_groups) - 3} 个群组")
            self.stats['groups'] = len(old_groups)
            return

        with self.target_conn.cursor() as target_cursor:
            for idx, old_group in enumerate(old_groups, 1):
                try:
                    creator_id = self._get_group_creator_id(old_group['id'])

                    conversation_data = {
                        'type': 'group',
                        'name': old_group['name'],
                        'avatar': old_group.get('avatar', ''),
                        'creator_id': creator_id,
                        'last_message_id': None,
                        'last_message_at': None,
                        'created_at': datetime.now(),
                        'updated_at': datetime.now(),
                    }

                    target_cursor.execute("""
                        INSERT INTO conversations (
                            type, name, avatar, creator_id, last_message_id,
                            last_message_at, created_at, updated_at
                        ) VALUES (
                            %(type)s, %(name)s, %(avatar)s, %(creator_id)s, %(last_message_id)s,
                            %(last_message_at)s, %(created_at)s, %(updated_at)s
                        )
                    """, conversation_data)

                    conversation_id = target_cursor.lastrowid

                    groups_data = {
                        'conversation_id': conversation_id,
                        'name': old_group['name'],
                        'avatar': old_group.get('avatar', ''),
                        'creator_id': creator_id,
                        'max_members': old_group.get('maxCount', 500),
                        'current_members': 0,
                        'announcement': old_group.get('intro', ''),
                        'join_permission': 'invite_only',
                        'is_muted': False,
                        'is_disbanded': False,
                        'created_at': datetime.now(),
                        'updated_at': datetime.now(),
                    }

                    target_cursor.execute("""
                        INSERT INTO groups (
                            conversation_id, name, avatar, creator_id, max_members,
                            current_members, announcement, join_permission, is_muted,
                            is_disbanded, created_at, updated_at
                        ) VALUES (
                            %(conversation_id)s, %(name)s, %(avatar)s, %(creator_id)s, %(max_members)s,
                            %(current_members)s, %(announcement)s, %(join_permission)s, %(is_muted)s,
                            %(is_disbanded)s, %(created_at)s, %(updated_at)s
                        )
                    """, groups_data)

                    self.group_id_map[old_group['id']] = conversation_id
                    self.conversation_id_map[f"group_{old_group['id']}"] = conversation_id
                    self.stats['groups'] += 1

                    if idx % 100 == 0:
                        logger.info(f"  已迁移 {idx}/{len(old_groups)} 群组")
                except Exception as e:
                    logger.error(f"  群组迁移失败: id={old_group['id']}, error={e}")

            self.target_conn.commit()

    def _get_group_creator_id(self, group_id: str) -> Optional[int]:
        with self.source_conn.cursor() as cursor:
            cursor.execute(
                "SELECT userId FROM w_group_member WHERE groupId=%s AND position='1' LIMIT 1",
                (group_id,)
            )
            result = cursor.fetchone()
            if result:
                return self.user_id_map.get(result['userId'])
        return None

    def migrate_private_conversations(self):
        with self.source_conn.cursor() as cursor:
            cursor.execute("""
                SELECT DISTINCT 
                    LEAST(sendUserId, receiveUserId) as user1,
                    GREATEST(sendUserId, receiveUserId) as user2
                FROM im_user_chat_content
                ORDER BY user1, user2
            """)
            chat_items = cursor.fetchall()

        logger.info(f"  找到 {len(chat_items)} 个单聊会话")

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将迁移 {len(chat_items)} 个单聊会话")
            for item in chat_items[:3]:
                logger.info(f"    - {item['user1']} <-> {item['user2']}")
            if len(chat_items) > 3:
                logger.info(f"    ... 及其他 {len(chat_items) - 3} 个会话")
            self.stats['conversations'] = len(chat_items)
            return

        with self.target_conn.cursor() as target_cursor:
            for idx, item in enumerate(chat_items, 1):
                try:
                    user1_new = self.user_id_map.get(item['user1'])
                    user2_new = self.user_id_map.get(item['user2'])

                    if not user1_new or not user2_new:
                        logger.warning(f"  跳过: 用户ID映射不存在 (user1={item['user1']}, user2={item['user2']})")
                        continue

                    conversation_data = {
                        'type': 'single',
                        'name': None,
                        'avatar': None,
                        'creator_id': user1_new,
                        'last_message_id': None,
                        'last_message_at': None,
                        'created_at': datetime.now(),
                        'updated_at': datetime.now(),
                    }

                    target_cursor.execute("""
                        INSERT INTO conversations (
                            type, name, avatar, creator_id, last_message_id,
                            last_message_at, created_at, updated_at
                        ) VALUES (
                            %(type)s, %(name)s, %(avatar)s, %(creator_id)s, %(last_message_id)s,
                            %(last_message_at)s, %(created_at)s, %(updated_at)s
                        )
                    """, conversation_data)

                    new_conv_id = target_cursor.lastrowid
                    conv_key = f"single_{item['user1']}_{item['user2']}"
                    self.conversation_id_map[conv_key] = new_conv_id
                    self.stats['conversations'] += 1

                    target_cursor.execute("""
                        INSERT INTO conversation_members (
                            conversation_id, user_id, role, unread_count,
                            muted, last_read_at, joined_at
                        ) VALUES (%s, %s, 'member', 0, FALSE, NULL, NOW())
                    """, (new_conv_id, user1_new))

                    target_cursor.execute("""
                        INSERT INTO conversation_members (
                            conversation_id, user_id, role, unread_count,
                            muted, last_read_at, joined_at
                        ) VALUES (%s, %s, 'member', 0, FALSE, NULL, NOW())
                    """, (new_conv_id, user2_new))

                    if idx % 100 == 0:
                        logger.info(f"  已迁移 {idx}/{len(chat_items)} 会话")
                except Exception as e:
                    logger.error(f"  单聊会话迁移失败: item={item}, error={e}")

            self.target_conn.commit()

    def migrate_members(self):
        with self.source_conn.cursor() as cursor:
            cursor.execute("SELECT * FROM w_group_member ORDER BY groupId, id")
            members = cursor.fetchall()

        logger.info(f"  找到 {len(members)} 个群成员")

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将迁移 {len(members)} 个群成员")
            self.stats['members'] = len(members)
            return

        with self.target_conn.cursor() as target_cursor:
            for idx, member in enumerate(members, 1):
                try:
                    conv_id = self.conversation_id_map.get(f"group_{member['groupId']}")
                    user_id = self.user_id_map.get(member['userId'])

                    if not conv_id or not user_id:
                        continue

                    role_map = {'1': 'owner', '2': 'admin', '3': 'member'}
                    role = role_map.get(member['position'], 'member')

                    target_cursor.execute("""
                        SELECT id FROM conversation_members 
                        WHERE conversation_id=%s AND user_id=%s
                    """, (conv_id, user_id))

                    if target_cursor.fetchone():
                        continue

                    target_cursor.execute("""
                        INSERT INTO conversation_members (
                            conversation_id, user_id, role, unread_count,
                            muted, last_read_at, joined_at
                        ) VALUES (%s, %s, %s, 0, FALSE, NULL, NOW())
                    """, (conv_id, user_id, role))

                    self.stats['members'] += 1

                    if idx % 100 == 0:
                        logger.info(f"  已迁移 {idx}/{len(members)} 成员")
                except Exception as e:
                    logger.error(f"  成员迁移失败: id={member.get('id')}, error={e}")

            self.target_conn.commit()

    def migrate_messages(self):
        self._migrate_private_messages()
        self._migrate_group_messages()

    def _migrate_private_messages(self):
        with self.source_conn.cursor() as cursor:
            cursor.execute("""
                SELECT * FROM im_user_chat_item 
                ORDER BY contentId, sort, section
            """)
            items = cursor.fetchall()

        logger.info(f"  [单聊] 找到 {len(items)} 条消息项")

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将迁移 {len(items)} 条单聊消息（每个item一条消息）")
            self.stats['messages'] += len(items)
            return

        batch_size = 1000
        batch = []
        processed_content_ids = set()

        for idx, item in enumerate(items, 1):
            try:
                conv_id = self._get_private_conversation_id(
                    item['sendUserId'], item['receiveUserId']
                )
                sender_id = self.user_id_map.get(item['sendUserId'])

                if not conv_id or not sender_id:
                    continue

                message_type = MESSAGE_TYPE_MAP.get(item['type'], 'text')
                content = item.get('filterValue') or item.get('originalValue') or ''

                message_data = {
                    'conversation_id': conv_id,
                    'sender_id': sender_id,
                    'type': message_type,
                    'content': content,
                    'quoted_message_id': None,
                    'is_recalled': False,
                    'is_read': False,
                    'recalled_at': None,
                    'created_at': self._datetime_or_default(item.get('dateTime')),
                    'updated_at': datetime.now(),
                    'deleted_at': None,
                }

                batch.append(message_data)

                if len(batch) >= batch_size:
                    self._batch_insert_messages(batch)
                    batch = []

                if idx % 1000 == 0:
                    logger.info(f"  [单聊] 已处理 {idx}/{len(items)} 条消息")
            except Exception as e:
                logger.error(f"  [单聊] 消息处理失败: contentId={item.get('contentId')}, error={e}")

        if batch:
            self._batch_insert_messages(batch)

    def _migrate_group_messages(self):
        with self.source_conn.cursor() as cursor:
            cursor.execute("""
                SELECT * FROM im_group_chat_item 
                ORDER BY contentId, sort, section
            """)
            items = cursor.fetchall()

        logger.info(f"  [群聊] 找到 {len(items)} 条消息项")

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将迁移 {len(items)} 条群聊消息（每个item一条消息）")
            self.stats['messages'] += len(items)
            return

        batch_size = 1000
        batch = []

        for idx, item in enumerate(items, 1):
            try:
                conv_id = self.conversation_id_map.get(f"group_{item['groupId']}")
                sender_id = self.user_id_map.get(item['userId'])

                if not conv_id or not sender_id:
                    continue

                message_type = MESSAGE_TYPE_MAP.get(item['type'], 'text')
                content = item.get('filterValue') or item.get('originalValue') or ''

                message_data = {
                    'conversation_id': conv_id,
                    'sender_id': sender_id,
                    'type': message_type,
                    'content': content,
                    'quoted_message_id': None,
                    'is_recalled': False,
                    'is_read': False,
                    'recalled_at': None,
                    'created_at': self._datetime_or_default(item.get('dateTime')),
                    'updated_at': datetime.now(),
                    'deleted_at': None,
                }

                batch.append(message_data)

                if len(batch) >= batch_size:
                    self._batch_insert_messages(batch)
                    batch = []

                if idx % 1000 == 0:
                    logger.info(f"  [群聊] 已处理 {idx}/{len(items)} 条消息")
            except Exception as e:
                logger.error(f"  [群聊] 消息处理失败: contentId={item.get('contentId')}, error={e}")

        if batch:
            self._batch_insert_messages(batch)

    def _batch_insert_messages(self, batch: List[Dict]):
        with self.target_conn.cursor() as cursor:
            for msg_data in batch:
                cursor.execute("""
                    INSERT INTO messages (
                        conversation_id, sender_id, type, content,
                        quoted_message_id, is_recalled, is_read, recalled_at,
                        created_at, updated_at, deleted_at
                    ) VALUES (
                        %(conversation_id)s, %(sender_id)s, %(type)s, %(content)s,
                        %(quoted_message_id)s, %(is_recalled)s, %(is_read)s, %(recalled_at)s,
                        %(created_at)s, %(updated_at)s, %(deleted_at)s
                    )
                """, msg_data)
                self.stats['messages'] += 1
            self.target_conn.commit()

    def _get_private_conversation_id(self, user1_id: str, user2_id: str) -> Optional[int]:
        key1 = f"single_{user1_id}_{user2_id}"
        key2 = f"single_{user2_id}_{user1_id}"
        return self.conversation_id_map.get(key1) or self.conversation_id_map.get(key2)

    def migrate_unread_counts(self):
        self._migrate_private_unread()
        self._migrate_group_unread()

    def _migrate_private_unread(self):
        with self.source_conn.cursor() as cursor:
            cursor.execute("SELECT * FROM im_user_chat_unread WHERE unread > 0")
            unread_records = cursor.fetchall()

        logger.info(f"  [单聊] 找到 {len(unread_records)} 条未读记录")

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将更新 {len(unread_records)} 条未读计数")
            return

        with self.target_conn.cursor() as target_cursor:
            updated = 0
            for record in unread_records:
                try:
                    conv_id = self._get_private_conversation_id(
                        record['userId'], record['targetUserId']
                    )
                    user_id = self.user_id_map.get(record['userId'])

                    if not conv_id or not user_id:
                        continue

                    target_cursor.execute("""
                        SELECT id FROM conversation_members 
                        WHERE conversation_id = %s AND user_id = %s
                    """, (conv_id, user_id))

                    if target_cursor.fetchone():
                        target_cursor.execute("""
                            UPDATE conversation_members 
                            SET unread_count = %s 
                            WHERE conversation_id = %s AND user_id = %s
                        """, (record['unread'], conv_id, user_id))
                        updated += 1
                    else:
                        logger.warning(f"  [单聊] 未找到会话成员: conv_id={conv_id}, user_id={user_id}")
                except Exception as e:
                    logger.error(f"  [单聊] 未读计数更新失败: error={e}")

            self.target_conn.commit()
            logger.info(f"  [单聊] 成功更新 {updated} 条未读计数")

    def _migrate_group_unread(self):
        with self.source_conn.cursor() as cursor:
            cursor.execute("SELECT * FROM im_group_chat_unread WHERE unread > 0")
            unread_records = cursor.fetchall()

        logger.info(f"  [群聊] 找到 {len(unread_records)} 条未读记录")

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将更新 {len(unread_records)} 条未读计数")
            return

        with self.target_conn.cursor() as target_cursor:
            updated = 0
            for record in unread_records:
                try:
                    conv_id = self.conversation_id_map.get(f"group_{record['groupId']}")
                    user_id = self.user_id_map.get(record['userId'])

                    if not conv_id or not user_id:
                        continue

                    target_cursor.execute("""
                        SELECT id FROM conversation_members 
                        WHERE conversation_id = %s AND user_id = %s
                    """, (conv_id, user_id))

                    if target_cursor.fetchone():
                        target_cursor.execute("""
                            UPDATE conversation_members 
                            SET unread_count = %s 
                            WHERE conversation_id = %s AND user_id = %s
                        """, (record['unread'], conv_id, user_id))
                        updated += 1
                except Exception as e:
                    logger.error(f"  [群聊] 未读计数更新失败: error={e}")

            self.target_conn.commit()
            logger.info(f"  [群聊] 成功更新 {updated} 条未读计数")

    def migrate_sessions(self):
        with self.source_conn.cursor() as cursor:
            cursor.execute("SELECT * FROM im_recent_chat ORDER BY timestamp DESC")
            sessions = cursor.fetchall()

        logger.info(f"  找到 {len(sessions)} 个会话记录")

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将迁移 {len(sessions)} 个会话记录")
            self.stats['sessions'] = len(sessions)
            return

        with self.target_conn.cursor() as target_cursor:
            for idx, session in enumerate(sessions, 1):
                try:
                    user_id = self.user_id_map.get(session['userId'])
                    if not user_id:
                        continue

                    conv_id = None
                    if session['type'] == '2':
                        conv_id = self.conversation_id_map.get(f"group_{session['chatId']}")
                    elif session['type'] == '1':
                        conv_id = self._get_private_conversation_id(session['userId'], session['chatId'])

                    if not conv_id:
                        continue

                    target_cursor.execute("""
                        SELECT id FROM conversation_sessions 
                        WHERE user_id=%s AND conversation_id=%s
                    """, (user_id, conv_id))

                    if target_cursor.fetchone():
                        continue

                    target_cursor.execute("""
                        INSERT INTO conversation_sessions (
                            user_id, conversation_id, is_pinned, pinned_at,
                            last_visited_at, created_at
                        ) VALUES (%s, %s, FALSE, NULL, %s, NOW())
                    """, (user_id, conv_id, self._datetime_or_default(session.get('dateTime'))))

                    self.stats['sessions'] += 1

                    if idx % 100 == 0:
                        logger.info(f"  已迁移 {idx}/{len(sessions)} 会话记录")
                except Exception as e:
                    logger.error(f"  会话记录迁移失败: id={session.get('id')}, error={e}")

            self.target_conn.commit()

    def migrate_notifications(self):
        self._migrate_system_messages()
        self._migrate_user_notifications()

    def _migrate_system_messages(self):
        with self.source_conn.cursor() as cursor:
            cursor.execute("SELECT * FROM w_text_notice ORDER BY timestamp DESC")
            notices = cursor.fetchall()

        logger.info(f"  [系统消息] 找到 {len(notices)} 条通知")

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将迁移 {len(notices)} 条系统消息")
            return

        with self.target_conn.cursor() as target_cursor:
            for idx, notice in enumerate(notices, 1):
                try:
                    target_cursor.execute("""
                        INSERT INTO system_messages (
                            type, title, content, target_type, target_id,
                            created_at
                        ) VALUES ('system', %s, %s, 'all', NULL, %s)
                    """, (
                        notice.get('title', ''),
                        notice.get('content', ''),
                        self._datetime_or_default(notice.get('timestamp'))
                    ))

                    if idx % 100 == 0:
                        logger.info(f"  [系统消息] 已迁移 {idx}/{len(notices)} 条")
                except Exception as e:
                    logger.error(f"  [系统消息] 迁移失败: id={notice.get('id')}, error={e}")

            self.target_conn.commit()

    def _migrate_user_notifications(self):
        with self.source_conn.cursor() as cursor:
            cursor.execute("SELECT * FROM w_user_text_notice ORDER BY timestamp DESC")
            notifications = cursor.fetchall()

        logger.info(f"  [用户通知] 找到 {len(notifications)} 条通知")

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将迁移 {len(notifications)} 条用户通知")
            return

        with self.target_conn.cursor() as target_cursor:
            for idx, notification in enumerate(notifications, 1):
                try:
                    user_id = self.user_id_map.get(notification['userId'])
                    if not user_id:
                        continue

                    target_cursor.execute("""
                        INSERT INTO notifications (
                            user_id, type, title, content, is_read, link,
                            created_at
                        ) VALUES (%s, 'system', %s, %s, 
                            CASE WHEN %s = '1' THEN TRUE ELSE FALSE END, NULL, %s)
                    """, (
                        user_id,
                        notification.get('title', ''),
                        notification.get('content', ''),
                        notification.get('isRead', '0'),
                        self._datetime_or_default(notification.get('timestamp'))
                    ))

                    if idx % 100 == 0:
                        logger.info(f"  [用户通知] 已迁移 {idx}/{len(notifications)} 条")
                except Exception as e:
                    logger.error(f"  [用户通知] 迁移失败: id={notification.get('id')}, error={e}")

            self.target_conn.commit()

    def _timestamp_to_datetime(self, timestamp) -> Optional[datetime]:
        if not timestamp or timestamp == 0:
            return None
        try:
            return datetime.fromtimestamp(timestamp / 1000)
        except:
            return None

    def _datetime_or_default(self, dt_value) -> datetime:
        if not dt_value or dt_value == '0001-01-01 00:00:00':
            return datetime.now()
        if isinstance(dt_value, datetime):
            return dt_value
        try:
            return datetime.strptime(str(dt_value), '%Y-%m-%d %H:%M:%S')
        except:
            return datetime.now()

    def close(self):
        self.source_conn.close()
        if self.target_conn:
            self.target_conn.close()


def main():
    parser = argparse.ArgumentParser(description='OIM -> QIM 数据迁移脚本')
    parser.add_argument('--source-host', required=True, help='源数据库主机')
    parser.add_argument('--source-port', type=int, default=3306, help='源数据库端口')
    parser.add_argument('--source-db', required=True, help='源数据库名')
    parser.add_argument('--source-user', required=True, help='源数据库用户')
    parser.add_argument('--source-pass', required=True, help='源数据库密码')

    parser.add_argument('--target-host', required=True, help='目标数据库主机')
    parser.add_argument('--target-port', type=int, default=3306, help='目标数据库端口')
    parser.add_argument('--target-db', required=True, help='目标数据库名')
    parser.add_argument('--target-user', required=True, help='目标数据库用户')
    parser.add_argument('--target-pass', required=True, help='目标数据库密码')

    parser.add_argument('--dry-run', action='store_true', help='测试模式，不实际写入数据')

    args = parser.parse_args()

    source_config = {
        'host': args.source_host,
        'port': args.source_port,
        'database': args.source_db,
        'user': args.source_user,
        'password': args.source_pass,
    }

    target_config = {
        'host': args.target_host,
        'port': args.target_port,
        'database': args.target_db,
        'user': args.target_user,
        'password': args.target_pass,
    }

    engine = MigrationEngine(source_config, target_config, dry_run=args.dry_run)
    try:
        engine.migrate_all()
    except Exception as e:
        logger.error(f"迁移失败: {e}")
        raise
    finally:
        engine.close()


if __name__ == '__main__':
    main()
