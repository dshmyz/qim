-- DDL for QIM Server Database
-- SQLite DDL
-- Version: 2.0.0
-- Updated: 2026-05-25

-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- Users table
-- type: 'user' | 'bot_assistant' | 'bot_avatar' | 'system' | 'api' | 'admin'
CREATE TABLE IF NOT EXISTS `users` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `username` VARCHAR(50) NOT NULL UNIQUE,
  `password_hash` VARCHAR(255) NOT NULL,
  `nickname` VARCHAR(100),
  `avatar` VARCHAR(500),
  `type` VARCHAR(30) DEFAULT 'user',
  `signature` TEXT,
  `phone` VARCHAR(20),
  `email` VARCHAR(100),
  `status` VARCHAR(20) DEFAULT 'offline',
  `last_online` DATETIME,
  `ip` VARCHAR(50),
  `two_factor_enabled` INTEGER DEFAULT 0,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);
CREATE INDEX IF NOT EXISTS `idx_users_deleted_at` ON `users`(`deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_users_type` ON `users`(`type`);
CREATE INDEX IF NOT EXISTS `idx_users_phone` ON `users`(`phone`);
CREATE INDEX IF NOT EXISTS `idx_users_email` ON `users`(`email`);

-- Departments table
CREATE TABLE IF NOT EXISTS `departments` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(100) NOT NULL,
  `external_id` VARCHAR(200),
  `parent_id` INTEGER,
  `level` INTEGER NOT NULL,
  `path` VARCHAR(500),
  `sort_order` INTEGER DEFAULT 0,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  FOREIGN KEY (`parent_id`) REFERENCES `departments`(`id`) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS `idx_departments_deleted_at` ON `departments`(`deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_departments_external_id` ON `departments`(`external_id`);

-- Department employees table
CREATE TABLE IF NOT EXISTS `department_employees` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `department_id` INTEGER NOT NULL,
  `position` VARCHAR(100),
  `is_primary` INTEGER DEFAULT 1,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`department_id`) REFERENCES `departments`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_department_employees_user_id` ON `department_employees`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_department_employees_department_id` ON `department_employees`(`department_id`);

-- Conversations table
CREATE TABLE IF NOT EXISTS `conversations` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `type` VARCHAR(20) NOT NULL,
  `is_deleted` INTEGER DEFAULT 0,
  `last_message_id` INTEGER,
  `last_message_at` DATETIME,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS `idx_conv_type_deleted` ON `conversations`(`type`, `is_deleted`);

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
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`conversation_id`) REFERENCES `conversations`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`creator_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_groups_name` ON `groups`(`name`);

-- Group documents table
CREATE TABLE IF NOT EXISTS `group_documents` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `group_id` INTEGER NOT NULL,
  `file_id` INTEGER NOT NULL,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`group_id`) REFERENCES `groups`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`file_id`) REFERENCES `files`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_group_documents_group_id` ON `group_documents`(`group_id`);

-- Conversation members table
CREATE TABLE IF NOT EXISTS `conversation_members` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `conversation_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `role` VARCHAR(20) DEFAULT 'member',
  `unread_count` INTEGER DEFAULT 0,
  `muted` INTEGER DEFAULT 0,
  `last_read_at` DATETIME,
  `joined_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`conversation_id`) REFERENCES `conversations`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS `idx_conv_member_conv_user` ON `conversation_members`(`conversation_id`, `user_id`);
CREATE INDEX IF NOT EXISTS `idx_conv_member_user` ON `conversation_members`(`user_id`, `conversation_id`);

-- Messages table
CREATE TABLE IF NOT EXISTS `messages` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `conversation_id` INTEGER NOT NULL,
  `sender_id` INTEGER NOT NULL,
  `type` VARCHAR(20) NOT NULL,
  `content` TEXT NOT NULL,
  `quoted_message_id` INTEGER,
  `is_recalled` INTEGER DEFAULT 0,
  `is_read` INTEGER DEFAULT 0,
  `ai_type` VARCHAR(30) DEFAULT '',
  `recalled_at` DATETIME,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  FOREIGN KEY (`conversation_id`) REFERENCES `conversations`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`sender_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`quoted_message_id`) REFERENCES `messages`(`id`) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS `idx_messages_sender_id` ON `messages`(`sender_id`);
CREATE INDEX IF NOT EXISTS `idx_messages_deleted_at` ON `messages`(`deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_msg_conv_created` ON `messages`(`conversation_id`, `created_at`);
CREATE INDEX IF NOT EXISTS `idx_msg_conv_read_sender` ON `messages`(`conversation_id`, `is_read`, `sender_id`);

-- Files table
CREATE TABLE IF NOT EXISTS `files` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `original_name` VARCHAR(255),
  `size` BIGINT NOT NULL,
  `mime_type` VARCHAR(100),
  `storage_path` VARCHAR(500) NOT NULL,
  `checksum` VARCHAR(64),
  `folder_id` INTEGER,
  `source` VARCHAR(20) DEFAULT 'upload',
  `source_id` VARCHAR(100),
  `is_starred` INTEGER DEFAULT 0,
  `starred_at` DATETIME,
  `tags` VARCHAR(500),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`folder_id`) REFERENCES `folders`(`id`) ON DELETE SET NULL
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
  `deleted_at` DATETIME,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`parent_id`) REFERENCES `folders`(`id`) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS `idx_folders_user_id` ON `folders`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_folders_parent_id` ON `folders`(`parent_id`);
CREATE INDEX IF NOT EXISTS `idx_folders_deleted_at` ON `folders`(`deleted_at`);

-- Approvals table
-- 统一的审批表，支持多种类型的审批流程
CREATE TABLE IF NOT EXISTS `approvals` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `target_type` VARCHAR(30) NOT NULL,
  `target_id` INTEGER NOT NULL,
  `status` VARCHAR(20) DEFAULT 'pending',
  `applied_at` DATETIME NOT NULL,
  `applied_by` INTEGER NOT NULL,
  `approved_at` DATETIME,
  `approved_by` INTEGER,
  `reject_reason` TEXT,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`applied_by`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`approved_by`) REFERENCES `users`(`id`) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS `idx_approvals_target` ON `approvals`(`target_type`, `target_id`);
CREATE INDEX IF NOT EXISTS `idx_approvals_status` ON `approvals`(`status`);
CREATE INDEX IF NOT EXISTS `idx_approvals_applied_by` ON `approvals`(`applied_by`);

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
  `deleted_at` DATETIME,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_notes_user_id` ON `notes`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_notes_deleted_at` ON `notes`(`deleted_at`);

-- Conversation sessions table
CREATE TABLE IF NOT EXISTS `conversation_sessions` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `conversation_id` INTEGER NOT NULL,
  `is_pinned` INTEGER DEFAULT 0,
  `is_hidden` INTEGER DEFAULT 0,
  `pinned_at` DATETIME,
  `hidden_at` DATETIME,
  `last_visited_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`conversation_id`) REFERENCES `conversations`(`id`) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS `idx_user_conv` ON `conversation_sessions`(`user_id`, `conversation_id`);
CREATE INDEX IF NOT EXISTS `idx_conversation_sessions_user_id` ON `conversation_sessions`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_conversation_sessions_conversation_id` ON `conversation_sessions`(`conversation_id`);

-- Message read receipts table
CREATE TABLE IF NOT EXISTS `message_read_receipts` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `message_id` INTEGER NOT NULL,
  `conversation_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`conversation_id`) REFERENCES `conversations`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS `idx_message_user_receipt` ON `message_read_receipts`(`message_id`, `user_id`);
CREATE INDEX IF NOT EXISTS `idx_message_read_receipts_conversation_id` ON `message_read_receipts`(`conversation_id`);

-- Bots table
CREATE TABLE IF NOT EXISTS `bots` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(100) NOT NULL,
  `avatar` VARCHAR(500),
  `description` TEXT,
  `type` VARCHAR(50) NOT NULL,
  `config` TEXT,
  `is_active` INTEGER DEFAULT 1,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  `creator_id` INTEGER DEFAULT 0,
  `creator_name` VARCHAR(100) DEFAULT '',
  `virtual_user_id` INTEGER,
  `group_id` INTEGER,
  `is_template` INTEGER DEFAULT 0,
  `user_config_id` INTEGER,
  `use_system_config` INTEGER DEFAULT 1,
  FOREIGN KEY (`virtual_user_id`) REFERENCES `users`(`id`) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS `idx_bots_deleted_at` ON `bots`(`deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_bots_user_config_id` ON `bots`(`user_config_id`);
CREATE INDEX IF NOT EXISTS `idx_bots_virtual_user_id` ON `bots`(`virtual_user_id`);
CREATE INDEX IF NOT EXISTS `idx_bots_group_id` ON `bots`(`group_id`);

-- Bot conversations table
CREATE TABLE IF NOT EXISTS `bot_conversations` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `bot_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `conversation_id` INTEGER NOT NULL,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`bot_id`) REFERENCES `bots`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`conversation_id`) REFERENCES `conversations`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_bot_conversations_bot_id` ON `bot_conversations`(`bot_id`);
CREATE INDEX IF NOT EXISTS `idx_bot_conversations_user_id` ON `bot_conversations`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_bot_conversations_conversation_id` ON `bot_conversations`(`conversation_id`);

-- AI usage logs table
CREATE TABLE IF NOT EXISTS `ai_usage_logs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `bot_id` INTEGER NOT NULL,
  `provider` VARCHAR(64),
  `model` VARCHAR(128),
  `task_type` VARCHAR(64),
  `message_preview` VARCHAR(100),
  `call_type` VARCHAR(20),
  `tokens_in` INTEGER DEFAULT 0,
  `tokens_out` INTEGER DEFAULT 0,
  `duration` BIGINT DEFAULT 0,
  `status` VARCHAR(32),
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
  `start_time` DATETIME NOT NULL,
  `end_time` DATETIME NOT NULL,
  `all_day` INTEGER DEFAULT 0,
  `reminder` INTEGER DEFAULT 0,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
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
  `deleted_at` DATETIME,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_tasks_user_id` ON `tasks`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_tasks_deleted_at` ON `tasks`(`deleted_at`);

-- User roles table
CREATE TABLE IF NOT EXISTS `user_roles` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `role` VARCHAR(50) NOT NULL,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
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
  `deleted_at` DATETIME,
  FOREIGN KEY (`sender_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
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
  `code` VARCHAR(100),
  `icon` VARCHAR(500),
  `category` VARCHAR(100),
  `url` VARCHAR(500),
  `status` VARCHAR(20) DEFAULT 'active',
  `open_type` VARCHAR(20) DEFAULT 'in-app',
  `is_global` INTEGER DEFAULT 0,
  `scope_type` VARCHAR(20) DEFAULT 'all',
  `scope_value` VARCHAR(1000),
  `available_org_ids` VARCHAR(1000),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_apps_user_id` ON `apps`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_apps_code` ON `apps`(`code`);
CREATE INDEX IF NOT EXISTS `idx_apps_deleted_at` ON `apps`(`deleted_at`);

-- Notifications table
CREATE TABLE IF NOT EXISTS `notifications` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `type` VARCHAR(30) NOT NULL,
  `title` VARCHAR(500) NOT NULL,
  `content` TEXT NOT NULL,
  `is_read` INTEGER DEFAULT 0,
  `read_at` DATETIME,
  `priority` VARCHAR(10) DEFAULT 'normal',
  `action_type` VARCHAR(30) DEFAULT '',
  `action_payload` TEXT DEFAULT '',
  `pinned` INTEGER DEFAULT 0,
  `important` INTEGER DEFAULT 0,
  `handled` INTEGER DEFAULT 0,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_notifications_user_id` ON `notifications`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_notifications_deleted_at` ON `notifications`(`deleted_at`);
CREATE INDEX IF NOT EXISTS `idx_notifications_user_is_read_created_at` ON `notifications`(`user_id`, `is_read`, `created_at`);

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
  `deleted_at` DATETIME,
  FOREIGN KEY (`creator_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_channels_deleted_at` ON `channels`(`deleted_at`);

-- Channel subscribers table
CREATE TABLE IF NOT EXISTS `channel_subscribers` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `channel_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `joined_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
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
  `deleted_at` DATETIME,
  FOREIGN KEY (`channel_id`) REFERENCES `channels`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`sender_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_channel_messages_channel_id` ON `channel_messages`(`channel_id`);
CREATE INDEX IF NOT EXISTS `idx_channel_messages_deleted_at` ON `channel_messages`(`deleted_at`);

-- Channel message likes table
CREATE TABLE IF NOT EXISTS `channel_message_likes` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `message_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`message_id`) REFERENCES `channel_messages`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS `idx_channel_msg_like_user` ON `channel_message_likes`(`message_id`, `user_id`);

-- Channel message comments table
CREATE TABLE IF NOT EXISTS `channel_message_comments` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `message_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `content` TEXT NOT NULL,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  FOREIGN KEY (`message_id`) REFERENCES `channel_messages`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_channel_msg_comments_message_id` ON `channel_message_comments`(`message_id`);
CREATE INDEX IF NOT EXISTS `idx_channel_msg_comments_deleted_at` ON `channel_message_comments`(`deleted_at`);

-- AI configs table
CREATE TABLE IF NOT EXISTS `ai_configs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `config_name` VARCHAR(50),
  `is_default` INTEGER DEFAULT 0,
  `provider` VARCHAR(50) DEFAULT 'openai',
  `config_json` TEXT,
  `api_key_encrypted` TEXT,
  `model_name` VARCHAR(50),
  `base_url` VARCHAR(255),
  `ai_enabled` INTEGER DEFAULT 1,
  `daily_limit` INTEGER DEFAULT 0,
  `max_tokens` INTEGER DEFAULT 1000,
  `temperature` REAL DEFAULT 0.70,
  `is_verified` INTEGER DEFAULT 0,
  `last_tested_at` DATETIME,
  `overrides` TEXT,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_ai_configs_user_id` ON `ai_configs`(`user_id`);

-- AI providers table
CREATE TABLE IF NOT EXISTS `ai_providers` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(100) NOT NULL,
  `provider` VARCHAR(50) NOT NULL,
  `api_type` VARCHAR(20) NOT NULL,
  `endpoint` VARCHAR(500),
  `api_key` VARCHAR(500),
  `models` TEXT,
  `enabled` INTEGER DEFAULT 1,
  `status` VARCHAR(20) DEFAULT 'connected',
  `priority` INTEGER DEFAULT 0,
  `config` TEXT,
  `last_test_at` DATETIME,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Sensitive words table
CREATE TABLE IF NOT EXISTS `sensitive_words` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `word` VARCHAR(100) NOT NULL UNIQUE,
  `level` VARCHAR(20) DEFAULT 'medium',
  `enabled` INTEGER DEFAULT 1,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);
CREATE INDEX IF NOT EXISTS `idx_sensitive_words_deleted_at` ON `sensitive_words`(`deleted_at`);

-- System configs table
CREATE TABLE IF NOT EXISTS `system_configs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `config_key` VARCHAR(100) NOT NULL UNIQUE,
  `value` TEXT NOT NULL,
  `type` VARCHAR(20) DEFAULT 'string',
  `description` VARCHAR(500),
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
  `force_update` INTEGER DEFAULT 0,
  `enabled` INTEGER DEFAULT 1,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);
CREATE INDEX IF NOT EXISTS `idx_client_versions_deleted_at` ON `client_versions`(`deleted_at`);

-- Blacklists table
CREATE TABLE IF NOT EXISTS `blacklists` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL UNIQUE,
  `reason` TEXT,
  `operator` VARCHAR(100),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
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
  `deleted_at` DATETIME,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_short_links_user_id` ON `short_links`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_short_links_custom_code` ON `short_links`(`custom_code`);
CREATE INDEX IF NOT EXISTS `idx_short_links_deleted_at` ON `short_links`(`deleted_at`);

-- Approval configs table
CREATE TABLE IF NOT EXISTS `approval_configs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `type` VARCHAR(20) NOT NULL UNIQUE,
  `enabled` INTEGER DEFAULT 0,
  `description` VARCHAR(200),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Auth providers table
CREATE TABLE IF NOT EXISTS `auth_providers` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(50) NOT NULL UNIQUE,
  `protocol` VARCHAR(20) NOT NULL DEFAULT 'ldap',
  `type` VARCHAR(20) NOT NULL,
  `enabled` INTEGER DEFAULT 1,
  `priority` INTEGER DEFAULT 100,
  `config` TEXT,
  `display_name` VARCHAR(100),
  `icon` VARCHAR(200),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);
CREATE INDEX IF NOT EXISTS `idx_auth_providers_deleted_at` ON `auth_providers`(`deleted_at`);

-- Org sync configs table
CREATE TABLE IF NOT EXISTS `org_sync_configs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(50) NOT NULL UNIQUE,
  `enabled` INTEGER DEFAULT 1,
  `sync_type` VARCHAR(20) NOT NULL,
  `schedule` VARCHAR(100),
  `config` TEXT,
  `last_sync_at` DATETIME,
  `last_sync_status` VARCHAR(20),
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);
CREATE INDEX IF NOT EXISTS `idx_org_sync_configs_deleted_at` ON `org_sync_configs`(`deleted_at`);

-- Org sync logs table
CREATE TABLE IF NOT EXISTS `org_sync_logs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `config_id` INTEGER NOT NULL,
  `status` VARCHAR(20) NOT NULL,
  `started_at` DATETIME NOT NULL,
  `finished_at` DATETIME,
  `stats` TEXT,
  `error_message` TEXT,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);
CREATE INDEX IF NOT EXISTS `idx_org_sync_logs_config_id` ON `org_sync_logs`(`config_id`);
CREATE INDEX IF NOT EXISTS `idx_org_sync_logs_deleted_at` ON `org_sync_logs`(`deleted_at`);

-- Alert rules table
CREATE TABLE IF NOT EXISTS `alert_rules` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(255) NOT NULL,
  `metric` VARCHAR(255) NOT NULL,
  `condition` VARCHAR(255) NOT NULL,
  `threshold` REAL NOT NULL,
  `duration` INTEGER NOT NULL,
  `notify_methods` TEXT,
  `notify_targets` TEXT,
  `enabled` INTEGER DEFAULT 1,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Alert history table
CREATE TABLE IF NOT EXISTS `alert_history` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `rule_id` INTEGER NOT NULL,
  `metric` VARCHAR(255) NOT NULL,
  `value` REAL NOT NULL,
  `status` VARCHAR(255) NOT NULL,
  `handled_at` DATETIME,
  `handler_id` INTEGER,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`rule_id`) REFERENCES `alert_rules`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_alert_history_rule_id` ON `alert_history`(`rule_id`);

-- Crash logs table
CREATE TABLE IF NOT EXISTS `crash_logs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER,
  `platform` VARCHAR(255) NOT NULL,
  `app_version` VARCHAR(255) NOT NULL,
  `device_model` VARCHAR(255),
  `os_version` VARCHAR(255),
  `error_stack` TEXT,
  `error_message` TEXT,
  `extra` TEXT,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS `idx_crash_logs_user_id` ON `crash_logs`(`user_id`);

-- User feedbacks table
CREATE TABLE IF NOT EXISTS `user_feedbacks` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER,
  `type` VARCHAR(255) NOT NULL,
  `content` TEXT NOT NULL,
  `status` VARCHAR(50) DEFAULT 'pending',
  `priority` VARCHAR(50) DEFAULT 'normal',
  `screenshot` VARCHAR(500),
  `reply` TEXT,
  `handler_id` INTEGER,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);
CREATE INDEX IF NOT EXISTS `idx_user_feedbacks_user_id` ON `user_feedbacks`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_user_feedbacks_deleted_at` ON `user_feedbacks`(`deleted_at`);

-- File chunks table
CREATE TABLE IF NOT EXISTS `file_chunks` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `upload_id` VARCHAR(64) NOT NULL,
  `file_hash` VARCHAR(64) NOT NULL,
  `chunk_index` INTEGER NOT NULL,
  `chunk_hash` VARCHAR(64) NOT NULL,
  `chunk_size` BIGINT NOT NULL,
  `storage_path` VARCHAR(500) NOT NULL,
  `status` VARCHAR(20) NOT NULL DEFAULT 'pending',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME
);
CREATE INDEX IF NOT EXISTS `idx_upload_chunk` ON `file_chunks`(`upload_id`, `chunk_index`);
CREATE INDEX IF NOT EXISTS `idx_file_chunks_deleted_at` ON `file_chunks`(`deleted_at`);

-- Upload tasks table
CREATE TABLE IF NOT EXISTS `upload_tasks` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `upload_id` VARCHAR(64) NOT NULL UNIQUE,
  `user_id` INTEGER NOT NULL,
  `filename` VARCHAR(255) NOT NULL,
  `file_size` BIGINT NOT NULL,
  `file_hash` VARCHAR(64) NOT NULL,
  `total_chunks` INTEGER NOT NULL,
  `uploaded_chunks` INTEGER DEFAULT 0,
  `folder_id` INTEGER,
  `status` VARCHAR(20) NOT NULL DEFAULT 'pending',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`folder_id`) REFERENCES `folders`(`id`) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS `idx_upload_tasks_user_id` ON `upload_tasks`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_upload_tasks_folder_id` ON `upload_tasks`(`folder_id`);
CREATE INDEX IF NOT EXISTS `idx_upload_tasks_deleted_at` ON `upload_tasks`(`deleted_at`);

-- Avatar configs table
CREATE TABLE IF NOT EXISTS `avatar_configs` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL UNIQUE,
  `name` VARCHAR(100) DEFAULT '我的分身',
  `enabled` INTEGER DEFAULT 0,
  `auto_learned_persona` TEXT,
  `custom_persona_addon` TEXT,
  `persona_version` INTEGER DEFAULT 0,
  `last_learned_at` DATETIME,
  `knowledge_scope_json` TEXT,
  `trigger_rules_json` TEXT,
  `reply_strategy_json` TEXT,
  `model_config_id` INTEGER,
  `use_system_config` INTEGER DEFAULT 1,
  `takeover_cooldown` INTEGER DEFAULT 10,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`model_config_id`) REFERENCES `ai_configs`(`id`) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS `idx_avatar_configs_deleted_at` ON `avatar_configs`(`deleted_at`);

-- Avatar sessions table
CREATE TABLE IF NOT EXISTS `avatar_sessions` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `conversation_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `avatar_enabled` INTEGER DEFAULT 0,
  `takeover_until` DATETIME,
  `last_reply_at` DATETIME,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`conversation_id`) REFERENCES `conversations`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS `idx_avatar_user_conv` ON `avatar_sessions`(`user_id`, `conversation_id`);

-- Avatar tool bindings table
CREATE TABLE IF NOT EXISTS `avatar_tool_bindings` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `avatar_id` INTEGER NOT NULL,
  `tool_id` VARCHAR(64),
  `enabled` INTEGER DEFAULT 1,
  `priority` INTEGER DEFAULT 1,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`avatar_id`) REFERENCES `avatar_configs`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_avatar_tool_bindings_avatar_id` ON `avatar_tool_bindings`(`avatar_id`);

-- Avatar learn tasks table
CREATE TABLE IF NOT EXISTS `avatar_learn_tasks` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `status` VARCHAR(20) DEFAULT 'pending',
  `progress` INTEGER DEFAULT 0,
  `message_count` INTEGER DEFAULT 0,
  `error` TEXT,
  `started_at` DATETIME,
  `completed_at` DATETIME,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_avatar_learn_tasks_user_id` ON `avatar_learn_tasks`(`user_id`);

-- Document process statuses table
CREATE TABLE IF NOT EXISTS `document_process_statuses` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `group_doc_id` INTEGER NOT NULL,
  `status` VARCHAR(20) DEFAULT 'pending',
  `error` TEXT,
  `chunk_count` INTEGER DEFAULT 0,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`group_doc_id`) REFERENCES `group_documents`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_doc_process_statuses_group_doc_id` ON `document_process_statuses`(`group_doc_id`);

-- Realtime sessions table
CREATE TABLE IF NOT EXISTS `realtime_sessions` (
  `id` VARCHAR(36) PRIMARY KEY,
  `type` VARCHAR(20) NOT NULL,
  `initiator_id` INTEGER NOT NULL,
  `conversation_id` INTEGER NOT NULL,
  `status` VARCHAR(20) DEFAULT 'pending',
  `started_at` DATETIME,
  `ended_at` DATETIME,
  `metadata` TEXT,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (`initiator_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`conversation_id`) REFERENCES `conversations`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_realtime_sessions_type` ON `realtime_sessions`(`type`);
CREATE INDEX IF NOT EXISTS `idx_realtime_sessions_initiator_id` ON `realtime_sessions`(`initiator_id`);
CREATE INDEX IF NOT EXISTS `idx_realtime_sessions_conversation_id` ON `realtime_sessions`(`conversation_id`);

-- Realtime participants table
CREATE TABLE IF NOT EXISTS `realtime_participants` (
  `id` VARCHAR(36) PRIMARY KEY,
  `session_id` VARCHAR(36) NOT NULL,
  `user_id` INTEGER NOT NULL,
  `role` VARCHAR(20) DEFAULT 'viewer',
  `status` VARCHAR(20) DEFAULT 'pending',
  `requested_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `approved_at` DATETIME,
  `joined_at` DATETIME,
  `left_at` DATETIME,
  FOREIGN KEY (`session_id`) REFERENCES `realtime_sessions`(`id`) ON DELETE CASCADE,
  FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS `idx_realtime_participants_session_id` ON `realtime_participants`(`session_id`);
CREATE INDEX IF NOT EXISTS `idx_realtime_participants_user_id` ON `realtime_participants`(`user_id`);