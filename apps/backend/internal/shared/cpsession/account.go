package cpsession

import (
	"time"

	"github.com/google/uuid"
)

type AccountID uuid.UUID

var NoAccount = AccountID{}

func (id AccountID) String() string {
	return uuid.UUID(id).String()
}

func (id AccountID) CreatedAt() (time.Time, bool) {
	u := uuid.UUID(id)
	if u.Version() != 7 || u.Variant() != uuid.RFC4122 {
		return time.Time{}, false
	}

	var ms int64
	for _, b := range u[:6] {
		ms = ms<<8 | int64(b)
	}

	return time.UnixMilli(ms).UTC(), true
}
