package flag_deleted_subscriber

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
	Delete(ctx context.Context, flag string, at time.Time) error
}

func New(catalog Catalog) Subscriber {
	return Subscriber{catalog: catalog}
}

// Subscriber forgets a flag nobody played for: it holds no tile and no player, so nothing else changes.
type Subscriber struct {
	catalog Catalog
}

var _ cpredis.Handler[*flagsv1.FlagDeleted] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *flagsv1.FlagDeleted) error {
	if event.GetFlag() == "" {
		return fmt.Errorf("%w: %v", ErrNoFlag, event)
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return s.catalog.Delete(ctx, event.GetFlag(), event.GetDeletedAt().AsTime()) //nolint:wrapcheck // the catalog named it.
}
