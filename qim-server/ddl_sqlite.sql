-- DDL for QIM Server Database
-- SQLite DDL
-- Version: 1.0.0
-- Updated: 2026-05-05

-- Users table
CREATE TABLE IF NOT EXISTS `users` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `username` VARCHAR(50) NOT NULL UNIQUE,
  `password_hash` VARCHAR(255) NOT NULL,
  `nickname` VARCHAR(100),
  `avatar` VARCHAR(500),
  `type` VARCHAR(20) DEFAULT 'user',
  `signature` TEXT,
  `phone` VARCHAR(20),
  `email` VARCHAR(100),
  `status` VARCHAR(20) DEFAULT 'offline',
  `ip` VARCHAR(50),
  `two_factor_enabled` BOOLEAN DEFAULT FALSE,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_users_deleted_at` ON `users`(`deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_users_type` ON `users`(`type`);

-- Departments table
CREATE TABLE IF NOT EXISTS `departments` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(100) NOT NULL,
  `parent_id` INTEGER,
  `level` INTEGER NOT NULL,
  `path` VARCHAR(500),
  `sort_order` INTEGER DEFAULT 0,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_departments_deleted_at` ON `departments`(`deleted_at`);

-- Department employees table
CREATE TABLE IF NOT EXISTS `department_employees` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `department_id` INTEGER NOT NULL,
  `position` VARCHAR(100),
  `is_primary` BOOLEAN DEFAULT TRUE,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS `idx_department_employees_user_id` ON `department_employees`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_department_employees_department_id` ON `department_employees`(`department_id`);

-- Conversations table
CREATE TABLE IF NOT EXISTS `conversations` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `type` VARCHAR(20) NOT NULL,
  `name` VARCHAR(200),
  `avatar` VARCHAR(500),
  `creator_id` INTEGER,
  `last_message_id` INTEGER,
  `last_message_at` DATETIME,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Groups table
CREATE TABLE IF NOT EXISTS `groups` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `conversation_id` INTEGER NOT NULL UNIQUE,
  `group_type` VARCHAR(20) NOT NULL DEFAULT 'group',
  `name` VARCHAR(200) NOT NULL,
  `avatar` VARCHAR(500),
  `creator_id` INTEGER NOT NULL,
  `announcement` TEXT,
  `invite_permission` VARCHAR(20) DEFAULT 'owner_admin',
  `ai_config` TEXT,
  `approval_status` VARCHAR(20) DEFAULT 'approved',
  `applied_at` DATETIME,
  `approved_at` DATETIME,
  `approved_by` INTEGER,
  `reject_reason` TEXT,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS `idx_groups_name` ON `groups`(`name`);

-- Group documents table
CREATE TABLE IF NOT EXISTS `group_documents` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `group_id` INTEGER NOT NULL,
  `file_id` INTEGER NOT NULL,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS `idx_group_documents_group_id` ON `group_documents`(`group_id`);

-- Conversation members table
CREATE TABLE IF NOT EXISTS `conversation_members` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `conversation_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `role` VARCHAR(20) DEFAULT 'member',
  `unread_count` INTEGER DEFAULT 0,
  `muted` BOOLEAN DEFAULT FALSE,
  `last_read_at` DATETIME,
  `joined_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS `idx_conversation_members_conversation_id` ON `conversation_members`(`conversation_id`);
CREATE INDEX IF NOT EXISTS `idx_conversation_members_user_id` ON `conversation_members`(`user_id`);
CREATE UNIQUE INDEX IF NOT EXISTS `idx_user_conversation` ON `conversation_members`(`user_id`, `conversation_id`);

-- Messages table
CREATE TABLE IF NOT EXISTS `messages` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `conversation_id` INTEGER NOT NULL,
  `sender_id` INTEGER NOT NULL,
  `type` VARCHAR(20) NOT NULL,
  `content` TEXT NOT NULL,
  `quoted_message_id` INTEGER,
  `is_recalled` BOOLEAN DEFAULT FALSE,
  `is_read` BOOLEAN DEFAULT FALSE,
  `recalled_at` DATETIME,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_messages_conversation_id` ON `messages`(`conversation_id`);
CREATE INDEX IF NOT EXISTS `idx_messages_sender_id` ON `messages`(`sender_id`);
CREATE INDEX IF NOT EXISTS `idx_messages_deleted_at` ON `messages`(`deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_messages_conversation_created_at` ON `messages`(`conversation_id`, `created_at`);

-- Files table
CREATE TABLE IF NOT EXISTS `files` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `original_name` VARCHAR(255),
  `size` INTEGER NOT NULL,
  `mime_type` VARCHAR(100),
  `storage_path` VARCHAR(500) NOT NULL,
  `checksum` VARCHAR(64),
  `folder_id` INTEGER,
  `source` VARCHAR(20) DEFAULT 'upload',
  `source_id` VARCHAR(100),
  `is_starred` BOOLEAN DEFAULT FALSE,
  `starred_at` DATETIME,
  `tags` VARCHAR(500),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_files_user_id` ON `files`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_files_folder_id` ON `files`(`folder_id`);
CREATE INDEX IF NOT EXISTS `idx_files_deleted_at` ON `files`(`deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_files_is_starred` ON `files`(`is_starred`);
CREATE INDEX IF NOT EXISTS `idx_files_source` ON `files`(`source`);

-- Folders table
CREATE TABLE IF NOT EXISTS `folders` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `parent_id` INTEGER,
  `sort_order` INTEGER DEFAULT 0,
  `icon` VARCHAR(50),
  `color` VARCHAR(20),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_folders_user_id` ON `folders`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_folders_parent_id` ON `folders`(`parent_id`);
CREATE INDEX IF NOT EXISTS `idx_folders_deleted_at` ON `folders`(`deleted_at`);

-- Notes table
CREATE TABLE IF NOT EXISTS `notes` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `title` VARCHAR(500) NOT NULL,
  `content` TEXT NOT NULL,
  `type` VARCHAR(20) DEFAULT 'note',
  `style` TEXT DEFAULT '{}',
  `tags` TEXT,
  `summary` TEXT,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_notes_user_id` ON `notes`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_notes_deleted_at` ON `notes`(`deleted_at`);

-- Conversation sessions table
CREATE TABLE IF NOT EXISTS `conversation_sessions` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `conversation_id` INTEGER NOT NULL,
  `is_pinned` BOOLEAN DEFAULT FALSE,
  `is_hidden` BOOLEAN DEFAULT FALSE,
  `pinned_at` DATETIME,
  `hidden_at` DATETIME,
  `last_visited_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS `idx_user_conversation_session` ON `conversation_sessions`(`user_id`, `conversation_id`);
CREATE INDEX IF NOT EXISTS `idx_conversation_sessions_user_id` ON `conversation_sessions`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_conversation_sessions_conversation_id` ON `conversation_sessions`(`conversation_id`);

-- Message read receipts table
CREATE TABLE IF NOT EXISTS `message_read_receipts` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `message_id` INTEGER NOT NULL,
  `conversation_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS `idx_message_read_receipts_message_id` ON `message_read_receipts`(`message_id`);
CREATE INDEX IF NOT EXISTS `idx_message_read_receipts_conversation_id` ON `message_read_receipts`(`conversation_id`);
CREATE INDEX IF NOT EXISTS `idx_message_read_receipts_user_id` ON `message_read_receipts`(`user_id`);

-- Bots table
CREATE TABLE IF NOT EXISTS `bots` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(100) NOT NULL,
  `avatar` VARCHAR(500),
  `description` TEXT,
  `type` VARCHAR(50) NOT NULL,
  `config` TEXT,
  `is_active` BOOLEAN DEFAULT TRUE,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  `approval_status` VARCHAR(20) DEFAULT 'approved',
  `creator_id` INTEGER DEFAULT 0,
  `creator_name` VARCHAR(100) DEFAULT '',
  `virtual_user_id` INTEGER NULL,
  `reject_reason` TEXT,
  `is_template` BOOLEAN DEFAULT FALSE,
  `user_config_id` INTEGER,
  `use_system_config` BOOLEAN DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS `idx_bots_deleted_at` ON `bots`(`deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_bots_user_config_id` ON `bots`(`user_config_id`);
CREATE INDEX IF NOT EXISTS `idx_bots_virtual_user_id` ON `bots`(`virtual_user_id`);

-- Bot conversations table
CREATE TABLE IF NOT EXISTS `bot_conversations` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `bot_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `conversation_id` INTEGER NOT NULL,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS `idx_bot_conversations_bot_id` ON `bot_conversations`(`bot_id`);
CREATE INDEX IF NOT EXISTS `idx_bot_conversations_user_id` ON `bot_conversations`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_bot_conversations_conversation_id` ON `bot_conversations`(`conversation_id`);

-- AI usage logs table
CREATE TABLE IF NOT EXISTS `ai_usage_logs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `bot_id` INTEGER NOT NULL,
  `message_preview` VARCHAR(100),
  `call_type` VARCHAR(20),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS `idx_ai_usage_logs_user_id` ON `ai_usage_logs`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_ai_usage_logs_bot_id` ON `ai_usage_logs`(`bot_id`);

-- Events table
CREATE TABLE IF NOT EXISTS `events` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `title` VARCHAR(500) NOT NULL,
  `description` TEXT,
  `start` DATETIME NOT NULL,
  `end` DATETIME NOT NULL,
  `all_day` BOOLEAN DEFAULT FALSE,
  `reminder` INTEGER DEFAULT 0,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_events_user_id` ON `events`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_events_deleted_at` ON `events`(`deleted_at`);

-- Tasks table
CREATE TABLE IF NOT EXISTS `tasks` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `title` VARCHAR(500) NOT NULL,
  `description` TEXT,
  `due_date` DATETIME,
  `priority` VARCHAR(20) DEFAULT 'medium',
  `status` VARCHAR(20) DEFAULT 'todo',
  `assignee_id` VARCHAR(100),
  `tags` TEXT,
  `sub_tasks` TEXT,
  `position` INTEGER DEFAULT 0,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_tasks_user_id` ON `tasks`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_tasks_deleted_at` ON `tasks`(`deleted_at`);

-- User roles table
CREATE TABLE IF NOT EXISTS `user_roles` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `role` VARCHAR(50) NOT NULL,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS `idx_user_role` ON `user_roles`(`user_id`, `role`);
CREATE INDEX IF NOT EXISTS `idx_user_roles_user_id` ON `user_roles`(`user_id`);

-- System messages table
CREATE TABLE IF NOT EXISTS `system_messages` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `title` VARCHAR(500) NOT NULL,
  `content` TEXT NOT NULL,
  `sender_id` INTEGER NOT NULL,
  `status` VARCHAR(20) DEFAULT 'active',
  `target_type` VARCHAR(20),
  `target_id` INTEGER,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_system_messages_deleted_at` ON `system_messages`(`deleted_at`);

-- Mini apps table
CREATE TABLE IF NOT EXISTS `mini_apps` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `app_id` VARCHAR(100) NOT NULL UNIQUE,
  `name` VARCHAR(200) NOT NULL,
  `description` TEXT,
  `icon` VARCHAR(500),
  `path` VARCHAR(500),
  `status` VARCHAR(20) DEFAULT 'inactive',
  `permissions` TEXT,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_mini_apps_deleted_at` ON `mini_apps`(`deleted_at`);

-- Apps table
CREATE TABLE IF NOT EXISTS `apps` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `name` VARCHAR(200) NOT NULL,
  `icon` VARCHAR(500),
  `category` VARCHAR(100),
  `url` VARCHAR(500),
  `status` VARCHAR(20) DEFAULT 'active',
  `open_type` VARCHAR(20) DEFAULT 'in-app',
  `is_global` BOOLEAN DEFAULT FALSE,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_apps_user_id` ON `apps`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_apps_deleted_at` ON `apps`(`deleted_at`);

-- Notifications table
CREATE TABLE IF NOT EXISTS `notifications` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `type` VARCHAR(30) NOT NULL,
  `title` VARCHAR(500) NOT NULL,
  `content` TEXT NOT NULL,
  `read` BOOLEAN DEFAULT FALSE,
  `read_at` DATETIME,
  `priority` VARCHAR(10) DEFAULT 'normal',
  `action_type` VARCHAR(30) DEFAULT '',
  `action_payload` TEXT DEFAULT '',
  `pinned` BOOLEAN DEFAULT FALSE,
  `important` BOOLEAN DEFAULT FALSE,
  `handled` BOOLEAN DEFAULT FALSE,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_notifications_user_id` ON `notifications`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_notifications_deleted_at` ON `notifications`(`deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_notifications_user_read_created_at` ON `notifications`(`user_id`, `read`, `created_at`);

-- Channels table
CREATE TABLE IF NOT EXISTS `channels` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(200) NOT NULL,
  `description` TEXT,
  `avatar` VARCHAR(500),
  `creator_id` INTEGER NOT NULL,
  `status` VARCHAR(20) DEFAULT 'active',
  `publish_permission` VARCHAR(20) DEFAULT 'creator_only',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_channels_deleted_at` ON `channels`(`deleted_at`);

-- Channel subscribers table
CREATE TABLE IF NOT EXISTS `channel_subscribers` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `channel_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `joined_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS `idx_channel_user` ON `channel_subscribers`(`channel_id`, `user_id`);
CREATE INDEX IF NOT EXISTS `idx_channel_subscribers_channel_id` ON `channel_subscribers`(`channel_id`);
CREATE INDEX IF NOT EXISTS `idx_channel_subscribers_user_id` ON `channel_subscribers`(`user_id`);

-- Channel messages table
CREATE TABLE IF NOT EXISTS `channel_messages` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `channel_id` INTEGER NOT NULL,
  `sender_id` INTEGER NOT NULL,
  `content` TEXT NOT NULL,
  `type` VARCHAR(20) NOT NULL DEFAULT 'text',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_channel_messages_channel_id` ON `channel_messages`(`channel_id`);
CREATE INDEX IF NOT EXISTS `idx_channel_messages_deleted_at` ON `channel_messages`(`deleted_at`);

-- AI configs table
CREATE TABLE IF NOT EXISTS `ai_configs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `config_name` VARCHAR(50),
  `is_default` BOOLEAN DEFAULT FALSE,
  `provider` VARCHAR(50) DEFAULT 'openai',
  `config_json` TEXT,
  `api_key_encrypted` TEXT,
  `model_name` VARCHAR(50),
  `base_url` VARCHAR(255),
  `ai_enabled` BOOLEAN DEFAULT TRUE,
  `daily_limit` INTEGER DEFAULT 0,
  `max_tokens` INTEGER DEFAULT 1000,
  `temperature` DECIMAL(3,2) DEFAULT 0.70,
  `is_verified` BOOLEAN DEFAULT FALSE,
  `last_tested_at` DATETIME,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS `idx_ai_configs_user_id` ON `ai_configs`(`user_id`);

-- Sensitive words table
CREATE TABLE IF NOT EXISTS `sensitive_words` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `word` VARCHAR(100) NOT NULL UNIQUE,
  `level` VARCHAR(20) DEFAULT 'medium',
  `enabled` BOOLEAN DEFAULT TRUE,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_sensitive_words_deleted_at` ON `sensitive_words`(`deleted_at`);

-- System configs table
CREATE TABLE IF NOT EXISTS `system_configs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `key` VARCHAR(100) NOT NULL UNIQUE,
  `value` TEXT NOT NULL,
  `type` VARCHAR(20) DEFAULT 'string',
  `desc` VARCHAR(500),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Operation logs table
CREATE TABLE IF NOT EXISTS `operation_logs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `username` VARCHAR(100),
  `action` VARCHAR(100) NOT NULL,
  `module` VARCHAR(50),
  `ip` VARCHAR(50),
  `user_agent` TEXT,
  `request_url` VARCHAR(500),
  `request_body` TEXT,
  `response` TEXT,
  `duration` INTEGER,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS `idx_operation_logs_user_id` ON `operation_logs`(`user_id`);

-- Client versions table
CREATE TABLE IF NOT EXISTS `client_versions` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `version` VARCHAR(50) NOT NULL UNIQUE,
  `platform` VARCHAR(20) NOT NULL,
  `type` VARCHAR(20) DEFAULT 'full',
  `download_url` VARCHAR(500),
  `changelog` TEXT,
  `force_update` BOOLEAN DEFAULT FALSE,
  `enabled` BOOLEAN DEFAULT TRUE,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_client_versions_deleted_at` ON `client_versions`(`deleted_at`);

-- Blacklist table
CREATE TABLE IF NOT EXISTS `blacklists` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL UNIQUE,
  `reason` TEXT,
  `operator` VARCHAR(100),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS `idx_blacklists_user_id` ON `blacklists`(`user_id`);

-- Short links table
CREATE TABLE IF NOT EXISTS `short_links` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `original_url` TEXT NOT NULL,
  `code` VARCHAR(20) NOT NULL UNIQUE,
  `custom_code` VARCHAR(50),
  `expires_at` DATETIME,
  `password` VARCHAR(255),
  `visit_count` INTEGER DEFAULT 0,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_short_links_user_id` ON `short_links`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_short_links_deleted_at` ON `short_links`(`deleted_at`);

-- Approval configs table
CREATE TABLE IF NOT EXISTS `approval_configs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `type` VARCHAR(20) NOT NULL UNIQUE,
  `enabled` BOOLEAN DEFAULT FALSE,
  `description` VARCHAR(200),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Avatars table
CREATE TABLE IF NOT EXISTS `avatars` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `name` VARCHAR(100) NOT NULL,
  `avatar_url` VARCHAR(500) NOT NULL,
  `avatar_style` TEXT,
  `is_active` BOOLEAN DEFAULT TRUE,
  `is_default` BOOLEAN DEFAULT FALSE,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS `idx_avatars_user_id` ON `avatars`(`user_id`);

-- Messages Full-Text Search (FTS5) Virtual Table
-- This virtual table enables full-text search on message content for SQLite databases
-- Usage: SELECT * FROM messages_fts5 WHERE messages_fts5 MATCH 'search_term'
CREATE VIRTUAL TABLE IF NOT EXISTS `messages_fts5` USING fts5(
  `content`,
  `conversation_id`,
  `created_at`,
  tokenize='unicode61'
);
