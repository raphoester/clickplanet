package graffiti

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
)

type ID uuid.UUID

type Kind string

const Swastika Kind = "swastika"

func (k Kind) known() bool {
	switch k {
	case Swastika:
		return true
	default:
		return false
	}
}

var (
	ErrInvalidGraffiti = errors.New("the graffiti is invalid")
	ErrInvalidAccount  = errors.New("not an account id")
)

func AccountIDOf(value string) (cpsession.AccountID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return cpsession.NoAccount, fmt.Errorf("%w: %q", ErrInvalidAccount, value)
	}
	return cpsession.AccountID(id), nil
}

type Sighting struct {
	Account cpsession.AccountID
	Flag    string
	Kind    Kind
	Drawing drawings.Drawing
	At      time.Time
}

type Graffiti struct {
	id        ID
	account   cpsession.AccountID
	flag      string
	kind      Kind
	tiles     []drawings.TileID
	flaggedAt time.Time
	seenAt    time.Time
	link      string
}

// A graffiti whose painter was forgotten is never seen again.
func (g Graffiti) shows(seen Sighting) bool {
	if g.account == cpsession.NoAccount || g.account != seen.Account || g.flag != seen.Flag || g.kind != seen.Kind {
		return false
	}
	for _, tile := range seen.Drawing.IDs() {
		if slices.Contains(g.tiles, tile) {
			return true
		}
	}
	return false
}

func (g Graffiti) seenAgain(seen Sighting, link string) Graffiti {
	g.tiles = seen.Drawing.IDs()
	g.seenAt = seen.At
	g.link = link
	return g
}

func (g Graffiti) Anonymized() Graffiti {
	g.account = cpsession.NoAccount
	return g
}

func AllAnonymized(painted []Graffiti) []Graffiti {
	anonymized := make([]Graffiti, 0, len(painted))
	for _, graffiti := range painted {
		anonymized = append(anonymized, graffiti.Anonymized())
	}
	return anonymized
}

func (g Graffiti) TakeSnapshot() Snapshot {
	return Snapshot{
		ID: g.id, Account: g.account, Flag: g.flag, Kind: g.kind, Tiles: slices.Clone(g.tiles),
		FlaggedAt: g.flaggedAt, SeenAt: g.seenAt, Link: g.link,
	}
}

type Snapshot struct {
	ID ID
	// NoAccount once the painter's account is forgotten.
	Account   cpsession.AccountID
	Flag      string
	Kind      Kind
	Tiles     []drawings.TileID
	FlaggedAt time.Time
	SeenAt    time.Time
	Link      string
}

func (s Snapshot) Restore() (Graffiti, error) {
	switch {
	case s.ID == ID{}:
		return Graffiti{}, fmt.Errorf("%w: no id", ErrInvalidGraffiti)
	case s.Flag == "":
		return Graffiti{}, fmt.Errorf("%w: no flag", ErrInvalidGraffiti)
	case !s.Kind.known():
		return Graffiti{}, fmt.Errorf("%w: no kind %q", ErrInvalidGraffiti, string(s.Kind))
	case len(s.Tiles) == 0:
		return Graffiti{}, fmt.Errorf("%w: no tile", ErrInvalidGraffiti)
	case s.FlaggedAt.IsZero():
		return Graffiti{}, fmt.Errorf("%w: never flagged", ErrInvalidGraffiti)
	case s.SeenAt.Before(s.FlaggedAt):
		return Graffiti{}, fmt.Errorf("%w: seen before it was flagged", ErrInvalidGraffiti)
	case s.Link == "":
		return Graffiti{}, fmt.Errorf("%w: no link", ErrInvalidGraffiti)
	}

	return Graffiti{
		id: s.ID, account: s.Account, flag: s.Flag, kind: s.Kind, tiles: slices.Clone(s.Tiles),
		flaggedAt: s.FlaggedAt, seenAt: s.SeenAt, link: s.Link,
	}, nil
}

type Storage interface {
	// All of them, or none.
	Save(ctx context.Context, graffiti ...Graffiti) error
	// The graffiti an account painted, first flagged first.
	PaintedBy(ctx context.Context, account cpsession.AccountID) ([]Graffiti, error)
}
