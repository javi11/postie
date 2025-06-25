export namespace backend {
	
	export class ProgressTracker {
	    currentFile: string;
	    totalFiles: number;
	    completedFiles: number;
	    stage: string;
	    details: string;
	    isRunning: boolean;
	    lastUpdate: number;
	    percentage: number;
	    currentFileProgress: number;
	    jobID: string;
	    totalBytes: number;
	    transferredBytes: number;
	    currentFileBytes: number;
	    speed: number;
	    secondsLeft: number;
	    elapsedTime: number;
	
	    static createFrom(source: any = {}) {
	        return new ProgressTracker(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentFile = source["currentFile"];
	        this.totalFiles = source["totalFiles"];
	        this.completedFiles = source["completedFiles"];
	        this.stage = source["stage"];
	        this.details = source["details"];
	        this.isRunning = source["isRunning"];
	        this.lastUpdate = source["lastUpdate"];
	        this.percentage = source["percentage"];
	        this.currentFileProgress = source["currentFileProgress"];
	        this.jobID = source["jobID"];
	        this.totalBytes = source["totalBytes"];
	        this.transferredBytes = source["transferredBytes"];
	        this.currentFileBytes = source["currentFileBytes"];
	        this.speed = source["speed"];
	        this.secondsLeft = source["secondsLeft"];
	        this.elapsedTime = source["elapsedTime"];
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

}

export namespace config {
	
	export class PostUploadScriptConfig {
	    enabled: boolean;
	    command: string;
	    timeout: number;
	
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
	    batch_size: number;
	    max_retries: number;
	    retry_delay: number;
	    max_queue_size: number;
	    cleanup_after: number;
	    priority_processing: boolean;
	    max_concurrent_uploads: number;
	
	    static createFrom(source: any = {}) {
	        return new QueueConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.database_type = source["database_type"];
	        this.database_path = source["database_path"];
	        this.batch_size = source["batch_size"];
	        this.max_retries = source["max_retries"];
	        this.retry_delay = source["retry_delay"];
	        this.max_queue_size = source["max_queue_size"];
	        this.cleanup_after = source["cleanup_after"];
	        this.priority_processing = source["priority_processing"];
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
	    check_interval: number;
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
	    }
	}
	export class PostCheck {
	    enabled?: boolean;
	    delay: number;
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
	    add_ngx_header: boolean;
	    default_from: string;
	    custom_headers: CustomHeader[];
	
	    static createFrom(source: any = {}) {
	        return new PostHeaders(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.add_ngx_header = source["add_ngx_header"];
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
	    retry_delay: number;
	    article_size_in_bytes: number;
	    groups: string[];
	    throttle_rate: number;
	    max_workers: number;
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
	        this.max_workers = source["max_workers"];
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
	    health_check_interval: number;
	    skip_providers_verification_on_creation: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionPoolConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.min_connections = source["min_connections"];
	        this.health_check_interval = source["health_check_interval"];
	        this.skip_providers_verification_on_creation = source["skip_providers_verification_on_creation"];
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
	    }
	}
	export class ConfigData {
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
	    status: string;
	    stage: string;
	    progress: number;
	
	    static createFrom(source: any = {}) {
	        return new RunningJobDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.size = source["size"];
	        this.status = source["status"];
	        this.stage = source["stage"];
	        this.progress = source["progress"];
	    }
	}
	export class RunningJobItem {
	    id: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new RunningJobItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	    }
	}

}

