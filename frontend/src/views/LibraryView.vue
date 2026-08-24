<script setup>
defineProps(['activeTab', 'backupNote', 'backups', 'dbBusy', 'dbDetected', 'dbPathDisplay', 'drives', 'editingTrack', 'formatBytes', 'formatDuration', 'formatTime', 'info', 'libraryStats', 'libraryStatsBusy', 'loading', 'missingBusy', 'missingScanned', 'missingSelected', 'missingTracks', 'msgs', 'path', 'playStats', 'playStatsBusy', 'playStatsTab', 'playlistLoading', 'playlistTracks', 'playlists', 'selectedBackup', 'selectedDrive', 'selectedPlaylist', 'status', 'toggleAllMissing', 'toggleMissingTrack'])
defineEmits(['update:backupNote', 'update:selectedBackup'])
</script>
<template>
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
                    :value="backupNote" @input="$emit('update:backupNote', $event.target.value)"
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
                <select class="text-input no-drag" :value="selectedBackup" @change="$emit('update:selectedBackup', $event.target.value)" :disabled="dbBusy || backups.length === 0">
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
</template>
