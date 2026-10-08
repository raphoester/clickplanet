package marking_take_census

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds/usecases/take_census_usecase"
)

type Race interface {
	MarkCounted()
}

func New(implementation take_census_usecase.Executor, race Race) *Decorator {
	return &Decorator{implementation: implementation, race: race}
}

type Decorator struct {
	implementation take_census_usecase.Executor
	race           Race
}

var _ take_census_usecase.Executor = (*Decorator)(nil)

func (d *Decorator) Execute(ctx context.Context) ([]rounds.Round, error) {
	closed, err := d.implementation.Execute(ctx)
	d.race.MarkCounted()
	return closed, err //nolint:wrapcheck // a decorator adds a mark, not a sentence.
}
