package letters_query_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters/postgres_letter_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_letters_handler/letters_query"
)

var (
	at  = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	ada = notes.AccountID{15: 1}
	bob = notes.AccountID{15: 2}
)

func TestTheLettersAreReadTheLatestFirstForOneAccountOneSubmissionOrTheOnesThePlayerRepliedTo(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "support", migrations.FS)
	store := postgres_letter_store.New(db)
	replied := letters.TestWritten(letters.LetterID{15: 1}, ada, at)
	replied.Replies = []conversations.Reply{{
		Mail: "r1@mail.example.com", From: "ada@example.com", Text: "Barcelona.", At: at.Add(2 * time.Hour),
	}}
	quiet := letters.TestWritten(letters.LetterID{15: 2}, ada, at.Add(time.Hour))
	closed := letters.TestWritten(letters.LetterID{15: 3}, bob, at)
	closed.Replies = []conversations.Reply{{Mail: "r2@mail.example.com", From: "bob@example.com", Text: "No.", At: at.Add(time.Hour)}}
	closed.ClosedAt = at.Add(90 * time.Minute)
	for _, row := range []letters.Snapshot{replied, quiet, closed} {
		require.NoError(t, store.Save(t.Context(), letters.Restored(row)))
	}
	query := letters_query.NewPostgresQuery(db)

	all, err := query.Letters(t.Context(), letters_query.Filter{Limit: 100})
	require.NoError(t, err)
	ids := func() []string {
		found := make([]string, 0, len(all.GetLetters()))
		for _, letter := range all.GetLetters() {
			found = append(found, letter.GetId()[34:])
		}
		return found
	}
	assert.Equal(t, []string{"01", "02", "03"}, ids(), "the latest written or replied to first")
	assert.Equal(t, "Barcelona.", all.GetLetters()[0].GetReplies()[0].GetText())

	all, err = query.Letters(t.Context(), letters_query.Filter{Account: &bob, Limit: 100})
	require.NoError(t, err)
	assert.Equal(t, []string{"03"}, ids())

	all, err = query.Letters(t.Context(), letters_query.Filter{Submission: &quiet.Submission, Limit: 100})
	require.NoError(t, err)
	assert.Equal(t, []string{"02"}, ids(), "a founder's other submission has its own letter")
	assert.Equal(t, quiet.Submission.String(), all.GetLetters()[0].GetSubmissionId())

	all, err = query.Letters(t.Context(), letters_query.Filter{OpenOnly: true, Limit: 100})
	require.NoError(t, err)
	assert.Equal(t, []string{"01"}, ids(), "a reply after the last mail, and no close after it")
}
