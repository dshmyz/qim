# 构建脚本使用说明

## 快速开始

### 方式一：使用 npm scripts（推荐）

```bash
# 构建 Windows 7 版本
npm run electron:build:win7

# 构建 Windows 10 版本
npm run electron:build:win10

# 构建 Linux 版本
npm run electron:build:linux

# 构建 macOS 版本
npm run electron:build:mac

# 构建所有平台
npm run electron:build:all
```

### 方式二：直接运行脚本

```bash
# 构建 Windows 7
./scripts/build-win7.sh

# 构建 Windows 10
./scripts/build-win10.sh

# 构建 Linux
./scripts/build-linux.sh

# 构建 macOS
./scripts/build-mac.sh

# 一键构建所有平台（默认）
./scripts/build-all.sh

# 自定义构建选项
./scripts/build-all.sh --win7 --linux        # 仅构建 Win7 和 Linux
./scripts/build-all.sh --win                 # 构建所有 Windows 版本
./scripts/build-all.sh --all --clean         # 清理后构建所有平台
./scripts/build-all.sh --all --skip-frontend # 跳过前端构建
```

## 脚本功能

所有构建脚本都包含以下功能：

1. ✅ **自动检查 Node.js 版本**（需要 18+）
2. ✅ **自动检查依赖安装**
3. ✅ **自动构建前端资源**
4. ✅ **平台依赖检查**（如 Linux 上检查 Wine）
5. ✅ **国内镜像加速**（自动配置 Electron 下载镜像）
6. ✅ **显示构建结果**

## 平台要求

### Windows 7 构建
- **Node.js**: 18+
- **Wine**: 仅在 Linux 上需要（`sudo apt-get install wine64 wine32`）
- **NSIS**: 仅在 Linux 上需要（`sudo apt-get install nsis`）
- **macOS**: 可以直接交叉编译，无需额外依赖

### Windows 10 构建
- 同 Windows 7

### Linux 构建
- **Node.js**: 18+
- **dpkg**: 用于构建 .deb（Debian/Ubuntu 系统自带）
- **rpm**: 可选，用于构建 .rpm（`sudo apt-get install rpm`）
- **FUSE**: 用于 AppImage（`sudo apt-get install libfuse2`）

### macOS 构建
- **Node.js**: 18+
- **系统**: 必须使用 macOS
- **代码签名**: 可选（需要 Apple 开发者账号）

## 常见问题

### Q: 如何在 Linux 上编译 Windows 版本？

A: 需要安装 Wine 和 NSIS：
```bash
sudo apt-get update
sudo apt-get install -y wine64 wine32 nsis
./scripts/build-win7.sh
```

### Q: 构建失败，提示找不到 Electron？

A: 设置国内镜像加速：
```bash
export ELECTRON_MIRROR="https://npmmirror.com/mirrors/electron/"
export ELECTRON_BUILDER_BINARIES_MIRROR="https://npmmirror.com/mirrors/electron-builder-binaries/"
```

### Q: 如何跳过前端构建？

A: 使用 `--skip-frontend` 参数：
```bash
./scripts/build-all.sh --all --skip-frontend
```

### Q: 如何清理构建缓存？

A: 使用 `--clean` 参数：
```bash
./scripts/build-all.sh --all --clean
```

## 输出目录

所有构建产物都输出到 `electron-dist/` 目录：

```
electron-dist/
├── 青雀 QIM-Setup-1.0.0-Win7.exe       # Windows 7 安装程序
├── 青雀 QIM-1.0.0-Win7.exe             # Windows 7 便携版
├── 青雀 QIM-Setup-1.0.0-Win10.exe      # Windows 10 安装程序
├── 青雀 QIM-1.0.0-Win10.exe            # Windows 10 便携版
├── 青雀 QIM-1.0.0.AppImage             # Linux AppImage
├── 青雀 QIM-1.0.0.deb                  # Linux deb 包
├── 青雀 QIM-1.0.0.rpm                  # Linux rpm 包
└── 青雀 QIM-1.0.0.dmg                  # macOS dmg
```
