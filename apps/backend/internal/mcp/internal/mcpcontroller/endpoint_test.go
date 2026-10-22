package mcpcontroller_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/mcpcontroller"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools/usecases/call_tool_usecase"
)

const metadataURL = "https://api.clickplanet.lol/.well-known/oauth-protected-resource/mcp"

type holders struct{}

func (holders) HolderError(_ context.Context, token string) error {
	switch token {
	case "cpat_ada":
		return nil
	case "cpat_bob":
		return fmt.Errorf("%w: unflagged", tools.ErrNotAnOperator)
	case "cpat_down":
		return errors.New("auth is down")
	}
	return fmt.Errorf("%w: nobody", tools.ErrNoToken)
}

type calls struct {
	asked *call_tool_usecase.In
}

func (c calls) Execute(_ context.Context, in call_tool_usecase.In) (tools.Result, error) {
	*c.asked = in
	switch in.Name {
	case "planet_ban_player":
		return tools.Result{Text: `{"offence":1}`}, nil
	case "planet_unban_player":
		return tools.Result{Text: "not_found: no offence left", Refused: true}, nil
	case "planet_broken":
		return tools.Result{}, errors.New("the admin listener is down")
	}
	return tools.Result{}, fmt.Errorf("%w: %q", tools.ErrUnknownTool, in.Name)
}

type fixture struct {
	server *httptest.Server
	asked  call_tool_usecase.In
}

func setUp(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{}
	catalog := []tools.Tool{
		{
			Name: "planet_top_players", Title: "Top players", Description: "Who takes the most.", ReadOnly: true,
			InputSchema: tools.Schema{"type": "object"}, Procedure: "/planet.v1.AdminService/TopPlayers",
		},
		{
			Name: "planet_ban_player", Title: "Ban player", Description: "A shadow ban.",
			InputSchema: tools.Schema{"type": "object"}, Procedure: "/planet.v1.AdminService/BanPlayer",
		},
	}
	endpoint := mcpcontroller.NewEndpoint(catalog, "the instructions", metadataURL, holders{}, calls{asked: &f.asked},
		slog.New(slog.DiscardHandler))
	f.server = httptest.NewServer(endpoint)
	t.Cleanup(f.server.Close)
	return f
}

// What came back, read whole: the body is closed before the test looks.
type answered struct {
	status int
	header http.Header
}

func (f *fixture) post(t *testing.T, token, body string) (answered, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.server.URL, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	answer, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return answered{status: res.StatusCode, header: res.Header}, string(answer)
}

func TestNoTokenIsAnswered401WithWhereToGetOne(t *testing.T) {
	f := setUp(t)

	res, _ := f.post(t, "", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)

	assert.Equal(t, http.StatusUnauthorized, res.status)
	assert.Equal(t, `Bearer resource_metadata="`+metadataURL+`"`, res.header.Get("WWW-Authenticate"))
}

func TestATokenThatCannotCallIsRefusedBeforeAnyMethod(t *testing.T) {
	f := setUp(t)

	res, _ := f.post(t, "cpat_lapsed", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	assert.Equal(t, http.StatusUnauthorized, res.status)
	assert.Contains(t, res.header.Get("WWW-Authenticate"), `error="invalid_token"`, "the app refreshes, then connects again")

	res, _ = f.post(t, "cpat_bob", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	assert.Equal(t, http.StatusForbidden, res.status, "the account no longer operates")

	res, body := f.post(t, "cpat_down", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	assert.Equal(t, http.StatusInternalServerError, res.status)
	assert.NotContains(t, body, "auth is down")
}

func TestTheHandshakeEchoesAVersionItSpeaksAndSaysWhatTheToolsAreFor(t *testing.T) {
	f := setUp(t)

	res, body := f.post(t, "cpat_ada", `{"jsonrpc":"2.0","id":1,"method":"initialize",
		"params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"claude-ai","version":"0.1.0"}}}`)

	assert.Equal(t, http.StatusOK, res.status)
	assert.Equal(t, "application/json", res.header.Get("Content-Type"))
	assert.JSONEq(t, `{"jsonrpc":"2.0","id":1,"result":{
		"protocolVersion":"2025-03-26",
		"capabilities":{"tools":{"listChanged":false}},
		"serverInfo":{"name":"clickplanet","title":"ClickPlanet","version":"1"},
		"instructions":"the instructions"}}`, body)

	_, body = f.post(t, "cpat_ada", `{"jsonrpc":"2.0","id":"two","method":"initialize","params":{"protocolVersion":"2099-01-01"}}`)
	assert.Contains(t, body, `"protocolVersion":"2025-06-18"`, "the newest, for a version it does not speak")
	assert.Contains(t, body, `"id":"two"`)
}

func TestANotificationIsAcceptedWithNoAnswer(t *testing.T) {
	f := setUp(t)

	res, body := f.post(t, "cpat_ada", `{"jsonrpc":"2.0","method":"notifications/initialized"}`)

	assert.Equal(t, http.StatusAccepted, res.status)
	assert.Empty(t, body)
}

func TestTheToolsAreListedWithWhetherTheyOnlyRead(t *testing.T) {
	f := setUp(t)

	_, body := f.post(t, "cpat_ada", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)

	assert.JSONEq(t, `{"jsonrpc":"2.0","id":1,"result":{"tools":[
		{"name":"planet_top_players","title":"Top players","description":"Who takes the most.","inputSchema":{"type":"object"},
		 "annotations":{"title":"Top players","readOnlyHint":true,"destructiveHint":false,"idempotentHint":true,"openWorldHint":false}},
		{"name":"planet_ban_player","title":"Ban player","description":"A shadow ban.","inputSchema":{"type":"object"},
		 "annotations":{"title":"Ban player","readOnlyHint":false,"destructiveHint":true,"idempotentHint":false,"openWorldHint":false}}
	]}}`, body)
}

func TestACallIsMadeWithTheTokenItCameWith(t *testing.T) {
	f := setUp(t)

	_, body := f.post(t, "cpat_ada", `{"jsonrpc":"2.0","id":7,"method":"tools/call",
		"params":{"name":"planet_ban_player","arguments":{"accountId":"the account"}}}`)

	assert.JSONEq(t, `{"jsonrpc":"2.0","id":7,"result":{"content":[{"type":"text","text":"{\"offence\":1}"}],"isError":false}}`, body)
	assert.Equal(t, "planet_ban_player", f.asked.Name)
	assert.Equal(t, "cpat_ada", f.asked.Token)
	assert.JSONEq(t, `{"accountId":"the account"}`, string(f.asked.Arguments))
}

func TestARefusalIsTheToolsResultAndOnlyAnUnknownToolIsAProtocolError(t *testing.T) {
	f := setUp(t)

	_, body := f.post(t, "cpat_ada", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"planet_unban_player"}}`)
	assert.JSONEq(t, `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"not_found: no offence left"}],"isError":true}}`, body)

	_, body = f.post(t, "cpat_ada", `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"planet_broken"}}`)
	assert.Contains(t, body, `"isError":true`)
	assert.NotContains(t, body, "down")

	_, body = f.post(t, "cpat_ada", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"planet_drop_database"}}`)
	assert.JSONEq(t, `{"jsonrpc":"2.0","id":3,"error":{"code":-32602,"message":"no tool planet_drop_database"}}`, body)
}

func TestWhatIsNotARequestOfThisProtocolIsAJSONRPCError(t *testing.T) {
	f := setUp(t)

	for name, c := range map[string]struct {
		body string
		code int
	}{
		"not json":       {`{"jsonrpc"`, -32700},
		"another method": {`{"jsonrpc":"2.0","id":1,"method":"resources/list"}`, -32601},
		"jsonrpc 1":      {`{"jsonrpc":"1.0","id":1,"method":"ping"}`, -32600},
	} {
		_, body := f.post(t, "cpat_ada", c.body)
		var answer struct {
			Error struct {
				Code int `json:"code"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal([]byte(body), &answer), name)
		assert.Equal(t, c.code, answer.Error.Code, name)
	}
}

func TestABatchIsAnsweredForItsRequests(t *testing.T) {
	f := setUp(t)

	_, body := f.post(t, "cpat_ada", `[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","method":"notifications/initialized"}]`)

	assert.JSONEq(t, `[{"jsonrpc":"2.0","id":1,"result":{}}]`, body)
}

func TestTheServerOpensNoStream(t *testing.T) {
	f := setUp(t)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.server.URL, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer cpat_ada")
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
}

func TestTheMetadataNamesTheResourceAndItsOwnOriginAsTheServer(t *testing.T) {
	resource, err := url.Parse("https://api.clickplanet.lol/mcp")
	require.NoError(t, err)

	inserted, bare := mcpcontroller.MetadataPaths(resource)
	assert.Equal(t, "/.well-known/oauth-protected-resource/mcp", inserted)
	assert.Equal(t, "/.well-known/oauth-protected-resource", bare)
	assert.Equal(t, metadataURL, mcpcontroller.MetadataURL(resource))

	recorder := httptest.NewRecorder()
	mcpcontroller.NewMetadata(resource).ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, inserted, nil))
	assert.JSONEq(t, `{"resource":"https://api.clickplanet.lol/mcp","authorization_servers":["https://api.clickplanet.lol"],
		"bearer_methods_supported":["header"],"resource_name":"ClickPlanet"}`, recorder.Body.String())
}
