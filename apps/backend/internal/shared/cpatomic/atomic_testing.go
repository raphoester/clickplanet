//go:build testing

package cpatomic

import (
	"context"
	"errors"
	"slices"
	"sync"

	"google.golang.org/protobuf/proto"
)

type pendingKey struct{}

type pending struct {
	commits []func()
}

// FakeTransactioner runs do as it is: an in-memory store keeps what a failed do wrote, a FakeOutbox drops it.
type FakeTransactioner struct{}

func (FakeTransactioner) InTx(ctx context.Context, do func(ctx context.Context) error) error {
	if _, open := ctx.Value(pendingKey{}).(*pending); open {
		return do(ctx)
	}

	inTx := &pending{}
	if err := do(context.WithValue(ctx, pendingKey{}, inTx)); err != nil {
		return err
	}
	for _, commit := range inTx.commits {
		commit()
	}
	return nil
}

var ErrNoTransaction = errors.New("no transaction is open")

type FakeOutbox struct {
	mu        sync.Mutex
	committed []proto.Message
}

func (o *FakeOutbox) Record(ctx context.Context, events ...proto.Message) error {
	inTx, open := ctx.Value(pendingKey{}).(*pending)
	if !open {
		return ErrNoTransaction
	}
	inTx.commits = append(inTx.commits, func() {
		o.mu.Lock()
		defer o.mu.Unlock()
		o.committed = append(o.committed, events...)
	})
	return nil
}

func (o *FakeOutbox) Recorded() []proto.Message {
	o.mu.Lock()
	defer o.mu.Unlock()

	return slices.Clone(o.committed)
}
