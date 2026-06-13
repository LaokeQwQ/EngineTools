# 工作原理

**中文** | **[English](HOW_IT_WORKS.md)** | **[日本語](HOW_IT_WORKS_ja.md)** | **[한국어](HOW_IT_WORKS_ko.md)**

## 1. 外部 Manifest 文件

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

## 2. 注册表键

设置 `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\SideBySide\PreferExternalManifest` 为 `1`（DWORD），使 Windows 读取外部 `.manifest` 文件。

## 3. 系统刷新

发送 `WM_SETTINGCHANGE` 广播刷新系统设置，通常无需重启。