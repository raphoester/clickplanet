package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

func (p *gamer) charges() *planetv1.ChargesHeld {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GetChargesRequest{})
	p.send(req.Header())
	res, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).GetCharges(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetCharges()
}

func (p *gamer) chargesFilled(ctx context.Context) <-chan struct{} {
	p.t.Helper()

	listen := connect.NewRequest(&planetv1.ListenForEventsRequest{})
	p.send(listen.Header())
	stream, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).ListenForEvents(ctx, listen)
	require.NoError(p.t, err)
	require.True(p.t, stream.Receive(), "the stream opens with a heartbeat")

	filled := make(chan struct{}, 4)
	go func() {
		for stream.Receive() {
			if stream.Msg().GetChargesFilled() != nil {
				filled <- struct{}{}
			}
		}
	}()
	return filled
}

func (p *gamer) dropBomb(tile uint32, country string) {
	p.t.Helper()

	drop := connect.NewRequest(&planetv1.DropBombRequest{Target: pointOf(p.t, tile), CountryId: country})
	p.send(drop.Header())
	_, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).DropBomb(p.t.Context(), drop)
	require.NoError(p.t, err)
}

func (s gameStack) fill(t *testing.T, request string) {
	t.Helper()

	_, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, s.consoleURL).
		FillCharges(t.Context(), connect.NewRequest(&planetv1.FillChargesRequest{RequestId: request}))
	require.NoError(t, err)
}

func (s gameStack) finishedFills(t *testing.T, count int) []*planetv1.ChargeFill {
	t.Helper()

	var fills []*planetv1.ChargeFill
	require.Eventually(t, func() bool {
		res, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, s.consoleURL).
			GetChargeFills(t.Context(), connect.NewRequest(&planetv1.GetChargeFillsRequest{}))
		require.NoError(t, err)
		fills = res.Msg.GetFills()
		for _, fill := range fills {
			if fill.GetFinishedAt() == nil {
				return false
			}
		}
		return len(fills) == count
	}, 10*time.Second, 50*time.Millisecond)
	return fills
}

func told(t *testing.T, notices <-chan struct{}, within time.Duration) {
	t.Helper()

	require.Eventually(t, func() bool {
		select {
		case <-notices:
			return true
		default:
			return false
		}
	}, within, 20*time.Millisecond)
}

func full() *planetv1.ChargesHeld {
	return &planetv1.ChargesHeld{Refill: true, Bomb: true, Enclosures: 3, SpreadClicksLeft: 8, Shields: 30}
}

func TestTheFinaleFillsOnceTheHandOfEveryoneWhoPlayedBeforeIt(t *testing.T) {
	starts := time.Now().Add(8 * time.Second)

	game := startGameWith(t, func(configs stackConfigs) {
		*configs.seasons = finaleAt(t, *configs.seasons, starts, time.Hour)
		configs.seasons.Fills.Finale = true
		configs.planet.Rules = rushIn(t).Rules
	})
	ada := game.newPlayer(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	notices := ada.chargesFilled(ctx)

	ada.click(1, "fr")
	require.True(t, time.Now().Before(starts), "the stack booted too slowly to see the days before the finale")
	assert.Equal(t, &planetv1.ChargesHeld{}, ada.charges())

	told(t, notices, time.Until(starts)+5*time.Second)
	bob := game.newPlayer(t)
	bob.click(2, "fr")

	fills := game.finishedFills(t, 1)
	assert.Equal(t, "season-0:finale:fill", fills[0].GetRequestId())
	assert.Equal(t, planetv1.FillSource_FILL_SOURCE_INTERNAL, fills[0].GetSource())
	assert.Equal(t, uint32(1), fills[0].GetAccountsFilled())
	assert.Equal(t, full(), ada.charges(), "the stream said to read them again, and they are full")
	assert.Equal(t, &planetv1.ChargesHeld{}, bob.charges(), "an account that first played after the start gets nothing")

	ada.dropBomb(1, "fr")
	game.fill(t, "season-0:finale:fill")
	again := game.finishedFills(t, 1)
	assert.Equal(t, uint32(1), again[0].GetAccountsFilled())
	assert.False(t, ada.charges().GetBomb(), "the fill sent again gives nothing")
	assert.Equal(t, &planetv1.ChargesHeld{}, bob.charges())

	game.fill(t, "saturday-backup")
	backup := game.finishedFills(t, 2)
	assert.Equal(t, "saturday-backup", backup[0].GetRequestId())
	assert.Equal(t, planetv1.FillSource_FILL_SOURCE_OPERATOR, backup[0].GetSource())
	assert.Equal(t, uint32(2), backup[0].GetAccountsFilled())
	assert.Equal(t, full(), ada.charges(), "another request fills again")
	assert.Equal(t, full(), bob.charges())
}
