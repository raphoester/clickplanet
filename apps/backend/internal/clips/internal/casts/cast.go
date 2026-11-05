package casts

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type CastID uuid.UUID

var ErrInvalidCastID = errors.New("not a cast id")

func CastIDOf(value string) (CastID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return CastID{}, fmt.Errorf("%w: %q", ErrInvalidCastID, value)
	}
	return CastID(id), nil
}

func (c CastID) String() string { return uuid.UUID(c).String() }

type IDProvider interface {
	NewID() (CastID, error)
}

type AccountID = cpsession.AccountID

var ErrInvalidAccount = errors.New("not an account id")

func AccountIDOf(value string) (AccountID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return AccountID{}, fmt.Errorf("%w: %q", ErrInvalidAccount, value)
	}
	return AccountID(id), nil
}

// A player's number in one cast, from 1.
type Number uint32

type Window struct {
	Since time.Time
	Until time.Time
}

// The ledger keeps who made a change this long: a cast cannot reach further back.
const longest = 72 * time.Hour

var ErrInvalidWindow = errors.New("a cast is made over a since before its until, at most 72 hours apart")

// A zero until is now.
func WindowOf(since, until, now time.Time) (Window, error) {
	if until.IsZero() {
		until = now
	}
	if since.IsZero() || !since.Before(until) || until.Sub(since) > longest {
		return Window{}, ErrInvalidWindow
	}
	return Window{Since: since, Until: until}, nil
}

// A change of the ledger an account made.
type Take struct {
	At       time.Time
	Tile     uint32
	Country  string
	Previous string
	Account  AccountID
}

// Filling empty ground is no fight: a cast counts tiles taken from another flag.
func (t Take) captured() bool {
	return t.Previous != "" && t.Country != "" && t.Previous != t.Country
}

type Cast struct {
	id       CastID
	madeAt   time.Time
	window   Window
	fighters []FighterSnapshot
}

type CastSnapshot struct {
	ID       CastID
	MadeAt   time.Time
	Window   Window
	Fighters []FighterSnapshot
}

type FighterSnapshot struct {
	Number Number
	// Zero, and the name empty, once the account is deleted.
	Account AccountID
	Name    string
}

// A tile taken from another flag, by a fighter of the cast.
type NumberedTake struct {
	At       time.Time
	Tile     uint32
	Country  string
	Previous string
	Fighter  Number
}

// names: the name each account may be shown under in the clips; an account with none is left out. The fighters are
// numbered by tiles taken, the most first, then by name.
func CastOf(id CastID, madeAt time.Time, window Window, takes []Take, names map[AccountID]string) Cast {
	took := map[AccountID]int{}
	for _, take := range takes {
		if _, named := names[take.Account]; named && take.captured() {
			took[take.Account]++
		}
	}

	fighters := make([]FighterSnapshot, 0, len(took))
	for account := range took {
		fighters = append(fighters, FighterSnapshot{Account: account, Name: names[account]})
	}
	slices.SortFunc(fighters, func(a, b FighterSnapshot) int {
		return cmp.Or(
			cmp.Compare(took[b.Account], took[a.Account]),
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.Account.String(), b.Account.String()),
		)
	})
	for i := range fighters {
		fighters[i].Number = Number(i + 1) //nolint:gosec // the players of 72 hours.
	}
	return Cast{id: id, madeAt: madeAt, window: window, fighters: fighters}
}

func (s CastSnapshot) Restore() Cast {
	return Cast{id: s.ID, madeAt: s.MadeAt, window: s.Window, fighters: s.Fighters}
}

func (c Cast) TakeSnapshot() CastSnapshot {
	return CastSnapshot{ID: c.id, MadeAt: c.madeAt, Window: c.window, Fighters: slices.Clone(c.fighters)}
}

// The takes of its fighters from another flag, under their numbers.
func (c Cast) Numbered(takes []Take) []NumberedTake {
	numbers := make(map[AccountID]Number, len(c.fighters))
	for _, fighter := range c.fighters {
		numbers[fighter.Account] = fighter.Number
	}

	numbered := make([]NumberedTake, 0, len(takes))
	for _, take := range takes {
		number, in := numbers[take.Account]
		if !in || !take.captured() {
			continue
		}
		numbered = append(numbered, NumberedTake{
			At: take.At, Tile: take.Tile, Country: take.Country, Previous: take.Previous, Fighter: number,
		})
	}
	return numbered
}

var ErrUnknownFighter = errors.New("a number the cast does not have")

// A fighter whose account was deleted since names no one.
func (c Cast) AccountsOf(numbers []Number) ([]AccountID, error) {
	accounts := make([]AccountID, 0, len(numbers))
	for _, number := range numbers {
		at := slices.IndexFunc(c.fighters, func(fighter FighterSnapshot) bool { return fighter.Number == number })
		if at < 0 {
			return nil, fmt.Errorf("%w: %d", ErrUnknownFighter, number)
		}
		if account := c.fighters[at].Account; account != (AccountID{}) {
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
}

// A run posts its clips within hours of its cast.
const keptFor = 3 * 24 * time.Hour

// The casts made before then are forgotten: which account a number was is needed no longer.
func ForgottenBefore(now time.Time) time.Time {
	return now.Add(-keptFor)
}

var ErrNoCast = errors.New("no cast of that id: forgotten, or never made")
