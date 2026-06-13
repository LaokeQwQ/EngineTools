# 仕組み

**[中文](HOW_IT_WORKS_zh.md)** | **[English](HOW_IT_WORKS.md)** | 日本語 | **[한국어](HOW_IT_WORKS_ko.md)**

## 1. 外部 Manifest ファイル

Engine DJ のインストールディレクトリに `Engine DJ.exe.manifest` を作成：

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

## 2. レジストリキー

`HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\SideBySide\PreferExternalManifest` を `1`（DWORD）に設定し、Windows が外部 `.manifest` ファイルを読み取るようにします。

## 3. システム更新

`WM_SETTINGCHANGE` ブロードキャストを送信してシステム設定を更新します。通常は再起動不要です。