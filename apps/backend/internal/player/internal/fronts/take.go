package fronts

import (
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Country string

var ErrNoCountry = errors.New("the take has no country")

type Take struct {
	account  players.AccountID
	country  Country
	previous Country
}

func NewTake(account players.AccountID, country, previous Country) (Take, error) {
	if country == "" {
		return Take{}, ErrNoCountry
	}
	return Take{account: account, country: country, previous: previous}, nil
}

func (t Take) Account() players.AccountID { return t.account }

func (t Take) Country() Country { return t.country }

func (t Take) Against() (Country, bool) {
	return t.previous, t.previous != "" && t.previous != t.country
}
