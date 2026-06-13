# Engine Tools

**[中文](README_zh.md)** | **[English](README.md)** | **[日本語](README_ja.md)** | 한국어

Engine DJ의 중일한 등 특수 문자 읽기 문제를 애플리케이션 수준의 UTF-8 manifest 설정으로 수정합니다.

## 문제

Engine DJ은 시스템 코드 페이지가 UTF-8로 설정되어 있지 않은 경우 CJK 파일 이름과 메타데이터가 깨지거나 누락되어 표시될 수 있습니다. 이 도구는 다음 방법으로 문제를 해결합니다:

1. 애플리케이션에 UTF-8 코드 페이지를 활성화하는 외부 manifest 파일(`Engine DJ.exe.manifest`) 작성
2. Windows가 manifest를 읽도록 `PreferExternalManifest` 레지스트리 키 설정

이 접근 방식은 애플리케이션 수준의 UTF-8 설정을 사용하므로, 시스템 전체 UTF-8 지원을 활성화하는 것(다른 소프트웨어의 호환성 문제를 일으킬 수 있음)보다 안전합니다.

## 기능

- 레지스트리에서 Engine DJ 설치 경로 자동 감지
- Windows 버전 및 Engine DJ 버전 표시
- 관리자 권한 상태, UTF-8 지원 상태, manifest 설정 상태 표시
- 실행 중인 Engine DJ 프로세스 감지 및 종료 옵션 제공
- UTF-8 activeCodePage 설정이 포함된 `Engine DJ.exe.manifest` 작성
- 레지스트리에 `PreferExternalManifest=1` 설정
- 변경 사항 적용 후 시스템 설정 새로고침
- 실시간 작업 로그 및 진행 표시기
- 다국어 UI: 中文 / 日本語 / 한국어 / English
- 실행 시 관리자 권한 자동 요청

## 시스템 요구 사항

- Windows 10 버전 1903 이상 (UTF-8 manifest 지원 필요)
- 관리자 권한 (프로그램이 실행 시 UAC 권한 상승 자동 요청)

## 다운로드

- [GitHub Releases](https://github.com/LaokeQwQ/EngineTools/releases)
- [Forgejo Releases](https://git.laoker.cc/Laoke/EngineTools/releases)

## 사용 방법

1. `EngineTools.exe` 실행 (관리자 권한 자동 요청)
2. 도구가 자동으로 감지하는 항목:
   - Engine DJ 설치 경로 및 버전
   - Windows 버전
   - 관리자 권한 획득 여부
   - 시스템 UTF-8 지원 활성화 여부
   - 외부 Manifest 설정 여부
3. **중일한 등 특수 문자 읽기 문제 수정** 버튼을 클릭하여 수정 적용
4. 수정 후에도 문자 표시 문제가 계속되면 컴퓨터를 재시작하세요

### 시스템 UTF-8 지원이 이미 활성화된 경우

도구가 시스템 전체 UTF-8 지원의 활성화를 감지한 경우("유니코드 UTF-8으로 전 세계 언어 지원 제공"):

1. 제어판 → 지역 → 관리 → 시스템 로캘 변경
2. '유니코드 UTF-8으로 전 세계 언어 지원 제공' 비활성화
3. 이 도구로 애플리케이션 수준에서 활성화

시스템 전체 UTF-8은 다른 소프트웨어의 호환성 문제를 일으킬 수 있기 때문입니다.

## 작동 원리

### 1. 외부 Manifest 파일

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

### 2. 레지스트리 키

`HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\SideBySide\PreferExternalManifest`를 `1`(DWORD)로 설정하여 Windows가 외부 `.manifest` 파일을 읽도록 합니다.

### 3. 시스템 새로고침

`WM_SETTINGCHANGE` 브로드캐스트를 전송하여 시스템 설정을 새로고칩니다. 일반적으로 재시작이 필요하지 않습니다.

## 개발

### 전제 조건

- Go 1.21+
- [Wails v2](https://wails.io/docs/gettingstarted/installation)
- Node.js 16+
- 플랫폼: Windows (amd64)

### 빌드

```bash
wails build
```

### 개발 모드

```bash
wails dev
```

## 기술 스택

- **백엔드**: Go + [Wails v2](https://wails.io/)
- **프론트엔드**: Vue 3 + vanilla CSS
- **레지스트리**: `golang.org/x/sys/windows/registry`
- **프로세스 관리**: Win32 API (syscall)

## 라이선스

[Apache License 2.0](LICENSE)