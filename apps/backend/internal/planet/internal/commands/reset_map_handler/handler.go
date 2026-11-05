package reset_map_handler

import (
	"context"
	"errors"
	"time"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets/usecases/reset_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

func New(reset reset_map_usecase.Executor) Handler {
	return Handler{reset: reset}
}

type Handler struct {
	reset reset_map_usecase.Executor
}

var _ cpredis.Handler[*planetv1.ResetMap] = Handler{}

// No timeout of its own: the wipe deletes the whole map in one transaction, and a cut one is delivered again.
func (h Handler) Handle(ctx context.Context, command *planetv1.ResetMap) error {
	var at time.Time
	if command.GetDecidedAt() != nil {
		at = command.GetDecidedAt().AsTime()
	}
	reset, err := resets.New(command.GetResetId(), at)
	// A command that names no reset is dropped: delivered again, it would hold back the commands behind it.
	if errors.Is(err, resets.ErrInvalidReset) {
		return nil
	}
	if err != nil {
		return err //nolint:wrapcheck // resets.New names it.
	}
	_, err = h.reset.Execute(ctx, reset)
	return err //nolint:wrapcheck // the use case named it.
}
