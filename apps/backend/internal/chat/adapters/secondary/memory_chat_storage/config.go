package memory_chat_storage

import "time"

type Config struct {
	LogPath string

	HistorySize int

	Retention time.Duration

	FlushInterval time.Duration

	PruneInterval time.Duration

	SubscriberBuffer int
}

const (
	defaultHistorySize      = 200
	defaultRetention        = 30 * 24 * time.Hour
	defaultFlushInterval    = 5 * time.Second
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
	if c.FlushInterval <= 0 {
		c.FlushInterval = defaultFlushInterval
	}
	if c.PruneInterval <= 0 {
		c.PruneInterval = defaultPruneInterval
	}
	if c.SubscriberBuffer <= 0 {
		c.SubscriberBuffer = defaultSubscriberBuffer
	}
	return c
}
