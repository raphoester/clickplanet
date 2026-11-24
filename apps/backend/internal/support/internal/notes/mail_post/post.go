package mail_post

import (
	"context"
	"fmt"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func New(mail cpmail.Mailer, from cpmail.Named, inbox notes.Inbox) *Post {
	return &Post{mail: mail, from: from, inbox: inbox}
}

type Post struct {
	mail  cpmail.Mailer
	from  cpmail.Named
	inbox notes.Inbox
}

func (p *Post) Send(ctx context.Context, letter notes.Letter) (cpmail.MessageID, error) {
	id, err := p.mail.Send(ctx, cpmail.Mail{
		From: p.from, To: letter.To, ReplyTo: p.inbox.Of(letter.Note), Subject: letter.Subject, Text: letter.Text,
		Headers: threaded(letter.Thread),
	})
	if err != nil {
		return "", fmt.Errorf("%w: %w", notes.ErrNotSent, err)
	}
	return id, nil
}

func threaded(thread []cpmail.MessageID) map[string]string {
	if len(thread) == 0 {
		return nil
	}
	references := make([]string, 0, len(thread))
	for _, id := range thread {
		references = append(references, id.Header())
	}
	return map[string]string{"In-Reply-To": thread[len(thread)-1].Header(), "References": strings.Join(references, " ")}
}
