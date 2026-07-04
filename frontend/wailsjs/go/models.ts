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
	    midi2Title: string;
	    midi2Description: string;
	    midi2Enabled: string;
	    midi2Disabled: string;
	    midi2Unavailable: string;
	    midi2DisableButton: string;
	    midi2EnableButton: string;
	    midi2Disabling: string;
	    midi2Enabling: string;
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
	        this.midi2Title = source["midi2Title"];
	        this.midi2Description = source["midi2Description"];
	        this.midi2Enabled = source["midi2Enabled"];
	        this.midi2Disabled = source["midi2Disabled"];
	        this.midi2Unavailable = source["midi2Unavailable"];
	        this.midi2DisableButton = source["midi2DisableButton"];
	        this.midi2EnableButton = source["midi2EnableButton"];
	        this.midi2Disabling = source["midi2Disabling"];
	        this.midi2Enabling = source["midi2Enabling"];
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
	    }
	}

}

export namespace id3 {
	
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

