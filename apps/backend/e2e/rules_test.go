package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

func (p *gamer) refillPerSecond() float64 {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GetBudgetRequest{CountryId: "fr"})
	p.send(req.Header())
	res, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).GetBudget(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetBudget().GetRefillPerSecond()
}

func (p *gamer) ruleChanges(ctx context.Context) <-chan struct{} {
	p.t.Helper()

	listen := connect.NewRequest(&planetv1.ListenForEventsRequest{})
	p.send(listen.Header())
	stream, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).ListenForEvents(ctx, listen)
	require.NoError(p.t, err)
	require.True(p.t, stream.Receive(), "the stream opens with a heartbeat")

	changes := make(chan struct{}, 4)
	go func() {
		for stream.Receive() {
			if stream.Msg().GetRulesChanged() != nil {
				changes <- struct{}{}
			}
		}
	}()
	return changes
}

func defaultWithTempo(tempo float64) *planetv1.RuleSet {
	return &planetv1.RuleSet{
		Name:  "default",
		Tempo: tempo,
		Boxes: &planetv1.BoxPace{
			MinInterval: durationpb.New(4 * time.Minute), MaxInterval: durationpb.New(8 * time.Minute), MaxChargesPerHour: 12,
		},
		AntiBot: &planetv1.AntiBotRules{Crowd: 6},
	}
}

func rushSet() *planetv1.RuleSet {
	rush := defaultWithTempo(3)
	rush.Name = "rush"
	rush.Boxes = &planetv1.BoxPace{
		MinInterval: durationpb.New(time.Minute), MaxInterval: durationpb.New(2 * time.Minute), MaxChargesPerHour: 30,
	}
	rush.AntiBot.Crowd = 50
	return rush
}

func nextChange(t *testing.T, changes <-chan struct{}, within time.Duration, what string) {
	t.Helper()

	select {
	case <-changes:
	case <-time.After(within):
		t.Fatalf("no change of rules %s", what)
	}
}

func TestTheGamePlaysTheRulesPostgresKeepsAndFollowsAnEditWithinASecond(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	changes := ada.ruleChanges(ctx)
	ada.click(1, "fr")
	desk := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL)

	rules, err := desk.GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)
	require.Len(t, rules.Msg.GetSets(), 1)
	assert.Equal(t, "default", rules.Msg.GetSets()[0].GetName(), "the migration keeps the game's own rules")
	assert.InDelta(t, 100, ada.refillPerSecond(), 1e-9)

	_, err = desk.SaveRuleSet(t.Context(), connect.NewRequest(&planetv1.SaveRuleSetRequest{Set: defaultWithTempo(3)}))
	require.NoError(t, err)

	select {
	case <-changes:
	case <-time.After(5 * time.Second):
		t.Fatal("no change of rules after the edit")
	}
	assert.InDelta(t, 300, ada.refillPerSecond(), 1e-9, "every open tab reads the faster refill")
}

func TestASetThatCannotPlayIsRefusedAndAnotherIsKept(t *testing.T) {
	game := startGame(t)
	desk := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL)

	broken := defaultWithTempo(0)
	_, err := desk.SaveRuleSet(t.Context(), connect.NewRequest(&planetv1.SaveRuleSetRequest{Set: broken}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = desk.SaveRuleSet(t.Context(), connect.NewRequest(&planetv1.SaveRuleSetRequest{Set: rushSet()}))
	require.NoError(t, err)

	rules, err := desk.GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)
	require.Len(t, rules.Msg.GetSets(), 2)
	assert.InDelta(t, 1, rules.Msg.GetSets()[0].GetTempo(), 1e-9, "the refused edit changed nothing")
	assert.Equal(t, "rush", rules.Msg.GetSets()[1].GetName())
	assert.Equal(t, uint32(50), rules.Msg.GetSets()[1].GetAntiBot().GetCrowd())
}

func TestTheFinalePlaysTheRushBaselineAndTheEndPlaysTheDefaultAgain(t *testing.T) {
	const finale = 4 * time.Second
	starts := time.Now().Add(20 * time.Second)

	game := startGameWith(t, func(configs stackConfigs) {
		configs.seasons.Calendar.List[0].EndsAt = starts.Add(finale)
		configs.seasons.Calendar.List[0].Finale = finale
		configs.seasons.RuleSets.Finale = "rush"
	})
	desk := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL)
	_, err := desk.SaveRuleSet(t.Context(), connect.NewRequest(&planetv1.SaveRuleSetRequest{Set: rushSet()}))
	require.NoError(t, err)
	ada := game.newPlayer(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	changes := ada.ruleChanges(ctx)

	ada.click(1, "fr")
	require.True(t, time.Now().Before(starts), "the stack booted too slowly to see the days before the finale")
	assert.InDelta(t, 100, ada.refillPerSecond(), 1e-9)

	nextChange(t, changes, time.Until(starts)+5*time.Second, "when the finale started")
	assert.InDelta(t, 300, ada.refillPerSecond(), 1e-9, "every open tab reads the faster refill")

	nextChange(t, changes, time.Until(starts.Add(finale))+5*time.Second, "when the season ended")
	assert.InDelta(t, 100, ada.refillPerSecond(), 1e-9)

	res, err := desk.GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)
	applied := make([]string, 0, len(res.Msg.GetBaselines()))
	for _, baseline := range res.Msg.GetBaselines() {
		applied = append(applied, baseline.GetRequestId()+" "+baseline.GetRuleSet())
	}
	assert.Equal(t, []string{"season-0:over default", "season-0:finale rush", "season-0:days default"}, applied)
	assert.Nil(t, res.Msg.GetOverride())
}

func TestAnOverridePlaysOverTheBaselineUntilItsEndOrUntilItIsEnded(t *testing.T) {
	game := startGame(t)
	desk := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL)
	_, err := desk.SaveRuleSet(t.Context(), connect.NewRequest(&planetv1.SaveRuleSetRequest{Set: rushSet()}))
	require.NoError(t, err)
	ada := game.newPlayer(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	changes := ada.ruleChanges(ctx)
	ada.click(1, "fr")

	_, err = desk.OverrideRules(t.Context(), connect.NewRequest(&planetv1.OverrideRulesRequest{
		RuleSet: "rush", Until: timestamppb.New(time.Now().Add(3 * time.Second)),
	}))
	require.NoError(t, err)
	nextChange(t, changes, 5*time.Second, "when the override started")
	assert.InDelta(t, 300, ada.refillPerSecond(), 1e-9)
	nextChange(t, changes, 8*time.Second, "when the override ended")
	assert.InDelta(t, 100, ada.refillPerSecond(), 1e-9, "the baseline comes back by itself")

	_, err = desk.OverrideRules(t.Context(), connect.NewRequest(&planetv1.OverrideRulesRequest{
		RuleSet: "rush", Until: timestamppb.New(time.Now().Add(time.Hour)),
	}))
	require.NoError(t, err)
	nextChange(t, changes, 5*time.Second, "when the second override started")
	_, err = desk.EndOverride(t.Context(), connect.NewRequest(&planetv1.EndOverrideRequest{}))
	require.NoError(t, err)
	nextChange(t, changes, 5*time.Second, "when the override was ended")
	assert.InDelta(t, 100, ada.refillPerSecond(), 1e-9)

	_, err = desk.OverrideRules(t.Context(), connect.NewRequest(&planetv1.OverrideRulesRequest{
		RuleSet: "rush", Until: timestamppb.New(time.Now().Add(25 * time.Hour)),
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "an override runs a day at most")
}

func TestASeasonsRequestSentAgainChangesNothing(t *testing.T) {
	game := startGame(t)
	desk := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL)
	_, err := desk.SaveRuleSet(t.Context(), connect.NewRequest(&planetv1.SaveRuleSetRequest{Set: rushSet()}))
	require.NoError(t, err)
	planet := planetv1connect.NewInternalServiceClient(http.DefaultClient, game.internalURL)

	for _, applied := range []*planetv1.ApplyRuleSetRequest{
		{RequestId: "test-1", RuleSet: "rush"},
		{RequestId: "test-2", RuleSet: "default"},
		{RequestId: "test-1", RuleSet: "rush"},
	} {
		_, err := planet.ApplyRuleSet(t.Context(), connect.NewRequest(applied))
		require.NoError(t, err, "a request sent again is no error")
	}
	_, err = planet.ApplyRuleSet(t.Context(), connect.NewRequest(&planetv1.ApplyRuleSetRequest{
		RequestId: "test-3", RuleSet: "frozen",
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	res, err := desk.GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, res.Msg.GetBaselines())
	assert.Equal(t, "test-2", res.Msg.GetBaselines()[0].GetRequestId(), "the request applied since stands")
}
