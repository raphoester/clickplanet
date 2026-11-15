package claim_box_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/usecases/claim_box_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now   = time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	clock = cptime.NewFixedClock(now)
)

type stubRegistry struct {
	caught    boxes.Caught
	claimable bool

	token   string
	entrant boxes.Entrant
	scope   string
}

func (s *stubRegistry) Claim(token string, entrant boxes.Entrant, scope string) (boxes.Caught, bool) {
	s.token, s.entrant, s.scope = token, entrant, scope
	return s.caught, s.claimable
}

// goneOutbox keeps the context's error at each record, and fails when told to.
type goneOutbox struct {
	cpatomic.FakeOutbox
	gone    []error
	failing error
}

func (o *goneOutbox) Record(ctx context.Context, events ...proto.Message) error {
	o.gone = append(o.gone, ctx.Err())
	if o.failing != nil {
		return o.failing
	}
	return o.FakeOutbox.Record(ctx, events...) //nolint:wrapcheck // the fake's own.
}

var guest = boxes.Caller{Scope: "1.2.3.4", Account: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"}

func claiming(registry *stubRegistry) (*claim_box_usecase.UseCase, *goneOutbox) {
	outbox := &goneOutbox{}
	return claim_box_usecase.New(registry, cpatomic.FakeTransactioner{}, outbox, clock), outbox
}

func TestABoxCaughtIsToldWithTheAccountItsTokenAndWhetherItWasAWelcome(t *testing.T) {
	useCase, outbox := claiming(&stubRegistry{claimable: true, caught: boxes.Caught{Welcome: true}})

	require.NoError(t, useCase.Execute(t.Context(), claim_box_usecase.In{Caller: guest, Token: "a-token"}))

	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&boxesv1.BoxClaimed{
		AccountId: string(guest.Account), ClaimedAt: timestamppb.New(now), Token: "a-token", Welcome: true,
	}, outbox.Recorded()[0]))
}

func TestAGuestsClaimIsTheScopesAndALinkedAccountsTheAccounts(t *testing.T) {
	registry := &stubRegistry{claimable: true}
	useCase, _ := claiming(registry)

	require.NoError(t, useCase.Execute(t.Context(), claim_box_usecase.In{Caller: guest, Token: "a-token"}))
	assert.Equal(t, boxes.Entrant("1.2.3.4"), registry.entrant, "a guest's offer is the scope's")

	player := boxes.Caller{Scope: "1.2.3.4", Account: "a-player", Linked: true}
	require.NoError(t, useCase.Execute(t.Context(), claim_box_usecase.In{Caller: player, Token: "b-token"}))
	assert.Equal(t, boxes.Entrant("account:a-player"), registry.entrant)
	assert.Equal(t, "1.2.3.4", registry.scope)
	assert.Equal(t, "b-token", registry.token)
}

func TestARefusedClaimTellsNothing(t *testing.T) {
	useCase, outbox := claiming(&stubRegistry{claimable: false})

	err := useCase.Execute(t.Context(), claim_box_usecase.In{Caller: guest, Token: "not-mine"})

	require.ErrorIs(t, err, claim_box_usecase.ErrNoSuchBox)
	assert.Empty(t, outbox.Recorded())
}

func TestACallerWithNoAccountIsToldNothing(t *testing.T) {
	useCase, outbox := claiming(&stubRegistry{claimable: true})

	require.NoError(t, useCase.Execute(t.Context(),
		claim_box_usecase.In{Caller: boxes.Caller{Scope: "1.2.3.4"}, Token: "a-token"}))

	assert.Empty(t, outbox.Recorded(), "nothing pays a caller with no account")
}

func TestABoxCaughtIsToldWhenTheCallerLeavesAfterIt(t *testing.T) {
	useCase, outbox := claiming(&stubRegistry{claimable: true})
	ctx, leave := context.WithCancel(t.Context())
	leave()

	require.NoError(t, useCase.Execute(ctx, claim_box_usecase.In{Caller: guest, Token: "a-token"}))

	assert.Equal(t, []error{nil}, outbox.gone, "the record runs on a context the caller's leaving does not end")
	assert.Len(t, outbox.Recorded(), 1)
}

func TestARecordThatFailsIsAnErrorAndNotARefusal(t *testing.T) {
	useCase, outbox := claiming(&stubRegistry{claimable: true})
	outbox.failing = errors.New("postgres is down")

	err := useCase.Execute(t.Context(), claim_box_usecase.In{Caller: guest, Token: "a-token"})

	require.Error(t, err)
	assert.NotErrorIs(t, err, claim_box_usecase.ErrNoSuchBox)
}
