package log_graffiti

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
)

func New(logger *slog.Logger) Reporter {
	return Reporter{logger: logger}
}

// The line an operator finds a graffiti by, at Warn: nothing is reverted or banned yet.
type Reporter struct {
	logger *slog.Logger
}

func (r Reporter) Report(ctx context.Context, flagged graffiti.Graffiti) {
	r.logger.WarnContext(ctx, "graffiti flagged",
		slog.String("kind", flagged.Kind.String()),
		slog.String("account", flagged.Account.String()),
		slog.String("flag", flagged.Flag),
		slog.Int("tiles", len(flagged.Tiles)),
		slog.String("link", flagged.Link),
	)
}
