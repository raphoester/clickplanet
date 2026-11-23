package log_switch_board

import (
	"context"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras/usecases/switch_board_usecase"
)

func New(inner switch_board_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  switch_board_usecase.Executor
	logger *slog.Logger
}

var _ switch_board_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, command switch_board_usecase.Command) (switch_board_usecase.Out, error) {
	started := time.Now()
	out, err := l.inner.Execute(ctx, command)
	carried := out.Carried
	attrs := []any{
		slog.String("map", string(command.Map)), slog.String("board", string(command.Board)), slog.Bool("dryRun", command.DryRun),
		slog.Duration("took", time.Since(started)),
		slog.String("from", string(carried.From)), slog.Int("tiles", int(carried.Tiles)), slog.Int("carried", int(carried.Carried)),
		slog.Int("blank", int(carried.Blank)), slog.Int("heldBefore", int(carried.HeldBefore)),
		slog.Int("heldAfter", int(carried.HeldAfter)), slog.Int("dropped", int(carried.Dropped)),
		slog.Int("droppedHeld", int(carried.DroppedHeld)), slog.Int("shieldsBefore", int(carried.ShieldsBefore)),
		slog.Int("shieldsAfter", int(carried.ShieldsAfter)), slog.Int("minesMoved", int(carried.MinesMoved)),
		slog.Int("minesLost", int(carried.MinesLost)), slog.Int("trapsMoved", int(carried.TrapsMoved)),
		slog.Int("trapsLost", int(carried.TrapsLost)), slog.Int("virusesEnded", int(carried.VirusesEnded)),
		slog.Int("locksDropped", int(carried.LocksDropped)),
	}
	switch {
	case err != nil:
		l.logger.Error("refused to switch the map's board", append(attrs, slog.Any("error", err))...)
	case command.DryRun:
		l.logger.Info("counted a switch of the map's board", attrs...)
	default:
		l.logger.Warn("the map switched boards: every page reloads onto the new one", attrs...)
	}
	return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
