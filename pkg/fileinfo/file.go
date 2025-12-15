package fileinfo

type FileInfo struct {
	Path         string
	Size         uint64
	RelativePath string // Path relative to root folder for subject generation (e.g., "MyFolder/subfolder/file.mp4")
}
