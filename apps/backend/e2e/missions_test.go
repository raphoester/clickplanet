package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
)

func (p *gamer) missions(country string) *playerv1.GetMissionsResponse {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.GetMissionsRequest{CountryId: country})
	p.send(req.Header())
	res, err := p.players().GetMissions(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg
}

func (p *gamer) collect(slot uint32) (*playerv1.CollectMissionResponse, error) {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.CollectMissionRequest{Slot: slot})
	p.send(req.Header())
	res, err := p.players().CollectMission(p.t.Context(), req)
	if err != nil {
		return nil, err //nolint:wrapcheck // the test reads the code.
	}
	return res.Msg, nil
}

func (p *gamer) charges() *planetv1.ChargesHeld {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GetChargesRequest{})
	p.send(req.Header())
	res, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).GetCharges(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetCharges()
}

func TestASignedInPlayerTakesItsBankForItsFlagAndCollectsShields(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("ada")

	before := ada.missions("fr")
	require.Len(t, before.GetMissions(), 3)
	assert.False(t, before.GetEnlisted())
	first := before.GetMissions()[0]
	require.Equal(t, playerv1.MissionKind_MISSION_KIND_TAKE_FOR, first.GetKind())
	require.Equal(t, "fr", first.GetTargetCountryId())

	for tile := uint32(1); tile <= first.GetGoal(); tile++ {
		ada.click(tile, "fr")
	}
	require.Eventually(t, func() bool {
		return ada.missions("de").GetMissions()[0].GetProgress() == first.GetGoal()
	}, 5*time.Second, 20*time.Millisecond, "the missions are the day's flag's, whatever is asked")
	assert.True(t, ada.missions("de").GetEnlisted())

	kept, err := ada.collect(1)
	require.NoError(t, err)
	assert.Equal(t, planetv1.BonusKind_BONUS_KIND_SHIELDS, kept.GetKind())
	assert.Equal(t, first.GetRewardAmount(), kept.GetAmount())
	assert.Equal(t, first.GetRewardAmount(), ada.charges().GetShields())
	assert.True(t, ada.missions("fr").GetMissions()[0].GetCollected())

	_, err = ada.collect(1)
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "a mission pays once")
}

func TestAGuestSeesItsProgressAndIsNotPaid(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)

	guest.click(1, "fr")
	require.Eventually(t, func() bool {
		return guest.missions("fr").GetMissions()[0].GetProgress() == 1
	}, 5*time.Second, 20*time.Millisecond)

	_, err := guest.collect(1)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}
