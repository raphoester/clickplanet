package admission

import (
	"errors"
	"fmt"
	"slices"
)

var (
	ErrForeignOrigin = errors.New("a call from another page than the admin page")
	ErrNoSession     = errors.New("no live session")
	ErrNotAnOperator = errors.New("not an operator")
)

type Operator string

type Visit struct {
	Origin    string
	Cookie    string
	Procedure string
}

func NewPage(origin string, signIn ...string) Page {
	return Page{origin: origin, signIn: signIn}
}

// Only the admin page's origin may call; its sign-in calls are the ones made before a session exists.
type Page struct {
	origin string
	signIn []string
}

func (p Page) SignIn(procedure string) bool {
	return slices.Contains(p.signIn, procedure)
}

func (p Page) OriginError(origin string) error {
	if origin != p.origin {
		return fmt.Errorf("%w: %q", ErrForeignOrigin, origin)
	}
	return nil
}
