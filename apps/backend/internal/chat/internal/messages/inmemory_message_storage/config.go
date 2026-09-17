package inmemory_message_storage

import "time"

type Config struct {
	HistorySize int

	Retention time.Duration

	PruneInterval time.Duration

	SubscriberBuffer int
}

const (
	defaultHistorySize      = 200
	defaultRetention        = 30 * 24 * time.Hour
	defaultPruneInterval    = time.Hour
	defaultSubscriberBuffer = 256
)

func (c Config) withDefaults() Config {
	if c.HistorySize <= 0 {
		c.HistorySize = defaultHistorySize
	}
	if c.Retention <= 0 {
		c.Retention = defaultRetention
	}
	if c.PruneInterval <= 0 {
		c.PruneInterval = defaultPruneInterval
	}
	if c.SubscriberBuffer <= 0 {
		c.SubscriberBuffer = defaultSubscriberBuffer
	}
	return c
}
