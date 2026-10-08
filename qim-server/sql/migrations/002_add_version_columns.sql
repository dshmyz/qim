-- 增量迁移：为 client_versions 表补充缺失字段和索引
-- 适用于已部署的旧库，新库无需执行（DDL 已包含这些字段）

-- MySQL 语法（SQLite 不支持 IF NOT EXISTS 的 ALTER TABLE ADD COLUMN，需手动调整）

-- 补充 sha512 字段
ALTER TABLE `client_versions` ADD COLUMN IF NOT EXISTS `sha512` VARCHAR(200);

-- 补充 file_size 字段
ALTER TABLE `client_versions` ADD COLUMN IF NOT EXISTS `file_size` BIGINT DEFAULT 0;

-- 补充 rollout_percentage 字段（灰度发布百分比，0-100，100=全量）
ALTER TABLE `client_versions` ADD COLUMN IF NOT EXISTS `rollout_percentage` INT DEFAULT 100;

-- 补充 min_version 字段（最低兼容版本）
ALTER TABLE `client_versions` ADD COLUMN IF NOT EXISTS `min_version` VARCHAR(50);

-- 添加 version+platform 联合唯一索引（如果不存在）
-- 注意：旧的 idx_version_deleted 唯一索引仅含 version+deleted_at，需手动删除
-- ALTER TABLE `client_versions` DROP INDEX IF EXISTS `idx_version_deleted`;
-- CREATE UNIQUE INDEX `idx_version_platform` ON `client_versions`(`version`, `platform`, `deleted_at`);
