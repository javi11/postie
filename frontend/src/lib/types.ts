export interface QueueItem {
  id: string;
  path: string;
  fileName: string;
  size: number;
  status: 'pending' | 'complete' | 'error';
  retryCount: number;
  priority: number;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
  nzbPath?: string;
}

export interface RunningJobItem {
  id: string;
  status: string;
}

export interface QueueStats {
  total: number;
  pending: number;
  running: number;
  complete: number;
  error: number;
}

export interface ProgressTracker {
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
}

export interface ServerConfig {
  host: string;
  port: number;
  username: string;
  password: string;
  ssl: boolean;
  max_connections: number;
  max_connection_idle_time_in_seconds: number;
  max_connection_ttl_in_seconds: number;
  insecure_ssl: boolean;
}

export interface ConnectionPoolConfig {
  min_connections: number;
  health_check_interval: string;
  skip_providers_verification_on_creation: boolean;
}

export interface CustomHeader {
  name: string;
  value: string;
}

export interface PostHeaders {
  add_ngx_header: boolean;
  default_from: string;
  custom_headers: CustomHeader[];
}

export interface PostingConfig {
  wait_for_par2: boolean;
  max_retries: number;
  retry_delay: string;
  article_size_in_bytes: number;
  groups: string[];
  throttle_rate: number;
  max_workers: number;
  message_id_format: string;
  post_headers: PostHeaders;
  obfuscation_policy: string;
  par2_obfuscation_policy: string;
  group_policy: string;
}

export interface PostCheck {
  enabled: boolean;
  delay: string;
  max_reposts: number;
}

export interface Par2Config {
  enabled: boolean;
  par2_path: string;
  redundancy: string;
  volume_size: number;
  max_input_slices: number;
  extra_par2_options: string[];
}

export interface ScheduleConfig {
  start_time: string;
  end_time: string;
}

export interface WatcherConfig {
  enabled: boolean;
  size_threshold: number;
  schedule: ScheduleConfig;
  ignore_patterns: string[];
  min_file_size: number;
  check_interval: number;
}

export interface NzbCompressionConfig {
  enabled: boolean;
  type: string;
  level: number;
}

export interface QueueConfig {
  database_type: string;
  database_path: string;
  batch_size: number;
  max_retries: number;
  retry_delay: string;
  max_queue_size: number;
  cleanup_after: string;
  priority_processing: boolean;
  max_concurrent_uploads: number;
}

export interface ConfigData {
  servers: ServerConfig[];
  connection_pool: ConnectionPoolConfig;
  posting: PostingConfig;
  post_check: PostCheck;
  par2: Par2Config;
  watcher: WatcherConfig;
  nzb_compression: NzbCompressionConfig;
  queue: QueueConfig;
  output_dir: string;
} 