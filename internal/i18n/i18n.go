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
	RefreshingSystem         string `json:"refreshingSystem"`
	SystemRefreshed          string `json:"systemRefreshed"`
	FixComplete             string `json:"fixComplete"`
	FixCompleteTip          string `json:"fixCompleteTip"`
	FixFailed               string `json:"fixFailed"`
	LogPrefix               string `json:"logPrefix"`
	Checking               string `json:"checking"`
	StatusChecking          string `json:"statusChecking"`
	ProgressDetecting       string `json:"progressDetecting"`
	ProgressFixing          string `json:"progressFixing"`
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
		FixButton:             "修复中日韩等特殊字符读取问题",
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
		RefreshingSystem:      "正在刷新系统设置...",
		SystemRefreshed:       "系统设置已刷新",
		FixComplete:          "修复完成",
		FixCompleteTip:        "修复已完成！如果仍遇到字符显示问题，请重启电脑后重试。",
		FixFailed:            "修复失败",
		LogPrefix:            "[%s] %s",
		Checking:             "正在检测",
		StatusChecking:       "正在检测系统状态...",
		ProgressDetecting:    "检测中...",
		ProgressFixing:       "修复中...",
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
		FixButton:             "中日韓などの特殊文字の読み取り問題を修正",
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
		RefreshingSystem:      "システム設定を更新中...",
		SystemRefreshed:       "システム設定を更新しました",
		FixComplete:          "修正完了",
		FixCompleteTip:        "修正が完了しました！文字の表示問題が続く場合は、コンピュータを再起動してください。",
		FixFailed:            "修正に失敗しました",
		LogPrefix:            "[%s] %s",
		Checking:             "確認中",
		StatusChecking:       "システム状態を確認中...",
		ProgressDetecting:    "確認中...",
		ProgressFixing:       "修正中...",
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
		FixButton:            "중일한 등 특수 문자 읽기 문제 수정",
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
		RefreshingSystem:      "시스템 설정 새로고침 중...",
		SystemRefreshed:        "시스템 설정이 새로고침되었습니다",
		FixComplete:          "수정 완료",
		FixCompleteTip:        "수정이 완료되었습니다! 문자 표시 문제가 계속되면 컴퓨터를 재시작하세요.",
		FixFailed:            "수정 실패",
		LogPrefix:            "[%s] %s",
		Checking:             "확인 중",
		StatusChecking:       "시스템 상태 확인 중...",
		ProgressDetecting:    "확인 중...",
		ProgressFixing:       "수정 중...",
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
		FixButton:             "Fix CJK Character Reading Issues",
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
		RefreshingSystem:      "Refreshing system settings...",
		SystemRefreshed:        "System settings refreshed",
		FixComplete:          "Fix Complete",
		FixCompleteTip:        "Fix completed! If you still experience character display issues, please restart your computer.",
		FixFailed:            "Fix Failed",
		LogPrefix:            "[%s] %s",
		Checking:             "Checking",
		StatusChecking:       "Checking system status...",
		ProgressDetecting:    "Detecting...",
		ProgressFixing:       "Fixing...",
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