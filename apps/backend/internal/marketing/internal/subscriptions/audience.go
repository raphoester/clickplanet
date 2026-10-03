package subscriptions

import "context"

type Audience interface {
	Join(ctx context.Context, address Address) error
	Invite(ctx context.Context, address Address) error
	Joined(ctx context.Context, address Address) (bool, error)
	Leave(ctx context.Context, address Address) error
	Forget(ctx context.Context, address Address) error
}
