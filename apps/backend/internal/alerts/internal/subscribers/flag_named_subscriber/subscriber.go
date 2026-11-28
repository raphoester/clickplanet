package flag_named_subscriber

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations/usecases/name_flag_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, in name_flag_usecase.In) error
}

// Hears flags.v1.FlagOpened: a sold flag opened, claimed or come back, and each open flag again at every boot.
func NewOpened(useCase UseCase) Opened {
	return Opened{useCase: useCase}
}

type Opened struct {
	useCase UseCase
}

var _ cpredis.Handler[*flagsv1.FlagOpened] = Opened{}

func (s Opened) Handle(ctx context.Context, event *flagsv1.FlagOpened) error {
	return hear(ctx, s.useCase, event.GetFlag(), event.GetName(), event.GetOpenedAt())
}

// Hears flags.v1.FlagRenamed: the operator's rename.
func NewRenamed(useCase UseCase) Renamed {
	return Renamed{useCase: useCase}
}

type Renamed struct {
	useCase UseCase
}

var _ cpredis.Handler[*flagsv1.FlagRenamed] = Renamed{}

func (s Renamed) Handle(ctx context.Context, event *flagsv1.FlagRenamed) error {
	return hear(ctx, s.useCase, event.GetFlag(), event.GetName(), event.GetRenamedAt())
}

// A code that is no sold flag's, or a fact that names none, has nothing to name.
func hear(ctx context.Context, useCase UseCase, code, name string, at *timestamppb.Timestamp) error {
	if !cpflags.IsOpenedCode(code) || name == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	return useCase.Execute(ctx, name_flag_usecase.In{Code: code, Name: name, At: at.AsTime()}) //nolint:wrapcheck // the use case named it.
}
