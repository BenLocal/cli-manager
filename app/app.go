package app

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// GetAppFS returns the dist folder as the FS root.
func GetAppFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// should not happen; return original FS to avoid panic
		return distFS
	}
	return sub
}
