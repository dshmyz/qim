-- Auth and Org Sync Tables
CREATE TABLE IF NOT EXISTS auth_providers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  type TEXT NOT NULL,
  enabled INTEGER DEFAULT 1,
  priority INTEGER DEFAULT 100,
  config TEXT,
  display_name TEXT,
  icon TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  deleted_at DATETIME
);

CREATE TABLE IF NOT EXISTS org_sync_configs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  enabled INTEGER DEFAULT 1,
  sync_type TEXT NOT NULL,
  schedule TEXT,
  config TEXT,
  last_sync_at DATETIME,
  last_sync_status TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  deleted_at DATETIME
);

CREATE TABLE IF NOT EXISTS org_sync_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  config_id INTEGER NOT NULL,
  status TEXT NOT NULL,
  started_at DATETIME NOT NULL,
  finished_at DATETIME,
  stats TEXT,
  error_message TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  deleted_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_auth_providers_deleted_at ON auth_providers(deleted_at);
CREATE INDEX IF NOT EXISTS idx_external_user_mappings_user_id ON external_user_mappings(user_id);
CREATE INDEX IF NOT EXISTS idx_external_user_mappings_deleted_at ON external_user_mappings(deleted_at);
CREATE INDEX IF NOT EXISTS idx_org_sync_configs_deleted_at ON org_sync_configs(deleted_at);
CREATE INDEX IF NOT EXISTS idx_org_sync_logs_config_id ON org_sync_logs(config_id);
CREATE INDEX IF NOT EXISTS idx_org_sync_logs_deleted_at ON org_sync_logs(deleted_at);
