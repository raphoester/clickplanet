package books

import (
	"errors"
	"fmt"
	"time"
)

// GrantID is the id another module gave a grant: scoring only repeats it.
type GrantID string

func ParseGrantID(id string) (GrantID, error) {
	if err := checkID(id); err != nil {
		return "", fmt.Errorf("grant %q: %w", id, err)
	}
	return GrantID(id), nil
}

var ErrEmptyGrant = errors.New("a grant gives points to a country")

// Grant gives a country points in every book open at its time, beside its rounds' points.
type Grant struct {
	ID      GrantID
	Country Country
	Points  uint32
	At      time.Time
}

func NewGrant(id GrantID, country Country, points uint32, at time.Time) (Grant, error) {
	if country == "" || points == 0 {
		return Grant{}, fmt.Errorf("%w: grant %q: %w", ErrRefused, id, ErrEmptyGrant)
	}
	return Grant{ID: id, Country: country, Points: points, At: at.UTC()}, nil
}
