// Package send_message_usecase checks a message, names and stamps it, and appends it to the log.
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

// Publisher is the live feed: a message goes out once it is kept.
type Publisher interface {
	Publish(update feed.Update)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

// Authors is the player module, asked who posts: the account's username, or its guest code.
type Authors interface {
	Author(ctx context.Context, account messages.AccountID) (messages.Author, error)
}

const writeTimeout = 5 * time.Second

type In struct {
	// Account is the one the sender's click token names, or cpsession.NoAccount, which is refused.
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

// Execute refuses a sender with no account before anything else: it has no name.
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

	// What is kept is the account, never a copy of the name: a reader is shown who that account is now, so a
	// rename shows on everything its player ever said and a deleted account stops being named at all.
	message := messages.Message{
		ID:        messages.MessageID(uuid.NewString()),
		SentAt:    u.clock.Now(),
		Account:   in.Account,
		CountryID: in.CountryID,
		Text:      text,
	}

	// The log is the audit trail: a message that cannot be kept is not sent.
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	if err := u.appender.Append(ctx, messages.NewRecord(message, in.AuthorID, ip, in.UserAgent)); err != nil {
		return messages.Message{}, fmt.Errorf("failed to store chat message: %w", err)
	}

	// What goes out carries the name, which this path already asked for: everyone watching the chat is shown
	// who is talking without a second read.
	named := messages.Named(message, map[messages.AccountID]messages.Author{in.Account: author})

	published := named
	u.publisher.Publish(feed.Update{Message: &published})

	return named, nil
}
