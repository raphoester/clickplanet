package books

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type AccountID = cpsession.AccountID

// Take is one tile taken, as a round counts it for its numbers: who took it, for which flag, when, and its place in
// planet's ledger.
type Take struct {
	Account  AccountID
	Country  Country
	At       time.Time
	Position uint64
	Index    uint32
}

// TakeMark is the ledger position and index past the last take a round counted: a take below it was counted already.
type TakeMark struct {
	Position uint64
	Index    uint32
}

// Uncounted is the takes, in ledger order, that are past the mark and past each one before them, and the mark past the
// last of them: a take told twice, in one batch or in two, is counted once.
func (m TakeMark) Uncounted(takes []Take) ([]Take, TakeMark) {
	var uncounted []Take
	for _, take := range takes {
		if take.Position < m.Position || take.Position == m.Position && take.Index < m.Index {
			continue
		}
		uncounted = append(uncounted, take)
		m = TakeMark{Position: take.Position, Index: take.Index + 1}
	}
	return uncounted, m
}
