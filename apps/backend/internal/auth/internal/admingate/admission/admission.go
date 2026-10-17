package admission

import (
	"errors"
	"fmt"
)

var (
	ErrForeignOrigin = errors.New("a call from another page than the admin page")
	ErrNoSession     = errors.New("no live session")
	ErrNotAnOperator = errors.New("not an operator")
)

type Operator string

type Visit struct {
	Origin string
	Cookie string
}

func NewPage(origin string) Page {
	return Page{origin: origin}
}

// The game page holds the same cookie on the same site: only the admin page's origin may ride it.
type Page struct {
	origin string
}

func (p Page) OriginError(origin string) error {
	if origin != p.origin {
		return fmt.Errorf("%w: %q", ErrForeignOrigin, origin)
	}
	return nil
}
