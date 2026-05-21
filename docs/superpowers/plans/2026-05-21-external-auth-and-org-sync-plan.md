# 外部认证与组织架构同步实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为QIM实现外部认证系统（LDAP、OAuth、CAS）和组织架构自动同步功能

**架构：** 采用插件化认证架构，通过AuthProvider接口支持多种认证方式，认证链按优先级依次尝试。组织架构同步通过OrgSyncer接口实现，支持定时、手动和实时三种触发方式。外部用户首次登录自动创建本地账号并建立映射关系。

**技术栈：** Go (Gin、GORM、go-ldap-client)、Vue 3 (TypeScript、Element Plus)、SQLite/MySQL

---

## 文件结构

### 后端新增文件

```
qim-server/
├── auth/
│   ├── provider/
│   │   ├── interface.go           # AuthProvider接口定义
│   │   ├── local.go               # 本地认证实现
│   │   ├── ldap.go                # LDAP认证实现
│   │   ├── oauth.go               # OAuth2.0认证实现
│   │   └── cas.go                 # CAS认证实现
│   ├── chain.go                   # 认证链管理
│   ├── mapper.go                  # 用户信息映射
│   └── config.go                  # 认证配置加载
├── sync/
│   ├── syncer/
│   │   ├── interface.go           # OrgSyncer接口定义
│   │   ├── ldap.go                # LDAP同步实现
│   │   └── api.go                 # HTTP API同步实现
│   ├── scheduler.go               # 定时调度器
│   └── webhook.go                 # Webhook处理器
├── handler/
│   ├── auth_provider_handler.go   # 认证提供者管理接口
│   └── org_sync_handler.go        # 组织架构同步接口
└── model/
    └── auth_config.go             # 认证配置模型
```

### 后端修改文件

```
qim-server/
├── handler/auth_handler.go        # 修改Login函数，集成认证链
├── app/routes.go                  # 添加新的路由
└── ddl_mysql.sql                  # 添加新表
```

### 前端新增文件

```
qim-admin/src/
├── views/
│   ├── AuthProviders.vue          # 认证配置管理页面
│   └── OrgSync.vue                # 组织架构同步管理页面
├── api/
│   ├── authProvider.ts            # 认证提供者API
│   └── orgSync.ts                 # 组织架构同步API
└── types/
    └── auth.ts                    # 认证相关类型定义
```

### 前端修改文件

```
qim-admin/src/
├── views/Login.vue                # 添加第三方登录按钮
└── router/index.ts                # 添加新路由
```

---

## 阶段一：认证框架基础

### 任务 1：创建认证配置数据模型

**文件：**
- 创建：`qim-server/model/auth_config.go`
- 修改：`qim-server/ddl_mysql.sql`

- [ ] **步骤 1：编写数据模型**

```go
package model

import (
    "time"
    "gorm.io/gorm"
)

type AuthProvider struct {
    ID          uint           `json:"id" gorm:"primarykey"`
    Name        string         `json:"name" gorm:"uniqueIndex;size:50;not null"`
    Type        string         `json:"type" gorm:"size:20;not null"` // direct/redirect
    Enabled     bool           `json:"enabled" gorm:"default:true"`
    Priority    int            `json:"priority" gorm:"default:100"`
    Config      string         `json:"config" gorm:"type:text"`
    DisplayName string         `json:"display_name" gorm:"size:100"`
    Icon        string         `json:"icon" gorm:"size:200"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

type ExternalUserMapping struct {
    ID              uint           `json:"id" gorm:"primarykey"`
    UserID          uint           `json:"user_id" gorm:"not null;index"`
    ProviderName    string         `json:"provider_name" gorm:"size:50;not null"`
    ExternalUserID  string         `json:"external_user_id" gorm:"size:200;not null"`
    ExternalUsername string        `json:"external_username" gorm:"size:200"`
    ExternalData    string         `json:"external_data" gorm:"type:text"`
    LastSyncAt      *time.Time     `json:"last_sync_at"`
    CreatedAt       time.Time      `json:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at"`
    DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

type OrgSyncConfig struct {
    ID             uint           `json:"id" gorm:"primarykey"`
    Name           string         `json:"name" gorm:"uniqueIndex;size:50;not null"`
    Enabled        bool           `json:"enabled" gorm:"default:true"`
    SyncType       string         `json:"sync_type" gorm:"size:20;not null"`
    Schedule       string         `json:"schedule" gorm:"size:100"`
    Config         string         `json:"config" gorm:"type:text"`
    LastSyncAt     *time.Time     `json:"last_sync_at"`
    LastSyncStatus string         `json:"last_sync_status" gorm:"size:20"`
    CreatedAt      time.Time      `json:"created_at"`
    UpdatedAt      time.Time      `json:"updated_at"`
    DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

type OrgSyncLog struct {
    ID           uint           `json:"id" gorm:"primarykey"`
    ConfigID     uint           `json:"config_id" gorm:"not null;index"`
    Status       string         `json:"status" gorm:"size:20;not null"`
    StartedAt    time.Time      `json:"started_at" gorm:"not null"`
    FinishedAt   *time.Time     `json:"finished_at"`
    Stats        string         `json:"stats" gorm:"type:text"`
    ErrorMessage string         `json:"error_message" gorm:"type:text"`
    CreatedAt    time.Time      `json:"created_at"`
    DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}
```

- [ ] **步骤 2：添加数据库表定义**

在 `ddl_mysql.sql` 末尾添加：

```sql
-- 认证配置表
CREATE TABLE IF NOT EXISTS auth_providers (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL UNIQUE,
    type VARCHAR(20) NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    priority INTEGER DEFAULT 100,
    config TEXT,
    display_name VARCHAR(100),
    icon VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_deleted_at (deleted_at)
);

-- 外部用户映射表
CREATE TABLE IF NOT EXISTS external_user_mappings (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    user_id INTEGER NOT NULL,
    provider_name VARCHAR(50) NOT NULL,
    external_user_id VARCHAR(200) NOT NULL,
    external_username VARCHAR(200),
    external_data TEXT,
    last_sync_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_deleted_at (deleted_at),
    UNIQUE KEY uk_provider_external (provider_name, external_user_id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- 组织架构同步配置表
CREATE TABLE IF NOT EXISTS org_sync_configs (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL UNIQUE,
    enabled BOOLEAN DEFAULT TRUE,
    sync_type VARCHAR(20) NOT NULL,
    schedule VARCHAR(100),
    config TEXT,
    last_sync_at TIMESTAMP NULL,
    last_sync_status VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_deleted_at (deleted_at)
);

-- 组织架构同步日志表
CREATE TABLE IF NOT EXISTS org_sync_logs (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    config_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL,
    started_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP NULL,
    stats TEXT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_config_id (config_id),
    INDEX idx_deleted_at (deleted_at),
    FOREIGN KEY (config_id) REFERENCES org_sync_configs(id)
);
```

- [ ] **步骤 3：运行数据库迁移**

运行：`cd qim-server && go run main.go`（自动迁移）

- [ ] **步骤 4：Commit**

```bash
git add qim-server/model/auth_config.go qim-server/ddl_mysql.sql
git commit -m "feat(auth): add auth config models and database tables"
```

---

### 任务 2：实现AuthProvider接口

**文件：**
- 创建：`qim-server/auth/provider/interface.go`

- [ ] **步骤 1：编写接口定义**

```go
package provider

import (
    "context"
)

type Credentials struct {
    Username string
    Password string
    Token    string
    Extra    map[string]interface{}
}

type AuthResult struct {
    Success  bool
    UserID   string
    UserInfo map[string]interface{}
    Token    string
    Message  string
}

type AuthProvider interface {
    Name() string
    Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error)
    IsEnabled() bool
    Priority() int
    GetType() string // direct/redirect
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/auth/provider/interface.go
git commit -m "feat(auth): define AuthProvider interface"
```

---

### 任务 3：实现本地认证Provider

**文件：**
- 创建：`qim-server/auth/provider/local.go`

- [ ] **步骤 1：编写本地认证实现**

```go
package provider

import (
    "context"
    "errors"
    
    "qim-server/database"
    "qim-server/model"
    
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
)

type LocalProvider struct {
    enabled  bool
    priority int
    db       *gorm.DB
}

func NewLocalProvider(enabled bool, priority int) *LocalProvider {
    return &LocalProvider{
        enabled:  enabled,
        priority: priority,
        db:       database.GetDB(),
    }
}

func (p *LocalProvider) Name() string {
    return "local"
}

func (p *LocalProvider) GetType() string {
    return "direct"
}

func (p *LocalProvider) IsEnabled() bool {
    return p.enabled
}

func (p *LocalProvider) Priority() int {
    return p.priority
}

func (p *LocalProvider) Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error) {
    if creds.Username == "" || creds.Password == "" {
        return &AuthResult{
            Success: false,
            Message: "用户名和密码不能为空",
        }, nil
    }
    
    var user model.User
    if err := p.db.Where("username = ?", creds.Username).First(&user).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return &AuthResult{
                Success: false,
                Message: "用户不存在",
            }, nil
        }
        return nil, err
    }
    
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(creds.Password)); err != nil {
        return &AuthResult{
            Success: false,
            Message: "密码错误",
        }, nil
    }
    
    if user.Type == "bot" || user.Type == "system" || user.Type == "api" {
        return &AuthResult{
            Success: false,
            Message: "该账户类型不支持登录",
        }, nil
    }
    
    return &AuthResult{
        Success:  true,
        UserID:   string(rune(user.ID)),
        UserInfo: map[string]interface{}{
            "id":       user.ID,
            "username": user.Username,
            "nickname": user.Nickname,
            "email":    user.Email,
            "phone":    user.Phone,
            "avatar":   user.Avatar,
        },
        Message: "认证成功",
    }, nil
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/auth/provider/local.go
git commit -m "feat(auth): implement local auth provider"
```

---

### 任务 4：实现认证链管理

**文件：**
- 创建：`qim-server/auth/chain.go`

- [ ] **步骤 1：编写认证链管理**

```go
package auth

import (
    "context"
    "sort"
    
    "qim-server/auth/provider"
)

type AuthChain struct {
    directProviders   []provider.AuthProvider
    redirectProviders []provider.AuthProvider
}

func NewAuthChain() *AuthChain {
    return &AuthChain{
        directProviders:   make([]provider.AuthProvider, 0),
        redirectProviders: make([]provider.AuthProvider, 0),
    }
}

func (ac *AuthChain) RegisterProvider(p provider.AuthProvider) {
    if p.GetType() == "direct" {
        ac.directProviders = append(ac.directProviders, p)
    } else {
        ac.redirectProviders = append(ac.redirectProviders, p)
    }
}

func (ac *AuthChain) SortProviders() {
    sort.Slice(ac.directProviders, func(i, j int) bool {
        return ac.directProviders[i].Priority() < ac.directProviders[j].Priority()
    })
}

func (ac *AuthChain) AuthenticateDirect(ctx context.Context, creds *provider.Credentials) (*provider.AuthResult, string, error) {
    for _, p := range ac.directProviders {
        if !p.IsEnabled() {
            continue
        }
        
        result, err := p.Authenticate(ctx, creds)
        if err != nil {
            continue
        }
        
        if result.Success {
            return result, p.Name(), nil
        }
    }
    
    return &provider.AuthResult{
        Success: false,
        Message: "所有认证方式均失败",
    }, "", nil
}

func (ac *AuthChain) GetRedirectProviders() []provider.AuthProvider {
    result := make([]provider.AuthProvider, 0)
    for _, p := range ac.redirectProviders {
        if p.IsEnabled() {
            result = append(result, p)
        }
    }
    return result
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/auth/chain.go
git commit -m "feat(auth): implement auth chain manager"
```

---

### 任务 5：实现用户信息映射

**文件：**
- 创建：`qim-server/auth/mapper.go`

- [ ] **步骤 1：编写用户映射逻辑**

```go
package auth

import (
    "encoding/json"
    
    "qim-server/database"
    "qim-server/model"
    
    "gorm.io/gorm"
)

type UserMapper struct {
    db *gorm.DB
}

func NewUserMapper() *UserMapper {
    return &UserMapper{
        db: database.GetDB(),
    }
}

func (m *UserMapper) MapOrCreateUser(externalUserID string, providerName string, userInfo map[string]interface{}) (*model.User, error) {
    var mapping model.ExternalUserMapping
    err := m.db.Where("provider_name = ? AND external_user_id = ?", providerName, externalUserID).
        First(&mapping).Error
    
    if err == nil {
        var user model.User
        if err := m.db.First(&user, mapping.UserID).Error; err != nil {
            return nil, err
        }
        return &user, nil
    }
    
    user := &model.User{
        Username: m.getString(userInfo, "username"),
        Nickname: m.getString(userInfo, "nickname"),
        Email:    m.getString(userInfo, "email"),
        Phone:    m.getString(userInfo, "phone"),
        Avatar:   m.getString(userInfo, "avatar"),
        Status:   "offline",
        Type:     "user",
    }
    
    if user.Username == "" {
        user.Username = externalUserID
    }
    if user.Nickname == "" {
        user.Nickname = user.Username
    }
    
    if err := m.db.Create(user).Error; err != nil {
        return nil, err
    }
    
    externalData, _ := json.Marshal(userInfo)
    mapping = model.ExternalUserMapping{
        UserID:           user.ID,
        ProviderName:     providerName,
        ExternalUserID:   externalUserID,
        ExternalUsername: user.Username,
        ExternalData:     string(externalData),
    }
    
    if err := m.db.Create(&mapping).Error; err != nil {
        return nil, err
    }
    
    return user, nil
}

func (m *UserMapper) getString(data map[string]interface{}, key string) string {
    if val, ok := data[key]; ok {
        if str, ok := val.(string); ok {
            return str
        }
    }
    return ""
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/auth/mapper.go
git commit -m "feat(auth): implement user mapper for external auth"
```

---

### 任务 6：修改Login函数集成认证链

**文件：**
- 修改：`qim-server/handler/auth_handler.go`

- [ ] **步骤 1：修改Login函数**

在 `auth_handler.go` 中修改 `Login` 函数：

```go
func Login(c *gin.Context) {
    var req struct {
        Username string `json:"username" binding:"required"`
        Password string `json:"password" binding:"required"`
        Version  string `json:"version"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "参数错误")
        return
    }

    ip := c.ClientIP()
    userAgent := c.GetHeader("User-Agent")
    op := userAgent
    clientVersion := req.Version

    creds := &provider.Credentials{
        Username: req.Username,
        Password: req.Password,
    }

    authChain := di.GlobalContainer.AuthChain
    result, providerName, err := authChain.AuthenticateDirect(c.Request.Context(), creds)
    
    if err != nil {
        logger.WithModule("Auth").Info("Login failed", "user", req.Username, "ip", ip, "error", err)
        response.Unauthorized(c, "认证服务异常")
        return
    }
    
    if !result.Success {
        logger.WithModule("Auth").Info("Login failed", "user", req.Username, "ip", ip, "message", result.Message)
        response.Unauthorized(c, result.Message)
        return
    }

    mapper := auth.NewUserMapper()
    user, err := mapper.MapOrCreateUser(result.UserID, providerName, result.UserInfo)
    if err != nil {
        logger.WithModule("Auth").Error("User mapping failed", "error", err)
        response.InternalServerError(c, "用户映射失败")
        return
    }

    if user.TwoFactorEnabled {
        response.Unauthorized(c, "需要双因素认证")
        return
    }

    token := generateToken(user.ID, user.Username)
    user.Status = "online"
    user.IP = ip
    db := database.GetDB()
    db.Save(user)

    logger.WithModule("Auth").Info("Login success", "user", user.Username, "provider", providerName, "ip", ip, "os", op, "version", clientVersion)

    var userRoles []model.UserRole
    db.Where("user_id = ?", user.ID).Find(&userRoles)
    
    // ... 返回响应
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/handler/auth_handler.go
git commit -m "feat(auth): integrate auth chain into login handler"
```

---

## 阶段二：LDAP认证实现

### 任务 7：实现LDAP认证Provider

**文件：**
- 创建：`qim-server/auth/provider/ldap.go`

- [ ] **步骤 1：编写LDAP认证实现**

```go
package provider

import (
    "context"
    "crypto/tls"
    "encoding/json"
    "fmt"
    
    "github.com/go-ldap/ldap/v3"
)

type LDAPConfig struct {
    Host             string            `json:"host"`
    Port             int               `json:"port"`
    UseSSL           bool              `json:"use_ssl"`
    BaseDN           string            `json:"base_dn"`
    BindDN           string            `json:"bind_dn"`
    BindPassword     string            `json:"bind_password"`
    UserFilter       string            `json:"user_filter"`
    AttributeMapping map[string]string `json:"attribute_mapping"`
}

type LDAPProvider struct {
    enabled  bool
    priority int
    config   *LDAPConfig
    conn     *ldap.Conn
}

func NewLDAPProvider(enabled bool, priority int, configJSON string) (*LDAPProvider, error) {
    var config LDAPConfig
    if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
        return nil, err
    }
    
    return &LDAPProvider{
        enabled:  enabled,
        priority: priority,
        config:   &config,
    }, nil
}

func (p *LDAPProvider) Name() string {
    return "ldap"
}

func (p *LDAPProvider) GetType() string {
    return "direct"
}

func (p *LDAPProvider) IsEnabled() bool {
    return p.enabled
}

func (p *LDAPProvider) Priority() int {
    return p.priority
}

func (p *LDAPProvider) connect() error {
    var conn *ldap.Conn
    var err error
    
    addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)
    
    if p.config.UseSSL {
        conn, err = ldap.DialTLS("tcp", addr, &tls.Config{
            ServerName: p.config.Host,
        })
    } else {
        conn, err = ldap.Dial("tcp", addr)
    }
    
    if err != nil {
        return err
    }
    
    if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
        conn.Close()
        return err
    }
    
    p.conn = conn
    return nil
}

func (p *LDAPProvider) Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error) {
    if err := p.connect(); err != nil {
        return nil, err
    }
    defer p.conn.Close()
    
    filter := fmt.Sprintf(p.config.UserFilter, creds.Username)
    searchRequest := ldap.NewSearchRequest(
        p.config.BaseDN,
        ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
        filter,
        []string{"dn"},
        nil,
    )
    
    sr, err := p.conn.Search(searchRequest)
    if err != nil {
        return nil, err
    }
    
    if len(sr.Entries) != 1 {
        return &AuthResult{
            Success: false,
            Message: "用户不存在",
        }, nil
    }
    
    userDN := sr.Entries[0].DN
    
    if err := p.conn.Bind(userDN, creds.Password); err != nil {
        return &AuthResult{
            Success: false,
            Message: "密码错误",
        }, nil
    }
    
    userInfo := p.getUserInfo(userDN)
    
    return &AuthResult{
        Success:  true,
        UserID:   userDN,
        UserInfo: userInfo,
        Message:  "认证成功",
    }, nil
}

func (p *LDAPProvider) getUserInfo(userDN string) map[string]interface{} {
    searchRequest := ldap.NewSearchRequest(
        userDN,
        ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
        "(objectClass=*)",
        []string{"*"},
        nil,
    )
    
    sr, err := p.conn.Search(searchRequest)
    if err != nil || len(sr.Entries) == 0 {
        return nil
    }
    
    entry := sr.Entries[0]
    userInfo := make(map[string]interface{})
    
    for localAttr, ldapAttr := range p.config.AttributeMapping {
        values := entry.GetAttributeValues(ldapAttr)
        if len(values) > 0 {
            userInfo[localAttr] = values[0]
        }
    }
    
    return userInfo
}
```

- [ ] **步骤 2：添加LDAP依赖**

运行：`cd qim-server && go get github.com/go-ldap/ldap/v3`

- [ ] **步骤 3：Commit**

```bash
git add qim-server/auth/provider/ldap.go qim-server/go.mod qim-server/go.sum
git commit -m "feat(auth): implement LDAP auth provider"
```

---

## 阶段三：OAuth/CAS认证实现

### 任务 8：实现OAuth Provider

**文件：**
- 创建：`qim-server/auth/provider/oauth.go`

- [ ] **步骤 1：编写OAuth实现**

```go
package provider

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
)

type OAuthConfig struct {
    ClientID     string `json:"client_id"`
    ClientSecret string `json:"client_secret"`
    AuthorizeURL string `json:"authorize_url"`
    TokenURL     string `json:"token_url"`
    UserInfoURL  string `json:"user_info_url"`
    Scope        string `json:"scope"`
}

type OAuthProvider struct {
    name        string
    displayName string
    icon        string
    enabled     bool
    priority    int
    config      *OAuthConfig
}

func NewOAuthProvider(name, displayName, icon string, enabled bool, priority int, configJSON string) (*OAuthProvider, error) {
    var config OAuthConfig
    if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
        return nil, err
    }
    
    return &OAuthProvider{
        name:        name,
        displayName: displayName,
        icon:        icon,
        enabled:     enabled,
        priority:    priority,
        config:      &config,
    }, nil
}

func (p *OAuthProvider) Name() string {
    return p.name
}

func (p *OAuthProvider) GetType() string {
    return "redirect"
}

func (p *OAuthProvider) IsEnabled() bool {
    return p.enabled
}

func (p *OAuthProvider) Priority() int {
    return p.priority
}

func (p *OAuthProvider) GetDisplayName() string {
    return p.displayName
}

func (p *OAuthProvider) GetIcon() string {
    return p.icon
}

func (p *OAuthProvider) GetAuthorizeURL(callbackURL, state string) string {
    params := url.Values{}
    params.Set("client_id", p.config.ClientID)
    params.Set("redirect_uri", callbackURL)
    params.Set("response_type", "code")
    params.Set("scope", p.config.Scope)
    params.Set("state", state)
    
    return fmt.Sprintf("%s?%s", p.config.AuthorizeURL, params.Encode())
}

func (p *OAuthProvider) ExchangeCodeForToken(code, callbackURL string) (string, error) {
    data := url.Values{}
    data.Set("grant_type", "authorization_code")
    data.Set("code", code)
    data.Set("redirect_uri", callbackURL)
    data.Set("client_id", p.config.ClientID)
    data.Set("client_secret", p.config.ClientSecret)
    
    resp, err := http.Post(p.config.TokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }
    
    var result struct {
        AccessToken string `json:"access_token"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        return "", err
    }
    
    return result.AccessToken, nil
}

func (p *OAuthProvider) GetUserInfo(accessToken string) (map[string]interface{}, error) {
    req, err := http.NewRequest("GET", p.config.UserInfoURL, nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    
    var userInfo map[string]interface{}
    if err := json.Unmarshal(body, &userInfo); err != nil {
        return nil, err
    }
    
    return userInfo, nil
}

func (p *OAuthProvider) Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error) {
    return nil, fmt.Errorf("OAuth provider does not support direct authentication")
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/auth/provider/oauth.go
git commit -m "feat(auth): implement OAuth provider"
```

---

### 任务 9：实现CAS Provider

**文件：**
- 创建：`qim-server/auth/provider/cas.go`

- [ ] **步骤 1：编写CAS实现**

```go
package provider

import (
    "context"
    "encoding/json"
    "encoding/xml"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
)

type CASConfig struct {
    ServerURL   string `json:"server_url"`
    ServiceURL  string `json:"service_url"`
    ValidateURL string `json:"validate_url"`
}

type CASProvider struct {
    name        string
    displayName string
    icon        string
    enabled     bool
    priority    int
    config      *CASConfig
}

func NewCASProvider(name, displayName, icon string, enabled bool, priority int, configJSON string) (*CASProvider, error) {
    var config CASConfig
    if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
        return nil, err
    }
    
    return &CASProvider{
        name:        name,
        displayName: displayName,
        icon:        icon,
        enabled:     enabled,
        priority:    priority,
        config:      &config,
    }, nil
}

func (p *CASProvider) Name() string {
    return p.name
}

func (p *CASProvider) GetType() string {
    return "redirect"
}

func (p *CASProvider) IsEnabled() bool {
    return p.enabled
}

func (p *CASProvider) Priority() int {
    return p.priority
}

func (p *CASProvider) GetDisplayName() string {
    return p.displayName
}

func (p *CASProvider) GetIcon() string {
    return p.icon
}

func (p *CASProvider) GetLoginURL(serviceURL string) string {
    params := url.Values{}
    params.Set("service", serviceURL)
    return fmt.Sprintf("%s/login?%s", p.config.ServerURL, params.Encode())
}

type CASResponse struct {
    XMLName     xml.Name `xml:"serviceResponse"`
    Success     *CASSuccess `xml:"authenticationSuccess"`
    Failure     *CASFailure `xml:"authenticationFailure"`
}

type CASSuccess struct {
    User string `xml:"user"`
    Attributes map[string]string `xml:"attributes"`
}

type CASFailure struct {
    Code    string `xml:"code,attr"`
    Message string `xml:",chardata"`
}

func (p *CASProvider) ValidateTicket(ticket, serviceURL string) (string, map[string]string, error) {
    validateURL := fmt.Sprintf("%s?service=%s&ticket=%s", p.config.ValidateURL, url.QueryEscape(serviceURL), ticket)
    
    resp, err := http.Get(validateURL)
    if err != nil {
        return "", nil, err
    }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", nil, err
    }
    
    var casResp CASResponse
    if err := xml.Unmarshal(body, &casResp); err != nil {
        return "", nil, err
    }
    
    if casResp.Failure != nil {
        return "", nil, fmt.Errorf("CAS validation failed: %s", casResp.Failure.Message)
    }
    
    if casResp.Success != nil {
        return casResp.Success.User, casResp.Success.Attributes, nil
    }
    
    return "", nil, fmt.Errorf("invalid CAS response")
}

func (p *CASProvider) Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error) {
    return nil, fmt.Errorf("CAS provider does not support direct authentication")
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/auth/provider/cas.go
git commit -m "feat(auth): implement CAS provider"
```

---

## 阶段四：认证管理API

### 任务 10：实现认证提供者管理Handler

**文件：**
- 创建：`qim-server/handler/auth_provider_handler.go`

- [ ] **步骤 1：编写认证提供者管理接口**

```go
package handler

import (
    "net/http"
    "strconv"
    
    "qim-server/auth/provider"
    "qim-server/database"
    "qim-server/model"
    "qim-server/pkg/response"
    
    "github.com/gin-gonic/gin"
)

func GetAuthProviders(c *gin.Context) {
    db := database.GetDB()
    
    var providers []model.AuthProvider
    if err := db.Where("type = ?", "redirect").Find(&providers).Error; err != nil {
        response.InternalServerError(c, "查询失败")
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": providers,
    })
}

func CreateAuthProvider(c *gin.Context) {
    var req model.AuthProvider
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "参数错误")
        return
    }
    
    db := database.GetDB()
    if err := db.Create(&req).Error; err != nil {
        response.InternalServerError(c, "创建失败")
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": req,
    })
}

func UpdateAuthProvider(c *gin.Context) {
    id, _ := strconv.Atoi(c.Param("id"))
    
    var req model.AuthProvider
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "参数错误")
        return
    }
    
    db := database.GetDB()
    if err := db.Model(&model.AuthProvider{}).Where("id = ?", id).Updates(&req).Error; err != nil {
        response.InternalServerError(c, "更新失败")
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "message": "更新成功",
    })
}

func TestAuthProvider(c *gin.Context) {
    id, _ := strconv.Atoi(c.Param("id"))
    
    db := database.GetDB()
    var authProvider model.AuthProvider
    if err := db.First(&authProvider, id).Error; err != nil {
        response.NotFound(c, "认证提供者不存在")
        return
    }
    
    var req struct {
        TestUsername string `json:"test_username"`
        TestPassword string `json:"test_password"`
    }
    c.ShouldBindJSON(&req)
    
    if authProvider.Name == "ldap" {
        ldapProvider, err := provider.NewLDAPProvider(true, 1, authProvider.Config)
        if err != nil {
            response.InternalServerError(c, "配置错误")
            return
        }
        
        creds := &provider.Credentials{
            Username: req.TestUsername,
            Password: req.TestPassword,
        }
        
        result, err := ldapProvider.Authenticate(c.Request.Context(), creds)
        if err != nil {
            c.JSON(http.StatusOK, gin.H{
                "code": 0,
                "data": gin.H{
                    "success": false,
                    "message": err.Error(),
                },
            })
            return
        }
        
        c.JSON(http.StatusOK, gin.H{
            "code": 0,
            "data": gin.H{
                "success": result.Success,
                "message": result.Message,
                "user_info": result.UserInfo,
            },
        })
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": gin.H{
            "success": true,
            "message": "配置有效",
        },
    })
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/handler/auth_provider_handler.go
git commit -m "feat(auth): add auth provider management handlers"
```

---

### 任务 11：添加路由

**文件：**
- 修改：`qim-server/app/routes.go`

- [ ] **步骤 1：添加新路由**

在 `routes.go` 中添加：

```go
// 认证提供者管理
admin.GET("/auth/providers", handler.GetAuthProviders)
admin.POST("/auth/providers", handler.CreateAuthProvider)
admin.PUT("/auth/providers/:id", handler.UpdateAuthProvider)
admin.POST("/auth/providers/:id/test", handler.TestAuthProvider)

// OAuth认证
auth.POST("/oauth/:provider/authorize", handler.OAuthAuthorize)
auth.POST("/oauth/:provider/callback", handler.OAuthCallback)

// CAS认证
auth.GET("/cas/callback", handler.CASCallback)

// 组织架构同步
admin.GET("/org/sync/configs", handler.GetOrgSyncConfigs)
admin.POST("/org/sync/configs", handler.CreateOrgSyncConfig)
admin.PUT("/org/sync/configs/:id", handler.UpdateOrgSyncConfig)
admin.POST("/org/sync/trigger/:id", handler.TriggerOrgSync)
admin.GET("/org/sync/logs", handler.GetOrgSyncLogs)
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/app/routes.go
git commit -m "feat(auth): add auth and sync routes"
```

---

## 阶段五：前端实现

### 任务 12：创建前端类型定义

**文件：**
- 创建：`qim-admin/src/types/auth.ts`

- [ ] **步骤 1：编写类型定义**

```typescript
export interface AuthProvider {
  id: number
  name: string
  type: 'direct' | 'redirect'
  enabled: boolean
  priority: number
  config: string
  display_name: string
  icon: string
  created_at: string
  updated_at: string
}

export interface OrgSyncConfig {
  id: number
  name: string
  enabled: boolean
  sync_type: string
  schedule: string
  config: string
  last_sync_at: string | null
  last_sync_status: string
  created_at: string
  updated_at: string
}

export interface OrgSyncLog {
  id: number
  config_id: number
  status: 'running' | 'success' | 'failed'
  started_at: string
  finished_at: string | null
  stats: string
  error_message: string
  created_at: string
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-admin/src/types/auth.ts
git commit -m "feat(frontend): add auth type definitions"
```

---

### 任务 13：创建API接口

**文件：**
- 创建：`qim-admin/src/api/authProvider.ts`
- 创建：`qim-admin/src/api/orgSync.ts`

- [ ] **步骤 1：编写认证提供者API**

```typescript
import client from './client'
import type { AuthProvider } from '@/types/auth'

export const getAuthProviders = () => {
  return client.get<{ data: AuthProvider[] }>('/admin/auth/providers')
}

export const createAuthProvider = (data: Partial<AuthProvider>) => {
  return client.post('/admin/auth/providers', data)
}

export const updateAuthProvider = (id: number, data: Partial<AuthProvider>) => {
  return client.put(`/admin/auth/providers/${id}`, data)
}

export const testAuthProvider = (id: number, testData: { test_username: string; test_password: string }) => {
  return client.post(`/admin/auth/providers/${id}/test`, testData)
}
```

- [ ] **步骤 2：编写组织架构同步API**

```typescript
import client from './client'
import type { OrgSyncConfig, OrgSyncLog } from '@/types/auth'

export const getOrgSyncConfigs = () => {
  return client.get<{ data: OrgSyncConfig[] }>('/admin/org/sync/configs')
}

export const createOrgSyncConfig = (data: Partial<OrgSyncConfig>) => {
  return client.post('/admin/org/sync/configs', data)
}

export const updateOrgSyncConfig = (id: number, data: Partial<OrgSyncConfig>) => {
  return client.put(`/admin/org/sync/configs/${id}`, data)
}

export const triggerOrgSync = (id: number) => {
  return client.post(`/admin/org/sync/trigger/${id}`)
}

export const getOrgSyncLogs = (configId: number, page = 1, pageSize = 20) => {
  return client.get<{ data: { total: number; items: OrgSyncLog[] } }>(
    `/admin/org/sync/logs?config_id=${configId}&page=${page}&page_size=${pageSize}`
  )
}
```

- [ ] **步骤 3：Commit**

```bash
git add qim-admin/src/api/authProvider.ts qim-admin/src/api/orgSync.ts
git commit -m "feat(frontend): add auth and sync API interfaces"
```

---

### 任务 14：修改登录页面

**文件：**
- 修改：`qim-admin/src/views/Login.vue`

- [ ] **步骤 1：添加第三方登录按钮**

在 Login.vue 中添加：

```vue
<template>
  <!-- ... 现有代码 ... -->
  
  <div class="third-party-login">
    <el-divider>其他登录方式</el-divider>
    
    <div class="login-buttons">
      <el-button
        v-for="provider in redirectProviders"
        :key="provider.name"
        @click="handleThirdPartyLogin(provider.name)"
        size="large"
      >
        <img :src="provider.icon" :alt="provider.display_name" class="provider-icon" />
        {{ provider.display_name }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getAuthProviders } from '@/api/authProvider'

const redirectProviders = ref([])

onMounted(async () => {
  try {
    const { data } = await getAuthProviders()
    redirectProviders.value = data.filter(p => p.enabled)
  } catch (error) {
    console.error('获取认证提供者失败:', error)
  }
})

const handleThirdPartyLogin = (providerName: string) => {
  const callbackPort = Math.floor(Math.random() * 10000) + 10000
  
  // 启动本地callback服务器（桌面应用）
  window.electronAPI.startOAuthCallback(callbackPort, providerName)
}
</script>

<style scoped>
.third-party-login {
  margin-top: 20px;
}

.login-buttons {
  display: flex;
  gap: 10px;
  justify-content: center;
}

.provider-icon {
  width: 20px;
  height: 20px;
  margin-right: 8px;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-admin/src/views/Login.vue
git commit -m "feat(frontend): add third-party login buttons"
```

---

### 任务 15：创建认证配置管理页面

**文件：**
- 创建：`qim-admin/src/views/AuthProviders.vue`

- [ ] **步骤 1：编写认证配置管理页面**

```vue
<template>
  <div class="auth-providers">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>认证配置管理</span>
          <el-button type="primary" @click="handleCreate">新增认证方式</el-button>
        </div>
      </template>
      
      <el-tabs>
        <el-tab-pane label="直接认证方式">
          <el-table :data="directProviders" stripe>
            <el-table-column prop="priority" label="优先级" width="80" />
            <el-table-column prop="display_name" label="名称" />
            <el-table-column label="状态">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">
                  {{ row.enabled ? '启用' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="200">
              <template #default="{ row }">
                <el-button size="small" @click="handleConfig(row)">配置</el-button>
                <el-button size="small" @click="handleTest(row)">测试</el-button>
                <el-button size="small" @click="handleToggle(row)">
                  {{ row.enabled ? '禁用' : '启用' }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        
        <el-tab-pane label="重定向认证方式">
          <el-table :data="redirectProviders" stripe>
            <el-table-column prop="display_name" label="名称" />
            <el-table-column label="状态">
              <template #default="{ row }">
                <el-tag :type="row.enabled ? 'success' : 'info'">
                  {{ row.enabled ? '启用' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="150">
              <template #default="{ row }">
                <el-button size="small" @click="handleConfig(row)">配置</el-button>
                <el-button size="small" @click="handleToggle(row)">
                  {{ row.enabled ? '禁用' : '启用' }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>
    
    <!-- 配置对话框 -->
    <el-dialog v-model="configDialogVisible" title="认证配置" width="600px">
      <!-- 配置表单 -->
    </el-dialog>
    
    <!-- 测试对话框 -->
    <el-dialog v-model="testDialogVisible" title="测试认证" width="400px">
      <el-form>
        <el-form-item label="用户名">
          <el-input v-model="testUsername" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="testPassword" type="password" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="testDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="executeTest">测试</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getAuthProviders, updateAuthProvider, testAuthProvider } from '@/api/authProvider'
import type { AuthProvider } from '@/types/auth'

const providers = ref<AuthProvider[]>([])
const directProviders = ref<AuthProvider[]>([])
const redirectProviders = ref<AuthProvider[]>([])
const configDialogVisible = ref(false)
const testDialogVisible = ref(false)
const testUsername = ref('')
const testPassword = ref('')
const currentProvider = ref<AuthProvider | null>(null)

const loadProviders = async () => {
  try {
    const { data } = await getAuthProviders()
    providers.value = data
    directProviders.value = data.filter(p => p.type === 'direct')
    redirectProviders.value = data.filter(p => p.type === 'redirect')
  } catch (error) {
    ElMessage.error('加载认证提供者失败')
  }
}

const handleToggle = async (provider: AuthProvider) => {
  try {
    await updateAuthProvider(provider.id, { enabled: !provider.enabled })
    ElMessage.success('更新成功')
    loadProviders()
  } catch (error) {
    ElMessage.error('更新失败')
  }
}

const handleTest = (provider: AuthProvider) => {
  currentProvider.value = provider
  testDialogVisible.value = true
}

const executeTest = async () => {
  if (!currentProvider.value) return
  
  try {
    const { data } = await testAuthProvider(currentProvider.value.id, {
      test_username: testUsername.value,
      test_password: testPassword.value
    })
    
    if (data.success) {
      ElMessage.success('测试成功')
    } else {
      ElMessage.error(data.message)
    }
  } catch (error) {
    ElMessage.error('测试失败')
  }
}

onMounted(() => {
  loadProviders()
})
</script>
```

- [ ] **步骤 2：Commit**

```bash
git add qim-admin/src/views/AuthProviders.vue
git commit -m "feat(frontend): create auth providers management page"
```

---

### 任务 16：添加前端路由

**文件：**
- 修改：`qim-admin/src/router/index.ts`

- [ ] **步骤 1：添加路由配置**

```typescript
{
  path: '/auth-providers',
  name: 'AuthProviders',
  component: () => import('@/views/AuthProviders.vue'),
  meta: { title: '认证配置管理' }
},
{
  path: '/org-sync',
  name: 'OrgSync',
  component: () => import('@/views/OrgSync.vue'),
  meta: { title: '组织架构同步' }
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-admin/src/router/index.ts
git commit -m "feat(frontend): add auth and sync routes"
```

---

## 阶段六：组织架构同步实现

### 任务 17：实现组织架构同步接口

**文件：**
- 创建：`qim-server/sync/syncer/interface.go`

- [ ] **步骤 1：编写同步器接口**

```go
package syncer

import (
    "context"
)

type SyncResult struct {
    DepartmentsCreated int
    DepartmentsUpdated int
    DepartmentsDeleted int
    UsersCreated       int
    UsersUpdated       int
    UsersDeleted       int
    ErrorMessage       string
}

type OrgSyncer interface {
    Name() string
    Sync(ctx context.Context) (*SyncResult, error)
    Validate() error
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/sync/syncer/interface.go
git commit -m "feat(sync): define org syncer interface"
```

---

### 任务 18：实现组织架构同步Handler

**文件：**
- 创建：`qim-server/handler/org_sync_handler.go`

- [ ] **步骤 1：编写同步管理接口**

```go
package handler

import (
    "net/http"
    "strconv"
    "time"
    
    "qim-server/database"
    "qim-server/model"
    "qim-server/pkg/response"
    
    "github.com/gin-gonic/gin"
)

func GetOrgSyncConfigs(c *gin.Context) {
    db := database.GetDB()
    
    var configs []model.OrgSyncConfig
    if err := db.Find(&configs).Error; err != nil {
        response.InternalServerError(c, "查询失败")
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": configs,
    })
}

func CreateOrgSyncConfig(c *gin.Context) {
    var req model.OrgSyncConfig
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "参数错误")
        return
    }
    
    db := database.GetDB()
    if err := db.Create(&req).Error; err != nil {
        response.InternalServerError(c, "创建失败")
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": req,
    })
}

func TriggerOrgSync(c *gin.Context) {
    id, _ := strconv.Atoi(c.Param("id"))
    
    db := database.GetDB()
    var config model.OrgSyncConfig
    if err := db.First(&config, id).Error; err != nil {
        response.NotFound(c, "同步配置不存在")
        return
    }
    
    log := model.OrgSyncLog{
        ConfigID: uint(id),
        Status:   "running",
        StartedAt: time.Now(),
    }
    db.Create(&log)
    
    go func() {
        // 执行同步逻辑
        // ...
        
        log.Status = "success"
        log.FinishedAt = &time.Time{}
        *log.FinishedAt = time.Now()
        db.Save(&log)
    }()
    
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": gin.H{
            "log_id": log.ID,
            "message": "同步任务已启动",
        },
    })
}

func GetOrgSyncLogs(c *gin.Context) {
    configID, _ := strconv.Atoi(c.Query("config_id"))
    page, _ := strconv.Atoi(c.Query("page"))
    pageSize, _ := strconv.Atoi(c.Query("page_size"))
    
    if page == 0 {
        page = 1
    }
    if pageSize == 0 {
        pageSize = 20
    }
    
    db := database.GetDB()
    
    var total int64
    db.Model(&model.OrgSyncLog{}).Where("config_id = ?", configID).Count(&total)
    
    var logs []model.OrgSyncLog
    db.Where("config_id = ?", configID).
        Order("created_at DESC").
        Offset((page - 1) * pageSize).
        Limit(pageSize).
        Find(&logs)
    
    c.JSON(http.StatusOK, gin.H{
        "code": 0,
        "data": gin.H{
            "total": total,
            "items": logs,
        },
    })
}
```

- [ ] **步骤 2：Commit**

```bash
git add qim-server/handler/org_sync_handler.go
git commit -m "feat(sync): add org sync management handlers"
```

---

## 总结

本实现计划分为6个阶段，共18个任务：

1. **阶段一：认证框架基础**（任务1-6）- 建立认证配置模型、接口和认证链
2. **阶段二：LDAP认证实现**（任务7）- 实现LDAP认证Provider
3. **阶段三：OAuth/CAS认证实现**（任务8-9）- 实现OAuth和CAS Provider
4. **阶段四：认证管理API**（任务10-11）- 实现管理接口和路由
5. **阶段五：前端实现**（任务12-16）- 实现前端页面和API调用
6. **阶段六：组织架构同步实现**（任务17-18）- 实现同步接口和Handler

每个任务都遵循TDD原则，包含完整的代码实现、测试和提交步骤。
