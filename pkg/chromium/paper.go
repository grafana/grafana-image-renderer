package chromium

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
)

var (
	_ encoding.TextUnmarshaler = (*PaperSize)(nil)
	_ json.Unmarshaler         = (*PaperSize)(nil)
)

// PaperSize is the size of various paper formats.
//
// Ref: https://pptr.dev/api/puppeteer.paperformat#remarks
type PaperSize string

// The valid paper sizes.
// These are all lower-case (even though it is not standard for e.g. A4), as that's easier to match against with case-insensitivity.
const (
	PaperLetter  PaperSize = "letter"
	PaperLegal   PaperSize = "legal"
	PaperTabloid PaperSize = "tabloid"
	PaperLedger  PaperSize = "ledger"
	PaperA0      PaperSize = "a0"
	PaperA1      PaperSize = "a1"
	PaperA2      PaperSize = "a2"
	PaperA3      PaperSize = "a3"
	PaperA4      PaperSize = "a4"
	PaperA5      PaperSize = "a5"
	PaperA6      PaperSize = "a6"
)

func (p *PaperSize) UnmarshalText(text []byte) error {
	text = bytes.ToLower(text)
	switch PaperSize(text) {
	case PaperLetter, PaperLegal, PaperTabloid, PaperLedger,
		PaperA0, PaperA1, PaperA2, PaperA3, PaperA4, PaperA5, PaperA6:
		*p = PaperSize(text)
		return nil
	default:
		return fmt.Errorf("invalid paper size name: %q", text)
	}
}

func (p *PaperSize) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("failed to unmarshal paper size: %w", err)
	}
	return p.UnmarshalText([]byte(text))
}

// FormatInches returns the dimensions of the paper size in inches.
// If the paper size is unknown, (-1, -1) is returned.
func (p PaperSize) FormatInches() (width float64, height float64, err error) {
	switch p {
	case PaperLetter:
		return 8.5, 11, nil
	case PaperLegal:
		return 8.5, 14, nil
	case PaperTabloid:
		return 11, 17, nil
	case PaperLedger:
		return 17, 11, nil
	case PaperA0:
		return 33.1102, 46.811, nil
	case PaperA1:
		return 23.3858, 33.1102, nil
	case PaperA2:
		return 16.5354, 23.3858, nil
	case PaperA3:
		return 11.6929, 16.5354, nil
	case PaperA4:
		return 8.2677, 11.6929, nil
	case PaperA5:
		return 5.8268, 8.2677, nil
	case PaperA6:
		return 4.1339, 5.8268, nil
	default:
		return -1, -1, fmt.Errorf("unknown paper size: %q", p)
	}
}
