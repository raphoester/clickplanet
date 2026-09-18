// Package send_message_usecase checks a message, names and stamps it, and appends it to the log.
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

// Authors is the player module, asked who posts: the username of the account, and the tag of the address.
type Authors interface {
	Author(ctx context.Context, account messages.AccountID, ip string) (messages.Author, error)
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
	authors Authors,
	clock cptime.Clock,
	config Config,
) *UseCase {
	return &UseCase{
		appender:       appender,
		countryChecker: countryChecker,
		authors:        authors,
		clock:          clock,
		limits:         messages.NewLimits(config.MaxTextLength, config.MaxNameLength),
	}
}

type UseCase struct {
	appender       Appender
	countryChecker CountryChecker
	authors        Authors
	clock          cptime.Clock
	limits         messages.Limits
}

func (u *UseCase) Execute(ctx context.Context, in In) (messages.Message, error) {
	text, err := u.limits.Text(in.Text)
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to check the message: %w", err)
	}

	if !u.countryChecker.CheckCountry(in.CountryID) {
		return messages.Message{}, fmt.Errorf("%w: invalid country code %q", messages.ErrInvalidMessage, in.CountryID)
	}

	ip := cpctx.GetSourceIP(ctx)

	author, err := u.authors.Author(ctx, in.Account, ip)
	if err != nil {
		return messages.Message{}, fmt.Errorf("%w: %w", messages.ErrAuthorUnavailable, err)
	}

	name, err := u.authorName(in, author)
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to check the message: %w", err)
	}

	message := messages.Message{
		ID:          messages.MessageID(uuid.NewString()),
		SentAt:      u.clock.Now(),
		AuthorName:  name,
		AuthorTag:   author.Tag,
		AuthorAdmin: author.PostsAsPlayer(in.Account) && author.Admin,
		CountryID:   in.CountryID,
		Text:        text,
	}

	if err := u.appender.Append(ctx, messages.NewRecord(message, in.AuthorID, ip, in.UserAgent)); err != nil {
		return messages.Message{}, fmt.Errorf("failed to store chat message: %w", err)
	}

	return message, nil
}

// authorName is the account's username, and the name the sender typed is then not read. Without one it is a
// guest's name.
func (u *UseCase) authorName(in In, author messages.Author) (string, error) {
	if author.PostsAsPlayer(in.Account) {
		return author.Username, nil
	}

	return u.limits.GuestName(in.AuthorName) //nolint:wrapcheck // Execute says what failed.
}
