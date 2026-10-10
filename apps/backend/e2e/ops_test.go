package e2e_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/ops/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/ops/v1/opsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/ops"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

const signedByAccess = "signed-by-access"

type opsStack struct {
	baseURL string
	owner   *cppg.Postgres
}

func opsDatabase(t *testing.T) (*cppg.Postgres, ops.Config) {
	t.Helper()

	postgres := cppg.StartTestServer(t)
	owner := postgres.OpenSchema(t, "planet", fstest.MapFS{
		"1_create_tiles.up.sql":   {Data: []byte(`CREATE TABLE tiles (id integer PRIMARY KEY, country text NOT NULL)`)},
		"1_create_tiles.down.sql": {Data: []byte(`DROP TABLE tiles`)},
	})
	_, err := owner.ExecContext(t.Context(), ops.ReaderRole)
	require.NoError(t, err)
	_, err = owner.ExecContext(t.Context(), `
		ALTER ROLE ops_reader PASSWORD 'reader';
		INSERT INTO planet.tiles VALUES (1, 'fr'), (2, 'bg');
	`)
	require.NoError(t, err)

	one, none := 1, 0
	config := ops.Config{Enabled: true, Database: postgres.ConfigFor("public")}
	config.Database.User, config.Database.Password = "ops_reader", "reader"
	config.Database.Pool.MaxOpenConns, config.Database.Pool.MaxIdleConns = &one, &none

	return owner, config
}

func serveOps(t *testing.T, module cpbootstrap.Module) string {
	t.Helper()

	public := listen(t)
	server := cpbootstrap.ServerConfig{BindAddress: public.Addr().String()}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- cpbootstrap.RunOn(ctx, cpbootstrap.Options{
			Server: server, Logger: slog.New(slog.DiscardHandler), Modules: []cpbootstrap.Module{module},
		}, public)
	}()
	t.Cleanup(func() {
		cancel()
		assert.NoError(t, <-done)
	})

	waitUntilServed(t, server.BindAddress)

	return "http://" + server.BindAddress
}

func startOps(t *testing.T, configure func(*ops.Config)) opsStack {
	t.Helper()

	owner, config := opsDatabase(t)
	configure(&config)
	module := ops.NewModuleWithKnownAssertions(config, ops.KnownAssertions{signedByAccess: "claude-cloud.access"})

	return opsStack{baseURL: serveOps(t, module), owner: owner}
}

func (s opsStack) query(t *testing.T, assertion, statement string) (*opsv1.QueryResponse, error) {
	t.Helper()

	req := connect.NewRequest(&opsv1.QueryRequest{Statement: statement})
	if assertion != "" {
		req.Header().Set("Cf-Access-Jwt-Assertion", assertion)
	}
	res, err := opsv1connect.NewOpsServiceClient(http.DefaultClient, s.baseURL).Query(t.Context(), req)
	if err != nil {
		return nil, err //nolint:wrapcheck // the test reads the code off the error as it came.
	}

	return res.Msg, nil
}

func (s opsStack) tiles(t *testing.T) int {
	t.Helper()

	var count int
	require.NoError(t, s.owner.QueryRowContext(t.Context(), `SELECT count(*) FROM planet.tiles`).Scan(&count))

	return count
}

func asIs(*ops.Config) {}

func TestACallerAccessLetInReadsAnotherModulesTable(t *testing.T) {
	stack := startOps(t, asIs)

	answer, err := stack.query(t, signedByAccess, `SELECT id, country FROM planet.tiles ORDER BY id`)

	require.NoError(t, err)
	assert.Equal(t, []string{"id", "country"}, answer.GetColumns())
	require.Len(t, answer.GetRows(), 2)
	assert.Equal(t, []any{float64(1), "fr"}, answer.GetRows()[0].AsSlice())
	assert.Equal(t, []any{float64(2), "bg"}, answer.GetRows()[1].AsSlice())
}

func TestACallerAccessDidNotLetInReadsNothing(t *testing.T) {
	stack := startOps(t, asIs)

	for _, assertion := range []string{"", "forged"} {
		_, err := stack.query(t, assertion, `SELECT id FROM planet.tiles`)

		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), assertion)
	}
}

func TestAWriteOverTheWireIsRefusedAndChangesNothing(t *testing.T) {
	stack := startOps(t, asIs)

	for _, statement := range []string{`DELETE FROM planet.tiles`, `SELECT 1; DELETE FROM planet.tiles`} {
		_, err := stack.query(t, signedByAccess, statement)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), statement)
	}

	assert.Equal(t, 2, stack.tiles(t))
}

func TestOpsBootsWithoutItsRoleAndSaysSoWhenAsked(t *testing.T) {
	stack := startOps(t, func(config *ops.Config) { config.Database.User = "nobody_made_this_role" })

	_, err := stack.query(t, signedByAccess, `SELECT 1`)

	assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	require.ErrorContains(t, err, "password authentication failed")
}

func TestTheModuleAsItShipsRefusesAnAssertionItsTeamDidNotSign(t *testing.T) {
	team := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	t.Cleanup(team.Close)
	owner, config := opsDatabase(t)
	config.Access.Issuer, config.Access.Audience = team.URL, "the-application"
	stack := opsStack{baseURL: serveOps(t, ops.NewModule(config)), owner: owner}

	for _, assertion := range []string{"", "forged", "eyJhbGciOiJub25lIn0.eyJhdWQiOlsidGhlLWFwcGxpY2F0aW9uIl19."} {
		_, err := stack.query(t, assertion, `SELECT id FROM planet.tiles`)

		assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), assertion)
	}
}
