package boxes

import "context"

// Room is the wallet's word on whether a box could pay the accounts playing behind a caller anything: true while their
// hands, together, have room for at least one kind. The wallet is planet's, and which kind a box holds is the bonuses
// module's to draw, so the boxes see no kind. It is asked outside the registry's lock.
type Room interface {
	Any(ctx context.Context, accounts []Account) (bool, error)
}
