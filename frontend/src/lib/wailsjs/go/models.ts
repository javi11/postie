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
	    version: string;
	
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
	        this.version = source["version"];
	    }
	}
	export class NntpProviderMetrics {
	    host: string;
	    activeConnections: number;
	    maxConnections: number;
	    totalErrors: number;
	    avgSpeed: number;
	    missing: number;
	    pingRTT: string;
	    inflight: number;

	    static createFrom(source: any = {}) {
	        return new NntpProviderMetrics(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.activeConnections = source["activeConnections"];
	        this.maxConnections = source["maxConnections"];
	        this.totalErrors = source["totalErrors"];
	        this.avgSpeed = source["avgSpeed"];
	        this.missing = source["missing"];
	        this.pingRTT = source["pingRTT"];
	        this.inflight = source["inflight"];
	    }
	}
	export class NntpPoolMetrics {
	    timestamp: string;
	    activeConnections: number;
	    totalErrors: number;
	    avgSpeed: number;
	    elapsed: string;
	    providerErrors: Record<string, number>;
	    providers: NntpProviderMetrics[];

	    static createFrom(source: any = {}) {
	        return new NntpPoolMetrics(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.activeConnections = source["activeConnections"];
	        this.totalErrors = source["totalErrors"];
	        this.avgSpeed = source["avgSpeed"];
	        this.elapsed = source["elapsed"];
	        this.providerErrors = source["providerErrors"];
	        this.providers = this.convertValues(source["providers"], NntpProviderMetrics);
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
	export class PaginatedQueueResult {
	    items: QueueItem[];
	    totalItems: number;
	    totalPages: number;
	    currentPage: number;
	    itemsPerPage: number;
	    hasNext: boolean;
	    hasPrev: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PaginatedQueueResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], QueueItem);
	        this.totalItems = source["totalItems"];
	        this.totalPages = source["totalPages"];
	        this.currentPage = source["currentPage"];
	        this.itemsPerPage = source["itemsPerPage"];
	        this.hasNext = source["hasNext"];
	        this.hasPrev = source["hasPrev"];
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
	export class PaginationParams {
	    page: number;
	    limit: number;
	    sortBy: string;
	    order: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new PaginationParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page = source["page"];
	        this.limit = source["limit"];
	        this.sortBy = source["sortBy"];
	        this.order = source["order"];
	        this.status = source["status"];
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
	    role: string;
	
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
	        this.role = source["role"] ?? "";
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
	    max_retries: number;
	    retry_delay: string;
	    max_backoff: string;
	    max_retry_duration: string;
	    retry_check_interval: string;
	
	    static createFrom(source: any = {}) {
	        return new PostUploadScriptConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.command = source["command"];
	        this.timeout = source["timeout"];
	        this.max_retries = source["max_retries"];
	        this.retry_delay = source["retry_delay"];
	        this.max_backoff = source["max_backoff"];
	        this.max_retry_duration = source["max_retry_duration"];
	        this.retry_check_interval = source["retry_check_interval"];
	    }
	}
	export class QueueConfig {
	    max_concurrent_uploads: number;
	
	    static createFrom(source: any = {}) {
	        return new QueueConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.max_concurrent_uploads = source["max_concurrent_uploads"];
	    }
	}
	export class DatabaseConfig {
	    database_type: string;
	    database_path: string;
	
	    static createFrom(source: any = {}) {
	        return new DatabaseConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.database_type = source["database_type"];
	        this.database_path = source["database_path"];
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
	    name: string;
	    enabled: boolean;
	    watch_directory: string;
	    size_threshold: number;
	    schedule: ScheduleConfig;
	    ignore_patterns: string[];
	    min_file_size: number;
	    check_interval: string;
	    delete_original_file: boolean;
	    single_nzb_per_folder: boolean;
	    follow_symlinks: boolean;
	    min_file_age: string;

	    static createFrom(source: any = {}) {
	        return new WatcherConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"] ?? "";
	        this.enabled = source["enabled"];
	        this.watch_directory = source["watch_directory"];
	        this.size_threshold = source["size_threshold"];
	        this.schedule = this.convertValues(source["schedule"], ScheduleConfig);
	        this.ignore_patterns = source["ignore_patterns"];
	        this.min_file_size = source["min_file_size"];
	        this.check_interval = source["check_interval"];
	        this.delete_original_file = source["delete_original_file"];
	        this.single_nzb_per_folder = source["single_nzb_per_folder"];
	        this.follow_symlinks = source["follow_symlinks"];
	        this.min_file_age = source["min_file_age"];
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
	    redundancy: string;
	    temp_dir: string;
	    maintain_par2_files?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Par2Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.redundancy = source["redundancy"];
	        this.temp_dir = source["temp_dir"];
	        this.maintain_par2_files = source["maintain_par2_files"];
	    }
	}
	export class PostCheck {
	    enabled?: boolean;
	    delay: string;
	    max_reposts: number;
	    deferred_check_delay: string;
	    deferred_max_retries: number;
	    deferred_max_backoff: string;
	    deferred_check_interval: string;
	
	    static createFrom(source: any = {}) {
	        return new PostCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.delay = source["delay"];
	        this.max_reposts = source["max_reposts"];
	        this.deferred_check_delay = source["deferred_check_delay"];
	        this.deferred_max_retries = source["deferred_max_retries"];
	        this.deferred_max_backoff = source["deferred_max_backoff"];
	        this.deferred_check_interval = source["deferred_check_interval"];
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
	export class NewsgroupConfig {
	    name: string;
	    enabled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NewsgroupConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	    }
	}
	export class PostingConfig {
	    wait_for_par2?: boolean;
	    max_retries: number;
	    retry_delay: string;
	    article_size_in_bytes: number;
	    groups: NewsgroupConfig[];
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
	        this.groups = this.convertValues(source["groups"], NewsgroupConfig);
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
	    role: string;
	    check_only?: boolean;
	    inflight: number;
	    proxy_url?: string;

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
	        this.role = source["role"] || "upload";
	        this.check_only = source["check_only"];
	        this.inflight = source["inflight"];
	        this.proxy_url = source["proxy_url"];
	    }
	}
	export class ConfigData {
	    version: number;
	    servers: ServerConfig[];
	    connection_pool: ConnectionPoolConfig;
	    posting: PostingConfig;
	    post_check: PostCheck;
	    par2: Par2Config;
	    watcher?: WatcherConfig;
	    watchers: WatcherConfig[];
	    nzb_compression: NzbCompressionConfig;
	    database: DatabaseConfig;
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
	        this.watcher = source["watcher"] ? this.convertValues(source["watcher"], WatcherConfig) : undefined;
	        this.watchers = this.convertValues(source["watchers"] || [], WatcherConfig);
	        this.nzb_compression = this.convertValues(source["nzb_compression"], NzbCompressionConfig);
	        this.database = this.convertValues(source["database"], DatabaseConfig);
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

export namespace watcher {
	
	export class WatcherScheduleInfo {
	    start_time: string;
	    end_time: string;
	
	    static createFrom(source: any = {}) {
	        return new WatcherScheduleInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start_time = source["start_time"];
	        this.end_time = source["end_time"];
	    }
	}
	export class WatcherStatusInfo {
	    name: string;
	    enabled: boolean;
	    initialized: boolean;
	    watch_directory: string;
	    check_interval: string;
	    next_run?: string;
	    is_within_schedule: boolean;
	    schedule?: WatcherScheduleInfo;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new WatcherStatusInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"] ?? "";
	        this.enabled = source["enabled"];
	        this.initialized = source["initialized"];
	        this.watch_directory = source["watch_directory"];
	        this.check_interval = source["check_interval"];
	        this.next_run = source["next_run"];
	        this.is_within_schedule = source["is_within_schedule"];
	        this.schedule = this.convertValues(source["schedule"], WatcherScheduleInfo);
	        this.error = source["error"];
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

