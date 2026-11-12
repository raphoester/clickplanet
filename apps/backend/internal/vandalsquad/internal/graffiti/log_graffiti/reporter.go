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
	seen := flagged.TakeSnapshot()
	r.logger.WarnContext(ctx, "graffiti flagged",
		slog.String("kind", string(seen.Kind)),
		slog.String("account", seen.Account.String()),
		slog.String("flag", seen.Flag),
		slog.Int("tiles", len(seen.Tiles)),
		slog.String("link", seen.Link),
	)
}
