package e2e_test

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1/antibotv1connect"
	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1/metagamev1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
)

// The set the entry plays is each module's default under another name, so the tests after it play as before.
func (s *testSuite) TestAnEntryOfTheScheduleIsPlayedByPlanetTheAntibotAndSeasons() {
	t := s.T()
	ctx := t.Context()
	planet := planetv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)
	antibot := antibotv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)
	seasons := seasonsv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)
	schedule := metagamev1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)

	planetRules, err := planet.GetRules(ctx, connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)
	_, err = planet.SaveRuleSet(ctx, connect.NewRequest(&planetv1.SaveRuleSetRequest{
		Set: &planetv1.RuleSet{Name: "encore", Rules: planetRules.Msg.GetSets()[0].GetRules()},
	}))
	require.NoError(t, err)
	antibotRules, err := antibot.GetRules(ctx, connect.NewRequest(&antibotv1.GetRulesRequest{}))
	require.NoError(t, err)
	_, err = antibot.SaveRuleSet(ctx, connect.NewRequest(&antibotv1.SaveRuleSetRequest{
		Set: &antibotv1.RuleSet{Name: "encore", Rules: antibotRules.Msg.GetSets()[0].GetRules()},
	}))
	require.NoError(t, err)
	_, err = seasons.SaveRuleSet(ctx, connect.NewRequest(&seasonsv1.SaveRuleSetRequest{
		Set: &seasonsv1.RuleSet{Name: "encore", Rules: &seasonsv1.Rules{OwnRound: &seasonsv1.OwnRoundRules{Enabled: true, Times: 2}}},
	}))
	require.NoError(t, err)

	startsAt, endsAt := time.Now().Add(2*time.Second), time.Now().Add(time.Hour)
	added, err := schedule.AddEntry(ctx, connect.NewRequest(&metagamev1.AddEntryRequest{
		RuleSet: "encore", Kind: metagamev1.EntryKind_ENTRY_KIND_BATTLE, Description: "The same rules, once more.",
		StartsAt: timestamppb.New(startsAt), EndsAt: timestamppb.New(endsAt), Priority: 1,
	}))
	require.NoError(t, err, "the metagame module asks planet, the antibot and seasons over the internal listener whether they keep the set")

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		planetRules, err := planet.GetRules(ctx, connect.NewRequest(&planetv1.GetRulesRequest{}))
		require.NoError(c, err)
		assert.Equal(c, "encore", planetRules.Msg.GetPlaying())
		antibotRules, err := antibot.GetRules(ctx, connect.NewRequest(&antibotv1.GetRulesRequest{}))
		require.NoError(c, err)
		assert.Equal(c, "encore", antibotRules.Msg.GetPlaying())
		seasonRules, err := seasons.GetRules(ctx, connect.NewRequest(&seasonsv1.GetRulesRequest{}))
		require.NoError(c, err)
		assert.Equal(c, "encore", seasonRules.Msg.GetPlaying())
	}, 15*time.Second, 100*time.Millisecond, "the metagame module commands the set over each module's command stream")

	playing, err := schedule.GetSchedule(ctx, connect.NewRequest(&metagamev1.GetScheduleRequest{}))
	require.NoError(t, err)
	assert.Equal(t, added.Msg.GetId(), playing.Msg.GetPlaying().GetEntryId())

	public := metagamev1connect.NewMetagameServiceClient(http.DefaultClient, s.game.baseURL)
	turn, err := public.GetTurn(ctx, connect.NewRequest(&metagamev1.GetTurnRequest{}))
	require.NoError(t, err, "the game client reads the turn on the public listener")
	assert.Equal(t, metagamev1.EntryKind_ENTRY_KIND_BATTLE, turn.Msg.GetTurn().GetKind())
	assert.Equal(t, "The same rules, once more.", turn.Msg.GetTurn().GetDescription())
	assert.Equal(t, endsAt.UnixMilli(), turn.Msg.GetTurn().GetEndsAt().AsTime().UnixMilli())

	listening, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stream, err := public.ListenForEvents(listening, connect.NewRequest(&metagamev1.ListenForEventsRequest{}))
	require.NoError(t, err)
	defer func() { _ = stream.Close() }()
	require.True(t, stream.Receive(), "the stream opens with the turn: %v", stream.Err())
	assert.Equal(t, metagamev1.EntryKind_ENTRY_KIND_BATTLE, stream.Msg().GetTurn().GetKind())
}
