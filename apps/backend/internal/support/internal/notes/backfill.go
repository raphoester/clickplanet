package notes

import (
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

// What a backfill did, or would do, to the notes sent before addresses were kept, counted in notes.
type BackfillCounts struct {
	Filled int
	// A guest holds no address.
	Guests int
	// A signed-in account that holds no verified address.
	NoAddress int
}

// The distinct accounts that sent the notes, in their order.
func AccountsOf(sent []Note) []AccountID {
	var accounts []AccountID
	for _, note := range sent {
		if note.account != (AccountID{}) && !slices.Contains(accounts, note.account) {
			accounts = append(accounts, note.account)
		}
	}
	return accounts
}

// The notes that had no address, given the one their account holds now; a note that has one, or no sender, is left as
// it is.
func Backfilled(
	unaddressed []Note, held map[AccountID]cpmail.Address, linked *cpcolls.Set[AccountID],
) ([]Note, BackfillCounts) {
	var (
		filled []Note
		counts BackfillCounts
	)
	for _, note := range unaddressed {
		if note.account == (AccountID{}) || note.replyAddress != "" {
			continue
		}
		address, found := held[note.account]
		switch {
		case found:
			note.replyAddress = address
			filled = append(filled, note)
			counts.Filled++
		case linked.Contains(note.account):
			counts.NoAddress++
		default:
			counts.Guests++
		}
	}
	return filled, counts
}
