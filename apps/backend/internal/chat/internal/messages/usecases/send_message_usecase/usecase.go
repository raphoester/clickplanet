package send_message_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Appender interface {
	Append(ctx context.Context, record messages.Record) error
}

type Publisher interface {
	Publish(update feed.Update)
}

type Authors interface {
	Author(ctx context.Context, account messages.AccountID) (messages.Author, error)
}

type Drafts interface {
	Draft(account messages.AccountID, country string, text string) (messages.Message, error)
}

type Thread interface {
	Reply(ctx context.Context, draft messages.Message, replyTo messages.MessageID) (messages.Message, error)
}

const writeTimeout = 5 * time.Second

type In struct {
	Account   messages.AccountID
	AuthorID  string
	CountryID string
	Text      string
	ReplyTo   messages.MessageID
	UserAgent string
}

func New(
	appender Appender,
	publisher Publisher,
	authors Authors,
	drafts Drafts,
	thread Thread,
) *UseCase {
	return &UseCase{
		appender:  appender,
		publisher: publisher,
		authors:   authors,
		drafts:    drafts,
		thread:    thread,
	}
}

type UseCase struct {
	appender  Appender
	publisher Publisher
	authors   Authors
	drafts    Drafts
	thread    Thread
}

func (u *UseCase) Execute(ctx context.Context, in In) (messages.Message, error) {
	if in.Account == cpsession.NoAccount {
		return messages.Message{}, messages.ErrNoAccount
	}

	draft, err := u.drafts.Draft(in.Account, in.CountryID, in.Text)
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to draft the message: %w", err)
	}

	message, err := u.thread.Reply(ctx, draft, in.ReplyTo)
	if err != nil {
		return messages.Message{}, fmt.Errorf("failed to reply: %w", err)
	}

	ip := cpctx.GetSourceIP(ctx)

	author, err := u.authors.Author(ctx, in.Account)
	if err != nil {
		return messages.Message{}, fmt.Errorf("%w: %w", messages.ErrAuthorUnavailable, err)
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	if err := u.appender.Append(ctx, messages.NewRecord(message, in.AuthorID, ip, in.UserAgent)); err != nil {
		return messages.Message{}, fmt.Errorf("failed to store chat message: %w", err)
	}

	named := message.Named(author)
	u.publisher.Publish(feed.MessageSent(named))

	return named, nil
}
