package browser

import "context"

type Orientation int

const (
	OrientationUnknown Orientation = iota
	OrientationPortrait
	OrientationLandscape
)

type Cookie struct {
	Name   string
	Value  string
	Domain string
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
	SetPageScale(ctx context.Context, scale float64) error
	SetViewPort(ctx context.Context, width, height int, orientation Orientation) error
	SetExtraHeaders(ctx context.Context, headers map[string]string) error
	SetCookie(ctx context.Context, cookie Cookie) error
	NavigateAndWait(ctx context.Context, url string) error
	Evaluate(ctx context.Context, js string) error
	EvaluateToInt(ctx context.Context, js string) (int, error)
	PrintPDF(ctx context.Context, options PDFOptions) ([]byte, error)
	PrintPNG(ctx context.Context) ([]byte, error)
}
