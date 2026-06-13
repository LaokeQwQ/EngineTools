# Engine Tools

**中文** | **[English](README.md)** | **[日本語](README_ja.md)** | **[한국어](README_ko.md)**

修复 Engine DJ 中日韩等特殊字符显示问题，通过应用级别的 UTF-8 manifest 配置实现。

## 功能

- 自动从注册表检测 Engine DJ 安装路径
- 显示 Windows 版本和 Engine DJ 版本
- 显示管理员权限状态、UTF-8 支持状态和 manifest 配置状态
- 检测正在运行的 Engine DJ 相关进程，提供终止选项
- 写入 `Engine DJ.exe.manifest` 含 UTF-8 activeCodePage 设置
- 设置注册表 `PreferExternalManifest=1`
- 应用更改后刷新系统设置
- 实时操作日志和进度指示
- 多语言界面：中文 / 日本語 / 한국어 / English
- 启动时自动请求管理员权限

## 系统要求

- Windows 10 1903 或更高版本（需支持 UTF-8 manifest）
- 管理员权限（程序启动时自动请求 UAC 提权）

## 下载

- [GitHub Releases](https://github.com/LaokeQwQ/EngineTools/releases)
- [Forgejo Releases](https://git.laoker.cc/Laoke/EngineTools/releases)

## 使用方法

1. 运行 `EngineTools.exe`（将自动请求管理员权限）
2. 工具自动检测：
   - Engine DJ 安装路径及版本
   - Windows 版本
   - 管理员权限是否已获取
   - 系统 UTF-8 支持是否已开启
   - 外部 Manifest 是否已配置
3. 点击 **修复中日韩等特殊字符读取问题** 应用修复
4. 如果修复后仍有问题，请重启电脑

## 工作原理

详见 [HOW_IT_WORKS_zh.md](HOW_IT_WORKS_zh.md)。

## 开发

### 前置条件

- Go 1.21+
- [Wails v2](https://wails.io/docs/gettingstarted/installation)
- Node.js 16+
- 平台：Windows (amd64)

### 构建

```bash
wails build
```

### 开发模式

```bash
wails dev
```

## 技术栈

- **后端**: Go + [Wails v2](https://wails.io/)
- **前端**: Vue 3 + 原生 CSS
- **注册表操作**: `golang.org/x/sys/windows/registry`
- **进程管理**: Win32 API（syscall）

## 许可证

[Apache License 2.0](LICENSE)