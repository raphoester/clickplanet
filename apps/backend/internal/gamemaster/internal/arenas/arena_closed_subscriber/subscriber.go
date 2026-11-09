package arena_closed_subscriber

import (
	"context"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/arenas/usecases/hear_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const timeout = 5 * time.Second

func New(useCase hear_arena_usecase.UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

type Subscriber struct {
	useCase hear_arena_usecase.UseCase
}

var _ cpredis.Handler[*planetv1.ArenaClosed] = Subscriber{}

// An event that names no arena is dropped: delivered again, it would hold back the ones after it.
func (s Subscriber) Handle(ctx context.Context, event *planetv1.ArenaClosed) error {
	if event.GetArenaId() == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := s.useCase.Execute(ctx, moments.RoundID(event.GetArenaId()))
	return err //nolint:wrapcheck // the use case named it.
}
