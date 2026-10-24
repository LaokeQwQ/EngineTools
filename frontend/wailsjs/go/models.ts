export namespace apperror {
	
	export class PublicError {
	    code: string;
	    messageKey: string;
	    operationId?: string;
	    retryable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PublicError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.messageKey = source["messageKey"];
	        this.operationId = source["operationId"];
	        this.retryable = source["retryable"];
	    }
	}

}

export namespace database {
	
	export class BPMBucket {
	    range: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new BPMBucket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.range = source["range"];
	        this.count = source["count"];
	    }
	}
	export class BackupInfo {
	    filename: string;
	    date: string;
	    size: number;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filename = source["filename"];
	        this.date = source["date"];
	        this.size = source["size"];
	        this.note = source["note"];
	    }
	}
	export class KVCount {
	    key: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new KVCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.count = source["count"];
	    }
	}
	export class TrackSummary {
	    id: number;
	    title: string;
	    artist: string;
	    bpm: number;
	    key: number;
	    keyName: string;
	    playCount: number;
	    lastPlayed: string;
	    dateAdded: string;
	
	    static createFrom(source: any = {}) {
	        return new TrackSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.bpm = source["bpm"];
	        this.key = source["key"];
	        this.keyName = source["keyName"];
	        this.playCount = source["playCount"];
	        this.lastPlayed = source["lastPlayed"];
	        this.dateAdded = source["dateAdded"];
	    }
	}
	export class LibraryStats {
	    totalTracks: number;
	    totalDuration: number;
	    totalFileBytes: number;
	    analyzedTracks: number;
	    missingTracks: number;
	    neverPlayed: number;
	    fileTypes: KVCount[];
	    topGenres: KVCount[];
	    bpmDistribution: BPMBucket[];
	    recentlyAdded: TrackSummary[];
	
	    static createFrom(source: any = {}) {
	        return new LibraryStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalTracks = source["totalTracks"];
	        this.totalDuration = source["totalDuration"];
	        this.totalFileBytes = source["totalFileBytes"];
	        this.analyzedTracks = source["analyzedTracks"];
	        this.missingTracks = source["missingTracks"];
	        this.neverPlayed = source["neverPlayed"];
	        this.fileTypes = this.convertValues(source["fileTypes"], KVCount);
	        this.topGenres = this.convertValues(source["topGenres"], KVCount);
	        this.bpmDistribution = this.convertValues(source["bpmDistribution"], BPMBucket);
	        this.recentlyAdded = this.convertValues(source["recentlyAdded"], TrackSummary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MissingTrack {
	    id: number;
	    title: string;
	    artist: string;
	    path: string;
	    filename: string;
	
	    static createFrom(source: any = {}) {
	        return new MissingTrack(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.path = source["path"];
	        this.filename = source["filename"];
	    }
	}
	export class PlayStats {
	    mostPlayed: TrackSummary[];
	    recentPlayed: TrackSummary[];
	    neverPlayed: TrackSummary[];
	
	    static createFrom(source: any = {}) {
	        return new PlayStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mostPlayed = this.convertValues(source["mostPlayed"], TrackSummary);
	        this.recentPlayed = this.convertValues(source["recentPlayed"], TrackSummary);
	        this.neverPlayed = this.convertValues(source["neverPlayed"], TrackSummary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PlaylistInfo {
	    id: number;
	    title: string;
	    parentId: number;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new PlaylistInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.parentId = source["parentId"];
	        this.count = source["count"];
	    }
	}
	export class SyncableTrack {
	    id: number;
	    title: string;
	    artist: string;
	    path: string;
	    bpm: number;
	    key: number;
	    keyName: string;
	    camelot: string;
	
	    static createFrom(source: any = {}) {
	        return new SyncableTrack(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.path = source["path"];
	        this.bpm = source["bpm"];
	        this.key = source["key"];
	        this.keyName = source["keyName"];
	        this.camelot = source["camelot"];
	    }
	}
	export class TrackInfo {
	    id: number;
	    title: string;
	    artist: string;
	    album: string;
	    genre: string;
	    bpm: number;
	    length: number;
	    filename: string;
	    path: string;
	    key: number;
	    keyName: string;
	    camelot: string;
	    rating: number;
	
	    static createFrom(source: any = {}) {
	        return new TrackInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.genre = source["genre"];
	        this.bpm = source["bpm"];
	        this.length = source["length"];
	        this.filename = source["filename"];
	        this.path = source["path"];
	        this.key = source["key"];
	        this.keyName = source["keyName"];
	        this.camelot = source["camelot"];
	        this.rating = source["rating"];
	    }
	}

}

export namespace i18n {
	
	export class Messages {
	    appTitle: string;
	    installPathLabel: string;
	    installPathNotFound: string;
	    engineVersionLabel: string;
	    windowsVersionLabel: string;
	    utf8StatusLabel: string;
	    utf8Enabled: string;
	    utf8Disabled: string;
	    manifestStatusLabel: string;
	    manifestExists: string;
	    manifestNotExists: string;
	    fixButton: string;
	    restoreButton: string;
	    openRegionSettings: string;
	    utf8AlreadyEnabled: string;
	    utf8AlreadyEnabledTip: string;
	    processRunningTitle: string;
	    processRunningMessage: string;
	    killingProcesses: string;
	    processKilled: string;
	    noProcessRunning: string;
	    writingManifest: string;
	    manifestWritten: string;
	    manifestWriteError: string;
	    exeNotFound: string;
	    writingRegistry: string;
	    registryWritten: string;
	    registryWriteError: string;
	    deletingManifests: string;
	    manifestsDeleted: string;
	    deletingRegistry: string;
	    registryDeleted: string;
	    refreshingSystem: string;
	    systemRefreshed: string;
	    fixComplete: string;
	    fixCompleteTip: string;
	    restoreComplete: string;
	    restoreCompleteTip: string;
	    fixFailed: string;
	    logPrefix: string;
	    checking: string;
	    statusChecking: string;
	    progressDetecting: string;
	    progressFixing: string;
	    progressRestoring: string;
	    progressDone: string;
	    language: string;
	    acpCodePage: string;
	    notInstalled: string;
	    version: string;
	    marqueeFree: string;
	    marqueeStar: string;
	    adminStatusLabel: string;
	    adminYes: string;
	    adminNo: string;
	    stemsStatusLabel: string;
	    stemsDetected: string;
	    stemsNotFound: string;
	    restoreConfirmTitle: string;
	    restoreConfirmMessage: string;
	    backupReminderTitle: string;
	    backupReminderMessage: string;
	    tabStatus: string;
	    tabDatabase: string;
	    tabTools: string;
	    dbLibraryPathLabel: string;
	    dbLibraryNotFound: string;
	    dbBackupButton: string;
	    dbBackupNoteLabel: string;
	    dbBackupNotePlaceholder: string;
	    dbBackingUp: string;
	    dbBackupComplete: string;
	    dbBackupError: string;
	    dbSelectDriveLabel: string;
	    dbSelectDrivePlaceholder: string;
	    dbSelectDriveConfirm: string;
	    dbDriveNotFound: string;
	    dbRestoreButton: string;
	    dbRestoreSelectDate: string;
	    dbRestoreConfirmTitle: string;
	    dbRestoreConfirmMessage: string;
	    dbRestoring: string;
	    dbRestoreComplete: string;
	    dbOptimizeButton: string;
	    dbOptimizing: string;
	    dbOptimizeComplete: string;
	    dbRepairButton: string;
	    dbRepairing: string;
	    dbRepairComplete: string;
	    dbRepairStart: string;
	    dbRepairDone: string;
	    dbNoteLabel: string;
	    dbNoneFound: string;
	    dbNoBackups: string;
	    msiCleanupButton: string;
	    msiCleanupTitle: string;
	    msiCleanupDescription: string;
	    msiScanning: string;
	    msiFoundOrphans: string;
	    msiNoOrphans: string;
	    msiCleaning: string;
	    msiCleanComplete: string;
	    msiCleanError: string;
	    msiConfirmTitle: string;
	    msiConfirmMessage: string;
	    id3EditorTitle: string;
	    id3SelectFile: string;
	    id3PickFileButton: string;
	    id3SaveButton: string;
	    id3ClearAllButton: string;
	    usbUnlockTitle: string;
	    usbUnlockDescription: string;
	    usbUnlockButton: string;
	    usbScanButton: string;
	    usbNoDevice: string;
	    stemsEngineLabel: string;
	    logAnalysisTitle: string;
	    logAnalysisDescription: string;
	    logAnalyzeButton: string;
	    logAnalyzing: string;
	    logOpenDir: string;
	    logTotalFiles: string;
	    logTotalLines: string;
	    logInfoCount: string;
	    logWarningCount: string;
	    logErrorCount: string;
	    logTopWarnings: string;
	    logTopErrors: string;
	    logNoFiles: string;
	    cacheCleanTitle: string;
	    cacheCleanDescription: string;
	    cacheCleanButton: string;
	    cacheCleaning: string;
	    cacheCleanComplete: string;
	    cacheLatestFile: string;
	    updateBadge: string;
	    updateChecking: string;
	    updateTooltip: string;
	    operationSuccess: string;
	    libraryStatsTitle: string;
	    libraryStatsButton: string;
	    libraryStatsBusy: string;
	    missingTracksTitle: string;
	    missingTracksScan: string;
	    missingTracksScanning: string;
	    missingTracksNone: string;
	    missingTracksRemove: string;
	    missingTracksSelectAll: string;
	    missingTracksDeselectAll: string;
	    playHistoryTitle: string;
	    playHistoryLoad: string;
	    playHistoryLoading: string;
	    playHistoryMostPlayed: string;
	    playHistoryRecent: string;
	    playHistoryNeverPlayed: string;
	    bpmKeySyncTitle: string;
	    bpmKeySyncLoad: string;
	    bpmKeySyncReady: string;
	    bpmKeySyncWrite: string;
	    bpmKeySyncWriting: string;
	    coverCompressTitle: string;
	    coverCompressButton: string;
	    coverCompressCompressing: string;
	    coverCompressAllLibrary: string;
	    coverCompressTip: string;
	    totalTracksLabel: string;
	    totalDurationLabel: string;
	    librarySizeLabel: string;
	    analyzedLabel: string;
	    missingTracksDesc: string;
	    bpmKeySyncDesc: string;
	    failedLabel: string;
	    savedLabel: string;
	    skippedLabel: string;
	    andMoreLabel: string;
	    tracksUnit: string;
	    tabSettings: string;
	    settingsLanguageLabel: string;
	    settingsExperimentalLabel: string;
	    settingsExperimentalDesc: string;
	    settingsExperimentalConfirmTitle: string;
	    settingsExperimentalConfirmMsg: string;
	    settingsExperimentalConfirmBtn: string;
	    settingsAboutTitle: string;
	    settingsAboutDesc: string;
	    settingsContributors: string;
	    metadataEditorTitle: string;
	    metadataChooseFile: string;
	    metadataDropHint: string;
	    metadataDropSubhint: string;
	    metadataFormatLabel: string;
	    metadataTagVersionLabel: string;
	    metadataWarnings: string;
	    metadataAddPicture: string;
	    metadataPictureCount: string;
	    metadataReplacePicture: string;
	    metadataRemovePicture: string;
	    metadataPicturePayloadMissing: string;
	    metadataSave: string;
	    metadataReset: string;
	    metadataUndo: string;
	    metadataUnchanged: string;
	    metadataSet: string;
	    metadataClear: string;
	    metadataLoading: string;
	    metadataSaving: string;
	    metadataUndoing: string;
	    metadataUnsaved: string;
	    metadataReady: string;
	    metadataUnsupported: string;
	    metadataTitleLabel: string;
	    metadataArtistsLabel: string;
	    metadataAlbumArtistsLabel: string;
	    metadataAlbumLabel: string;
	    metadataTrackLabel: string;
	    metadataDiscLabel: string;
	    metadataDateLabel: string;
	    metadataGenresLabel: string;
	    metadataComposersLabel: string;
	    metadataCommentLabel: string;
	    metadataBpmLabel: string;
	    metadataKeyLabel: string;
	    metadataRatingLabel: string;
	    metadataIsrcLabel: string;
	    metadataLyricsLabel: string;
	    metadataPicturesLabel: string;
	    metadataSingleMode: string;
	    metadataBatchMode: string;
	    metadataBatchTitle: string;
	    metadataBatchFiles: string;
	    metadataBatchNoFiles: string;
	    metadataBatchChooseFolder: string;
	    metadataBatchClearFiles: string;
	    metadataBatchPreview: string;
	    metadataBatchPreviewTitle: string;
	    metadataBatchStart: string;
	    metadataBatchRunning: string;
	    metadataBatchEligible: string;
	    metadataBatchResults: string;
	    metadataBatchUpdated: string;
	    metadataBatchSkipped: string;
	    metadataBatchFailed: string;
	    metadataBatchBlocked: string;
	    metadataBatchCancelled: string;
	    metadataBatchStatusUpdated: string;
	    metadataBatchStatusSkipped: string;
	    metadataBatchStatusFailed: string;
	    metadataBatchStatusBlocked: string;
	    metadataCancel: string;
	    id3DropZoneHint: string;
	    noCoverLabel: string;
	    changeCoverLabel: string;
	    removeCoverLabel: string;
	    id3TitlePlaceholder: string;
	    id3ArtistPlaceholder: string;
	    id3AlbumPlaceholder: string;
	    id3YearPlaceholder: string;
	    id3GenrePlaceholder: string;
	
	    static createFrom(source: any = {}) {
	        return new Messages(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appTitle = source["appTitle"];
	        this.installPathLabel = source["installPathLabel"];
	        this.installPathNotFound = source["installPathNotFound"];
	        this.engineVersionLabel = source["engineVersionLabel"];
	        this.windowsVersionLabel = source["windowsVersionLabel"];
	        this.utf8StatusLabel = source["utf8StatusLabel"];
	        this.utf8Enabled = source["utf8Enabled"];
	        this.utf8Disabled = source["utf8Disabled"];
	        this.manifestStatusLabel = source["manifestStatusLabel"];
	        this.manifestExists = source["manifestExists"];
	        this.manifestNotExists = source["manifestNotExists"];
	        this.fixButton = source["fixButton"];
	        this.restoreButton = source["restoreButton"];
	        this.openRegionSettings = source["openRegionSettings"];
	        this.utf8AlreadyEnabled = source["utf8AlreadyEnabled"];
	        this.utf8AlreadyEnabledTip = source["utf8AlreadyEnabledTip"];
	        this.processRunningTitle = source["processRunningTitle"];
	        this.processRunningMessage = source["processRunningMessage"];
	        this.killingProcesses = source["killingProcesses"];
	        this.processKilled = source["processKilled"];
	        this.noProcessRunning = source["noProcessRunning"];
	        this.writingManifest = source["writingManifest"];
	        this.manifestWritten = source["manifestWritten"];
	        this.manifestWriteError = source["manifestWriteError"];
	        this.exeNotFound = source["exeNotFound"];
	        this.writingRegistry = source["writingRegistry"];
	        this.registryWritten = source["registryWritten"];
	        this.registryWriteError = source["registryWriteError"];
	        this.deletingManifests = source["deletingManifests"];
	        this.manifestsDeleted = source["manifestsDeleted"];
	        this.deletingRegistry = source["deletingRegistry"];
	        this.registryDeleted = source["registryDeleted"];
	        this.refreshingSystem = source["refreshingSystem"];
	        this.systemRefreshed = source["systemRefreshed"];
	        this.fixComplete = source["fixComplete"];
	        this.fixCompleteTip = source["fixCompleteTip"];
	        this.restoreComplete = source["restoreComplete"];
	        this.restoreCompleteTip = source["restoreCompleteTip"];
	        this.fixFailed = source["fixFailed"];
	        this.logPrefix = source["logPrefix"];
	        this.checking = source["checking"];
	        this.statusChecking = source["statusChecking"];
	        this.progressDetecting = source["progressDetecting"];
	        this.progressFixing = source["progressFixing"];
	        this.progressRestoring = source["progressRestoring"];
	        this.progressDone = source["progressDone"];
	        this.language = source["language"];
	        this.acpCodePage = source["acpCodePage"];
	        this.notInstalled = source["notInstalled"];
	        this.version = source["version"];
	        this.marqueeFree = source["marqueeFree"];
	        this.marqueeStar = source["marqueeStar"];
	        this.adminStatusLabel = source["adminStatusLabel"];
	        this.adminYes = source["adminYes"];
	        this.adminNo = source["adminNo"];
	        this.stemsStatusLabel = source["stemsStatusLabel"];
	        this.stemsDetected = source["stemsDetected"];
	        this.stemsNotFound = source["stemsNotFound"];
	        this.restoreConfirmTitle = source["restoreConfirmTitle"];
	        this.restoreConfirmMessage = source["restoreConfirmMessage"];
	        this.backupReminderTitle = source["backupReminderTitle"];
	        this.backupReminderMessage = source["backupReminderMessage"];
	        this.tabStatus = source["tabStatus"];
	        this.tabDatabase = source["tabDatabase"];
	        this.tabTools = source["tabTools"];
	        this.dbLibraryPathLabel = source["dbLibraryPathLabel"];
	        this.dbLibraryNotFound = source["dbLibraryNotFound"];
	        this.dbBackupButton = source["dbBackupButton"];
	        this.dbBackupNoteLabel = source["dbBackupNoteLabel"];
	        this.dbBackupNotePlaceholder = source["dbBackupNotePlaceholder"];
	        this.dbBackingUp = source["dbBackingUp"];
	        this.dbBackupComplete = source["dbBackupComplete"];
	        this.dbBackupError = source["dbBackupError"];
	        this.dbSelectDriveLabel = source["dbSelectDriveLabel"];
	        this.dbSelectDrivePlaceholder = source["dbSelectDrivePlaceholder"];
	        this.dbSelectDriveConfirm = source["dbSelectDriveConfirm"];
	        this.dbDriveNotFound = source["dbDriveNotFound"];
	        this.dbRestoreButton = source["dbRestoreButton"];
	        this.dbRestoreSelectDate = source["dbRestoreSelectDate"];
	        this.dbRestoreConfirmTitle = source["dbRestoreConfirmTitle"];
	        this.dbRestoreConfirmMessage = source["dbRestoreConfirmMessage"];
	        this.dbRestoring = source["dbRestoring"];
	        this.dbRestoreComplete = source["dbRestoreComplete"];
	        this.dbOptimizeButton = source["dbOptimizeButton"];
	        this.dbOptimizing = source["dbOptimizing"];
	        this.dbOptimizeComplete = source["dbOptimizeComplete"];
	        this.dbRepairButton = source["dbRepairButton"];
	        this.dbRepairing = source["dbRepairing"];
	        this.dbRepairComplete = source["dbRepairComplete"];
	        this.dbRepairStart = source["dbRepairStart"];
	        this.dbRepairDone = source["dbRepairDone"];
	        this.dbNoteLabel = source["dbNoteLabel"];
	        this.dbNoneFound = source["dbNoneFound"];
	        this.dbNoBackups = source["dbNoBackups"];
	        this.msiCleanupButton = source["msiCleanupButton"];
	        this.msiCleanupTitle = source["msiCleanupTitle"];
	        this.msiCleanupDescription = source["msiCleanupDescription"];
	        this.msiScanning = source["msiScanning"];
	        this.msiFoundOrphans = source["msiFoundOrphans"];
	        this.msiNoOrphans = source["msiNoOrphans"];
	        this.msiCleaning = source["msiCleaning"];
	        this.msiCleanComplete = source["msiCleanComplete"];
	        this.msiCleanError = source["msiCleanError"];
	        this.msiConfirmTitle = source["msiConfirmTitle"];
	        this.msiConfirmMessage = source["msiConfirmMessage"];
	        this.id3EditorTitle = source["id3EditorTitle"];
	        this.id3SelectFile = source["id3SelectFile"];
	        this.id3PickFileButton = source["id3PickFileButton"];
	        this.id3SaveButton = source["id3SaveButton"];
	        this.id3ClearAllButton = source["id3ClearAllButton"];
	        this.usbUnlockTitle = source["usbUnlockTitle"];
	        this.usbUnlockDescription = source["usbUnlockDescription"];
	        this.usbUnlockButton = source["usbUnlockButton"];
	        this.usbScanButton = source["usbScanButton"];
	        this.usbNoDevice = source["usbNoDevice"];
	        this.stemsEngineLabel = source["stemsEngineLabel"];
	        this.logAnalysisTitle = source["logAnalysisTitle"];
	        this.logAnalysisDescription = source["logAnalysisDescription"];
	        this.logAnalyzeButton = source["logAnalyzeButton"];
	        this.logAnalyzing = source["logAnalyzing"];
	        this.logOpenDir = source["logOpenDir"];
	        this.logTotalFiles = source["logTotalFiles"];
	        this.logTotalLines = source["logTotalLines"];
	        this.logInfoCount = source["logInfoCount"];
	        this.logWarningCount = source["logWarningCount"];
	        this.logErrorCount = source["logErrorCount"];
	        this.logTopWarnings = source["logTopWarnings"];
	        this.logTopErrors = source["logTopErrors"];
	        this.logNoFiles = source["logNoFiles"];
	        this.cacheCleanTitle = source["cacheCleanTitle"];
	        this.cacheCleanDescription = source["cacheCleanDescription"];
	        this.cacheCleanButton = source["cacheCleanButton"];
	        this.cacheCleaning = source["cacheCleaning"];
	        this.cacheCleanComplete = source["cacheCleanComplete"];
	        this.cacheLatestFile = source["cacheLatestFile"];
	        this.updateBadge = source["updateBadge"];
	        this.updateChecking = source["updateChecking"];
	        this.updateTooltip = source["updateTooltip"];
	        this.operationSuccess = source["operationSuccess"];
	        this.libraryStatsTitle = source["libraryStatsTitle"];
	        this.libraryStatsButton = source["libraryStatsButton"];
	        this.libraryStatsBusy = source["libraryStatsBusy"];
	        this.missingTracksTitle = source["missingTracksTitle"];
	        this.missingTracksScan = source["missingTracksScan"];
	        this.missingTracksScanning = source["missingTracksScanning"];
	        this.missingTracksNone = source["missingTracksNone"];
	        this.missingTracksRemove = source["missingTracksRemove"];
	        this.missingTracksSelectAll = source["missingTracksSelectAll"];
	        this.missingTracksDeselectAll = source["missingTracksDeselectAll"];
	        this.playHistoryTitle = source["playHistoryTitle"];
	        this.playHistoryLoad = source["playHistoryLoad"];
	        this.playHistoryLoading = source["playHistoryLoading"];
	        this.playHistoryMostPlayed = source["playHistoryMostPlayed"];
	        this.playHistoryRecent = source["playHistoryRecent"];
	        this.playHistoryNeverPlayed = source["playHistoryNeverPlayed"];
	        this.bpmKeySyncTitle = source["bpmKeySyncTitle"];
	        this.bpmKeySyncLoad = source["bpmKeySyncLoad"];
	        this.bpmKeySyncReady = source["bpmKeySyncReady"];
	        this.bpmKeySyncWrite = source["bpmKeySyncWrite"];
	        this.bpmKeySyncWriting = source["bpmKeySyncWriting"];
	        this.coverCompressTitle = source["coverCompressTitle"];
	        this.coverCompressButton = source["coverCompressButton"];
	        this.coverCompressCompressing = source["coverCompressCompressing"];
	        this.coverCompressAllLibrary = source["coverCompressAllLibrary"];
	        this.coverCompressTip = source["coverCompressTip"];
	        this.totalTracksLabel = source["totalTracksLabel"];
	        this.totalDurationLabel = source["totalDurationLabel"];
	        this.librarySizeLabel = source["librarySizeLabel"];
	        this.analyzedLabel = source["analyzedLabel"];
	        this.missingTracksDesc = source["missingTracksDesc"];
	        this.bpmKeySyncDesc = source["bpmKeySyncDesc"];
	        this.failedLabel = source["failedLabel"];
	        this.savedLabel = source["savedLabel"];
	        this.skippedLabel = source["skippedLabel"];
	        this.andMoreLabel = source["andMoreLabel"];
	        this.tracksUnit = source["tracksUnit"];
	        this.tabSettings = source["tabSettings"];
	        this.settingsLanguageLabel = source["settingsLanguageLabel"];
	        this.settingsExperimentalLabel = source["settingsExperimentalLabel"];
	        this.settingsExperimentalDesc = source["settingsExperimentalDesc"];
	        this.settingsExperimentalConfirmTitle = source["settingsExperimentalConfirmTitle"];
	        this.settingsExperimentalConfirmMsg = source["settingsExperimentalConfirmMsg"];
	        this.settingsExperimentalConfirmBtn = source["settingsExperimentalConfirmBtn"];
	        this.settingsAboutTitle = source["settingsAboutTitle"];
	        this.settingsAboutDesc = source["settingsAboutDesc"];
	        this.settingsContributors = source["settingsContributors"];
	        this.metadataEditorTitle = source["metadataEditorTitle"];
	        this.metadataChooseFile = source["metadataChooseFile"];
	        this.metadataDropHint = source["metadataDropHint"];
	        this.metadataDropSubhint = source["metadataDropSubhint"];
	        this.metadataFormatLabel = source["metadataFormatLabel"];
	        this.metadataTagVersionLabel = source["metadataTagVersionLabel"];
	        this.metadataWarnings = source["metadataWarnings"];
	        this.metadataAddPicture = source["metadataAddPicture"];
	        this.metadataPictureCount = source["metadataPictureCount"];
	        this.metadataReplacePicture = source["metadataReplacePicture"];
	        this.metadataRemovePicture = source["metadataRemovePicture"];
	        this.metadataPicturePayloadMissing = source["metadataPicturePayloadMissing"];
	        this.metadataSave = source["metadataSave"];
	        this.metadataReset = source["metadataReset"];
	        this.metadataUndo = source["metadataUndo"];
	        this.metadataUnchanged = source["metadataUnchanged"];
	        this.metadataSet = source["metadataSet"];
	        this.metadataClear = source["metadataClear"];
	        this.metadataLoading = source["metadataLoading"];
	        this.metadataSaving = source["metadataSaving"];
	        this.metadataUndoing = source["metadataUndoing"];
	        this.metadataUnsaved = source["metadataUnsaved"];
	        this.metadataReady = source["metadataReady"];
	        this.metadataUnsupported = source["metadataUnsupported"];
	        this.metadataTitleLabel = source["metadataTitleLabel"];
	        this.metadataArtistsLabel = source["metadataArtistsLabel"];
	        this.metadataAlbumArtistsLabel = source["metadataAlbumArtistsLabel"];
	        this.metadataAlbumLabel = source["metadataAlbumLabel"];
	        this.metadataTrackLabel = source["metadataTrackLabel"];
	        this.metadataDiscLabel = source["metadataDiscLabel"];
	        this.metadataDateLabel = source["metadataDateLabel"];
	        this.metadataGenresLabel = source["metadataGenresLabel"];
	        this.metadataComposersLabel = source["metadataComposersLabel"];
	        this.metadataCommentLabel = source["metadataCommentLabel"];
	        this.metadataBpmLabel = source["metadataBpmLabel"];
	        this.metadataKeyLabel = source["metadataKeyLabel"];
	        this.metadataRatingLabel = source["metadataRatingLabel"];
	        this.metadataIsrcLabel = source["metadataIsrcLabel"];
	        this.metadataLyricsLabel = source["metadataLyricsLabel"];
	        this.metadataPicturesLabel = source["metadataPicturesLabel"];
	        this.metadataSingleMode = source["metadataSingleMode"];
	        this.metadataBatchMode = source["metadataBatchMode"];
	        this.metadataBatchTitle = source["metadataBatchTitle"];
	        this.metadataBatchFiles = source["metadataBatchFiles"];
	        this.metadataBatchNoFiles = source["metadataBatchNoFiles"];
	        this.metadataBatchChooseFolder = source["metadataBatchChooseFolder"];
	        this.metadataBatchClearFiles = source["metadataBatchClearFiles"];
	        this.metadataBatchPreview = source["metadataBatchPreview"];
	        this.metadataBatchPreviewTitle = source["metadataBatchPreviewTitle"];
	        this.metadataBatchStart = source["metadataBatchStart"];
	        this.metadataBatchRunning = source["metadataBatchRunning"];
	        this.metadataBatchEligible = source["metadataBatchEligible"];
	        this.metadataBatchResults = source["metadataBatchResults"];
	        this.metadataBatchUpdated = source["metadataBatchUpdated"];
	        this.metadataBatchSkipped = source["metadataBatchSkipped"];
	        this.metadataBatchFailed = source["metadataBatchFailed"];
	        this.metadataBatchBlocked = source["metadataBatchBlocked"];
	        this.metadataBatchCancelled = source["metadataBatchCancelled"];
	        this.metadataBatchStatusUpdated = source["metadataBatchStatusUpdated"];
	        this.metadataBatchStatusSkipped = source["metadataBatchStatusSkipped"];
	        this.metadataBatchStatusFailed = source["metadataBatchStatusFailed"];
	        this.metadataBatchStatusBlocked = source["metadataBatchStatusBlocked"];
	        this.metadataCancel = source["metadataCancel"];
	        this.id3DropZoneHint = source["id3DropZoneHint"];
	        this.noCoverLabel = source["noCoverLabel"];
	        this.changeCoverLabel = source["changeCoverLabel"];
	        this.removeCoverLabel = source["removeCoverLabel"];
	        this.id3TitlePlaceholder = source["id3TitlePlaceholder"];
	        this.id3ArtistPlaceholder = source["id3ArtistPlaceholder"];
	        this.id3AlbumPlaceholder = source["id3AlbumPlaceholder"];
	        this.id3YearPlaceholder = source["id3YearPlaceholder"];
	        this.id3GenrePlaceholder = source["id3GenrePlaceholder"];
	    }
	}

}

export namespace logs {
	
	export class LogEntry {
	    level: string;
	    // Go type: time
	    timestamp: any;
	    thread: string;
	    category: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.thread = source["thread"];
	        this.category = source["category"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MessageCount {
	    message: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new MessageCount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.count = source["count"];
	    }
	}
	export class LogStats {
	    totalFiles: number;
	    totalLines: number;
	    infoCount: number;
	    warningCount: number;
	    errorCount: number;
	    latestLog: string;
	    topWarnings: MessageCount[];
	    topErrors: MessageCount[];
	    recentEntries: LogEntry[];
	
	    static createFrom(source: any = {}) {
	        return new LogStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalFiles = source["totalFiles"];
	        this.totalLines = source["totalLines"];
	        this.infoCount = source["infoCount"];
	        this.warningCount = source["warningCount"];
	        this.errorCount = source["errorCount"];
	        this.latestLog = source["latestLog"];
	        this.topWarnings = this.convertValues(source["topWarnings"], MessageCount);
	        this.topErrors = this.convertValues(source["topErrors"], MessageCount);
	        this.recentEntries = this.convertValues(source["recentEntries"], LogEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class DriveInfo {
	    letter: string;
	    trackCount: number;
	
	    static createFrom(source: any = {}) {
	        return new DriveInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.letter = source["letter"];
	        this.trackCount = source["trackCount"];
	    }
	}
	export class ProcessItem {
	    name: string;
	    pid: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcessItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.pid = source["pid"];
	    }
	}
	export class StatusInfo {
	    installPath: string;
	    engineVersion: string;
	    windowsVersion: string;
	    utf8Enabled: boolean;
	    acpValue: string;
	    manifestConfigured: boolean;
	    isAdmin: boolean;
	    stemsDetected: boolean;
	    dbDetected: boolean;
	    dbPath: string;
	    processRunning: boolean;
	    runningProcesses: ProcessItem[];
	
	    static createFrom(source: any = {}) {
	        return new StatusInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installPath = source["installPath"];
	        this.engineVersion = source["engineVersion"];
	        this.windowsVersion = source["windowsVersion"];
	        this.utf8Enabled = source["utf8Enabled"];
	        this.acpValue = source["acpValue"];
	        this.manifestConfigured = source["manifestConfigured"];
	        this.isAdmin = source["isAdmin"];
	        this.stemsDetected = source["stemsDetected"];
	        this.dbDetected = source["dbDetected"];
	        this.dbPath = source["dbPath"];
	        this.processRunning = source["processRunning"];
	        this.runningProcesses = this.convertValues(source["runningProcesses"], ProcessItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SystemActionResult {
	    operationId: string;
	    changed: number;
	    publicError?: apperror.PublicError;
	
	    static createFrom(source: any = {}) {
	        return new SystemActionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operationId = source["operationId"];
	        this.changed = source["changed"];
	        this.publicError = this.convertValues(source["publicError"], apperror.PublicError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TagInfo {
	    filePath: string;
	    title: string;
	    artist: string;
	    album: string;
	    year: string;
	    genre: string;
	    hasCover: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TagInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.year = source["year"];
	        this.genre = source["genre"];
	        this.hasCover = source["hasCover"];
	    }
	}

}

export namespace metadata {
	
	export class BatchItem {
	    path: string;
	    status: string;
	    code?: string;
	    messageKey?: string;
	    detail?: string;
	    snapshotId?: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.status = source["status"];
	        this.code = source["code"];
	        this.messageKey = source["messageKey"];
	        this.detail = source["detail"];
	        this.snapshotId = source["snapshotId"];
	    }
	}
	export class BatchPreview {
	    operationId: string;
	    total: number;
	    eligible: number;
	    blocked: number;
	    items: BatchItem[];
	
	    static createFrom(source: any = {}) {
	        return new BatchPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operationId = source["operationId"];
	        this.total = source["total"];
	        this.eligible = source["eligible"];
	        this.blocked = source["blocked"];
	        this.items = this.convertValues(source["items"], BatchItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BatchResult {
	    operationId: string;
	    total: number;
	    updated: number;
	    skipped: number;
	    failed: number;
	    blocked: number;
	    cancelled: boolean;
	    items: BatchItem[];
	
	    static createFrom(source: any = {}) {
	        return new BatchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operationId = source["operationId"];
	        this.total = source["total"];
	        this.updated = source["updated"];
	        this.skipped = source["skipped"];
	        this.failed = source["failed"];
	        this.blocked = source["blocked"];
	        this.cancelled = source["cancelled"];
	        this.items = this.convertValues(source["items"], BatchItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Capabilities {
	    format: string;
	    readable: Record<string, boolean>;
	    writable: Record<string, boolean>;
	    maxPictures: number;
	    pictureMIMEs: string[];
	    tagVersions: string[];
	
	    static createFrom(source: any = {}) {
	        return new Capabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.readable = source["readable"];
	        this.writable = source["writable"];
	        this.maxPictures = source["maxPictures"];
	        this.pictureMIMEs = source["pictureMIMEs"];
	        this.tagVersions = source["tagVersions"];
	    }
	}
	export class Warning {
	    code: string;
	    messageKey: string;
	
	    static createFrom(source: any = {}) {
	        return new Warning(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.messageKey = source["messageKey"];
	    }
	}
	export class Picture {
	    id: string;
	    type: number;
	    mime: string;
	    description?: string;
	    data?: number[];
	    thumbnail?: number[];
	    thumbnailMime?: string;
	    bytes?: number;
	    width?: number;
	    height?: number;
	
	    static createFrom(source: any = {}) {
	        return new Picture(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.mime = source["mime"];
	        this.description = source["description"];
	        this.data = source["data"];
	        this.thumbnail = source["thumbnail"];
	        this.thumbnailMime = source["thumbnailMime"];
	        this.bytes = source["bytes"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class NumberPair {
	    value: number;
	    total?: number;
	
	    static createFrom(source: any = {}) {
	        return new NumberPair(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.total = source["total"];
	    }
	}
	export class Document {
	    format: string;
	    tagVersion: string;
	    title: string;
	    artists: string[];
	    albumArtists: string[];
	    album: string;
	    track: NumberPair;
	    disc: NumberPair;
	    date: string;
	    genres: string[];
	    composers: string[];
	    comment: string;
	    bpm: number;
	    key: string;
	    rating: number;
	    isrc: string;
	    lyrics: string;
	    pictures: Picture[];
	    warnings: Warning[];
	    capabilities?: Capabilities;
	
	    static createFrom(source: any = {}) {
	        return new Document(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.tagVersion = source["tagVersion"];
	        this.title = source["title"];
	        this.artists = source["artists"];
	        this.albumArtists = source["albumArtists"];
	        this.album = source["album"];
	        this.track = this.convertValues(source["track"], NumberPair);
	        this.disc = this.convertValues(source["disc"], NumberPair);
	        this.date = source["date"];
	        this.genres = source["genres"];
	        this.composers = source["composers"];
	        this.comment = source["comment"];
	        this.bpm = source["bpm"];
	        this.key = source["key"];
	        this.rating = source["rating"];
	        this.isrc = source["isrc"];
	        this.lyrics = source["lyrics"];
	        this.pictures = this.convertValues(source["pictures"], Picture);
	        this.warnings = this.convertValues(source["warnings"], Warning);
	        this.capabilities = this.convertValues(source["capabilities"], Capabilities);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FieldPatch_EngineTools_internal_metadata_NumberPair_ {
	    mode: string;
	    value?: NumberPair;
	
	    static createFrom(source: any = {}) {
	        return new FieldPatch_EngineTools_internal_metadata_NumberPair_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.value = this.convertValues(source["value"], NumberPair);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FieldPatch___EngineTools_internal_metadata_Picture_ {
	    mode: string;
	    value?: Picture[];
	
	    static createFrom(source: any = {}) {
	        return new FieldPatch___EngineTools_internal_metadata_Picture_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.value = this.convertValues(source["value"], Picture);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FieldPatch___string_ {
	    mode: string;
	    value?: string[];
	
	    static createFrom(source: any = {}) {
	        return new FieldPatch___string_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.value = source["value"];
	    }
	}
	export class FieldPatch_float64_ {
	    mode: string;
	    value?: number;
	
	    static createFrom(source: any = {}) {
	        return new FieldPatch_float64_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.value = source["value"];
	    }
	}
	export class FieldPatch_int_ {
	    mode: string;
	    value?: number;
	
	    static createFrom(source: any = {}) {
	        return new FieldPatch_int_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.value = source["value"];
	    }
	}
	export class FieldPatch_string_ {
	    mode: string;
	    value?: string;
	
	    static createFrom(source: any = {}) {
	        return new FieldPatch_string_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.value = source["value"];
	    }
	}
	
	export class Patch {
	    title: FieldPatch_string_;
	    artists: FieldPatch___string_;
	    album: FieldPatch_string_;
	    albumArtists: FieldPatch___string_;
	    track: FieldPatch_EngineTools_internal_metadata_NumberPair_;
	    disc: FieldPatch_EngineTools_internal_metadata_NumberPair_;
	    date: FieldPatch_string_;
	    genres: FieldPatch___string_;
	    composers: FieldPatch___string_;
	    comment: FieldPatch_string_;
	    bpm: FieldPatch_float64_;
	    key: FieldPatch_string_;
	    rating: FieldPatch_int_;
	    isrc: FieldPatch_string_;
	    lyrics: FieldPatch_string_;
	    pictures: any;
	
	    static createFrom(source: any = {}) {
	        return new Patch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = this.convertValues(source["title"], FieldPatch_string_);
	        this.artists = this.convertValues(source["artists"], FieldPatch___string_);
	        this.album = this.convertValues(source["album"], FieldPatch_string_);
	        this.albumArtists = this.convertValues(source["albumArtists"], FieldPatch___string_);
	        this.track = this.convertValues(source["track"], FieldPatch_EngineTools_internal_metadata_NumberPair_);
	        this.disc = this.convertValues(source["disc"], FieldPatch_EngineTools_internal_metadata_NumberPair_);
	        this.date = this.convertValues(source["date"], FieldPatch_string_);
	        this.genres = this.convertValues(source["genres"], FieldPatch___string_);
	        this.composers = this.convertValues(source["composers"], FieldPatch___string_);
	        this.comment = this.convertValues(source["comment"], FieldPatch_string_);
	        this.bpm = this.convertValues(source["bpm"], FieldPatch_float64_);
	        this.key = this.convertValues(source["key"], FieldPatch_string_);
	        this.rating = this.convertValues(source["rating"], FieldPatch_int_);
	        this.isrc = this.convertValues(source["isrc"], FieldPatch_string_);
	        this.lyrics = this.convertValues(source["lyrics"], FieldPatch_string_);
	        this.pictures = source["pictures"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class WriteResult {
	    ok: boolean;
	    changed: number;
	    snapshotId?: string;
	    operationId?: string;
	
	    static createFrom(source: any = {}) {
	        return new WriteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.changed = source["changed"];
	        this.snapshotId = source["snapshotId"];
	        this.operationId = source["operationId"];
	    }
	}

}

export namespace msi {
	
	export class OrphanedMSI {
	    productCode: string;
	    displayName: string;
	
	    static createFrom(source: any = {}) {
	        return new OrphanedMSI(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.productCode = source["productCode"];
	        this.displayName = source["displayName"];
	    }
	}

}

export namespace unlock {
	
	export class HandleInfo {
	    pid: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new HandleInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.name = source["name"];
	    }
	}

}

