export namespace i18n {
	
	export class Messages {
	    AppTitle: string;
	    InstallPathLabel: string;
	    InstallPathNotFound: string;
	    UTF8StatusLabel: string;
	    UTF8Enabled: string;
	    UTF8Disabled: string;
	    ManifestStatusLabel: string;
	    ManifestExists: string;
	    ManifestNotExists: string;
	    FixButton: string;
	    OpenRegionSettings: string;
	    UTF8AlreadyEnabled: string;
	    UTF8AlreadyEnabledTip: string;
	    ProcessRunningTitle: string;
	    ProcessRunningMessage: string;
	    KillingProcesses: string;
	    ProcessKilled: string;
	    NoProcessRunning: string;
	    WritingManifest: string;
	    ManifestWritten: string;
	    ManifestWriteError: string;
	    ExeNotFound: string;
	    WritingRegistry: string;
	    RegistryWritten: string;
	    RegistryWriteError: string;
	    RefreshingSystem: string;
	    SystemRefreshed: string;
	    FixComplete: string;
	    FixCompleteTip: string;
	    FixFailed: string;
	    LogPrefix: string;
	    Checking: string;
	    StatusChecking: string;
	    ProgressDetecting: string;
	    ProgressFixing: string;
	    ProgressDone: string;
	    Language: string;
	
	    static createFrom(source: any = {}) {
	        return new Messages(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AppTitle = source["AppTitle"];
	        this.InstallPathLabel = source["InstallPathLabel"];
	        this.InstallPathNotFound = source["InstallPathNotFound"];
	        this.UTF8StatusLabel = source["UTF8StatusLabel"];
	        this.UTF8Enabled = source["UTF8Enabled"];
	        this.UTF8Disabled = source["UTF8Disabled"];
	        this.ManifestStatusLabel = source["ManifestStatusLabel"];
	        this.ManifestExists = source["ManifestExists"];
	        this.ManifestNotExists = source["ManifestNotExists"];
	        this.FixButton = source["FixButton"];
	        this.OpenRegionSettings = source["OpenRegionSettings"];
	        this.UTF8AlreadyEnabled = source["UTF8AlreadyEnabled"];
	        this.UTF8AlreadyEnabledTip = source["UTF8AlreadyEnabledTip"];
	        this.ProcessRunningTitle = source["ProcessRunningTitle"];
	        this.ProcessRunningMessage = source["ProcessRunningMessage"];
	        this.KillingProcesses = source["KillingProcesses"];
	        this.ProcessKilled = source["ProcessKilled"];
	        this.NoProcessRunning = source["NoProcessRunning"];
	        this.WritingManifest = source["WritingManifest"];
	        this.ManifestWritten = source["ManifestWritten"];
	        this.ManifestWriteError = source["ManifestWriteError"];
	        this.ExeNotFound = source["ExeNotFound"];
	        this.WritingRegistry = source["WritingRegistry"];
	        this.RegistryWritten = source["RegistryWritten"];
	        this.RegistryWriteError = source["RegistryWriteError"];
	        this.RefreshingSystem = source["RefreshingSystem"];
	        this.SystemRefreshed = source["SystemRefreshed"];
	        this.FixComplete = source["FixComplete"];
	        this.FixCompleteTip = source["FixCompleteTip"];
	        this.FixFailed = source["FixFailed"];
	        this.LogPrefix = source["LogPrefix"];
	        this.Checking = source["Checking"];
	        this.StatusChecking = source["StatusChecking"];
	        this.ProgressDetecting = source["ProgressDetecting"];
	        this.ProgressFixing = source["ProgressFixing"];
	        this.ProgressDone = source["ProgressDone"];
	        this.Language = source["Language"];
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
	    utf8Enabled: boolean;
	    manifestConfigured: boolean;
	    processRunning: boolean;
	    runningProcesses: ProcessItem[];
	
	    static createFrom(source: any = {}) {
	        return new StatusInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installPath = source["installPath"];
	        this.utf8Enabled = source["utf8Enabled"];
	        this.manifestConfigured = source["manifestConfigured"];
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

