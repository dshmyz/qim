-- DDL for QIM Server Database
-- SQLite DDL

-- Users table
CREATE TABLE IF NOT EXISTS `users` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `username` VARCHAR(50) NOT NULL UNIQUE,
  `password_hash` VARCHAR(255) NOT NULL,
  `nickname` VARCHAR(100),
  `avatar` VARCHAR(500),
  `signature` TEXT,
  `phone` VARCHAR(20),
  `email` VARCHAR(100),
  `status` VARCHAR(20) DEFAULT 'offline',
  `ip` VARCHAR(50),
  `two_factor_enabled` BOOLEAN DEFAULT FALSE,
  `created_at` DATETIME,
  `updated_at` DATETIME,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_users_deleted_at` ON `users`(`deleted_at`);

-- Departments table
CREATE TABLE IF NOT EXISTS `departments` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(100) NOT NULL,
  `parent_id` INTEGER,
  `level` INTEGER NOT NULL,
  `path` VARCHAR(500),
  `sort_order` INTEGER DEFAULT 0,
  `created_at` DATETIME,
  `updated_at` DATETIME,
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
  `created_at` DATETIME
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
  `created_at` DATETIME,
  `updated_at` DATETIME
);

-- Conversation members table
CREATE TABLE IF NOT EXISTS `conversation_members` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `conversation_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `role` VARCHAR(20) DEFAULT 'member',
  `unread_count` INTEGER DEFAULT 0,
  `muted` BOOLEAN DEFAULT FALSE,
  `last_read_at` DATETIME,
  `joined_at` DATETIME
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
  `file_id` INTEGER,
  `file_name` VARCHAR(255),
  `file_size` INTEGER,
  `quoted_message_id` INTEGER,
  `is_recalled` BOOLEAN DEFAULT FALSE,
  `is_read` BOOLEAN DEFAULT FALSE,
  `recalled_at` DATETIME,
  `created_at` DATETIME,
  `updated_at` DATETIME,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_messages_conversation_id` ON `messages`(`conversation_id`);
CREATE INDEX IF NOT EXISTS `idx_messages_sender_id` ON `messages`(`sender_id`);
CREATE INDEX IF NOT EXISTS `idx_messages_deleted_at` ON `messages`(`deleted_at`);

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
  `created_at` DATETIME,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_files_user_id` ON `files`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_files_deleted_at` ON `files`(`deleted_at`);

-- Folders table
CREATE TABLE IF NOT EXISTS `folders` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `parent_id` INTEGER,
  `created_at` DATETIME,
  `updated_at` DATETIME,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_folders_user_id` ON `folders`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_folders_deleted_at` ON `folders`(`deleted_at`);

-- Notes table
CREATE TABLE IF NOT EXISTS `notes` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `title` VARCHAR(500) NOT NULL,
  `content` TEXT NOT NULL,
  `color` VARCHAR(20) DEFAULT 'yellow',
  `type` VARCHAR(20) DEFAULT 'note',
  `created_at` DATETIME,
  `updated_at` DATETIME,
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
  `pinned_at` DATETIME,
  `last_visited_at` DATETIME,
  `created_at` DATETIME
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
  `created_at` DATETIME
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
  `created_at` DATETIME,
  `updated_at` DATETIME
);

-- Bot conversations table
CREATE TABLE IF NOT EXISTS `bot_conversations` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `bot_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `conversation_id` INTEGER NOT NULL,
  `created_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_bot_conversations_bot_id` ON `bot_conversations`(`bot_id`);
CREATE INDEX IF NOT EXISTS `idx_bot_conversations_user_id` ON `bot_conversations`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_bot_conversations_conversation_id` ON `bot_conversations`(`conversation_id`);

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
  `created_at` DATETIME,
  `updated_at` DATETIME,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_events_user_id` ON `events`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_events_deleted_at` ON `events`(`deleted_at`);

-- User roles table
CREATE TABLE IF NOT EXISTS `user_roles` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `role` VARCHAR(50) NOT NULL,
  `created_at` DATETIME
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
  `created_at` DATETIME,
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
  `status` VARCHAR(20) DEFAULT 'active',
  `created_at` DATETIME,
  `updated_at` DATETIME,
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
  `created_at` DATETIME,
  `updated_at` DATETIME,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_apps_user_id` ON `apps`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_apps_deleted_at` ON `apps`(`deleted_at`);

-- Notifications table
CREATE TABLE IF NOT EXISTS `notifications` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `user_id` INTEGER NOT NULL,
  `type` VARCHAR(20) NOT NULL,
  `title` VARCHAR(500) NOT NULL,
  `content` TEXT NOT NULL,
  `read` BOOLEAN DEFAULT FALSE,
  `read_at` DATETIME,
  `created_at` DATETIME,
  `updated_at` DATETIME,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_notifications_user_id` ON `notifications`(`user_id`);
CREATE INDEX IF NOT EXISTS `idx_notifications_deleted_at` ON `notifications`(`deleted_at`);

-- Channels table
CREATE TABLE IF NOT EXISTS `channels` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `name` VARCHAR(200) NOT NULL,
  `description` TEXT,
  `avatar` VARCHAR(500),
  `creator_id` INTEGER NOT NULL,
  `status` VARCHAR(20) DEFAULT 'active',
  `created_at` DATETIME,
  `updated_at` DATETIME,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_channels_deleted_at` ON `channels`(`deleted_at`);

-- Channel subscribers table
CREATE TABLE IF NOT EXISTS `channel_subscribers` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT,
  `channel_id` INTEGER NOT NULL,
  `user_id` INTEGER NOT NULL,
  `joined_at` DATETIME
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
  `created_at` DATETIME,
  `deleted_at` DATETIME
);

CREATE INDEX IF NOT EXISTS `idx_channel_messages_channel_id` ON `channel_messages`(`channel_id`);
CREATE INDEX IF NOT EXISTS `idx_channel_messages_deleted_at` ON `channel_messages`(`deleted_at`);
