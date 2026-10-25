package e2e_test

import (
	"errors"
	"fmt"
	"io"
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
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
)

func gapBetween(t *testing.T, kept seasons.Config, ends, starts time.Time) seasons.Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "gap.yaml")
	require.NoError(t, os.WriteFile(path, fmt.Appendf(nil, `
seasons:
  list:
    - number: 0
      endsAt: %s
      finale: 1h
    - number: 1
      startsAt: %s
      endsAt: %s
      finale: 2h
  ruleSets:
    over: frozen
`, ends.UTC().Format(time.RFC3339Nano), starts.UTC().Format(time.RFC3339Nano),
		starts.Add(24*time.Hour).UTC().Format(time.RFC3339Nano)), 0o600))
	var config struct{ Seasons seasons.Config }
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(path)))
	config.Seasons.Database, config.Seasons.Snapshot = kept.Database, kept.Snapshot

	return config.Seasons
}

func frozenIn(t *testing.T) planet.Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "rules.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
planet:
  rules:
    sets:
      frozen:
        frozen: true
`), 0o600))
	var config struct{ Planet planet.Config }
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(path)))

	return config.Planet
}

func frozen(err error) bool {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeFailedPrecondition {
		return false
	}
	for _, detail := range connectErr.Details() {
		if value, err := detail.Value(); err == nil {
			if _, ok := value.(*planetv1.MapFrozen); ok {
				return true
			}
		}
	}
	return false
}

func (p *gamer) dropBomb(tile uint32) error {
	p.t.Helper()

	drop := connect.NewRequest(&planetv1.DropBombRequest{Target: pointOf(p.t, tile), CountryId: "fr"})
	p.send(drop.Header())
	_, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).DropBomb(p.t.Context(), drop)
	return err //nolint:wrapcheck // the test reads the code.
}

func (p *gamer) charges() *planetv1.ChargesHeld {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GetChargesRequest{})
	p.send(req.Header())
	res, err := planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL).GetCharges(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetCharges()
}

func (s gameStack) metrics(t *testing.T) string {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, s.baseURL+"/metrics", nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return string(body)
}

func TestTheMapFreezesBetweenTwoSeasonsAndOpensWhenTheNextStarts(t *testing.T) {
	ends := time.Now().Add(8 * time.Second)
	starts := ends.Add(6 * time.Second)

	game := startGameWith(t, func(configs stackConfigs) {
		*configs.seasons = gapBetween(t, *configs.seasons, ends, starts)
		configs.planet.Rules = frozenIn(t).Rules
	})
	ada := game.newPlayer(t)
	ada.click(1, "fr")
	require.True(t, time.Now().Before(ends), "the stack booted too slowly to see Season 0 open")
	_, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL).GrantCharges(t.Context(),
		connect.NewRequest(&planetv1.GrantChargesRequest{AccountId: ada.account(), Bomb: true}))
	require.NoError(t, err)

	require.Eventually(t, func() bool { return frozen(ada.tryClick(2, "fr")) },
		time.Until(ends)+5*time.Second, 100*time.Millisecond, "the map never froze")
	require.True(t, time.Now().Before(starts), "Season 1 started before the freeze was seen")

	assert.True(t, frozen(ada.dropBomb(1)))
	assert.True(t, ada.charges().GetBomb(), "a bomb refused as frozen is kept")
	assert.Contains(t, game.metrics(t), "\nfrozen_clicks ")

	season, err := ada.seasons().GetSeason(t.Context(), connect.NewRequest(&seasonsv1.GetSeasonRequest{}))
	require.NoError(t, err)
	assert.Nil(t, season.Msg.GetSeason(), "no season is in play")
	assert.Equal(t, uint32(1), season.Msg.GetNext().GetNumber())
	assert.Equal(t, starts.UnixMilli(), season.Msg.GetNext().GetStartsAtUnixMs())

	require.Eventually(t, func() bool { return ada.tryClick(2, "fr") == nil },
		time.Until(starts)+5*time.Second, 100*time.Millisecond, "the map never opened again")

	rules, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.consoleURL).
		GetRules(t.Context(), connect.NewRequest(&planetv1.GetRulesRequest{}))
	require.NoError(t, err)
	asked := make([]string, 0, len(rules.Msg.GetChanges()))
	for _, change := range rules.Msg.GetChanges() {
		asked = append(asked, change.GetRequestId()+" "+change.GetRuleSet())
	}
	assert.Equal(t, []string{"season-1:days default", "season-0:over frozen", "season-0:finale default"}, asked)
}
