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
	    refreshingSystem: string;
	    systemRefreshed: string;
	    fixComplete: string;
	    fixCompleteTip: string;
	    fixFailed: string;
	    logPrefix: string;
	    checking: string;
	    statusChecking: string;
	    progressDetecting: string;
	    progressFixing: string;
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
	    libraryStatusLabel: string;
	    restoringOverview: string;
	    restoreComplete: string;
	    restoreError: string;
	    restoreWritten: string;
	    restoreSkipped: string;
	    restoreButton: string;
	    restoreConfirmTitle: string;
	    restoreConfirmMsg: string;
	    restoreDone: string;
	    restoreDoneTip: string;
	    restoreNone: string;
	    restoreAllButton: string;
	
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
	        this.refreshingSystem = source["refreshingSystem"];
	        this.systemRefreshed = source["systemRefreshed"];
	        this.fixComplete = source["fixComplete"];
	        this.fixCompleteTip = source["fixCompleteTip"];
	        this.fixFailed = source["fixFailed"];
	        this.logPrefix = source["logPrefix"];
	        this.checking = source["checking"];
	        this.statusChecking = source["statusChecking"];
	        this.progressDetecting = source["progressDetecting"];
	        this.progressFixing = source["progressFixing"];
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
	        this.libraryStatusLabel = source["libraryStatusLabel"];
	        this.restoringOverview = source["restoringOverview"];
	        this.restoreComplete = source["restoreComplete"];
	        this.restoreError = source["restoreError"];
	        this.restoreWritten = source["restoreWritten"];
	        this.restoreSkipped = source["restoreSkipped"];
	        this.restoreButton = source["restoreButton"];
	        this.restoreConfirmTitle = source["restoreConfirmTitle"];
	        this.restoreConfirmMsg = source["restoreConfirmMsg"];
	        this.restoreDone = source["restoreDone"];
	        this.restoreDoneTip = source["restoreDoneTip"];
	        this.restoreNone = source["restoreNone"];
	        this.restoreAllButton = source["restoreAllButton"];
	    }
	}

}

export namespace main {
	
	export class LibraryInfo {
	    path: string;
	    drive: string;
	    uuid: string;
	    totalTracks: number;
	    missingRGB: number;
	
	    static createFrom(source: any = {}) {
	        return new LibraryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.drive = source["drive"];
	        this.uuid = source["uuid"];
	        this.totalTracks = source["totalTracks"];
	        this.missingRGB = source["missingRGB"];
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

