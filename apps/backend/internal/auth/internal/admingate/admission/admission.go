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
	ErrNoToken       = errors.New("no live access token")
	ErrSessionOnly   = errors.New("a call only the admin page's session makes")
)

type Operator string

// What a call carries to say who makes it: the admin page's cookie, or the access token of an app it connected.
type Visit struct {
	Origin string
	Cookie string
	// The access token after "Bearer ", empty with none.
	Token     string
	Procedure string
}

func NewPage(origin string, signIn ...string) Page {
	return Page{origin: origin, signIn: signIn}
}

// Only the admin page's origin may call; its sign-in calls are the ones made before a session exists.
type Page struct {
	origin      string
	signIn      []string
	sessionOnly []string
}

// The calls only the page's session makes, beside its sign-in: an app cannot let another app in.
func (p Page) WithSessionOnly(procedures ...string) Page {
	p.sessionOnly = slices.Concat(p.sessionOnly, procedures)
	return p
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

func (p Page) TokenError(procedure string) error {
	if p.SignIn(procedure) || slices.Contains(p.sessionOnly, procedure) {
		return fmt.Errorf("%w: %s", ErrSessionOnly, procedure)
	}
	return nil
}

// Who a call was let through as: an operator, and the app it connected when the call came with a token. Nobody for
// the sign-in calls, which are made before a session exists.
type Caller struct {
	operator Operator
	client   string
}

func SessionOf(operator Operator) Caller {
	return Caller{operator: operator}
}

func ConnectionOf(operator Operator, client string) Caller {
	return Caller{operator: operator, client: client}
}

func (c Caller) Operator() Operator {
	return c.operator
}

// The name the app gave itself when it registered; empty for the admin page's own session.
func (c Caller) Client() string {
	return c.client
}
