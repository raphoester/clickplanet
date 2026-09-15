// Package send_message_usecase checks a message, stamps it and appends it to the log.
package send_message_usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Appender interface {
	Append(ctx context.Context, record messages.Record) error
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	AuthorName string
	AuthorID   string
	CountryID  string
	Text       string
	UserAgent  string
}

func New(
	appender Appender,
	countryChecker CountryChecker,
	clock cptime.Clock,
	config Config,
) *UseCase {
	return &UseCase{
		appender:       appender,
		countryChecker: countryChecker,
		clock:          clock,
		limits:         messages.NewLimits(config.MaxTextLength, config.MaxNameLength),
		tagSalt:        config.TagSalt,
	}
}

type UseCase struct {
	appender       Appender
	countryChecker CountryChecker
	clock          cptime.Clock
	limits         messages.Limits
	tagSalt        string
}

func (u *UseCase) Execute(ctx context.Context, in In) (messages.Message, error) {
	name, err := u.limits.Name(in.AuthorName)
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to check the message: %w", err)
	}

	text, err := u.limits.Text(in.Text)
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to check the message: %w", err)
	}

	if !u.countryChecker.CheckCountry(in.CountryID) {
		return messages.Message{}, fmt.Errorf("%w: invalid country code %q", messages.ErrInvalidMessage, in.CountryID)
	}

	ip := cpctx.GetSourceIP(ctx)

	message := messages.Message{
		ID:         uuid.NewString(),
		SentAt:     u.clock.Now(),
		AuthorName: name,
		AuthorTag:  messages.Tag(u.tagSalt, ip),
		CountryID:  in.CountryID,
		Text:       text,
	}

	if err := u.appender.Append(ctx, messages.NewRecord(message, in.AuthorID, ip, in.UserAgent)); err != nil {
		return messages.Message{}, fmt.Errorf("failed to store chat message: %w", err)
	}

	return message, nil
}
