# Engine Tools

**[中文](README_zh.md)** | **[English](README.md)** | 日本語 | **[한국어](README_ko.md)**

Engine DJ の中日韓などの特殊文字の読み取り問題を、アプリケーションレベルの UTF-8 manifest 設定で修正します。

## 問題

Engine DJ は、システムのコードページが UTF-8 に設定されていない場合、CJK のファイル名やメタデータが文字化けたり欠落したりする可能性があります。本ツールは以下の方法で問題を解決します：

1. アプリケーションに UTF-8 コードページを有効にする外部 manifest ファイル（`Engine DJ.exe.manifest`）を書き込む
2. Windows が manifest を読み取るように `PreferExternalManifest` レジストリキーを設定する

このアプローチはアプリケーションレベルの UTF-8 設定を使用するため、システム全体の UTF-8 サポートを有効にする（他のアプリケーションの互換性問題を引き起こす可能性がある）よりも安全です。

## 機能

- レジストリから Engine DJ のインストールパスを自動検出
- Windows バージョンと Engine DJ バージョンの表示
- 管理者権限の状態、UTF-8 サポートの状態、manifest 設定の状態を表示
- 実行中の Engine DJ プロセスを検出して終了オプションを提供
- UTF-8 activeCodePage 設定を含む `Engine DJ.exe.manifest` を書き込み
- レジストリに `PreferExternalManifest=1` を設定
- 変更適用後にシステム設定を更新
- リアルタイムの操作ログと進捗インジケーター
- 多言語 UI：中文 / 日本語 / 한국어 / English
- 起動時に自動的に管理者権限を要求

## 動作要件

- Windows 10 バージョン 1903 以降（UTF-8 manifest サポートが必要）
- 管理者権限（起動時に UAC 昇格を自動要求）

## ダウンロード

- [GitHub Releases](https://github.com/LaokeQwQ/EngineTools/releases)
- [Forgejo Releases](https://git.laoker.cc/Laoke/EngineTools/releases)

## 使い方

1. `EngineTools.exe` を実行（管理者権限を自動要求）
2. ツールが自動検出する項目：
   - Engine DJ のインストールパスとバージョン
   - Windows バージョン
   - 管理者権限の取得状態
   - システムの UTF-8 サポートが有効かどうか
   - 外部 Manifest が設定されているかどうか
3. **中日韓などの特殊文字の読み取り問題を修正** をクリックして修正を適用
4. 修正後も文字表示問題が続く場合は、コンピュータを再起動してください

### システムの UTF-8 サポートが既に有効な場合

ツールがシステム全体の UTF-8 サポートの有効化（「Unicode UTF-8 で世界的な言語サポートを提供する」）を検出した場合、以下を提案します：

1. コントロールパネル → 地域 → 管理 → システムロケールの変更
2. 「Unicode UTF-8 で世界的な言語サポートを提供する」を無効にする
3. その後、このツールでアプリケーションレベルで有効にする

システム全体の UTF-8 は他のソフトウェアの互換性問題を引き起こす可能性があるためです。

## 仕組み

### 1. 外部 Manifest ファイル

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

### 2. レジストリキー

`HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\SideBySide\PreferExternalManifest` を `1`（DWORD）に設定し、Windows が外部 `.manifest` ファイルを読み取るようにします。

### 3. システム更新

`WM_SETTINGCHANGE` ブロードキャストを送信してシステム設定を更新します。通常は再起動不要です。

## 開発

### 前提条件

- Go 1.21+
- [Wails v2](https://wails.io/docs/gettingstarted/installation)
- Node.js 16+
- プラットフォーム：Windows (amd64)

### ビルド

```bash
wails build
```

### 開発モード

```bash
wails dev
```

## 技術スタック

- **バックエンド**: Go + [Wails v2](https://wails.io/)
- **フロントエンド**: Vue 3 + バニラ CSS
- **レジストリ**: `golang.org/x/sys/windows/registry`
- **プロセス管理**: Win32 API（syscall）

## ライセンス

[Apache License 2.0](LICENSE)