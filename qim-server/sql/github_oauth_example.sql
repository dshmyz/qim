-- GitHub OAuth 示例配置
-- 注意：请替换 client_id 和 client_secret 为您自己的值

INSERT INTO auth_providers (
  name,
  type,
  enabled,
  priority,
  config,
  display_name,
  icon,
  created_at,
  updated_at
) VALUES (
  'github',
  'redirect',
  1,
  100,
  '{"client_id":"Ov23liYOUR_CLIENT_ID","client_secret":"d5e3f2YOUR_CLIENT_SECRET","auth_url":"https://github.com/login/oauth/authorize","token_url":"https://github.com/login/oauth/access_token","user_info_url":"https://api.github.com/user","redirect_url":"http://localhost:3000/oauth/callback","scope":"user:email"}',
  'GitHub登录',
  'fab fa-github',
  datetime('now'),
  datetime('now')
);

-- 查询验证
SELECT * FROM auth_providers WHERE name = 'github';
