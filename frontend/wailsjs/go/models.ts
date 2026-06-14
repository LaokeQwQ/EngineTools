export namespace database {
	
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
	    }
	}

}

export namespace main {
	
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

