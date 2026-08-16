# EngineTools 桌面安全基础与 MIDI 2.0 移除设计

## 1. 目标

本规格建立后续账号、云备份、设备同步和 ID3 编辑所需的桌面安全边界，并完整移除 MIDI 2.0 功能。

当前应用在启动时无条件请求管理员权限，Wails WebView、网络请求、文件解析和系统修改都运行在同一高权限进程中。加入 OIDC、远端数据库和音频文件后，这一结构会放大任何解析漏洞的影响，因此必须先拆分权限。

## 2. 范围

本阶段包含：

- 主 Wails 应用改为普通用户权限运行。
- 增加按需启动的一次性提权 Helper。
- 将现有大体量 `App` 状态拆到明确的本地服务边界。
- Wails API 改用结构化结果和 Go `error`。
- 为并发状态、任务进度和取消建立统一模型。
- 删除 MIDI 2.0 后端、前端、文案和生成绑定。
- 为现有本地数据库备份/恢复建立原子性与错误传播基线。

本阶段不实现中心服务、P2P、跨设备数据库合并或多格式 ID3；这些由后续规格负责。

## 3. 进程与权限模型

### 3.1 普通权限主进程

`EngineTools.exe` 默认以当前用户权限运行，负责：

- Wails UI 与本地应用状态。
- OIDC 登录和 Windows Credential Manager 会话读取。
- 设备列表、备份、同步和传输。
- SQLite 快照、验证、暂存和普通用户可访问文件操作。
- ID3/FLAC/WAV/AIFF 元数据解析和写入。

主进程不得因为单个系统功能而在启动时整体提权。

### 3.2 提权 Helper

新增 `EngineTools.Helper.exe`，仅暴露枚举后的系统操作：

- 写入或删除 Engine DJ 外部 manifest。
- 写入或删除 `PreferExternalManifest` 注册表值。
- 终止明确属于 Engine DJ 安装目录的进程。
- 执行需要管理员权限的 MSI 清理和 USB 句柄操作。

Helper 不提供任意命令执行、任意注册表路径或任意文件复制接口。MSI 清理和 USB 句柄操作只接收产品/卷的稳定标识，Helper 必须重新枚举并验证合法目标，不接受主进程传入的任意 PID、句柄、文件或注册表项。终止进程时，Helper 在同一个进程句柄上校验映像最终路径和进程创建时间，避免 PID 复用及检查/执行竞态。

主进程按需通过 `ShellExecuteW("runas")` 启动 Helper。双方使用带当前用户 SID ACL 的本地命名管道通信；每次调用通过 `crypto/rand` 生成 nonce，并绑定协议版本、双方 PID、进程创建时间、action、参数摘要和短时有效期。

双方通过 `GetNamedPipeClientProcessId`/`GetNamedPipeServerProcessId` 获取实际对端 PID，再以进程句柄校验最终映像路径、用户 SID、登录会话、创建时间和 Authenticode 发布者。生产构建的主程序与 Helper 必须安装在普通用户不可写目录，并由同一发布者签名。请求必须匹配固定 action schema；文件与目录验证基于已打开句柄的最终路径完成，不能采用先检查字符串、后重新打开路径的方式，并拒绝重解析点逃逸。

Helper 完成单次请求后退出，设有超时，不保存登录令牌，不连接中心服务，不解析远端文件。

## 4. 桌面模块边界

主进程按职责拆分为：

- `AuthService`：OIDC 会话与 Credential Manager。
- `DeviceService`：本机设备身份、在线状态和远端设备目录。
- `LibraryService`：显式 `LibraryRef`、曲库发现、锁和路径策略。
- `SnapshotService`：SQLite 一致性快照、manifest、验证和恢复。
- `BackupService`：本地/S3 备份编排。
- `SyncService`：同步任务状态机，不与 BPM/Key 写回共用名称。
- `MetadataService`：音频元数据读取、编辑和批量任务。
- `PrivilegeClient`：调用提权 Helper。

Wails 绑定层只做 DTO 转换和服务调用，不承载业务状态机。

前端将当前单文件页面拆成按功能加载的视图和 composable：状态、设备、曲库、备份、同步、工具、元数据编辑器。共享任务状态统一以 `operationId` 标识，避免不同任务共用全局进度。

## 5. 结构化结果与并发

所有可失败 Wails 方法在内部返回 `(Result, error)`。Wails 边界把内部错误转换为稳定错误码、message key、operation/request ID 和可重试标识；SQL、路径和网络原始错误只进入脱敏本地日志。前端 Promise 只有在持久化成功时才进入成功状态，不再把空字符串或错误字符串视为正常返回值。

任务事件至少包含：

- `operationId`
- `phase`
- `completedBytes`
- `totalBytes`
- `messageCode`
- `cancellable`

`App` 中的日志、检测状态和任务集合必须由互斥锁或单一事件循环保护。每个曲库具有独立读写锁；应用数据库或恢复时取得写锁，统计和预览只取得读锁。

## 6. 本地备份安全基线

在云能力落地前，现有裸文件备份先满足以下约束：

- 使用 SQLite Backup API 生成一致性快照，不直接复制活跃 `m.db`。
- 快照后执行 `PRAGMA integrity_check` 和关键表读取检查。
- 恢复文件名必须解析为备份目录内的已登记 ID，禁止路径穿越。
- 恢复先写暂存文件，校验后使用 Windows 原子替换能力提交。
- 提交前保存本地回滚副本，失败时自动恢复。
- 备份列表按结构化创建时间倒序，不从文件名猜测全部元数据。

## 7. MIDI 2.0 移除

删除范围：

- 删除 `internal/midi/midi.go` 和 `internal/midi` 包。
- 删除 `app.go` 中的 MIDI import、`MIDI2Status`、`MIDI2Disable`、`MIDI2Enable`。
- 删除 `frontend/src/App.vue` 中的绑定 import、状态、处理函数、加载调用和 MIDI UI。
- 删除 `internal/i18n/i18n.go` 中四种语言的 MIDI 字段与文案。
- 重新生成 Wails 绑定，删除 `App.js`、`App.d.ts` 和 `models.ts` 中的 MIDI 表面。

`golang.org/x/sys` 仍被其他 Windows 模块使用，不因 MIDI 删除而移除。

旧版本可能已经修改 MIDI 服务的 `Start` 值，但未保存原值。升级时不得猜测并写回 `3`；移除功能只表示 EngineTools 停止管理该状态。

## 8. 安全与日志

- Credential Manager 中的 OIDC/设备凭据只由普通权限主进程读取。
- 日志不得包含 token、授权码、签名 URL、歌曲绝对路径或原始数据库内容。
- Helper 日志只记录 action、结果码和脱敏目标标识。
- 主进程校验所有从 WebView 进入的路径、枚举值和数值范围。
- 任意远端内容在验证 manifest、大小、哈希和格式前不得传给 Helper。

## 9. 测试与验收

- 普通启动不出现 UAC；只有受保护操作触发一次性 UAC。
- Helper 拒绝未知 action、越界路径、过期 nonce、错误调用方和重解析点逃逸。
- 并发状态通过 `go test -race` 可验证部分的竞态检查。
- 本地备份覆盖 WAL 活跃、目标锁定、磁盘不足、校验失败和自动回滚。
- 前端验证后端失败时不显示成功，不清空当前状态。
- 重新生成绑定后执行 `rg "MIDI2|midi2|Midi2\\.|internal/midi" app.go internal frontend/src frontend/wailsjs` 无结果；规格文档不属于检索目标。
- Windows Go 测试/构建、前端测试/构建及 Wails 绑定一致性全部通过。

## 10. 完成标准

主应用可在普通权限下完成所有非系统功能；提权操作只能通过固定 Helper action；现有本地备份不会直接覆盖活跃数据库；MIDI 2.0 相关代码和 UI 完全移除。

## 11. 相关规格与顺序

本规格必须先于以下规格实施：

- `2026-08-16-central-service-device-backup-design.md`
- `2026-08-16-p2p-database-sync-design.md`
- `2026-08-16-id3-editor-redesign-design.md`
