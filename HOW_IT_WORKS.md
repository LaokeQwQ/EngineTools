# How It Works

**[中文](HOW_IT_WORKS_zh.md)** | English | **[日本語](HOW_IT_WORKS_ja.md)** | **[한국어](HOW_IT_WORKS_ko.md)**

## 1. External Manifest File

Creates `Engine DJ.exe.manifest` in the Engine DJ installation directory:

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

## 2. Registry Key

Sets `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\SideBySide\PreferExternalManifest` to `1` (DWORD), which tells Windows to read external `.manifest` files.

## 3. System Refresh

Sends `WM_SETTINGCHANGE` broadcast to refresh system settings without requiring a full reboot.