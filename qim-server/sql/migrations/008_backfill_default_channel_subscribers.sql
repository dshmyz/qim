-- 一次性回填：给所有存量真人用户补订阅默认频道
-- 适用场景：默认频道（seedDefaultChannel，2026-07-01 引入）创建时未回填存量用户，
-- 导致 2026-07-01 之前注册、之后从未打开过频道列表的用户未订阅默认频道，
-- 管理后台「订阅数」小于用户总数。
--
-- 背景：默认频道的自动订阅只有两条路径，均不覆盖存量漏网用户：
--   1. User.AfterCreate（新用户注册时自动订阅，只对新人生效）
--   2. GetChannels 懒订阅兜底（用户打开频道列表时补订，前提是打开过频道 Tab）
--
-- 本脚本为一次性补齐；执行后无需维护——后续新用户仍走 AfterCreate 钩子自动订阅。
-- 可重复执行（INSERT IGNORE 依赖 channel_subscribers 的 (channel_id, user_id) 唯一索引去重）。

-- MySQL 语法（默认数据库）
INSERT IGNORE INTO channel_subscribers (channel_id, user_id, joined_at)
SELECT c.id, u.id, NOW()
FROM channels c
JOIN users u ON u.deleted_at IS NULL AND (u.type = '' OR u.type = 'user')
WHERE c.is_default = 1 AND c.status = 'active';

-- 查询验证：执行后「用户总数」应等于「已订阅数」
SELECT c.name,
       COUNT(DISTINCT u.id)      AS total_users,
       COUNT(DISTINCT s.user_id) AS subscribed
FROM channels c
JOIN users u ON u.deleted_at IS NULL AND (u.type = '' OR u.type = 'user')
LEFT JOIN channel_subscribers s ON s.channel_id = c.id AND s.user_id = u.id
WHERE c.is_default = 1 AND c.status = 'active'
GROUP BY c.id;

-- SQLite 版本（开发环境切 SQLite 时使用）：
--   INSERT IGNORE  -> INSERT OR IGNORE
--   NOW()          -> datetime('now')
-- 如需给所有人订阅所有频道（不限默认），去掉 WHERE 中的 c.is_default = 1 即可。
