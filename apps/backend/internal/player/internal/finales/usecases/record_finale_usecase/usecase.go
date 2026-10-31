package record_finale_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

type Scoring interface {
	Table(ctx context.Context, book finales.BookID) ([]finales.Place, error)
	Fighters(ctx context.Context, round finales.RoundID) ([]finales.Fighter, error)
}

type Fights interface {
	Record(ctx context.Context, fights []finales.Fight) error
}

type Awards interface {
	Execute(ctx context.Context, account players.AccountID) (titles.IDs, error)
}

type In struct {
	Season finales.Season
	Book   finales.BookID
	Finale finales.RoundID
}

func New(scoring Scoring, fights Fights, awards Awards) *UseCase {
	return &UseCase{scoring: scoring, fights: fights, awards: awards}
}

type UseCase struct {
	scoring Scoring
	fights  Fights
	awards  Awards
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	table, err := u.scoring.Table(ctx, in.Book)
	if err != nil {
		return fmt.Errorf("failed to read the table of book %q: %w", in.Book, err)
	}
	fighters, err := u.scoring.Fighters(ctx, in.Finale)
	if err != nil {
		return fmt.Errorf("failed to read who fought round %q: %w", in.Finale, err)
	}

	fights, err := finales.Result{Season: in.Season, Table: table, Fighters: fighters}.Fights()
	if err != nil {
		return fmt.Errorf("cannot record the fights of season %d: %w", in.Season, err)
	}
	if err := u.fights.Record(ctx, fights); err != nil {
		return fmt.Errorf("failed to record the fights of season %d: %w", in.Season, err)
	}

	for _, fight := range fights {
		if _, err := u.awards.Execute(ctx, fight.Account()); err != nil {
			return fmt.Errorf("failed to award the titles of a fighter of season %d: %w", in.Season, err)
		}
	}
	return nil
}
