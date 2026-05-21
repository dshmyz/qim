# GitHub OAuth 集成指南

## 一、在GitHub创建OAuth应用

### 1. 访问GitHub开发者设置
访问：https://github.com/settings/developers

### 2. 创建新的OAuth应用
点击 "OAuth Apps" → "New OAuth App"

填写以下信息：
```
Application name:     QIM Desktop
Homepage URL:         http://localhost:3000
Authorization callback URL: http://localhost:3000/oauth/callback
Application description: QIM桌面应用GitHub登录
```

### 3. 获取凭证
创建成功后，您会获得：
- **Client ID**: 类似 `Ov23li...`
- **Client Secret**: 点击"Generate a new client secret"生成，类似 `d5e3f2...`

**⚠️ 重要：妥善保管Client Secret，不要泄露！**

---

## 二、在管理后台配置

### 方式一：使用配置模板（推荐）

1. 登录管理后台：http://localhost:3008
2. 进入"系统设置" → "认证配置"
3. 点击"新建认证提供者"
4. 在"配置模板"下拉框中选择"OAuth2.0 配置"
5. 修改以下字段：

| 字段 | 值 |
|------|-----|
| 名称 | github |
| 显示名称 | GitHub登录 |
| 认证类型 | 重定向认证（已自动选择） |
| 优先级 | 100 |
| 图标 | fab fa-github |

6. 修改配置JSON：

```json
{
  "client_id": "Ov23liYOUR_CLIENT_ID",
  "client_secret": "d5e3f2YOUR_CLIENT_SECRET",
  "auth_url": "https://github.com/login/oauth/authorize",
  "token_url": "https://github.com/login/oauth/access_token",
  "user_info_url": "https://api.github.com/user",
  "redirect_url": "http://localhost:3000/oauth/callback",
  "scope": "user:email"
}
```

**注意**：
- 替换 `client_id` 和 `client_secret` 为您自己的值
- `redirect_url` 需要与GitHub OAuth应用中配置的一致

7. 点击"确定"保存

### 方式二：直接执行SQL

```bash
cd qim-server
# 编辑 github_oauth_example.sql，替换 client_id 和 client_secret
sqlite3 qim.db < github_oauth_example.sql
```

---

## 三、配置字段说明

### OAuth配置字段

| 字段 | 说明 | GitHub值 |
|------|------|----------|
| client_id | GitHub OAuth应用ID | 从GitHub获取 |
| client_secret | GitHub OAuth密钥 | 从GitHub获取 |
| auth_url | 授权页面URL | https://github.com/login/oauth/authorize |
| token_url | 获取token的URL | https://github.com/login/oauth/access_token |
| user_info_url | 获取用户信息的URL | https://api.github.com/user |
| redirect_url | 回调地址 | http://localhost:3000/oauth/callback |
| scope | 权限范围 | user:email |

### scope权限说明

- `user:email` - 读取用户邮箱（推荐）
- `user` - 读取用户基本信息
- `read:user` - 只读用户信息
- `repo` - 访问仓库（如需要）

---

## 四、测试集成

### 1. 重启后端服务
```bash
cd qim-server
go run main.go
```

### 2. 启动桌面应用
```bash
cd qim-client
npm run dev
```

### 3. 测试登录
1. 打开桌面应用登录页面
2. 应该能看到"GitHub登录"按钮
3. 点击按钮，跳转到GitHub授权页面
4. 授权后自动返回并完成登录

---

## 五、生产环境配置

### 1. 修改回调地址

**GitHub OAuth应用设置**：
```
Authorization callback URL: https://your-domain.com/oauth/callback
```

**管理后台配置**：
```json
{
  "redirect_url": "https://your-domain.com/oauth/callback"
}
```

### 2. 安全建议

- ✅ 使用HTTPS
- ✅ 限制scope权限范围
- ✅ 定期轮换Client Secret
- ✅ 在GitHub设置中限制应用访问权限

---

## 六、常见问题

### Q: 点击GitHub登录没反应？
A: 检查：
1. 后端服务是否运行
2. auth_providers表中是否有github配置
3. enabled字段是否为1
4. 浏览器控制台是否有错误

### Q: 授权后没有返回应用？
A: 检查：
1. redirect_url是否与GitHub配置一致
2. 是否正确处理OAuth回调

### Q: 获取用户信息失败？
A: 检查：
1. user_info_url是否正确
2. scope是否包含必要权限
3. token是否有效

---

## 七、其他OAuth提供商配置

### Google OAuth
```json
{
  "client_id": "your_google_client_id",
  "client_secret": "your_google_client_secret",
  "auth_url": "https://accounts.google.com/o/oauth2/v2/auth",
  "token_url": "https://oauth2.googleapis.com/token",
  "user_info_url": "https://www.googleapis.com/oauth2/v2/userinfo",
  "redirect_url": "http://localhost:3000/oauth/callback",
  "scope": "openid email profile"
}
```

### 微信OAuth
```json
{
  "client_id": "your_wechat_appid",
  "client_secret": "your_wechat_secret",
  "auth_url": "https://open.weixin.qq.com/connect/qrconnect",
  "token_url": "https://api.weixin.qq.com/sns/oauth2/access_token",
  "user_info_url": "https://api.weixin.qq.com/sns/userinfo",
  "redirect_url": "http://localhost:3000/oauth/callback",
  "scope": "snsapi_login"
}
```

### 钉钉OAuth
```json
{
  "client_id": "your_dingtalk_appkey",
  "client_secret": "your_dingtalk_appsecret",
  "auth_url": "https://login.dingtalk.com/oauth2/auth",
  "token_url": "https://api.dingtalk.com/v1.0/oauth2/userAccessToken",
  "user_info_url": "https://api.dingtalk.com/v1.0/contact/users/me",
  "redirect_url": "http://localhost:3000/oauth/callback",
  "scope": "openid"
}
```
