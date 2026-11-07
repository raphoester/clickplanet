package tile_taken_subscriber

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/paint"
)

type Window interface {
	Keep(take paint.Take)
}

func New(window Window) Subscriber {
	return Subscriber{window: window}
}

type Subscriber struct {
	window Window
}

var _ cpredis.Handler[*planetv1.TileTaken] = Subscriber{}

// Every take with an account, whatever it took: a drawing on empty land is a drawing too.
func (s Subscriber) Handle(_ context.Context, event *planetv1.TileTaken) error {
	if event.GetAccountId() == "" {
		return nil
	}
	account, err := uuid.Parse(event.GetAccountId())
	if err != nil {
		return fmt.Errorf("planet named a taker %q: %w", event.GetAccountId(), err)
	}
	s.window.Keep(paint.Take{
		Account: cpsession.AccountID(account),
		Tile:    drawings.TileID(event.GetTileId()),
		Flag:    event.GetCountry(),
		At:      event.GetTakenAt().AsTime(),
	})
	return nil
}
