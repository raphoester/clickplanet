package outbox_planet_enforcer_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans/outbox_planet_enforcer"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var at = time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC)

func recorded(t *testing.T, caller bans.Caller, until time.Time) []proto.Message {
	t.Helper()
	outbox := &cpatomic.FakeOutbox{}

	require.NoError(t, cpatomic.FakeTransactioner{}.InTx(t.Context(), func(ctx context.Context) error {
		return outbox_planet_enforcer.New(outbox).Enforce(ctx, caller, until, at)
	}))
	return outbox.Recorded()
}

func TestARunningBanIsACallerPlanetRefuses(t *testing.T) {
	got := recorded(t, bans.Caller{Scope: "2001:db8::/64"}, at.Add(time.Hour))

	require.Len(t, got, 1)
	assert.True(t, proto.Equal(&planetv1.RefuseCaller{
		Caller:    &planetv1.RefuseCaller_Scope{Scope: "2001:db8::/64"},
		Until:     timestamppb.New(at.Add(time.Hour)),
		DecidedAt: timestamppb.New(at),
	}, got[0]), "got %v", got[0])
}

func TestABanThatEndsNowIsACallerPlanetAdmits(t *testing.T) {
	got := recorded(t, bans.Caller{Account: "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"}, at)

	require.Len(t, got, 1)
	assert.True(t, proto.Equal(&planetv1.AdmitCaller{
		Caller:    &planetv1.AdmitCaller_AccountId{AccountId: "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"},
		DecidedAt: timestamppb.New(at),
	}, got[0]), "got %v", got[0])
}

func TestACommandIsNeverRecordedOutsideTheBansTransaction(t *testing.T) {
	err := outbox_planet_enforcer.New(&cpatomic.FakeOutbox{}).Enforce(t.Context(), bans.Caller{Scope: "1.2.3.4"}, at.Add(time.Hour), at)

	assert.ErrorIs(t, err, cpatomic.ErrNoTransaction)
}
