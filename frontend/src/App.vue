<script setup>
import { ref, onMounted, computed } from 'vue'
import { GetStatus, FixCJKIssues, HandleUTF8AlreadyEnabled, OpenRegionSettings, SetLanguage, GetMessages, GetAvailableLanguages, Refresh } from '../wailsjs/go/main/App.js'
import { EventsOn } from '../wailsjs/runtime/runtime.js'

const installPath = ref('')
const utf8Enabled = ref(false)
const manifestConfigured = ref(false)
const processRunning = ref(false)
const runningProcesses = ref([])
const loading = ref(true)
const fixing = ref(false)
const progress = ref(0)
const showProgress = ref(false)
const msgs = ref({})
const languages = ref([])
const currentLang = ref('zh')

const pathDisplay = computed(() => {
    return installPath.value || msgs.value?.installPathNotFound || '—'
})

const utf8Class = computed(() => {
    return utf8Enabled.value ? 'enabled' : 'disabled'
})

const utf8Status = computed(() => {
    if (!msgs.value) return ''
    return utf8Enabled.value ? msgs.value.utf8Enabled : msgs.value.utf8Disabled
})

const manifestStatus = computed(() => {
    if (!msgs.value) return ''
    return manifestConfigured.value ? msgs.value.manifestExists : msgs.value.manifestNotExists
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
}

async function detectStatus() {
    loading.value = true
    try {
        const status = await GetStatus()
        installPath.value = status.installPath || ''
        utf8Enabled.value = status.utf8Enabled || false
        manifestConfigured.value = status.manifestConfigured || false
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
            utf8Enabled.value = false
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

async function changeLanguage(lang) {
    currentLang.value = lang
    await SetLanguage(lang)
    await loadMessages()
    await detectStatus()
}

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
})
</script>

<template>
    <div class="header wails-drag">
        <h1>{{ msgs.appTitle || 'Engine DJ Unicode Fix Tool' }}</h1>
        <select class="lang-select no-drag" :value="currentLang" @change="changeLanguage($event.target.value)">
            <option v-for="lang in languages" :key="lang.code" :value="lang.code">{{ lang.native }}</option>
        </select>
    </div>
    
    <div class="content">
        <div class="status-card">
            <div class="status-icon" :class="installPath ? 'success' : 'danger'">
                <span v-if="!loading">{{ installPath ? '📁' : '❌' }}</span>
                <span v-else class="loading-spinner"></span>
            </div>
            <div class="status-info">
                <div class="status-label">{{ msgs.installPathLabel || 'Install Path' }}</div>
                <div class="status-value">{{ pathDisplay }}</div>
            </div>
        </div>
        
        <div class="status-card">
            <div class="status-icon" :class="utf8Enabled ? 'warning' : 'success'">
                <span v-if="!loading">{{ utf8Enabled ? '⚠️' : '✓' }}</span>
                <span v-else class="loading-spinner"></span>
            </div>
            <div class="status-info">
                <div class="status-label">{{ msgs.utf8StatusLabel || 'System UTF-8 Support' }}</div>
                <div class="status-value" :class="utf8Class">{{ utf8Status }}</div>
            </div>
        </div>
        
        <div class="status-card">
            <div class="status-icon" :class="manifestConfigured ? 'success' : 'danger'">
                <span v-if="!loading">{{ manifestConfigured ? '✓' : '✗' }}</span>
                <span v-else class="loading-spinner"></span>
            </div>
            <div class="status-info">
                <div class="status-label">{{ msgs.manifestStatusLabel || 'External Manifest' }}</div>
                <div class="status-value" :class="manifestConfigured ? 'enabled' : 'disabled'">{{ manifestStatus }}</div>
            </div>
        </div>
        
        <div class="actions">
            <button 
                class="btn btn-primary no-drag" 
                @click="handleFix"
                :disabled="fixing || !installPath"
            >
                <span v-if="fixing" class="loading-spinner"></span>
                <span v-else>🔧</span>
                {{ fixing ? (msgs.progressFixing || 'Fixing...') : (msgs.fixButton || 'Fix CJK Character Reading Issues') }}
            </button>
            
            <button 
                v-if="utf8Enabled" 
                class="btn btn-secondary no-drag" 
                @click="handleOpenRegionSettings"
            >
                📋 {{ msgs.openRegionSettings || 'Open Region Settings' }}
            </button>
        </div>
    </div>
    
    <div class="progress-wrapper">
        <div class="progress-bar" :class="{ active: showProgress }">
            <div class="progress-fill" :style="{ width: progress + '%' }"></div>
        </div>
    </div>
    
    <div class="log-section">
        <div class="log-header">{{ msgs.logPrefix ? 'Console' : 'Console' }}</div>
        <div class="log-body" ref="logContainer">
            <div v-for="(entry, i) in logs" :key="i" class="log-entry">{{ entry }}</div>
        </div>
    </div>
</template>