package cpatomic

import (
	"context"

	"google.golang.org/protobuf/proto"
)

type Transactioner interface {
	InTx(ctx context.Context, do func(ctx context.Context) error) error
}

type Outbox interface {
	Record(ctx context.Context, events ...proto.Message) error
}
