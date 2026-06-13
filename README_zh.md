# Engine Tools

**中文** | **[English](README.md)** | **[日本語](README_ja.md)** | **[한국어](README_ko.md)**

修复 Engine DJ 中日韩等特殊字符显示问题，通过应用级别的 UTF-8 manifest 配置实现。

## 问题

Engine DJ 在系统代码页未设置为 UTF-8 时，可能对中日韩文件名和元数据显示乱码或缺失字符。本工具通过以下方式解决：

1. 写入外部 manifest 文件（`Engine DJ.exe.manifest`），为应用程序启用 UTF-8 代码页
2. 设置注册表 `PreferExternalManifest` 键，使 Windows 读取该 manifest

此方法使用应用级别的 UTF-8 配置，比启用系统全局 UTF-8 支持更安全（系统全局 UTF-8 可能导致其他软件兼容性问题）。

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

### 如果系统 UTF-8 支持已开启

如果工具检测到系统全局 UTF-8 支持已开启（"使用 Unicode UTF-8 提供全球语言支持"），会建议你：

1. 前往 控制面板 → 区域 → 管理 → 更改系统区域设置
2. 关闭「使用 Unicode UTF-8 提供全球语言支持」
3. 然后使用本工具按应用级别开启

因为系统全局 UTF-8 可能导致其他软件出现兼容性问题。

## 工作原理

### 1. 外部 Manifest 文件

在 Engine DJ 安装目录下创建 `Engine DJ.exe.manifest`：

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly manifestVersion="1.0" xmlns="urn:schemas-microsoft-com:asm.v1">
  <assemblyIdentity type="win32" name="EngineDJ" version="1.0.0.0"/>
  <application>
    <windowsSettings>
      <activeCodePage xmlns="http://schemas.microsoft.com/SMI/2019/WindowsSettings">UTF-8</activeCodePage>
    </windowsSettings>
  </application>
</assembly>
```

### 2. 注册表键

设置 `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\SideBySide\PreferExternalManifest` 为 `1`（DWORD），使 Windows 读取外部 `.manifest` 文件。

### 3. 系统刷新

发送 `WM_SETTINGCHANGE` 广播刷新系统设置，通常无需重启。

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