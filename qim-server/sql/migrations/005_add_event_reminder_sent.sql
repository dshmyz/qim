-- 迁移 005: 为 events 表添加 reminder_sent 列和复合索引
-- 适用于: SQLite / MySQL

-- 添加 reminder_sent 列
ALTER TABLE events ADD COLUMN reminder_sent BOOLEAN DEFAULT FALSE;

-- 添加复合索引（用于事件提醒查询优化）
CREATE INDEX IF NOT EXISTS idx_events_reminder ON events(reminder, reminder_sent, start_time);
