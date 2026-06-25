-- Migration: 修复历史 AI 消息的已读状态
-- Date: 2026-06-25
-- Description: 将历史 AI 消息（origin = 'assistant' 或 'avatar'）的 is_read 更新为 true

-- SQLite
UPDATE messages
SET is_read = true
WHERE (origin = 'assistant' OR origin = 'avatar')
  AND is_read = false
  AND sender_id != 0;
