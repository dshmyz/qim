-- 迁移 001: 扩展 Message 表 content 字段以支持长 AI 回复
-- 适用于: MySQL 5.7+, TiDB
-- 执行前建议: 备份 messages 表

ALTER TABLE messages MODIFY COLUMN content MEDIUMTEXT NOT NULL;

-- 验证迁移结果
SELECT COLUMN_NAME, COLUMN_TYPE 
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = DATABASE() 
  AND TABLE_NAME = 'messages' 
  AND COLUMN_NAME = 'content';
