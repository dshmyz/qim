#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import importlib.util
import sys
import types
import unittest
from pathlib import Path


if 'pymysql' not in sys.modules:
    fake_pymysql = types.ModuleType('pymysql')
    fake_pymysql.cursors = types.SimpleNamespace(DictCursor=object)
    fake_pymysql.connect = lambda *args, **kwargs: None
    sys.modules['pymysql'] = fake_pymysql


MODULE_PATH = Path(__file__).with_name('migrate.py')
spec = importlib.util.spec_from_file_location('migrate', MODULE_PATH)
migrate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(migrate)


class FakeCursor:
    def __init__(self, responses=None):
        self.responses = responses or []
        self.executed = []
        self.lastrowid = 100

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        return False

    def execute(self, sql, params=None):
        self.executed.append((sql, params))

    def fetchall(self):
        return self.responses.pop(0) if self.responses else []

    def fetchone(self):
        rows = self.fetchall()
        return rows[0] if rows else None


class FakeConnection:
    def __init__(self, cursor):
        self._cursor = cursor
        self.commits = 0

    def cursor(self):
        return self._cursor

    def commit(self):
        self.commits += 1


class DuplicateAwareTargetCursor(FakeCursor):
    def execute(self, sql, params=None):
        if 'INSERT INTO users' in sql and 'ON DUPLICATE KEY UPDATE' not in sql:
            raise Exception('Duplicate entry')
        super().execute(sql, params)


def make_engine():
    engine = migrate.MigrationEngine.__new__(migrate.MigrationEngine)
    engine.dry_run = False
    engine.user_id_map = {}
    engine.group_id_map = {}
    engine.conversation_id_map = {}
    engine._group_invite_settings = {}
    engine._group_blocked_users = {}
    engine._notice_cache = {}
    engine._source_table_cache = {}
    engine._source_column_cache = {}
    engine._group_creator_user_map = {}
    engine.stats = {
        'users': 0,
        'groups': 0,
        'conversations': 0,
        'messages': 0,
        'members': 0,
        'sessions': 0,
    }
    engine.skip_stats = {
        'users': 0,
        'groups_no_creator': 0,
        'members_no_conv': 0,
        'members_no_user': 0,
        'messages_no_conv': 0,
        'messages_no_sender': 0,
    }
    return engine


class MigrationCompatibilityTest(unittest.TestCase):
    def test_preload_ignores_missing_notice_table(self):
        engine = make_engine()
        engine._source_table_exists = lambda name: name != 'w_text_notice'
        cursor = FakeCursor([
            [{'groupId': 'g1', 'joinType': '2', 'inviteType': '4'}],
            [{'groupId': 'g1', 'userId': 'u1'}],
        ])
        engine.source_conn = FakeConnection(cursor)

        engine._preload_data()

        self.assertEqual({}, engine._notice_cache)
        executed_sql = ' '.join(sql for sql, _ in cursor.executed)
        self.assertNotIn('SELECT * FROM w_text_notice', executed_sql)

    def test_preload_ignores_missing_is_blocked_column(self):
        engine = make_engine()
        engine._source_table_exists = lambda name: name != 'w_text_notice'
        engine._source_column_exists = lambda table, column: not (
            table == 'w_group_relation' and column == 'isBlocked'
        )
        cursor = FakeCursor([
            [{'groupId': 'g1', 'joinType': '2', 'inviteType': '4'}],
        ])
        engine.source_conn = FakeConnection(cursor)

        engine._preload_data()

        self.assertEqual({}, engine._group_blocked_users)
        executed_sql = ' '.join(sql for sql, _ in cursor.executed)
        self.assertNotIn('isBlocked', executed_sql)

    def test_group_insert_uses_is_deleted_and_creates_owner_member(self):
        engine = make_engine()
        engine.user_id_map = {'u1': 1}
        source_cursor = FakeCursor([
            [{
                'id': 'g1',
                'name': 'legacy group',
                'avatar': '',
                'introduce': '',
                'createdTimestamp': 0,
                'updatedTimestamp': 1710000000000,
                'isDeleted': 1,
            }],
            [],
            [],
        ])
        target_cursor = FakeCursor()
        engine.source_conn = FakeConnection(source_cursor)
        engine.target_conn = FakeConnection(target_cursor)

        engine.migrate_groups()

        conversation_sql = target_cursor.executed[0][0]
        owner_sql = target_cursor.executed[2][0]
        self.assertIn('is_deleted', conversation_sql)
        self.assertNotIn('deleted_at', conversation_sql)
        self.assertIn('conversation_members', owner_sql)
        self.assertEqual(100, target_cursor.executed[2][1][0])
        self.assertEqual(1, target_cursor.executed[2][1][1])

    def test_existing_target_user_backfills_user_id_map(self):
        engine = make_engine()
        source_cursor = FakeCursor([
            [{
                'id': 'old-u1',
                'number': 10001,
                'account': 'alice',
                'password': 'hash',
                'nickname': 'Alice',
                'name': 'Alice',
                'gender': '3',
                'createdTimestamp': 0,
                'isDisable': 0,
                'isDeleted': 0,
            }],
        ])
        target_cursor = DuplicateAwareTargetCursor()
        target_cursor.lastrowid = 42
        engine.source_conn = FakeConnection(source_cursor)
        engine.target_conn = FakeConnection(target_cursor)

        engine.migrate_users()

        self.assertEqual(42, engine.user_id_map['old-u1'])
        insert_sql = target_cursor.executed[0][0]
        self.assertIn('ON DUPLICATE KEY UPDATE', insert_sql)

    def test_group_creator_can_be_resolved_from_existing_target_user(self):
        engine = make_engine()
        source_cursor = FakeCursor([
            [{'userId': 'old-u1'}],
            [{
                'id': 'old-u1',
                'number': 10001,
                'account': 'alice',
            }],
        ])
        target_cursor = FakeCursor([
            [{'id': 42}],
        ])
        engine.source_conn = FakeConnection(source_cursor)
        engine.target_conn = FakeConnection(target_cursor)

        creator_id = engine._get_group_creator_id('g1')

        self.assertEqual(42, creator_id)
        self.assertEqual(42, engine.user_id_map['old-u1'])

    def test_self_private_conversation_inserts_one_member(self):
        engine = make_engine()
        engine.user_id_map = {'old-u1': 42}
        source_cursor = FakeCursor([
            [{'user1': 'old-u1', 'user2': 'old-u1'}],
        ])
        target_cursor = FakeCursor()
        target_cursor.lastrowid = 200
        engine.source_conn = FakeConnection(source_cursor)
        engine.target_conn = FakeConnection(target_cursor)

        engine.migrate_private_conversations()

        member_inserts = [
            sql for sql, _ in target_cursor.executed
            if 'conversation_members' in sql
        ]
        self.assertEqual(1, len(member_inserts))
        self.assertEqual(200, engine.conversation_id_map['single_old-u1_old-u1'])

    def test_at_content_is_migrated_as_plain_text(self):
        engine = make_engine()

        self.assertEqual('@所有人', engine._transform_content('at', '{"text":"@所有人","userid":0}'))
        self.assertEqual('@所有人', engine._transform_content('at', '{text: @所有人， userid: 0}'))

    def test_group_message_migration_skips_empty_items(self):
        engine = make_engine()
        engine.user_id_map = {'old-u1': 42}
        engine.conversation_id_map = {'group_g1': 200}
        source_cursor = FakeCursor([
            [
                {
                    'contentId': 'c1',
                    'groupId': 'g1',
                    'userId': 'old-u1',
                    'type': 'at',
                    'originalValue': '{"text":"@所有人","userid":0}',
                    'filterValue': '',
                    'timestamp': 1710000000000,
                    'isDeleted': 0,
                },
                {
                    'contentId': 'c1',
                    'groupId': 'g1',
                    'userId': 'old-u1',
                    'type': 'text',
                    'originalValue': '',
                    'filterValue': '',
                    'timestamp': 1710000000000,
                    'isDeleted': 0,
                },
            ],
        ])
        target_cursor = FakeCursor()
        engine.source_conn = FakeConnection(source_cursor)
        engine.target_conn = FakeConnection(target_cursor)

        engine._migrate_group_messages()

        message_inserts = [
            params for sql, params in target_cursor.executed
            if 'INSERT INTO messages' in sql
        ]
        self.assertEqual(1, len(message_inserts))
        self.assertEqual('@所有人', message_inserts[0]['content'])

    def test_missing_notice_tables_skip_notification_migration(self):
        engine = make_engine()
        engine._source_table_exists = lambda name: False
        engine.source_conn = FakeConnection(FakeCursor())
        engine.target_conn = FakeConnection(FakeCursor())

        engine.migrate_notifications()

        self.assertEqual([], engine.source_conn._cursor.executed)
        self.assertEqual([], engine.target_conn._cursor.executed)


if __name__ == '__main__':
    unittest.main()
