# 작동 원리

**[中文](HOW_IT_WORKS_zh.md)** | **[English](HOW_IT_WORKS.md)** | **[日本語](HOW_IT_WORKS_ja.md)** | 한국어

## 1. 외부 Manifest 파일

Engine DJ 설치 디렉토리에 `Engine DJ.exe.manifest` 생성:

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

## 2. 레지스트리 키

`HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\SideBySide\PreferExternalManifest`를 `1`(DWORD)로 설정하여 Windows가 외부 `.manifest` 파일을 읽도록 합니다.

## 3. 시스템 새로고침

`WM_SETTINGCHANGE` 브로드캐스트를 전송하여 시스템 설정을 새로고침합니다. 일반적으로 재시작이 필요하지 않습니다.