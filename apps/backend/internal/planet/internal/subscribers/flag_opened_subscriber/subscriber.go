package flag_opened_subscriber

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

type Catalog interface {
	Open(ctx context.Context, flag string, at time.Time) error
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

	return s.catalog.Open(ctx, event.GetFlag(), event.GetOpenedAt().AsTime()) //nolint:wrapcheck // the catalog named it.
}
