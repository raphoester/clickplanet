package players

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type AccountID = cpsession.AccountID

var ErrInvalidAccount = errors.New("not an account id")

func AccountIDOf(value string) (AccountID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return AccountID{}, fmt.Errorf("%w: %q", ErrInvalidAccount, value)
	}
	return AccountID(id), nil
}

type Account struct {
	linked    bool
	createdAt time.Time
}

func AccountOf(linked bool, createdAt time.Time) Account {
	return Account{linked: linked, createdAt: createdAt}
}

func (a Account) Linked() bool { return a.linked }

func (a Account) CreatedAt() time.Time { return a.createdAt }
