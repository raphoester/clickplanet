package messages

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Quote struct {
	id      MessageID
	account AccountID
	author  Author
	text    string
}

func QuoteOf(id MessageID, account AccountID, text string) Quote {
	return Quote{id: id, account: account, text: text}
}

func (q Quote) ID() MessageID { return q.id }

func (q Quote) Account() AccountID { return q.account }

func (q Quote) Author() Author { return q.author }

func (q Quote) Text() string { return q.text }

func (q Quote) Empty() bool { return q.id == "" }

func (q Quote) Named(author Author) Quote {
	q.author = author
	return q
}

const DeletedName = "[deleted]"

var ErrNotShown = errors.New("the message is not shown")

type Quotable interface {
	Quote(ctx context.Context, id MessageID, since time.Time, limit int) (Quote, error)
}

type Namer interface {
	Authors(ctx context.Context, accounts []AccountID) (map[AccountID]Author, error)
}

const quoteTimeout = 5 * time.Second

func NewQuotes(shown Quotable, namer Namer, clock cptime.Clock, window Window) Quotes {
	return Quotes{shown: shown, namer: namer, clock: clock, window: window}
}

type Quotes struct {
	shown  Quotable
	namer  Namer
	clock  cptime.Clock
	window Window
}

func (q Quotes) Quote(ctx context.Context, replyTo MessageID) (Quote, error) {
	if replyTo == "" {
		return Quote{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, quoteTimeout)
	defer cancel()

	quote, err := q.shown.Quote(ctx, replyTo, q.window.Since(q.clock.Now()), q.window.Size())
	if err != nil {
		return Quote{}, fmt.Errorf("failed to read the message replied to: %w", err)
	}
	if quote.Account() == NoAccount {
		return quote, nil
	}

	named, err := q.namer.Authors(ctx, []AccountID{quote.Account()})
	if err != nil {
		return Quote{}, fmt.Errorf("%w: %w", ErrAuthorUnavailable, err)
	}
	author, known := named[quote.Account()]
	if !known {
		return quote.Named(AuthorOf(DeletedName, false, 0, 0, Title{})), nil
	}
	return quote.Named(author), nil
}
