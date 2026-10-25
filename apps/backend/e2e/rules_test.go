package e2e_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
)

func finaleAt(t *testing.T, kept seasons.Config, starts time.Time, length time.Duration) seasons.Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "finale.yaml")
	require.NoError(t, os.WriteFile(path, fmt.Appendf(nil, `
seasons:
  list:
    - number: 0
      endsAt: %s
      finale: %s
  ruleSets:
    finale: rush
`, starts.Add(length).UTC().Format(time.RFC3339Nano), length), 0o600))
	var config struct{ Seasons seasons.Config }
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(path)))
	config.Seasons.Database, config.Seasons.Snapshot = kept.Database, kept.Snapshot

	return config.Seasons
}

func rushIn(t *testing.T) planet.Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "rules.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
planet:
  rules:
    sets:
      rush:
        tempo: 3
`), 0o600))
	var config struct{ Planet planet.Config }
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(path)))

	return config.Planet
}

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

func TestTheFinalePlaysTheRushRulesAndTheEndPlaysTheDefaultsAgain(t *testing.T) {
	const finale = 4 * time.Second
	starts := time.Now().Add(8 * time.Second)

	game := startGameWith(t, func(configs stackConfigs) {
		*configs.seasons = finaleAt(t, *configs.seasons, starts, finale)
		configs.planet.Rules = rushIn(t).Rules
	})
	ada := game.newPlayer(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	changes := ada.ruleChanges(ctx)

	ada.click(1, "fr")
	require.True(t, time.Now().Before(starts), "the stack booted too slowly to see the days before the finale")
	assert.InDelta(t, 100, ada.refillPerSecond(), 1e-9)

	select {
	case <-changes:
	case <-time.After(time.Until(starts) + 5*time.Second):
		t.Fatal("no change of rules when the finale started")
	}
	assert.InDelta(t, 300, ada.refillPerSecond(), 1e-9, "every open tab reads the faster refill")

	select {
	case <-changes:
	case <-time.After(time.Until(starts.Add(finale)) + 5*time.Second):
		t.Fatal("no change of rules when the season ended")
	}
	assert.InDelta(t, 100, ada.refillPerSecond(), 1e-9)

	res, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL).
		GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)

	asked := make([]string, 0, len(res.Msg.GetChanges()))
	for _, change := range res.Msg.GetChanges() {
		asked = append(asked, change.GetRequestId()+" "+change.GetRuleSet())
		assert.Equal(t, planetv1.RuleChangeSource_RULE_CHANGE_SOURCE_INTERNAL, change.GetSource())
	}
	assert.Equal(t, []string{"season-0:over default", "season-0:finale rush", "season-0:days default"}, asked)
	require.Len(t, res.Msg.GetSets(), 2)
	assert.InDelta(t, 3, res.Msg.GetSets()[1].GetTempo(), 1e-9)
}

func TestAnOperatorSwitchesTheRulesByHandAndTheSwitchSurvivesARequestSentAgain(t *testing.T) {
	game := startGameWith(t, func(configs stackConfigs) {
		configs.planet.Rules = rushIn(t).Rules
	})
	ada := game.newPlayer(t)
	ada.click(1, "fr")
	desk := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL)

	_, err := desk.ApplyRuleSet(t.Context(), connect.NewRequest(&planetv1.ApplyRuleSetRequest{RequestId: "test-1", RuleSet: "rush"}))
	require.NoError(t, err)
	assert.InDelta(t, 300, ada.refillPerSecond(), 1e-9)

	_, err = desk.ApplyRuleSet(t.Context(), connect.NewRequest(&planetv1.ApplyRuleSetRequest{RequestId: "test-2", RuleSet: "default"}))
	require.NoError(t, err)
	_, err = desk.ApplyRuleSet(t.Context(), connect.NewRequest(&planetv1.ApplyRuleSetRequest{RequestId: "test-1", RuleSet: "rush"}))
	require.NoError(t, err, "a request sent again is no error")
	assert.InDelta(t, 100, ada.refillPerSecond(), 1e-9, "and changes nothing")

	_, err = desk.ApplyRuleSet(t.Context(), connect.NewRequest(&planetv1.ApplyRuleSetRequest{RequestId: "test-3", RuleSet: "frozen"}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	res, err := desk.GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)
	var operator []string
	for _, change := range res.Msg.GetChanges() {
		if change.GetSource() == planetv1.RuleChangeSource_RULE_CHANGE_SOURCE_OPERATOR {
			operator = append(operator, change.GetRequestId())
		}
	}
	assert.Equal(t, []string{"test-2", "test-1"}, operator)
}
