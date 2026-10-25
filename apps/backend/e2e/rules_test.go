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

	rush := defaultWithTempo(3)
	rush.Name = "rush"
	rush.AntiBot.Crowd = 50
	_, err = desk.SaveRuleSet(t.Context(), connect.NewRequest(&planetv1.SaveRuleSetRequest{Set: rush}))
	require.NoError(t, err)

	rules, err := desk.GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)
	require.Len(t, rules.Msg.GetSets(), 2)
	assert.InDelta(t, 1, rules.Msg.GetSets()[0].GetTempo(), 1e-9, "the refused edit changed nothing")
	assert.Equal(t, "rush", rules.Msg.GetSets()[1].GetName())
	assert.Equal(t, uint32(50), rules.Msg.GetSets()[1].GetAntiBot().GetCrowd())
}
