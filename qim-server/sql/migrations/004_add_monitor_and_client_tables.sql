-- 告警规则表
CREATE TABLE IF NOT EXISTS alert_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL,
    metric VARCHAR(50) NOT NULL,
    condition VARCHAR(10) NOT NULL,
    threshold REAL NOT NULL,
    duration INTEGER NOT NULL,
    notify_methods TEXT,
    notify_targets TEXT,
    enabled BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 告警历史表
CREATE TABLE IF NOT EXISTS alert_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id INTEGER NOT NULL,
    metric VARCHAR(50) NOT NULL,
    value REAL NOT NULL,
    status VARCHAR(20) NOT NULL,
    handled_at DATETIME,
    handler_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (rule_id) REFERENCES alert_rules(id)
);

CREATE INDEX IF NOT EXISTS idx_alert_history_rule_id ON alert_history(rule_id);

-- 崩溃日志表
CREATE TABLE IF NOT EXISTS crash_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    platform VARCHAR(50) NOT NULL,
    app_version VARCHAR(50) NOT NULL,
    device_model VARCHAR(100),
    os_version VARCHAR(50),
    error_stack TEXT,
    error_message TEXT,
    extra TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 用户反馈表
CREATE TABLE IF NOT EXISTS user_feedbacks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    priority VARCHAR(20) DEFAULT 'normal',
    screenshot VARCHAR(500),
    reply TEXT,
    handler_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_user_feedbacks_deleted_at ON user_feedbacks(deleted_at);
