package signin

import (
	"context"
	"fmt"
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
	if allowed, _ := p.sends.Take(string(to)); !allowed {
		return ErrTooManyCodes
	}
	if err := p.mailer.Send(ctx, to, letter); err != nil {
		return fmt.Errorf("failed to send the letter: %w", err)
	}
	return nil
}
