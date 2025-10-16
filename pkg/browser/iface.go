package browser

import (
	"embed"
	"errors"
	"io/fs"
)

var ErrNoEmbeddedBrowser = errors.New("no browser is embedded in the binary")

type FS interface {
	fs.FS
	fs.ReadDirFS
	fs.ReadFileFS
}

var _ FS = embed.FS{}
