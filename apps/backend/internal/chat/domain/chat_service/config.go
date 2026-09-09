package chat_service

// Config tunes what a sender is allowed to say. Every field has a usable
// default, except TagSalt — see the composition root, which generates one when
// it is left empty.
type Config struct {
	// MaxTextLength bounds a message, in runes. Defaults to 280.
	MaxTextLength int

	// MaxNameLength bounds a display name, in runes. Defaults to 24.
	MaxNameLength int

	// TagSalt keeps the per-sender tag from being reversible into an IP by
	// anyone willing to hash the address space.
	TagSalt string
}

const (
	defaultMaxTextLength = 280
	defaultMaxNameLength = 24
)

func (c Config) withDefaults() Config {
	if c.MaxTextLength <= 0 {
		c.MaxTextLength = defaultMaxTextLength
	}
	if c.MaxNameLength <= 0 {
		c.MaxNameLength = defaultMaxNameLength
	}
	return c
}
