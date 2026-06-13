# Engine DJ Unicode Fix Tool

Fix CJK (Chinese/Japanese/Korean) and other special character display issues in Engine DJ by applying per-application UTF-8 manifest configuration.

## Problem

Engine DJ may display garbled or missing characters for CJK file names and metadata when the system code page is not set to UTF-8. This tool resolves the issue by:

1. Writing an external manifest file (`Engine DJ.exe.manifest`) that enables UTF-8 code page for the application
2. Setting the `PreferExternalManifest` registry key so Windows reads the manifest

This approach uses per-application UTF-8 configuration, which is safer than enabling system-wide UTF-8 support (which can break other applications).

## Features

- Auto-detects Engine DJ installation path from registry
- Checks system UTF-8 code page status (ACP=65001)
- Detects running Engine DJ processes and offers to terminate them
- Writes `Engine DJ.exe.manifest` with UTF-8 activeCodePage setting
- Sets `PreferExternalManifest=1` in Windows registry
- Refreshes system settings after applying changes
- Real-time operation log with progress indicator
- Multi-language UI (Chinese / Japanese / Korean / English)

## Requirements

- Windows 10 version 1903 or later (for UTF-8 manifest support)
- Administrator privileges (the program requests UAC elevation automatically)

## Download

See the [Releases](https://github.com/LaokeQwQ/EngineTools/releases) page.

## Usage

1. Run `EngineTools.exe` (requires admin privileges)
2. The tool automatically detects:
   - Engine DJ installation path
   - Whether system UTF-8 support is enabled
   - Whether the external manifest is already configured
3. Click **Fix CJK Character Reading Issues** to apply the fix
4. If the issue persists after fixing, restart your computer

### If System UTF-8 Is Already Enabled

If the tool detects that system-wide UTF-8 support is already enabled ("Use Unicode UTF-8 for worldwide language support"), it will suggest you:

1. Go to Control Panel → Region → Administrative → Change system locale
2. Disable "Beta: Use Unicode UTF-8 for worldwide language support"
3. Then use this tool to apply the per-application fix

This is because system-wide UTF-8 can cause compatibility issues with other software.

## How It Works

The tool applies the following changes:

### 1. External Manifest File

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

### 2. Registry Key

Sets `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\SideBySide\PreferExternalManifest` to `1` (DWORD), which tells Windows to read external `.manifest` files.

### 3. System Refresh

Sends `WM_SETTINGCHANGE` broadcast to refresh system settings without requiring a full reboot (though a reboot may still be needed in some cases).

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

The output binary will be in `build/bin/EngineTools.exe`.

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