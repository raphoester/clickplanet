package cpflags

import (
	"context"
	"errors"
	"fmt"
	"time"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

var ErrNoFlag = errors.New("the fact names no flag")

// Facts is what each fact changes: the catalog, or a module's use case that changes more with it.
type Facts interface {
	Open(ctx context.Context, flag string, at time.Time) error
	Close(ctx context.Context, flag string, at time.Time) error
}

func NewOpenedHandler(catalog Facts) OpenedHandler {
	return OpenedHandler{catalog: catalog}
}

// OpenedHandler hears flags.v1.FlagOpened.
type OpenedHandler struct {
	catalog Facts
}

var _ cpredis.Handler[*flagsv1.FlagOpened] = OpenedHandler{}

func (h OpenedHandler) Handle(ctx context.Context, event *flagsv1.FlagOpened) error {
	if event.GetFlag() == "" {
		return fmt.Errorf("%w: %v", ErrNoFlag, event)
	}
	return h.catalog.Open(ctx, event.GetFlag(), event.GetOpenedAt().AsTime()) //nolint:wrapcheck // the catalog named it.
}

func NewClosedHandler(catalog Facts) ClosedHandler {
	return ClosedHandler{catalog: catalog}
}

// ClosedHandler hears flags.v1.FlagClosed.
type ClosedHandler struct {
	catalog Facts
}

var _ cpredis.Handler[*flagsv1.FlagClosed] = ClosedHandler{}

func (h ClosedHandler) Handle(ctx context.Context, event *flagsv1.FlagClosed) error {
	if event.GetFlag() == "" {
		return fmt.Errorf("%w: %v", ErrNoFlag, event)
	}
	return h.catalog.Close(ctx, event.GetFlag(), event.GetClosedAt().AsTime()) //nolint:wrapcheck // the catalog named it.
}
