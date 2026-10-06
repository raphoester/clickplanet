package e2e_test

import (
	"database/sql"
	"encoding/binary"
	"math"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mapdata "github.com/raphoester/clickplanet.lol-backend/generated/map"
	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func (s gameStack) schema(t *testing.T, name string) *cppg.Postgres {
	t.Helper()

	db := cppg.New(s.postgres.ConfigFor(name))
	require.NoError(t, db.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type keptTake struct {
	tile    int
	country string
	takenAt time.Time
	account sql.NullString
}

func TestADeletedAccountsTakesKeepTheirTileFlagAndTimeAndNameNoAccount(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.link("google-ada")
	before := time.Now().Add(-time.Minute)
	ada.click(1, "fr")
	ada.click(2, "de")

	deletion := connect.NewRequest(&authv1.DeleteAccountRequest{})
	ada.send(deletion.Header())
	_, err := authv1connect.NewAuthServiceClient(http.DefaultClient, game.baseURL).DeleteAccount(t.Context(), deletion)
	require.NoError(t, err)

	planet := game.schema(t, "planet")
	require.Eventually(t, func() bool {
		var anonymous int
		err := planet.QueryRowContext(t.Context(), `SELECT count(*) FROM ledger_events WHERE account IS NULL`).Scan(&anonymous)
		return err == nil && anonymous == 2
	}, 5*time.Second, 20*time.Millisecond)

	rows, err := planet.QueryContext(t.Context(),
		`SELECT tile, country, taken_at, account::text FROM ledger_events ORDER BY position`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()

	var takes []keptTake
	for rows.Next() {
		var take keptTake
		require.NoError(t, rows.Scan(&take.tile, &take.country, &take.takenAt, &take.account))
		takes = append(takes, take)
	}
	require.NoError(t, rows.Err())

	require.Len(t, takes, 2)
	for i, want := range []struct {
		tile    int
		country string
	}{{1, "fr"}, {2, "de"}} {
		assert.Equal(t, want.tile, takes[i].tile)
		assert.Equal(t, want.country, takes[i].country)
		assert.WithinRange(t, takes[i].takenAt, before, time.Now())
		assert.False(t, takes[i].account.Valid, "the take names no account")
	}
}

func (p *gamer) account() string {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.GetProfileRequest{})
	p.send(req.Header())
	res, err := p.players().GetProfile(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetProfile().GetAccountId()
}

func pointOf(t *testing.T, tile uint32) *planetv1.GlobePoint {
	t.Helper()

	blob, _, err := mapdata.Coordinates()
	require.NoError(t, err)

	float := func(i uint32) float64 {
		offset := 12 + 4*((tile-1)*3+i)
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(blob[offset:])))
	}
	return &planetv1.GlobePoint{X: float(0), Y: float(1), Z: float(2)}
}

func TestABombIsOneEventInTheLedgerWithTheFlagEachTileItClearedWore(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	ada.click(1, "fr")

	grant := connect.NewRequest(&planetv1.GrantChargesRequest{AccountId: ada.account(), Bomb: true})
	_, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, game.adminURL).GrantCharges(t.Context(), grant)
	require.NoError(t, err)

	drop := connect.NewRequest(&planetv1.DropBombRequest{Target: pointOf(t, 1), CountryId: "de"})
	ada.send(drop.Header())
	_, err = planetv1connect.NewClickServiceClient(http.DefaultClient, game.baseURL).DropBomb(t.Context(), drop)
	require.NoError(t, err)

	planet := game.schema(t, "planet")
	var (
		tile           int
		country, flag  string
		previous, ours string
	)
	require.Eventually(t, func() bool {
		err := planet.QueryRowContext(t.Context(), `
			SELECT tile, country, previous, payload->>'flag', payload->'cleared'->>'fr'
			FROM ledger_events WHERE kind = 'bomb'`).Scan(&tile, &country, &previous, &flag, &ours)
		return err == nil
	}, 5*time.Second, 20*time.Millisecond)

	assert.Equal(t, 1, tile, "the bomb landed on the tile it was aimed at")
	assert.Empty(t, country, "and left it empty")
	assert.Equal(t, "fr", previous)
	assert.Equal(t, "de", flag)
	assert.Equal(t, "[1]", ours, "the only tile it cleared was the one fr held")
}
