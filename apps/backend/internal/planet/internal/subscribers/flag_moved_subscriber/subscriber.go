package flag_moved_subscriber

import (
	"context"
	"errors"
	"fmt"
	"time"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

var ErrNoPin = errors.New("the fact names no flag or no pin")

type Catalog interface {
	Move(ctx context.Context, flag string, pin clicks.Pin, at time.Time) error
}

func New(catalog Catalog) Subscriber {
	return Subscriber{catalog: catalog}
}

type Subscriber struct {
	catalog Catalog
}

var _ cpredis.Handler[*flagsv1.FlagMoved] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *flagsv1.FlagMoved) error {
	if event.GetFlag() == "" || event.GetPin() == nil {
		return fmt.Errorf("%w: %v", ErrNoPin, event)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	pin := clicks.Pin{Latitude: event.GetPin().GetLatitude(), Longitude: event.GetPin().GetLongitude()}
	return s.catalog.Move(ctx, event.GetFlag(), pin, event.GetMovedAt().AsTime()) //nolint:wrapcheck // the catalog named it.
}
