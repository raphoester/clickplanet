package account_deleted_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_attachment_bucket"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_note_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/anonymize_notes_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/subscribers/account_deleted_subscriber"
)

var at = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

func TestADeletedAccountsNotesKeepTheirTextAndLoseTheirSender(t *testing.T) {
	store := inmemory_note_storage.New()
	ada, bob := notes.AccountID{15: 1}, notes.AccountID{15: 2}
	kept := notes.Snapshot{ID: notes.NoteID{15: 2}, Kind: notes.KindSuggestion, Account: bob, Text: "more flags", At: at.Add(time.Minute)}
	store.Seed(notes.Snapshot{ID: notes.NoteID{15: 1}, Kind: notes.KindSuggestion, Account: ada, Text: "a night mode", At: at}, kept)

	err := account_deleted_subscriber.New(anonymize_notes_usecase.New(store, inmemory_attachment_bucket.New())).Handle(t.Context(),
		&authv1.AccountDeleted{AccountId: ada.String()})

	require.NoError(t, err)
	assert.Equal(t, []notes.Snapshot{
		{ID: notes.NoteID{15: 1}, Kind: notes.KindSuggestion, Text: "a night mode", At: at},
		kept,
	}, store.Kept())
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := account_deleted_subscriber.New(anonymize_notes_usecase.New(inmemory_note_storage.New(), inmemory_attachment_bucket.New())).Handle(t.Context(),
		&authv1.AccountDeleted{})

	require.ErrorIs(t, err, notes.ErrInvalidAccount)
}
