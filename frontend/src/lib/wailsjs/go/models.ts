export namespace backend {
	
	export class AppStatus {
	    hasConfig: boolean;
	    configPath: string;
	    uploading: boolean;
	    criticalConfigError: boolean;
	    error: string;
	    isFirstStart: boolean;
	    hasServers: boolean;
	    serverCount: number;
	    validServerCount: number;
	    configValid: boolean;
	    needsConfiguration: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasConfig = source["hasConfig"];
	        this.configPath = source["configPath"];
	        this.uploading = source["uploading"];
	        this.criticalConfigError = source["criticalConfigError"];
	        this.error = source["error"];
	        this.isFirstStart = source["isFirstStart"];
	        this.hasServers = source["hasServers"];
	        this.serverCount = source["serverCount"];
	        this.validServerCount = source["validServerCount"];
	        this.configValid = source["configValid"];
	        this.needsConfiguration = source["needsConfiguration"];
	    }
	}
	export class MetricSummary {
	    startTime: string;
	    endTime: string;
	    totalConnectionsCreated: number;
	    totalConnectionsDestroyed: number;
	    totalAcquires: number;
	    totalReleases: number;
	    totalErrors: number;
	    totalRetries: number;
	    totalAcquireWaitTime: number;
	    totalBytesDownloaded: number;
	    totalBytesUploaded: number;
	    totalArticlesRetrieved: number;
	    totalArticlesPosted: number;
	    totalCommandCount: number;
	    totalCommandErrors: number;
	    averageConnectionsPerHour: number;
	    averageErrorRate: number;
	    averageSuccessRate: number;
	    averageAcquireWaitTime: number;
	    windowCount: number;
	
	    static createFrom(source: any = {}) {
	        return new MetricSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.startTime = source["startTime"];
	        this.endTime = source["endTime"];
	        this.totalConnectionsCreated = source["totalConnectionsCreated"];
	        this.totalConnectionsDestroyed = source["totalConnectionsDestroyed"];
	        this.totalAcquires = source["totalAcquires"];
	        this.totalReleases = source["totalReleases"];
	        this.totalErrors = source["totalErrors"];
	        this.totalRetries = source["totalRetries"];
	        this.totalAcquireWaitTime = source["totalAcquireWaitTime"];
	        this.totalBytesDownloaded = source["totalBytesDownloaded"];
	        this.totalBytesUploaded = source["totalBytesUploaded"];
	        this.totalArticlesRetrieved = source["totalArticlesRetrieved"];
	        this.totalArticlesPosted = source["totalArticlesPosted"];
	        this.totalCommandCount = source["totalCommandCount"];
	        this.totalCommandErrors = source["totalCommandErrors"];
	        this.averageConnectionsPerHour = source["averageConnectionsPerHour"];
	        this.averageErrorRate = source["averageErrorRate"];
	        this.averageSuccessRate = source["averageSuccessRate"];
	        this.averageAcquireWaitTime = source["averageAcquireWaitTime"];
	        this.windowCount = source["windowCount"];
	    }
	}
	export class NntpProviderMetrics {
	    host: string;
	    username: string;
	    state: string;
	    totalConnections: number;
	    maxConnections: number;
	    acquiredConnections: number;
	    idleConnections: number;
	    totalBytesUploaded: number;
	    totalArticlesPosted: number;
	    successRate: number;
	    averageConnectionAge: number;
	
	    static createFrom(source: any = {}) {
	        return new NntpProviderMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.username = source["username"];
	        this.state = source["state"];
	        this.totalConnections = source["totalConnections"];
	        this.maxConnections = source["maxConnections"];
	        this.acquiredConnections = source["acquiredConnections"];
	        this.idleConnections = source["idleConnections"];
	        this.totalBytesUploaded = source["totalBytesUploaded"];
	        this.totalArticlesPosted = source["totalArticlesPosted"];
	        this.successRate = source["successRate"];
	        this.averageConnectionAge = source["averageConnectionAge"];
	    }
	}
	export class NntpPoolMetrics {
	    timestamp: string;
	    uptime: number;
	    activeConnections: number;
	    uploadSpeed: number;
	    commandSuccessRate: number;
	    errorRate: number;
	    totalAcquires: number;
	    totalBytesUploaded: number;
	    totalArticlesRetrieved: number;
	    totalArticlesPosted: number;
	    averageAcquireWaitTime: number;
	    totalErrors: number;
	    providers: NntpProviderMetrics[];
	    dailyMetrics?: MetricSummary;
	    weeklyMetrics?: MetricSummary;
	
	    static createFrom(source: any = {}) {
	        return new NntpPoolMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.uptime = source["uptime"];
	        this.activeConnections = source["activeConnections"];
	        this.uploadSpeed = source["uploadSpeed"];
	        this.commandSuccessRate = source["commandSuccessRate"];
	        this.errorRate = source["errorRate"];
	        this.totalAcquires = source["totalAcquires"];
	        this.totalBytesUploaded = source["totalBytesUploaded"];
	        this.totalArticlesRetrieved = source["totalArticlesRetrieved"];
	        this.totalArticlesPosted = source["totalArticlesPosted"];
	        this.averageAcquireWaitTime = source["averageAcquireWaitTime"];
	        this.totalErrors = source["totalErrors"];
	        this.providers = this.convertValues(source["providers"], NntpProviderMetrics);
	        this.dailyMetrics = this.convertValues(source["dailyMetrics"], MetricSummary);
	        this.weeklyMetrics = this.convertValues(source["weeklyMetrics"], MetricSummary);
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
	
	export class ProcessorStatus {
	    hasProcessor: boolean;
	    runningJobs: number;
	    runningJobIDs: string[];
	
	    static createFrom(source: any = {}) {
	        return new ProcessorStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasProcessor = source["hasProcessor"];
	        this.runningJobs = source["runningJobs"];
	        this.runningJobIDs = source["runningJobIDs"];
	    }
	}
	export class QueueItem {
	    id: string;
	    path: string;
	    fileName: string;
	    size: number;
	    status: string;
	    retryCount: number;
	    priority: number;
	    errorMessage?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    // Go type: time
	    completedAt?: any;
	    nzbPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new QueueItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.size = source["size"];
	        this.status = source["status"];
	        this.retryCount = source["retryCount"];
	        this.priority = source["priority"];
	        this.errorMessage = source["errorMessage"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.completedAt = this.convertValues(source["completedAt"], null);
	        this.nzbPath = source["nzbPath"];
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
	export class QueueStats {
	    total: number;
	    pending: number;
	    running: number;
	    complete: number;
	    error: number;
	
	    static createFrom(source: any = {}) {
	        return new QueueStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.pending = source["pending"];
	        this.running = source["running"];
	        this.complete = source["complete"];
	        this.error = source["error"];
	    }
	}
	export class ServerData {
	    host: string;
	    port: number;
	    username: string;
	    password: string;
	    ssl: boolean;
	    maxConnections: number;
	
	    static createFrom(source: any = {}) {
	        return new ServerData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.ssl = source["ssl"];
	        this.maxConnections = source["maxConnections"];
	    }
	}
	export class SetupWizardData {
	    servers: ServerData[];
	    outputDirectory: string;
	    watchDirectory: string;
	
	    static createFrom(source: any = {}) {
	        return new SetupWizardData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers = this.convertValues(source["servers"], ServerData);
	        this.outputDirectory = source["outputDirectory"];
	        this.watchDirectory = source["watchDirectory"];
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
	export class ValidationResult {
	    valid: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.error = source["error"];
	    }
	}

}

export namespace config {
	
	export class PostUploadScriptConfig {
	    enabled: boolean;
	    command: string;
	    timeout: string;
	
	    static createFrom(source: any = {}) {
	        return new PostUploadScriptConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.command = source["command"];
	        this.timeout = source["timeout"];
	    }
	}
	export class QueueConfig {
	    database_type: string;
	    database_path: string;
	    max_concurrent_uploads: number;
	
	    static createFrom(source: any = {}) {
	        return new QueueConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.database_type = source["database_type"];
	        this.database_path = source["database_path"];
	        this.max_concurrent_uploads = source["max_concurrent_uploads"];
	    }
	}
	export class NzbCompressionConfig {
	    enabled: boolean;
	    type: string;
	    level: number;
	
	    static createFrom(source: any = {}) {
	        return new NzbCompressionConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.type = source["type"];
	        this.level = source["level"];
	    }
	}
	export class ScheduleConfig {
	    start_time: string;
	    end_time: string;
	
	    static createFrom(source: any = {}) {
	        return new ScheduleConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start_time = source["start_time"];
	        this.end_time = source["end_time"];
	    }
	}
	export class WatcherConfig {
	    enabled: boolean;
	    watch_directory: string;
	    size_threshold: number;
	    schedule: ScheduleConfig;
	    ignore_patterns: string[];
	    min_file_size: number;
	    check_interval: string;
	    delete_original_file: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WatcherConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.watch_directory = source["watch_directory"];
	        this.size_threshold = source["size_threshold"];
	        this.schedule = this.convertValues(source["schedule"], ScheduleConfig);
	        this.ignore_patterns = source["ignore_patterns"];
	        this.min_file_size = source["min_file_size"];
	        this.check_interval = source["check_interval"];
	        this.delete_original_file = source["delete_original_file"];
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
	export class Par2Config {
	    enabled?: boolean;
	    par2_path: string;
	    redundancy: string;
	    volume_size: number;
	    max_input_slices: number;
	    extra_par2_options: string[];
	    temp_dir: string;
	
	    static createFrom(source: any = {}) {
	        return new Par2Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.par2_path = source["par2_path"];
	        this.redundancy = source["redundancy"];
	        this.volume_size = source["volume_size"];
	        this.max_input_slices = source["max_input_slices"];
	        this.extra_par2_options = source["extra_par2_options"];
	        this.temp_dir = source["temp_dir"];
	    }
	}
	export class PostCheck {
	    enabled?: boolean;
	    delay: string;
	    max_reposts: number;
	
	    static createFrom(source: any = {}) {
	        return new PostCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.delay = source["delay"];
	        this.max_reposts = source["max_reposts"];
	    }
	}
	export class CustomHeader {
	    name: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomHeader(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.value = source["value"];
	    }
	}
	export class PostHeaders {
	    add_nxg_header: boolean;
	    default_from: string;
	    custom_headers: CustomHeader[];
	
	    static createFrom(source: any = {}) {
	        return new PostHeaders(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.add_nxg_header = source["add_nxg_header"];
	        this.default_from = source["default_from"];
	        this.custom_headers = this.convertValues(source["custom_headers"], CustomHeader);
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
	export class PostingConfig {
	    wait_for_par2?: boolean;
	    max_retries: number;
	    retry_delay: string;
	    article_size_in_bytes: number;
	    groups: string[];
	    throttle_rate: number;
	    message_id_format: string;
	    post_headers: PostHeaders;
	    obfuscation_policy: string;
	    par2_obfuscation_policy: string;
	    group_policy: string;
	
	    static createFrom(source: any = {}) {
	        return new PostingConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.wait_for_par2 = source["wait_for_par2"];
	        this.max_retries = source["max_retries"];
	        this.retry_delay = source["retry_delay"];
	        this.article_size_in_bytes = source["article_size_in_bytes"];
	        this.groups = source["groups"];
	        this.throttle_rate = source["throttle_rate"];
	        this.message_id_format = source["message_id_format"];
	        this.post_headers = this.convertValues(source["post_headers"], PostHeaders);
	        this.obfuscation_policy = source["obfuscation_policy"];
	        this.par2_obfuscation_policy = source["par2_obfuscation_policy"];
	        this.group_policy = source["group_policy"];
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
	export class ConnectionPoolConfig {
	    min_connections: number;
	    health_check_interval: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionPoolConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.min_connections = source["min_connections"];
	        this.health_check_interval = source["health_check_interval"];
	    }
	}
	export class ServerConfig {
	    host: string;
	    port: number;
	    username: string;
	    password: string;
	    ssl: boolean;
	    max_connections: number;
	    max_connection_idle_time_in_seconds: number;
	    max_connection_ttl_in_seconds: number;
	    insecure_ssl: boolean;
	    enabled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.ssl = source["ssl"];
	        this.max_connections = source["max_connections"];
	        this.max_connection_idle_time_in_seconds = source["max_connection_idle_time_in_seconds"];
	        this.max_connection_ttl_in_seconds = source["max_connection_ttl_in_seconds"];
	        this.insecure_ssl = source["insecure_ssl"];
	        this.enabled = source["enabled"];
	    }
	}
	export class ConfigData {
	    version: number;
	    servers: ServerConfig[];
	    connection_pool: ConnectionPoolConfig;
	    posting: PostingConfig;
	    post_check: PostCheck;
	    par2: Par2Config;
	    watcher: WatcherConfig;
	    nzb_compression: NzbCompressionConfig;
	    queue: QueueConfig;
	    output_dir: string;
	    maintain_original_extension?: boolean;
	    post_upload_script: PostUploadScriptConfig;
	
	    static createFrom(source: any = {}) {
	        return new ConfigData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.servers = this.convertValues(source["servers"], ServerConfig);
	        this.connection_pool = this.convertValues(source["connection_pool"], ConnectionPoolConfig);
	        this.posting = this.convertValues(source["posting"], PostingConfig);
	        this.post_check = this.convertValues(source["post_check"], PostCheck);
	        this.par2 = this.convertValues(source["par2"], Par2Config);
	        this.watcher = this.convertValues(source["watcher"], WatcherConfig);
	        this.nzb_compression = this.convertValues(source["nzb_compression"], NzbCompressionConfig);
	        this.queue = this.convertValues(source["queue"], QueueConfig);
	        this.output_dir = source["output_dir"];
	        this.maintain_original_extension = source["maintain_original_extension"];
	        this.post_upload_script = this.convertValues(source["post_upload_script"], PostUploadScriptConfig);
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

export namespace processor {
	
	export class RunningJobDetails {
	    id: string;
	    path: string;
	    fileName: string;
	    size: number;
	    progress: progress.ProgressState[];
	
	    static createFrom(source: any = {}) {
	        return new RunningJobDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.size = source["size"];
	        this.progress = this.convertValues(source["progress"], progress.ProgressState);
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
	export class RunningJobItem {
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new RunningJobItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	    }
	}

}

export namespace progress {
	
	export class ProgressState {
	    Max: number;
	    CurrentNum: number;
	    CurrentPercent: number;
	    CurrentBytes: number;
	    SecondsSince: number;
	    SecondsLeft: number;
	    KBsPerSecond: number;
	    Description: string;
	    Type: string;
	    IsStarted: boolean;
	    IsPaused: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProgressState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Max = source["Max"];
	        this.CurrentNum = source["CurrentNum"];
	        this.CurrentPercent = source["CurrentPercent"];
	        this.CurrentBytes = source["CurrentBytes"];
	        this.SecondsSince = source["SecondsSince"];
	        this.SecondsLeft = source["SecondsLeft"];
	        this.KBsPerSecond = source["KBsPerSecond"];
	        this.Description = source["Description"];
	        this.Type = source["Type"];
	        this.IsStarted = source["IsStarted"];
	        this.IsPaused = source["IsPaused"];
	    }
	}

}

