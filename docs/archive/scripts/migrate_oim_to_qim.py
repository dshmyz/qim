#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
OIM 数据库迁移脚本
将数据从 OIM Schema 迁移到 QIM 系统
"""

import pymysql
import logging
from datetime import datetime
from typing import Dict, List, Optional
import hashlib
import uuid

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class OIMToQIMMigrator:
    """OIM 到 QIM 数据迁移器"""
    
    def __init__(self, oim_config: Dict, qim_config: Dict):
        """
        初始化迁移器
        
        Args:
            oim_config: OIM 数据库配置
            qim_config: QIM 数据库配置
        """
        self.oim_config = oim_config
        self.qim_config = qim_config
        self.oim_conn = None
        self.qim_conn = None
        
    def connect(self):
        """连接到两个数据库"""
        try:
            # 连接 OIM 数据库
            self.oim_conn = pymysql.connect(**self.oim_config)
            logger.info("成功连接到 OIM 数据库")
            
            # 连接 QIM 数据库
            self.qim_conn = pymysql.connect(**self.qim_config)
            logger.info("成功连接到 QIM 数据库")
            
        except Exception as e:
            logger.error(f"数据库连接失败: {e}")
            raise
    
    def close(self):
        """关闭数据库连接"""
        if self.oim_conn:
            self.oim_conn.close()
            logger.info("OIM 数据库连接已关闭")
        if self.qim_conn:
            self.qim_conn.close()
            logger.info("QIM 数据库连接已关闭")
    
    def migrate_users(self):
        """迁移用户数据"""
        logger.info("开始迁移用户数据...")
        
        try:
            oim_cursor = self.oim_conn.cursor(pymysql.cursors.DictCursor)
            qim_cursor = self.qim_conn.cursor()
            
            # 从 OIM 读取用户数据
            oim_cursor.execute("SELECT * FROM w_user")
            oim_users = oim_cursor.fetchall()
            
            logger.info(f"找到 {len(oim_users)} 个 OIM 用户")
            
            migrated_count = 0
            for oim_user in oim_users:
                try:
                    # 转换用户数据
                    qim_user = self._convert_user(oim_user)
                    
                    # 插入到 QIM 系统
                    sql = """
                    INSERT INTO users 
                    (username, password_hash, nickname, avatar, type, signature, phone, email, status, created_at, updated_at)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                    ON DUPLICATE KEY UPDATE
                    password_hash = VALUES(password_hash),
                    nickname = VALUES(nickname),
                    avatar = VALUES(avatar),
                    signature = VALUES(signature),
                    phone = VALUES(phone),
                    email = VALUES(email),
                    updated_at = VALUES(updated_at)
                    """
                    
                    qim_cursor.execute(sql, (
                        qim_user['username'],
                        qim_user['password_hash'],
                        qim_user['nickname'],
                        qim_user['avatar'],
                        qim_user['type'],
                        qim_user['signature'],
                        qim_user['phone'],
                        qim_user['email'],
                        qim_user['status'],
                        qim_user['created_at'],
                        qim_user['updated_at']
                    ))
                    
                    migrated_count += 1
                    
                except Exception as e:
                    logger.error(f"迁移用户 {oim_user.get('account')} 失败: {e}")
                    continue
            
            self.qim_conn.commit()
            logger.info(f"用户数据迁移完成，成功迁移 {migrated_count} 个用户")
            
        except Exception as e:
            logger.error(f"用户数据迁移失败: {e}")
            self.qim_conn.rollback()
            raise
        finally:
            oim_cursor.close()
            qim_cursor.close()
    
    def _convert_user(self, oim_user: Dict) -> Dict:
        """
        转换 OIM 用户数据到 QIM 格式
        
        Args:
            oim_user: OIM 用户数据
            
        Returns:
            QIM 用户数据
        """
        # 确定用户类型
        user_type_map = {
            '0': 'user',
            '1': 'admin',
            '2': 'super_admin'
        }
        user_type = user_type_map.get(oim_user.get('type', '0'), 'user')
        
        # 确定状态
        status = 'offline' if oim_user.get('isDisable', 0) == 1 else 'online'
        
        # 生成用户名（优先使用 account，其次使用 email）
        username = oim_user.get('account') or oim_user.get('email') or f"user_{oim_user['id']}"
        
        return {
            'username': username,
            'password_hash': oim_user.get('password', ''),
            'nickname': oim_user.get('nickname') or oim_user.get('name', ''),
            'avatar': oim_user.get('avatar', ''),
            'type': user_type,
            'signature': oim_user.get('signature', ''),
            'phone': oim_user.get('mobile', ''),
            'email': oim_user.get('email', ''),
            'status': status,
            'created_at': datetime.now(),
            'updated_at': datetime.now()
        }
    
    def migrate_groups(self):
        """迁移群组数据"""
        logger.info("开始迁移群组数据...")
        
        try:
            oim_cursor = self.oim_conn.cursor(pymysql.cursors.DictCursor)
            qim_cursor = self.qim_conn.cursor()
            
            # 从 OIM 读取群组数据
            oim_cursor.execute("SELECT * FROM w_group")
            oim_groups = oim_cursor.fetchall()
            
            logger.info(f"找到 {len(oim_groups)} 个 OIM 群组")
            
            migrated_count = 0
            for oim_group in oim_groups:
                try:
                    # 创建会话
                    conversation_id = self._create_conversation(qim_cursor, oim_group)
                    
                    # 转换群组数据
                    qim_group = self._convert_group(oim_group, conversation_id)
                    
                    # 插入群组
                    sql = """
                    INSERT INTO groups 
                    (conversation_id, group_type, name, avatar, creator_id, announcement, invite_permission, created_at, updated_at)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                    """
                    
                    qim_cursor.execute(sql, (
                        qim_group['conversation_id'],
                        qim_group['group_type'],
                        qim_group['name'],
                        qim_group['avatar'],
                        qim_group['creator_id'],
                        qim_group['announcement'],
                        qim_group['invite_permission'],
                        qim_group['created_at'],
                        qim_group['updated_at']
                    ))
                    
                    # 迁移群成员
                    self._migrate_group_members(qim_cursor, oim_group['id'], conversation_id)
                    
                    migrated_count += 1
                    
                except Exception as e:
                    logger.error(f"迁移群组 {oim_group.get('name')} 失败: {e}")
                    continue
            
            self.qim_conn.commit()
            logger.info(f"群组数据迁移完成，成功迁移 {migrated_count} 个群组")
            
        except Exception as e:
            logger.error(f"群组数据迁移失败: {e}")
            self.qim_conn.rollback()
            raise
        finally:
            oim_cursor.close()
            qim_cursor.close()
    
    def _create_conversation(self, cursor, oim_group: Dict) -> int:
        """
        创建会话记录
        
        Args:
            cursor: 数据库游标
            oim_group: OIM 群组数据
            
        Returns:
            会话ID
        """
        sql = """
        INSERT INTO conversations 
        (type, name, avatar, created_at, updated_at)
        VALUES (%s, %s, %s, %s, %s)
        """
        
        cursor.execute(sql, (
            'group',
            oim_group.get('name', ''),
            oim_group.get('avatar', ''),
            datetime.now(),
            datetime.now()
        ))
        
        return cursor.lastrowid
    
    def _convert_group(self, oim_group: Dict, conversation_id: int) -> Dict:
        """
        转换 OIM 群组数据到 QIM 格式
        
        Args:
            oim_group: OIM 群组数据
            conversation_id: 会话ID
            
        Returns:
            QIM 群组数据
        """
        return {
            'conversation_id': conversation_id,
            'group_type': 'group',
            'name': oim_group.get('name', ''),
            'avatar': oim_group.get('avatar', ''),
            'creator_id': 0,  # 需要从群成员中获取群主
            'announcement': oim_group.get('introduce', ''),
            'invite_permission': 'owner_admin',
            'created_at': datetime.now(),
            'updated_at': datetime.now()
        }
    
    def _migrate_group_members(self, cursor, oim_group_id: str, conversation_id: int):
        """
        迁移群成员
        
        Args:
            cursor: 数据库游标
            oim_group_id: OIM 群组ID
            conversation_id: 会话ID
        """
        oim_cursor = self.oim_conn.cursor(pymysql.cursors.DictCursor)
        
        try:
            # 获取群成员
            oim_cursor.execute(
                "SELECT * FROM w_group_member WHERE groupId = %s",
                (oim_group_id,)
            )
            members = oim_cursor.fetchall()
            
            for member in members:
                # 转换用户ID（这里需要根据实际的用户ID映射关系进行调整）
                qim_user_id = self._map_user_id(member['userId'])
                
                if qim_user_id:
                    # 确定角色
                    role_map = {
                        '1': 'owner',
                        '2': 'admin',
                        '3': 'member'
                    }
                    role = role_map.get(member.get('position', '3'), 'member')
                    
                    # 插入会话成员
                    sql = """
                    INSERT INTO conversation_members 
                    (conversation_id, user_id, role, joined_at)
                    VALUES (%s, %s, %s, %s)
                    ON DUPLICATE KEY UPDATE
                    role = VALUES(role)
                    """
                    
                    cursor.execute(sql, (
                        conversation_id,
                        qim_user_id,
                        role,
                        datetime.now()
                    ))
                    
        finally:
            oim_cursor.close()
    
    def _map_user_id(self, oim_user_id: str) -> Optional[int]:
        """
        映射 OIM 用户ID 到 QIM 用户ID
        
        Args:
            oim_user_id: OIM 用户ID
            
        Returns:
            QIM 用户ID，如果找不到则返回 None
        """
        # 这里需要根据实际的ID映射关系实现
        # 可以通过查询数据库或使用映射表
        cursor = self.qim_conn.cursor()
        
        try:
            # 示例：通过用户名查找
            oim_cursor = self.oim_conn.cursor(pymysql.cursors.DictCursor)
            oim_cursor.execute(
                "SELECT account, email FROM w_user WHERE id = %s",
                (oim_user_id,)
            )
            oim_user = oim_cursor.fetchone()
            
            if oim_user:
                username = oim_user.get('account') or oim_user.get('email')
                if username:
                    cursor.execute(
                        "SELECT id FROM users WHERE username = %s",
                        (username,)
                    )
                    result = cursor.fetchone()
                    if result:
                        return result[0]
            
            return None
            
        finally:
            cursor.close()
    
    def migrate_messages(self):
        """迁移消息数据"""
        logger.info("开始迁移消息数据...")
        
        try:
            oim_cursor = self.oim_conn.cursor(pymysql.cursors.DictCursor)
            qim_cursor = self.qim_conn.cursor()
            
            # 迁移单聊消息
            self._migrate_user_messages(oim_cursor, qim_cursor)
            
            # 迁移群聊消息
            self._migrate_group_messages(oim_cursor, qim_cursor)
            
            self.qim_conn.commit()
            logger.info("消息数据迁移完成")
            
        except Exception as e:
            logger.error(f"消息数据迁移失败: {e}")
            self.qim_conn.rollback()
            raise
        finally:
            oim_cursor.close()
            qim_cursor.close()
    
    def _migrate_user_messages(self, oim_cursor, qim_cursor):
        """迁移单聊消息"""
        logger.info("迁移单聊消息...")
        
        oim_cursor.execute("SELECT * FROM im_user_chat_content")
        messages = oim_cursor.fetchall()
        
        logger.info(f"找到 {len(messages)} 条单聊消息")
        
        migrated_count = 0
        for msg in messages:
            try:
                # 查找或创建会话
                conversation_id = self._find_or_create_user_conversation(
                    qim_cursor,
                    msg['sendUserId'],
                    msg['receiveUserId']
                )
                
                if not conversation_id:
                    continue
                
                # 转换消息
                qim_message = self._convert_user_message(msg, conversation_id)
                
                # 插入消息
                sql = """
                INSERT INTO messages 
                (conversation_id, sender_id, type, content, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s)
                """
                
                qim_cursor.execute(sql, (
                    qim_message['conversation_id'],
                    qim_message['sender_id'],
                    qim_message['type'],
                    qim_message['content'],
                    qim_message['created_at'],
                    qim_message['updated_at']
                ))
                
                migrated_count += 1
                
            except Exception as e:
                logger.error(f"迁移消息 {msg.get('id')} 失败: {e}")
                continue
        
        logger.info(f"单聊消息迁移完成，成功迁移 {migrated_count} 条消息")
    
    def _migrate_group_messages(self, oim_cursor, qim_cursor):
        """迁移群聊消息"""
        logger.info("迁移群聊消息...")
        
        oim_cursor.execute("SELECT * FROM im_group_chat_content")
        messages = oim_cursor.fetchall()
        
        logger.info(f"找到 {len(messages)} 条群聊消息")
        
        migrated_count = 0
        for msg in messages:
            try:
                # 查找群组会话
                conversation_id = self._find_group_conversation(qim_cursor, msg['groupId'])
                
                if not conversation_id:
                    continue
                
                # 转换消息
                qim_message = self._convert_group_message(msg, conversation_id)
                
                # 插入消息
                sql = """
                INSERT INTO messages 
                (conversation_id, sender_id, type, content, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s)
                """
                
                qim_cursor.execute(sql, (
                    qim_message['conversation_id'],
                    qim_message['sender_id'],
                    qim_message['type'],
                    qim_message['content'],
                    qim_message['created_at'],
                    qim_message['updated_at']
                ))
                
                migrated_count += 1
                
            except Exception as e:
                logger.error(f"迁移群聊消息 {msg.get('id')} 失败: {e}")
                continue
        
        logger.info(f"群聊消息迁移完成，成功迁移 {migrated_count} 条消息")
    
    def _find_or_create_user_conversation(self, cursor, user1_id: str, user2_id: str) -> Optional[int]:
        """查找或创建单聊会话"""
        qim_user1_id = self._map_user_id(user1_id)
        qim_user2_id = self._map_user_id(user2_id)
        
        if not qim_user1_id or not qim_user2_id:
            return None
        
        # 查找现有会话
        cursor.execute("""
            SELECT c.id FROM conversations c
            JOIN conversation_members cm1 ON c.id = cm1.conversation_id AND cm1.user_id = %s
            JOIN conversation_members cm2 ON c.id = cm2.conversation_id AND cm2.user_id = %s
            WHERE c.type = 'direct'
        """, (qim_user1_id, qim_user2_id))
        
        result = cursor.fetchone()
        if result:
            return result[0]
        
        # 创建新会话
        cursor.execute("""
            INSERT INTO conversations (type, created_at, updated_at)
            VALUES ('direct', %s, %s)
        """, (datetime.now(), datetime.now()))
        
        conversation_id = cursor.lastrowid
        
        # 添加会话成员
        cursor.execute("""
            INSERT INTO conversation_members (conversation_id, user_id, joined_at)
            VALUES (%s, %s, %s), (%s, %s, %s)
        """, (conversation_id, qim_user1_id, datetime.now(),
              conversation_id, qim_user2_id, datetime.now()))
        
        return conversation_id
    
    def _find_group_conversation(self, cursor, oim_group_id: str) -> Optional[int]:
        """查找群组会话"""
        # 通过群组名称查找
        oim_cursor = self.oim_conn.cursor(pymysql.cursors.DictCursor)
        
        try:
            oim_cursor.execute(
                "SELECT name FROM w_group WHERE id = %s",
                (oim_group_id,)
            )
            oim_group = oim_cursor.fetchone()
            
            if oim_group:
                cursor.execute("""
                    SELECT g.conversation_id FROM groups g
                    WHERE g.name = %s
                """, (oim_group['name'],))
                
                result = cursor.fetchone()
                if result:
                    return result[0]
            
            return None
            
        finally:
            oim_cursor.close()
    
    def _convert_user_message(self, oim_msg: Dict, conversation_id: int) -> Dict:
        """转换单聊消息"""
        sender_id = self._map_user_id(oim_msg['sendUserId'])
        
        # 获取消息内容
        content = self._get_message_content(oim_msg['id'], 'user')
        
        return {
            'conversation_id': conversation_id,
            'sender_id': sender_id or 0,
            'type': 'text',
            'content': content,
            'created_at': datetime.now(),
            'updated_at': datetime.now()
        }
    
    def _convert_group_message(self, oim_msg: Dict, conversation_id: int) -> Dict:
        """转换群聊消息"""
        sender_id = self._map_user_id(oim_msg['userId'])
        
        # 获取消息内容
        content = self._get_message_content(oim_msg['id'], 'group')
        
        return {
            'conversation_id': conversation_id,
            'sender_id': sender_id or 0,
            'type': 'text',
            'content': content,
            'created_at': datetime.now(),
            'updated_at': datetime.now()
        }
    
    def _get_message_content(self, message_id: str, msg_type: str) -> str:
        """获取消息内容"""
        cursor = self.oim_conn.cursor(pymysql.cursors.DictCursor)
        
        try:
            if msg_type == 'user':
                cursor.execute(
                    "SELECT * FROM im_user_chat_item WHERE contentId = %s ORDER BY sort",
                    (message_id,)
                )
            else:
                cursor.execute(
                    "SELECT * FROM im_group_chat_item WHERE contentId = %s ORDER BY sort",
                    (message_id,)
                )
            
            items = cursor.fetchall()
            
            if items:
                # 合并所有消息项
                contents = []
                for item in items:
                    if item.get('originalValue'):
                        contents.append(item['originalValue'])
                return ' '.join(contents)
            
            return ''
            
        finally:
            cursor.close()
    
    def migrate_contacts(self):
        """迁移联系人数据"""
        logger.info("开始迁移联系人数据...")
        
        try:
            oim_cursor = self.oim_conn.cursor(pymysql.cursors.DictCursor)
            qim_cursor = self.qim_conn.cursor()
            
            # 从 OIM 读取联系人关系
            oim_cursor.execute("SELECT * FROM w_contact_relation")
            contacts = oim_cursor.fetchall()
            
            logger.info(f"找到 {len(contacts)} 个联系人关系")
            
            migrated_count = 0
            for contact in contacts:
                try:
                    qim_owner_id = self._map_user_id(contact['ownerUserId'])
                    qim_contact_id = self._map_user_id(contact['contactUserId'])
                    
                    if not qim_owner_id or not qim_contact_id:
                        continue
                    
                    # 创建会话
                    conversation_id = self._find_or_create_user_conversation(
                        qim_cursor,
                        contact['ownerUserId'],
                        contact['contactUserId']
                    )
                    
                    if conversation_id:
                        migrated_count += 1
                    
                except Exception as e:
                    logger.error(f"迁移联系人关系失败: {e}")
                    continue
            
            self.qim_conn.commit()
            logger.info(f"联系人数据迁移完成，成功迁移 {migrated_count} 个联系人关系")
            
        except Exception as e:
            logger.error(f"联系人数据迁移失败: {e}")
            self.qim_conn.rollback()
            raise
        finally:
            oim_cursor.close()
            qim_cursor.close()
    
    def run_migration(self):
        """执行完整的数据迁移"""
        logger.info("=" * 50)
        logger.info("开始 OIM 到 QIM 数据迁移")
        logger.info("=" * 50)
        
        try:
            self.connect()
            
            # 按顺序执行迁移
            self.migrate_users()
            self.migrate_groups()
            self.migrate_contacts()
            self.migrate_messages()
            
            logger.info("=" * 50)
            logger.info("数据迁移完成！")
            logger.info("=" * 50)
            
        except Exception as e:
            logger.error(f"数据迁移过程中发生错误: {e}")
            raise
        finally:
            self.close()


def main():
    """主函数"""
    # OIM 数据库配置
    oim_config = {
        'host': 'localhost',
        'port': 3306,
        'user': 'root',
        'password': 'your_password',
        'database': 'oim_database',
        'charset': 'utf8mb4'
    }
    
    # QIM 数据库配置
    qim_config = {
        'host': 'localhost',
        'port': 3306,
        'user': 'root',
        'password': 'your_password',
        'database': 'qim_server',
        'charset': 'utf8mb4'
    }
    
    # 创建迁移器并执行迁移
    migrator = OIMToQIMMigrator(oim_config, qim_config)
    migrator.run_migration()


if __name__ == '__main__':
    main()
