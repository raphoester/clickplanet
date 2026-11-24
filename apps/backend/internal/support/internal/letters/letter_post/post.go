package letter_post

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/mail_post"
)

func New(mail cpmail.Mailer, from cpmail.Named, inbox notes.Inbox) *Post {
	return &Post{mail: mail, from: from, inbox: inbox}
}

// Post mails a letter's mail from the inbox, Reply-To the letter's own address, in its thread.
type Post struct {
	mail  cpmail.Mailer
	from  cpmail.Named
	inbox notes.Inbox
}

func (p *Post) Send(ctx context.Context, mail letters.Mail) (cpmail.MessageID, error) {
	id, err := p.mail.Send(ctx, cpmail.Mail{
		From: p.from, To: mail.To, ReplyTo: letters.AddressOf(p.inbox, mail.Letter), Subject: mail.Subject,
		Text: mail.Text, Headers: mail_post.Threaded(mail.Thread),
	})
	if err != nil {
		return "", fmt.Errorf("%w: %w", notes.ErrNotSent, err)
	}
	return id, nil
}
