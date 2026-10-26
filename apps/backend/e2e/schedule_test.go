package e2e_test

import (
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
)

// The set the entry plays is each module's default under another name, so the tests after it play as before.
func (s *testSuite) TestAnEntryOfTheScheduleIsPlayedByPlanetAndTheAntibot() {
	t := s.T()
	ctx := t.Context()
	planet := planetv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)
	antibot := antibotv1connect.NewAdminServiceClient(http.DefaultClient, s.game.consoleURL)
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

	startsAt := time.Now().Add(2 * time.Second)
	added, err := schedule.AddEntry(ctx, connect.NewRequest(&metagamev1.AddEntryRequest{
		RuleSet: "encore", StartsAt: timestamppb.New(startsAt), EndsAt: timestamppb.New(startsAt.Add(time.Hour)), Priority: 1,
	}))
	require.NoError(t, err, "the metagame module asks planet and the antibot over the internal listener whether they keep the set")

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		planetRules, err := planet.GetRules(ctx, connect.NewRequest(&planetv1.GetRulesRequest{}))
		require.NoError(c, err)
		assert.Equal(c, "encore", planetRules.Msg.GetPlaying())
		antibotRules, err := antibot.GetRules(ctx, connect.NewRequest(&antibotv1.GetRulesRequest{}))
		require.NoError(c, err)
		assert.Equal(c, "encore", antibotRules.Msg.GetPlaying())
	}, 15*time.Second, 100*time.Millisecond, "the metagame module commands the set over each module's command stream")

	playing, err := schedule.GetSchedule(ctx, connect.NewRequest(&metagamev1.GetScheduleRequest{}))
	require.NoError(t, err)
	assert.Equal(t, added.Msg.GetId(), playing.Msg.GetPlaying().GetEntryId())
}
