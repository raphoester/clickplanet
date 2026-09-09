package domain

import "context"

type Storage interface {
	Append(ctx context.Context, record ChatRecord) error
	History(ctx context.Context) []ChatMessage
}

type CountryChecker interface {
	CheckCountry(country string) bool
}
