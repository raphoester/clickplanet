package memory_tile_storage

import "time"

type Config struct {
	SnapshotPath string

	SnapshotInterval time.Duration

	SubscriberBuffer int
}

const (
	defaultSnapshotInterval = 30 * time.Second
	defaultSubscriberBuffer = 1024
)

func (c Config) withDefaults() Config {
	if c.SnapshotInterval <= 0 {
		c.SnapshotInterval = defaultSnapshotInterval
	}
	if c.SubscriberBuffer <= 0 {
		c.SubscriberBuffer = defaultSubscriberBuffer
	}
	return c
}
