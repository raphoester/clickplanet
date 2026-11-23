package cpmail_mailer

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

type Mailer struct {
	mail cpmail.Mailer
	from cpmail.Named
}

var _ signin.Mailer = (*Mailer)(nil)

func New(mail cpmail.Mailer, from cpmail.Named) *Mailer {
	return &Mailer{mail: mail, from: from}
}

func (m *Mailer) Send(ctx context.Context, to signin.Address, letter signin.Letter) error {
	if _, err := m.mail.Send(ctx, cpmail.Mail{
		From: m.from, To: to, Subject: letter.Subject(), Text: letter.Text(), HTML: letter.HTML(),
	}); err != nil {
		return fmt.Errorf("failed to mail the letter: %w", err)
	}
	return nil
}
