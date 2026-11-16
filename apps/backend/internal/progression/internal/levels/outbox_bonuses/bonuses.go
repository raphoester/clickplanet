package outbox_bonuses

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/google/uuid"
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

// Bonuses speaks the bonuses module's words: what a level unlocked is a bonuses.v1.KeepUnlocks command, and a level
// reached by playing a bonuses.v1.PayLevelUp, each recorded in the transaction that writes the level.
type Bonuses struct {
	outbox cpatomic.Outbox
}

func (b Bonuses) KeepUnlocks(ctx context.Context, account accounts.AccountID, kinds []levels.Kind, at time.Time) error {
	theirs, err := kindsOf(kinds)
	if err != nil {
		return err
	}
	return b.outbox.Record(ctx, &bonusesv1.KeepUnlocks{ //nolint:wrapcheck // the book names the change.
		AccountId: account.String(), Kinds: theirs, DecidedAt: timestamppb.New(at),
	})
}

func (b Bonuses) PayLevelUp(ctx context.Context, account accounts.AccountID, level levels.Level, unlocked []levels.Kind) error {
	theirs, err := kindsOf(unlocked)
	if err != nil {
		return err
	}
	return b.outbox.Record(ctx, &bonusesv1.PayLevelUp{ //nolint:wrapcheck // the book names the change.
		GrantId: grantOf(account, level).String(), AccountId: account.String(), Level: uint32(level), Unlocked: theirs,
	})
}

var levelUpGrants = uuid.MustParse("1e7e1a5c-0b5e-4c1e-9a7e-6c3e1d2b4f60")

// A level reached names one grant: told twice, or raised to again after a forget, it is paid once.
func grantOf(account accounts.AccountID, level levels.Level) uuid.UUID {
	name := binary.BigEndian.AppendUint32(append([]byte{}, account[:]...), uint32(level))
	return uuid.NewSHA1(levelUpGrants, name)
}

func kindsOf(kinds []levels.Kind) ([]planetv1.BonusKind, error) {
	theirs := make([]planetv1.BonusKind, 0, len(kinds))
	for _, kind := range kinds {
		kind, err := planetwords.BonusKind(kind)
		if err != nil {
			return nil, err //nolint:wrapcheck // the sentinel names the kind.
		}
		theirs = append(theirs, kind)
	}
	return theirs, nil
}
