package send_message_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Appender interface {
	Append(ctx context.Context, record messages.Record) error
}

type Publisher interface {
	Publish(update feed.Update)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type Authors interface {
	Author(ctx context.Context, account messages.AccountID) (messages.Author, error)
}

const writeTimeout = 5 * time.Second

type In struct {
	Account   messages.AccountID
	AuthorID  string
	CountryID string
	Text      string
	UserAgent string
}

func New(
	appender Appender,
	publisher Publisher,
	countryChecker CountryChecker,
	authors Authors,
	clock cptime.Clock,
	config Config,
) *UseCase {
	return &UseCase{
		appender:       appender,
		publisher:      publisher,
		countryChecker: countryChecker,
		authors:        authors,
		clock:          clock,
		limits:         messages.NewLimits(config.MaxTextLength),
	}
}

type UseCase struct {
	appender       Appender
	publisher      Publisher
	countryChecker CountryChecker
	authors        Authors
	clock          cptime.Clock
	limits         messages.Limits
}

func (u *UseCase) Execute(ctx context.Context, in In) (messages.Message, error) {
	if in.Account == cpsession.NoAccount {
		return messages.Message{}, messages.ErrNoAccount
	}

	text, err := u.limits.Text(in.Text)
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to check the message: %w", err)
	}

	if !u.countryChecker.CheckCountry(in.CountryID) {
		return messages.Message{}, fmt.Errorf("%w: invalid country code %q", messages.ErrInvalidMessage, in.CountryID)
	}

	ip := cpctx.GetSourceIP(ctx)

	author, err := u.authors.Author(ctx, in.Account)
	if err != nil {
		return messages.Message{}, fmt.Errorf("%w: %w", messages.ErrAuthorUnavailable, err)
	}

	message := messages.Message{
		ID:        messages.MessageID(uuid.NewString()),
		SentAt:    u.clock.Now(),
		Account:   in.Account,
		CountryID: in.CountryID,
		Text:      text,
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	if err := u.appender.Append(ctx, messages.NewRecord(message, in.AuthorID, ip, in.UserAgent)); err != nil {
		return messages.Message{}, fmt.Errorf("failed to store chat message: %w", err)
	}

	named := messages.Named(message, map[messages.AccountID]messages.Author{in.Account: author})

	published := named
	u.publisher.Publish(feed.Update{Message: &published})

	return named, nil
}
