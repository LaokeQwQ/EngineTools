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
    <StatusView
    :activeTab="activeTab"
    :adminStatusText="adminStatusText"
    :engineVersion="engineVersion"
    :fixing="fixing"
    :info="info"
    :installPath="installPath"
    :isAdmin="isAdmin"
    :loading="loading"
    :manifestConfigured="manifestConfigured"
    :manifestStatusText="manifestStatusText"
    :msgs="msgs"
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
    :info="info"
    :libraryStats="libraryStats"
    :libraryStatsBusy="libraryStatsBusy"
    :loading="loading"
    :missingBusy="missingBusy"
    :missingScanned="missingScanned"
    :missingSelected="missingSelected"
    :missingTracks="missingTracks"
    :msgs="msgs"
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
    @update:apPlaylistId="val => apPlaylistId = val"
    @update:coverCompressPlaylistId="val => coverCompressPlaylistId = val"
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
