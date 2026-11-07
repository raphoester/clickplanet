package flag_tiles_taken_subscriber

import (
	"context"
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, takes ...flagmissions.Take) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.BatchHandler[*planetv1.TilesTaken] = Subscriber{}

var errNoTime = errors.New("the takes have no time")

// HandleBatch records each flag's takes of the batch in one call, in the order they were told: an enclose is one write,
// not one per tile.
func (s Subscriber) HandleBatch(ctx context.Context, events []*planetv1.TilesTaken) error {
	var flags []string
	takes := map[string][]flagmissions.Take{}
	for _, event := range events {
		if err := event.GetTakenAt().CheckValid(); err != nil {
			return fmt.Errorf("the takes at %d: %w: %w", event.GetPosition(), errNoTime, err)
		}
		if len(event.GetTiles()) == 0 {
			continue
		}
		flag := event.GetCountry()
		if _, known := takes[flag]; !known {
			flags = append(flags, flag)
		}
		account, at := subscribers.AccountOrNone(event.GetAccountId()), event.GetTakenAt().AsTime()
		for _, tile := range event.GetTiles() {
			takes[flag] = append(takes[flag], flagmissions.Take{
				Account: account, Flag: flag, Ground: tile.GetGround(), Position: event.GetPosition(), Index: tile.GetIndex(), At: at,
			})
		}
	}

	for _, flag := range flags {
		if err := s.record(ctx, takes[flag]); err != nil {
			return err
		}
	}
	return nil
}

func (s Subscriber) record(ctx context.Context, takes []flagmissions.Take) error {
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, takes...) //nolint:wrapcheck // the use case named it.
}
