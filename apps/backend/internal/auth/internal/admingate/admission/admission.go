package admission

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

var (
	ErrForeignOrigin = errors.New("a call from another page than the admin page")
	ErrNoSession     = errors.New("no live session")
	ErrNotAnOperator = errors.New("not an operator")
	ErrNoToken       = errors.New("no live access token")
	ErrSessionOnly   = errors.New("a call only the admin page's session makes")
)

type Operator string

type Visit struct {
	Origin    string
	Cookie    string
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
	sessionOnly string
}

func (p Page) WithSessionOnly(service string) Page {
	p.sessionOnly = service
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
	if p.SignIn(procedure) || (p.sessionOnly != "" && strings.HasPrefix(procedure, p.sessionOnly)) {
		return fmt.Errorf("%w: %s", ErrSessionOnly, procedure)
	}
	return nil
}

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

func (c Caller) LogValue() slog.Value {
	return slog.GroupValue(slog.String("operator", string(c.operator)), slog.String("client", c.client))
}
