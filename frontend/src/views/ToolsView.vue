<script setup>
defineProps(['activeTab', 'addLog', 'apBusy', 'apDir', 'apFile', 'apIsReady', 'apPlaylistId', 'apScope', 'cacheBusy', 'cacheSizeText', 'coverCompressBusy', 'coverCompressPlaylistId', 'coverCompressResult', 'editingTrack', 'experimentalEnabled', 'files', 'handleAnalyzeLogs', 'handleAntiPiracyRestore', 'handleAntiPiracyV1', 'handleAntiPiracyV2', 'handleClearCache', 'handleCleanMSI', 'handleCompressCovers', 'handleID3ClearAll', 'handleID3ClearCover', 'handleID3FileDrop', 'handleID3Pick', 'handleID3Save', 'handleID3SetCover', 'handleLoadSyncPreview', 'handleOpenLogsDir', 'handlePickApDir', 'handlePickApFile', 'handleScanMSI', 'handleSyncBPMKey', 'handleUSBScan', 'handleUSBUnlock', 'id3Album', 'id3Artist', 'id3Busy', 'id3Cover', 'id3DragOver', 'id3File', 'id3FileName', 'id3Genre', 'id3Title', 'id3Year', 'info', 'loading', 'logBusy', 'logStats', 'msgs', 'msiBusy', 'msiOrphans', 'msiScanned', 'msiSelected', 'playlists', 'status', 'syncBusy', 'syncPreviewLoaded', 'syncPreviewTracks', 'toggleMSI', 'usbBlockers', 'usbBusy', 'usbDrive'])
defineEmits(['update:id3Title', 'update:id3Artist', 'update:id3Album', 'update:id3Year', 'update:id3Genre', 'update:apPlaylistId', 'update:coverCompressPlaylistId', 'update:apScope', 'update:editingTrack', 'update:id3File', 'update:id3DragOver'])
</script>
<template>
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
                @dragover.prevent="$emit('update:id3DragOver', true)"
                @dragleave="$emit('update:id3DragOver', false)"
                @drop.prevent="handleID3FileDrop">
                <span v-if="id3File" class="id3-drop-filename">{{ id3FileName }}</span>
                <span v-else class="id3-drop-hint">{{ msgs.id3DropZoneHint || '拖入音频文件，或点击选择' }}</span>
            </div>

            <!-- Hint if loaded from playlist -->
            <div v-if="editingTrack" class="library-stats" style="margin-top:4px;font-size:11px;">
                {{ editingTrack.artist ? editingTrack.artist + ' — ' : '' }}{{ editingTrack.title || editingTrack.filename }}
                <span style="margin-left:8px;cursor:pointer;color:var(--text-secondary);" @click="$emit('update:editingTrack', null);$emit('update:id3File', '')" class="no-drag">✕</span>
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
                <input class="text-input no-drag" :value="id3Title" @input="$emit('update:id3Title', $event.target.value)" :placeholder="msgs.id3TitlePlaceholder || '标题'" :disabled="id3Busy" />
                <input class="text-input no-drag" :value="id3Artist" @input="$emit('update:id3Artist', $event.target.value)" :placeholder="msgs.id3ArtistPlaceholder || '艺术家'" :disabled="id3Busy" />
                <input class="text-input no-drag" :value="id3Album" @input="$emit('update:id3Album', $event.target.value)" :placeholder="msgs.id3AlbumPlaceholder || '专辑'" :disabled="id3Busy" />
                <div class="drive-select-row">
                    <input class="text-input no-drag" :value="id3Year" @input="$emit('update:id3Year', $event.target.value)" :placeholder="msgs.id3YearPlaceholder || '年份'" :disabled="id3Busy" style="flex:1" />
                    <input class="text-input no-drag" :value="id3Genre" @input="$emit('update:id3Genre', $event.target.value)" :placeholder="msgs.id3GenrePlaceholder || '风格'" :disabled="id3Busy" style="flex:1" />
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
                <div class="library-stats" style="margin-bottom:8px;">对所选范围内的音频文件批量操作 ID3 标签，操作前自动备份</div>

                <!-- Scope selector -->
                <div class="playlist-chips" style="margin-bottom:8px;">
                    <span class="playlist-chip no-drag" :class="{ active: apScope === 'file' }" @click="$emit('update:apScope', 'file')">单个文件</span>
                    <span class="playlist-chip no-drag" :class="{ active: apScope === 'folder' }" @click="$emit('update:apScope', 'folder')">文件夹</span>
                    <span class="playlist-chip no-drag" :class="{ active: apScope === 'playlist' }" @click="$emit('update:apScope', 'playlist')">Playlist</span>
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
                    :value="apPlaylistId" @input="$emit('update:apPlaylistId', $event.target.value)" :disabled="apBusy" style="margin-bottom:4px;">
                    <option :value="-1" disabled>选择播放列表</option>
                    <option v-for="pl in playlists" :key="pl.id" :value="pl.id">
                        {{ pl.title }} ({{ pl.count }})
                    </option>
                </select>

                <!-- Action buttons -->
                <template v-if="apIsReady()">
                    <div class="drive-select-row" style="margin-top: 8px;">
                        <button class="btn btn-restore-all no-drag" style="flex:1" @click="handleAntiPiracyV1" :disabled="apBusy || apScope === 'file'">
                            <span v-if="apBusy" class="loading-spinner"></span>
                            v1 清除文本标签
                        </button>
                        <button class="btn btn-restore-all no-drag" style="flex:1" @click="handleAntiPiracyV2" :disabled="apBusy || apScope === 'file'">
                            <span v-if="apBusy" class="loading-spinner"></span>
                            v2 打乱文本标签
                        </button>
                    </div>
                    <button class="btn btn-secondary no-drag" style="margin-top: 6px;width:100%;" @click="handleAntiPiracyRestore" :disabled="apBusy">
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
            <select class="text-input no-drag" :value="coverCompressPlaylistId" @change="$emit('update:coverCompressPlaylistId', $event.target.value)"
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
</template>
