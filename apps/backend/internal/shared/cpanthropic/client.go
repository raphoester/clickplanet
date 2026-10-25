// Package cpanthropic builds the client of Anthropic's API, and keeps its key out of what an error says.
package cpanthropic

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Config struct {
	APIKey string
	// Empty is Anthropic's API: a test's fake.
	BaseURL string
}

var ErrNoKey = errors.New("no ANTHROPIC_API_KEY")

// Timeouts and retries are the caller's, per request: one client serves calls of any length.
func NewClient(config Config) (*anthropic.Client, error) {
	if config.APIKey == "" {
		return nil, ErrNoKey
	}
	options := []option.RequestOption{option.WithAPIKey(config.APIKey)}
	if config.BaseURL != "" {
		options = append(options, option.WithBaseURL(config.BaseURL))
	}
	client := anthropic.NewClient(options...)
	return &client, nil
}

// What a failed call says with the request left out: the SDK's error dumps the request it failed on, headers and
// key included, and the logs of a public repo are public.
func Redacted(err error) error {
	var failed *anthropic.Error
	if errors.As(err, &failed) {
		return fmt.Errorf("HTTP %d: %s", failed.StatusCode, failed.Type())
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return errors.New("no answer")
}
