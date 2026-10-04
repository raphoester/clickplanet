package signin

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/mail"
	"regexp"
	"strings"
)

var (
	ErrAddressInvalid    = errors.New("this is not an email address")
	ErrAddressDisposable = errors.New("this address is from a service that hands them out for nothing")
	ErrTooManyCodes      = errors.New("too many codes went to this address lately")
	ErrWrongCode         = errors.New("this is not the code that was sent")
)

// RFC 5321's limit on a path.
const maxAddressLength = 254

var hostname = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

type Address string

func AddressOf(raw string) (Address, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) > maxAddressLength {
		return "", fmt.Errorf("%w: it is %d bytes, over %d", ErrAddressInvalid, len(trimmed), maxAddressLength)
	}

	parsed, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrAddressInvalid, err)
	}
	if parsed.Name != "" || parsed.Address != trimmed {
		return "", fmt.Errorf("%w: it is more than a bare address", ErrAddressInvalid)
	}

	address := Address(strings.ToLower(trimmed))
	if !hostname.MatchString(address.Domain()) {
		return "", fmt.Errorf("%w: %q is not a domain mail is sent to", ErrAddressInvalid, address.Domain())
	}
	return address, nil
}

func (a Address) Domain() string {
	return string(a)[strings.LastIndex(string(a), "@")+1:]
}

type Blocklist interface {
	Disposable(domain string) bool
}

type Letter struct {
	subject string
	text    string
	html    string
}

func (l Letter) Subject() string {
	return l.subject
}

func (l Letter) Text() string {
	return l.text
}

func (l Letter) HTML() string {
	return l.html
}

func CodeLetter(code string) Letter {
	minutes := int(ChallengeTTL.Minutes())
	return Letter{
		subject: fmt.Sprintf("Your ClickPlanet code: %s", code),
		text: fmt.Sprintf("Your ClickPlanet sign-in code is:\n\n%s\n\n"+
			"It works for %d minutes, in the browser where you asked for it.\n\n"+
			"Did you not ask for it? Ignore this email. Nobody can sign in without the code.\n", code, minutes),
		html: fmt.Sprintf(`<!doctype html><html><body style="font-family:sans-serif;color:#222">`+
			`<p>Your ClickPlanet sign-in code is:</p>`+
			`<p style="font-size:32px;font-weight:bold;letter-spacing:6px">%s</p>`+
			`<p>It works for %d minutes, in the browser where you asked for it.</p>`+
			`<p style="color:#666">Did you not ask for it? Ignore this email. Nobody can sign in without the code.</p>`+
			`</body></html>`, html.EscapeString(code), minutes),
	}
}

type Mailer interface {
	Send(ctx context.Context, to Address, letter Letter) error
}
