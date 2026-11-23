package signin

import (
	"context"
	"errors"
	"fmt"
	"html"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

var (
	ErrAddressDisposable = errors.New("this address is from a service that hands them out for nothing")
	ErrTooManyCodes      = errors.New("too many codes went to this address lately")
	ErrWrongCode         = errors.New("this is not the code that was sent")
)

type Address = cpmail.Address

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
