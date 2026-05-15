-- Migration: AI 角色定位重构
-- Date: 2026-05-15
-- Description: 为 users 表添加 is_bot/bot_type 字段，为 messages 表添加 is_avatar_reply/ai_type 字段，为 bots 表添加 group_id 字段

-- Users table
ALTER TABLE `users` ADD COLUMN `bot_type` VARCHAR(30) DEFAULT '';

-- Messages table
ALTER TABLE `messages` ADD COLUMN `ai_type` VARCHAR(30) DEFAULT '';

-- Bots table
ALTER TABLE `bots` ADD COLUMN `group_id` INT UNSIGNED NULL;
CREATE INDEX `idx_bots_group_id` ON `bots`(`group_id`);
