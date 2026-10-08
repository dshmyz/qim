-- 插入默认客户端版本记录
-- 用于支持 electron-updater 自动更新功能
-- 执行时间: 2026-06-05

-- 插入初始版本记录（如果不存在）
INSERT OR IGNORE INTO client_versions (version, platform, download_url, changelog, force_update, enabled, created_at, updated_at)
VALUES 
-- Windows 平台
('1.0.0', 'windows', 'https://api.qim.work/downloads/QIM-1.0.0.exe', '初始版本发布', false, true, datetime('now'), datetime('now')),

-- macOS 平台
('1.0.0', 'macos', 'https://api.qim.work/downloads/QIM-1.0.0.dmg', '初始版本发布', false, true, datetime('now'), datetime('now')),

-- Linux 平台
('1.0.0', 'linux', 'https://api.qim.work/downloads/QIM-1.0.0.AppImage', '初始版本发布', false, true, datetime('now'), datetime('now'));

-- 查询验证
SELECT 
    id,
    version,
    platform,
    download_url,
    enabled,
    created_at
FROM client_versions
WHERE enabled = true
ORDER BY created_at DESC;
