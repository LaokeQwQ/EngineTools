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
        <button class="tab" :class="{ active: activeTab === 'metadata' }" @click="switchTab('metadata')">
            {{ msgs.metadataEditorTitle || 'Metadata' }}
        </button>
        <button class="tab" :class="{ active: activeTab === 'settings' }" @click="switchTab('settings')">
            {{ msgs.tabSettings || 'Settings' }}
        </button>
    </div>

    <!-- STATUS TAB -->
    <StatusView
    :activeTab="activeTab"
    :adminStatusText="adminStatusText"
    :engineVersion="engineVersion"
    :fixing="fixing"
    :handleFix="handleFix"
    :handleOpenRegionSettings="handleOpenRegionSettings"
    :handleRestore="handleRestore"
    :info="info"
    :installPath="installPath"
    :isAdmin="isAdmin"
    :loading="loading"
    :manifestConfigured="manifestConfigured"
    :manifestStatusText="manifestStatusText"
    :msgs="msgs"
    :openInstallDir="openInstallDir"
    :openStemsDir="openStemsDir"
    :path="path"
    :pathDisplay="pathDisplay"
    :status="status"
    :stemsDetected="stemsDetected"
    :stemsStatusText="stemsStatusText"
    :utf8Enabled="utf8Enabled"
    :utf8StatusText="utf8StatusText"
    :versionDisplay="versionDisplay"
    :windowsVersion="windowsVersion"
    />

    <!-- DATABASE TAB -->
    <LibraryView
    :activeTab="activeTab"
    :backupNote="backupNote"
    :backups="backups"
    :dbBusy="dbBusy"
    :dbDetected="dbDetected"
    :dbPathDisplay="dbPathDisplay"
    :drives="drives"
    :editingTrack="editingTrack"
    :formatBytes="formatBytes"
    :formatDuration="formatDuration"
    :formatTime="formatTime"
    :handleBackup="handleBackup"
    :handleClickDrive="handleClickDrive"
    :handleEditTrackFromPlaylist="handleEditTrackFromPlaylist"
    :handleLibraryStats="handleLibraryStats"
    :handleOptimize="handleOptimize"
    :handlePlayStats="handlePlayStats"
    :handleRemoveMissing="handleRemoveMissing"
    :handleRepair="handleRepair"
    :handleRestoreDB="handleRestoreDB"
    :handleScanMissing="handleScanMissing"
    :handleSelectPlaylist="handleSelectPlaylist"
    :info="info"
    :libraryStats="libraryStats"
    :libraryStatsBusy="libraryStatsBusy"
    :loading="loading"
    :missingBusy="missingBusy"
    :missingScanned="missingScanned"
    :missingSelected="missingSelected"
    :missingTracks="missingTracks"
    :msgs="msgs"
    :openDBDir="openDBDir"
    :path="path"
    :playStats="playStats"
    :playStatsBusy="playStatsBusy"
    :playStatsTab="playStatsTab"
    :playlistLoading="playlistLoading"
    :playlistTracks="playlistTracks"
    :playlists="playlists"
    :selectedBackup="selectedBackup"
    :selectedDrive="selectedDrive"
    :selectedPlaylist="selectedPlaylist"
    :status="status"
    :toggleAllMissing="toggleAllMissing"
    :toggleMissingTrack="toggleMissingTrack"
    @update:backupNote="val => backupNote = val"
    @update:selectedBackup="val => selectedBackup = val"
    @update:playStatsTab="val => playStatsTab = val"
    />

    <!-- TOOLS TAB -->
    <ToolsView
    :activeTab="activeTab"
    :apBusy="apBusy"
    :apDir="apDir"
    :apFile="apFile"
    :apIsReady="apIsReady"
    :apPlaylistId="apPlaylistId"
    :apScope="apScope"
    :cacheBusy="cacheBusy"
    :cacheSizeText="cacheSizeText"
    :coverCompressBusy="coverCompressBusy"
    :coverCompressPlaylistId="coverCompressPlaylistId"
    :coverCompressResult="coverCompressResult"
    :editingTrack="editingTrack"
    :experimentalEnabled="experimentalEnabled"
    :files="files"
    :handleAnalyzeLogs="handleAnalyzeLogs"
    :handleAntiPiracyRestore="handleAntiPiracyRestore"
    :handleAntiPiracyV1="handleAntiPiracyV1"
    :handleAntiPiracyV2="handleAntiPiracyV2"
    :handleClearCache="handleClearCache"
    :handleCleanMSI="handleCleanMSI"
    :handleCompressCovers="handleCompressCovers"
    :handleID3ClearAll="handleID3ClearAll"
    :handleID3ClearCover="handleID3ClearCover"
    :handleID3FileDrop="handleID3FileDrop"
    :handleID3Pick="handleID3Pick"
    :handleID3Save="handleID3Save"
    :handleID3SetCover="handleID3SetCover"
    :handleLoadSyncPreview="handleLoadSyncPreview"
    :handleOpenLogsDir="handleOpenLogsDir"
    :handlePickApDir="handlePickApDir"
    :handlePickApFile="handlePickApFile"
    :handleScanMSI="handleScanMSI"
    :handleSyncBPMKey="handleSyncBPMKey"
    :handleUSBScan="handleUSBScan"
    :handleUSBUnlock="handleUSBUnlock"
    :id3Album="id3Album"
    :id3Artist="id3Artist"
    :id3Busy="id3Busy"
    :id3Cover="id3Cover"
    :id3DragOver="id3DragOver"
    :id3File="id3File"
    :id3FileName="id3FileName"
    :id3Genre="id3Genre"
    :id3Title="id3Title"
    :id3Year="id3Year"
    :info="info"
    :loading="loading"
    :logBusy="logBusy"
    :logStats="logStats"
    :msgs="msgs"
    :msiBusy="msiBusy"
    :msiOrphans="msiOrphans"
    :msiScanned="msiScanned"
    :msiSelected="msiSelected"
    :playlists="playlists"
    :status="status"
    :syncBusy="syncBusy"
    :syncPreviewLoaded="syncPreviewLoaded"
    :syncPreviewTracks="syncPreviewTracks"
    :syncResult="syncResult"
    :toggleMSI="toggleMSI"
    :usbBlockers="usbBlockers"
    :usbBusy="usbBusy"
    :usbDrive="usbDrive"
    @update:id3Title="val => id3Title = val"
    @update:id3Artist="val => id3Artist = val"
    @update:id3Album="val => id3Album = val"
    @update:id3Year="val => id3Year = val"
    @update:id3Genre="val => id3Genre = val"
    @update:id3DragOver="val => id3DragOver = val"
    @update:id3File="val => id3File = val"
    @update:editingTrack="val => editingTrack = val"
    @update:apScope="val => apScope = val"
    @update:apPlaylistId="val => apPlaylistId = val"
    @update:coverCompressPlaylistId="val => coverCompressPlaylistId = val"
    />

    <MetadataEditorView
    :activeTab="activeTab"
    :initialPath="id3File"
    :msgs="msgs"
    @dirty-change="metadataDirty = $event"
    />

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
<script setup>
import { ref, onMounted, computed } from 'vue'
import MetadataEditorView from './views/MetadataEditorView.vue'
import StatusView from './views/StatusView.vue'
import LibraryView from './views/LibraryView.vue'
import ToolsView from './views/ToolsView.vue'
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

const APP_VERSION = '1.8.0'

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
const metadataDirty = ref(false)
// Compatibility aliases retained for the extracted child-view prop contract.
// The views now consume the focused values below, but keeping these defined
// avoids undefined bindings for older Wails payloads.
const info = ref(null)
const path = ref('')
const status = ref('')
const files = ref([])

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
const apFile = ref('')
const apPlaylistId = ref(-1)
const apScope = ref('folder') // 'file' | 'folder' | 'playlist'
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
    if (apScope.value === 'file') return apFile.value || ''
    return 'playlist_' + apPlaylistId.value
}

function apIsReady() {
    if (apScope.value === 'folder') return !!apDir.value
    if (apScope.value === 'file') return !!apFile.value
    return apPlaylistId.value >= 0
}

function requireOk(response, fallback) {
    if (response !== 'ok') throw new Error(fallback)
}

async function handleAntiPiracyV1() {
    if (!apIsReady()) return
    const ok = window.confirm('防偷歌 v1：清除所选范围内所有音频文件的文本标签（标题、艺术家、专辑等，操作前自动备份）。确认执行？')
    if (!ok) return
    apBusy.value = true
    try {
        let response
        if (apScope.value === 'folder') {
            response = await ID3AntiPiracyV1(apDir.value)
        } else if (apScope.value === 'file') {
            response = await ID3AntiPiracyV1Paths([apFile.value], apBackupKey())
        } else {
            response = await ID3AntiPiracyV1Playlist(apPlaylistId.value)
        }
        requireOk(response, 'v1 执行失败')
        showToast('v1 执行完成')
    } catch (e) {
        addLog('Error: ' + e)
    }
    apBusy.value = false
}

async function handleAntiPiracyV2() {
    if (!apIsReady()) return
    const ok = window.confirm('防偷歌 v2：随机打乱所选范围内所有音频文件的文本标签（标题、艺术家、专辑等，操作前自动备份）。确认执行？')
    if (!ok) return
    apBusy.value = true
    try {
        let response
        if (apScope.value === 'folder') {
            response = await ID3AntiPiracyV2(apDir.value)
        } else if (apScope.value === 'file') {
            response = await ID3AntiPiracyV2Paths([apFile.value], apBackupKey())
        } else {
            response = await ID3AntiPiracyV2Playlist(apPlaylistId.value)
        }
        requireOk(response, 'v2 执行失败')
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
        let response
        if (apScope.value === 'folder') {
            response = await ID3AntiPiracyRestore(apDir.value)
        } else if (apScope.value === 'file') {
            response = await ID3AntiPiracyRestorePaths(apBackupKey())
        } else {
            response = await ID3AntiPiracyRestorePlaylist(apPlaylistId.value)
        }
        requireOk(response, '恢复失败')
        showToast('恢复完成')
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
        const response = await ID3WriteTag(id3File.value, id3Title.value, id3Artist.value, id3Album.value, id3Year.value, id3Genre.value)
        requireOk(response, 'ID3 保存失败')
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
        const response = await ID3ClearCover(id3File.value)
        requireOk(response, '清除封面失败')
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
        const response = await ID3ClearAll(id3File.value)
        requireOk(response, '清除标签失败')
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


async function switchTab(tab) {
    if (activeTab.value === 'metadata' && tab !== 'metadata' && metadataDirty.value) {
        const discard = window.confirm(msgs.value.metadataUnsaved || 'You have unsaved metadata changes. Discard them?')
        if (!discard) return
    }
    activeTab.value = tab
    if (tab === 'database' && !dbTabLoaded.value) {
        await loadDrives()
        await refreshBackups()
        await loadPlaylists()
        dbTabLoaded.value = true
    }
    if (tab === 'tools' && !toolsTabLoaded.value) {
        await checkUSBDrive()
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
