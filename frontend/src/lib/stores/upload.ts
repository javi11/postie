import { writable } from "svelte/store";

export interface UploadFile {
	id: string;
	name: string;
	size: number;
	progress: number;
	status: "pending" | "uploading" | "processing" | "completed" | "error";
	error?: string;
}

export interface UploadState {
	files: UploadFile[];
	totalProgress: number;
	isUploading: boolean;
	currentRequest: XMLHttpRequest | null;
}

const initialState: UploadState = {
	files: [],
	totalProgress: 0,
	isUploading: false,
	currentRequest: null,
};

export const uploadStore = writable<UploadState>(initialState);

// Helper functions to update the store
export const uploadActions = {
	startUpload: (files: FileList) => {
		const uploadFiles: UploadFile[] = Array.from(files).map((file, index) => ({
			id: `${Date.now()}-${index}`,
			name: file.name,
			size: file.size,
			progress: 0,
			status: "pending" as const,
		}));

		uploadStore.update((state) => ({
			...state,
			files: uploadFiles,
			totalProgress: 0,
			isUploading: true,
		}));

		return uploadFiles;
	},

	updateFileProgress: (
		fileId: string,
		progress: number,
		status?: UploadFile["status"],
	) => {
		uploadStore.update((state) => {
			const files = state.files.map((file) =>
				file.id === fileId
					? { ...file, progress, status: status || file.status }
					: file,
			);

			const totalProgress =
				files.reduce((sum, file) => sum + file.progress, 0) / files.length;

			return {
				...state,
				files,
				totalProgress,
			};
		});
	},

	setFileStatus: (
		fileId: string,
		status: UploadFile["status"],
		error?: string,
	) => {
		uploadStore.update((state) => ({
			...state,
			files: state.files.map((file) =>
				file.id === fileId ? { ...file, status, error } : file,
			),
		}));
	},

	completeUpload: () => {
		uploadStore.update((state) => ({
			...state,
			isUploading: false,
			totalProgress: 100,
		}));
	},

	clearUploads: () => {
		uploadStore.set(initialState);
	},

	setError: (fileId: string, error: string) => {
		uploadStore.update((state) => ({
			...state,
			files: state.files.map((file) =>
				file.id === fileId ? { ...file, status: "error", error } : file,
			),
			isUploading: false,
			currentRequest: null,
		}));
	},

	setCurrentRequest: (request: XMLHttpRequest | null) => {
		uploadStore.update((state) => ({
			...state,
			currentRequest: request,
		}));
	},

	updateTotalProgress: (progress: number) => {
		uploadStore.update((state) => ({
			...state,
			totalProgress: progress,
		}));
	},

	cancelUpload: () => {
		uploadStore.update((state) => {
			// Abort the current request if it exists
			if (state.currentRequest) {
				state.currentRequest.abort();
			}

			return {
				...state,
				files: state.files.map((file) => ({
					...file,
					status:
						file.status === "uploading" || file.status === "processing"
							? ("error" as const)
							: file.status,
					error:
						file.status === "uploading" || file.status === "processing"
							? "Upload cancelled"
							: file.error,
				})),
				isUploading: false,
				currentRequest: null,
			};
		});
	},
};
