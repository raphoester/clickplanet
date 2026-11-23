package flag_closed_subscriber

import (
	"context"
	"errors"
	"fmt"
	"time"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

var ErrNoFlag = errors.New("the fact names no flag")

// Catalog is where a close goes: step 2 of the regional flags hands a use case that also asks the flag's players to
// pick another flag.
type Catalog interface {
	Close(ctx context.Context, flag string, at time.Time) error
}

func New(catalog Catalog) Subscriber {
	return Subscriber{catalog: catalog}
}

type Subscriber struct {
	catalog Catalog
}

var _ cpredis.Handler[*flagsv1.FlagClosed] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *flagsv1.FlagClosed) error {
	if event.GetFlag() == "" {
		return fmt.Errorf("%w: %v", ErrNoFlag, event)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.catalog.Close(ctx, event.GetFlag(), event.GetClosedAt().AsTime()) //nolint:wrapcheck // the catalog named it.
}
