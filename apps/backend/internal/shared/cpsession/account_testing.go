//go:build testing

package cpsession

import (
	"time"

	"github.com/google/uuid"
)

func AccountCreatedAt(at time.Time) AccountID {
	var id uuid.UUID
	ms := at.UnixMilli()
	for i := 5; i >= 0; i-- {
		id[i] = byte(ms)
		ms >>= 8
	}
	id[6], id[8] = 0x70, 0x80

	return AccountID(id)
}
