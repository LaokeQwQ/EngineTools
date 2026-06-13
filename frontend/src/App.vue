<script setup>
import { ref, onMounted, computed } from 'vue'
import { GetStatus, FixCJKIssues, HandleUTF8AlreadyEnabled, OpenRegionSettings, SetLanguage, GetMessages, GetAvailableLanguages, OpenRepository, ScanLibraries, RestoreOverviewFiles, RestoreAllLibraries } from '../wailsjs/go/main/App.js'
import { EventsOn } from '../wailsjs/runtime/runtime.js'

const APP_VERSION = '1.3.0'

const installPath = ref('')
const engineVersion = ref('')
const windowsVersion = ref('')
const utf8Enabled = ref(false)
const acpValue = ref('')
const manifestConfigured = ref(false)
const isAdmin = ref(false)
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

const libraries = ref([])
const scanningLibraries = ref(false)
const restoringLib = ref('')
const restoringAll = ref(false)

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

async function handleOpenRegionSettings() {
    await OpenRegionSettings()
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

async function scanLibraries() {
    scanningLibraries.value = true
    try {
        libraries.value = await ScanLibraries()
    } catch (e) {
        addLog('Error: ' + e)
    }
    scanningLibraries.value = false
}

async function handleRestore(dbPath, missingCount, totalCount) {
    restoringLib.value = dbPath
    showProgress.value = true
    progress.value = 0

    try {
        const res = await RestoreOverviewFiles(dbPath)
        if (res === 'ok') {
            await scanLibraries()
        }
    } catch (e) {
        addLog('Error: ' + e)
    }

    restoringLib.value = ''
    setTimeout(() => { showProgress.value = false }, 1000)
}

async function handleRestoreAll() {
    restoringAll.value = true
    showProgress.value = true
    progress.value = 0

    try {
        const res = await RestoreAllLibraries()
        if (res.startsWith('ok')) {
            await scanLibraries()
        }
    } catch (e) {
        addLog('Error: ' + e)
    }

    restoringAll.value = false
    setTimeout(() => { showProgress.value = false }, 1000)
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
    await scanLibraries()

    marqueeInterval = setInterval(() => {
        currentMarquee.value = (currentMarquee.value + 1) % marqueeTexts.value.length
    }, 4000)
})
</script>

<template>
    <div class="header wails-drag">
        <div class="header-left">
            <span class="app-title">{{ msgs.appTitle || 'Engine Tools' }}</span>
            <span class="marquee-text" :key="currentMarquee">{{ marqueeTexts[currentMarquee] }}</span>
        </div>
        <div class="header-right">
            <span class="github-link no-drag" @click="openGitHub">
                <svg class="github-icon" viewBox="0 0 16 16" fill="currentColor" width="16" height="16"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>
            </span>
            <span class="author-handle no-drag" @click="openGitHub">@LaokeQwQ</span>
            <select class="lang-select no-drag" :value="currentLang" @change="changeLanguage($event.target.value)">
                <option v-for="lang in languages" :key="lang.code" :value="lang.code">{{ lang.native }}</option>
            </select>
        </div>
    </div>

    <div class="content">
        <div class="info-card">
            <div class="info-row" v-if="windowsVersion">
                <span class="info-label">{{ msgs.windowsVersionLabel || 'Windows' }}</span>
                <span class="info-value">{{ windowsVersion }}</span>
            </div>
            <div class="info-row">
                <span class="info-label">{{ msgs.installPathLabel || 'Path' }}</span>
                <span class="info-value" :class="{ muted: !installPath }">{{ pathDisplay }}</span>
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
                v-if="utf8Enabled"
                class="btn btn-secondary no-drag"
                @click="handleOpenRegionSettings"
            >
                {{ msgs.openRegionSettings || 'Open Region Settings' }}
            </button>

            <button
                class="btn btn-restore-all no-drag"
                :disabled="restoringAll || scanningLibraries"
                @click="handleRestoreAll"
            >
                <span v-if="restoringAll" class="loading-spinner"></span>
                {{ restoringAll ? (msgs.progressFixing || 'Restoring...') : (msgs.restoreAllButton || 'Restore Waveform Preview Files (All Drives)') }}
            </button>
        </div>

        <!-- Library overview restore section -->
        <div v-if="libraries.length > 0" class="library-section">
            <div class="section-label">{{ msgs.libraryStatusLabel || 'Waveform Preview Files' }}</div>
            <div
                v-for="lib in libraries"
                :key="lib.path"
                class="library-item"
                :class="lib.missingRGB > 0 ? 'lib-warning' : 'lib-ok'"
            >
                <div class="lib-info">
                    <span class="lib-drive">{{ lib.drive }}</span>
                    <span class="lib-stat">
                        {{ lib.totalTracks - lib.missingRGB }} / {{ lib.totalTracks }}
                    </span>
                </div>
                <button
                    v-if="lib.missingRGB > 0"
                    class="btn btn-restore no-drag"
                    :disabled="restoringLib === lib.path"
                    @click="handleRestore(lib.path, lib.missingRGB, lib.totalTracks)"
                >
                    <span v-if="restoringLib === lib.path" class="loading-spinner"></span>
                    {{ restoringLib === lib.path
                        ? (msgs.progressFixing || 'Restoring...')
                        : (msgs.restoreButton || 'Restore Missing Waveform Files') }}
                </button>
                <span v-else class="lib-all-ok">✓</span>
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