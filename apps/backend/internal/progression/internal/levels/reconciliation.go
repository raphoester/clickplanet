package levels

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

type ByAccount map[accounts.AccountID]Level

type Reconciliation struct {
	raised  ByAccount
	removed []accounts.AccountID
	reached ByAccount
}

func (r Reconciliation) Raised() ByAccount { return r.raised }

func (r Reconciliation) Removed() []accounts.AccountID { return r.removed }

// Reached is every linked account of the page at the level it holds after the reconciliation, changed or not: the
// level its stats reach, or the one kept when it is higher.
func (r Reconciliation) Reached() ByAccount { return r.reached }

// ReconciliationOf raises every linked account of the page to the level its stats reach, and never lowers one: a level
// the players saw stays. A guest's level is removed.
func (s Scale) ReconciliationOf(
	page []stats.Stats, known map[accounts.AccountID]accounts.Account, kept ByAccount,
) Reconciliation {
	reconciliation := Reconciliation{raised: ByAccount{}, reached: ByAccount{}}
	for _, counted := range page {
		account := counted.Account()
		standing := s.StandingOf(known[account], counted)
		if standing.Empty() {
			if _, held := kept[account]; held {
				reconciliation.removed = append(reconciliation.removed, account)
			}
			continue
		}
		reconciliation.reached[account] = max(standing.Level(), kept[account])
		if standing.Level() > kept[account] {
			reconciliation.raised[account] = standing.Level()
		}
	}
	return reconciliation
}
