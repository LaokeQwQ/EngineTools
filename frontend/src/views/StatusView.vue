<script setup>
defineProps(['activeTab', 'adminStatusText', 'engineVersion', 'fixing', 'handleFix', 'handleOpenRegionSettings', 'handleRestore', 'info', 'installPath', 'isAdmin', 'loading', 'manifestConfigured', 'manifestStatusText', 'msgs', 'openInstallDir', 'openStemsDir', 'path', 'pathDisplay', 'status', 'stemsDetected', 'stemsStatusText', 'utf8Enabled', 'utf8StatusText', 'versionDisplay', 'windowsVersion'])
</script>
<template>
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
</template>
