package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1/boxesv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

// A box's whole way, across four modules: the boxes offer it and take the catch, and tell it through their outbox; the
// bonuses draw what it holds and ask planet to grant it; planet tells the grant on the player's own stream.
func (s *testSuite) TestAGuestCatchesABoxAndPlanetTellsWhatItHeldOnTheGuestsStream() {
	t := s.T()
	s.game.boxesComeEverySecond(t)
	guest := s.game.newPlayer(t)
	grants := guest.grants()
	offers := guest.boxOffers()

	guest.click(5000, "nl")

	offer := nextOf(t, offers, 10*time.Second, "the boxes offer a box to a guest that clicked, its stream open")
	claim := connect.NewRequest(&boxesv1.ClaimBoxRequest{Token: offer.GetToken()})
	guest.send(claim.Header())
	_, err := guest.boxes().ClaimBox(t.Context(), claim)
	require.NoError(t, err, "the boxes take the catch")

	granted := nextOf(t, grants, 10*time.Second,
		"the boxes tell the catch (BoxClaimed), the bonuses draw what it held and planet grants it")
	require.NotEmpty(t, granted.GetGained(), "a fresh hand has room for what the box held")
	for _, gained := range granted.GetGained() {
		assert.Positive(t, gained.GetAmount(), "%s", gained.GetKind())
	}
	assert.True(t, proto.Equal(guest.charges(), granted.GetCharges()), "the hand told is the hand planet keeps")
}

// boxesComeEverySecond makes the boxes' default offer a box every second while the test runs, and puts it back after:
// a box waits minutes otherwise. Saving the set in force plays it at once.
func (s gameStack) boxesComeEverySecond(t *testing.T) {
	t.Helper()

	admin := boxesv1connect.NewAdminServiceClient(http.DefaultClient, s.consoleURL)
	rules, err := admin.GetRules(t.Context(), connect.NewRequest(&boxesv1.GetRulesRequest{}))
	require.NoError(t, err)
	var kept *boxesv1.Rules
	for _, set := range rules.Msg.GetSets() {
		if set.GetName() == "default" {
			kept = set.GetRules()
		}
	}
	require.NotNil(t, kept, "the boxes keep a default")

	save := func(ctx context.Context, rules *boxesv1.Rules) error {
		_, err := admin.SaveRuleSet(ctx, connect.NewRequest(&boxesv1.SaveRuleSetRequest{
			Set: &boxesv1.RuleSet{Name: "default", Rules: rules},
		}))
		return err //nolint:wrapcheck // the test reads it.
	}
	quick := proto.CloneOf(kept)
	quick.MinInterval = durationpb.New(time.Second)
	quick.MaxInterval = durationpb.New(time.Second)
	require.NoError(t, save(t.Context(), quick))
	// The test's context is cancelled before its cleanups run.
	t.Cleanup(func() { assert.NoError(t, save(context.WithoutCancel(t.Context()), kept)) })
}

func (p *gamer) boxes() boxesv1connect.BoxServiceClient {
	return boxesv1connect.NewBoxServiceClient(http.DefaultClient, p.stack.baseURL)
}

// boxOffers are the boxes offered to the player, on the boxes' stream: a box goes only to a caller with it open.
func (p *gamer) boxOffers() <-chan *boxesv1.BoxOffered {
	p.t.Helper()

	ctx, cancel := context.WithCancel(p.t.Context())
	p.t.Cleanup(cancel)
	req := connect.NewRequest(&boxesv1.ListenForEventsRequest{})
	p.send(req.Header())
	stream, err := p.boxes().ListenForEvents(ctx, req)
	require.NoError(p.t, err)
	require.True(p.t, stream.Receive(), "the stream opens with a heartbeat: %v", stream.Err())

	offers := make(chan *boxesv1.BoxOffered, 16)
	go func() {
		defer close(offers)
		for stream.Receive() {
			if offer := stream.Msg().GetBoxOffered(); offer != nil {
				offers <- offer
			}
		}
	}()
	return offers
}

// grants are what planet tells the player's own stream of a grant from outside its clicks: a box caught, a right
// answer, a mission paid.
func (p *gamer) grants() <-chan *planetv1.ChargesGranted {
	p.t.Helper()

	ctx, cancel := context.WithCancel(p.t.Context())
	p.t.Cleanup(cancel)
	req := connect.NewRequest(&planetv1.ListenForEventsRequest{})
	p.send(req.Header())
	stream, err := p.clicks().ListenForEvents(ctx, req)
	require.NoError(p.t, err)
	require.True(p.t, stream.Receive(), "the stream opens with a heartbeat: %v", stream.Err())

	grants := make(chan *planetv1.ChargesGranted, 16)
	go func() {
		defer close(grants)
		for stream.Receive() {
			if granted := stream.Msg().GetChargesGranted(); granted != nil {
				grants <- granted
			}
		}
	}()
	return grants
}

func nextOf[T any](t *testing.T, events <-chan T, within time.Duration, what string) T {
	t.Helper()

	select {
	case event, open := <-events:
		require.True(t, open, "the stream ended before: %s", what)
		return event
	case <-time.After(within):
		require.FailNow(t, "nothing came within "+within.String(), what)
		var none T
		return none
	}
}
