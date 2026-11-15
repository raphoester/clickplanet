package bonuses

import (
	"context"

	"github.com/google/uuid"
)

// The bonuses module grants charges and holds none: the wallet is planet's, reached through these ports, which the
// module answers with planet's internal calls (GetRoom, GrantCharges). A win draws a kind and an amount, and the wallet
// keeps what fits.

// Room is the wallet's word on what each account has room for: for each account asked, the kinds it holds as many of as
// it may, and a bomb while planet's bombs are off. The wallet applies its own sizes, which the bonuses module never
// sees.
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

// Payment is a charge a win grants an account, once per Grant.
type Payment struct {
	Grant   GrantID
	Account Account
	Kind    Kind
	Amount  int
}
