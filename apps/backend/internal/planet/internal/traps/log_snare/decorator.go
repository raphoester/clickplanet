package log_snare

import (
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func New(snare clicks.Snare, logger *slog.Logger) Decorator {
	return Decorator{snare: snare, logger: logger}
}

type Decorator struct {
	snare  clicks.Snare
	logger *slog.Logger
}

var _ clicks.Snare = Decorator{}

func (d Decorator) Spring(payer clicks.Payer, trap clicks.TrapSnapshot) float64 {
	taken := d.snare.Spring(payer, trap)
	d.logger.Info("trap sprung", slog.String("account", payer.Account), slog.String("scope", payer.Scope),
		slog.Bool("linked", payer.Linked), slog.Uint64("tile", uint64(trap.Tile)), slog.String("flag", trap.Flag),
		slog.String("layer", trap.Account.String()), slog.Float64("clicks", taken))

	return taken
}
