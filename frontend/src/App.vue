<script setup>
import { ref, onMounted, computed } from 'vue'
import {
    GetStatus,
    FixUnicodeIssues,
    RestoreUnicodeFix,
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
import { EventsOn } from '../wailsjs/runtime/runtime.js'

const APP_VERSION = '1.7.0'

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
const marqueeTexts = ref([])
const currentMarquee = ref(0)

const activeTab = ref('status')

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
const toolsTabLoaded = ref(false)

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
const missingBusy = ref(false)
const missingScanned = ref(false)

// BPM/Key sync state
const syncBusy = ref(false)
const syncResult = ref(null)
const syncPreviewTracks = ref([])
const syncPreviewLoaded = ref(false)

// Cover compression state
const coverCompressBusy = ref(false)
const coverCompressResult = ref(null)
const coverCompressPlaylistId = ref(-1) // -1 = entire library

// Anti-piracy easter egg state (controlled by experimental toggle)
const apDir = ref('')
const apBusy = ref(false)

// Experimental features state
const experimentalEnabled = ref(localStorage.getItem('experimentalEnabled') === 'true')
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
const selectedPlaylist = ref(null)
const playlistTracks = ref([])
const playlistLoading = ref(false)

// Toast notification
const toastMsg = ref('')
const toastVisible = ref(false)
let toastTimer = null

function showToast(msg) {
    toastMsg.value = msg
    toastVisible.value = true
    if (toastTimer) clearTimeout(toastTimer)
    toastTimer = setTimeout(() => { toastVisible.value = false }, 3000)
}

function showErrorToast(msg) {
    toastMsg.value = '❌ ' + msg
    toastVisible.value = true
    if (toastTimer) clearTimeout(toastTimer)
    toastTimer = setTimeout(() => { toastVisible.value = false }, 4000)
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

function addLog(entry) {
    logs.value.push(entry)
    requestAnimationFrame(() => {
        if (logContainer.value) {
            logContainer.value.scrollTop = logContainer.value.scrollHeight
        }
    })
}

async function loadMessages() {
    msgs.value = await GetMessages()
    languages.value = await GetAvailableLanguages()
    updateMarqueeTexts()
}

function updateMarqueeTexts() {
    const m = msgs.value
    if (!m) return
    marqueeTexts.value = [
        `v${APP_VERSION}`,
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
        await HandleUTF8AlreadyEnabled()
        return
    }

    fixing.value = true
    showProgress.value = true
    progress.value = 0

    try {
        const result = await FixUnicodeIssues()
        if (result === 'ok') {
            manifestConfigured.value = true
        }
    } catch (e) {
        addLog('Error: ' + e)
    }

    fixing.value = false
    setTimeout(() => {
        showProgress.value = false
    }, 1000)
    showToast(msgs.value.fixComplete || 'Fix complete')
    await detectStatus()
}

async function handleRestore() {
    fixing.value = true
    showProgress.value = true
    progress.value = 0

    try {
        await RestoreUnicodeFix()
    } catch (e) {
        addLog('Error: ' + e)
    }

    fixing.value = false
    setTimeout(() => {
        showProgress.value = false
    }, 1000)
    showToast(msgs.value.restoreComplete || 'Restore complete')
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
        await RestoreDatabase(selectedBackup.value)
        showToast(msgs.value.dbRestoreComplete || 'Restore complete')
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
        missingTracks.value = await ScanMissingTracks()
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
    if (missingSelected.value.length === 0) return
    const ok = window.confirm(`确认从数据库删除 ${missingSelected.value.length} 条缺失记录？此操作不可撤销。`)
    if (!ok) return
    missingBusy.value = true
    try {
        await RemoveMissingTracks(missingSelected.value)
        showToast(`已删除 ${missingSelected.value.length} 条记录`)
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
        syncPreviewTracks.value = await GetSyncableTracks()
        syncPreviewLoaded.value = true
    } catch (e) {
        addLog('Error: ' + e)
    }
}

async function handleSyncBPMKey() {
    if (!syncPreviewLoaded.value) return
    const ok = window.confirm(`将向 ${syncPreviewTracks.value.length} 个文件写回 BPM 和 Key 标签。确认继续？`)
    if (!ok) return
    syncBusy.value = true
    syncResult.value = null
    try {
        syncResult.value = await SyncBPMKeyToTags()
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
    const scope = coverCompressPlaylistId.value < 0 ? (msgs.value.coverCompressAllLibrary || '全部曲库') : `播放列表 ${coverCompressPlaylistId.value}`
    const ok = window.confirm(`将压缩 ${scope} 中所有大于 1 MB 的封面至 1 MB 以内。此操作会修改音频文件的 ID3 标签。\n\n确认继续？`)
    if (!ok) return
    coverCompressBusy.value = true
    coverCompressResult.value = null
    try {
        coverCompressResult.value = await CompressCovers(coverCompressPlaylistId.value, 1024)
        const r = coverCompressResult.value
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
        localStorage.setItem('experimentalEnabled', 'false')
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
        await LogExperimentalEnabled()
        experimentalEnabled.value = true
        localStorage.setItem('experimentalEnabled', 'true')
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

async function handleAntiPiracyV1() {
    if (!apDir.value) return
    const ok = window.confirm('⚠️ 防偷歌 v1：将清除目录下所有音频文件的ID3标签（操作前自动备份）。\n\n确认执行？')
    if (!ok) return
    apBusy.value = true
    try {
        await ID3AntiPiracyV1(apDir.value)
    } catch (e) {
        addLog('Error: ' + e)
    }
    apBusy.value = false
}

async function handleAntiPiracyV2() {
    if (!apDir.value) return
    const ok = window.confirm('⚠️ 防偷歌 v2：将随机打乱目录下所有音频文件的ID3标签（操作前自动备份）。\n\n确认执行？')
    if (!ok) return
    apBusy.value = true
    try {
        await ID3AntiPiracyV2(apDir.value)
    } catch (e) {
        addLog('Error: ' + e)
    }
    apBusy.value = false
}

async function handleAntiPiracyRestore() {
    if (!apDir.value) return
    apBusy.value = true
    try {
        await ID3AntiPiracyRestore(apDir.value)
    } catch (e) {
        addLog('Error: ' + e)
    }
    apBusy.value = false
}

async function loadPlaylists() {
    playlistLoading.value = true
    try {
        playlists.value = await ListPlaylists()
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
        await CleanMSIOrphans(msiSelected.value)
        await handleScanMSI()
    } catch (e) {
        addLog('Error: ' + e)
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
    await switchTab('tools')
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
    if (tab === 'database' && !dbTabLoaded.value) {
        await loadDrives()
        await refreshBackups()
        await loadPlaylists()
        dbTabLoaded.value = true
    }
    if (tab === 'tools' && !toolsTabLoaded.value) {
        await checkUSBDrive()
        await checkMIDI2Status()
        // Load playlists for Cover Compression scope selector
        if (playlists.value.length === 0) await loadPlaylists()
        toolsTabLoaded.value = true
    }
}

async function openGitHub() {
    await OpenRepository()
}

async function changeLanguage(lang) {
    currentLang.value = lang
    await SetLanguage(lang)
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
    await loadMessages()

    EventsOn('log', (entry) => { addLog(entry) })
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
</script>

<template>
    <!-- Toast notification -->
    <div class="toast" :class="{ visible: toastVisible }">{{ toastMsg }}</div>

    <div class="header wails-drag">
        <div class="header-left">
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
        </div>
    </div>

    <div class="tabs no-drag">
        <button class="tab" :class="{ active: activeTab === 'status' }" @click="switchTab('status')">
            {{ msgs.tabStatus || 'Status' }}
        </button>
        <button class="tab" :class="{ active: activeTab === 'database' }" @click="switchTab('database')">
            {{ msgs.tabDatabase || 'Database' }}
        </button>
        <button class="tab" :class="{ active: activeTab === 'tools' }" @click="switchTab('tools')">
            {{ msgs.tabTools || 'Tools' }}
        </button>
        <button class="tab" :class="{ active: activeTab === 'settings' }" @click="switchTab('settings')">
            {{ msgs.tabSettings || 'Settings' }}
        </button>
    </div>

    <!-- STATUS TAB -->
    <div class="content" v-show="activeTab === 'status'">
        <div class="info-card">
            <div class="info-row" v-if="windowsVersion">
                <span class="info-label">{{ msgs.windowsVersionLabel || 'Windows' }}</span>
                <span class="info-value">{{ windowsVersion }}</span>
            </div>
            <div class="info-row">
                <span class="info-label">{{ msgs.installPathLabel || 'Path' }}</span>
                <span class="info-value-group">
                    <span class="info-value" :class="{ muted: !installPath }">{{ pathDisplay }}</span>
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
                <span class="info-value" :class="{ muted: !engineVersion }">{{ versionDisplay }}</span>
            </div>
        </div>

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

        <div v-if="stemsDetected" class="status-card status-success status-card-row">
            <div class="status-text">
                <div class="status-title">{{ msgs.stemsStatusLabel || 'STEM Processor' }}</div>
                <div class="status-detail">{{ stemsStatusText }}</div>
            </div>
            <span
                class="folder-btn no-drag"
                :title="msgs.stemsStatusLabel || 'STEM Processor'"
                @click="openStemsDir"
            >
                <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
            </span>
        </div>

        <div class="actions">
            <button
                class="btn btn-primary no-drag"
                @click="handleFix"
                :disabled="fixing || !installPath"
            >
                <span v-if="fixing" class="loading-spinner"></span>
                {{ fixing ? (msgs.progressFixing || 'Fixing...') : (msgs.fixButton || 'Fix Unicode Character Encoding Issues') }}
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
    </div>

    <!-- DATABASE TAB -->
    <div class="content" v-show="activeTab === 'database'">
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

            <!-- 点击曲目将跳转到 Tools > ID3 编辑器 -->
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
                    <span class="playlist-chip no-drag" :class="{ active: playStatsTab === 'most' }" @click="playStatsTab = 'most'">{{ msgs.playHistoryMostPlayed || '最多播放' }}</span>
                    <span class="playlist-chip no-drag" :class="{ active: playStatsTab === 'recent' }" @click="playStatsTab = 'recent'">{{ msgs.playHistoryRecent || '最近播放' }}</span>
                    <span class="playlist-chip no-drag" :class="{ active: playStatsTab === 'never' }" @click="playStatsTab = 'never'">{{ msgs.playHistoryNeverPlayed || '从未播放' }}</span>
                </div>

                <!-- Most Played: bar chart -->
                <div v-if="playStatsTab === 'most'" class="vis-chart" style="margin-top:8px;">
                    <div v-if="playStats.mostPlayed.length === 0" class="library-stats">{{ msgs.dbNoneFound || '暂无数据' }}</div>
                    <div v-for="t in playStats.mostPlayed" :key="t.id" class="vis-bar-row">
                        <div class="vis-bar-name">{{ t.title || t.artist || '—' }}</div>
                        <div class="vis-bar-outer">
                            <div class="vis-bar-fill"
                                :style="{ width: Math.max(4, (t.playCount / maxPlayCount * 100)) + '%' }">
                            </div>
                        </div>
                        <span class="vis-bar-count">×{{ t.playCount }}</span>
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

    <!-- TOOLS TAB -->
    <div class="content" v-show="activeTab === 'tools'">
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

        <!-- Anti-piracy (visible only when experimental features enabled) -->
        <template v-if="experimentalEnabled">
            <div class="library-section" style="margin-top: 12px; border: 1px dashed var(--warning); border-radius: var(--radius-sm); padding: 12px;">
                <div class="library-section-title" style="color: var(--warning);">防偷歌模式</div>
                <div class="library-stats">对指定目录下的所有音频文件批量操作ID3标签，操作前自动备份</div>
                <button class="btn btn-secondary no-drag" @click="handlePickApDir" :disabled="apBusy">
                    {{ apDir || '选择音乐目录' }}
                </button>
                <template v-if="apDir">
                    <div class="drive-select-row" style="margin-top: 8px;">
                        <button class="btn btn-restore-all no-drag" style="flex:1" @click="handleAntiPiracyV1" :disabled="apBusy">
                            <span v-if="apBusy" class="loading-spinner"></span>
                            v1 清除全部标签
                        </button>
                        <button class="btn btn-restore-all no-drag" style="flex:1" @click="handleAntiPiracyV2" :disabled="apBusy">
                            <span v-if="apBusy" class="loading-spinner"></span>
                            v2 随机打乱标签
                        </button>
                    </div>
                    <button class="btn btn-secondary no-drag" style="margin-top: 6px;" @click="handleAntiPiracyRestore" :disabled="apBusy">
                        恢复备份
                    </button>
                </template>
            </div>
        </template>

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

        <!-- Cover Art Compression -->
        <div class="library-section" style="margin-top: 12px;">
            <div class="library-section-title">{{ msgs.coverCompressTitle || 'Cover Art Compression' }}</div>

            <!-- Tips -->
            <div class="library-stats" style="border-left:3px solid var(--warning);padding-left:10px;margin-bottom:10px;">
                {{ msgs.coverCompressTip || 'Compressing covers to ≤1 MB improves Engine OS USB load performance.' }}
            </div>

            <!-- Scope selector -->
            <div class="drive-select-row" style="margin-bottom:8px;">
                <button class="drive-chip no-drag"
                    :class="{ active: coverCompressPlaylistId === -1 }"
                    @click="coverCompressPlaylistId = -1"
                    :disabled="coverCompressBusy">
                    <span class="drive-letter" style="font-size:11px;">All</span>
                    <span class="drive-count">{{ msgs.coverCompressAllLibrary || 'Entire Library' }}</span>
                </button>
                <button v-for="pl in playlists" :key="pl.id"
                    class="drive-chip no-drag"
                    :class="{ active: coverCompressPlaylistId === pl.id }"
                    @click="coverCompressPlaylistId = pl.id"
                    :disabled="coverCompressBusy"
                    style="margin-left:4px;">
                    <span class="drive-letter" style="font-size:11px;max-width:80px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">{{ pl.title }}</span>
                    <span class="drive-count">{{ pl.count }}</span>
                </button>
            </div>

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
    </div>

    <!-- SETTINGS TAB -->
    <div class="content" v-show="activeTab === 'settings'">
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
                    <span class="info-value">v{{ APP_VERSION }}</span>
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

    <div class="log-section">
        <div class="log-header">Console</div>
        <div class="log-body" ref="logContainer">
            <div v-for="(entry, i) in logs" :key="i" class="log-entry">{{ entry }}</div>
        </div>
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
  transition: width 0.6s cubic-bezier(.4,0,.2,1);
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
  transition: border-color 0.15s, background 0.15s;
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
  transition: background 0.2s;
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
  transition: transform 0.2s;
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
  animation: marqueeFadeIn 0.4s ease forwards;
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
