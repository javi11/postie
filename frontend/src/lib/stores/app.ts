import { writable } from 'svelte/store';

export interface AppStatus {
  needsConfiguration: boolean;
  criticalConfigError: boolean;
  [key: string]: any;
}

// Create app status store
export const appStatus = writable<AppStatus>({
  needsConfiguration: false,
  criticalConfigError: false
});

// Create progress store
export const progress = writable({
  currentFile: '',
  totalFiles: 0,
  completedFiles: 0,
  stage: 'Ready',
  details: '',
  isRunning: false,
  lastUpdate: Math.floor(Date.now() / 1000),
  percentage: 0,
  currentFileProgress: 0
});

// Create upload state store
export const isUploading = writable(false); 