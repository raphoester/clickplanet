package mail_post_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/mail_post"
)

type stubMailer struct {
	sent []cpmail.Mail
	err  error
}

func (s *stubMailer) Send(_ context.Context, mail cpmail.Mail) (cpmail.MessageID, error) {
	s.sent = append(s.sent, mail)
	return "sent@clickplanet.lol", s.err
}

var (
	from  = cpmail.Named{Address: "support@clickplanet.lol", Name: "ClickPlanet"}
	inbox = notes.NewInbox("support@clickplanet.lol")
)

func TestAnAnswerGoesFromTheInboxWithTheNotesOwnAddressToReplyTo(t *testing.T) {
	mailer := &stubMailer{}

	id, err := mail_post.New(mailer, from, inbox).Send(t.Context(), notes.Letter{
		Note: notes.NoteID{15: 1}, To: "ada@example.com", Subject: "Re: your idea", Text: "Thanks!",
	})

	require.NoError(t, err)
	assert.Equal(t, cpmail.MessageID("sent@clickplanet.lol"), id)
	assert.Equal(t, []cpmail.Mail{{
		From: from, To: "ada@example.com", ReplyTo: inbox.Of(notes.NoteID{15: 1}), Subject: "Re: your idea", Text: "Thanks!",
	}}, mailer.sent)
}

func TestALaterAnswerRepliesToTheLastMailAndReferencesThemAll(t *testing.T) {
	mailer := &stubMailer{}

	_, err := mail_post.New(mailer, from, inbox).Send(t.Context(), notes.Letter{
		Note: notes.NoteID{15: 1}, To: "ada@example.com", Subject: "Re: your idea", Text: "Done.",
		Thread: []cpmail.MessageID{"a1@clickplanet.lol", "r1@mail.example.com"},
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"In-Reply-To": "<r1@mail.example.com>",
		"References":  "<a1@clickplanet.lol> <r1@mail.example.com>",
	}, mailer.sent[0].Headers)
}

func TestAMailThatCouldNotBeSentIsNotSent(t *testing.T) {
	_, err := mail_post.New(&stubMailer{err: assert.AnError}, from, inbox).Send(t.Context(), notes.Letter{To: "ada@example.com"})

	require.ErrorIs(t, err, notes.ErrNotSent)
	require.ErrorIs(t, err, assert.AnError)
}
