package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type row struct {
	kind    string
	outcome string
	tile    int64
	held    string
	scope   string
	account string
}

func (s gameStack) activityRows(t *testing.T) []row {
	t.Helper()

	db := cppg.New(s.activity)
	require.NoError(t, db.ConnectCtx(t.Context()))
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(t.Context(), `
		SELECT kind, coalesce(data->>'outcome', ''), coalesce((data->>'tile')::bigint, 0), coalesce(data->>'held', ''),
			scope, coalesce(account::text, '')
		FROM events ORDER BY id`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()

	var out []row
	for rows.Next() {
		var r row
		require.NoError(t, rows.Scan(&r.kind, &r.outcome, &r.tile, &r.held, &r.scope, &r.account))
		out = append(out, r)
	}
	require.NoError(t, rows.Err())
	return out
}

func (p *gamer) account() string {
	p.t.Helper()

	_, public := cpsession.TestKeyPair()
	verifier, err := cpsession.NewVerifier(public)
	require.NoError(p.t, err)
	claims, err := verifier.Verify(p.token, callerIP, time.Now())
	require.NoError(p.t, err)
	return uuid.UUID(claims.Account).String()
}

func TestEveryEventOfACallerReachesTheActivity(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)
	client := planetv1connect.NewClickServiceClient(http.DefaultClient, game.baseURL)

	ada.click(1, "fr")
	ada.click(1, "fr")

	read := connect.NewRequest(&planetv1.GetMapRequest{StartTileId: 1, EndTileId: 11})
	ada.send(read.Header())
	_, err := client.GetMap(t.Context(), read)
	require.NoError(t, err)

	// The call returns with the first frame, a heartbeat away; the row is written when the stream opens.
	listen := connect.NewRequest(&planetv1.ListenForEventsRequest{})
	ada.send(listen.Header())
	listening, stop := context.WithCancel(t.Context())
	defer stop()
	go func() { _, _ = client.ListenForEvents(listening, listen) }()

	account := ada.account()
	want := []row{
		{kind: "take", tile: 1, scope: callerIP, account: account},
		{kind: "click", outcome: "accepted", tile: 1, scope: callerIP, account: account},
		{kind: "click", outcome: "noop", tile: 1, scope: callerIP, account: account},
		{kind: "map", outcome: "accepted", scope: callerIP},
		{kind: "stream", scope: callerIP, account: account},
	}
	require.Eventually(t, func() bool { return len(game.activityRows(t)) == len(want) }, 5*time.Second, 20*time.Millisecond)
	assert.Equal(t, want, game.activityRows(t), "a take is recorded before its try; GetMap reads no token")
}

func TestARefusedClickIsInTheActivityAndTakesNothing(t *testing.T) {
	game := startGame(t)
	ada := game.newPlayer(t)

	req := connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "atlantis"})
	ada.send(req.Header())
	_, err := planetv1connect.NewClickServiceClient(http.DefaultClient, game.baseURL).Click(t.Context(), req)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	require.Eventually(t, func() bool { return len(game.activityRows(t)) == 1 }, 5*time.Second, 20*time.Millisecond)
	assert.Equal(t, "invalid", game.activityRows(t)[0].outcome)
}
