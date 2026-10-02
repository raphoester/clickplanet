package activity

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// Kind is stored as is, so a kind is never renamed.
type Kind string

const (
	KindClick      Kind = "click"
	KindTake       Kind = "take"
	KindMap        Kind = "map"
	KindStream     Kind = "stream"
	KindBoxOffered Kind = "box_offered"
	KindBoxCaught  Kind = "box_caught"
	KindBoxLapsed  Kind = "box_lapsed"
	KindBoxForeign Kind = "box_foreign"
)

type Outcome string

const (
	OutcomeNone Outcome = ""

	// OutcomeAccepted includes a click the shadow ban dropped: it was answered OK all the same.
	OutcomeAccepted  Outcome = "accepted"
	OutcomeNoOp      Outcome = "noop"
	OutcomeThrottled Outcome = "throttled"
	OutcomeInvalid   Outcome = "invalid"
	OutcomeFailed    Outcome = "failed"
)

type Caller struct {
	Scope    string
	Account  cpsession.AccountID
	SignedIn bool
}

func CallerOf(payer clicks.Payer) Caller {
	account := cpsession.NoAccount
	if id, err := uuid.Parse(payer.Account); err == nil {
		account = cpsession.AccountID(id)
	}

	return Caller{Scope: payer.Scope, Account: account, SignedIn: payer.Linked}
}

func CallerOfScope(scope string) Caller {
	return Caller{Scope: scope, Account: cpsession.NoAccount}
}

type Event struct {
	At     time.Time
	Kind   Kind
	Caller Caller

	Tile    uint32
	Country string

	Outcome Outcome

	Held    string
	Cleared bool

	// KindMap, as asked: End 0 is the end of the map.
	Start  uint32
	End    uint32
	OffMap bool

	Delay time.Duration
}

const maxText = 64

// Trimmed makes each client string storable: one NUL or invalid UTF-8 would fail its batch forever.
func (e Event) Trimmed() Event {
	e.Caller.Scope = safe(e.Caller.Scope)
	e.Country = safe(e.Country)
	e.Held = safe(e.Held)

	return e
}

func safe(text string) string {
	text = strings.ReplaceAll(strings.ToValidUTF8(text, "�"), "\x00", "")
	if len(text) <= maxText {
		return text
	}

	cut := maxText
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}

	return text[:cut]
}

type Recorder interface {
	Record(event Event)
}

type Discard struct{}

func (Discard) Record(Event) {}
