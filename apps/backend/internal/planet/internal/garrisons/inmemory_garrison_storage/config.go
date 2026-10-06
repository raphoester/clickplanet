package inmemory_garrison_storage

import "time"

type Config struct {
	FlushInterval time.Duration

	SubscriberBuffer int
}

const (
	defaultFlushInterval    = time.Second
	defaultSubscriberBuffer = 1024
	flushTimeout            = 10 * time.Second
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
