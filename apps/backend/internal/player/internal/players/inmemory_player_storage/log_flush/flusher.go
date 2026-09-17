// Package log_flush logs a failed flush of the players.
package log_flush

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
)

func New(inner inmemory_player_storage.Flusher, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  inmemory_player_storage.Flusher
	logger *slog.Logger
}

var _ inmemory_player_storage.Flusher = (*Logged)(nil)

func (l *Logged) Flush(ctx context.Context) error {
	err := l.inner.Flush(ctx)
	if err != nil {
		l.logger.Error("failed to flush the players, retrying next tick", slog.Any("error", err))
	}
	return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
