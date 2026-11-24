package notes_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func kept(t *testing.T, n byte, account notes.AccountID, address cpmail.Address) notes.Note {
	t.Helper()

	note, err := notes.Snapshot{
		ID: notes.NoteID{15: n}, Kind: notes.KindBug, Account: account, Text: "the globe is black", ReplyAddress: address, At: sentAt,
	}.Restore()
	require.NoError(t, err)
	return note
}

func TestEachNoteWithNoAddressTakesItsAccountsAndTheOthersAreCountedByWhyNot(t *testing.T) {
	bob, cara := notes.AccountID{15: 2}, notes.AccountID{15: 3}
	unaddressed := []notes.Note{
		kept(t, 1, ada, ""), kept(t, 2, ada, ""), kept(t, 3, bob, ""), kept(t, 4, cara, ""),
		kept(t, 5, ada, "ada+old@example.com"), kept(t, 6, notes.AccountID{}, ""),
	}

	filled, counts := notes.Backfilled(unaddressed, map[notes.AccountID]cpmail.Address{ada: "ada@example.com"}, cpcolls.NewSet(ada, cara))

	assert.Equal(t, notes.BackfillCounts{Filled: 2, Guests: 1, NoAddress: 1}, counts)
	require.Len(t, filled, 2)
	for n, note := range filled {
		assert.Equal(t, notes.Snapshot{
			ID: notes.NoteID{15: byte(n + 1)}, Kind: notes.KindBug, Account: ada, Text: "the globe is black",
			ReplyAddress: "ada@example.com", At: sentAt,
		}, note.TakeSnapshot())
	}
}

func TestTheSendersAreEachAccountOnceAndNoneForANoteWithNoSender(t *testing.T) {
	bob := notes.AccountID{15: 2}

	senders := notes.AccountsOf([]notes.Note{kept(t, 1, ada, ""), kept(t, 2, bob, ""), kept(t, 3, ada, ""), kept(t, 4, notes.AccountID{}, "")})

	assert.Equal(t, []notes.AccountID{ada, bob}, senders)
}
