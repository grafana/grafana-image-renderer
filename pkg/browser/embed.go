//go:build embedded_browser

package browser

import "embed"

//go:embed all:browser
var browserFS embed.FS

func OpenFS() (FS, error) {
	return &browserFS, nil
}
