# EngineTools ID3/音频元数据编辑器重做设计

## 1. 目标

把当前仅适用于 MP3 ID3v2、但 UI 又接受 FLAC/WAV/AIFF 的实现，重做为按格式分派、保留未知元数据、支持撤销和批量编辑的安全元数据系统。

“ID3 编辑器”作为用户可见名称保留，内部统一称 `MetadataService`，避免把所有音频格式误当成 ID3。

## 2. 支持范围

首版正式支持：

- MP3：ID3v2.3、ID3v2.4，并识别 ID3v1。
- FLAC：Vorbis Comment 和 Picture metadata block。
- WAV：RIFF INFO；富字段使用标准 ID3 chunk；保留 BEXT、iXML、cue、smpl 等非目标 chunk。
- AIFF/AIFF-C：NAME、AUTH、ANNO；富字段使用 ID3 chunk；保留 COMM、SSND 和其他非目标 chunk。

M4A/MP4、OGG/Opus、APE 和 WMA 不在本阶段范围内。后端根据文件魔数和容器结构识别格式，扩展名只用于文件选择器提示。

## 3. Adapter 接口

每种格式实现统一接口：

- `Probe(path) -> FormatCapabilities`
- `Read(path) -> MetadataDocument`
- `Apply(path, MetadataPatch, WriteOptions) -> WriteResult`
- `ReadPictures(path) -> []Picture`
- `ReplacePicture(path, PicturePatch) -> WriteResult`
- `SnapshotTags(path) -> TagSnapshot`
- `RestoreTags(path, TagSnapshot) -> WriteResult`

adapter 必须明确支持的字段、封面类型、编码和版本，不支持的写入在预览阶段返回 capability error，不能静默丢弃。

adapter 优先封装经过维护、可处理对应容器的成熟库，不使用字节搜索或未验证的自制解析器。任何必须补充的容器写入代码都限制在独立包，并由 corpus、golden 和 fuzz 测试约束。

## 4. 元数据模型

`MetadataDocument` 包含：

- 标题、艺术家列表、专辑艺术家列表、专辑。
- 曲号/总曲数、碟号/总碟数。
- 日期/年份、流派列表、作曲者列表。
- 评论、BPM、调号、Rating、ISRC、歌词。
- 多个封面及其类型、MIME、尺寸和说明。
- 源格式、标签版本、只读能力、解析警告和未知字段摘要。

写入使用 `MetadataPatch`。每个字段都是三态：

- `unchanged`：保留原值。
- `set`：设置为给定值。
- `clear`：明确删除。

空字符串不再隐式表示“不修改”，从而修复清空输入框后旧标签仍保留的问题。

## 5. 格式字段映射

MP3 常用映射包括 TIT2、TPE1、TPE2、TALB、TRCK、TPOS、TDRC/TYER、TCON、TCOM、COMM、TBPM、TKEY、POPM、TSRC、USLT 和 APIC。已有标签版本默认保持；新建标签默认 v2.3，以提高 Engine DJ/Engine OS 兼容性。未知帧按原字节保留。

ID3v1-only 文件首次编辑时创建 v2.3，并同步更新原来已经存在的 ID3v1 可表达字段；默认不为原本没有 ID3v1 的文件新建尾标签。明确 clear 的字段同时从已有 ID3v1/ID3v2 删除。多个 POPM owner 等无法安全合并的帧由 adapter 保留原值并在 capability/预览中说明。

FLAC 使用规范化 Vorbis key，并保留未知 comment 和非目标 metadata block。封面使用 Picture block，替换 front cover 时不删除 back cover、artist 或其他图片。

WAV 和 AIFF 的核心字段优先更新原生文本 chunk；原生格式无法表达的富字段写入标准 ID3 chunk。adapter 不重排或重编码音频数据 chunk，不删除广播、采样点或设备私有 chunk。

## 6. 安全写入与不变量

所有修改遵循：

1. 读取并解析原文件，记录标签快照、文件属性和音频载荷哈希。
2. 在同目录创建随机临时文件，保持权限并流式复制/重写容器。
3. 同步临时文件到磁盘，重新解析并验证请求字段。
4. 验证音频载荷哈希未变化、未知字段和非目标图片仍存在。
5. 使用 Windows 原子替换提交，并保留可撤销快照。

文件被占用、只读、空间不足、格式损坏或验证失败时不得修改原文件。临时文件在安全失败后清理，诊断记录不包含完整媒体路径。

## 7. 封面处理

- 通过文件魔数和解码结果确认 MIME，不按扩展名猜测。
- 默认限制单张输入 20 MiB、5000 万像素和 256 MiB 解码预算；部署/客户端设置只能收紧或在明确高级设置中调整，超限在写入前拒绝。
- UI 只加载最长边不超过 1024 像素的受限缩略图，不把任意大原图完整 Base64 传入 WebView。
- 更换正面封面只替换 front cover；其他图片保持。
- 压缩只处理用户选择的图片类型，并在结果中列出跳过和失败原因。
- JPEG、PNG 为默认写入格式；其他输入在解码后按用户选择转换，不把 GIF/BMP/WebP MIME 与实际字节混用。

## 8. 单曲与批量编辑 UI

单曲编辑器展示格式、标签版本、文件只读状态和脏状态。切换文件、播放列表或关闭页面时，如有未保存修改必须确认。

字段使用明确标签，不仅依赖 placeholder。保存、撤销、清空和封面操作显示独立进度与结果。

批量编辑要求用户为每个字段勾选是否参与，并选择 set 或 clear。预览显示影响文件数、格式能力不匹配和预计备份空间。任务支持取消；进入单文件原子替换后完成该文件再响应取消。

每个文件返回 `updated`、`skipped` 或 `failed` 及稳定错误码。部分成功不会显示为整体成功，也不会丢弃失败文件的恢复信息。

## 9. 清除、撤销与防偷歌

“清除全部”属于破坏性操作：

- 执行前展示会删除的字段/图片数量并二次确认。
- 保存完整标签快照，而不是仅保存五个文本字段。
- 清除范围由 adapter 定义，并同时处理 MP3 的 ID3v1/ID3v2。
- 撤销完成且验证成功前不删除快照。

防偷歌 V1/V2 复用同一快照和写入层：

- V1 清除所选字段前保存每个文件的完整标签快照。
- V2 只对成功读取并成功备份的 `{path, snapshot}` 集合操作。
- 文件数少于 2 时拒绝 V2。
- 使用无原位映射，保证每个参与文件获得不同来源的标签。
- 任意恢复失败时保留整个恢复 manifest，允许重试单个文件。

V2 只打乱用户在预览中选择的规范化文本字段，默认不移动封面、未知帧或容器私有 chunk；跨格式不复制原始标签块。V1 默认清除用户可见字段，但未知字段是否清除必须作为显式高级选项展示。

旧版不完整 `.id3_backup.json` 只能作为有限恢复来源，UI 必须提示它无法还原封面和未知帧，不能声称完整恢复。

## 10. BPM/Key 与封面压缩

现有 BPM/Key 写回、封面设置/清理和封面压缩全部改为调用 `MetadataService`：

- 查询阶段按 adapter capability 过滤。
- BPM/Key 使用各格式规范字段。
- 写入失败通过 Go `error` 传播到 Wails Promise。
- UI 只在后端验证成功后更新状态。

不再允许 `database/sync.go` 查询到 FLAC/WAV/AIFF 后交给 MP3-only 写入器。

## 11. 备份与恢复格式

`TagSnapshot` 包含：

- 文件稳定标识、原文件大小和哈希。
- 源格式和标签版本。
- 完整原始标签/chunk 数据及其位置。
- 创建时间、操作 ID 和 adapter 版本。

快照原子写入当前用户的 EngineTools 元数据恢复目录，文件名使用随机 ID，不使用播放列表标题直接构造路径。恢复前验证当前文件身份；若音频文件已被外部替换，要求用户显式确认，不自动把旧标签写入新内容。

默认保留最近 20 次已完成操作且不少于 30 天；用户可主动导出或清理。包含失败恢复、未解决部分失败或当前唯一可用撤销点的快照不会被自动清理。

## 12. 错误模型

稳定错误类别包括：不支持格式、格式损坏、不支持字段、文件占用、只读、空间不足、图片超限、解析失败、验证失败、原子替换失败和恢复身份不匹配。

Wails API 使用 `(Result, error)`。错误日志使用脱敏文件标识；逐文件详细结果可在当前用户本地导出，但不会自动上传中心服务。

## 13. 测试

Golden fixtures 至少覆盖：

- MP3 ID3v2.3、v2.4、ID3v1-only、无标签和损坏标签。
- FLAC 多 comment、多 Picture 和未知 metadata block。
- WAV RIFF INFO、ID3、BEXT、iXML、cue、smpl 组合。
- AIFF/AIFF-C 文本、ID3、COMM 和未知 chunk。
- 空值 clear、三态 unchanged、多值字段和编码边界。
- 多 APIC/Picture，仅替换 front cover。
- 超大、伪造 MIME、损坏和资源消耗型图片。
- 文件占用、只读、磁盘不足和每个原子提交故障点。

每次成功编辑必须验证：

- 请求字段正确。
- 未编辑字段与未知数据保留。
- 音频载荷哈希不变。
- 重新读取结果与 UI 一致。

批量测试覆盖部分失败、取消、恢复保留、V2 路径集合不偏移和无原位映射。前端测试覆盖脏状态、错误不伪成功、清除确认和撤销。

## 14. 完成标准

MP3、FLAC、WAV 和 AIFF 均使用正确容器元数据；清空字段确实删除旧值；未知元数据和非目标封面不丢失；破坏性操作可撤销；批量部分失败可准确报告并恢复；音频载荷在元数据编辑后保持不变。

## 15. 相关规格与顺序

本规格依赖桌面安全基础规格中的普通权限进程、结构化错误和任务模型，不依赖中心服务或数据库同步上线。
