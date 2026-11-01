package record_podium_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

type Scoring interface {
	Board(ctx context.Context, book finales.BookID) ([]podiums.Entry, error)
}

type Places interface {
	Record(ctx context.Context, places []podiums.Place) error
}

type Awards interface {
	Execute(ctx context.Context, account players.AccountID) (titles.IDs, error)
}

type In struct {
	Season finales.Season
	Book   finales.BookID
}

func New(scoring Scoring, places Places, awards Awards) *UseCase {
	return &UseCase{scoring: scoring, places: places, awards: awards}
}

type UseCase struct {
	scoring Scoring
	places  Places
	awards  Awards
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	entries, err := u.scoring.Board(ctx, in.Book)
	if err != nil {
		return fmt.Errorf("failed to read the board of book %q: %w", in.Book, err)
	}

	places := podiums.Board{Season: in.Season, Entries: entries}.Places()
	if err := u.places.Record(ctx, places); err != nil {
		return fmt.Errorf("failed to record the podium of season %d: %w", in.Season, err)
	}
	for _, place := range places {
		if _, err := u.awards.Execute(ctx, place.Account()); err != nil {
			return fmt.Errorf("failed to award the titles of a player on the podium of season %d: %w", in.Season, err)
		}
	}
	return nil
}
