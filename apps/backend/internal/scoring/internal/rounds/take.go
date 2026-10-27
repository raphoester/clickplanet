package rounds

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type AccountID = cpsession.AccountID

// Take is one tile taken, as a round of its own counts it for its numbers: who took it, for which flag, when, and its
// place in planet's ledger.
type Take struct {
	Account  AccountID
	Country  Country
	At       time.Time
	Position uint64
}
