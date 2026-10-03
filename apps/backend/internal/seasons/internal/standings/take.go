package standings

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
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

type Country string

var (
	ErrNoCountry      = errors.New("the take has no country")
	ErrUnknownCountry = errors.New("not a country")
)

type Take struct {
	Account AccountID
	Country Country
	At      time.Time
}

func (t Take) Season(seasons calendar.Calendar) (calendar.Number, bool) {
	season, ok := seasons.Current(t.At)
	return season.Number, ok
}
