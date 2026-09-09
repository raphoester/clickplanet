package memory_tile_storage

import "time"

type Config struct {
	SnapshotPath string

	SnapshotInterval time.Duration

	SubscriberBuffer int

	PastUpdatesBuffer int

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
