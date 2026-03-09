package handler

import (
	"io/fs"

	"github.com/benlocal/cli-manager/app"
)

func embeddedUIFS() (fs.FS, error) {
	return fs.Sub(app.Dist, "dist")
}
