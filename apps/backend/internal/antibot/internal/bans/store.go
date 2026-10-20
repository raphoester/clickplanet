package bans

import (
	"context"
	"time"
)

type Record struct {
	Key           string
	Flags         int
	Offences      int
	ExpiresAt     time.Time
	LastFlaggedAt time.Time
	Pardon        Pardon
}

type Pardon struct {
	At     time.Time
	Lifted Sentence
	Until  time.Time
}

type Store interface {
	Record(ctx context.Context, key string) (Record, bool, error)

	Running(ctx context.Context, now time.Time) (int, error)

	// Change runs under a lock on the key, and keeps what it answers only when it answers true.
	Change(ctx context.Context, key string, change func(record Record, found bool) (Record, bool)) error
}
