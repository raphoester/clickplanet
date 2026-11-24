package letter_post_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters/letter_post"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

type stubMailer struct {
	sent []cpmail.Mail
}

func (s *stubMailer) Send(_ context.Context, mail cpmail.Mail) (cpmail.MessageID, error) {
	s.sent = append(s.sent, mail)
	return "sent@clickplanet.lol", nil
}

func TestALetterGoesFromTheInboxWithItsOwnAddressToReplyTo(t *testing.T) {
	mailer, inbox := &stubMailer{}, notes.NewInbox("support@clickplanet.lol")
	from := cpmail.Named{Address: "support@clickplanet.lol", Name: "ClickPlanet"}

	_, err := letter_post.New(mailer, from, inbox).Send(t.Context(), letters.Mail{
		Letter: letters.LetterID{15: 1}, To: "ada@example.com", Subject: "Re: Your flag", Text: "Thanks!",
		Thread: []cpmail.MessageID{"l1@clickplanet.lol"},
	})
	require.NoError(t, err)

	assert.Equal(t, []cpmail.Mail{{
		From: from, To: "ada@example.com", ReplyTo: letters.AddressOf(inbox, letters.LetterID{15: 1}),
		Subject: "Re: Your flag", Text: "Thanks!",
		Headers: map[string]string{"In-Reply-To": "<l1@clickplanet.lol>", "References": "<l1@clickplanet.lol>"},
	}}, mailer.sent)
}
