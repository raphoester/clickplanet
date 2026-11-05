package stats

import "errors"

var ErrNoAnalyticsAccess = errors.New("zernio gives this account no analytics (hasAnalyticsAccess is false): " +
	"turn on its Analytics add-on, then run again; nothing was written")

// The connected accounts, as far as the snapshot needs them.
type Accounts struct {
	// Whether the service's Analytics add-on is on: without it there is nothing to snapshot.
	AnalyticsAccess bool
	TikTok          []TikTokAccount
}

type TikTokAccount struct {
	ID       string
	Username string
	// What TikTok said the account was when it was connected: business, personal or unknown.
	Type string
	// Whether TikTok's commercial music library answered for it: it does only for an account connected through the
	// TikTok for Business app.
	CommercialMusic bool
	// Why not, when it did not.
	CommercialMusicRefusal string
}
