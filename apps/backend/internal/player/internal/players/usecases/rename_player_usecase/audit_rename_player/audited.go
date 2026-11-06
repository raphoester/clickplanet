package audit_rename_player

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/rename_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in rename_player_usecase.In) (players.Profile, error)
}

func New(inner UseCase, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  UseCase
	logger *slog.Logger
}

var _ UseCase = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context, in rename_player_usecase.In) (players.Profile, error) {
	profile, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.String("account", in.Account.String()), slog.String("name", in.Name), slog.String("reason", in.Reason),
	}
	if err != nil {
		a.logger.Warn("admin player rename failed", append(attrs, slog.Any("error", err))...)
		return profile, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin player rename", attrs...)
	return profile, nil
}
