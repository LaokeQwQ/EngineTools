package i18n

import (
	"os"
	"strings"
	"syscall"
)

type Lang string

const (
	ZH Lang = "zh"
	JA Lang = "ja"
	KO Lang = "ko"
	EN Lang = "en"
)

type Messages struct {
	AppTitle                string `json:"appTitle"`
	InstallPathLabel        string `json:"installPathLabel"`
	InstallPathNotFound     string `json:"installPathNotFound"`
	EngineVersionLabel      string `json:"engineVersionLabel"`
	WindowsVersionLabel     string `json:"windowsVersionLabel"`
	UTF8StatusLabel         string `json:"utf8StatusLabel"`
	UTF8Enabled             string `json:"utf8Enabled"`
	UTF8Disabled            string `json:"utf8Disabled"`
	ManifestStatusLabel     string `json:"manifestStatusLabel"`
	ManifestExists          string `json:"manifestExists"`
	ManifestNotExists       string `json:"manifestNotExists"`
	FixButton               string `json:"fixButton"`
	RestoreButton           string `json:"restoreButton"`
	OpenRegionSettings      string `json:"openRegionSettings"`
	UTF8AlreadyEnabled      string `json:"utf8AlreadyEnabled"`
	UTF8AlreadyEnabledTip   string `json:"utf8AlreadyEnabledTip"`
	ProcessRunningTitle     string `json:"processRunningTitle"`
	ProcessRunningMessage   string `json:"processRunningMessage"`
	KillingProcesses        string `json:"killingProcesses"`
	ProcessKilled           string `json:"processKilled"`
	NoProcessRunning        string `json:"noProcessRunning"`
	WritingManifest         string `json:"writingManifest"`
	ManifestWritten         string `json:"manifestWritten"`
	ManifestWriteError      string `json:"manifestWriteError"`
	ExeNotFound             string `json:"exeNotFound"`
	WritingRegistry         string `json:"writingRegistry"`
	RegistryWritten         string `json:"registryWritten"`
	RegistryWriteError      string `json:"registryWriteError"`
	DeletingManifests       string `json:"deletingManifests"`
	ManifestsDeleted        string `json:"manifestsDeleted"`
	DeletingRegistry        string `json:"deletingRegistry"`
	RegistryDeleted         string `json:"registryDeleted"`
	RefreshingSystem        string `json:"refreshingSystem"`
	SystemRefreshed         string `json:"systemRefreshed"`
	FixComplete             string `json:"fixComplete"`
	FixCompleteTip          string `json:"fixCompleteTip"`
	RestoreComplete         string `json:"restoreComplete"`
	RestoreCompleteTip      string `json:"restoreCompleteTip"`
	FixFailed               string `json:"fixFailed"`
	LogPrefix               string `json:"logPrefix"`
	Checking                string `json:"checking"`
	StatusChecking          string `json:"statusChecking"`
	ProgressDetecting       string `json:"progressDetecting"`
	ProgressFixing          string `json:"progressFixing"`
	ProgressRestoring       string `json:"progressRestoring"`
	ProgressDone            string `json:"progressDone"`
	Language                string `json:"language"`
	ACPCodePage             string `json:"acpCodePage"`
	NotInstalled            string `json:"notInstalled"`
	Version                 string `json:"version"`
	MarqueeFree             string `json:"marqueeFree"`
	MarqueeStar             string `json:"marqueeStar"`
	AdminStatusLabel        string `json:"adminStatusLabel"`
	AdminYes                string `json:"adminYes"`
	AdminNo                 string `json:"adminNo"`
	StemsStatusLabel        string `json:"stemsStatusLabel"`
	StemsDetected           string `json:"stemsDetected"`
	StemsNotFound           string `json:"stemsNotFound"`
	RestoreConfirmTitle     string `json:"restoreConfirmTitle"`
	RestoreConfirmMessage   string `json:"restoreConfirmMessage"`
	BackupReminderTitle     string `json:"backupReminderTitle"`
	BackupReminderMessage   string `json:"backupReminderMessage"`
	TabStatus               string `json:"tabStatus"`
	TabDatabase             string `json:"tabDatabase"`
	TabTools                string `json:"tabTools"`
	DBLibraryPathLabel      string `json:"dbLibraryPathLabel"`
	DBLibraryNotFound       string `json:"dbLibraryNotFound"`
	DBBackupButton          string `json:"dbBackupButton"`
	DBBackupNoteLabel       string `json:"dbBackupNoteLabel"`
	DBBackupNotePlaceholder string `json:"dbBackupNotePlaceholder"`
	DBBackingUp             string `json:"dbBackingUp"`
	DBBackupComplete        string `json:"dbBackupComplete"`
	DBBackupError           string `json:"dbBackupError"`
	DBSelectDriveLabel       string `json:"dbSelectDriveLabel"`
	DBSelectDrivePlaceholder string `json:"dbSelectDrivePlaceholder"`
	DBSelectDriveConfirm     string `json:"dbSelectDriveConfirm"`
	DBDriveNotFound          string `json:"dbDriveNotFound"`
	DBRestoreButton         string `json:"dbRestoreButton"`
	DBRestoreSelectDate     string `json:"dbRestoreSelectDate"`
	DBRestoreConfirmTitle   string `json:"dbRestoreConfirmTitle"`
	DBRestoreConfirmMessage string `json:"dbRestoreConfirmMessage"`
	DBRestoring             string `json:"dbRestoring"`
	DBRestoreComplete       string `json:"dbRestoreComplete"`
	DBOptimizeButton        string `json:"dbOptimizeButton"`
	DBOptimizing            string `json:"dbOptimizing"`
	DBOptimizeComplete      string `json:"dbOptimizeComplete"`
	DBRepairButton          string `json:"dbRepairButton"`
	DBRepairing             string `json:"dbRepairing"`
	DBRepairComplete        string `json:"dbRepairComplete"`
	DBRepairStart           string `json:"dbRepairStart"`
	DBRepairDone            string `json:"dbRepairDone"`
	DBNoteLabel             string `json:"dbNoteLabel"`
	DBNoneFound             string `json:"dbNoneFound"`
	DBNoBackups             string `json:"dbNoBackups"`
	MSICleanupButton        string `json:"msiCleanupButton"`
	MSICleanupTitle         string `json:"msiCleanupTitle"`
	MSICleanupDescription   string `json:"msiCleanupDescription"`
	MSIScanning             string `json:"msiScanning"`
	MSIFoundOrphans         string `json:"msiFoundOrphans"`
	MSINoOrphans            string `json:"msiNoOrphans"`
	MSICleaning             string `json:"msiCleaning"`
	MSICleanComplete        string `json:"msiCleanComplete"`
	MSICleanError           string `json:"msiCleanError"`
	MSIConfirmTitle         string `json:"msiConfirmTitle"`
	MSIConfirmMessage       string `json:"msiConfirmMessage"`
	ID3EditorTitle          string `json:"id3EditorTitle"`
	ID3SelectFile           string `json:"id3SelectFile"`
	ID3PickFileButton       string `json:"id3PickFileButton"`
	ID3SaveButton           string `json:"id3SaveButton"`
	ID3ClearAllButton       string `json:"id3ClearAllButton"`
	USBUnlockTitle          string `json:"usbUnlockTitle"`
	USBUnlockDescription    string `json:"usbUnlockDescription"`
	USBUnlockButton         string `json:"usbUnlockButton"`
	USBScanButton           string `json:"usbScanButton"`
	USBNoDevice             string `json:"usbNoDevice"`
	MIDI2Title              string `json:"midi2Title"`
	MIDI2Description        string `json:"midi2Description"`
	MIDI2Enabled            string `json:"midi2Enabled"`
	MIDI2Disabled           string `json:"midi2Disabled"`
	MIDI2Unavailable        string `json:"midi2Unavailable"`
	MIDI2DisableButton      string `json:"midi2DisableButton"`
	MIDI2EnableButton       string `json:"midi2EnableButton"`
	MIDI2Disabling          string `json:"midi2Disabling"`
	MIDI2Enabling           string `json:"midi2Enabling"`
	StemsEngineLabel        string `json:"stemsEngineLabel"`
	LogAnalysisTitle        string `json:"logAnalysisTitle"`
	LogAnalysisDescription  string `json:"logAnalysisDescription"`
	LogAnalyzeButton        string `json:"logAnalyzeButton"`
	LogAnalyzing            string `json:"logAnalyzing"`
	LogOpenDir              string `json:"logOpenDir"`
	LogTotalFiles           string `json:"logTotalFiles"`
	LogTotalLines           string `json:"logTotalLines"`
	LogInfoCount            string `json:"logInfoCount"`
	LogWarningCount         string `json:"logWarningCount"`
	LogErrorCount           string `json:"logErrorCount"`
	LogTopWarnings          string `json:"logTopWarnings"`
	LogTopErrors            string `json:"logTopErrors"`
	LogNoFiles              string `json:"logNoFiles"`
	CacheCleanTitle         string `json:"cacheCleanTitle"`
	CacheCleanDescription   string `json:"cacheCleanDescription"`
	CacheCleanButton        string `json:"cacheCleanButton"`
	CacheCleaning           string `json:"cacheCleaning"`
	CacheCleanComplete      string `json:"cacheCleanComplete"`
	CacheLatestFile         string `json:"cacheLatestFile"`
	UpdateBadge             string `json:"updateBadge"`
	UpdateChecking          string `json:"updateChecking"`
	UpdateTooltip           string `json:"updateTooltip"`
	OperationSuccess        string `json:"operationSuccess"`
}

var translations = map[Lang]Messages{
	ZH: {
		AppTitle:              "Engine Tools",
		InstallPathLabel:      "安装路径",
		InstallPathNotFound:   "未找到 Engine DJ 安装路径",
		EngineVersionLabel:     "Engine DJ 版本",
		WindowsVersionLabel:    "Windows 版本",
		UTF8StatusLabel:       "系统 UTF-8 支持",
		UTF8Enabled:           "已开启",
		UTF8Disabled:          "未开启",
		ManifestStatusLabel:   "外部 Manifest",
		ManifestExists:        "已配置",
		ManifestNotExists:     "未配置",
		FixButton:             "修复 Unicode 特殊字符读取问题（中日韩/希腊文/拉丁扩展等）",
		RestoreButton:         "还原修复",
		OpenRegionSettings:    "前往区域设置",
		UTF8AlreadyEnabled:   "系统已开启 UTF-8 支持",
		UTF8AlreadyEnabledTip: "检测到系统已开启 UTF-8 支持，建议先前往 控制面板 → 区域 → 管理 → 更改系统区域设置，关闭「使用 Unicode UTF-8 提供全球语言支持」，然后使用本工具按应用级别开启。",
		ProcessRunningTitle:    "检测到正在运行的程序",
		ProcessRunningMessage:  "以下程序正在运行，需要关闭后才能继续：\n\n%s\n\n是否关闭这些程序？",
		KillingProcesses:       "正在结束进程...",
		ProcessKilled:         "已终止进程",
		NoProcessRunning:      "未检测到运行中的程序",
		WritingManifest:       "正在写入 manifest 文件...",
		ManifestWritten:       "Manifest 文件写入成功",
		ManifestWriteError:    "Manifest 文件写入失败",
		ExeNotFound:           "未找到 Engine DJ.exe",
		WritingRegistry:       "正在写入注册表...",
		RegistryWritten:       "注册表写入成功",
		RegistryWriteError:    "注册表写入失败",
		DeletingManifests:       "正在删除 manifest 文件...",
		ManifestsDeleted:        "Manifest 文件已删除",
		DeletingRegistry:        "正在删除注册表项...",
		RegistryDeleted:         "注册表项已删除",
		RefreshingSystem:      "正在刷新系统设置...",
		SystemRefreshed:       "系统设置已刷新",
		FixComplete:          "修复完成",
		FixCompleteTip:        "修复已完成！如果仍遇到字符显示问题，请重启电脑后重试。",
		RestoreComplete:         "还原完成",
		RestoreCompleteTip:      "修复已还原！如需重新修复，请再次点击修复按钮。",
		FixFailed:            "修复失败",
		LogPrefix:            "[%s] %s",
		Checking:             "正在检测",
		StatusChecking:       "正在检测系统状态...",
		ProgressDetecting:    "检测中...",
		ProgressFixing:       "修复中...",
		ProgressRestoring:       "还原中...",
		ProgressDone:         "完成",
		Language:             "语言",
		ACPCodePage:          "ACP 代码页",
		NotInstalled:          "未安装",
		Version:               "版本",
		MarqueeFree:           "该程序免费分发，若您为收费获取请及时申诉",
		MarqueeStar:           "若您喜欢，请给项目点个 Star",
		AdminStatusLabel:     "管理员权限",
		AdminYes:            "已获取",
		AdminNo:             "未获取",
		StemsStatusLabel:    "STEM 分离引擎",
		StemsDetected:       "已检测到",
		StemsNotFound:       "未检测到",
		RestoreConfirmTitle:     "确认还原",
		RestoreConfirmMessage:   "确定要还原修复吗？所有修改将被撤销。",
		BackupReminderTitle:     "备份提醒",
		BackupReminderMessage:   "建议在操作前备份数据库，是否继续？",
		TabStatus:               "状态",
		TabDatabase:             "数据库",
		TabTools:                "工具",
		DBLibraryPathLabel:      "数据库路径",
		DBLibraryNotFound:       "未找到 Engine Library",
		DBBackupButton:          "备份数据库",
		DBBackupNoteLabel:       "备份备注",
		DBBackupNotePlaceholder: "可选：输入备注信息",
		DBBackingUp:             "正在备份...",
		DBBackupComplete:        "备份完成",
		DBBackupError:           "备份失败",
		DBRestoreButton:         "还原数据库",
		DBRestoreSelectDate:     "选择要还原的日期",
		DBSelectDriveLabel:       "选择盘符",
		DBSelectDrivePlaceholder: "选择驱动器",
		DBSelectDriveConfirm:     "确定",
		DBRestoreConfirmTitle:   "确认还原数据库",
		DBRestoreConfirmMessage: "确定要还原到选定日期的备份吗？当前数据将被替换。",
		DBRestoring:             "正在还原...",
		DBRestoreComplete:       "数据库还原完成",
		DBOptimizeButton:        "优化数据库",
		DBOptimizing:            "正在优化...",
		DBOptimizeComplete:      "优化完成",
		DBRepairButton:          "修复数据库",
		DBRepairing:             "正在修复...",
		DBRepairComplete:        "修复完成",
		DBRepairStart:           "正在检查和修复数据库...",
		DBRepairDone:            "数据库修复完成",
		DBNoteLabel:             "备注",
		DBNoneFound:             "未找到",
		DBNoBackups:             "暂无备份",
		MSICleanupButton:        "MSI 残留清理",
		MSICleanupTitle:         "MSI 残留清理",
		MSICleanupDescription:   "用于 Engine DJ 安装/卸载/更新时遇到残留文件导致的问题",
		MSIScanning:             "正在扫描...",
		MSIFoundOrphans:         "发现 %d 个残留",
		MSINoOrphans:            "未发现 MSI 残留",
		MSICleaning:             "正在清理...",
		MSICleanComplete:        "清理完成",
		MSICleanError:           "清理失败",
		MSIConfirmTitle:         "确认清理",
		MSIConfirmMessage:       "确定要清理选中的 MSI 残留吗？此操作不可恢复。",
		ID3EditorTitle:          "ID3 标签编辑器",
		ID3SelectFile:           "选择音频文件",
		ID3PickFileButton:       "选择文件",
		ID3SaveButton:           "保存",
		ID3ClearAllButton:       "清除全部",
		USBUnlockTitle:          "U盘解锁",
		USBUnlockDescription:    "用于 Engine DJ 安装/卸载/更新时遇到文件占用问题",
		USBUnlockButton:         "解锁",
		USBScanButton:           "扫描占用进程",
		USBNoDevice:             "未检测到带有 Engine Library 的 U盘",
		MIDI2Title:              "MIDI 2.0 控制",
		MIDI2Description:        "禁用 Windows 11 的 MIDI 2.0 功能，同时保留 MIDI 1.0 服务",
		MIDI2Enabled:            "MIDI 2.0 当前已启用",
		MIDI2Disabled:           "MIDI 2.0 当前已禁用",
		MIDI2Unavailable:        "此系统未找到 MIDI 2.0 服务",
		MIDI2DisableButton:      "禁用 MIDI 2.0",
		MIDI2EnableButton:       "启用 MIDI 2.0",
		MIDI2Disabling:          "正在禁用...",
		MIDI2Enabling:           "正在启用...",
		StemsEngineLabel:        "STEM 分离引擎",
		LogAnalysisTitle:        "日志分析",
		LogAnalyzeButton:       "分析日志",
		LogAnalyzing:            "正在分析...",
		LogOpenDir:              "打开日志目录",
		LogTotalFiles:           "日志文件数",
		LogTotalLines:           "总行数",
		LogInfoCount:            "信息",
		LogWarningCount:         "警告",
		LogErrorCount:           "错误",
		LogTopWarnings:          "高频警告",
		LogTopErrors:            "高频错误",
		LogNoFiles:              "未找到日志文件",
		CacheCleanTitle:         "缓存清理",
		CacheCleanDescription:   "清除 Engine DJ 界面缓存，修复更新后的显示异常",
		CacheCleanButton:        "清除缓存",
		CacheCleaning:           "正在清除...",
		CacheCleanComplete:      "缓存已清除",
		UpdateBadge:             "有更新",
		UpdateChecking:          "检查中...",
		UpdateTooltip:           "有新版本可用",
		OperationSuccess:        "操作完成",
	},
	JA: {
		AppTitle:              "Engine Tools",
		InstallPathLabel:      "インストールパス",
		InstallPathNotFound:   "Engine DJ のインストールパスが見つかりません",
		EngineVersionLabel:     "Engine DJ バージョン",
		WindowsVersionLabel:    "Windows バージョン",
		UTF8StatusLabel:       "システム UTF-8 サポート",
		UTF8Enabled:           "有効",
		UTF8Disabled:          "無効",
		ManifestStatusLabel:   "外部 Manifest",
		ManifestExists:        "設定済み",
		ManifestNotExists:     "未設定",
		FixButton:             "中日韓・ギリシャ文字・ラテン拡張など Unicode 特殊文字の文字化けを修正",
		RestoreButton:         "修正を元に戻す",
		OpenRegionSettings:    "地域設定を開く",
		UTF8AlreadyEnabled:   "システムの UTF-8 サポートは既に有効です",
		UTF8AlreadyEnabledTip: "システムの UTF-8 サポートが既に有効になっています。先に コントロールパネル → 地域 → 管理 → システムロケールの変更 で「Unicode UTF-8 で世界的な言語サポートを提供する」を無効にしてから、このツールでアプリケーションレベルで有効にしてください。",
		ProcessRunningTitle:    "実行中のプログラムが検出されました",
		ProcessRunningMessage:  "以下のプログラムが実行中です。続行するには閉じる必要があります：\n\n%s\n\nこれらのプログラムを閉じますか？",
		KillingProcesses:       "プロセスを終了中...",
		ProcessKilled:         "プロセスを終了しました",
		NoProcessRunning:      "実行中のプログラムはありません",
		WritingManifest:       "Manifest ファイルを書き込み中...",
		ManifestWritten:       "Manifest ファイルの書き込みに成功しました",
		ManifestWriteError:    "Manifest ファイルの書き込みに失敗しました",
		ExeNotFound:           "Engine DJ.exe が見つかりません",
		WritingRegistry:       "レジストリを書き込み中...",
		RegistryWritten:       "レジストリの書き込みに成功しました",
		RegistryWriteError:    "レジストリの書き込みに失敗しました",
		DeletingManifests:       "Manifest ファイルを削除中...",
		ManifestsDeleted:        "Manifest ファイルを削除しました",
		DeletingRegistry:        "レジストリエントリを削除中...",
		RegistryDeleted:         "レジストリエントリを削除しました",
		RefreshingSystem:      "システム設定を更新中...",
		SystemRefreshed:       "システム設定を更新しました",
		FixComplete:          "修正完了",
		FixCompleteTip:        "修正が完了しました！文字の表示問題が続く場合は、コンピュータを再起動してください。",
		RestoreComplete:         "復元完了",
		RestoreCompleteTip:      "修正を元に戻しました。再度修正するには修正ボタンをクリックしてください。",
		FixFailed:            "修正に失敗しました",
		LogPrefix:            "[%s] %s",
		Checking:             "確認中",
		StatusChecking:       "システム状態を確認中...",
		ProgressDetecting:    "確認中...",
		ProgressFixing:       "修正中...",
		ProgressRestoring:       "復元中...",
		ProgressDone:         "完了",
		Language:             "言語",
		ACPCodePage:          "ACP コードページ",
		NotInstalled:         "未インストール",
		Version:              "バージョン",
		MarqueeFree:           "このツールは無料で配布されています。有料で入手された場合は返金をご申請ください",
		MarqueeStar:           "気に入ったら、プロジェクトに Star をお願いします",
		AdminStatusLabel:     "管理者権限",
		AdminYes:            "取得済み",
		AdminNo:             "未取得",
		StemsStatusLabel:    "STEM 分離エンジン",
		StemsDetected:       "検出されました",
		StemsNotFound:       "検出されません",
		RestoreConfirmTitle:     "復元の確認",
		RestoreConfirmMessage:   "修正を元に戻しますか？すべての変更が取り消されます。",
		BackupReminderTitle:     "バックアップの注意",
		BackupReminderMessage:   "操作前にデータベースのバックアップを推奨します。続行しますか？",
		TabStatus:               "ステータス",
		TabDatabase:             "データベース",
		TabTools:                "ツール",
		DBLibraryPathLabel:      "データベースパス",
		DBLibraryNotFound:       "Engine Library が見つかりません",
		DBBackupButton:          "バックアップ",
		DBBackupNoteLabel:       "メモ",
		DBBackupNotePlaceholder: "任意：メモを入力",
		DBBackingUp:             "バックアップ中...",
		DBBackupComplete:        "バックアップ完了",
		DBBackupError:           "バックアップ失敗",
		DBRestoreButton:         "復元",
		DBRestoreSelectDate:     "復元する日付を選択",
		DBSelectDriveLabel:       "ドライブを選択",
		DBSelectDrivePlaceholder: "ドライブを選んでください",
		DBSelectDriveConfirm:     "確認",
		DBRestoreConfirmTitle:   "データベース復元の確認",
		DBRestoreConfirmMessage: "選択した日付のバックアップに復元しますか？現在のデータは上書きされます。",
		DBRestoring:             "復元中...",
		DBRestoreComplete:       "データベース復元完了",
		DBOptimizeButton:        "最適化",
		DBOptimizing:            "最適化中...",
		DBOptimizeComplete:      "最適化完了",
		DBRepairButton:          "修復",
		DBRepairing:             "修復中...",
		DBRepairComplete:        "修復完了",
		DBRepairStart:           "データベースを修復しています...",
		DBRepairDone:            "データベース修復完了",
		DBNoteLabel:             "メモ",
		DBNoneFound:             "見つかりません",
		DBNoBackups:             "バックアップなし",
		MSICleanupButton:        "MSI クリーンアップ",
		MSICleanupTitle:         "MSI クリーンアップ",
		MSICleanupDescription:   "システム内の不要な MSI インストール残留をスキャンして削除します",
		MSIScanning:             "スキャン中...",
		MSIFoundOrphans:         "%d 件の残留を発見",
		MSINoOrphans:            "MSI 残留は見つかりません",
		MSICleaning:             "クリーンアップ中...",
		MSICleanComplete:        "クリーンアップ完了",
		MSICleanError:           "クリーンアップ失敗",
		MSIConfirmTitle:         "クリーンアップの確認",
		MSIConfirmMessage:       "選択した MSI 残留を削除しますか？この操作は元に戻せません。",
		ID3EditorTitle:          "ID3 タグエディタ",
		ID3SelectFile:           "オーディオファイルを選択",
		ID3PickFileButton:       "ファイルを選択",
		ID3SaveButton:           "保存",
		ID3ClearAllButton:       "すべてクリア",
		USBUnlockTitle:          "USB ロック解除",
		USBUnlockDescription:    "Engine DJ のインストール/アンインストール/更新時のファイルロックの問題に使用",
		USBUnlockButton:         "ロック解除",
		USBScanButton:           "ブロックプロセスをスキャン",
		USBNoDevice:             "Engine Library を含む USB デバイスが検出されません",
		MIDI2Title:              "MIDI 2.0 制御",
		MIDI2Description:        "Windows 11 の MIDI 2.0 機能を無効にし、MIDI 1.0 サービスを保持",
		MIDI2Enabled:            "MIDI 2.0 は現在有効です",
		MIDI2Disabled:           "MIDI 2.0 は現在無効です",
		MIDI2Unavailable:        "このシステムに MIDI 2.0 サービスが見つかりません",
		MIDI2DisableButton:      "MIDI 2.0 を無効にする",
		MIDI2EnableButton:       "MIDI 2.0 を有効にする",
		MIDI2Disabling:          "無効化中...",
		MIDI2Enabling:           "有効化中...",
		StemsEngineLabel:        "STEM 分離エンジン",
		LogAnalysisTitle:        "ログ分析",
		LogAnalyzeButton:       "ログを分析",
		LogAnalyzing:            "分析中...",
		LogOpenDir:              "ログフォルダを開く",
		LogTotalFiles:           "ログファイル数",
		LogTotalLines:           "総行数",
		LogInfoCount:            "情報",
		LogWarningCount:         "警告",
		LogErrorCount:           "エラー",
		LogTopWarnings:          "頻出警告",
		LogTopErrors:            "頻出エラー",
		LogNoFiles:              "ログファイルが見つかりません",
		CacheCleanTitle:         "キャッシュクリア",
		CacheCleanDescription:   "Engine DJ の UI キャッシュを削除して表示の問題を修正",
		CacheCleanButton:        "キャッシュを削除",
		CacheCleaning:           "削除中...",
		CacheCleanComplete:      "キャッシュを削除しました",
		UpdateBadge:             "更新あり",
		UpdateChecking:          "確認中...",
		UpdateTooltip:           "新しいバージョンが利用可能です",
		OperationSuccess:        "操作完了",
	},
	KO: {
		AppTitle:              "Engine Tools",
		InstallPathLabel:      "설치 경로",
		InstallPathNotFound:   "Engine DJ 설치 경로를 찾을 수 없습니다",
		EngineVersionLabel:     "Engine DJ 버전",
		WindowsVersionLabel:    "Windows 버전",
		UTF8StatusLabel:       "시스템 UTF-8 지원",
		UTF8Enabled:           "활성화됨",
		UTF8Disabled:          "비활성화됨",
		ManifestStatusLabel:   "외부 Manifest",
		ManifestExists:        "구성됨",
		ManifestNotExists:     "미구성",
		FixButton:            "중일한·그리스 문자·라틴 확장 등 Unicode 특수 문자 인코딩 문제 수정",
		RestoreButton:         "수정 되돌리기",
		OpenRegionSettings:    "지역 설정 열기",
		UTF8AlreadyEnabled:   "시스템 UTF-8 지원이 이미 활성화되어 있습니다",
		UTF8AlreadyEnabledTip: "시스템 UTF-8 지원이 이미 활성화되어 있습니다. 먼저 제어판 → 지역 → 관리 → 시스템 로캘 변경에서 '유니코드 UTF-8으로 전 세계 언어 지원 제공'을 비활성화한 다음 이 도구로 애플리케이션 수준에서 활성화하세요.",
		ProcessRunningTitle:    "실행 중인 프로그램이 감지되었습니다",
		ProcessRunningMessage:  "다음 프로그램이 실행 중입니다. 계속하려면 종료해야 합니다:\n\n%s\n\n이 프로그램들을 종료하시겠습니까?",
		KillingProcesses:       "프로세스 종료 중...",
		ProcessKilled:        "프로세스가 종료되었습니다",
		NoProcessRunning:      "실행 중인 프로그램이 없습니다",
		WritingManifest:       "Manifest 파일 쓰는 중...",
		ManifestWritten:       "Manifest 파일 쓰기 성공",
		ManifestWriteError:    "Manifest 파일 쓰기 실패",
		ExeNotFound:          "Engine DJ.exe를 찾을 수 없습니다",
		WritingRegistry:       "레지스트리 쓰는 중...",
		RegistryWritten:       "레지스트리 쓰기 성공",
		RegistryWriteError:    "레지스트리 쓰기 실패",
		DeletingManifests:       "Manifest 파일 삭제 중...",
		ManifestsDeleted:        "Manifest 파일이 삭제되었습니다",
		DeletingRegistry:        "레지스트리 항목 삭제 중...",
		RegistryDeleted:         "레지스트리 항목이 삭제되었습니다",
		RefreshingSystem:      "시스템 설정 새로고침 중...",
		SystemRefreshed:        "시스템 설정이 새로고침되었습니다",
		FixComplete:          "수정 완료",
		FixCompleteTip:        "수정이 완료되었습니다! 문자 표시 문제가 계속되면 컴퓨터를 재시작하세요.",
		RestoreComplete:         "복원 완료",
		RestoreCompleteTip:      "수정이 되돌려졌습니다. 다시 수정하려면 수정 버튼을 클릭하세요.",
		FixFailed:            "수정 실패",
		LogPrefix:            "[%s] %s",
		Checking:             "확인 중",
		StatusChecking:       "시스템 상태 확인 중...",
		ProgressDetecting:    "확인 중...",
		ProgressFixing:       "수정 중...",
		ProgressRestoring:       "복원 중...",
		ProgressDone:         "완료",
		Language:             "언어",
		ACPCodePage:          "ACP 코드 페이지",
		NotInstalled:         "설치되지 않음",
		Version:              "버전",
		MarqueeFree:           "이 프로그램은 무료로 배포됩니다. 유료로 얻으셨다면 환불을 신청하세요",
		MarqueeStar:           "마음에 드시면 프로젝트에 Star를 부탁드립니다",
		AdminStatusLabel:     "관리자 권한",
		AdminYes:            "활성",
		AdminNo:             "미활성",
		StemsStatusLabel:    "STEM 분리 엔진",
		StemsDetected:       "감지됨",
		StemsNotFound:       "감지되지 않음",
		RestoreConfirmTitle:     "복원 확인",
		RestoreConfirmMessage:   "수정 사항을 되돌리시겠습니까? 모든 변경 사항이 취소됩니다.",
		BackupReminderTitle:     "백업 알림",
		BackupReminderMessage:   "작업 전에 데이터베이스 백업을 권장합니다. 계속하시겠습니까?",
		TabStatus:               "상태",
		TabDatabase:             "데이터베이스",
		TabTools:                "도구",
		DBLibraryPathLabel:      "데이터베이스 경로",
		DBLibraryNotFound:       "Engine Library를 찾을 수 없습니다",
		DBBackupButton:          "백업",
		DBBackupNoteLabel:       "메모",
		DBBackupNotePlaceholder: "선택: 메모 입력",
		DBBackingUp:             "백업 중...",
		DBBackupComplete:        "백업 완료",
		DBBackupError:           "백업 실패",
		DBRestoreButton:         "복원",
		DBRestoreSelectDate:     "복원할 날짜 선택",
		DBSelectDriveLabel:       "드라이브 선택",
		DBSelectDrivePlaceholder: "데이터베이스를 검색할 드라이브 선택",
		DBSelectDriveConfirm:     "확인",
		DBRestoreConfirmTitle:   "데이터베이스 복원 확인",
		DBRestoreConfirmMessage: "선택한 날짜의 백업으로 복원하시겠습니까? 현재 데이터가 대체됩니다.",
		DBRestoring:             "복원 중...",
		DBRestoreComplete:       "데이터베이스 복원 완료",
		DBOptimizeButton:        "최적화",
		DBOptimizing:            "최적화 중...",
		DBOptimizeComplete:      "최적화 완료",
		DBRepairButton:          "복구",
		DBRepairing:             "복구 중...",
		DBRepairComplete:        "복구 완료",
		DBRepairStart:           "데이터베이스 수리 중...",
		DBRepairDone:            "데이터베이스 수리 완료",
		DBNoteLabel:             "메모",
		DBNoneFound:             "찾을 수 없음",
		DBNoBackups:             "백업 없음",
		MSICleanupButton:        "MSI 정리",
		MSICleanupTitle:         "MSI 정리",
		MSICleanupDescription:   "시스템에서 불필요한 MSI 설치 잔여물을 검사하여 삭제합니다",
		MSIScanning:             "검사 중...",
		MSIFoundOrphans:         "%d개의 잔여물 발견",
		MSINoOrphans:            "MSI 잔여물이 발견되지 않음",
		MSICleaning:             "정리 중...",
		MSICleanComplete:        "정리 완료",
		MSICleanError:           "정리 실패",
		MSIConfirmTitle:         "정리 확인",
		MSIConfirmMessage:       "선택한 MSI 잔여물을 삭제하시겠습니까? 이 작업은 되돌릴 수 없습니다.",
		ID3EditorTitle:          "ID3 태그 편집기",
		ID3SelectFile:           "오디오 파일 선택",
		ID3PickFileButton:       "파일 선택",
		ID3SaveButton:           "저장",
		ID3ClearAllButton:       "모두 지우기",
		USBUnlockTitle:          "USB 잠금 해제",
		USBUnlockDescription:    "Engine DJ 설치/제거/업데이트 시 파일 잠금 문제에 사용",
		USBUnlockButton:         "잠금 해제",
		USBScanButton:           "차단 프로세스 스캔",
		USBNoDevice:             "Engine Library가 포함된 USB 장치가 감지되지 않음",
		MIDI2Title:              "MIDI 2.0 제어",
		MIDI2Description:        "Windows 11의 MIDI 2.0 기능을 비활성화하고 MIDI 1.0 서비스를 유지",
		MIDI2Enabled:            "MIDI 2.0이 현재 활성화되어 있습니다",
		MIDI2Disabled:           "MIDI 2.0이 현재 비활성화되어 있습니다",
		MIDI2Unavailable:        "이 시스템에서 MIDI 2.0 서비스를 찾을 수 없습니다",
		MIDI2DisableButton:      "MIDI 2.0 비활성화",
		MIDI2EnableButton:       "MIDI 2.0 활성화",
		MIDI2Disabling:          "비활성화 중...",
		MIDI2Enabling:           "활성화 중...",
		StemsEngineLabel:        "STEM 분리 엔진",
		LogAnalysisTitle:        "로그 분석",
		LogAnalyzeButton:       "로그 분석",
		LogAnalyzing:            "분석 중...",
		LogOpenDir:              "로그 폴더 열기",
		LogTotalFiles:           "로그 파일 수",
		LogTotalLines:           "총 줄 수",
		LogInfoCount:            "정보",
		LogWarningCount:         "경고",
		LogErrorCount:           "오류",
		LogTopWarnings:          "빈번한 경고",
		LogTopErrors:            "빈번한 오류",
		LogNoFiles:              "로그 파일을 찾을 수 없습니다",
		CacheCleanTitle:         "캐시 정리",
		CacheCleanDescription:   "Engine DJ UI 캐시를 삭제하여 화면 표시 문제를 해결",
		CacheCleanButton:        "캐시 삭제",
		CacheCleaning:           "삭제 중...",
		CacheCleanComplete:      "캐시가 삭제되었습니다",
		UpdateBadge:             "업데이트",
		UpdateChecking:          "확인 중...",
		UpdateTooltip:           "새 버전을 사용할 수 있습니다",
		OperationSuccess:        "작업 완료",
	},
	EN: {
		AppTitle:              "Engine Tools",
		InstallPathLabel:      "Install Path",
		InstallPathNotFound:   "Engine DJ install path not found",
		EngineVersionLabel:     "Engine DJ Version",
		WindowsVersionLabel:    "Windows Version",
		UTF8StatusLabel:       "System UTF-8 Support",
		UTF8Enabled:           "Enabled",
		UTF8Disabled:          "Disabled",
		ManifestStatusLabel:   "External Manifest",
		ManifestExists:        "Configured",
		ManifestNotExists:     "Not Configured",
		FixButton:             "Fix Unicode Character Encoding Issues (CJK / Greek / Latin Extended…)",
		RestoreButton:         "Restore Fix",
		OpenRegionSettings:    "Open Region Settings",
		UTF8AlreadyEnabled:   "System UTF-8 support is already enabled",
		UTF8AlreadyEnabledTip: "System UTF-8 support is already enabled. It is recommended to go to Control Panel → Region → Administrative → Change system locale and disable 'Beta: Use Unicode UTF-8 for worldwide language support', then use this tool to enable it at the application level.",
		ProcessRunningTitle:    "Running Programs Detected",
		ProcessRunningMessage:  "The following programs are running and need to be closed to continue:\n\n%s\n\nClose these programs?",
		KillingProcesses:       "Terminating processes...",
		ProcessKilled:         "Processes terminated",
		NoProcessRunning:      "No running programs detected",
		WritingManifest:       "Writing manifest file...",
		ManifestWritten:       "Manifest file written successfully",
		ManifestWriteError:    "Failed to write manifest file",
		ExeNotFound:           "Engine DJ.exe not found",
		WritingRegistry:       "Writing registry...",
		RegistryWritten:       "Registry written successfully",
		RegistryWriteError:    "Failed to write registry",
		DeletingManifests:       "Deleting manifest files...",
		ManifestsDeleted:        "Manifest files deleted",
		DeletingRegistry:        "Deleting registry entries...",
		RegistryDeleted:         "Registry entries deleted",
		RefreshingSystem:      "Refreshing system settings...",
		SystemRefreshed:        "System settings refreshed",
		FixComplete:          "Fix Complete",
		FixCompleteTip:        "Fix completed! If you still experience character display issues, please restart your computer.",
		RestoreComplete:         "Restore Complete",
		RestoreCompleteTip:      "Fix has been restored! Click the fix button again to re-apply.",
		FixFailed:            "Fix Failed",
		LogPrefix:            "[%s] %s",
		Checking:             "Checking",
		StatusChecking:       "Checking system status...",
		ProgressDetecting:    "Detecting...",
		ProgressFixing:       "Fixing...",
		ProgressRestoring:       "Restoring...",
		ProgressDone:         "Done",
		Language:             "Language",
		ACPCodePage:          "ACP Code Page",
		NotInstalled:         "Not Installed",
		Version:              "Version",
		MarqueeFree:           "This program is distributed free of charge. If you paid for it, please request a refund",
		MarqueeStar:           "If you like this project, please give it a Star on GitHub",
		AdminStatusLabel:     "Admin Privileges",
		AdminYes:            "Granted",
		AdminNo:             "Not Granted",
		StemsStatusLabel:    "STEM Separation Engine",
		StemsDetected:       "Detected",
		StemsNotFound:       "Not Detected",
		RestoreConfirmTitle:     "Confirm Restore",
		RestoreConfirmMessage:   "Are you sure you want to restore the fix? All changes will be reverted.",
		BackupReminderTitle:     "Backup Reminder",
		BackupReminderMessage:   "It is recommended to backup your database before proceeding. Continue?",
		TabStatus:               "Status",
		TabDatabase:             "Database",
		TabTools:                "Tools",
		DBLibraryPathLabel:      "Database Path",
		DBLibraryNotFound:       "Engine Library not found",
		DBBackupButton:          "Backup",
		DBBackupNoteLabel:       "Note",
		DBBackupNotePlaceholder: "Optional: enter a note",
		DBBackingUp:             "Backing up...",
		DBBackupComplete:        "Backup Complete",
		DBBackupError:           "Backup Failed",
		DBRestoreButton:         "Restore",
		DBRestoreSelectDate:     "Select date to restore",
		DBSelectDriveLabel:       "Select Drive",
		DBSelectDrivePlaceholder: "Choose a drive to scan for the database",
		DBSelectDriveConfirm:     "OK",
		DBRestoreConfirmTitle:   "Confirm Database Restore",
		DBRestoreConfirmMessage: "Restore to the selected backup date? Current data will be overwritten.",
		DBRestoring:             "Restoring...",
		DBRestoreComplete:       "Database Restore Complete",
		DBOptimizeButton:        "Optimize",
		DBOptimizing:            "Optimizing...",
		DBOptimizeComplete:      "Optimize Complete",
		DBRepairButton:          "Repair",
		DBRepairing:             "Repairing...",
		DBRepairComplete:        "Repair Complete",
		DBRepairStart:           "Repairing database...",
		DBRepairDone:            "Database repair complete",
		DBNoteLabel:             "Note",
		DBNoneFound:             "Not Found",
		DBNoBackups:             "No Backups",
		MSICleanupButton:        "MSI Cleanup",
		MSICleanupTitle:         "MSI Cleanup",
		MSICleanupDescription:   "Scan and remove orphaned MSI installation residuals",
		MSIScanning:             "Scanning...",
		MSIFoundOrphans:         "%d orphans found",
		MSINoOrphans:            "No MSI orphans found",
		MSICleaning:             "Cleaning...",
		MSICleanComplete:        "Cleanup Complete",
		MSICleanError:           "Cleanup Failed",
		MSIConfirmTitle:         "Confirm Cleanup",
		MSIConfirmMessage:       "Delete selected MSI residuals? This operation cannot be undone.",
		ID3EditorTitle:          "ID3 Tag Editor",
		ID3SelectFile:           "Select Audio File",
		ID3PickFileButton:       "Select File",
		ID3SaveButton:           "Save",
		ID3ClearAllButton:       "Clear All",
		USBUnlockTitle:          "USB Unlock",
		USBUnlockDescription:    "For Engine DJ install/uninstall/update file lock issues",
		USBUnlockButton:         "Unlock",
		USBScanButton:           "Scan blocking processes",
		USBNoDevice:             "No USB with Engine Library detected",
		MIDI2Title:              "MIDI 2.0 Control",
		MIDI2Description:        "Disable Windows 11 MIDI 2.0 features while preserving MIDI 1.0 service",
		MIDI2Enabled:            "MIDI 2.0 is currently enabled",
		MIDI2Disabled:           "MIDI 2.0 is currently disabled",
		MIDI2Unavailable:        "MIDI 2.0 services not found on this system",
		MIDI2DisableButton:      "Disable MIDI 2.0",
		MIDI2EnableButton:       "Enable MIDI 2.0",
		MIDI2Disabling:          "Disabling...",
		MIDI2Enabling:           "Enabling...",
		StemsEngineLabel:        "STEM Separation Engine",
		LogAnalysisTitle:        "Log Analysis",
		LogAnalyzeButton:       "Analyze Logs",
		LogAnalyzing:            "Analyzing...",
		LogOpenDir:              "Open Logs Folder",
		LogTotalFiles:           "Log Files",
		LogTotalLines:           "Total Lines",
		LogInfoCount:            "Info",
		LogWarningCount:         "Warnings",
		LogErrorCount:           "Errors",
		LogTopWarnings:          "Top Warnings",
		LogTopErrors:            "Top Errors",
		LogNoFiles:              "No log files found",
		CacheCleanTitle:         "Cache Cleanup",
		CacheCleanDescription:   "Clear Engine DJ UI cache to fix display glitches after updates",
		CacheCleanButton:        "Clear Cache",
		CacheCleaning:           "Clearing...",
		CacheCleanComplete:      "Cache cleared",
		UpdateBadge:             "Update",
		UpdateChecking:          "Checking...",
		UpdateTooltip:           "New version available",
		OperationSuccess:        "Operation complete",
	},
}

func DetectLang() Lang {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getUserDefaultUILanguage := kernel32.NewProc("GetUserDefaultUILanguage")

	if getUserDefaultUILanguage.Find() == nil {
		langID, _, _ := getUserDefaultUILanguage.Call()
		primaryLang := uint16(langID) & 0xFF

		switch primaryLang {
		case 0x04:
			return ZH
		case 0x11:
			return JA
		case 0x12:
			return KO
		}
	}

	lang := strings.ToLower(os.Getenv("LANG"))
	if lang == "" {
		lang = strings.ToLower(os.Getenv("LANGUAGE"))
	}

	if strings.HasPrefix(lang, "zh") {
		return ZH
	}
	if strings.HasPrefix(lang, "ja") {
		return JA
	}
	if strings.HasPrefix(lang, "ko") {
		return KO
	}

	for _, envVar := range os.Environ() {
		if strings.HasPrefix(envVar, "LC_") || strings.HasPrefix(envVar, "LANG=") {
			lower := strings.ToLower(envVar)
			if strings.Contains(lower, "zh") {
				return ZH
			}
			if strings.Contains(lower, "ja") {
				return JA
			}
			if strings.Contains(lower, "ko") {
				return KO
			}
		}
	}

	return EN
}

func Get(lang Lang) Messages {
	if m, ok := translations[lang]; ok {
		return m
	}
	return translations[EN]
}

func AvailableLangs() []Lang {
	return []Lang{ZH, JA, KO, EN}
}

func LangDisplayName(lang Lang) string {
	switch lang {
	case ZH:
		return "中文"
	case JA:
		return "日本語"
	case KO:
		return "한국어"
	case EN:
		return "English"
	default:
		return string(lang)
	}
}