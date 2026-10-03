package subscriptions

import "context"

type Store interface {
	Subscription(ctx context.Context, account AccountID) (*Subscription, error)
	SubscriptionsTo(ctx context.Context, address Address) ([]*Subscription, error)
	Save(ctx context.Context, subscription *Subscription) error
	Delete(ctx context.Context, account AccountID) error
}
