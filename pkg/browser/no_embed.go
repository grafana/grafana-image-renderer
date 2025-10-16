//go:build !embedded_browser

package browser

func OpenFS() (FS, error) {
	return nil, ErrNoEmbeddedBrowser
}
