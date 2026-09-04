package memory_tile_storage

import "time"

// Config tunes the in-memory tile storage. Every field has a usable default so
// an empty Config yields a working (but non-durable) storage.
type Config struct {
	// SnapshotPath is the file the tile state is periodically persisted to.
	// Empty disables durability: the state only lives in memory.
	SnapshotPath string

	// SnapshotInterval is how often a modified state is flushed to
	// SnapshotPath. Defaults to 30s.
	SnapshotInterval time.Duration

	// SubscriberBuffer is the capacity of each subscriber's channel. Updates
	// for a subscriber whose buffer is full are dropped rather than blocking
	// the click path. Defaults to 1024.
	SubscriberBuffer int

	// PastUpdatesBuffer is how many recent updates are kept around to serve
	// PastUpdates. Defaults to 65536.
	PastUpdatesBuffer int

	// PastUpdatesRetention is how long a recent update is kept regardless of
	// the buffer still having room. Defaults to 24h, comfortably above any
	// realistic bookkeeper poll interval.
	PastUpdatesRetention time.Duration
}

const (
	defaultSnapshotInterval     = 30 * time.Second
	defaultSubscriberBuffer     = 1024
	defaultPastUpdatesBuffer    = 65536
	defaultPastUpdatesRetention = 24 * time.Hour
)

func (c Config) withDefaults() Config {
	if c.SnapshotInterval <= 0 {
		c.SnapshotInterval = defaultSnapshotInterval
	}
	if c.SubscriberBuffer <= 0 {
		c.SubscriberBuffer = defaultSubscriberBuffer
	}
	if c.PastUpdatesBuffer <= 0 {
		c.PastUpdatesBuffer = defaultPastUpdatesBuffer
	}
	if c.PastUpdatesRetention <= 0 {
		c.PastUpdatesRetention = defaultPastUpdatesRetention
	}
	return c
}
