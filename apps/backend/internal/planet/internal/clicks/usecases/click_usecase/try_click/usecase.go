package try_click

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
)

type Owners interface {
	Owner(tile uint32) (string, bool)
}

type Positions interface {
	Position(tile uint32) (clicks.Vec3, bool)
}

type Tempo interface {
	Tempo() float64
}

type Facts interface {
	Tell(fact facts.Fact)
}

func New(implementation click_usecase.IUseCase, owners Owners, positions Positions, tempo Tempo, facts Facts) *UseCase {
	return &UseCase{implementation: implementation, owners: owners, positions: positions, tempo: tempo, facts: facts}
}

// Outside every gate, so a refused or throttled try is told too: a loop just above the refill rate shows in its 429s.
type UseCase struct {
	implementation click_usecase.IUseCase
	owners         Owners
	positions      Positions
	tempo          Tempo
	facts          Facts
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	payer := clicks.PayerOf(ctx)

	out, err := u.implementation.Execute(ctx, in)

	tried := &planetv1.ClickTried{
		Scope:     payer.Scope,
		AccountId: payer.Account,
		Linked:    payer.Linked,
		TileId:    in.TileID,
		Country:   in.CountryID,
		Outcome:   outcomeOf(out, err),
		Owner:     out.Owner,
		Tempo:     u.tempo.Tempo(),
	}
	if !payer.Created.IsZero() {
		tried.AccountCreatedAt = timestamppb.New(payer.Created)
	}
	if err != nil {
		tried.Owner, _ = u.owners.Owner(in.TileID)
	}
	if at, ok := u.positions.Position(in.TileID); ok {
		tried.Position = &planetv1.GlobePoint{X: at.X, Y: at.Y, Z: at.Z}
	}

	u.facts.Tell(func(at time.Time) proto.Message {
		tried.TriedAt = timestamppb.New(at)
		return tried
	})

	return out, err
}

func outcomeOf(out click_usecase.Out, err error) planetv1.ClickOutcome {
	var otherFlag allegiances.Refusal

	switch {
	case err == nil:
		return answered(out.Outcome)
	case errors.Is(err, clicks.ErrRefused):
		return planetv1.ClickOutcome_CLICK_OUTCOME_REFUSED
	case errors.Is(err, clicks.ErrThrottled):
		return planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED
	case errors.As(err, &otherFlag):
		return planetv1.ClickOutcome_CLICK_OUTCOME_OTHER_FLAG
	case errors.Is(err, clicks.ErrUnknownCountry), errors.Is(err, clicks.ErrTileOutOfRange),
		errors.Is(err, clicks.ErrBonusesTogether):
		return planetv1.ClickOutcome_CLICK_OUTCOME_INVALID
	default:
		return planetv1.ClickOutcome_CLICK_OUTCOME_FAILED
	}
}

func answered(outcome clicks.Outcome) planetv1.ClickOutcome {
	switch outcome {
	case clicks.Taken:
		return planetv1.ClickOutcome_CLICK_OUTCOME_TAKEN
	case clicks.Shielded:
		return planetv1.ClickOutcome_CLICK_OUTCOME_STRUCK
	case clicks.Unchanged:
		return planetv1.ClickOutcome_CLICK_OUTCOME_UNCHANGED
	default:
		return planetv1.ClickOutcome_CLICK_OUTCOME_FAILED
	}
}
