package memory_chat_storage

import "time"

// Config tunes the chat storage. Every field has a usable default, and an empty
// LogPath yields a working but non-durable storage.
type Config struct {
	// LogPath is the append-only JSONL file every message is written to, sender
	// IP included. Empty keeps chat entirely in memory.
	LogPath string

	// HistorySize is how many recent messages a joining client is served.
	// Defaults to 200.
	HistorySize int

	// Retention is how long a message stays in the log. Older lines are dropped
	// on the next prune. Defaults to 30 days.
	Retention time.Duration

	// FlushInterval bounds what a hard kill can lose, the same way the tile
	// snapshot interval does. Defaults to 5s.
	FlushInterval time.Duration

	// PruneInterval is how often the log is rewritten to apply Retention.
	// Defaults to 1h.
	PruneInterval time.Duration

	// SubscriberBuffer is the capacity of each websocket subscriber's channel.
	// Defaults to 256.
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
