package told_subscriber

import (
	"context"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/tell_streams_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in tell_streams_usecase.In) error
}

func NewOpened(useCase UseCase) Opened {
	return Opened{useCase: useCase}
}

// Opened hears the module's own flags.v1.FlagOpened, for the open streams.
type Opened struct {
	useCase UseCase
}

var _ cpredis.Handler[*flagsv1.FlagOpened] = Opened{}

func (s Opened) Handle(ctx context.Context, event *flagsv1.FlagOpened) error {
	return s.useCase.Execute(ctx, tell_streams_usecase.In{ //nolint:wrapcheck // the use case named it.
		Flag: event.GetFlag(), DecidedAt: event.GetOpenedAt().AsTime(),
	})
}

func NewClosed(useCase UseCase) Closed {
	return Closed{useCase: useCase}
}

// Closed hears the module's own flags.v1.FlagClosed, for the open streams.
type Closed struct {
	useCase UseCase
}

var _ cpredis.Handler[*flagsv1.FlagClosed] = Closed{}

func (s Closed) Handle(ctx context.Context, event *flagsv1.FlagClosed) error {
	return s.useCase.Execute(ctx, tell_streams_usecase.In{ //nolint:wrapcheck // the use case named it.
		Flag: event.GetFlag(), DecidedAt: event.GetClosedAt().AsTime(),
	})
}
