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
	SourceRecruit Source = "recruit"
	SourceYouTube Source = "youtube"
	SourceUnknown Source = "unknown"
)

func (s Source) known() bool {
	return s == SourceRecruit || s == SourceYouTube || s == SourceUnknown
}

const youtubeTag = "yt"

// The query of the browser's first page of the game, as the client sent it.
type Landing struct {
	Flag string
	Tag  string
}

type Countries interface {
	CheckCountry(code string) bool
}

var ErrInvalidArrival = errors.New("the arrival is invalid")

type Arrival struct {
	id      ArrivalID
	account AccountID
	source  Source
	flag    string
	at      time.Time
}

// The flag is the one the link named, for every source: a link with no tag that names one is a recruit's.
func NewArrival(id ArrivalID, account AccountID, landing Landing, countries Countries, at time.Time) (Arrival, error) {
	if account == (AccountID{}) {
		return Arrival{}, fmt.Errorf("%w: no account", ErrInvalidArrival)
	}
	flag := strings.ToLower(strings.TrimSpace(landing.Flag))
	if !countries.CheckCountry(flag) {
		flag = ""
	}
	return Snapshot{ID: id, Account: account, Source: sourceOf(landing.Tag, flag), Flag: flag, At: at}.Restore()
}

func sourceOf(tag string, flag string) Source {
	switch {
	case strings.EqualFold(strings.TrimSpace(tag), youtubeTag):
		return SourceYouTube
	case flag != "":
		return SourceRecruit
	default:
		return SourceUnknown
	}
}

// The arrival with nothing that says whose it was: the source, the flag and the time stay.
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
	return Snapshot{ID: a.id, Account: a.account, Source: a.source, Flag: a.flag, At: a.at}
}

type Snapshot struct {
	ID ArrivalID
	// Zero once the arrival is anonymized.
	Account AccountID
	Source  Source
	// "" when the link named none.
	Flag string
	At   time.Time
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
	return Arrival{id: s.ID, account: s.Account, source: s.Source, flag: s.Flag, at: s.At}, nil
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
