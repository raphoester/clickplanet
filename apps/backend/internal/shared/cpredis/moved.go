package cpredis

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

// SubscribeMoved is Subscribe for an event that moved to another proto package with its fields, so another stream
// (events:<from> before, events:<T> now): the group first handles what its namesake left of the old stream, read as T,
// and only then reads the new one. Nothing a module told before the move is lost, and a receiver's marks see the old
// facts before the new ones. A group that never read the old stream, or an old stream deleted, starts on the new one.
func SubscribeMoved[T proto.Message](
	ctx context.Context, streams *Streams, from protoreflect.FullName, subscriber string, handler Handler[T],
) (cpbootstrap.Runner, error) {
	handled := delivery[T]{one: handler.Handle}
	next, err := subscribe(ctx, streams, streamOf, subscriber, handled)
	if err != nil {
		return nil, err
	}

	old := streamOf(from)
	groups, err := streams.client.XInfoGroups(ctx, old).Result()
	if err != nil {
		return next, nil //nolint:nilerr // no old stream is nothing left on it.
	}
	read := false
	for _, group := range groups {
		read = read || group.Name == subscriber
	}
	if !read {
		return next, nil
	}

	if err := streams.claim(from, subscriber); err != nil {
		return nil, err
	}
	left, err := newGroup(streams, old, from, subscriber)
	if err != nil {
		return nil, fmt.Errorf("failed to read what %s left of %s: %w", subscriber, old, err)
	}
	return &moved[T]{left: &subscription[T]{group: left, handler: handled}, next: next}, nil
}

type moved[T proto.Message] struct {
	left *subscription[T]
	next cpbootstrap.Runner
}

func (m *moved[T]) Name() string { return m.next.Name() }

// Run reads the new stream once the old one has nothing left; a shutdown before that leaves both where they were.
func (m *moved[T]) Run(ctx context.Context) {
	if m.left.exhaust(ctx) {
		m.next.Run(ctx)
	}
}

// exhaust handles what the group left pending, then every entry it has not read, each tried again in place as Run
// does: true once nothing is left.
func (s *subscription[T]) exhaust(ctx context.Context) bool {
	if !s.recover(ctx) {
		return false
	}
	for ctx.Err() == nil {
		messages, err := s.group.read(ctx, noBlock)
		if err != nil {
			pause(ctx, block)
			continue
		}
		if len(messages) == 0 {
			return true
		}
		if !s.deliver(ctx, messages, true) {
			return false
		}
	}
	return false
}
