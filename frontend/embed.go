package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:build
var EmbeddedFS embed.FS

// GetBuildFS returns the embedded build filesystem
func GetBuildFS() (fs.FS, error) {
	return fs.Sub(EmbeddedFS, "build")
}
