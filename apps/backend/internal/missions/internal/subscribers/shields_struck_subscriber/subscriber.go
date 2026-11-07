package shields_struck_subscriber

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
	Execute(ctx context.Context, strikes ...flagmissions.Strike) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase UseCase
}

var _ cpredis.BatchHandler[*planetv1.ShieldsStruck] = Subscriber{}

var errNoTime = errors.New("the strikes have no time")

// HandleBatch records each flag's shields broken of the batch in one call, in the order they were told: a bomb is one
// write, not one per shield.
func (s Subscriber) HandleBatch(ctx context.Context, events []*planetv1.ShieldsStruck) error {
	var flags []string
	strikes := map[string][]flagmissions.Strike{}
	for _, event := range events {
		if err := event.GetStruckAt().CheckValid(); err != nil {
			return fmt.Errorf("the strikes at %d: %w: %w", event.GetPosition(), errNoTime, err)
		}
		if len(event.GetTiles()) == 0 {
			continue
		}
		flag := event.GetCountry()
		if _, known := strikes[flag]; !known {
			flags = append(flags, flag)
		}
		account, at := subscribers.AccountOrNone(event.GetAccountId()), event.GetStruckAt().AsTime()
		for _, tile := range event.GetTiles() {
			strikes[flag] = append(strikes[flag], flagmissions.Strike{
				Account: account, Flag: flag, Owner: tile.GetOwner(), Position: event.GetPosition(), Index: tile.GetIndex(), At: at,
			})
		}
	}

	for _, flag := range flags {
		if err := s.record(ctx, strikes[flag]); err != nil {
			return err
		}
	}
	return nil
}

func (s Subscriber) record(ctx context.Context, strikes []flagmissions.Strike) error {
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.useCase.Execute(ctx, strikes...) //nolint:wrapcheck // the use case named it.
}
