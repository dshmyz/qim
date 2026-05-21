-- GitHub OAuth 配置
-- Client ID: Ov23lijk8o3GY2jK7dfG
-- Client Secret: 692e45f0f1df3ec173a46af3b119aa8347fb1c38

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
  '{"client_id":"Ov23lijk8o3GY2jK7dfG","client_secret":"692e45f0f1df3ec173a46af3b119aa8347fb1c38","auth_url":"https://github.com/login/oauth/authorize","token_url":"https://github.com/login/oauth/access_token","user_info_url":"https://api.github.com/user","redirect_url":"http://localhost:3000/oauth/callback","scope":"user:email"}',
  'GitHub登录',
  'fab fa-github',
  datetime('now'),
  datetime('now')
);

-- 查询验证
SELECT id, name, display_name, type, enabled, priority FROM auth_providers WHERE name = 'github';
