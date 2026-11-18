package bonuses

import (
	"github.com/google/uuid"
)

// The bonuses module grants charges and holds none: the wallet is planet's, reached through planet's internal call
// GrantCharges. A win draws a kind and an amount whatever the hand holds, and the wallet keeps what fits: it applies its
// own sizes and its own kinds turned off, which the bonuses module never sees.

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

// Stacks are the kinds a win fills up to their size, once per Grant.
type Stacks struct {
	Grant   GrantID
	Account Account
	Kinds   []Kind
}
