package prune_letters_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters/usecases/anonymize_letters_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/letters/usecases/prune_letters_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var now = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)

func TestALetterPastTheRetentionNamesNobody(t *testing.T) {
	store := letters.NewMemoryStorage()
	store.Seed(letters.TestWritten(letters.LetterID{15: 1}, notes.AccountID{15: 1}, now.Add(-91*24*time.Hour)),
		letters.TestWritten(letters.LetterID{15: 2}, notes.AccountID{15: 2}, now.Add(-89*24*time.Hour)))

	pruned, err := prune_letters_usecase.New(90*24*time.Hour, cptime.NewFixedClock(now), store).Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, int64(1), pruned)
	assert.Equal(t, notes.AccountID{}, store.Kept()[0].Account)
	assert.Equal(t, notes.AccountID{15: 2}, store.Kept()[1].Account)
}

func TestADeletedAccountsLettersNameItNoMore(t *testing.T) {
	store := letters.NewMemoryStorage()
	store.Seed(letters.TestWritten(letters.LetterID{15: 1}, notes.AccountID{15: 1}, now))

	require.NoError(t, anonymize_letters_usecase.New(store).Execute(t.Context(), notes.AccountID{15: 1}))

	assert.Equal(t, notes.AccountID{}, store.Kept()[0].Account)
	assert.Empty(t, store.Kept()[0].To)
}
