package outbox_planet_teller

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/warnings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var _ warnings.Teller = Teller{}

func New(outbox cpatomic.Outbox) Teller {
	return Teller{outbox: outbox}
}

type Teller struct {
	outbox cpatomic.Outbox
}

func (t Teller) Tell(ctx context.Context, warning warnings.Warning) error {
	return t.outbox.Record(ctx, &planetv1.WarnCaller{ //nolint:wrapcheck // the book names the warning.
		AccountId: warning.Account,
		DecidedAt: timestamppb.New(warning.At),
	})
}
