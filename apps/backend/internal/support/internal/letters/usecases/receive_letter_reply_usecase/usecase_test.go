package receive_letter_reply_usecase_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters/usecases/receive_letter_reply_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var (
	now   = time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	id    = letters.LetterID{15: 1}
	inbox = notes.NewInbox("support@clickplanet.lol")
)

type stubReader struct {
	read int
}

func (s *stubReader) Read(io.Reader) (conversations.Received, error) {
	s.read++
	return conversations.Received{
		Mail: "r1@mail.example.com", From: "ada@example.com",
		Text: "Barcelona.\n\nOn Mon, ClickPlanet <support+l-x@clickplanet.lol> wrote:\n> Which city?",
	}, nil
}

func TestAReplyToALetterIsKeptWithoutWhatItQuotes(t *testing.T) {
	store, reader := letters.NewMemoryStorage(), &stubReader{}
	store.Seed(letters.TestWritten(id, notes.AccountID{15: 1}, now.Add(-time.Hour)))

	_, err := receive_letter_reply_usecase.New(inbox, reader, store, cptime.NewFixedClock(now)).Execute(t.Context(),
		receive_letter_reply_usecase.In{Recipient: string(letters.AddressOf(inbox, id)), Raw: strings.NewReader("")})
	require.NoError(t, err)

	assert.Equal(t, []conversations.Reply{{Mail: "r1@mail.example.com", From: "ada@example.com", Text: "Barcelona.", At: now}},
		store.Kept()[0].Replies)
}

func TestANotesAddressIsNoLettersAndTheMailIsNotRead(t *testing.T) {
	reader := &stubReader{}

	_, err := receive_letter_reply_usecase.New(inbox, reader, letters.NewMemoryStorage(), cptime.NewFixedClock(now)).
		Execute(t.Context(), receive_letter_reply_usecase.In{
			Recipient: string(inbox.Of(notes.NoteID{15: 1})), Raw: strings.NewReader(""),
		})

	require.ErrorIs(t, err, letters.ErrNotALetter)
	assert.Zero(t, reader.read, "the notes read it")
}

func TestAReplyToAnAnonymizedOrUnknownLetterIsStray(t *testing.T) {
	store := letters.NewMemoryStorage()
	anonymized := letters.TestWritten(id, notes.AccountID{}, now.Add(-time.Hour))
	anonymized.To = ""
	store.Seed(anonymized)
	useCase := receive_letter_reply_usecase.New(inbox, &stubReader{}, store, cptime.NewFixedClock(now))

	_, err := useCase.Execute(t.Context(), receive_letter_reply_usecase.In{
		Recipient: string(letters.AddressOf(inbox, id)), Raw: strings.NewReader(""),
	})
	assert.True(t, receive_letter_reply_usecase.IsStray(err), "%v", err)
	_, err = useCase.Execute(t.Context(), receive_letter_reply_usecase.In{
		Recipient: string(letters.AddressOf(inbox, letters.LetterID{15: 9})), Raw: strings.NewReader(""),
	})
	assert.True(t, receive_letter_reply_usecase.IsStray(err), "%v", err)
}
