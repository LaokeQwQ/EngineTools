# EngineTools 中心服务、设备与备份设计

## 1. 目标

建立纯 Go 中心服务，接入现有 PostgreSQL、Redis、OIDC 和现场 S3，为设备目录、云备份、身份组策略及后续同步提供控制面。

同步仍由用户手动发起。唯一自动上传例外是用户显式开启的每周完整云备份。

## 2. 服务组成

中心部署包含：

- Go REST API：认证、设备、策略、备份和任务元数据。
- Go WebSocket 信令：设备心跳、在线路由、任务事件和 ICE 信令。
- Go STUN 服务：自建 UDP 3478 为首选；公共 STUN 列表由部署配置提供备用。
- PostgreSQL：所有持久业务状态和审计记录。
- Redis：在线 TTL、限流计数、跨实例事件分发以及锁/幂等查询的非权威缓存。
- S3：云备份内容对象和 P2P 失败时的临时中转对象。

不部署 TURN。P2P 不可用时使用 S3 fallback。

中心服务放在独立 `server/` Go module 中，避免 Windows-only 桌面依赖进入 Linux 服务构建。服务入口、迁移、后台任务和协议定义分别组织；客户端/服务端契约由版本化 OpenAPI/JSON Schema 生成或校验，不复制手写两套 DTO。Redis 中的锁和幂等键仅作性能优化，不能承担最终正确性。

## 3. OIDC 与设备会话

桌面端使用系统浏览器执行 OIDC Authorization Code with PKCE：

1. 客户端先在 `127.0.0.1`/`[::1]` 绑定一次性临时监听器，再生成 `state`、`nonce` 和 PKCE S256 verifier/challenge；redirect URI 必须与实际监听地址精确一致。
2. 系统浏览器打开由 OIDC discovery 得到的授权地址。
3. 客户端只接受一次回调，校验 `state` 后交换授权码；ID token 必须校验 HTTPS issuer allowlist、签名、`aud/azp`、nonce、`exp/iat`。
4. 客户端使用专为 EngineTools API 签发、包含所需 audience 和 scope 的 access token 调用设备注册挑战端点。ID token 不得作为 API bearer token。
5. 中心服务校验 access token 的 issuer、签名、audience、scope 和时间，通过 `issuer + subject` 映射用户并返回一次性设备挑战。
6. 客户端以设备私钥签名挑战后才完成公钥绑定并取得 EngineTools 设备会话。

设备首次登录生成随机 UUID 和 Ed25519 设备密钥对。私钥及 refresh credential 保存到 Windows Credential Manager；服务端只保存公钥、key thumbprint 和 refresh credential 哈希，不使用 `MachineGuid` 作为身份。API access token 通过 DPoP 或等价请求签名绑定设备 key，refresh credential 单次轮换并支持 token family 重放检测。重装生成新设备 ID/密钥；旧设备必须从设备列表撤销，不能恢复或复制私钥冒充原安装。

用户可重命名、轮换密钥和撤销设备。安全敏感 API 每次检查 session/device 状态；access token 包含 session ID。撤销时通过 Redis 广播关闭 WebSocket、拒绝任务 finalize、终止 multipart 并删除 presence。已经签发的极短时 S3 URL 只能等待 TTL 到期，已经建立且不合作的直接 P2P 也无法保证瞬间停止；UI 和审计必须准确表达这一限制。

## 4. PostgreSQL 模型

核心表：

- `users(id, oidc_issuer, oidc_subject, created_at, disabled_at)`，唯一键为 issuer/subject。
- `devices(id, user_id, display_name, public_key, key_thumbprint, app_version, os_version, last_seen_at, revoked_at, created_at, updated_at)`。
- `device_sessions(id, user_id, device_id, token_family_id, refresh_hash, key_thumbprint, expires_at, used_at, revoked_at, rotated_from_id, created_at)`。
- `identity_group_policies(id, group_name, priority, version, policy_json, enabled_at, disabled_at)`。
- `backup_schedules(id, user_id, device_id, enabled, last_attempt_at, last_success_at, next_due_at, state_version, updated_at)`；自动周备份固定包含歌曲，是否可启用由策略决定。
- `backup_jobs(id, user_id, device_id, destination, state, state_version, policy_id, policy_version, idempotency_key, created_at, updated_at, completed_at, error_code)`。
- `libraries(id, user_id, device_id, local_library_id, created_at, updated_at)`，账号/设备/local ID 唯一。
- `library_revisions(id, user_id, library_id, canonical_hash, created_by_job_id, created_at)`。
- `revision_parents(user_id, revision_id, parent_revision_id, parent_role)`；普通合并同时记录 source 和 target 双亲。
- `snapshots(id, user_id, device_id, library_id, revision_id, kind, state, state_version, format_version, schema_fingerprint, adapter_version, manifest_hash, source_database_hash, derived_database_hash, path_mapping_hash, captured_at, updated_at, expires_at)`；snapshot 是 revision 的一次捕获，不兼任 revision 本身。
- `backups(id, user_id, snapshot_id, label, automatic, include_media, storage_bytes, verified_at, deleted_at)`。
- `snapshot_objects(user_id, snapshot_id, object_id, logical_path, size_bytes, content_type)`，账号/snapshot/logical path 唯一。
- `objects(id, user_id, sha256, size_bytes, s3_key, state, state_version, verified_at, updated_at, deleted_at)`，账号内唯一键为 user/sha256。
- `upload_sessions(id, user_id, job_id, object_id, s3_upload_id, state, state_version, reserved_bytes, expires_at, updated_at)`。
- `upload_parts(user_id, upload_session_id, part_number, size_bytes, checksum_sha256, etag, completed_at)`。
- `sync_jobs`、确认令牌、信令租约和临时传输对象由同步迁移加入，其契约遵循 P2P 同步规格。
- `audit_events(id, user_id, device_id, task_type, task_id, event_code, metadata_json, occurred_at)`。

所有外部 ID 使用 UUID。时间统一存 UTC。关键状态转换使用 `state_version` 乐观锁，避免重复回调覆盖终态。幂等键具有 PostgreSQL 唯一约束；目标写任务互斥使用数据库事务和 advisory/row lock，Redis 清空或过期不能产生重复任务或双写。

所有租户关系使用包含 `user_id` 的复合唯一键与复合外键，保证 device、session、job、snapshot、backup 和 object 永远属于同一账号；生产库同时启用 PostgreSQL RLS 作为纵深防御。任何对象、快照或设备查询都必须以认证 `user_id + id` 联合查找，签名 URL 不能只凭全局 UUID 签发。

## 5. 身份组策略

组成员关系只读取 OIDC token 中部署可配置的 claim 路径，例如 `groups` 或嵌套 roles。客户端提交的组名不可信。服务端配置 claim 最大陈旧时间；到期后必须向 OIDC 重新获取有效 token/claims，长期 EngineTools refresh session 不能无限沿用旧组。账号/设备禁用和全局 kill switch 即时生效，只有配额等非安全参数固定到已创建任务的策略版本。

`identity_group_policies.policy_json` 支持：

- 云备份最大保留数量。
- 账号云存储总容量。
- 单次上传和单对象大小上限。
- 最大设备数量。
- 是否允许自动周备份。
- 自动备份是否允许包含歌曲。
- 是否允许 P2P、公共 STUN 和强制同步。

用户属于多个组时按显式 `priority` 选择一条策略；同优先级配置被后台拒绝，避免不确定结果。不自动取最宽松策略。未匹配时使用受限 `default` 策略。

任务创建时记录生效策略版本。任务执行中策略变更不导致半途损坏；新任务使用新策略。后台配额收敛只在新备份验证完成或周期检查时执行，并保留审计事件。

## 6. 备份清单与本地格式

备份 manifest 是版本化 JSON，至少包含：

- 格式版本、快照 ID、设备 ID 和创建时间。
- Engine DJ 版本、SQLite schema 指纹、adapter 版本、revision 和零到多个父 revision。
- 备份范围和是否包含歌曲。
- 每个条目的逻辑路径、类型、大小、SHA-256 和内容类型。
- 不可变 `source_database_hash`、可选派生 `derived_database_hash`、path-mapping hash、资产统计、缺失/跳过文件列表。

本地 `.etbackup` 使用 ZIP64 容器：`manifest.json`、一致性数据库快照和 `objects/<sha256>`。数据库与 JSON 可压缩，已压缩音频默认使用 store 模式。恢复端不信任 ZIP 路径，只按 manifest 中经过规范化的逻辑路径读取，并限制条目数量、展开大小和压缩比。

S3 使用相同 manifest 语义，但对象按 `accounts/<user-id>/objects/<sha256>` 账号隔离存放，不生成单一大归档。相同账号中内容相同的歌曲只存一份；不得跨账号查询或复用哈希对象。

## 7. 一致性快照与内容范围

SQLite 数据库通过 go-sqlite3 Backup API 生成在线一致性快照，随后执行完整性检查和关键表读取。歌曲和资产清单必须从该不可变数据库副本枚举，不能在快照前从活跃库预先枚举。

`schema_fingerprint` 使用版本化算法，输入包括 SQLite `application_id`、`user_version`，以及规范化排序后的表、列、类型、约束、索引和触发器定义；忽略无语义空白和 SQLite 自动生成名称。算法版本写入 manifest，并用不同 SQLite 版本产生的等价 schema 做确定性测试。

完整备份包含：

- 数据库快照。
- 已知可移植的波形、分析和封面资产。
- 手动任务中由用户选择的歌曲。
- 自动周备份中数据库引用且实际可读的全部歌曲。

缓存、日志、锁文件、临时文件和未知内部文件不进入默认完整备份。每个受支持的 Engine DJ/schema adapter 必须列出必需、可选和禁止的资产路径，并负责路径字段重写；未知 schema 禁止执行带路径恢复。文件枚举必须限制在允许的曲库/媒体来源，解析重解析点后仍需位于允许根内，防止损坏数据库诱导上传任意本地文件。歌曲哈希期间检测文件 ID、大小和修改时间变化；发生变化则重试不可变暂存副本或把该文件标记为不一致，不能上传前后内容混合的对象。

缺失或不可读歌曲记录为不完整结果。自动任务不得把带缺失文件的包标记为完全成功；它可以保留为 `completed_with_warnings`，并在客户端明确展示。

## 8. 路径与托管音乐目录

manifest 不保存原始盘符、用户名或绝对路径。客户端将每个批准的来源根映射为随机不透明 `source_root_id`，逻辑路径只保存 `source_root_id + approved-root-relative-path`。规范化规则统一处理 Windows 大小写、Unicode、保留名、ADS、尾随点/空格和分隔符；落盘后再次通过句柄最终路径验证没有越出托管根。

- Engine Library 和外部媒体都保留相对于各自批准根目录的层级，但不暴露根目录原名。

恢复或带歌曲同步时，所有歌曲进入目标设备可配置的 EngineTools 托管音乐根目录，默认位于当前用户音乐目录下。数据库路径在暂存事务中重写为目标路径。

目标已有相同 SHA-256 时复用文件；同名不同内容增加稳定哈希后缀，绝不静默覆盖。

## 9. 自动周备份

自动周备份必须由用户显式开启，调度事实保存于服务端 `backup_schedules`。应用启动、持续运行跨越到期点以及系统睡眠恢复后都重新计算调度：

- 未满 7 天不执行。
- 满 7 天且 Engine DJ 未写库时开始逻辑全量备份。
- Engine DJ 正在运行或曲库被写锁占用时延后，不自动终止进程。
- 失败不更新时间，下次启动重试，并支持断点续传。
- 自动任务固定包含全部可读引用歌曲；若身份组不允许带歌曲的自动备份，则 UI 不允许启用该计划。

客户端需要有效设备会话才能补做。`completed_with_warnings` 快照可以保留，但不更新 `last_success_at`；它更新 `last_attempt_at`，并在部署配置的退避时间后只重试缺失/变化对象，避免每次启动重复上传整库。

每份快照逻辑上可独立恢复，物理对象按 SHA-256 去重。新备份完成 manifest、对象和数据库验证后，服务端才按身份组保留数量轮换最旧云备份。手动与自动云备份共同计数；本地备份不受云端数量策略影响。

`snapshot_objects` 是引用关系的事实源，不依赖可漂移的 `ref_count`。备份按 `intent -> uploading -> verifying -> committed` 两阶段可见性协议推进，失败进入 `aborted`。草稿快照验证完成后，在一个 PostgreSQL 事务中转为 ready、建立全部引用并让 backup committed。对象状态为 `uploading/available/gc_pending/deleted`。GC 锁定候选并标记 `gc_pending`；宽限期后再次确认没有引用和恢复 lease，删除 S3 对象后再落库终态。失败任务的临时对象由生命周期规则和补偿任务共同清理。

## 10. 上传与 S3 安全

- 客户端不能指定任意 S3 key，只能请求服务端为已登记对象签发短时 URL。
- 创建上传时在 PostgreSQL 事务中预留账号配额。单 PUT URL 签入 method、content length、S3 checksum 和有效期。
- 大对象使用 multipart upload，服务端持久化 upload ID 和 part；完成时使用 S3 支持的 SHA-256 checksum 与 `HeadObject` 校验总大小，绝不把 multipart ETag 当内容哈希。失败、撤销或过期必须调用 `AbortMultipartUpload` 并释放预留配额。
- 传输使用 TLS，静态加密使用部署配置的 SSE-S3 或 SSE-KMS。
- 临时同步对象与长期备份对象使用不同前缀和生命周期策略。
- 日志不得记录签名 URL、token、完整本地路径或对象正文。

本产品选择信任中心服务和 S3 管理员可在授权运维场景读取明文内容；TLS 与 SSE 不是端到端加密。若未来要求零知识备份，必须另行设计账号主密钥恢复、设备密钥分发以及加密后去重，不能在本协议中隐式声称端到端隐私。

## 11. 恢复事务

恢复步骤：

1. 下载 manifest 和所需对象到用户专属暂存目录。
2. 校验格式版本、策略权限、条目限制、全部哈希和数据库完整性。
3. 计算所需磁盘空间并验证目标目录。
4. 创建当前数据库和受影响资产的本地回滚包。
5. 写入托管歌曲/资产暂存区并重写数据库路径。
6. Engine DJ 关闭后提交文件，数据库最后原子替换。
7. 重新打开数据库并执行完整性与关键表检查。
8. 先验证不可变源数据库 hash；路径重写后记录 adapter 版本、path-mapping hash 和派生数据库 hash，再执行完整性、外键和 adapter 查询。
9. 失败或启动恢复时按 fsync 操作日志继续回滚；成功后保留可撤销回滚包到配置的本地期限。

跨目录、多卷、多文件和断电场景无法形成单一文件系统原子事务。每个目标卷建立 sibling staging，分别计算空间并写入同一持久操作日志；数据库最后提交。Windows 实现明确使用 `ReplaceFileW` 或等价 write-through 原语，提交前关闭全部 SQLite handle 并处理旧 `m.db-wal`/`m.db-shm`。NTFS、exFAT 和 FAT32 分别做能力探测及故障测试，不支持所需 replace/flush 语义的卷阻止应用。恢复保证的是故障可检测和可继续回滚，不承诺任意时刻所有文件瞬间保持不变。

## 12. API 与错误模型

客户端 API 至少包含：

- `POST /v1/device-sessions`
- `POST /v1/device-registration-challenges`
- `GET/PATCH/DELETE /v1/devices`
- `GET /v1/policy`
- `POST/GET/DELETE /v1/backups`
- `POST /v1/backups/{id}/restore-plan`
- `POST /v1/uploads` 与 multipart 完成/中止端点
- `GET /v1/events` WebSocket 升级
- 同步任务、确认和信令端点按 `2026-08-16-p2p-database-sync-design.md` 增加

错误响应使用稳定 code、用户安全 message key、request ID 和可重试标识；不把数据库或 S3 原始错误直接返回客户端。

## 13. 测试与验收

- 使用临时 PostgreSQL、Redis、MinIO 和测试 OIDC issuer 做集成测试。
- 覆盖 token 过期、issuer/audience 错误、设备撤销、refresh rotation 和重放。
- 覆盖 ID token 被错误用作 API token、DPoP/key proof 错误、设备重装和密钥轮换。
- 覆盖跨账号 device/snapshot/object UUID 的 IDOR、复合外键和 RLS。
- 覆盖多组优先级、default 策略、配额、轮换和策略版本固定。
- 覆盖 multipart 中断、重复完成、checksum 错误、幂等重试和垃圾回收。
- 覆盖恶意 ZIP 路径、压缩炸弹、重解析点、越界绝对路径和重复 manifest 条目。
- 覆盖自动备份到期、未到期、Engine DJ 运行、失败重试和 `completed_with_warnings`。
- 覆盖应用持续运行超过七天、系统睡眠恢复和永久缺失文件的退避，避免每次启动全量重传。
- 覆盖 source/derived database hash、adapter/path-mapping hash 和双亲 revision。
- 覆盖每次日志 flush、rename、DB replace 前后崩溃，以及实际支持的 NTFS/USB 文件系统恢复。
- 恢复测试必须证明数据库提交前失败保持旧数据库，数据库提交后故障可被检测并完成回滚；成功后数据库和全部引用对象哈希正确。

## 14. 完成标准

用户可通过 OIDC 登录并管理设备；身份组策略由服务端强制执行；本地和 S3 备份使用同一可验证清单；自动周备份按启动补做规则运行；恢复故障可被检测并继续回滚到可打开的原曲库。

## 15. 相关规格与顺序

本规格依赖桌面安全基础规格；它提供 P2P 同步规格所需的认证、设备、策略、信令和 S3 控制面，但完成本规格不表示已经交付数据库同步或 ID3 编辑器。
