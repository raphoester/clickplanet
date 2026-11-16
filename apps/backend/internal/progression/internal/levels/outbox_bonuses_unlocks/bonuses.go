package outbox_bonuses_unlocks

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/planetwords"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var _ levels.Bonuses = Bonuses{}

func New(outbox cpatomic.Outbox) Bonuses {
	return Bonuses{outbox: outbox}
}

// Bonuses speaks the bonuses module's words: what a level unlocked is a bonuses.v1.KeepUnlocks command, recorded in the
// transaction that writes the level.
type Bonuses struct {
	outbox cpatomic.Outbox
}

func (b Bonuses) KeepUnlocks(ctx context.Context, account accounts.AccountID, kinds []levels.Kind, at time.Time) error {
	theirs := make([]planetv1.BonusKind, 0, len(kinds))
	for _, kind := range kinds {
		kind, err := planetwords.BonusKind(kind)
		if err != nil {
			return err //nolint:wrapcheck // the sentinel names the kind.
		}
		theirs = append(theirs, kind)
	}
	return b.outbox.Record(ctx, &bonusesv1.KeepUnlocks{ //nolint:wrapcheck // the book names the change.
		AccountId: account.String(), Kinds: theirs, DecidedAt: timestamppb.New(at),
	})
}
