#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
OIM Server -> QIM Server 文件数据迁移脚本

迁移源表（4 张）:
  - base_file_data       (通用文件)        -> files.source='upload'
  - base_image_data      (图片)            -> files.source='upload'
  - base_user_head_data  (用户头像)        -> files.source='avatar'
  - base_group_head_data (群头像)          -> files.source='group_avatar'

目标表: files

字段映射:
  老系统                      -> 新系统
  ----------------------------------------------------
  id (VARCHAR UUID)           -> source_id (保留原 UUID 便于追溯)
  userId                      -> user_id (通过 w_user.account -> users.username 反查)
  originalFullName            -> original_name
  saveFullName                -> name
  size                        -> size
  extension / type            -> mime_type
  fullPathName                -> storage_path
  md5                         -> checksum
  createdTimestamp            -> created_at
  updatedTimestamp            -> updated_at
  isDeleted=1                 -> deleted_at = created_at
  url                         -> tags (保留原始 URL 便于后续校验/重定向)

依赖:
    pip install pymysql

使用方法:
    python migrate_files.py --source-host=localhost --source-db=oim_db \
                            --target-host=localhost --target-db=qim_server \
                            --source-user=root --source-pass=xxx \
                            --target-user=root --target-pass=xxx

    测试模式（不实际写入）:
    python migrate_files.py ... --dry-run

    跳过指定表:
    python migrate_files.py ... --skip-tables=base_user_head_data,base_group_head_data

注意:
    本脚本只迁移文件元数据，不搬运物理文件。
    物理文件需要单独通过 rsync / 对象存储同步工具搬运到新系统存储路径下。
"""

import argparse
import logging
import pymysql
from datetime import datetime
from typing import Dict, List, Optional, Set

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger(__name__)


# 源表配置: 表名 -> 写入 files 表时使用的 source 值
SOURCE_TABLES = {
    'base_file_data':       'upload',
    'base_image_data':      'upload',
    'base_user_head_data':  'avatar',
    'base_group_head_data': 'group_avatar',
}


class FileMigrationEngine:
    """文件元数据迁移引擎。

    与 migrate.py 相互独立，可单独运行。
    用户 ID 映射通过源库 w_user.account -> 目标库 users.username 反查，带缓存。
    """

    def __init__(self, source_config: Dict, target_config: Dict,
                 dry_run: bool = False, batch_size: int = 1000,
                 skip_tables: Optional[Set[str]] = None):
        self.dry_run = dry_run
        self.batch_size = batch_size
        self.skip_tables = skip_tables or set()

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

        # old_user_id(VARCHAR) -> new_user_id(INT)
        self.user_id_map: Dict[str, int] = {}
        # 反查失败的老 user_id 集合，避免重复查询
        self._missing_user_ids: Set[str] = set()

        # 已迁移的 (source, source_id) 集合，用于去重
        self._migrated_keys: Set[tuple] = set()

        self.stats = {
            'total_rows': 0,
            'migrated': 0,
            'skipped_deleted': 0,
            'skipped_no_user': 0,
            'skipped_duplicate': 0,
            'skipped_invalid_path': 0,
            'failed': 0,
        }

        self._source_table_cache: Dict[str, bool] = {}

    # ------------------------------------------------------------
    # 主流程
    # ------------------------------------------------------------
    def migrate_all(self):
        if self.dry_run:
            logger.warning("【测试模式】不会实际写入数据")

        logger.info("开始迁移文件元数据...")
        logger.info("=" * 50)

        # 预加载已存在的 (source, source_id) 用于去重（仅在非 dry-run 时）
        if not self.dry_run:
            self._preload_migrated_keys()

        for idx, (table_name, source_value) in enumerate(SOURCE_TABLES.items(), 1):
            if table_name in self.skip_tables:
                logger.info(f"\n[{idx}/{len(SOURCE_TABLES)}] 跳过表 {table_name}（--skip-tables）")
                continue

            logger.info(f"\n[{idx}/{len(SOURCE_TABLES)}] 迁移表 {table_name} (source={source_value})...")
            self._migrate_file_table(table_name, source_value)

        logger.info("\n" + "=" * 50)
        logger.info("迁移完成！统计信息:")
        for key, value in self.stats.items():
            logger.info(f"  {key}: {value}")

    def _migrate_file_table(self, table_name: str, source_value: str):
        """通用迁移方法：把指定的源表数据迁移到 files 表。

        四张源表结构完全一致，通过 source_value 参数区分来源。
        """
        if not self._source_table_exists(table_name):
            logger.warning(f"  源表 {table_name} 不存在，跳过")
            return

        with self.source_conn.cursor() as cursor:
            cursor.execute(f"SELECT * FROM `{table_name}` ORDER BY createdTimestamp")
            rows = cursor.fetchall()

        logger.info(f"  找到 {len(rows)} 条记录")
        self.stats['total_rows'] += len(rows)

        if self.dry_run:
            logger.info(f"  [DRY-RUN] 将迁移 {len(rows)} 条记录")
            for row in rows[:3]:
                logger.info(f"    - {row.get('originalFullName', '')} (id={row['id']})")
            if len(rows) > 3:
                logger.info(f"    ... 及其他 {len(rows) - 3} 条记录")
            self.stats['migrated'] += len(rows)
            return

        batch: List[Dict] = []
        for idx, row in enumerate(rows, 1):
            try:
                file_data = self._transform_row(row, source_value, table_name)
                if file_data is None:
                    continue

                # 去重检查
                dedup_key = (file_data['source'], file_data['source_id'])
                if dedup_key in self._migrated_keys:
                    self.stats['skipped_duplicate'] += 1
                    continue
                self._migrated_keys.add(dedup_key)

                batch.append(file_data)

                if len(batch) >= self.batch_size:
                    self._batch_insert(batch)
                    batch = []

                if idx % 1000 == 0:
                    logger.info(f"  已处理 {idx}/{len(rows)} 条")
            except Exception as e:
                logger.error(f"  记录处理失败: table={table_name}, id={row.get('id')}, error={e}")
                self.stats['failed'] += 1

        if batch:
            self._batch_insert(batch)

    def _transform_row(self, row: Dict, source_value: str, table_name: str) -> Optional[Dict]:
        """把源表一行转换为 files 表的写入数据。

        返回 None 表示该行应被跳过（已删除/无用户/无路径）。
        """
        # 跳过已删除记录：写入 deleted_at 但仍迁移元数据
        is_deleted = self._is_truthy_flag(row.get('isDeleted', 0))

        # 必须有存储路径
        storage_path = (row.get('fullPathName') or '').strip()
        if not storage_path:
            # 退化使用 rootPath + nodePath + saveFullName
            parts = [row.get('rootPath', ''), row.get('nodePath', ''), row.get('saveFullName', '')]
            storage_path = '/'.join(p for p in parts if p)
        if not storage_path:
            self.stats['skipped_invalid_path'] += 1
            return None

        # 用户 ID 映射
        old_user_id = row.get('userId', '')
        new_user_id = self._lookup_target_user_id(old_user_id) if old_user_id else None
        if not new_user_id:
            self.stats['skipped_no_user'] += 1
            return None

        # 时间处理
        created_at = self._timestamp_to_datetime(row.get('createdTimestamp', 0)) or datetime.now()
        updated_at = self._timestamp_to_datetime(row.get('updatedTimestamp', 0)) or created_at
        deleted_at = created_at if is_deleted else None

        # mime_type: 优先用 extension，没有则用 type
        extension = (row.get('extension') or '').strip().lstrip('.')
        old_type = (row.get('type') or '').strip()
        mime_type = extension if extension else old_type

        # name: 优先 saveFullName，没有则 originalFullName
        name = (row.get('saveFullName') or row.get('originalFullName') or '').strip()
        if not name:
            name = f"file_{row.get('id', '')}"

        original_name = (row.get('originalFullName') or '').strip()

        # checksum: 优先 md5，没有则 sha1
        checksum = (row.get('md5') or '').strip()
        if not checksum:
            checksum = (row.get('sha1') or '').strip()

        # url 保留到 tags 字段，便于后续校验/重定向
        old_url = (row.get('url') or '').strip()

        return {
            'user_id': new_user_id,
            'name': name[:255],
            'original_name': original_name[:255] if original_name else None,
            'size': int(row.get('size', 0) or 0),
            'mime_type': mime_type[:100] if mime_type else None,
            'storage_path': storage_path[:500],
            'checksum': checksum[:64] if checksum else None,
            'folder_id': None,
            'source': source_value,
            'source_id': str(row.get('id', ''))[:100],
            'is_starred': False,
            'starred_at': None,
            'tags': old_url[:500] if old_url else None,
            'created_at': created_at,
            'updated_at': updated_at,
            'deleted_at': deleted_at,
        }

    def _batch_insert(self, batch: List[Dict]):
        with self.target_conn.cursor() as cursor:
            for data in batch:
                cursor.execute("""
                    INSERT INTO files (
                        user_id, name, original_name, size, mime_type,
                        storage_path, checksum, folder_id, source, source_id,
                        is_starred, starred_at, tags,
                        created_at, updated_at, deleted_at
                    ) VALUES (
                        %(user_id)s, %(name)s, %(original_name)s, %(size)s, %(mime_type)s,
                        %(storage_path)s, %(checksum)s, %(folder_id)s, %(source)s, %(source_id)s,
                        %(is_starred)s, %(starred_at)s, %(tags)s,
                        %(created_at)s, %(updated_at)s, %(deleted_at)s
                    )
                """, data)
                self.stats['migrated'] += 1
            self.target_conn.commit()

    # ------------------------------------------------------------
    # 用户 ID 映射
    # ------------------------------------------------------------
    def _lookup_target_user_id(self, old_user_id: str) -> Optional[int]:
        """通过源库 w_user.account -> 目标库 users.username 反查新用户 ID。

        带缓存，避免重复查询。
        """
        if not old_user_id:
            return None
        if old_user_id in self.user_id_map:
            return self.user_id_map[old_user_id]
        if old_user_id in self._missing_user_ids:
            return None

        # 查源库 w_user
        try:
            with self.source_conn.cursor() as cursor:
                cursor.execute(
                    "SELECT account, number FROM w_user WHERE id=%s LIMIT 1",
                    (old_user_id,)
                )
                old_user = cursor.fetchone()
        except Exception as e:
            logger.warning(f"  查询源用户失败: old_user_id={old_user_id}, error={e}")
            self._missing_user_ids.add(old_user_id)
            return None

        if not old_user:
            self._missing_user_ids.add(old_user_id)
            return None

        # 与 migrate.py 的 _legacy_username 保持一致
        username = old_user['account'] if old_user.get('account') else f"user_{old_user.get('number', '')}"
        if not username:
            self._missing_user_ids.add(old_user_id)
            return None

        # 查目标库 users
        try:
            with self.target_conn.cursor() as cursor:
                cursor.execute(
                    "SELECT id FROM users WHERE username=%s LIMIT 1",
                    (username,)
                )
                target_user = cursor.fetchone()
        except Exception as e:
            logger.warning(f"  查询目标用户失败: old_user_id={old_user_id}, username={username}, error={e}")
            self._missing_user_ids.add(old_user_id)
            return None

        if not target_user:
            self._missing_user_ids.add(old_user_id)
            return None

        new_id = target_user['id']
        self.user_id_map[old_user_id] = new_id
        return new_id

    # ------------------------------------------------------------
    # 工具方法
    # ------------------------------------------------------------
    def _preload_migrated_keys(self):
        """预加载目标 files 表已存在的 (source, source_id) 用于去重。

        仅加载 source_id 非空的记录，避免重复迁移。
        """
        logger.info("预加载已迁移文件的去重键...")
        with self.target_conn.cursor() as cursor:
            cursor.execute(
                "SELECT source, source_id FROM files WHERE source_id IS NOT NULL AND source_id <> ''"
            )
            for row in cursor.fetchall():
                self._migrated_keys.add((row['source'], row['source_id']))
        logger.info(f"  已加载 {len(self._migrated_keys)} 条去重键")

    def _source_table_exists(self, table_name: str) -> bool:
        if table_name in self._source_table_cache:
            return self._source_table_cache[table_name]
        try:
            with self.source_conn.cursor() as cursor:
                cursor.execute("SHOW TABLES LIKE %s", (table_name,))
                exists = cursor.fetchone() is not None
        except Exception as e:
            logger.warning(f"  检查源表 {table_name} 是否存在失败，按不存在处理: {e}")
            exists = False
        self._source_table_cache[table_name] = exists
        return exists

    def _timestamp_to_datetime(self, timestamp) -> Optional[datetime]:
        if not timestamp or timestamp == 0:
            return None
        try:
            # 先按毫秒时间戳处理，如果结果远在未来则按秒处理
            dt = datetime.fromtimestamp(timestamp / 1000)
            if dt.year > 2030:
                dt = datetime.fromtimestamp(timestamp)
            return dt
        except Exception:
            try:
                return datetime.fromtimestamp(timestamp)
            except Exception:
                return None

    def _is_truthy_flag(self, value) -> bool:
        return str(value).strip().lower() in ('1', 'true', 'yes')

    def close(self):
        self.source_conn.close()
        if self.target_conn:
            self.target_conn.close()


def main():
    parser = argparse.ArgumentParser(description='OIM -> QIM 文件元数据迁移脚本')
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
    parser.add_argument('--batch-size', type=int, default=1000, help='批量插入大小')
    parser.add_argument('--skip-tables', default='', help='跳过的源表，逗号分隔。可选: base_file_data,base_image_data,base_user_head_data,base_group_head_data')

    args = parser.parse_args()

    skip_tables = set()
    if args.skip_tables:
        skip_tables = {t.strip() for t in args.skip_tables.split(',') if t.strip()}
        invalid = skip_tables - set(SOURCE_TABLES.keys())
        if invalid:
            parser.error(f"未知的表名: {invalid}。可选: {list(SOURCE_TABLES.keys())}")

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

    engine = FileMigrationEngine(
        source_config, target_config,
        dry_run=args.dry_run,
        batch_size=args.batch_size,
        skip_tables=skip_tables,
    )
    try:
        engine.migrate_all()
    except Exception as e:
        logger.error(f"迁移失败: {e}")
        raise
    finally:
        engine.close()


if __name__ == '__main__':
    main()
