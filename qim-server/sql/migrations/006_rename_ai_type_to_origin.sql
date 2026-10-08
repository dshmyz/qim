-- 迁移 006: 为 messages 表添加 origin 列
-- 适用于: SQLite / MySQL
--
-- 当前代码会写入 messages.origin，用于区分普通用户消息、AI 助手消息和数字分身消息。
-- 旧数据库如果缺少该列，会在保存 AI 回复时出现：
--   table messages has no column named origin

ALTER TABLE `messages` ADD COLUMN `origin` VARCHAR(30) DEFAULT '';
