# 检查更新功能修复说明

## 修复时间
2026-06-05

## 修复内容

### 1. 后端错误处理和日志改进 ✅

**文件**: `qim-server/handler/update_handler.go`

**改进点**:
- ✅ 区分"无版本记录"和"数据库错误"两种情况
- ✅ 添加详细的日志记录（请求参数、查询结果）
- ✅ 返回更友好的错误信息

**关键代码**:
```go
if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        logger.WithModule("Update").Warn("无可用版本记录")
        c.JSON(http.StatusNotFound, gin.H{
            "code": 404,
            "message": "暂无可用更新",
        })
    } else {
        logger.WithModule("Update").Error("查询版本失败")
        c.JSON(http.StatusInternalServerError, gin.H{
            "code": 500,
            "message": "查询更新失败",
        })
    }
    return
}
```

### 2. Electron 更新服务器地址配置修复 ✅

**文件**: `qim-client/electron/main.js`

**改进点**:
- ✅ 新增 `getUpdateServerUrl()` 函数
- ✅ 优先级：环境变量 > 配置文件 > 根据 isPackaged 判断
- ✅ 生产环境自动使用 `https://api.qim.work`
- ✅ 开发环境自动使用 `http://localhost:8080`

**关键代码**:
```javascript
function getUpdateServerUrl() {
  if (process.env.QIM_UPDATE_URL) {
    return process.env.QIM_UPDATE_URL
  }
  
  const savedUrl = loadServerConfig()
  if (savedUrl) {
    return savedUrl
  }
  
  return app.isPackaged 
    ? 'https://api.qim.work' 
    : 'http://localhost:8080'
}
```

### 3. 默认版本记录 SQL 脚本 ✅

**文件**: 
- `qim-server/migrations/001_insert_default_versions.sql` (SQLite)
- `qim-server/migrations/001_insert_default_versions_mysql.sql` (MySQL)

**内容**:
- 插入三个平台的初始版本记录（windows/macos/linux）
- 版本号：1.0.0
- 下载 URL：`https://api.qim.work/downloads/QIM-1.0.0.{ext}`

### 4. 超时时间优化 ✅

**文件**: `qim-client/electron/main.js`

**改进**:
- 从 5 秒增加到 10 秒
- 更适合网络较慢的环境

### 5. 更新健康检查接口 ✅

**文件**: 
- `qim-server/handler/update_handler.go` (新增 `CheckUpdateHealth` 函数)
- `qim-server/app/routes.go` (注册路由)

**接口**: `GET /api/v1/updates/health`

**返回示例**:
```json
{
  "status": "ok",
  "platform_stats": {
    "windows": 1,
    "macos": 1,
    "linux": 1
  },
  "supported_platforms": ["windows", "macos", "linux"],
  "timestamp": 1780630988
}
```

## 使用步骤

### 1. 执行数据库迁移

**SQLite**:
```bash
cd qim-server
sqlite3 qim.db < migrations/001_insert_default_versions.sql
```

**MySQL**:
```bash
cd qim-server
mysql -u root -p qim_server < migrations/001_insert_default_versions_mysql.sql
```

### 2. 重启后端服务

```bash
cd qim-server
go run main.go
```

### 3. 测试更新功能

#### 方法一：使用健康检查接口
```bash
curl http://localhost:8080/api/v1/updates/health
```

#### 方法二：测试 latest.yml 接口
```bash
# macOS
curl http://localhost:8080/api/v1/updates/mac/latest-mac.yml

# Windows
curl http://localhost:8080/api/v1/updates/win/latest.yml

# Linux
curl http://localhost:8080/api/v1/updates/linux/latest.yml
```

#### 方法三：在 Electron 应用中测试
1. 启动 Electron 应用
2. 打开 DevTools
3. 执行：
```javascript
window.electron.ipcRenderer.send('check-for-updates')
```
4. 监听事件：
```javascript
window.electron.ipcRenderer.on('update-checking', () => console.log('checking'))
window.electron.ipcRenderer.on('update-available', (e, info) => console.log('available', info))
window.electron.ipcRenderer.on('update-not-available', () => console.log('not available'))
window.electron.ipcRenderer.on('update-error', (e, err) => console.log('error', err))
```

## 验证清单

- [ ] 数据库中有版本记录
- [ ] 健康检查接口返回正常
- [ ] latest.yml 接口返回正确格式
- [ ] Electron 应用能正常检查更新
- [ ] 日志中有详细的请求记录

## 后续建议

### 1. 生产环境配置
确保生产环境的更新服务器地址正确：
```bash
# 方式一：环境变量
export QIM_UPDATE_URL=https://api.qim.work

# 方式二：配置文件（用户设置）
# 在应用中通过 IPC 设置
window.electron.ipcRenderer.send('set-server-url', 'https://api.qim.work')
```

### 2. 版本发布流程
1. 构建新版本安装包
2. 上传到服务器（`https://api.qim.work/downloads/`）
3. 在数据库中插入新版本记录：
```sql
INSERT INTO client_versions (version, platform, download_url, changelog, enabled, created_at, updated_at)
VALUES ('1.1.0', 'windows', 'https://api.qim.work/downloads/QIM-1.1.0.exe', '修复若干问题', true, NOW(), NOW());
```

### 3. 监控和日志
- 定期检查更新成功率
- 监控更新接口响应时间
- 记录用户更新失败的错误信息

## 故障排查

### 问题 1: 仍然返回 404
**原因**: 数据库中没有版本记录
**解决**: 执行 SQL 脚本插入默认版本

### 问题 2: 更新服务器地址不正确
**原因**: 环境变量或配置文件未设置
**解决**: 
```javascript
// 在 Electron DevTools 中
window.electron.ipcRenderer.send('set-server-url', 'https://api.qim.work')
```

### 问题 3: 检查更新超时
**原因**: 网络问题或服务器响应慢
**解决**: 
- 检查网络连接
- 检查后端服务是否正常运行
- 查看后端日志

## 相关文件

- `qim-server/handler/update_handler.go` - 更新处理器
- `qim-server/app/routes.go` - 路由配置
- `qim-client/electron/main.js` - Electron 主进程
- `qim-client/src/views/Main.vue` - 前端 UI
- `qim-client/src/composables/useUI.ts` - 更新事件处理
