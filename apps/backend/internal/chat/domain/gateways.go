package domain

import "context"

// Storage serves recent messages and appends every one to the durable log.
type Storage interface {
	Append(ctx context.Context, record ChatRecord) error
	History(ctx context.Context) []ChatMessage
}

// CountryChecker validates the ISO code a message is flagged with. The tile
// game owns the country list and has its own port of the same shape; the
// composition root hands the one implementation to both.
type CountryChecker interface {
	CheckCountry(country string) bool
}
