package refused_subscriber

import (
	"context"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, grant string) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

// Subscriber refunds a payment the flags module refused.
type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*flagsv1.TimeRefused] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *flagsv1.TimeRefused) error {
	return s.useCase.Execute(ctx, event.GetGrantId()) //nolint:wrapcheck // the use case named it.
}
