<script setup>
import { ref, onMounted, onUnmounted, computed, watchEffect, watch } from 'vue'
import {
    GetStatus,
    GetSettings,
    GetLogEntries,
    WriteFrontendLog,
    SaveSettings,
    ClearCompatibilityRecords,
    GetEngineCacheStatus,
    InstallEngineCache,
    UninstallEngineCache,
    FixUnicodeIssues,
    RestoreUnicodeFix,
    RespondToConfirmation,
    HandleUTF8AlreadyEnabled,
    OpenRegionSettings,
    SetLanguage,
    GetMessages,
    GetAvailableLanguages,
    OpenRepository,
    BackupDatabase,
    ListBackups,
    RestoreDatabase,
    OptimizeDatabase,
    RepairDatabase,
    ScanMSIOrphans,
    CleanMSIOrphans,
    OpenInstallDir,
    OpenStemsDir,
    OpenDBDir,
    ListDrives,
    ListPlaylists,
    GetPlaylistTracks,
    SelectDrive,
    ID3PickFile,
    ID3ReadTag,
    ID3WriteTag,
    ID3GetCover,
    ID3SetCover,
    ID3ClearCover,
    ID3ClearAll,
    ID3AntiPiracyV1,
    ID3AntiPiracyV2,
    ID3AntiPiracyRestore,
    ID3AntiPiracyV1Paths,
    ID3AntiPiracyV2Paths,
    ID3AntiPiracyRestorePaths,
    ID3AntiPiracyV1Playlist,
    ID3AntiPiracyV2Playlist,
    ID3AntiPiracyRestorePlaylist,
    ID3PickDir,
    USBUnlockAvailable,
    USBUnlockScan,
    USBUnlockKill,
    MIDI2Status,
    MIDI2Disable,
    MIDI2Enable,
    CheckForUpdates,
    AnalyzeLogs,
    OpenLogsDir,
    CleanCache,
    GetCacheSize,
    GetLibraryStats,
    GetPlayStats,
    ScanMissingTracks,
    RemoveMissingTracks,
    GetSyncableTracks,
    SyncBPMKeyToTags,
    CompressCovers,
    LogExperimentalEnabled,
} from '../wailsjs/go/main/App.js'
import { EventsOn, WindowSetAlwaysOnTop, WindowMinimise, WindowToggleMaximise, WindowIsMaximised, Quit } from '../wailsjs/runtime/runtime.js'

import { NAV_ITEMS, UI_TEXT } from './appearance.js'
import appLogo from '../../build/appicon.png'

const APP_VERSION = '1.9.0'
const APP_VERSION_LABEL = '1.9.0 Preview'

const installPath = ref('')
const engineVersion = ref('')
const windowsVersion = ref('')
const utf8Enabled = ref(false)
const acpValue = ref('')
const manifestConfigured = ref(false)
const isAdmin = ref(false)
const stemsDetected = ref(false)
const dbDetected = ref(false)
const dbPath = ref('')
const processRunning = ref(false)
const runningProcesses = ref([])
const loading = ref(true)
const fixing = ref(false)
const progress = ref(0)
const showProgress = ref(false)
const msgs = ref({})
const languages = ref([])
const currentLang = ref('zh')
const alwaysOnTop = ref(false)

function toggleAlwaysOnTop() {
    const enabled = !alwaysOnTop.value
    WindowSetAlwaysOnTop(enabled)
    alwaysOnTop.value = enabled
}
const marqueeTexts = ref([])
const currentMarquee = ref(0)

const activeTab = ref('status')
const uiTheme = ref('orange')
const squareCorners = ref(false)
const consoleExpanded = ref(false)
const windowMaximised = ref(false)
async function syncWindowState() {
    windowMaximised.value = await WindowIsMaximised()
}
function toggleWindowMaximise() {
    WindowToggleMaximise()
    requestAnimationFrame(syncWindowState)
}

const uiText = computed(() => UI_TEXT[currentLang.value] || UI_TEXT.en)
const encodingHint = computed(() => ({
    zh: '通过外部 Manifest 修复 Engine DJ 读取中文、韩文、希腊文等特殊字符的问题。还原修复可撤销更改；操作前请先退出 Engine DJ。',
    en: 'Uses external manifests to fix reading Chinese, Korean, Greek and other special characters in Engine DJ. Restore Fix reverses the changes. Close Engine DJ before applying or restoring the fix.',
    ja: '外部 Manifest を使用して Engine DJ の特殊文字の読み込みを修復します。元に戻す操作で変更を取り消せます。操作前に Engine DJ を終了してください。',
    ko: '외부 Manifest로 Engine DJ의 특수 문자 읽기 문제를 수정합니다. 복원으로 변경을 취소할 수 있습니다. 작업 전에 Engine DJ를 종료하세요.',
}[currentLang.value] || 'Fixes special-character reading through external manifests. Close Engine DJ before applying or restoring the fix.'))
const navigation = computed(() => NAV_ITEMS.map(item => ({ ...item, title: uiText.value[item.id], description: uiText.value[item.id + 'Desc'] })))
const currentPage = computed(() => navigation.value.find(item => item.id === activeTab.value) || navigation.value[0])
watchEffect(() => {
    document.documentElement.dataset.theme = uiTheme.value
    document.documentElement.dataset.corners = squareCorners.value ? 'square' : 'rounded'
})


const engineCache = ref({ running: false, active: false, busy: false, phase: 'idle', error: '', stats: {} })
const cacheActionBusy = ref(false)
const cacheInstallCopy = computed(() => ({
    zh: { install: '安装DLL', reinstall: '重新安装DLL', uninstall: '卸载DLL', hint: '安装后无需保持 EngineTools 运行；重新打开可查看缓存状态。安装、卸载前请先退出 Engine DJ。' },
    en: { install: 'Install cache', reinstall: 'Reinstall cache', uninstall: 'Uninstall cache', hint: 'Installed caching runs without EngineTools. Reopen EngineTools to monitor it. Close Engine DJ before installing or uninstalling.' },
    ja: { install: 'キャッシュをインストール', reinstall: '再インストール', uninstall: 'アンインストール', hint: 'インストール後は EngineTools を終了できます。再度開くと状態を確認できます。インストール・削除前に Engine DJ を終了してください。' },
    ko: { install: '캐시 설치', reinstall: '캐시 재설치', uninstall: '캐시 제거', hint: '설치 후 EngineTools를 종료해도 캐시가 실행됩니다. 다시 열어 상태를 확인할 수 있습니다. 설치 및 제거 전 Engine DJ를 종료하세요.' },
}[currentLang.value] || { install: 'Install cache', reinstall: 'Reinstall cache', uninstall: 'Uninstall cache', hint: 'Installed caching runs without EngineTools. Close Engine DJ before installing or uninstalling.' }))
async function handleEngineCacheInstall(remove = false) {
    cacheActionBusy.value = true
    const reinstall = engineCache.value.installed
    try {
        const error = await (remove ? UninstallEngineCache() : InstallEngineCache())
        if (error) { addLog(error); showErrorToast(error) }
        else {
            const labels = {
                zh: ['DLL 安装完成', 'DLL 重新安装完成', 'DLL 卸载完成'],
                en: ['DLL installed', 'DLL reinstalled', 'DLL uninstalled'],
                ja: ['DLL をインストールしました', 'DLL を再インストールしました', 'DLL を削除しました'],
                ko: ['DLL 설치 완료', 'DLL 재설치 완료', 'DLL 제거 완료'],
            }[currentLang.value] || ['DLL installed', 'DLL reinstalled', 'DLL uninstalled']
            const details = {
                zh: ['随 Engine DJ 自动运行，无需保持 EngineTools 开启。', '已移除界面卡顿修复 DLL。'],
                en: ['Runs with Engine DJ without keeping EngineTools open.', 'The UI fix DLL has been removed.'],
                ja: ['Engine DJ とともに動作し、EngineTools の常駐は不要です。', '画面修復 DLL を削除しました。'],
                ko: ['EngineTools를 열어 두지 않아도 Engine DJ와 함께 실행됩니다.', 'UI 수정 DLL이 제거되었습니다.'],
            }[currentLang.value] || ['Runs with Engine DJ without keeping EngineTools open.', 'The UI fix DLL has been removed.']
            showToast(labels[remove ? 2 : reinstall ? 1 : 0], 'success', details[remove ? 1 : 0])
        }
        await refreshEngineCache()
    } catch (error) { addLog(String(error)); showErrorToast(String(error)) }
    finally { cacheActionBusy.value = false }
}
let cachePoll = null
let cachePolling = false
const cacheCopy = computed(() => ({
    zh: { title: '界面卡顿修复', process: '运行状态', stopped: '未启动', cache: '缓存命中', inactive: '未启用', starting: '正在启动…', stopping: '正在退出…', degraded: '未启用 · 出错', misses: '回退', refreshes: '刷新', errors: '错误' },
    en: { title: 'Engine DJ background cache', process: 'Process', stopped: 'Not running', cache: 'Cache hits', inactive: 'Inactive', starting: 'Starting…', stopping: 'Stopping…', degraded: 'Inactive · Error', misses: 'fallbacks', refreshes: 'refreshes', errors: 'errors' },
    ja: { title: 'Engine DJ バックグラウンドキャッシュ', process: '実行状態', stopped: '未起動', cache: 'キャッシュヒット', inactive: '無効', starting: '起動中…', stopping: '終了中…', degraded: '無効 · エラー', misses: 'フォールバック', refreshes: '更新', errors: 'エラー' },
    ko: { title: 'Engine DJ 백그라운드 캐시', process: '실행 상태', stopped: '실행 안 됨', cache: '캐시 적중', inactive: '비활성', starting: '시작 중…', stopping: '종료 중…', degraded: '비활성 · 오류', misses: '대체 조회', refreshes: '새로고침', errors: '오류' },
}[currentLang.value] || {
    title: 'Engine DJ background cache', process: 'Process', stopped: 'Not running', cache: 'Cache hits', inactive: 'Inactive', starting: 'Starting…', stopping: 'Stopping…', degraded: 'Inactive · Error', misses: 'fallbacks', refreshes: 'refreshes', errors: 'errors',
}))
const cacheProcessLabel = computed(() => !engineCache.value.running ? cacheCopy.value.stopped : ({ zh: '已启动', en: 'Running', ja: '起動中', ko: '실행 중' }[currentLang.value] || 'Running'))
const cacheStatusLabel = computed(() => {
    if (engineCache.value.phase === 'starting') return cacheCopy.value.starting
    if (engineCache.value.phase === 'stopping') return cacheCopy.value.stopping
    if (engineCache.value.active) {
        const { hits = 0, lookups = 0 } = engineCache.value.stats
        return `${hits} / ${lookups} (${lookups ? (hits / lookups * 100).toFixed(1) : '0.0'}%)`
    }
    return engineCache.value.error ? cacheCopy.value.degraded : cacheCopy.value.inactive
})
const cacheStatsLabel = computed(() => `${cacheCopy.value.misses} ${engineCache.value.stats.misses || 0} · ${cacheCopy.value.refreshes} ${engineCache.value.stats.refreshes || 0} · ${cacheCopy.value.errors} ${engineCache.value.stats.errors || 0}`)
async function refreshEngineCache() {
    if (cachePolling) return
    cachePolling = true
    try { engineCache.value = await GetEngineCacheStatus() }
    catch (error) { engineCache.value.error = String(error) }
    finally { cachePolling = false }
}


// Database state
const backups = ref([])
const backupNote = ref('')
const selectedBackup = ref('')
const dbBusy = ref(false)
const drives = ref([])
const selectedDrive = ref('')

// MSI tools state
const msiOrphans = ref([])
const msiSelected = ref([])
const msiScanned = ref(false)
const msiBusy = ref(false)

// ID3 Editor state
const id3File = ref('')
const editingTrack = ref(null) // TrackInfo from playlist click
const id3Title = ref('')
const id3Artist = ref('')
const id3Album = ref('')
const id3Year = ref('')
const id3Genre = ref('')
const id3Cover = ref('')
const id3Busy = ref(false)
const id3DragOver = ref(false)

// USB Unlock state
const usbDrive = ref('')
const usbBlockers = ref([])
const usbBusy = ref(false)

// MIDI 2.0 state
const midi2Status = ref('') // 'enabled' | 'disabled' | 'unavailable'
const midi2Busy = ref(false)

// Update checker state
const updateAvailable = ref(false)
const updateInfo = ref(null)
const checkingUpdate = ref(false)

// Log analysis state
const logStats = ref(null)
const logBusy = ref(false)

// Cache cleanup state
const cacheBusy = ref(false)
const cacheSize = ref(0)

// Tab lazy-load flags
const dbTabLoaded = ref(false)
const devicesTabLoaded = ref(false)
const audioTabLoaded = ref(false)

// Library stats state
const libraryStats = ref(null)
const libraryStatsBusy = ref(false)

// Play stats state
const playStats = ref(null)
const playStatsBusy = ref(false)
const playStatsTab = ref('most') // 'most' | 'recent' | 'never'

// Missing tracks state
const missingTracks = ref([])
const missingSelected = ref([])
const missingTracksLibrary = ref('')
const missingBusy = ref(false)
const missingScanned = ref(false)

// BPM/Key sync state
const syncBusy = ref(false)
const syncResult = ref(null)
const syncPreviewTracks = ref([])
const syncPreviewLoaded = ref(false)
const syncPreviewLibrary = ref('')

// Cover compression state
const coverCompressBusy = ref(false)
const coverCompressResult = ref(null)
const coverCompressPlaylistId = ref(-1) // -1 = entire library

// Anti-piracy easter egg state (controlled by experimental toggle)
const apDir = ref('')
const apFile = ref('')
const apPlaylistId = ref(-1)
const apScope = ref('folder') // 'file' | 'folder' | 'playlist'
const apBusy = ref(false)

// Experimental features state
const experimentalEnabled = ref(false)
let settingsReady = false
const compatibilityRecords = ref([])
const compatibilityBusy = ref(false)
const compatibilityText = computed(() => ({
    zh: { title: '兼容性检测记录', description: '保存未知 Engine DJ 版本的检测结果，文件未变化时复用。清除后下次安装将重新检测，不影响已安装的 DLL。', empty: '暂无兼容性检测记录', clear: '清除此条', clearAll: '清空全部', checked: '检测时间', callers: '调用位置', validator: '检测组件', details: '查看详情', refresh: '刷新', cleared: '检测记录已清除' },
    en: { title: 'Compatibility checks', description: 'Saved checks for unknown Engine DJ builds are reused when files are unchanged. Clearing requires a new check on the next installation and does not affect the installed DLL.', empty: 'No saved compatibility checks', clear: 'Clear record', clearAll: 'Clear all', checked: 'Checked at', callers: 'Call sites', validator: 'Detector', details: 'View details', refresh: 'Refresh', cleared: 'Compatibility records cleared' },
    ja: { title: '互換性チェック履歴', description: '未検証の Engine DJ のチェック結果を保存し、ファイルが同じ場合に再利用します。削除すると次回インストール時に再チェックします。インストール済み DLL には影響しません。', empty: '保存された履歴はありません', clear: 'この履歴を削除', clearAll: 'すべて削除', checked: 'チェック日時', callers: '呼び出し位置', validator: '検証コンポーネント', details: '詳細を表示', refresh: '更新', cleared: '履歴を削除しました' },
    ko: { title: '호환성 검사 기록', description: '알 수 없는 Engine DJ 버전의 검사 결과를 저장하고 파일이 같으면 재사용합니다. 삭제하면 다음 설치 시 다시 검사하며 설치된 DLL에는 영향을 주지 않습니다.', empty: '저장된 검사 기록이 없습니다', clear: '기록 삭제', clearAll: '전체 삭제', checked: '검사 시간', callers: '호출 위치', validator: '검사 구성 요소', details: '상세 보기', refresh: '새로고침', cleared: '검사 기록이 삭제되었습니다' },
}[currentLang.value] || {
    title: 'Compatibility checks', description: 'Saved checks are reused when files are unchanged. Clearing does not affect the installed DLL.', empty: 'No saved compatibility checks', clear: 'Clear record', clearAll: 'Clear all', checked: 'Checked at', callers: 'Call sites', validator: 'Detector', details: 'View details', refresh: 'Refresh', cleared: 'Compatibility records cleared',
}))
function updateCompatibilityRecords(settings) {
    compatibilityRecords.value = Object.entries(settings.engineDJCompatibility || {})
        .map(([key, value]) => ({ ...value, key }))
        .sort((a, b) => (b.checkedAt || '').localeCompare(a.checkedAt || ''))
}
function formatCheckTime(value) {
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString(currentLang.value)
}
async function refreshCompatibilityRecords() {
    compatibilityBusy.value = true
    try { updateCompatibilityRecords(await GetSettings()) }
    catch (error) { addLog(String(error)); showErrorToast(String(error)) }
    finally { compatibilityBusy.value = false }
}
async function clearCompatibilityRecords(key = '') {
    compatibilityBusy.value = true
    try {
        await ClearCompatibilityRecords(key)
        updateCompatibilityRecords(await GetSettings())
        showToast(compatibilityText.value.cleared)
    } catch (error) { addLog(String(error)); showErrorToast(String(error)) }
    finally { compatibilityBusy.value = false }
}
let settingsSaveQueue = Promise.resolve()
function settingsSnapshot() {
    return { language: currentLang.value, theme: uiTheme.value, squareCorners: squareCorners.value,
        alwaysOnTop: alwaysOnTop.value, experimentalEnabled: experimentalEnabled.value }
}
watch([currentLang, uiTheme, squareCorners, alwaysOnTop, experimentalEnabled], () => {
    if (!settingsReady) return
    const snapshot = settingsSnapshot()
    settingsSaveQueue = settingsSaveQueue.then(() => SaveSettings(snapshot)).catch(error => {
        addLog('Save setting.json failed: ' + error)
        showErrorToast('setting.json: ' + error)
    })
}, { flush: 'sync' })

async function loadSettings() {
    const settings = await GetSettings()
    updateCompatibilityRecords(settings)
    currentLang.value = settings.language || 'zh'
    uiTheme.value = settings.theme || 'orange'
    squareCorners.value = !!settings.squareCorners
    alwaysOnTop.value = !!settings.alwaysOnTop
    experimentalEnabled.value = !!settings.experimentalEnabled
    settingsReady = true
}
const showExperimentalConfirm = ref(false)
const experimentalCountdown = ref(0)
let experimentalTimer = null

// Konami Code easter egg
const konamiUrls = [
    'https://www.bilibili.com/video/BV1dg4y1Z7B1',
    'https://www.bilibili.com/video/BV1GJ411x7h7',
    'https://www.bilibili.com/video/BV17KQcBfE69',
    'https://www.bilibili.com/video/BV1XXVh65EgE',
    'https://www.bilibili.com/video/BV1t9wszgE61',
    'https://www.bilibili.com/video/BV1SqxHzoEuf',
    'https://www.bilibili.com/video/BV1j4411W7F7',
]
const konamiCode = ['ArrowUp','ArrowUp','ArrowDown','ArrowDown','ArrowLeft','ArrowLeft','ArrowRight','ArrowRight','b','a']
const konamiProgress = ref([])

// Playlist viewer state
const playlists = ref([])
const playlistsLibrary = ref('')
const selectedPlaylist = ref(null)
const playlistTracks = ref([])
const playlistLoading = ref(false)

// Toast notification
const toasts = ref([])
const toastTimers = new Map()
let nextToastId = 0
const toastIcons = {
    success: 'm8 12 3 3 5-6',
    error: 'm9 9 6 6m0-6-6 6',
    warning: 'M12 7v6m0 4h.01',
    info: 'M12 11v6m0-10h.01',
}
const toastLabels = computed(() => ({
    zh: { done: '操作已完成', error: '操作失败', warning: '请留意操作说明', info: '提示' },
    en: { done: 'Operation completed', error: 'Operation failed', warning: 'Please review the details', info: 'Information' },
    ja: { done: '操作が完了しました', error: '操作に失敗しました', warning: '説明をご確認ください', info: 'お知らせ' },
    ko: { done: '작업이 완료되었습니다', error: '작업 실패', warning: '안내를 확인하세요', info: '알림' },
}[currentLang.value] || { done: 'Operation completed', error: 'Operation failed', warning: 'Please review the details', info: 'Information' }))

function showToast(message, type = 'success', description = '') {
    const lines = String(message ?? '').split(/\r?\n/).map(line => line.trim()).filter(Boolean)
    if (!lines.length) return
    const title = type === 'error' && lines.length === 1 ? toastLabels.value.error : lines.shift()
    const detail = description || lines.join('\n') || toastLabels.value[type === 'success' ? 'done' : type] || toastLabels.value.info
    const id = ++nextToastId
    toasts.value.push({ id, type, title, description: detail })
    return id
}

function startToastTimer(element) {
    const id = Number(element.dataset.toastId)
    if (!toasts.value.some(toast => toast.id === id)) return
    toastTimers.set(id, setTimeout(() => closeToast(id), 3000))
}

function closeToast(id) {
    clearTimeout(toastTimers.get(id))
    toastTimers.delete(id)
    toasts.value = toasts.value.filter(toast => toast.id !== id)
}

function showErrorToast(msg) {
    showToast(msg, 'error')
}

const confirmationQueue = ref([])
const confirmation = computed(() => confirmationQueue.value[0])
const confirmationBusy = ref(false)
const confirmationLabels = computed(() => ({
    zh: ['确认', '取消'], en: ['Confirm', 'Cancel'], ja: ['確認', 'キャンセル'], ko: ['확인', '취소'],
}[currentLang.value] || ['Confirm', 'Cancel']))
function confirmAction(message) {
    return new Promise(resolve => {
        confirmationQueue.value.push({ title: confirmationLabels.value[0], message, resolve })
        requestAnimationFrame(() => document.getElementById('confirmation-cancel')?.focus())
    })
}
async function answerConfirmation(accepted) {
    if (!confirmation.value || confirmationBusy.value) return
    confirmationBusy.value = true
    try {
        if (confirmation.value.resolve) confirmation.value.resolve(accepted)
        else await RespondToConfirmation(confirmation.value.id, accepted)
        confirmationQueue.value.shift()
    } catch (error) { showErrorToast(String(error)) }
    finally { confirmationBusy.value = false }
}

const cacheSizeText = computed(() => {
    if (cacheSize.value <= 0) return ''
    const mb = (cacheSize.value / 1024 / 1024).toFixed(1)
    return (msgs.value.cacheSizeLabel || 'Cache size') + ': ' + mb + ' MB'
})

const maxPlayCount = computed(() => {
    if (!playStats.value?.mostPlayed?.length) return 1
    return Math.max(1, ...playStats.value.mostPlayed.map(t => t.playCount))
})

const id3FileName = computed(() => {
    if (!id3File.value) return ''
    return id3File.value.split(/[/\\]/).pop()
})

const pathDisplay = computed(() => {
    return installPath.value || msgs.value?.installPathNotFound || '—'
})

const versionDisplay = computed(() => {
    return engineVersion.value || msgs.value?.notInstalled || '—'
})

const utf8StatusText = computed(() => {
    if (!msgs.value) return ''
    const status = utf8Enabled.value ? msgs.value.utf8Enabled : msgs.value.utf8Disabled
    const acp = acpValue.value ? ` (ACP=${acpValue.value})` : ''
    return `${status}${acp}`
})

const manifestStatusText = computed(() => {
    if (!msgs.value) return ''
    return manifestConfigured.value ? msgs.value.manifestExists : msgs.value.manifestNotExists
})

const adminStatusText = computed(() => {
    if (!msgs.value) return ''
    return isAdmin.value ? msgs.value.adminYes : msgs.value.adminNo
})

const stemsStatusText = computed(() => {
    if (!msgs.value) return ''
    return stemsDetected.value ? msgs.value.stemsDetected : msgs.value.stemsNotFound
})

const dbPathDisplay = computed(() => {
    return dbPath.value || msgs.value?.dbLibraryNotFound || '—'
})

const logs = ref([])
const logContainer = ref(null)

function appendLog(entry) {
    logs.value.push(entry)
    if (logs.value.length > 2000) logs.value.splice(0, logs.value.length - 2000)
    requestAnimationFrame(() => {
        if (logContainer.value) {
            logContainer.value.scrollTop = logContainer.value.scrollHeight
        }
    })
}

function addLog(message) {
    Promise.resolve().then(() => WriteFrontendLog(String(message))).catch(error => appendLog('Log write failed: ' + error))
}
function logWindowError(event) { addLog('Unhandled frontend error: ' + (event.error?.stack || event.message)) }
function logUnhandledRejection(event) { addLog('Unhandled promise rejection: ' + (event.reason?.stack || String(event.reason))) }

async function loadMessages() {
    msgs.value = await GetMessages()
    languages.value = await GetAvailableLanguages()
    updateMarqueeTexts()
}

function updateMarqueeTexts() {
    const m = msgs.value
    if (!m) return
    marqueeTexts.value = [
        `v${APP_VERSION_LABEL}`,
        m.marqueeFree || 'This program is free. If you paid, request a refund.',
        m.marqueeStar || 'If you like this project, please give it a Star on GitHub.',
    ]
}

async function detectStatus() {
    loading.value = true
    try {
        const status = await GetStatus()
        installPath.value = status.installPath || ''
        engineVersion.value = status.engineVersion || ''
        windowsVersion.value = status.windowsVersion || ''
        utf8Enabled.value = status.utf8Enabled || false
        acpValue.value = status.acpValue || ''
        manifestConfigured.value = status.manifestConfigured || false
        isAdmin.value = status.isAdmin || false
        stemsDetected.value = status.stemsDetected || false
        dbDetected.value = status.dbDetected || false
        dbPath.value = status.dbPath || ''
        processRunning.value = status.processRunning || false
        runningProcesses.value = status.runningProcesses || []
    } catch (e) {
        addLog('Error: ' + e)
    }
    loading.value = false
}

async function handleFix() {
    if (utf8Enabled.value) {
        try { showToast(await HandleUTF8AlreadyEnabled(), 'warning') }
        catch (error) { addLog(String(error)); showErrorToast(String(error)) }
        return
    }

    fixing.value = true
    showProgress.value = true
    progress.value = 0

    try {
        const result = await FixUnicodeIssues()
        if (result === 'ok') {
            manifestConfigured.value = true
            showToast([msgs.value.fixComplete || 'Fix complete', msgs.value.fixCompleteTip].filter(Boolean).join('\n'))
        } else if (result !== 'cancelled') showErrorToast(result || msgs.value.fixFailed || 'Fix failed')
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(String(e))
    }

    fixing.value = false
    setTimeout(() => {
        showProgress.value = false
    }, 1000)
    await detectStatus()
}

async function handleRestore() {
    fixing.value = true
    showProgress.value = true
    progress.value = 0

    try {
        const result = await RestoreUnicodeFix()
        if (result === 'ok') showToast([msgs.value.restoreComplete || 'Restore complete', msgs.value.restoreCompleteTip].filter(Boolean).join('\n'))
        else if (result !== 'cancelled') showErrorToast(result || msgs.value.fixFailed || 'Restore failed')
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(String(e))
    }

    fixing.value = false
    setTimeout(() => {
        showProgress.value = false
    }, 1000)
    await detectStatus()
}

async function handleOpenRegionSettings() {
    await OpenRegionSettings()
}

async function openInstallDir() {
    await OpenInstallDir()
}

async function openStemsDir() {
    await OpenStemsDir()
}

async function openDBDir() {
    await OpenDBDir()
}

// ---- Database tab ----

async function refreshBackups() {
    try {
        backups.value = await ListBackups()
    } catch (e) {
        addLog('Error: ' + e)
        backups.value = []
    }
}

async function handleBackup() {
    dbBusy.value = true
    try {
        await BackupDatabase(backupNote.value || '')
        backupNote.value = ''
        await refreshBackups()
        showToast(msgs.value.dbBackupComplete || 'Backup complete')
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.dbBackupError || 'Backup failed')
    }
    dbBusy.value = false
}

async function handleRestoreDB() {
    if (!selectedBackup.value) return
    dbBusy.value = true
    try {
        const result = await RestoreDatabase(selectedBackup.value)
        if (result === 'ok') showToast(msgs.value.dbRestoreComplete || 'Restore complete')
        else if (result !== 'cancelled') showErrorToast(result || msgs.value.fixFailed || 'Restore failed')
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.dbRestoreConfirmTitle || 'Restore failed')
    }
    dbBusy.value = false
}

async function handleOptimize() {
    dbBusy.value = true
    try {
        await OptimizeDatabase()
        showToast(msgs.value.dbOptimizeComplete || 'Optimization complete')
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.dbOptimizeButton || 'Optimization failed')
    }
    dbBusy.value = false
}

async function handleRepair() {
    dbBusy.value = true
    try {
        await RepairDatabase()
        showToast(msgs.value.dbRepairComplete || 'Repair complete')
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.dbRepairButton || 'Repair failed')
    }
    dbBusy.value = false
    await detectStatus()
}

// ---- Library Stats ----

async function handleLibraryStats() {
    libraryStatsBusy.value = true
    try {
        libraryStats.value = await GetLibraryStats()
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.libraryStatsTitle || '曲库统计失败')
    }
    libraryStatsBusy.value = false
}

function formatDuration(seconds) {
    if (!seconds) return '0h 0m'
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    return `${h}h ${m}m`
}

function formatBytes(bytes) {
    if (!bytes) return '0 MB'
    const gb = bytes / (1024 * 1024 * 1024)
    if (gb >= 1) return gb.toFixed(1) + ' GB'
    return (bytes / (1024 * 1024)).toFixed(0) + ' MB'
}

// ---- Play Stats ----

async function handlePlayStats() {
    playStatsBusy.value = true
    try {
        playStats.value = await GetPlayStats()
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.playHistoryTitle || '播放统计失败')
    }
    playStatsBusy.value = false
}

// ---- Missing Tracks ----

async function handleScanMissing() {
    missingBusy.value = true
    missingSelected.value = []
    try {
        const library = dbPath.value
        const tracks = await ScanMissingTracks(library)
        if (dbPath.value !== library) {
            missingBusy.value = false
            return
        }
        missingTracksLibrary.value = library
        missingTracks.value = tracks
        missingScanned.value = true
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.missingTracksScan || '扫描失败')
    }
    missingBusy.value = false
}

function toggleMissingTrack(id) {
    const idx = missingSelected.value.indexOf(id)
    if (idx === -1) missingSelected.value.push(id)
    else missingSelected.value.splice(idx, 1)
}

function toggleAllMissing() {
    if (missingSelected.value.length === missingTracks.value.length) {
        missingSelected.value = []
    } else {
        missingSelected.value = missingTracks.value.map(t => t.id)
    }
}

async function handleRemoveMissing() {
    if (missingBusy.value) return
    const library = missingTracksLibrary.value
    if (library !== dbPath.value) return
    const ids = [...missingSelected.value]
    if (ids.length === 0) return
    missingBusy.value = true
    const ok = await confirmAction(`确认从数据库删除 ${ids.length} 条缺失记录？此操作不可撤销。`)
    if (!ok) {
        missingBusy.value = false
        return
    }
    try {
        if (await RemoveMissingTracks(ids, library) !== 'ok') throw new Error('删除缺失记录失败')
        showToast(`已删除 ${ids.length} 条记录`)
        await handleScanMissing()
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.missingTracksRemove || '删除失败')
    }
    missingBusy.value = false
}

// ---- BPM/Key Sync ----

async function handleLoadSyncPreview() {
    syncPreviewLoaded.value = false
    try {
        const library = dbPath.value
        const tracks = await GetSyncableTracks(library)
        if (dbPath.value !== library) return
        syncPreviewLibrary.value = library
        syncPreviewTracks.value = tracks
        syncPreviewLoaded.value = true
    } catch (e) {
        addLog('Error: ' + e)
    }
}

async function handleSyncBPMKey() {
    if (!syncPreviewLoaded.value || syncBusy.value || syncPreviewLibrary.value !== dbPath.value) return
    const library = syncPreviewLibrary.value
    syncBusy.value = true
    const ok = await confirmAction(`将向 ${syncPreviewTracks.value.length} 个文件写回 BPM 和 Key 标签。确认继续？`)
    if (!ok) {
        syncBusy.value = false
        return
    }
    syncResult.value = null
    try {
        syncResult.value = await SyncBPMKeyToTags(library)
        if (syncResult.value.error) throw new Error(syncResult.value.error)
        if (syncResult.value.success > 0) {
            showToast(`写回完成：${syncResult.value.success} 成功`)
        }
        if (syncResult.value.failed > 0) {
            showErrorToast(`${syncResult.value.failed} 个文件写入失败`)
        }
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.bpmKeySyncTitle || '写回失败')
    }
    syncBusy.value = false
}

// ---- Cover Art Compression ----

async function handleCompressCovers() {
    if (coverCompressBusy.value) return
    const library = dbPath.value
    const playlistId = coverCompressPlaylistId.value
    if (playlistId >= 0 && playlistsLibrary.value !== library) return
    const scope = playlistId < 0 ? (msgs.value.coverCompressAllLibrary || '全部曲库') : `播放列表 ${playlistId}`
    coverCompressBusy.value = true
    const ok = await confirmAction(`将压缩 ${scope} 中所有大于 1 MB 的封面至 1 MB 以内。此操作会修改音频文件的 ID3 标签。\n\n确认继续？`)
    if (!ok) {
        coverCompressBusy.value = false
        return
    }
    coverCompressResult.value = null
    try {
        coverCompressResult.value = await CompressCovers(playlistId, 1024, library)
        const r = coverCompressResult.value
        if (r.error) throw new Error(r.error)
        if (r.compressed > 0) {
            const saved = r.savedBytes > 1048576
                ? (r.savedBytes / 1048576).toFixed(1) + ' MB'
                : Math.round(r.savedBytes / 1024) + ' KB'
            showToast(`压缩完成：${r.compressed} 首，节省 ${saved}`)
        } else {
            showToast('所有封面已在 1 MB 以内，无需压缩')
        }
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.coverCompressTitle || '封面压缩失败')
    }
    coverCompressBusy.value = false
}

async function handleAnalyzeLogs() {
    logBusy.value = true
    try {
        logStats.value = await AnalyzeLogs()
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.logAnalysisTitle || 'Log analysis failed')
        logStats.value = null
    }
    logBusy.value = false
}

async function handleOpenLogsDir() {
    try {
        await OpenLogsDir()
    } catch (e) {
        addLog('Error: ' + e)
    }
}

async function handleClearCache() {
    cacheBusy.value = true
    try {
        await CleanCache()
        cacheSize.value = 0
        showToast(msgs.value.cacheCleanComplete || 'Cache cleared')
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.cacheCleanTitle || 'Cache clear failed')
    }
    cacheBusy.value = false
}

// ---- Experimental Features ----

async function handleExperimentalToggle() {
    if (experimentalEnabled.value) {
        experimentalEnabled.value = false
        return
    }
    showExperimentalConfirm.value = true
    experimentalCountdown.value = 5
    clearInterval(experimentalTimer)
    experimentalTimer = setInterval(() => {
        experimentalCountdown.value--
        if (experimentalCountdown.value <= 0) clearInterval(experimentalTimer)
    }, 1000)
}

function cancelExperimental() {
    showExperimentalConfirm.value = false
    clearInterval(experimentalTimer)
    experimentalCountdown.value = 0
}

async function confirmExperimental() {
    if (experimentalCountdown.value > 0) return
    try {
        const result = await LogExperimentalEnabled()
        if (result !== 'ok') throw new Error(result)
        experimentalEnabled.value = true
    } catch (e) {
        addLog('Error logging experimental enable: ' + e)
    }
    showExperimentalConfirm.value = false
}

// ---- Konami Code ----

function handleKeydown(e) {
    const expected = konamiCode[konamiProgress.value.length]
    if (e.key === expected) {
        konamiProgress.value = [...konamiProgress.value, e.key]
        if (konamiProgress.value.length === konamiCode.length) {
            konamiProgress.value = []
            const url = konamiUrls[Math.floor(Math.random() * konamiUrls.length)]
            window.open(url, '_blank')
        }
    } else {
        konamiProgress.value = e.key === konamiCode[0] ? [e.key] : []
    }
}

async function handlePickApDir() {
    try {
        const dir = await ID3PickDir()
        if (dir) apDir.value = dir
    } catch (e) {
        addLog('Error: ' + e)
    }
}

async function handlePickApFile() {
    try {
        const path = await ID3PickFile()
        if (path) apFile.value = path
    } catch (e) {
        addLog('Error: ' + e)
    }
}

function apBackupKey() {
    if (apScope.value === 'folder') return apDir.value
    if (apScope.value === 'file') return apFile.value ? apFile.value + '_ap' : 'single'
    return 'playlist_' + apPlaylistId.value
}

function apIsReady() {
    if (apScope.value === 'folder') return !!apDir.value
    if (apScope.value === 'file') return !!apFile.value
    return apPlaylistId.value >= 0
}

async function handleAntiPiracyV1() {
    if (apBusy.value) return
    if (!apIsReady()) return
    const library = dbPath.value
    const scope = apScope.value
    if (scope === 'playlist' && playlistsLibrary.value !== library) return
    const directory = apDir.value
    const file = apFile.value
    const playlistId = apPlaylistId.value
    const backupKey = apBackupKey()
    apBusy.value = true
    const ok = await confirmAction('防偷歌 v1：清除所选范围内所有音频文件的ID3标签（操作前自动备份）。确认执行？')
    if (!ok) {
        apBusy.value = false
        return
    }
    try {
        if (scope === 'folder') {
            await ID3AntiPiracyV1(directory)
        } else if (scope === 'file') {
            await ID3AntiPiracyV1Paths([file], backupKey)
        } else {
            await ID3AntiPiracyV1Playlist(playlistId)
        }
        showToast('v1 执行完成')
    } catch (e) {
        addLog('Error: ' + e)
    }
    apBusy.value = false
}

async function handleAntiPiracyV2() {
    if (apBusy.value) return
    if (!apIsReady()) return
    const library = dbPath.value
    const scope = apScope.value
    if (scope === 'playlist' && playlistsLibrary.value !== library) return
    const directory = apDir.value
    const file = apFile.value
    const playlistId = apPlaylistId.value
    const backupKey = apBackupKey()
    apBusy.value = true
    const ok = await confirmAction('防偷歌 v2：随机打乱所选范围内所有音频文件的ID3标签（操作前自动备份）。确认执行？')
    if (!ok) {
        apBusy.value = false
        return
    }
    try {
        if (scope === 'folder') {
            await ID3AntiPiracyV2(directory)
        } else if (scope === 'file') {
            await ID3AntiPiracyV2Paths([file], backupKey)
        } else {
            await ID3AntiPiracyV2Playlist(playlistId)
        }
        showToast('v2 执行完成')
    } catch (e) {
        addLog('Error: ' + e)
    }
    apBusy.value = false
}

async function handleAntiPiracyRestore() {
    if (!apIsReady()) return
    apBusy.value = true
    try {
        if (apScope.value === 'folder') {
            await ID3AntiPiracyRestore(apDir.value)
        } else if (apScope.value === 'file') {
            await ID3AntiPiracyRestorePaths(apBackupKey())
        } else {
            await ID3AntiPiracyRestorePlaylist(apPlaylistId.value)
        }
        showToast('恢复完成')
    } catch (e) {
        addLog('Error: ' + e)
    }
    apBusy.value = false
}

async function loadPlaylists() {
    playlistLoading.value = true
    try {
        const library = dbPath.value
        const items = await ListPlaylists(library)
        if (dbPath.value === library) {
            if (playlistsLibrary.value !== library) {
                apPlaylistId.value = -1
                coverCompressPlaylistId.value = -1
            }
            playlistsLibrary.value = library
            playlists.value = items
        }
    } catch (e) {
        addLog('Error: ' + e)
        playlists.value = []
    }
    playlistLoading.value = false
}

async function handleSelectPlaylist(pl) {
    selectedPlaylist.value = pl
    playlistLoading.value = true
    try {
        playlistTracks.value = await GetPlaylistTracks(pl.id)
    } catch (e) {
        addLog('Error: ' + e)
        playlistTracks.value = []
    }
    playlistLoading.value = false
}

function formatTime(seconds) {
    if (!seconds) return '--:--'
    const m = Math.floor(seconds / 60)
    const s = Math.floor(seconds % 60)
    return m + ':' + (s < 10 ? '0' : '') + s
}

async function loadDrives() {
    try {
        drives.value = await ListDrives()
    } catch (e) {
        addLog('Error: ' + e)
        drives.value = []
    }
}

async function handleClickDrive(driveInfo) {
    selectedDrive.value = driveInfo.letter
    dbBusy.value = true
    try {
        await SelectDrive(driveInfo.letter)
        // Backend emits statusUpdate which triggers detectStatus() via event listener
    } catch (e) {
        addLog('Error: ' + e)
    }
    dbBusy.value = false
}

// ---- Tools tab (MSI cleanup) ----

async function handleScanMSI() {
    msiBusy.value = true
    msiSelected.value = []
    try {
        msiOrphans.value = await ScanMSIOrphans()
        msiScanned.value = true
    } catch (e) {
        addLog('Error: ' + e)
        msiOrphans.value = []
    }
    msiBusy.value = false
}

function toggleMSI(code) {
    const idx = msiSelected.value.indexOf(code)
    if (idx === -1) {
        msiSelected.value.push(code)
    } else {
        msiSelected.value.splice(idx, 1)
    }
}

async function handleCleanMSI() {
    if (msiSelected.value.length === 0) return
    msiBusy.value = true
    try {
        const result = await CleanMSIOrphans(msiSelected.value)
        if (result === 'ok') {
            showToast(msgs.value.msiCleanComplete || 'Cleanup complete')
            await handleScanMSI()
        } else if (result !== 'cancelled') showErrorToast(result || msgs.value.msiCleanError || 'Cleanup failed')
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(String(e))
    }
    msiBusy.value = false
}

// ---- ID3 Editor ----

async function handleEditTrackFromPlaylist(track) {
    if (!track.path) return
    editingTrack.value = track
    id3File.value = track.path
    await loadID3(track.path)
    // Navigate to Tools tab so the user sees the dedicated editor
    await switchTab('audio')
}

async function handleID3Pick() {
    const path = await ID3PickFile()
    if (!path) return
    editingTrack.value = null
    id3File.value = path
    await loadID3(path)
}

async function handleID3FileDrop(event) {
    id3DragOver.value = false
    const files = event.dataTransfer?.files
    if (!files?.length) return
    const audioExts = /\.(mp3|flac|wav|aiff|aif)$/i
    for (const f of files) {
        if (audioExts.test(f.name)) {
            // In Wails WebView2, file.path may be available
            const path = f.path || ''
            if (path) {
                editingTrack.value = null
                id3File.value = path
                await loadID3(path)
                return
            }
        }
    }
    // Fallback: open file picker if path unavailable
    await handleID3Pick()
}

async function loadID3(path) {
    id3Busy.value = true
    try {
        const info = await ID3ReadTag(path)
        id3Title.value = info.title || ''
        id3Artist.value = info.artist || ''
        id3Album.value = info.album || ''
        id3Year.value = info.year || ''
        id3Genre.value = info.genre || ''
        id3Cover.value = await ID3GetCover(path) || ''
    } catch (e) {
        addLog('Error: ' + e)
    }
    id3Busy.value = false
}

async function handleID3Save() {
    if (!id3File.value) return
    id3Busy.value = true
    try {
        await ID3WriteTag(id3File.value, id3Title.value, id3Artist.value, id3Album.value, id3Year.value, id3Genre.value)
    } catch (e) {
        addLog('Error: ' + e)
    }
    id3Busy.value = false
}

async function handleID3SetCover() {
    if (!id3File.value) return
    id3Busy.value = true
    try {
        const res = await ID3SetCover(id3File.value)
        if (res === 'ok') {
            id3Cover.value = await ID3GetCover(id3File.value) || ''
        }
    } catch (e) {
        addLog('Error: ' + e)
    }
    id3Busy.value = false
}

async function handleID3ClearCover() {
    if (!id3File.value) return
    id3Busy.value = true
    try {
        await ID3ClearCover(id3File.value)
        id3Cover.value = ''
    } catch (e) {
        addLog('Error: ' + e)
    }
    id3Busy.value = false
}

async function handleID3ClearAll() {
    if (!id3File.value) return
    id3Busy.value = true
    try {
        await ID3ClearAll(id3File.value)
        id3Title.value = ''
        id3Artist.value = ''
        id3Album.value = ''
        id3Year.value = ''
        id3Genre.value = ''
        id3Cover.value = ''
    } catch (e) {
        addLog('Error: ' + e)
    }
    id3Busy.value = false
}

// ---- USB Unlock ----

async function checkUSBDrive() {
    try {
        usbDrive.value = await USBUnlockAvailable()
    } catch (e) {
        usbDrive.value = ''
    }
}

async function handleUSBScan() {
    if (!usbDrive.value) return
    usbBusy.value = true
    try {
        usbBlockers.value = await USBUnlockScan(usbDrive.value)
    } catch (e) {
        addLog('Error: ' + e)
        usbBlockers.value = []
    }
    usbBusy.value = false
}

async function handleUSBUnlock() {
    if (!usbDrive.value) return
    usbBusy.value = true
    try {
        await USBUnlockKill(usbDrive.value)
        usbBlockers.value = []
        await checkUSBDrive()
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.usbUnlockTitle || 'USB unlock failed')
    }
    usbBusy.value = false
}

// ---- MIDI 2.0 ----

async function checkMIDI2Status() {
    try {
        midi2Status.value = await MIDI2Status()
    } catch (e) {
        midi2Status.value = 'unavailable'
    }
}

async function handleMIDI2Disable() {
    midi2Busy.value = true
    try {
        const res = await MIDI2Disable()
        addLog(res)
        await checkMIDI2Status()
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.midi2Title || 'MIDI 2.0 disable failed')
    }
    midi2Busy.value = false
}

async function handleMIDI2Enable() {
    midi2Busy.value = true
    try {
        const res = await MIDI2Enable()
        addLog(res)
        await checkMIDI2Status()
    } catch (e) {
        addLog('Error: ' + e)
        showErrorToast(msgs.value.midi2Title || 'MIDI 2.0 enable failed')
    }
    midi2Busy.value = false
}

async function switchTab(tab) {
    activeTab.value = tab
    if (tab === 'diagnostics') await refreshCompatibilityRecords()
    if (tab === 'database' && !dbTabLoaded.value) {
        await loadDrives()
        await refreshBackups()
        await loadPlaylists()
        dbTabLoaded.value = true
    }
    if (tab === 'devices' && !devicesTabLoaded.value) {
        await checkUSBDrive()
        await checkMIDI2Status()
        devicesTabLoaded.value = true
    }
    if (tab === 'audio' && !audioTabLoaded.value) {
        if (playlists.value.length === 0) await loadPlaylists()
        audioTabLoaded.value = true
    }
}

async function openGitHub() {
    await OpenRepository()
}

async function changeLanguage(lang) {
    const error = await SetLanguage(lang)
    if (error) { addLog(error); showErrorToast(error); return }
    currentLang.value = lang
    await loadMessages()
    await detectStatus()
}

async function checkForUpdates() {
    checkingUpdate.value = true
    try {
        const result = await CheckForUpdates(APP_VERSION)
        if (result.error) {
            addLog('Update check failed: ' + result.error)
        } else if (result.hasUpdate) {
            updateAvailable.value = true
            updateInfo.value = result
            addLog('New version available: ' + result.version)
        } else {
            addLog('You are using the latest version')
        }
    } catch (e) {
        addLog('Update check error: ' + e)
    }
    checkingUpdate.value = false
}

async function openUpdatePage() {
    if (updateInfo.value && updateInfo.value.url) {
        window.open(updateInfo.value.url, '_blank')
    }
}

let marqueeInterval = null

onMounted(async () => {
    EventsOn('log', appendLog)
    window.addEventListener('error', logWindowError)
    window.addEventListener('unhandledrejection', logUnhandledRejection)
    try { logs.value = [...new Set([...(await GetLogEntries()), ...logs.value])].slice(-2000) }
    catch (error) { addLog(String(error)) }
    try { await loadSettings() }
    catch (error) { addLog('Load setting.json failed: ' + error); showErrorToast(String(error)) }
    window.addEventListener('resize', syncWindowState)
    await syncWindowState()
    await refreshEngineCache()
    cachePoll = setInterval(refreshEngineCache, 2000)
    WindowSetAlwaysOnTop(alwaysOnTop.value)
    await loadMessages()

    EventsOn('confirmation', request => {
        confirmationQueue.value.push(request)
        requestAnimationFrame(() => document.getElementById('confirmation-cancel')?.focus())
    })
    EventsOn('progress', (value) => { progress.value = value * 100 })
    EventsOn('statusUpdate', async () => { await detectStatus() })

    await detectStatus()
    try { cacheSize.value = await GetCacheSize() } catch (_) {}

    // Konami Code listener
    window.addEventListener('keydown', handleKeydown)

    marqueeInterval = setInterval(() => {
        currentMarquee.value = (currentMarquee.value + 1) % marqueeTexts.value.length
    }, 5000)

    setTimeout(() => { checkForUpdates() }, 1000)
})
onUnmounted(() => {
    for (const timer of toastTimers.values()) clearTimeout(timer)
    toastTimers.clear()
    for (const request of confirmationQueue.value) {
        if (request.resolve) request.resolve(false)
        else RespondToConfirmation(request.id, false).catch(() => {})
    }
    window.removeEventListener('error', logWindowError)
    window.removeEventListener('unhandledrejection', logUnhandledRejection)
    if (cachePoll) clearInterval(cachePoll)
    window.removeEventListener('resize', syncWindowState)
})
</script>

<template>
    <div class="notification-stack">
        <Transition name="toast">
            <section v-if="confirmation" class="confirmation-bubble" role="alertdialog" aria-labelledby="confirmation-title" aria-describedby="confirmation-message" @keydown.esc.prevent="answerConfirmation(false)">
                <svg class="notification-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path d="M12 7v6m0 4h.01"/></svg>
                <div class="toast-copy">
                    <strong id="confirmation-title" class="toast-title">{{ confirmation.title }}</strong>
                    <p id="confirmation-message" class="toast-description">{{ confirmation.message }}</p>
                    <div class="confirmation-actions">
                        <button id="confirmation-cancel" class="btn btn-secondary no-drag" :disabled="confirmationBusy" @click="answerConfirmation(false)">{{ confirmationLabels[1] }}</button>
                        <button class="btn btn-primary no-drag" :disabled="confirmationBusy" @click="answerConfirmation(true)">{{ confirmationLabels[0] }}</button>
                    </div>
                </div>
                <button class="toast-close no-drag" :disabled="confirmationBusy" :aria-label="uiText.closeWindow" @click="answerConfirmation(false)">×</button>
            </section>
        </Transition>
        <TransitionGroup name="toast" tag="div" class="toast-list" @after-enter="startToastTimer">
            <div v-for="toast in toasts" :key="toast.id" :data-toast-id="toast.id" class="toast" :class="'toast-' + toast.type" :role="toast.type === 'error' ? 'alert' : 'status'">
                <svg class="notification-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="9"/><path :d="toastIcons[toast.type] || toastIcons.info"/></svg>
                <div class="toast-copy"><strong class="toast-title">{{ toast.title }}</strong><p class="toast-description">{{ toast.description }}</p></div>
                <button class="toast-close no-drag" :aria-label="uiText.closeWindow" @click="closeToast(toast.id)">×</button>
            </div>
        </TransitionGroup>
    </div>

    <div class="app-shell">
    <div class="header wails-drag">
        <div class="header-left">
            <img class="app-logo" :src="appLogo" alt="" draggable="false" />
            <span class="app-title">{{ msgs.appTitle || 'Engine Tools' }}</span>
            <span class="marquee-text" :key="currentMarquee">{{ marqueeTexts[currentMarquee] }}</span>
        </div>
        <div class="header-right">
            <span v-if="updateAvailable" class="update-badge no-drag" @click="openUpdatePage" :title="msgs.updateTooltip || 'New version available'">
                {{ msgs.updateBadge || 'Update' }}
            </span>
            <span v-else-if="checkingUpdate" class="update-badge no-drag checking">
                {{ msgs.updateChecking || 'Checking...' }}
            </span>
            <span class="github-link no-drag" @click="openGitHub">
                <svg class="github-icon" viewBox="0 0 16 16" fill="currentColor" width="16" height="16"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>
            </span>
            <span class="author-handle no-drag" @click="openGitHub">@LaokeQwQ</span>
            <button class="update-check-btn no-drag" @click="checkForUpdates" title="Check for updates">
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="23 4 23 10 17 10"></polyline>
                    <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
                </svg>
            </button>
            <div class="window-controls no-drag">
                <button type="button" class="window-control" :aria-label="uiText.minimise" :title="uiText.minimise" @click="WindowMinimise">
                    <svg viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="currentColor" aria-hidden="true"><path d="M1 6.5h10" /></svg>
                </button>
                <button type="button" class="window-control" :aria-label="windowMaximised ? uiText.restoreWindow : uiText.maximise" :title="windowMaximised ? uiText.restoreWindow : uiText.maximise" @click="toggleWindowMaximise">
                    <svg v-if="windowMaximised" viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="currentColor" aria-hidden="true"><path d="M3.5 3.5v-2h7v7h-2" /><rect x="1.5" y="3.5" width="7" height="7" /></svg>
                    <svg v-else viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="currentColor" aria-hidden="true"><rect x="1.5" y="1.5" width="9" height="9" /></svg>
                </button>
                <button type="button" class="window-control window-close" :aria-label="uiText.closeWindow" :title="uiText.closeWindow" @click="Quit">
                    <svg viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="currentColor" aria-hidden="true"><path d="m1.5 1.5 9 9m0-9-9 9" /></svg>
                </button>
            </div>
        </div>
    </div>

    <aside class="sidebar no-drag">
        <nav :aria-label="uiText.workspace">
            <div v-for="group in ['workspace', 'maintenance', 'preferences']" :key="group" class="nav-group">
                <div class="nav-group-title">{{ uiText[group] }}</div>
                <button v-for="item in navigation.filter(entry => entry.group === group)" :key="item.id"
                    class="nav-item" :class="{ active: activeTab === item.id }"
                    :aria-current="activeTab === item.id ? 'page' : undefined" @click="switchTab(item.id)">
                    <svg viewBox="0 0 24 24" width="17" height="17" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="item.icon" /></svg>
                    <span>{{ item.title }}</span>
                </button>
            </div>
        </nav>
        <div class="sidebar-version">v{{ APP_VERSION_LABEL }}</div>
    </aside>
    <main class="workspace" id="workspace">
        <div class="page-heading" :key="activeTab">
            <div>
                <div class="page-eyebrow">{{ uiText[currentPage.group] }}</div>
                <h1>{{ currentPage.title }}</h1>
                <p>{{ currentPage.description }}</p>
            </div>
        </div>
    <!-- STATUS TAB -->
    <div class="content overview-content" v-show="activeTab === 'status'">
        <div class="info-card">
            <div class="info-row" v-if="windowsVersion">
                <span class="info-label">{{ msgs.windowsVersionLabel || 'Windows' }}</span>
                <span class="info-value" :title="windowsVersion">{{ windowsVersion }}</span>
            </div>
            <div class="info-row">
                <span class="info-label">{{ msgs.installPathLabel || 'Path' }}</span>
                <span class="info-value-group">
                    <span class="info-value" :class="{ muted: !installPath }" :title="installPath">{{ pathDisplay }}</span>
                    <span
                        v-if="installPath"
                        class="folder-btn no-drag"
                        :title="msgs.installPathLabel || 'Path'"
                        @click="openInstallDir"
                    >
                        <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                    </span>
                </span>
            </div>
            <div class="info-row">
                <span class="info-label">{{ msgs.engineVersionLabel || 'Version' }}</span>
                <span class="info-value" :class="{ muted: !engineVersion }" :title="versionDisplay">{{ versionDisplay }}</span>
            </div>
        </div>

        <section class="overview-panel compatibility-panel">
            <div class="panel-heading cache-panel-heading">
                <span>{{ uiText.compatibility }}</span>
                <span class="cache-help">
                    <button type="button" class="cache-help-button no-drag" :aria-label="encodingHint" aria-describedby="encoding-help-tooltip">?</button>
                    <span id="encoding-help-tooltip" class="cache-help-tooltip" role="tooltip">{{ encodingHint }}</span>
                </span>
            </div>
        <div class="status-grid">
            <div class="status-card" :class="isAdmin ? 'status-success' : 'status-error'">
                <div class="status-text">
                    <div class="status-title">{{ msgs.adminStatusLabel || 'Admin Privileges' }}</div>
                    <div class="status-detail">{{ adminStatusText }}</div>
                </div>
            </div>

            <div class="status-card" :class="utf8Enabled ? 'status-warning' : 'status-success'">
                <div class="status-text">
                    <div class="status-title">{{ msgs.utf8StatusLabel || 'System UTF-8 Support' }}</div>
                    <div class="status-detail">{{ utf8StatusText }}</div>
                </div>
            </div>

            <div class="status-card" :class="manifestConfigured ? 'status-success' : 'status-error'">
                <div class="status-text">
                    <div class="status-title">{{ msgs.manifestStatusLabel || 'External Manifest' }}</div>
                    <div class="status-detail">{{ manifestStatusText }}</div>
                </div>
            </div>

            <div class="status-card status-card-row" :class="stemsDetected ? 'status-success' : 'status-warning'">
                <div class="status-text">
                    <div class="status-title">{{ msgs.stemsStatusLabel || 'STEM Processor' }}</div>
                    <div class="status-detail">{{ stemsStatusText }}</div>
                </div>
                <span
                    v-if="stemsDetected"
                    class="folder-btn no-drag"
                    :title="msgs.stemsStatusLabel || 'STEM Processor'"
                    @click="openStemsDir"
                >
                    <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                </span>
            </div>
        </div>

        <div class="actions status-actions">
            <button
                class="btn btn-primary no-drag"
                @click="handleFix"
                :disabled="fixing || !installPath"
            >
                <span v-if="fixing" class="loading-spinner"></span>
                <span class="fix-button-label">{{ fixing ? (msgs.progressFixing || 'Fixing...') : (msgs.fixButton || 'Fix Unicode Character Encoding Issues').replace(/\s*([（(])/, '\n$1') }}</span>
            </button>

            <button
                v-if="manifestConfigured"
                class="btn btn-restore-all no-drag"
                @click="handleRestore"
                :disabled="fixing || !installPath"
            >
                {{ msgs.restoreButton || 'Restore Fix' }}
            </button>

            <button
                v-if="utf8Enabled"
                class="btn btn-secondary no-drag"
                @click="handleOpenRegionSettings"
            >
                {{ msgs.openRegionSettings || 'Open Region Settings' }}
            </button>
        </div>
        </section>
        <section class="engine-cache-section overview-panel" :aria-label="cacheCopy.title">
            <div class="library-section-title cache-panel-heading">
                <span>{{ cacheCopy.title }}</span>
                <span class="cache-help">
                    <button type="button" class="cache-help-button no-drag" :aria-label="cacheInstallCopy.hint" aria-describedby="cache-help-tooltip">?</button>
                    <span id="cache-help-tooltip" class="cache-help-tooltip" role="tooltip">{{ cacheInstallCopy.hint }}</span>
                </span>
            </div>
            <div class="status-grid">
                <div class="status-card" :class="engineCache.running ? 'status-success' : 'status-warning'">
                    <div class="status-text">
                        <div class="status-title">{{ cacheCopy.process }}</div>
                        <div class="status-detail">{{ cacheProcessLabel }}</div>
                    </div>
                </div>
                <div class="status-card cache-hit-card" :class="engineCache.active ? 'status-success' : engineCache.error ? 'status-error' : 'status-warning'">
                    <div class="status-text" aria-live="polite">
                        <div class="cache-hit-heading">
                            <div class="status-title">{{ cacheCopy.cache }}</div>
                            <span v-if="engineCache.active" class="cache-hit-stats" :title="cacheStatsLabel">{{ cacheStatsLabel }}</span>
                        </div>
                        <div class="status-detail">{{ cacheStatusLabel }}</div>
                    </div>
                </div>
                <div class="status-card cache-install-status" :class="engineCache.installed ? 'status-success' : engineCache.installStatus === 'conflict' ? 'status-error' : 'status-warning'">
                    <div class="status-text">
                        <div class="status-title">{{ ({ zh: '安装状态', en: 'Installation', ja: 'インストール状態', ko: '설치 상태' }[currentLang] || 'Installation') }}</div>
                        <div class="status-detail">{{ ({ zh: { installed: '已安装', missing: '未安装', outdated: '需要更新', conflict: 'DLL 冲突' }, en: { installed: 'Installed', missing: 'Not installed', outdated: 'Update required', conflict: 'DLL conflict' }, ja: { installed: 'インストール済み', missing: '未インストール', outdated: '更新が必要', conflict: 'DLL の競合' }, ko: { installed: '설치됨', missing: '설치 안 됨', outdated: '업데이트 필요', conflict: 'DLL 충돌' } }[currentLang] || { installed: 'Installed', missing: 'Not installed', outdated: 'Update required', conflict: 'DLL conflict' })[engineCache.installStatus || 'missing'] }}</div>
                    </div>
                </div>
                <div class="status-card cache-proxy-version" :class="engineCache.installed ? 'status-success' : 'status-warning'">
                    <div class="status-text">
                        <div class="status-title">version.dll {{ ({ zh: '版本', en: 'version', ja: 'バージョン', ko: '버전' }[currentLang] || 'version') }}</div>
                        <div class="status-detail">{{ engineCache.proxyVersion || '—' }}</div>
                    </div>
                </div>
            </div>
            <div class="actions status-actions">
                <button class="btn btn-primary no-drag" @click="handleEngineCacheInstall(false)"
                    :disabled="cacheActionBusy || engineCache.busy || engineCache.running || engineCache.installStatus === 'conflict' || !installPath">
                    {{ engineCache.installed ? cacheInstallCopy.reinstall : cacheInstallCopy.install }}
                </button>
                <button class="btn btn-restore-all no-drag" @click="handleEngineCacheInstall(true)"
                    :disabled="cacheActionBusy || engineCache.busy || engineCache.running || !engineCache.installed">
                    {{ cacheInstallCopy.uninstall }}
                </button>
            </div>
            <div v-if="engineCache.installError" class="engine-cache-error" role="alert">{{ engineCache.installError }}</div>
            <div v-if="engineCache.error" class="engine-cache-error" role="alert">{{ engineCache.error }}</div>
        </section>
    </div>

    <!-- DATABASE TAB -->
    <div class="content library-content" v-show="activeTab === 'database'">
        <div class="library-section">
            <div class="library-section-title">{{ msgs.dbSelectDriveLabel || 'Select Drive' }}</div>
            <div class="drive-grid no-drag">
                <button
                    v-for="d in drives"
                    :key="d.letter"
                    class="drive-chip"
                    :class="{ active: selectedDrive === d.letter }"
                    :disabled="dbBusy"
                    @click="handleClickDrive(d)"
                >
                    <span class="drive-letter">{{ d.letter }}</span>
                    <span class="drive-count">{{ d.trackCount }} tracks</span>
                </button>
            </div>
        </div>

        <div class="info-card" v-if="dbDetected">
            <div class="info-row">
                <span class="info-label">{{ msgs.dbLibraryPathLabel || 'Database Path' }}</span>
                <span class="info-value-group">
                    <span class="info-value">{{ dbPathDisplay }}</span>
                    <span
                        class="folder-btn no-drag"
                        :title="msgs.dbLibraryPathLabel || 'Database Path'"
                        @click="openDBDir"
                    >
                        <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                    </span>
                </span>
            </div>
        </div>
        <div v-else-if="selectedDrive && !dbDetected" class="status-card status-error">
            <div class="status-text">
                <div class="status-detail">{{ msgs.dbLibraryNotFound || 'Engine Library not found' }}</div>
            </div>
        </div>

        <template v-if="dbDetected">
            <div class="library-section">
                <div class="library-section-title">{{ msgs.dbBackupButton || 'Backup' }}</div>
                <input
                    class="text-input no-drag"
                    type="text"
                    v-model="backupNote"
                    :placeholder="msgs.dbBackupNotePlaceholder || 'Optional: enter a note'"
                    :disabled="dbBusy"
                />
                <button class="btn btn-primary no-drag" @click="handleBackup" :disabled="dbBusy">
                    <span v-if="dbBusy" class="loading-spinner"></span>
                    {{ dbBusy ? (msgs.dbBackingUp || 'Backing up...') : (msgs.dbBackupButton || 'Backup') }}
                </button>
            </div>

            <div class="library-section">
                <div class="library-section-title">{{ msgs.dbRestoreButton || 'Restore' }}</div>
                <select class="text-input no-drag" v-model="selectedBackup" :disabled="dbBusy || backups.length === 0">
                    <option value="" disabled>{{ msgs.dbRestoreSelectDate || 'Select date to restore' }}</option>
                    <option v-for="b in backups" :key="b.filename" :value="b.filename">
                        {{ b.date }}{{ b.note ? ' — ' + b.note : '' }}
                    </option>
                </select>
                <div v-if="backups.length === 0" class="library-stats">{{ msgs.dbNoBackups || 'No Backups' }}</div>
                <button
                    class="btn btn-restore-all no-drag"
                    @click="handleRestoreDB"
                    :disabled="dbBusy || !selectedBackup"
                >
                    {{ dbBusy ? (msgs.dbRestoring || 'Restoring...') : (msgs.dbRestoreButton || 'Restore') }}
                </button>
            </div>

            <div class="library-section">
                <div class="library-section-title">{{ msgs.dbOptimizeButton || 'Optimize' }}</div>
                <button class="btn btn-secondary no-drag" @click="handleOptimize" :disabled="dbBusy">
                    <span v-if="dbBusy" class="loading-spinner"></span>
                    {{ dbBusy ? (msgs.dbOptimizing || 'Optimizing...') : (msgs.dbOptimizeButton || 'Optimize') }}
                </button>
            </div>

            <div class="library-section">
                <div class="library-section-title">{{ msgs.dbRepairButton || 'Repair Database' }}</div>
                <button class="btn btn-secondary no-drag" @click="handleRepair" :disabled="dbBusy">
                    <span v-if="dbBusy" class="loading-spinner"></span>
                    {{ dbBusy ? (msgs.dbRepairing || 'Repairing...') : (msgs.dbRepairButton || 'Repair Database') }}
                </button>
            </div>

            <!-- Playlist Viewer -->
            <div class="library-section" style="margin-top: 12px;">
                <div class="library-section-title">{{ msgs.tabPlaylist || 'Playlist' }}</div>
                <div v-if="playlists.length === 0 && !playlistLoading" class="library-stats">{{ msgs.dbNoneFound || 'No playlists found' }}</div>
                <div v-if="playlistLoading && !selectedPlaylist" class="library-stats">Loading...</div>
                <div v-if="playlists.length > 0" class="playlist-chips">
                    <span
                        v-for="pl in playlists"
                        :key="pl.id"
                        class="playlist-chip no-drag"
                        :class="{ active: selectedPlaylist && selectedPlaylist.id === pl.id }"
                        @click="handleSelectPlaylist(pl)"
                    >
                        {{ pl.title }} <span class="playlist-chip-count">{{ pl.count }}</span>
                    </span>
                </div>
            </div>

            <div v-if="selectedPlaylist && playlistTracks.length > 0" class="library-section" style="margin-top: 4px;">
                <div class="playlist-track-list">
                    <div v-for="(t, i) in playlistTracks" :key="t.id"
                        class="playlist-track-row no-drag"
                        :class="{ 'track-editing': editingTrack && editingTrack.id === t.id }"
                        :style="t.path ? 'cursor:pointer;' : ''"
                        @click="t.path && handleEditTrackFromPlaylist(t)">
                        <span class="playlist-track-num">{{ i + 1 }}</span>
                        <div class="playlist-track-info">
                            <div class="playlist-track-title">{{ t.title || t.filename }}</div>
                            <div class="playlist-track-artist">{{ t.artist }}</div>
                        </div>
                        <span class="playlist-track-bpm" v-if="t.bpm">{{ Math.round(t.bpm) }}</span>
                        <span class="playlist-track-bpm" v-if="t.camelot" style="color:var(--accent);font-size:10px;">{{ t.camelot }}</span>
                        <span class="playlist-track-time">{{ formatTime(t.length) }}</span>
                    </div>
                </div>
            </div>

            <!-- 点击曲目将跳转到音频文件的 ID3 编辑器 -->
        </template>

        <!-- Library Stats -->
        <div class="library-section" style="margin-top:12px;" v-if="dbDetected">
            <div class="library-section-title">{{ msgs.libraryStatsTitle || 'Library Stats' }}</div>
            <button class="btn btn-secondary no-drag" @click="handleLibraryStats" :disabled="libraryStatsBusy">
                <span v-if="libraryStatsBusy" class="loading-spinner"></span>
                {{ libraryStatsBusy ? (msgs.libraryStatsBusy || 'Analyzing...') : (msgs.libraryStatsButton || 'Show Stats') }}
            </button>
            <template v-if="libraryStats">
                <div class="info-card" style="margin-top:8px;">
                    <div class="info-row"><span class="info-label">{{ msgs.totalTracksLabel || '曲目总数' }}</span><span class="info-value">{{ libraryStats.totalTracks }}</span></div>
                    <div class="info-row"><span class="info-label">{{ msgs.totalDurationLabel || '总时长' }}</span><span class="info-value">{{ formatDuration(libraryStats.totalDuration) }}</span></div>
                    <div class="info-row"><span class="info-label">{{ msgs.librarySizeLabel || '占用空间' }}</span><span class="info-value">{{ formatBytes(libraryStats.totalFileBytes) }}</span></div>
                    <div class="info-row"><span class="info-label">{{ msgs.analyzedLabel || '已分析' }}</span><span class="info-value">{{ libraryStats.analyzedTracks }}<span class="muted" style="font-size:11px;"> / {{ libraryStats.totalTracks }}</span></span></div>
                    <div class="info-row" v-if="libraryStats.missingTracks > 0">
                        <span class="info-label" style="color:var(--error)">{{ msgs.missingTracksTitle || '缺失文件' }}</span>
                        <span class="info-value" style="color:var(--error)">{{ libraryStats.missingTracks }}</span>
                    </div>
                    <div class="info-row"><span class="info-label">{{ msgs.playHistoryNeverPlayed || '从未播放' }}</span><span class="info-value">{{ libraryStats.neverPlayed }}</span></div>
                </div>
                <div v-if="libraryStats.topGenres.length > 0" class="info-card" style="margin-top:6px;">
                    <div class="info-row" v-for="g in libraryStats.topGenres.slice(0,5)" :key="g.key">
                        <span class="info-label">{{ g.key }}</span><span class="info-value muted">{{ g.count }}</span>
                    </div>
                </div>
                <div v-if="libraryStats.fileTypes.length > 0" class="info-card" style="margin-top:6px;">
                    <div class="info-row" v-for="f in libraryStats.fileTypes" :key="f.key">
                        <span class="info-label">{{ f.key }}</span><span class="info-value muted">{{ f.count }}</span>
                    </div>
                </div>
            </template>
        </div>

        <!-- Missing Tracks -->
        <div class="library-section" style="margin-top:12px;" v-if="dbDetected">
            <div class="library-section-title" style="color:var(--error)">{{ msgs.missingTracksTitle || 'Missing Tracks' }}</div>
            <div class="library-stats">{{ msgs.missingTracksDesc || '扫描文件路径已失效的曲目' }}</div>
            <button class="btn btn-primary no-drag" @click="handleScanMissing" :disabled="missingBusy">
                <span v-if="missingBusy" class="loading-spinner"></span>
                {{ missingBusy ? (msgs.missingTracksScanning || 'Scanning...') : (msgs.missingTracksScan || 'Scan Missing Files') }}
            </button>
            <template v-if="missingScanned">
                <div v-if="missingTracks.length === 0" class="status-card status-success" style="margin-top:8px;">
                    <div class="status-text"><div class="status-detail">{{ msgs.missingTracksNone || 'No missing tracks ✓' }}</div></div>
                </div>
                <template v-else>
                    <div class="library-section-title" style="margin-top:8px;color:var(--error)">{{ missingTracks.length }} {{ msgs.missingTracksTitle || '条缺失' }}</div>
                    <div class="drive-select-row" style="margin-bottom:6px;">
                        <button class="btn btn-secondary no-drag" style="flex:1;font-size:11px;" @click="toggleAllMissing" :disabled="missingBusy">
                            {{ missingSelected.length === missingTracks.length ? (msgs.missingTracksDeselectAll || 'Deselect All') : (msgs.missingTracksSelectAll || 'Select All') }}
                        </button>
                        <button class="btn btn-restore-all no-drag" style="flex:1;font-size:11px;" @click="handleRemoveMissing" :disabled="missingBusy || missingSelected.length === 0">
                            {{ msgs.missingTracksRemove || 'Remove' }} {{ missingSelected.length > 0 ? '(' + missingSelected.length + ')' : '' }}
                        </button>
                    </div>
                    <div class="playlist-track-list" style="max-height:200px;overflow-y:auto;">
                        <div v-for="t in missingTracks" :key="t.id" class="playlist-track-row no-drag" @click="toggleMissingTrack(t.id)"
                            :style="missingSelected.includes(t.id) ? 'background:rgba(255,80,80,0.1);' : ''">
                            <input type="checkbox" :checked="missingSelected.includes(t.id)" @click.stop="toggleMissingTrack(t.id)" style="flex-shrink:0;" />
                            <div class="playlist-track-info">
                                <div class="playlist-track-title" style="color:var(--error)">{{ t.title || t.filename }}</div>
                                <div class="playlist-track-artist">{{ t.artist }}</div>
                            </div>
                        </div>
                    </div>
                </template>
            </template>
        </div>

        <!-- Play Stats -->
        <div class="library-section" style="margin-top:12px;" v-if="dbDetected">
            <div class="library-section-title">{{ msgs.playHistoryTitle || '播放历史' }}</div>
            <button class="btn btn-secondary no-drag" @click="handlePlayStats" :disabled="playStatsBusy">
                <span v-if="playStatsBusy" class="loading-spinner"></span>
                {{ playStatsBusy ? (msgs.playHistoryLoading || '加载中...') : (msgs.playHistoryLoad || '加载统计') }}
            </button>
            <template v-if="playStats">
                <div class="playlist-chips" style="margin-top:8px;">
                    <span class="playlist-chip no-drag" :class="{ active: playStatsTab === 'most' }" @click="playStatsTab = 'most'">{{ msgs.playHistoryMostPlayed || '已播放' }}</span>
                    <span class="playlist-chip no-drag" :class="{ active: playStatsTab === 'recent' }" @click="playStatsTab = 'recent'">{{ msgs.playHistoryRecent || '最近播放' }}</span>
                    <span class="playlist-chip no-drag" :class="{ active: playStatsTab === 'never' }" @click="playStatsTab = 'never'">{{ msgs.playHistoryNeverPlayed || '从未播放' }}</span>
                </div>

                <!-- Played tracks (sorted by recency — Engine DJ has no per-track play count) -->
                <div v-if="playStatsTab === 'most'" class="playlist-track-list" style="max-height:220px;overflow-y:auto;margin-top:4px;">
                    <div v-if="playStats.mostPlayed.length === 0" class="library-stats">{{ msgs.dbNoneFound || '暂无数据' }}</div>
                    <div v-for="t in playStats.mostPlayed" :key="t.id" class="playlist-track-row">
                        <div class="playlist-track-info">
                            <div class="playlist-track-title">{{ t.title || '—' }}</div>
                            <div class="playlist-track-artist">{{ t.artist }}</div>
                        </div>
                        <span class="playlist-track-time" style="font-size:10px;">{{ t.lastPlayed.slice(0,10) }}</span>
                    </div>
                </div>

                <!-- Recently Played: list -->
                <div v-if="playStatsTab === 'recent'" class="playlist-track-list" style="max-height:220px;overflow-y:auto;margin-top:4px;">
                    <div v-if="playStats.recentPlayed.length === 0" class="library-stats">{{ msgs.dbNoneFound || '暂无数据' }}</div>
                    <div v-for="t in playStats.recentPlayed" :key="t.id" class="playlist-track-row">
                        <div class="playlist-track-info">
                            <div class="playlist-track-title">{{ t.title }}</div>
                            <div class="playlist-track-artist">{{ t.artist }}</div>
                        </div>
                        <span class="playlist-track-time" style="font-size:10px;">{{ t.lastPlayed.slice(0,10) }}</span>
                    </div>
                </div>

                <!-- Never Played: list -->
                <div v-if="playStatsTab === 'never'" class="playlist-track-list" style="max-height:220px;overflow-y:auto;margin-top:4px;">
                    <div v-if="playStats.neverPlayed.length === 0" class="library-stats">{{ msgs.dbNoneFound || '暂无数据' }}</div>
                    <div v-for="t in playStats.neverPlayed" :key="t.id" class="playlist-track-row">
                        <div class="playlist-track-info">
                            <div class="playlist-track-title">{{ t.title }}</div>
                            <div class="playlist-track-artist">{{ t.artist }}</div>
                        </div>
                        <span class="playlist-track-bpm" style="color:var(--text-secondary)">{{ t.bpm > 0 ? Math.round(t.bpm) : '' }}</span>
                    </div>
                </div>
            </template>
        </div>
    </div>

    <!-- AUDIO -->
    <div class="content tools-content" v-show="activeTab === 'audio'">
        <section class="tool-panel panel-wide">
        <!-- ID3 Editor -->
        <div class="library-section" style="margin-top: 12px;">
            <div class="library-section-title">{{ msgs.id3EditorTitle || 'ID3 标签编辑' }}</div>

            <!-- Drop zone / file picker -->
            <div class="id3-drop-zone no-drag"
                :class="{ 'drop-active': id3DragOver }"
                @click="handleID3Pick"
                @dragover.prevent="id3DragOver = true"
                @dragleave="id3DragOver = false"
                @drop.prevent="handleID3FileDrop">
                <span v-if="id3File" class="id3-drop-filename">{{ id3FileName }}</span>
                <span v-else class="id3-drop-hint">{{ msgs.id3DropZoneHint || '拖入音频文件，或点击选择' }}</span>
            </div>

            <!-- Hint if loaded from playlist -->
            <div v-if="editingTrack" class="library-stats" style="margin-top:4px;font-size:11px;">
                {{ editingTrack.artist ? editingTrack.artist + ' — ' : '' }}{{ editingTrack.title || editingTrack.filename }}
                <span style="margin-left:8px;cursor:pointer;color:var(--text-secondary);" @click="editingTrack=null;id3File=''" class="no-drag">✕</span>
            </div>
        </div>

        <template v-if="id3File">
            <div class="info-card">
                <div class="id3-cover-row">
                    <img v-if="id3Cover" :src="id3Cover" class="id3-cover-img" />
                    <div v-else class="id3-cover-placeholder">{{ msgs.noCoverLabel || '无封面' }}</div>
                    <div class="id3-cover-actions">
                        <button class="btn btn-secondary no-drag id3-cover-btn" @click="handleID3SetCover" :disabled="id3Busy">{{ msgs.changeCoverLabel || '更换' }}</button>
                        <button class="btn btn-secondary no-drag id3-cover-btn" @click="handleID3ClearCover" :disabled="id3Busy || !id3Cover">{{ msgs.removeCoverLabel || '删除' }}</button>
                    </div>
                </div>
            </div>
            <div class="library-section">
                <input class="text-input no-drag" v-model="id3Title" :placeholder="msgs.id3TitlePlaceholder || '标题'" :disabled="id3Busy" />
                <input class="text-input no-drag" v-model="id3Artist" :placeholder="msgs.id3ArtistPlaceholder || '艺术家'" :disabled="id3Busy" />
                <input class="text-input no-drag" v-model="id3Album" :placeholder="msgs.id3AlbumPlaceholder || '专辑'" :disabled="id3Busy" />
                <div class="drive-select-row">
                    <input class="text-input no-drag" v-model="id3Year" :placeholder="msgs.id3YearPlaceholder || '年份'" :disabled="id3Busy" style="flex:1" />
                    <input class="text-input no-drag" v-model="id3Genre" :placeholder="msgs.id3GenrePlaceholder || '风格'" :disabled="id3Busy" style="flex:1" />
                </div>
                <div class="drive-select-row">
                    <button class="btn btn-primary no-drag" style="flex:1" @click="handleID3Save" :disabled="id3Busy">{{ msgs.id3SaveButton || '保存' }}</button>
                    <button class="btn btn-restore-all no-drag" style="flex:1" @click="handleID3ClearAll" :disabled="id3Busy">{{ msgs.id3ClearAllButton || '清除全部' }}</button>
                </div>
            </div>
        </template>
        </section>
        <section class="tool-panel" v-if="experimentalEnabled">
        <!-- BPM / Key Write-back (experimental) -->
        <div class="library-section" style="margin-top: 12px;" v-if="experimentalEnabled">
            <div class="library-section-title">{{ msgs.bpmKeySyncTitle || 'BPM / Key → ID3 Sync' }}</div>
            <div class="library-stats">{{ msgs.bpmKeySyncDesc || '将 Engine DJ 分析出的 BPM 和调号写回音频文件标签（TBPM / TKEY）' }}</div>
            <button class="btn btn-secondary no-drag" @click="handleLoadSyncPreview" :disabled="syncBusy">
                {{ syncPreviewLoaded ? syncPreviewTracks.length + ' ' + (msgs.bpmKeySyncReady || 'tracks ready') : (msgs.bpmKeySyncLoad || 'Load Tracks') }}
            </button>
            <template v-if="syncPreviewLoaded">
                <div class="info-card" style="margin-top:6px;" v-if="syncPreviewTracks.length > 0">
                    <div class="info-row" v-for="t in syncPreviewTracks.slice(0,5)" :key="t.id">
                        <span class="info-label" style="max-width:60%;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">{{ t.artist ? t.artist + ' – ' + t.title : t.title }}</span>
                        <span class="info-value" style="color:var(--text-secondary);font-size:11px;">{{ t.bpm > 0 ? Math.round(t.bpm) + ' BPM' : '' }}{{ t.camelot ? ' · ' + t.camelot : '' }}</span>
                    </div>
                    <div v-if="syncPreviewTracks.length > 5" class="info-row">
                        <span class="info-label muted">… {{ msgs.andMoreLabel || '及其他' }} {{ syncPreviewTracks.length - 5 }} {{ msgs.tracksUnit || '首' }}</span>
                    </div>
                </div>
                <button v-if="syncPreviewTracks.length > 0"
                    class="btn btn-primary no-drag" style="margin-top:6px;"
                    @click="handleSyncBPMKey" :disabled="syncBusy">
                    <span v-if="syncBusy" class="loading-spinner"></span>
                    {{ syncBusy ? (msgs.bpmKeySyncWriting || 'Writing...') : (msgs.bpmKeySyncWrite || 'Write Tags') + ' (' + syncPreviewTracks.length + ')' }}
                </button>
            </template>
            <template v-if="syncResult">
                <div class="status-card" :class="syncResult.failed > 0 ? 'status-warning' : 'status-success'" style="margin-top:8px;">
                    <div class="status-text">
                        <div class="status-detail">✓ {{ syncResult.success }} {{ msgs.bpmKeySyncWrite || '写回' }}{{ syncResult.failed > 0 ? '   ✗ ' + syncResult.failed + ' ' + (msgs.failedLabel || '失败') : '' }}</div>
                    </div>
                </div>
            </template>
        </div>
        </section>
        <section class="tool-panel">
        <!-- Cover Art Compression -->
        <div class="library-section" style="margin-top: 12px;">
            <div class="library-section-title">{{ msgs.coverCompressTitle || 'Cover Art Compression' }}</div>

            <!-- Tips -->
            <div class="library-stats" style="border-left:3px solid var(--warning);padding-left:10px;margin-bottom:10px;">
                {{ msgs.coverCompressTip || 'Compressing covers to ≤1 MB improves Engine OS USB load performance.' }}
            </div>

            <!-- Scope selector -->
            <select class="text-input no-drag" v-model="coverCompressPlaylistId"
                :disabled="coverCompressBusy" style="margin-bottom:8px;">
                <option :value="-1">{{ msgs.coverCompressAllLibrary || '全部曲库' }}</option>
                <option v-for="pl in playlists" :key="pl.id" :value="pl.id">
                    {{ pl.title }} ({{ pl.count }})
                </option>
            </select>

            <button class="btn btn-primary no-drag" @click="handleCompressCovers" :disabled="coverCompressBusy">
                <span v-if="coverCompressBusy" class="loading-spinner"></span>
                {{ coverCompressBusy ? (msgs.coverCompressCompressing || 'Compressing...') : (msgs.coverCompressButton || 'Compress Covers ≤ 1 MB') }}
            </button>

            <!-- Result -->
            <template v-if="coverCompressResult">
                <div class="status-card"
                    :class="coverCompressResult.failed > 0 ? 'status-warning' : 'status-success'"
                    style="margin-top:8px;">
                    <div class="status-text">
                        <div class="status-detail">
                            ✓ {{ coverCompressResult.compressed }} {{ msgs.coverCompressTitle || '已压缩' }}
                            <span v-if="coverCompressResult.savedBytes > 0" style="margin-left:6px;color:var(--accent)">
                                ({{ msgs.savedLabel || '节省' }} {{ coverCompressResult.savedBytes > 1048576
                                    ? (coverCompressResult.savedBytes/1048576).toFixed(1)+' MB'
                                    : Math.round(coverCompressResult.savedBytes/1024)+' KB' }})
                            </span>
                            · {{ coverCompressResult.skipped }} {{ msgs.skippedLabel || '已跳过' }}
                            <span v-if="coverCompressResult.failed > 0" style="color:var(--error);margin-left:4px;">
                                · ✗ {{ coverCompressResult.failed }} {{ msgs.failedLabel || '失败' }}
                            </span>
                        </div>
                    </div>
                </div>
            </template>
        </div>
        </section>
        <section class="tool-panel panel-wide" v-if="experimentalEnabled">
        <!-- Anti-piracy (visible only when experimental features enabled) -->
        <template v-if="experimentalEnabled">
            <div class="library-section" style="margin-top: 12px; border: 1px dashed var(--warning); border-radius: var(--radius-sm); padding: 12px;">
                <div class="library-section-title" style="color: var(--warning);">防偷歌模式</div>
                <div class="library-stats" style="margin-bottom:8px;">对所选范围内的音频文件批量操作 ID3 标签，操作前自动备份</div>

                <!-- Scope selector -->
                <div class="playlist-chips" style="margin-bottom:8px;">
                    <span class="playlist-chip no-drag" :class="{ active: apScope === 'file' }" @click="apScope = 'file'">单个文件</span>
                    <span class="playlist-chip no-drag" :class="{ active: apScope === 'folder' }" @click="apScope = 'folder'">文件夹</span>
                    <span class="playlist-chip no-drag" :class="{ active: apScope === 'playlist' }" @click="apScope = 'playlist'">Playlist</span>
                </div>

                <!-- File picker -->
                <button v-if="apScope === 'file'" class="btn btn-secondary no-drag" @click="handlePickApFile" :disabled="apBusy">
                    {{ apFile ? apFile.split(/[/\\]/).pop() : '选择单个音频文件' }}
                </button>

                <!-- Folder picker -->
                <button v-if="apScope === 'folder'" class="btn btn-secondary no-drag" @click="handlePickApDir" :disabled="apBusy">
                    {{ apDir || '选择音乐目录' }}
                </button>

                <!-- Playlist selector -->
                <select v-if="apScope === 'playlist'" class="text-input no-drag"
                    v-model="apPlaylistId" :disabled="apBusy" style="margin-bottom:4px;">
                    <option :value="-1" disabled>选择播放列表</option>
                    <option v-for="pl in playlists" :key="pl.id" :value="pl.id">
                        {{ pl.title }} ({{ pl.count }})
                    </option>
                </select>

                <!-- Action buttons -->
                <template v-if="apIsReady()">
                    <div class="drive-select-row" style="margin-top: 8px;">
                        <button class="btn btn-restore-all no-drag" style="flex:1" @click="handleAntiPiracyV1" :disabled="apBusy">
                            <span v-if="apBusy" class="loading-spinner"></span>
                            v1 清除标签
                        </button>
                        <button class="btn btn-restore-all no-drag" style="flex:1" @click="handleAntiPiracyV2" :disabled="apBusy">
                            <span v-if="apBusy" class="loading-spinner"></span>
                            v2 打乱标签
                        </button>
                    </div>
                    <button class="btn btn-secondary no-drag" style="margin-top: 6px;width:100%;" @click="handleAntiPiracyRestore" :disabled="apBusy">
                        恢复备份
                    </button>
                </template>
            </div>
        </template>
        </section>
    </div>

    <!-- DEVICES -->
    <div class="content tools-content" v-show="activeTab === 'devices'">
        <section class="tool-panel">
        <!-- USB Unlock -->
        <div class="library-section" style="margin-top: 12px;">
            <div class="library-section-title">{{ msgs.usbUnlockTitle || 'USB Unlock' }}</div>
            <div class="library-stats">
                {{ msgs.usbUnlockDescription || 'For Engine DJ install/uninstall/update file lock issues' }}
            </div>
            <button
                class="btn btn-primary no-drag"
                @click="handleUSBUnlock"
                :disabled="usbBusy || !usbDrive"
            >
                <span v-if="usbBusy" class="loading-spinner"></span>
                {{ usbDrive ? ((msgs.usbUnlockButton || 'Unlock') + ' ' + usbDrive) : (msgs.usbNoDevice || 'No USB with Engine Library detected') }}
            </button>
            <button
                v-if="usbDrive"
                class="btn btn-secondary no-drag"
                @click="handleUSBScan"
                :disabled="usbBusy"
            >
                {{ msgs.usbScanButton || 'Scan blocking processes' }}
            </button>
            <div v-if="usbBlockers.length > 0" class="info-card" style="margin-top: 6px;">
                <div v-for="p in usbBlockers" :key="p.pid" class="info-row">
                    <span class="info-label">{{ p.name }}</span>
                    <span class="info-value">PID {{ p.pid }}</span>
                </div>
            </div>
        </div>
        </section>
        <section class="tool-panel">
        <!-- MIDI 2.0 Toggle -->
        <div class="library-section" style="margin-top: 12px;">
            <div class="library-section-title">{{ msgs.midi2Title || 'MIDI 2.0 Control' }}</div>
            <div class="library-stats">
                {{ msgs.midi2Description || 'Disable Windows 11 MIDI 2.0 features while preserving MIDI 1.0 service' }}
            </div>
            <div v-if="midi2Status === 'enabled'" class="status-card status-warning" style="margin-bottom: 8px;">
                <div class="status-text">
                    <div class="status-detail">{{ msgs.midi2Enabled || 'MIDI 2.0 is currently enabled' }}</div>
                </div>
            </div>
            <div v-if="midi2Status === 'disabled'" class="status-card status-success" style="margin-bottom: 8px;">
                <div class="status-text">
                    <div class="status-detail">{{ msgs.midi2Disabled || 'MIDI 2.0 is currently disabled' }}</div>
                </div>
            </div>
            <div v-if="midi2Status === 'unavailable'" class="status-card status-error" style="margin-bottom: 8px;">
                <div class="status-text">
                    <div class="status-detail">{{ msgs.midi2Unavailable || 'MIDI 2.0 services not found on this system' }}</div>
                </div>
            </div>
            <div class="drive-select-row">
                <button
                    v-if="midi2Status === 'enabled'"
                    class="btn btn-primary no-drag"
                    style="flex:1"
                    @click="handleMIDI2Disable"
                    :disabled="midi2Busy"
                >
                    <span v-if="midi2Busy" class="loading-spinner"></span>
                    {{ midi2Busy ? (msgs.midi2Disabling || 'Disabling...') : (msgs.midi2DisableButton || 'Disable MIDI 2.0') }}
                </button>
                <button
                    v-if="midi2Status === 'disabled'"
                    class="btn btn-secondary no-drag"
                    style="flex:1"
                    @click="handleMIDI2Enable"
                    :disabled="midi2Busy"
                >
                    <span v-if="midi2Busy" class="loading-spinner"></span>
                    {{ midi2Busy ? (msgs.midi2Enabling || 'Enabling...') : (msgs.midi2EnableButton || 'Enable MIDI 2.0') }}
                </button>
                <button
                    v-if="midi2Status === 'unavailable'"
                    class="btn btn-secondary no-drag"
                    style="flex:1"
                    disabled
                >
                    Not Available
                </button>
            </div>
        </div>
        </section>
        <section class="tool-panel panel-wide">
        <!-- MSI Cleanup -->
        <div class="library-section">
            <div class="library-section-title">{{ msgs.msiCleanupTitle || 'MSI Cleanup' }}</div>
            <div class="library-stats">
                {{ msgs.msiCleanupDescription || 'For Engine DJ install/uninstall/update residual file issues' }}
            </div>
            <button class="btn btn-primary no-drag" @click="handleScanMSI" :disabled="msiBusy">
                <span v-if="msiBusy" class="loading-spinner"></span>
                {{ msiBusy ? (msgs.msiScanning || 'Scanning...') : (msgs.msiCleanupButton || 'Scan MSI Residuals') }}
            </button>
        </div>

        <div v-if="msiScanned && msiOrphans.length === 0" class="status-card status-success">
            <div class="status-text">
                <div class="status-detail">{{ msgs.msiNoOrphans || 'No MSI orphans found' }}</div>
            </div>
        </div>

        <div v-if="msiOrphans.length > 0" class="library-section" style="margin-top: 12px;">
            <div class="library-section-title">{{ msgs.msiFoundOrphans ? msgs.msiFoundOrphans.replace('%d', msiOrphans.length) : `Found ${msiOrphans.length} orphans` }}</div>
            <label
                v-for="o in msiOrphans"
                :key="o.productCode"
                class="msi-item no-drag"
            >
                <input
                    type="checkbox"
                    :checked="msiSelected.includes(o.productCode)"
                    @change="toggleMSI(o.productCode)"
                    :disabled="msiBusy"
                />
                <span class="msi-name">{{ o.displayName || o.productCode }}</span>
            </label>
            <button
                class="btn btn-restore-all no-drag"
                @click="handleCleanMSI"
                :disabled="msiBusy || msiSelected.length === 0"
            >
                {{ msiBusy ? (msgs.msiCleaning || 'Cleaning...') : (msgs.msiCleanupButton || 'MSI Cleanup') }}
            </button>
        </div>
        </section>
    </div>

    <!-- DIAGNOSTICS -->
    <div class="content tools-content" v-show="activeTab === 'diagnostics'">
        <section class="tool-panel panel-wide">
        <!-- Log Analysis -->
        <div class="library-section" style="margin-top: 12px;">
            <div class="library-section-title">{{ msgs.logAnalysisTitle || 'Log Analysis' }}</div>
            <div class="library-stats">
                {{ msgs.logAnalysisDescription || 'Analyze Engine DJ log files for errors and warnings' }}
            </div>
            <div class="drive-select-row">
                <button class="btn btn-primary no-drag" style="flex:1" @click="handleAnalyzeLogs" :disabled="logBusy">
                    <span v-if="logBusy" class="loading-spinner"></span>
                    {{ logBusy ? (msgs.logAnalyzing || 'Analyzing...') : (msgs.logAnalyzeButton || 'Analyze Logs') }}
                </button>
                <button class="btn btn-secondary no-drag" style="flex:1" @click="handleOpenLogsDir">
                    {{ msgs.logOpenDir || 'Open Log Folder' }}
                </button>
            </div>
        </div>

        <template v-if="logStats">
            <div class="info-card">
                <div class="info-row">
                    <span class="info-label">{{ msgs.logTotalFiles || 'Log Files' }}</span>
                    <span class="info-value">{{ logStats.totalFiles }}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">{{ msgs.logTotalLines || 'Total Lines' }}</span>
                    <span class="info-value">{{ logStats.totalLines }}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">{{ msgs.logLatestFile || 'Latest Log' }}</span>
                    <span class="info-value">{{ logStats.latestLog }}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">Info</span>
                    <span class="info-value" style="color: var(--text-secondary)">{{ logStats.infoCount }}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">Warnings</span>
                    <span class="info-value" style="color: var(--warning)">{{ logStats.warningCount }}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">Errors</span>
                    <span class="info-value" style="color: var(--error)">{{ logStats.errorCount }}</span>
                </div>
            </div>

            <div v-if="logStats.topErrors && logStats.topErrors.length > 0" class="library-section" style="margin-top: 8px;">
                <div class="library-section-title" style="color: var(--error)">{{ msgs.logTopErrors || 'Top Errors' }}</div>
                <div v-for="(e, i) in logStats.topErrors" :key="'err'+i" class="log-stat-item">
                    <span class="log-stat-count">×{{ e.count }}</span>
                    <span class="log-stat-msg">{{ e.message }}</span>
                </div>
            </div>

            <div v-if="logStats.topWarnings && logStats.topWarnings.length > 0" class="library-section" style="margin-top: 8px;">
                <div class="library-section-title" style="color: var(--warning)">{{ msgs.logTopWarnings || 'Top Warnings' }}</div>
                <div v-for="(w, i) in logStats.topWarnings" :key="'warn'+i" class="log-stat-item">
                    <span class="log-stat-count">×{{ w.count }}</span>
                    <span class="log-stat-msg">{{ w.message }}</span>
                </div>
            </div>
        </template>
        </section>
        <section class="tool-panel panel-wide">
        <!-- Cache Cleanup -->
        <div class="library-section" style="margin-top: 12px;">
            <div class="library-section-title">{{ msgs.cacheCleanTitle || 'Cache Cleanup' }}</div>
            <div class="library-stats">
                {{ msgs.cacheCleanDescription || 'Clear Engine DJ UI cache to fix display glitches' }}
            </div>
            <div v-if="cacheSizeText" class="library-stats" style="color: var(--warning)">{{ cacheSizeText }}</div>
            <button class="btn btn-primary no-drag" @click="handleClearCache" :disabled="cacheBusy">
                <span v-if="cacheBusy" class="loading-spinner"></span>
                {{ cacheBusy ? (msgs.cacheClearing || 'Clearing...') : (msgs.cacheCleanButton || 'Clear Cache') }}
            </button>
        </div>
        </section>
        <section class="tool-panel panel-wide compatibility-records">
            <div class="compatibility-records-heading">
                <div class="panel-heading">{{ compatibilityText.title }} <span class="compatibility-count">{{ compatibilityRecords.length }}</span></div>
                <div class="compatibility-record-actions">
                    <button type="button" class="btn btn-secondary no-drag" :disabled="compatibilityBusy" @click="refreshCompatibilityRecords">{{ compatibilityText.refresh }}</button>
                    <button type="button" class="btn btn-restore-all no-drag" :disabled="compatibilityBusy || !compatibilityRecords.length" @click="clearCompatibilityRecords()">{{ compatibilityText.clearAll }}</button>
                </div>
            </div>
            <p class="setting-description">{{ compatibilityText.description }}</p>
            <p v-if="!compatibilityRecords.length" class="compatibility-empty">{{ compatibilityText.empty }}</p>
            <div v-for="record in compatibilityRecords" :key="record.key" class="compatibility-record">
                <div class="compatibility-records-heading">
                    <div>
                        <div class="setting-label">Engine DJ · {{ (record.hashes?.['Engine DJ.exe'] || record.key).slice(0, 12) }}</div>
                        <div class="setting-description">{{ compatibilityText.checked }}: {{ formatCheckTime(record.checkedAt) }} · {{ compatibilityText.callers }}: {{ record.callers?.length || 0 }}</div>
                    </div>
                    <button type="button" class="btn btn-restore-all no-drag" :disabled="compatibilityBusy" @click="clearCompatibilityRecords(record.key)">{{ compatibilityText.clear }}</button>
                </div>
                <details class="compatibility-details">
                    <summary>{{ compatibilityText.details }}</summary>
                    <dl>
                        <template v-for="file in ['Engine DJ.exe', 'Qt6Core.dll', 'Qt6Network.dll']" :key="file">
                            <dt>{{ file }} · SHA-256</dt><dd><code>{{ record.hashes?.[file] || '—' }}</code></dd>
                        </template>
                        <dt>{{ compatibilityText.validator }}</dt><dd><code>{{ record.validator }}</code></dd>
                        <dt>{{ compatibilityText.callers }} · RVA</dt><dd><code>{{ (record.callers || []).map(value => '0x' + value.toString(16)).join(', ') }}</code></dd>
                    </dl>
                </details>
            </div>
        </section>
    </div>

    <!-- SETTINGS TAB -->
    <div class="content settings-content" v-show="activeTab === 'settings'">
        <section class="appearance-section">
            <div class="panel-heading">{{ uiText.appearance }}</div>
            <p class="setting-description">{{ uiText.appearanceDesc }}</p>
            <div class="theme-choices">
                <button v-for="theme in ['orange', 'dark', 'light']" :key="theme" type="button"
                    class="theme-option no-drag" :class="{ selected: uiTheme === theme }"
                    :aria-pressed="uiTheme === theme" @click="uiTheme = theme">
                    <span class="theme-preview" :class="'preview-' + theme" aria-hidden="true">
                        <span class="preview-sidebar"></span>
                        <span class="preview-main"><span></span><span></span><i></i></span>
                    </span>
                    <span class="theme-option-label">{{ uiText[theme] }}<span v-if="uiTheme === theme" aria-hidden="true">✓</span></span>
                </button>
            </div>
            <div class="setting-row">
                <div><div class="setting-label">{{ uiText.square }}</div><p class="setting-description">{{ uiText.squareDesc }}</p></div>
                <button type="button" class="setting-switch no-drag" role="switch" :aria-label="uiText.square"
                    :aria-checked="squareCorners" :class="{ on: squareCorners }" @click="squareCorners = !squareCorners"><span></span></button>
            </div>
            <p class="setting-footnote">{{ uiText.session }}</p>
        </section>
        <!-- Language -->
        <div class="library-section">
            <div class="library-section-title">{{ msgs.settingsLanguageLabel || '显示语言' }}</div>
            <div class="drive-grid no-drag" style="flex-wrap:wrap;">
                <button v-for="lang in languages" :key="lang.code"
                    class="drive-chip"
                    :class="{ active: currentLang === lang.code }"
                    @click="changeLanguage(lang.code)">
                    <span class="drive-letter" style="font-size:12px;">{{ lang.native }}</span>
                </button>
            </div>
        </div>

        <!-- Window -->
        <div class="library-section" style="margin-top:12px;">
            <button type="button" class="always-on-top-setting no-drag" role="switch"
                :aria-checked="alwaysOnTop" @click="toggleAlwaysOnTop">
                <span class="toggle-switch" :class="{ on: alwaysOnTop }" aria-hidden="true">
                    <span class="toggle-knob"></span>
                </span>
                <span>{{ msgs.settingsAlwaysOnTop || '始终置顶' }}</span>
            </button>
        </div>
        <!-- Experimental Features -->
        <div class="library-section" style="margin-top:12px;">
            <div class="library-section-title">{{ msgs.settingsExperimentalLabel || '实验性功能' }}</div>
            <div class="library-stats" style="margin-bottom:8px;">{{ msgs.settingsExperimentalDesc || '解锁隐藏的开发者功能，请谨慎使用。' }}</div>

            <!-- Toggle -->
            <label class="no-drag" style="display:flex;align-items:center;gap:10px;cursor:pointer;">
                <div class="toggle-switch" :class="{ on: experimentalEnabled }" @click="handleExperimentalToggle">
                    <div class="toggle-knob"></div>
                </div>
                <span style="font-size:13px;">{{ experimentalEnabled ? (msgs.settingsExperimentalLabel || '已开启') : (msgs.settingsExperimentalLabel || '已关闭') }}</span>
            </label>

            <!-- Countdown confirm -->
            <div v-if="showExperimentalConfirm" class="info-card" style="margin-top:10px;border:1px solid var(--warning);">
                <div style="font-size:12px;color:var(--text-secondary);margin-bottom:8px;">
                    {{ msgs.settingsExperimentalConfirmMsg || '' }}
                </div>
                <div class="drive-select-row">
                    <button class="btn btn-restore-all no-drag" style="flex:1;"
                        :disabled="experimentalCountdown > 0"
                        @click="confirmExperimental">
                        {{ experimentalCountdown > 0
                            ? (experimentalCountdown + 's...')
                            : (msgs.settingsExperimentalConfirmBtn || '确认开启') }}
                    </button>
                    <button class="btn btn-secondary no-drag" style="flex:1;" @click="cancelExperimental">
                        {{ msgs.restoreButton || '取消' }}
                    </button>
                </div>
            </div>
        </div>

        <!-- About -->
        <div class="library-section" style="margin-top:12px;">
            <div class="library-section-title">{{ msgs.settingsAboutTitle || '关于' }}</div>
            <div class="info-card">
                <div class="info-row">
                    <span class="info-label">Engine Tools</span>
                    <span class="info-value">v{{ APP_VERSION_LABEL }}</span>
                </div>
                <div class="info-row">
                    <span class="info-label">{{ msgs.settingsContributors || '贡献者' }}</span>
                    <span class="info-value" style="color:var(--accent);cursor:pointer;" @click="openGitHub">@LaokeQwQ</span>
                </div>
                <div class="info-row">
                    <span class="info-label muted" style="font-size:11px;">{{ msgs.settingsAboutDesc || '非官方个人工具' }}</span>
                </div>
            </div>
        </div>
    </div>

    <div class="progress-wrapper">
        <div class="progress-bar" :class="{ active: showProgress }">
            <div class="progress-fill" :style="{ width: progress + '%' }"></div>
        </div>
    </div>

    </main>
    <section class="log-section" :class="{ collapsed: !consoleExpanded }">
        <button class="log-header no-drag" type="button" :aria-expanded="consoleExpanded" :aria-label="consoleExpanded ? uiText.collapse : uiText.expand" @click="consoleExpanded = !consoleExpanded">
            <span>{{ uiText.console }} <span class="log-count">{{ logs.length }}</span></span>
            <svg viewBox="0 0 20 20" width="15" height="15" fill="none" stroke="currentColor" stroke-width="1.6" :class="{ rotated: consoleExpanded }" aria-hidden="true"><path d="m5 8 5 5 5-5" /></svg>
        </button>
        <div class="log-body" v-show="consoleExpanded" ref="logContainer">
            <div v-if="!logs.length" class="log-empty">{{ uiText.emptyLog }}</div>
            <div v-for="(entry, i) in logs" :key="i" class="log-entry">{{ entry }}</div>
        </div>
    </section>
    </div>
</template>

<style>
/* Bar chart visualization */
.vis-chart {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 260px;
  overflow-y: auto;
}
.vis-bar-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
}
.vis-bar-name {
  width: 100px;
  min-width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary);
  text-align: right;
  font-size: 10px;
}
.vis-bar-outer {
  flex: 1;
  height: 14px;
  background: rgba(255,255,255,0.06);
  border-radius: 2px;
  overflow: hidden;
}
.vis-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--accent) 0%, color-mix(in srgb, var(--accent) 60%, transparent) 100%);
  border-radius: 2px;
  transition: width var(--motion-duration) var(--motion-ease);
}
.vis-bar-count {
  min-width: 24px;
  text-align: right;
  color: var(--text-secondary);
  font-size: 10px;
}

/* ID3 drop zone */
.id3-drop-zone {
  border: 1px dashed rgba(255,255,255,0.2);
  border-radius: var(--radius-sm, 6px);
  padding: 14px 12px;
  text-align: center;
  cursor: pointer;
  transition: border-color var(--motion-duration), background var(--motion-duration);
  font-size: 12px;
  color: var(--text-secondary);
}
.id3-drop-zone:hover,
.id3-drop-zone.drop-active {
  border-color: var(--accent);
  background: rgba(var(--accent-rgb, 99,179,237), 0.07);
  color: var(--accent);
}
.id3-drop-filename {
  color: var(--text-primary);
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
  max-width: 100%;
}
.id3-drop-hint {
  opacity: 0.7;
}

/* Toggle switch */
.toggle-switch {
  width: 40px;
  height: 22px;
  background: rgba(255,255,255,0.12);
  border-radius: 11px;
  position: relative;
  cursor: pointer;
  transition: background var(--motion-duration);
  flex-shrink: 0;
}
.toggle-switch.on {
  background: var(--accent);
}
.toggle-knob {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  background: #fff;
  border-radius: 50%;
  transition: transform var(--motion-duration);
  box-shadow: 0 1px 3px rgba(0,0,0,0.3);
}
.toggle-switch.on .toggle-knob {
  transform: translateX(18px);
}

.track-editing {
  background: rgba(var(--accent-rgb, 99,179,237), 0.12) !important;
  outline: 1px solid var(--accent);
}

.marquee-text {
  font-size: 11px;
  color: var(--text-secondary);
  opacity: 0;
  animation: marqueeFadeIn var(--motion-duration) ease forwards;
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-block;
  vertical-align: middle;
}

@keyframes marqueeFadeIn {
  from { opacity: 0; transform: translateY(3px); }
  to   { opacity: 1; transform: translateY(0); }
}
</style>

<style scoped>
.always-on-top-setting {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 0;
    border: 0;
    background: transparent;
    color: inherit;
    font: inherit;
    font-size: 13px;
    cursor: pointer;
}

.always-on-top-setting .toggle-switch,
.always-on-top-setting .toggle-knob {
    display: block;
}

.always-on-top-setting:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 5px;
    border-radius: 4px;
}
</style>
