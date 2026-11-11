package bonuses

import (
	"context"

	"github.com/google/uuid"
)

// The offers grant charges and hold none: the wallet is planet's, reached through these ports, which the module answers
// with planet's internal calls (GetRoom, GrantCharges, GetChargesOf). An offer draws a kind and an amount, and the
// wallet keeps what fits.

// Room is the wallet's word on what each account has room for: for each account asked, the kinds it holds as many of as
// it may, and a bomb while planet's bombs are off. The wallet applies its own sizes, which the offers never see. It is
// asked outside the registry's lock.
type Room interface {
	Full(ctx context.Context, accounts []Account) (map[Account][]Kind, error)
}

// GrantID names one payment: the wallet pays each once, so a payment told again grants nothing more.
type GrantID uuid.UUID

func (id GrantID) String() string { return uuid.UUID(id).String() }

// A box's or a quiz's payment is named by the token it was offered under, which is random and used once.
var offerGrants = uuid.MustParse("28a9e973-5c37-434e-a86f-e64e8b894348")

func OfferGrantOf(token string) GrantID {
	return GrantID(uuid.NewSHA1(offerGrants, []byte(token)))
}

// Payment is a charge an offer grants an account, once per Grant.
type Payment struct {
	Grant   GrantID
	Account Account
	Kind    Kind
	Amount  int
}

// Paid is how many of the payment's kind the wallet kept, which a full stack makes fewer than it was asked, and the
// hand after it. A payment told again keeps none and answers the hand as it is.
type Paid struct {
	Kept int
	Hand Hand
}

// Hand is what an account holds, as the wallet answers it: the offers only hand it back to the player.
type Hand struct {
	Refill bool
	Bomb   bool

	Enclosures int

	SpreadClicks int

	Shields int
}
