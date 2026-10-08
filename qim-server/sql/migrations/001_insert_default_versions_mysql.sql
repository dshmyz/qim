-- 插入默认客户端版本记录（MySQL 版本）
-- 用于支持 electron-updater 自动更新功能
-- 执行时间: 2026-06-05

-- 插入初始版本记录（如果不存在）
INSERT IGNORE INTO client_versions (version, platform, download_url, changelog, force_update, enabled, created_at, updated_at)
VALUES 
-- Windows 平台
('1.0.0', 'windows', 'https://api.qim.work/downloads/QIM-1.0.0.exe', '初始版本发布', false, true, NOW(), NOW()),

-- macOS 平台
('1.0.0', 'macos', 'https://api.qim.work/downloads/QIM-1.0.0.dmg', '初始版本发布', false, true, NOW(), NOW()),

-- Linux 平台
('1.0.0', 'linux', 'https://api.qim.work/downloads/QIM-1.0.0.AppImage', '初始版本发布', false, true, NOW(), NOW());

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
