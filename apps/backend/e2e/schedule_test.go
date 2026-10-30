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
	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
)

// The set the entry plays is each module's default under another name, so the tests after it play as before.
func (s *testSuite) TestABattleOfTheScheduleIsPlayedByPlanetAndTheAntibotAndScoredByScoring() {
	t := s.T()
	ctx := t.Context()
	planet := planetv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)
	antibot := antibotv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)
	schedule := gamemasterv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)

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

	startsAt := time.Now().Add(2 * time.Second)
	endsAt := startsAt.UTC().Truncate(24 * time.Hour).Add(21 * time.Hour)
	if !endsAt.After(startsAt) {
		endsAt = endsAt.Add(24 * time.Hour)
	}
	added, err := schedule.AddEntry(ctx, connect.NewRequest(&gamemasterv1.AddEntryRequest{
		RuleSet: "encore", Kind: gamemasterv1.EntryKind_ENTRY_KIND_BATTLE, Description: "The same rules, once more.",
		StartsAt: timestamppb.New(startsAt), EndsAt: timestamppb.New(endsAt), Priority: 1, Times: 2,
	}))
	require.NoError(t, err, "the gamemaster module asks planet and the antibot over the internal listener whether they keep the set")

	scoring := scoringv1connect.NewScoringServiceClient(http.DefaultClient, s.game.baseURL)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		planetRules, err := planet.GetRules(ctx, connect.NewRequest(&planetv1.GetRulesRequest{}))
		require.NoError(c, err)
		assert.Equal(c, "encore", planetRules.Msg.GetPlaying())
		antibotRules, err := antibot.GetRules(ctx, connect.NewRequest(&antibotv1.GetRulesRequest{}))
		require.NoError(c, err)
		assert.Equal(c, "encore", antibotRules.Msg.GetPlaying())
		round, err := scoring.GetRound(ctx, connect.NewRequest(&scoringv1.GetRoundRequest{RoundId: added.Msg.GetId()}))
		require.NoError(c, err)
		assert.Equal(c, uint32(2), round.Msg.GetRound().GetTimes())
	}, 15*time.Second, 100*time.Millisecond, "the gamemaster commands the set to planet and the antibot, and opens the battle's round in scoring")

	playing, err := schedule.GetSchedule(ctx, connect.NewRequest(&gamemasterv1.GetScheduleRequest{}))
	require.NoError(t, err)
	assert.Equal(t, added.Msg.GetId(), playing.Msg.GetPlaying().GetEntryId())

	public := gamemasterv1connect.NewGamemasterServiceClient(http.DefaultClient, s.game.baseURL)
	turn, err := public.GetTurn(ctx, connect.NewRequest(&gamemasterv1.GetTurnRequest{}))
	require.NoError(t, err, "the game client reads the turn on the public listener")
	assert.Equal(t, gamemasterv1.EntryKind_ENTRY_KIND_BATTLE, turn.Msg.GetTurn().GetKind())
	assert.Equal(t, "The same rules, once more.", turn.Msg.GetTurn().GetDescription())
	assert.Equal(t, endsAt.UnixMilli(), turn.Msg.GetTurn().GetEndsAt().AsTime().UnixMilli())
	assert.Equal(t, added.Msg.GetId(), turn.Msg.GetTurn().GetRoundId(), "a battle's round is its entry")
	assert.Equal(t, uint32(2), turn.Msg.GetTurn().GetTimes())

	listening, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stream, err := public.ListenForEvents(listening, connect.NewRequest(&gamemasterv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	defer func() { _ = stream.Close() }()
	require.True(t, stream.Receive(), "the stream opens with the turn: %v", stream.Err())
	assert.Equal(t, gamemasterv1.EntryKind_ENTRY_KIND_BATTLE, stream.Msg().GetTurn().GetKind())
}
