package seen_get_history

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, account messages.AccountID) (get_history_usecase.History, error)
}

type Reader interface {
	SeenUntil(ctx context.Context, account messages.AccountID) (time.Time, error)
}

func New(inner UseCase, reader Reader) *Decorator {
	return &Decorator{inner: inner, reader: reader}
}

type Decorator struct {
	inner  UseCase
	reader Reader
}

var _ UseCase = (*Decorator)(nil)

func (d *Decorator) Execute(ctx context.Context, account messages.AccountID) (get_history_usecase.History, error) {
	history, err := d.inner.Execute(ctx, account)
	if err != nil || account == messages.NoAccount {
		return history, err //nolint:wrapcheck // a decorator adds a field, not a sentence.
	}

	history.SeenUntil, err = d.reader.SeenUntil(ctx, account)
	if err != nil {
		return get_history_usecase.History{}, fmt.Errorf("failed to read what the caller has seen: %w", err)
	}
	return history, nil
}
