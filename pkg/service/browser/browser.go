package browser

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-image-renderer/pkg/config"
)

type Orientation int

const (
	OrientationUnknown Orientation = iota
	OrientationPortrait
	OrientationLandscape
)

func (o Orientation) IsValid() bool {
	return o == OrientationPortrait || o == OrientationLandscape
}

func (o Orientation) String() string {
	switch o {
	case OrientationPortrait:
		return "OrientationPortrait"
	case OrientationLandscape:
		return "OrientationLandscape"
	default:
		return fmt.Sprintf("Orientation(%d)", int(o))
	}
}

type PDFOptions struct {
	IncludeBackground bool
	Landscape         bool
	PaperWidth        float64
	PaperHeight       float64
	Scale             float64
	PageRanges        string
}

type Browser interface {
	GetPID(ctx context.Context) (int32, error)
	GetCurrentNetworkRequests(ctx context.Context) (int, error)
	SetPageScale(ctx context.Context, scale float64) error
	SetViewPort(ctx context.Context, width, height int, orientation Orientation) error
	SetExtraHeaders(ctx context.Context, headers map[string]string) error
	SetCookie(ctx context.Context, cookie config.Cookie) error
	NavigateAndWait(ctx context.Context, url string) error
	Evaluate(ctx context.Context, js string) error
	EvaluateToInt(ctx context.Context, js string) (int, error)
	PrintPDF(ctx context.Context, options PDFOptions) ([]byte, error)
	PrintPNG(ctx context.Context) ([]byte, error)
	PutDownloadsIn(ctx context.Context, dir string) error
}

type Action = func(context.Context, Browser) error

type Printer interface {
	ContentType() string
}
