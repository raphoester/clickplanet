package signin

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type Post struct {
	blocklist Blocklist
	sends     Limiter
	mailer    Mailer
}

func NewPost(blocklist Blocklist, sends Limiter, mailer Mailer) *Post {
	return &Post{blocklist: blocklist, sends: sends, mailer: mailer}
}

func (p *Post) Send(ctx context.Context, to Address, letter Letter) error {
	if p.blocklist.Disposable(to.Domain()) {
		return fmt.Errorf("%w: %s", ErrAddressDisposable, to.Domain())
	}
	// Per mailbox: each form of an address would be a budget of its own, and one inbox could be filled through them.
	if allowed, _ := p.sends.Take(accounts.MailboxOf(string(to)).String()); !allowed {
		return ErrTooManyCodes
	}
	if err := p.mailer.Send(ctx, to, letter); err != nil {
		return fmt.Errorf("failed to send the letter: %w", err)
	}
	return nil
}
