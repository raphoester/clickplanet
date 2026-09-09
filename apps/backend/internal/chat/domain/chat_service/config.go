package chat_service

type Config struct {
	MaxTextLength int

	MaxNameLength int

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
