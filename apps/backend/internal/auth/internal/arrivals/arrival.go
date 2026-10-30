package arrivals

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type ArrivalID uuid.UUID

type AccountID = accounts.AccountID

type Source string

const (
	SourceRecruit    Source = "recruit"
	SourceYouTube    Source = "youtube"
	SourceCrazyGames Source = "crazygames"
	SourceUnknown    Source = "unknown"
)

func (s Source) known() bool {
	return s == SourceRecruit || s == SourceYouTube || s == SourceCrazyGames || s == SourceUnknown
}

const (
	recruitTag    = "rc"
	youtubeTag    = "yt"
	crazyGamesTag = "cg"
)

// What the browser's first page said, as the client sent it: its ?s= and the host it came from.
type Landing struct {
	Tag      string
	Referrer string
}

const maxHostLength = 253

var ErrInvalidArrival = errors.New("the arrival is invalid")

type Arrival struct {
	id       ArrivalID
	account  AccountID
	source   Source
	referrer string
	at       time.Time
}

func NewArrival(id ArrivalID, account AccountID, landing Landing, at time.Time) (Arrival, error) {
	if account == (AccountID{}) {
		return Arrival{}, fmt.Errorf("%w: no account", ErrInvalidArrival)
	}
	return Snapshot{
		ID: id, Account: account, Source: sourceOf(landing.Tag), Referrer: hostOf(landing.Referrer), At: at,
	}.Restore()
}

// A tag nobody posted is no source: anybody can type one.
func sourceOf(tag string) Source {
	switch strings.ToLower(strings.TrimSpace(tag)) {
	case recruitTag:
		return SourceRecruit
	case youtubeTag:
		return SourceYouTube
	case crazyGamesTag:
		return SourceCrazyGames
	default:
		return SourceUnknown
	}
}

// A host name and nothing else: a path, a query, a port or a whole address is "".
func hostOf(referrer string) string {
	host := strings.ToLower(strings.TrimSpace(referrer))
	if len(host) > maxHostLength {
		return ""
	}
	for _, r := range host {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '.' && r != '-' {
			return ""
		}
	}
	return host
}

// The arrival with nothing that says whose it was: the source, the referrer and the time stay.
func (a Arrival) Anonymized() Arrival {
	a.account = AccountID{}
	return a
}

func AllAnonymized(kept []Arrival) []Arrival {
	anonymized := make([]Arrival, 0, len(kept))
	for _, arrival := range kept {
		anonymized = append(anonymized, arrival.Anonymized())
	}
	return anonymized
}

func (a Arrival) TakeSnapshot() Snapshot {
	return Snapshot{ID: a.id, Account: a.account, Source: a.source, Referrer: a.referrer, At: a.at}
}

type Snapshot struct {
	ID ArrivalID
	// Zero once the arrival is anonymized.
	Account AccountID
	Source  Source
	// The host the browser came from, "" for none.
	Referrer string
	At       time.Time
}

func (s Snapshot) Restore() (Arrival, error) {
	switch {
	case s.ID == ArrivalID{}:
		return Arrival{}, fmt.Errorf("%w: no id", ErrInvalidArrival)
	case !s.Source.known():
		return Arrival{}, fmt.Errorf("%w: unknown source %q", ErrInvalidArrival, string(s.Source))
	case s.At.IsZero():
		return Arrival{}, fmt.Errorf("%w: no time", ErrInvalidArrival)
	}
	return Arrival{id: s.ID, account: s.Account, source: s.Source, referrer: s.Referrer, at: s.At}, nil
}

// A store finds arrivals and saves them. It holds no rule: what changes an arrival is a method of Arrival.
type Store interface {
	// An arrival saved again replaces itself. All of them, or none.
	Save(ctx context.Context, arrivals ...Arrival) error
	// The arrivals that still name the account, oldest first.
	Of(ctx context.Context, account AccountID) ([]Arrival, error)
}

type IDProvider interface {
	NewID() (ArrivalID, error)
}
