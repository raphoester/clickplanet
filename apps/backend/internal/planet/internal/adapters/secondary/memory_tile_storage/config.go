package memory_tile_storage

import "time"

type Config struct {
	FlushInterval time.Duration

	LegacySnapshotPath string

	SubscriberBuffer int
}

const (
	defaultFlushInterval    = time.Second
	defaultSubscriberBuffer = 1024
)

func (c Config) withDefaults() Config {
	if c.FlushInterval <= 0 {
		c.FlushInterval = defaultFlushInterval
	}
	if c.SubscriberBuffer <= 0 {
		c.SubscriberBuffer = defaultSubscriberBuffer
	}
	return c
}
