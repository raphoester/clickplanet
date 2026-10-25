package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	metricsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metrics/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/metrics/v1/metricsv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
)

// A sequencer that is sure after four steps, alone: a guest walking the tile ids is banned at once.
func withAQuickSequencer(configs stackConfigs) {
	configs.planet.AntiBot.Enabled = true
	configs.planet.AntiBot.Database = configs.planet.Database
	configs.planet.AntiBot.Database.Schema = "antibot"
	configs.planet.AntiBot.Bans.Enforce = true
	configs.planet.AntiBot.Sequencer.Enabled = true
	configs.planet.AntiBot.Sequencer.Detector.MinSteps = 3
	configs.planet.AntiBot.Sequencer.Detector.MinShare = 0.5
	configs.planet.AntiBot.Sequencer.Detector.CertainSteps = 4
	configs.planet.AntiBot.Sequencer.Detector.CertainShare = 0.5
}

func (p *gamer) convictionsFromThePage(since time.Time) (*planetv1.ListConvictionsResponse, error) {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.ListConvictionsRequest{Since: timestamppb.New(since)})
	p.fromThePage(req.Header(), adminOrigin)
	res, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, p.stack.adminURL).ListConvictions(p.t.Context(), req)
	if err != nil {
		return nil, err //nolint:wrapcheck // the tests read the connect code.
	}
	return res.Msg, nil
}

func (p *gamer) metricsFromThePage() map[string]float64 {
	p.t.Helper()

	req := connect.NewRequest(&metricsv1.GetMetricsRequest{})
	p.fromThePage(req.Header(), adminOrigin)
	res, err := metricsv1connect.NewAdminServiceClient(http.DefaultClient, p.stack.adminURL).GetMetrics(p.t.Context(), req)
	require.NoError(p.t, err)

	counters := map[string]float64{}
	for _, metric := range res.Msg.GetMetrics() {
		counters[metric.GetName()] += metric.GetCounter()
	}
	return counters
}

func (p *gamer) raceFromThePage() *seasonsv1.Race {
	p.t.Helper()

	req := connect.NewRequest(&seasonsv1.GetRaceRequest{})
	p.fromThePage(req.Header(), adminOrigin)
	res, err := seasonsv1connect.NewAdminServiceClient(http.DefaultClient, p.stack.adminURL).GetRace(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetRace()
}

func TestTheOperatorWatchesTheBattleFromThePage(t *testing.T) {
	game := startGameWith(t, withAQuickSequencer)
	operator := newOperator(t, game)
	require.NoError(t, operator.adminSignIn("the-operator"))
	since := time.Now().Add(-time.Second)
	sweeper := game.newPlayer(t)

	var refused error
	for tile := uint32(1); tile <= 10 && refused == nil; tile++ {
		refused = sweeper.tryClick(tile, "fr")
	}
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(refused), "the click after the ban is refused")

	listed, err := operator.convictionsFromThePage(since)
	require.NoError(t, err)
	require.NotEmpty(t, listed.GetConvictions())
	conviction := listed.GetConvictions()[0]
	assert.Equal(t, sweeper.account(), conviction.GetAccountId())
	assert.Equal(t, callerIP, conviction.GetScope())
	assert.Equal(t, uint32(1), conviction.GetOffence())
	require.Len(t, conviction.GetReadings(), 1)
	assert.Equal(t, "sequencer", conviction.GetReadings()[0].GetWatchdog())
	assert.Equal(t, "certain", conviction.GetReadings()[0].GetLevel())

	counters := operator.metricsFromThePage()
	assert.InDelta(t, 1, counters["banned_clicks"], 0)
	assert.Positive(t, counters["clicks_total"])
	assert.Contains(t, counters, "throttled_clicks")

	public := game.raceWhere(t, func(race *seasonsv1.Race) bool { return len(race.GetRound().GetStandings()) == 1 })
	read := operator.raceFromThePage()
	require.Len(t, read.GetRound().GetStandings(), 1)
	assert.Equal(t, public.GetRound().GetNumber(), read.GetRound().GetNumber())
	assert.Equal(t, "fr", read.GetRound().GetStandings()[0].GetCountryId(), "the share moves with every snapshot, the country does not")
	assert.Equal(t, public.GetRound().GetStandings()[0].GetPoints(), read.GetRound().GetStandings()[0].GetPoints())

	_, err = sweeper.convictionsFromThePage(since)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "cp_sid alone is no admin session")
}
