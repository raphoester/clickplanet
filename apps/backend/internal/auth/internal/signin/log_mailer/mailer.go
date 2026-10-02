package log_mailer

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
)

type Mailer struct {
	logger *slog.Logger
}

var _ signin.Mailer = (*Mailer)(nil)

func New(logger *slog.Logger) *Mailer {
	return &Mailer{logger: logger}
}

func (m *Mailer) Send(_ context.Context, to signin.Address, letter signin.Letter) error {
	m.logger.Warn("a letter was logged, not sent", slog.String("to", string(to)),
		slog.String("subject", letter.Subject), slog.String("text", letter.Text))
	return nil
}
