// Package send_message_usecase checks a message, names and stamps it, and appends it to the log.
package send_message_usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Appender interface {
	Append(ctx context.Context, record messages.Record) error
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

// Usernames is the player module, asked for the username an account chose. found is false for an account
// with none.
type Usernames interface {
	Username(ctx context.Context, account messages.AccountID) (username string, found bool, err error)
}

type In struct {
	// Account is the one the sender's click token names, or cpsession.NoAccount for a guest.
	Account    messages.AccountID
	AuthorName string
	AuthorID   string
	CountryID  string
	Text       string
	UserAgent  string
}

func New(
	appender Appender,
	countryChecker CountryChecker,
	usernames Usernames,
	clock cptime.Clock,
	config Config,
) *UseCase {
	return &UseCase{
		appender:       appender,
		countryChecker: countryChecker,
		usernames:      usernames,
		clock:          clock,
		limits:         messages.NewLimits(config.MaxTextLength, config.MaxNameLength),
		tagSalt:        config.TagSalt,
	}
}

type UseCase struct {
	appender       Appender
	countryChecker CountryChecker
	usernames      Usernames
	clock          cptime.Clock
	limits         messages.Limits
	tagSalt        string
}

func (u *UseCase) Execute(ctx context.Context, in In) (messages.Message, error) {
	name, err := u.authorName(ctx, in)
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

// authorName is the account's username, and the name the sender typed is then not read. Without one it is a
// guest's name. A username that could not be read is a guest's name too: a post never fails because the
// player module did not answer, and the decorator around the port says it did not.
func (u *UseCase) authorName(ctx context.Context, in In) (string, error) {
	if in.Account != cpsession.NoAccount {
		username, found, err := u.usernames.Username(ctx, in.Account)
		if err == nil && found {
			return username, nil
		}
	}

	return u.limits.GuestName(in.AuthorName) //nolint:wrapcheck // Execute says what failed.
}
