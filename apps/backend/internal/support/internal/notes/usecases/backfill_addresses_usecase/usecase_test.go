package backfill_addresses_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_note_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/backfill_addresses_usecase"
)

var (
	at    = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	ada   = notes.AccountID{15: 1}
	guest = notes.AccountID{15: 2}
	quiet = notes.AccountID{15: 3}
)

type book map[notes.AccountID]cpmail.Address

func (b book) Addresses(_ context.Context, accounts []notes.AccountID) (map[notes.AccountID]cpmail.Address, error) {
	held := map[notes.AccountID]cpmail.Address{}
	for _, account := range accounts {
		if address, found := b[account]; found {
			held[account] = address
		}
	}
	return held, nil
}

type signedIn []notes.AccountID

func (s signedIn) Linked(context.Context, []notes.AccountID) (*cpcolls.Set[notes.AccountID], error) {
	return cpcolls.NewSet(s...), nil
}

func note(n byte, account notes.AccountID) notes.Snapshot {
	return notes.Snapshot{ID: notes.NoteID{15: n}, Kind: notes.KindSuggestion, Account: account, Text: "an idea", At: at.Add(time.Duration(n) * time.Minute)}
}

func store() *inmemory_note_storage.Storage {
	kept := inmemory_note_storage.New()
	kept.Seed(note(1, ada), note(2, guest), note(3, quiet), note(4, ada))
	return kept
}

func backfill(kept *inmemory_note_storage.Storage) *backfill_addresses_usecase.UseCase {
	return backfill_addresses_usecase.New(kept, book{ada: "ada@example.com"}, signedIn{ada, quiet})
}

func TestADryRunCountsAndKeepsNothing(t *testing.T) {
	kept := store()

	counts, err := backfill(kept).Execute(t.Context(), true)

	require.NoError(t, err)
	assert.Equal(t, notes.BackfillCounts{Filled: 2, Guests: 1, NoAddress: 1}, counts)
	for _, snapshot := range kept.Kept() {
		assert.Empty(t, snapshot.ReplyAddress)
	}
}

func TestARunKeepsTheAddressesAndASecondOneFillsNothingNew(t *testing.T) {
	kept := store()

	_, err := backfill(kept).Execute(t.Context(), false)
	require.NoError(t, err)
	again, err := backfill(kept).Execute(t.Context(), false)
	require.NoError(t, err)

	addressed := map[notes.NoteID]cpmail.Address{}
	for _, snapshot := range kept.Kept() {
		addressed[snapshot.ID] = snapshot.ReplyAddress
	}
	assert.Equal(t, map[notes.NoteID]cpmail.Address{
		{15: 1}: "ada@example.com", {15: 2}: "", {15: 3}: "", {15: 4}: "ada@example.com",
	}, addressed)
	assert.Equal(t, notes.BackfillCounts{Guests: 1, NoAddress: 1}, again)
}
