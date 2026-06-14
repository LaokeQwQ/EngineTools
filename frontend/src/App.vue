<script setup>
import { ref, onMounted, computed } from 'vue'
import {
    GetStatus,
    FixCJKIssues,
    RestoreCJKFix,
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
    ScanMSIOrphans,
    CleanMSIOrphans,
    OpenInstallDir,
    OpenStemsDir,
    OpenDBDir,
    ListDrives,
    SelectDrive,
    ID3PickFile,
    ID3ReadTag,
    ID3WriteTag,
    ID3GetCover,
    ID3SetCover,
    ID3ClearCover,
    ID3ClearAll,
    USBUnlockAvailable,
    USBUnlockScan,
    USBUnlockKill,
    MIDI2Status,
    MIDI2Disable,
    MIDI2Enable,
    CheckForUpdates,
} from '../wailsjs/go/main/App.js'
import { EventsOn } from '../wailsjs/runtime/runtime.js'

const APP_VERSION = '1.5.0'

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
const id3Title = ref('')
const id3Artist = ref('')
const id3Album = ref('')
const id3Year = ref('')
const id3Genre = ref('')
const id3Cover = ref('')
const id3Busy = ref(false)

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
        const result = await FixCJKIssues()
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

    await detectStatus()
}

async function handleRestore() {
    fixing.value = true
    showProgress.value = true
    progress.value = 0

    try {
        await RestoreCJKFix()
    } catch (e) {
        addLog('Error: ' + e)
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
    } catch (e) {
        addLog('Error: ' + e)
    }
    dbBusy.value = false
}

async function handleRestoreDB() {
    if (!selectedBackup.value) return
    dbBusy.value = true
    try {
        await RestoreDatabase(selectedBackup.value)
    } catch (e) {
        addLog('Error: ' + e)
    }
    dbBusy.value = false
}

async function handleOptimize() {
    dbBusy.value = true
    try {
        await OptimizeDatabase()
    } catch (e) {
        addLog('Error: ' + e)
    }
    dbBusy.value = false
}

async function loadDrives() {
    try {
        drives.value = await ListDrives()
    } catch (e) {
        addLog('Error: ' + e)
        drives.value = []
    }
}

async function handleClickDrive(drive) {
    selectedDrive.value = drive
    dbBusy.value = true
    try {
        await SelectDrive(drive)
    } catch (e) {
        addLog('Error: ' + e)
    }
    dbBusy.value = false
    await detectStatus()
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

async function handleID3Pick() {
    const path = await ID3PickFile()
    if (!path) return
    id3File.value = path
    await loadID3(path)
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
    }
    midi2Busy.value = false
}

async function switchTab(tab) {
    activeTab.value = tab
    if (tab === 'database') {
        if (drives.value.length === 0) await loadDrives()
        if (backups.value.length === 0) await refreshBackups()
    }
    if (tab === 'tools') {
        await checkUSBDrive()
        await checkMIDI2Status()
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

    EventsOn('log', (entry) => {
        addLog(entry)
    })

    EventsOn('progress', (value) => {
        progress.value = value * 100
    })

    EventsOn('statusUpdate', async () => {
        await detectStatus()
    })

    await detectStatus()

    marqueeInterval = setInterval(() => {
        currentMarquee.value = (currentMarquee.value + 1) % marqueeTexts.value.length
    }, 4000)

    // Auto check for updates 1 second after startup
    setTimeout(() => {
        checkForUpdates()
    }, 1000)
})
</script>

<template>
    <div class="header wails-drag">
        <div class="header-left">
            <span class="app-title">{{ msgs.appTitle || 'Engine Tools' }}</span>
            <span class="marquee-text" :key="currentMarquee">{{ marqueeTexts[currentMarquee] }}</span>
        </div>
        <div class="header-right">
            <span v-if="updateAvailable" class="update-badge no-drag" @click="openUpdatePage" title="New version available!">
                🎉 Update
            </span>
            <span v-else-if="checkingUpdate" class="update-badge no-drag checking">
                ⏳
            </span>
            <span class="github-link no-drag" @click="openGitHub">
                <svg class="github-icon" viewBox="0 0 16 16" fill="currentColor" width="16" height="16"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>
            </span>
            <span class="author-handle no-drag" @click="openGitHub">@LaokeQwQ</span>
            <span class="update-check-btn no-drag" @click="checkForUpdates" title="Check for updates">
                🔄
            </span>
            <select class="lang-select no-drag" :value="currentLang" @change="changeLanguage($event.target.value)">
                <option v-for="lang in languages" :key="lang.code" :value="lang.code">{{ lang.native }}</option>
            </select>
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
                {{ fixing ? (msgs.progressFixing || 'Fixing...') : (msgs.fixButton || 'Fix CJK Character Reading Issues') }}
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
                    :key="d"
                    class="drive-chip"
                    :class="{ active: selectedDrive === d }"
                    :disabled="dbBusy"
                    @click="handleClickDrive(d)"
                >
                    {{ d }}
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
        </template>
    </div>

    <!-- TOOLS TAB -->
    <div class="content" v-show="activeTab === 'tools'">
        <div class="info-card">
            <div class="info-row">
                <span class="info-label">{{ msgs.msiCleanupTitle || 'MSI Cleanup' }}</span>
            </div>
            <div class="library-stats">{{ msgs.msiCleanupDescription || 'Scan and remove orphaned MSI installation residuals' }}</div>
        </div>

        <div class="actions">
            <button class="btn btn-primary no-drag" @click="handleScanMSI" :disabled="msiBusy">
                <span v-if="msiBusy" class="loading-spinner"></span>
                {{ msiBusy ? (msgs.msiScanning || 'Scanning...') : (msgs.msiCleanupButton || 'MSI Cleanup') }}
            </button>
        </div>

        <div v-if="msiScanned && msiOrphans.length === 0" class="status-card status-success">
            <div class="status-text">
                <div class="status-detail">{{ msgs.msiNoOrphans || 'No MSI orphans found' }}</div>
            </div>
        </div>

        <div v-if="msiOrphans.length > 0" class="library-section">
            <div class="library-section-title">{{ msgs.msiCleanupTitle || 'MSI Cleanup' }}</div>
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
            <div class="library-section-title">ID3 Tag Editor</div>
            <button class="btn btn-secondary no-drag" @click="handleID3Pick" :disabled="id3Busy">
                {{ id3File ? id3FileName : 'Select Audio File' }}
            </button>
        </div>

        <template v-if="id3File">
            <div class="info-card">
                <div class="id3-cover-row">
                    <img v-if="id3Cover" :src="id3Cover" class="id3-cover-img" />
                    <div v-else class="id3-cover-placeholder">No Cover</div>
                    <div class="id3-cover-actions">
                        <button class="btn btn-secondary no-drag id3-cover-btn" @click="handleID3SetCover" :disabled="id3Busy">Change</button>
                        <button class="btn btn-secondary no-drag id3-cover-btn" @click="handleID3ClearCover" :disabled="id3Busy || !id3Cover">Remove</button>
                    </div>
                </div>
            </div>
            <div class="library-section">
                <input class="text-input no-drag" v-model="id3Title" placeholder="Title" :disabled="id3Busy" />
                <input class="text-input no-drag" v-model="id3Artist" placeholder="Artist" :disabled="id3Busy" />
                <input class="text-input no-drag" v-model="id3Album" placeholder="Album" :disabled="id3Busy" />
                <div class="drive-select-row">
                    <input class="text-input no-drag" v-model="id3Year" placeholder="Year" :disabled="id3Busy" style="flex:1" />
                    <input class="text-input no-drag" v-model="id3Genre" placeholder="Genre" :disabled="id3Busy" style="flex:1" />
                </div>
                <div class="drive-select-row">
                    <button class="btn btn-primary no-drag" style="flex:1" @click="handleID3Save" :disabled="id3Busy">Save</button>
                    <button class="btn btn-restore-all no-drag" style="flex:1" @click="handleID3ClearAll" :disabled="id3Busy">Clear All</button>
                </div>
            </div>
        </template>

        <!-- USB Unlock -->
        <div class="library-section" style="margin-top: 12px;">
            <div class="library-section-title">USB Unlock</div>
            <div class="library-stats">
                Release file locks on your USB drive (excluding Engine DJ / Offline Analyzer / stems-processor)
            </div>
            <button
                class="btn btn-primary no-drag"
                @click="handleUSBUnlock"
                :disabled="usbBusy || !usbDrive"
            >
                <span v-if="usbBusy" class="loading-spinner"></span>
                {{ usbDrive ? ('Unlock ' + usbDrive) : 'No USB with Engine Library detected' }}
            </button>
            <button
                v-if="usbDrive"
                class="btn btn-secondary no-drag"
                @click="handleUSBScan"
                :disabled="usbBusy"
            >
                Scan blocking processes
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
            <div class="library-section-title">MIDI 2.0 Control</div>
            <div class="library-stats">
                Disable Windows 11 MIDI 2.0 features while preserving MIDI 1.0 service
            </div>
            <div v-if="midi2Status === 'enabled'" class="status-card status-warning" style="margin-bottom: 8px;">
                <div class="status-text">
                    <div class="status-detail">MIDI 2.0 is currently enabled</div>
                </div>
            </div>
            <div v-if="midi2Status === 'disabled'" class="status-card status-success" style="margin-bottom: 8px;">
                <div class="status-text">
                    <div class="status-detail">MIDI 2.0 is currently disabled</div>
                </div>
            </div>
            <div v-if="midi2Status === 'unavailable'" class="status-card status-error" style="margin-bottom: 8px;">
                <div class="status-text">
                    <div class="status-detail">MIDI 2.0 services not found on this system</div>
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
                    {{ midi2Busy ? 'Disabling...' : 'Disable MIDI 2.0' }}
                </button>
                <button
                    v-if="midi2Status === 'disabled'"
                    class="btn btn-secondary no-drag"
                    style="flex:1"
                    @click="handleMIDI2Enable"
                    :disabled="midi2Busy"
                >
                    <span v-if="midi2Busy" class="loading-spinner"></span>
                    {{ midi2Busy ? 'Enabling...' : 'Enable MIDI 2.0' }}
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
