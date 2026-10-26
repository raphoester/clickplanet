package bonuses

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrInvalidGrant = errors.New("not a grant id")

type GrantID uuid.UUID

func GrantIDOf(value string) (GrantID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return GrantID{}, fmt.Errorf("%w: %q", ErrInvalidGrant, value)
	}
	return GrantID(id), nil
}

func (id GrantID) String() string { return uuid.UUID(id).String() }
