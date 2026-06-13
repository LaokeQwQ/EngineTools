# Engine Tools

**[中文](README_zh.md)** | English | **[日本語](README_ja.md)** | **[한국어](README_ko.md)**

Fix CJK (Chinese/Japanese/Korean) and other special character display issues in Engine DJ by applying per-application UTF-8 manifest configuration.

## Features

- Auto-detects Engine DJ installation path from registry
- Displays Windows version and Engine DJ version
- Shows admin privilege status, UTF-8 support status, and manifest configuration status
- Detects running Engine DJ processes and offers to terminate them
- Writes `Engine DJ.exe.manifest` with UTF-8 activeCodePage setting
- Sets `PreferExternalManifest=1` in Windows registry
- Refreshes system settings after applying changes
- Real-time operation log with progress indicator
- Multi-language UI: Chinese / Japanese / Korean / English
- Auto UAC elevation on launch

## Requirements

- Windows 10 version 1903 or later (for UTF-8 manifest support)
- Administrator privileges (the program requests UAC elevation automatically)

## Download

- [GitHub Releases](https://github.com/LaokeQwQ/EngineTools/releases)
- [Forgejo Releases](https://git.laoker.cc/Laoke/EngineTools/releases)

## Usage

1. Run `EngineTools.exe` (will request admin privileges)
2. The tool automatically detects:
   - Engine DJ installation path and version
   - Windows version
   - Whether admin privileges are granted
   - Whether system UTF-8 support is enabled
   - Whether the external manifest is already configured
3. Click **Fix CJK Character Reading Issues** to apply the fix
4. If the issue persists after fixing, restart your computer

## How It Works

See [HOW_IT_WORKS.md](HOW_IT_WORKS.md) for technical details.

## Development

### Prerequisites

- Go 1.21+
- [Wails v2](https://wails.io/docs/gettingstarted/installation)
- Node.js 16+
- Platform: Windows (amd64)

### Build

```bash
wails build
```

### Development Mode

```bash
wails dev
```

## Tech Stack

- **Backend**: Go with [Wails v2](https://wails.io/)
- **Frontend**: Vue 3 + vanilla CSS
- **Registry**: `golang.org/x/sys/windows/registry`
- **Process Management**: Win32 API via `syscall`

## License

[Apache License 2.0](LICENSE)