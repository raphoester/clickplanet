package flag_played_subscriber

import (
	"context"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog/usecases/keep_played_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in keep_played_flag_usecase.In) error
}

func NewTaken(useCase UseCase) Taken {
	return Taken{useCase: useCase}
}

// Taken hears planet's own planet.v1.TilesTaken: a take plays for its flag.
type Taken struct {
	useCase UseCase
}

var _ cpredis.BatchHandler[*planetv1.TilesTaken] = Taken{}

// Once per flag of the batch, at its first take: nearly every take is a flag's the game ships with, which it skips.
func (s Taken) HandleBatch(ctx context.Context, events []*planetv1.TilesTaken) error {
	var flags []string
	first := map[string]time.Time{}
	for _, event := range events {
		flag := event.GetCountry()
		if !cpflags.IsOpenedCode(flag) {
			continue
		}
		if _, seen := first[flag]; !seen {
			flags = append(flags, flag)
			first[flag] = event.GetTakenAt().AsTime()
		}
	}
	for _, flag := range flags {
		if err := s.execute(ctx, flag, first[flag]); err != nil {
			return err
		}
	}
	return nil
}

func (s Taken) execute(ctx context.Context, flag string, at time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()
	return s.useCase.Execute(ctx, keep_played_flag_usecase.In{Flag: flag, PlayedAt: at}) //nolint:wrapcheck // the use case named it.
}

func NewPledged(useCase UseCase) Pledged {
	return Pledged{useCase: useCase}
}

// Pledged hears planet's own planet.v1.AllegiancePledged: a pledge plays for its flag.
type Pledged struct {
	useCase UseCase
}

var _ cpredis.Handler[*planetv1.AllegiancePledged] = Pledged{}

func (s Pledged) Handle(ctx context.Context, event *planetv1.AllegiancePledged) error {
	if !cpflags.IsOpenedCode(event.GetCountry()) {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()
	return s.useCase.Execute(ctx, keep_played_flag_usecase.In{ //nolint:wrapcheck // the use case named it.
		Flag: event.GetCountry(), PlayedAt: event.GetPledgedAt().AsTime(),
	})
}
