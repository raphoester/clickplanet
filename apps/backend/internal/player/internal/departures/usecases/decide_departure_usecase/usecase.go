package decide_departure_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Departures interface {
	Change(ctx context.Context, account players.AccountID, change func(departures.Departure) departures.Departure) error
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	departures    Departures
	outbox        cpatomic.Outbox
}

func New(transactioner cpatomic.Transactioner, departures Departures, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{transactioner: transactioner, departures: departures, outbox: outbox}
}

// Every decision is told, the ones the departure keeps no more too: a listener keeps the latest, as player does.
func (u *UseCase) Execute(ctx context.Context, account players.AccountID, decision departures.Decision) error {
	if err := decision.TimeError(); err != nil {
		return fmt.Errorf("failed to read the decision on %s: %w", account, err)
	}
	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each step names what failed.
		if err := u.departures.Change(ctx, account, func(departure departures.Departure) departures.Departure {
			return departure.Decided(decision)
		}); err != nil {
			return fmt.Errorf("failed to keep the decision on %s: %w", account, err)
		}
		if err := u.outbox.Record(ctx, toldOf(account, decision.TakeSnapshot())); err != nil {
			return fmt.Errorf("failed to record the decision on %s: %w", account, err)
		}
		return nil
	})
}

func toldOf(account players.AccountID, decision departures.DecisionSnapshot) proto.Message {
	if decision.Leaving {
		return &playerv1.PlayerLeaving{AccountId: account.String(), DecidedAt: timestamppb.New(decision.At)}
	}
	return &playerv1.PlayerReturned{AccountId: account.String(), DecidedAt: timestamppb.New(decision.At)}
}
