package odsutil

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"math"

	"github.com/autobutler-org/quark/pkg/util/officezip"
	"github.com/autobutler-org/quark/pkg/util/xlsxutil"
)

// canceled reports whether ctx is already done.
func canceled(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// classify turns a package or parse error into one of this package's
// sentinels. Context cancellation is returned as itself.
func classify(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, ErrTooLarge) || errors.Is(err, ErrMalformed) || errors.Is(err, ErrEncrypted) {
		return err
	}
	if errors.Is(err, officezip.ErrMissingPart) {
		return fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	return fmt.Errorf("odsutil: %v", err)
}

func tooWide() error {
	return fmt.Errorf("%w: a sheet is wider than %d columns", ErrTooLarge, xlsxutil.MaxColumns)
}

func tooManyRows() error {
	return fmt.Errorf("%w: the workbook holds more than %d rows", ErrTooLarge, xlsxutil.MaxRows)
}

func tooManySheets() error {
	return fmt.Errorf("%w: the workbook holds more than %d sheets", ErrTooLarge, xlsxutil.MaxSheets)
}

func tooManyCells() error {
	return fmt.Errorf("%w: the workbook holds more than %d cells", ErrTooLarge, xlsxutil.MaxCells)
}

func tooMuchOutput() error {
	return fmt.Errorf("%w: the workbook would expand past %d bytes", ErrTooLarge, maxOutputBytes)
}

func tooLongCell() error {
	return fmt.Errorf("%w: a cell holds more than %d bytes", ErrTooLarge, maxCellBytes)
}

// attrFirst returns the first attribute with the local name, in any
// namespace. Repeat counts and sheet names are unique by local name.
func attrFirst(start xml.StartElement, local string) string {
	for _, a := range start.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// attrOffice returns the attribute in the office namespace, or the first
// attribute of that local name when the office one is absent. Calc's
// extension repeats value-type, and the office attribute names the stored
// value.
func attrOffice(start xml.StartElement, local string) string {
	fallback := ""
	seen := false
	for _, a := range start.Attr {
		if a.Name.Local != local {
			continue
		}
		if a.Name.Space == nsOffice {
			return a.Value
		}
		if !seen {
			fallback = a.Value
			seen = true
		}
	}
	return fallback
}

func mulInt(a, b int64) (int64, bool) {
	if a < 0 || b < 0 {
		return 0, false
	}
	if a != 0 && b > math.MaxInt64/a {
		return 0, false
	}
	return a * b, true
}

func addInt(a, b int64) (int64, bool) {
	if a < 0 || b < 0 || a > math.MaxInt64-b {
		return 0, false
	}
	return a + b, true
}
