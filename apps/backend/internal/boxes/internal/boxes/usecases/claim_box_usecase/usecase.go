package claim_box_usecase

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// One error for every refusal: telling them apart helps a script guessing tokens.
var ErrNoSuchBox = errors.New("no box to claim")

const tellTimeout = 5 * time.Second

type Registry interface {
	Claim(token string, entrant boxes.Entrant, scope string) (boxes.Caught, bool)
}

type In struct {
	Caller boxes.Caller
	Token  string
}

func New(registry Registry, transactioner cpatomic.Transactioner, outbox cpatomic.Outbox, clock cptime.Clock) *UseCase {
	return &UseCase{registry: registry, transactioner: transactioner, outbox: outbox, clock: clock}
}

// UseCase takes a box offered to the caller and tells it caught. It pays nothing: the bonuses module draws what the box
// holds and pays it from BoxClaimed.
type UseCase struct {
	registry      Registry
	transactioner cpatomic.Transactioner
	outbox        cpatomic.Outbox
	clock         cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	caught, ok := u.registry.Claim(in.Token, boxes.EntrantOf(in.Caller), in.Caller.Scope)
	if !ok {
		return ErrNoSuchBox
	}

	return u.tell(ctx, in, caught)
}

// tell keeps BoxClaimed in the outbox, so a box caught reaches whoever pays it though Redis be away a moment. The box is
// taken: a caller who leaves now is still told. A caller with no account is paid nothing, so nothing is told.
func (u *UseCase) tell(ctx context.Context, in In, caught boxes.Caught) error {
	if in.Caller.Account == boxes.NoAccount {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), tellTimeout)
	defer cancel()

	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // the outbox names the event.
		return u.outbox.Record(ctx, &boxesv1.BoxClaimed{
			AccountId: string(in.Caller.Account),
			ClaimedAt: timestamppb.New(u.clock.Now()),
			Token:     in.Token,
			Welcome:   caught.Welcome,
		})
	})
}
