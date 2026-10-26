package planetv1controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_holdings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_holdings_handler/holdings_query"
)

type movingMap struct{ version clicks.MapVersion }

func (m *movingMap) Version() clicks.MapVersion { return m.version }

func (m *movingMap) Tally() (clicks.MapVersion, clicks.Tally) {
	return m.version, clicks.Tally{Countries: []clicks.FlagTiles{{Flag: "fr", Tiles: 3}}}
}

func holdingsServer(t *testing.T, board *movingMap) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewClickServiceHandler(
		ClickService{GetHoldingsHandler: get_holdings_handler.New(holdings_query.NewMemoryQuery(board))},
		connect.WithInterceptors(errorNet(), NewCacheInterceptor(board)),
	))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

func getHoldings(t *testing.T, server *httptest.Server, ifNoneMatch string) (int, http.Header) {
	t.Helper()

	query := url.Values{"connect": {"v1"}, "encoding": {"json"}, "message": {"{}"}}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		server.URL+planetv1connect.ClickServiceGetHoldingsProcedure+"?"+query.Encode(), nil)
	require.NoError(t, err)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	res, err := server.Client().Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	return res.StatusCode, res.Header
}

func TestTheHoldingsAreCachedFiveSecondsAndTaggedWithTheirVersion(t *testing.T) {
	server := holdingsServer(t, &movingMap{version: clicks.MapVersion{Epoch: 1_790_000_000, Seq: 42}})

	status, header := getHoldings(t, server, "")

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "public, max-age=5, s-maxage=5", header.Get("Cache-Control"))
	assert.Equal(t, `"1790000000.42"`, header.Get("Etag"))
}

func TestACopyAtTheMapsVersionIsNotSentAgain(t *testing.T) {
	board := &movingMap{version: clicks.MapVersion{Epoch: 1_790_000_000, Seq: 42}}
	server := holdingsServer(t, board)

	for _, held := range []string{`"1790000000.42"`, `W/"1790000000.42"`, `"1790000000.41", "1790000000.42"`} {
		status, header := getHoldings(t, server, held)
		assert.Equal(t, http.StatusNotModified, status, held)
		assert.Equal(t, `"1790000000.42"`, header.Get("Etag"), held)
		assert.Equal(t, "public, max-age=5, s-maxage=5", header.Get("Cache-Control"), held)
	}

	board.version.Seq++
	status, header := getHoldings(t, server, `"1790000000.42"`)
	assert.Equal(t, http.StatusOK, status, "a write since: the copy is old")
	assert.Equal(t, `"1790000000.43"`, header.Get("Etag"))
}

func TestACopyFromAnEarlierStartIsSentAgain(t *testing.T) {
	server := holdingsServer(t, &movingMap{version: clicks.MapVersion{Epoch: 1_790_000_100, Seq: 42}})

	status, _ := getHoldings(t, server, `"1790000000.42"`)

	assert.Equal(t, http.StatusOK, status)
}
