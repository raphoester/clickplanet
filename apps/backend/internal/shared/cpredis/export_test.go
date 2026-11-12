package cpredis

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

// Group lets the trim test stay external, where it shares the package's Redis: an internal test cannot import cpredistest.
type Group struct {
	group *group
}

func GroupOf(runner cpbootstrap.Runner) (Group, bool) {
	subscribed, ok := runner.(*subscription[*wrapperspb.StringValue])
	if !ok {
		return Group{}, false
	}
	return Group{group: subscribed.group}, true
}

func (g Group) Read(ctx context.Context) ([]string, error) {
	messages, err := g.group.read(ctx, noBlock)
	if err != nil {
		return nil, err
	}

	positions := make([]string, len(messages))
	for i, message := range messages {
		positions[i] = message.ID
	}
	return positions, nil
}

func (g Group) Ack(ctx context.Context, position string) {
	g.group.ack(ctx, position)
}

func (g Group) Trim(ctx context.Context) {
	g.group.trim(ctx)
}

func (g Group) Len(ctx context.Context) (int64, error) {
	length, err := g.group.client.XLen(ctx, g.group.stream).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to read the stream's length: %w", err)
	}
	return length, nil
}
