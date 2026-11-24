package flag_opened_subscriber

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

var ErrNoFlag = errors.New("the fact names no flag")

type Catalog interface {
	Open(ctx context.Context, flag string, at time.Time) error
	Move(ctx context.Context, flag string, pin clicks.Pin, at time.Time) error
}

func New(catalog Catalog) Subscriber {
	return Subscriber{catalog: catalog}
}

type Subscriber struct {
	catalog Catalog
}

var _ cpredis.Handler[*flagsv1.FlagOpened] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *flagsv1.FlagOpened) error {
	if event.GetFlag() == "" {
		return fmt.Errorf("%w: %v", ErrNoFlag, event)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	at := event.GetOpenedAt().AsTime()
	if err := s.catalog.Open(ctx, event.GetFlag(), at); err != nil {
		return err //nolint:wrapcheck // the catalog named it.
	}
	// A fact recorded by a build before pins named none.
	if event.GetPin() == nil {
		return nil
	}
	pin := clicks.Pin{Latitude: event.GetPin().GetLatitude(), Longitude: event.GetPin().GetLongitude()}
	return s.catalog.Move(ctx, event.GetFlag(), pin, at) //nolint:wrapcheck // the catalog named it.
}
