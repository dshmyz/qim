-- =====================================================================
-- 迁移历史文件消息的存储路径：/uploads/xxx、/s3/uploads/xxx → /static/uploads/xxx
-- 数据库：MySQL（8.0+，含 utf8mb4）
-- 背景：静态资源服务统一改走 /static/* 前缀（storage.StaticPrefix = "/static/"）。
--       迁移函数 migrateStoragePaths 只覆盖了 files / user_feedbacks，
--       messages.content 里的历史路径仍为旧格式。
-- 说明：新增的 /uploads/*filepath 兼容路由已能让旧 URL 直接下载；
--       本脚本用于把 message 数据同步成新的 /static/ 统一前缀（可选，双保险）。
-- 幂等：已经是 /static/ 的不会被删除；重复执行结果不变。
-- =====================================================================

-- ---------------------------------------------------------------------
-- 第 0 步：备份（建议）
-- 生产环境务必先备份，或先在测试/灰度库执行。
-- ---------------------------------------------------------------------
-- CREATE TABLE messages_bak_YYYYMMDD AS SELECT * FROM messages;

-- ---------------------------------------------------------------------
-- 第 1 步：巡检——迁移前统计
-- ---------------------------------------------------------------------
-- 1.1 命中的消息条数
SELECT COUNT(*) AS legacy_uploads_count
FROM messages
WHERE content LIKE '%/uploads/%';

-- 1.2 抽样查看命中的消息内容形态（含 JSON 内 url 与纯文本路径），确认格式
SELECT id, msg_type, LEFT(content, 200) AS content_preview
FROM messages
WHERE content LIKE '%/uploads/%'
   OR content LIKE '%/s3/uploads/%'
ORDER BY id
LIMIT 100;

-- 1.3 若有历史 S3 前缀需要一并迁移，先看数量
SELECT COUNT(*) AS legacy_s3_count
FROM messages
WHERE content LIKE '%/s3/uploads/%';

-- ---------------------------------------------------------------------
-- 第 2 步：执行迁移（事务）
-- ---------------------------------------------------------------------
START TRANSACTION;

-- 2.1 迁移 /uploads/ → /static/uploads/（覆盖 JSON 内 url 与纯段落内容）
--     WHERE 的 LIKE '%/uploads/%' 天然排除 /static/uploads/（不含 /uploads/ 子串），故幂等。
UPDATE messages
SET content = REPLACE(content, '/uploads/', '/static/uploads/')
WHERE content LIKE '%/uploads/%';

-- 2.2 （可选）迁移历史 S3 格式 /s3/uploads/ → /static/uploads/
--     若 1.3 结果为 0 可跳过。
UPDATE messages
SET content = REPLACE(content, '/s3/uploads/', '/static/uploads/')
WHERE content LIKE '%/s3/uploads/%';

COMMIT;
-- 若中途出错，回滚：
-- ROLLBACK;

-- ---------------------------------------------------------------------
-- 第 3 步：巡检——迁移后校验
-- ---------------------------------------------------------------------
-- 3.1 应恒为 0
SELECT COUNT(*) AS remaining_legacy
FROM messages
WHERE content LIKE '%/uploads/%'
   OR content LIKE '%/s3/%';

-- 3.2 确认新前缀条数与第 1.1 步一致
SELECT COUNT(*) AS migrated_static_count
FROM messages
WHERE content LIKE '%/static/uploads/%';
