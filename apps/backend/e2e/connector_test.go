package e2e_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

const (
	claudeCallback = "https://claude.ai/api/mcp/auth_callback"
	verifier       = "the-verifier-claude-draws-for-its-connection-0001"
)

// What Claude does, from the URL alone: an app that never redirects on its own, so each hop is read.
type app struct {
	t        *testing.T
	stack    gameStack
	clientID string
	access   string
	refresh  string
}

var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

// What came back, read whole: the body is closed before the test looks.
type answered struct {
	StatusCode int
	Header     http.Header
}

func (a *app) do(method, address, contentType string, body io.Reader, header http.Header) (answered, []byte) {
	a.t.Helper()
	req, err := http.NewRequestWithContext(a.t.Context(), method, address, body)
	require.NoError(a.t, err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for name, values := range header {
		req.Header[name] = values
	}
	res, err := noRedirect.Do(req)
	require.NoError(a.t, err)
	defer func() { _ = res.Body.Close() }()
	read, err := io.ReadAll(res.Body)
	require.NoError(a.t, err)
	return answered{StatusCode: res.StatusCode, Header: res.Header}, read
}

func (a *app) mcp(token, body string) (answered, map[string]any) {
	a.t.Helper()
	header := http.Header{"Accept": {"application/json, text/event-stream"}}
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	res, read := a.do(http.MethodPost, a.stack.adminURL+"/mcp", "application/json", strings.NewReader(body), header)
	var answer map[string]any
	if len(read) > 0 && res.Header.Get("Content-Type") == "application/json" {
		require.NoError(a.t, json.Unmarshal(read, &answer))
	}
	return res, answer
}

func (a *app) json(address string) map[string]any {
	a.t.Helper()
	res, read := a.do(http.MethodGet, address, "", nil, nil)
	require.Equal(a.t, http.StatusOK, res.StatusCode, address)
	var answer map[string]any
	require.NoError(a.t, json.Unmarshal(read, &answer))
	return answer
}

// Discovery, registration and the authorize hop: where the operator's browser is sent, on the admin page.
func (a *app) askToConnect() (metadata map[string]any, consent *url.URL) {
	a.t.Helper()

	res, _ := a.mcp("", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	require.Equal(a.t, http.StatusUnauthorized, res.StatusCode)
	found := regexp.MustCompile(`resource_metadata="([^"]+)"`).FindStringSubmatch(res.Header.Get("WWW-Authenticate"))
	require.Len(a.t, found, 2)

	resource := a.json(found[1])
	assert.Equal(a.t, a.stack.adminURL+"/mcp", resource["resource"])
	servers, ok := resource["authorization_servers"].([]any)
	require.True(a.t, ok)
	metadata = a.json(servers[0].(string) + "/.well-known/oauth-authorization-server")

	res, read := a.do(http.MethodPost, metadata["registration_endpoint"].(string), "application/json",
		strings.NewReader(`{"client_name":"Claude","redirect_uris":["`+claudeCallback+`"],"token_endpoint_auth_method":"none"}`), nil)
	require.Equal(a.t, http.StatusCreated, res.StatusCode, string(read))
	var registered map[string]any
	require.NoError(a.t, json.Unmarshal(read, &registered))
	a.clientID = registered["client_id"].(string)

	sum := sha256.Sum256([]byte(verifier))
	authorize := metadata["authorization_endpoint"].(string) + "?" + url.Values{
		"response_type": {"code"}, "client_id": {a.clientID}, "redirect_uri": {claudeCallback}, "state": {"the state"},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"},
		"resource": {a.stack.adminURL + "/mcp"},
	}.Encode()
	res, _ = a.do(http.MethodGet, authorize, "", nil, nil)
	require.Equal(a.t, http.StatusFound, res.StatusCode)
	consent, err := url.Parse(res.Header.Get("Location"))
	require.NoError(a.t, err)
	return metadata, consent
}

func (a *app) trade(tokenEndpoint string, form url.Values) (int, map[string]any) {
	a.t.Helper()
	form.Set("client_id", a.clientID)
	res, read := a.do(http.MethodPost, tokenEndpoint, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()), nil)
	var answer map[string]any
	require.NoError(a.t, json.Unmarshal(read, &answer))
	if res.StatusCode == http.StatusOK {
		a.access, a.refresh = answer["access_token"].(string), answer["refresh_token"].(string)
	}
	return res.StatusCode, answer
}

func (p *gamer) describeConnection(request string) (*authv1.DescribeConnectionResponse, error) {
	p.t.Helper()
	req := connect.NewRequest(&authv1.DescribeConnectionRequest{Request: request})
	p.fromThePage(req.Header(), adminOrigin)
	res, err := p.desk().DescribeConnection(p.t.Context(), req)
	if err != nil {
		return nil, err //nolint:wrapcheck // the tests read the connect code.
	}
	return res.Msg, nil
}

func (p *gamer) approveConnection(request string) string {
	p.t.Helper()
	req := connect.NewRequest(&authv1.ApproveConnectionRequest{Request: request})
	p.fromThePage(req.Header(), adminOrigin)
	res, err := p.desk().ApproveConnection(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetRedirectUrl()
}

// The whole connection, as Claude makes it: the URL, a 401, the metadata, a registration, the operator's yes on the
// admin page, a code traded for tokens, then tools.
func connectClaude(t *testing.T, game gameStack, operator *gamer) (*app, map[string]any) {
	t.Helper()
	claude := &app{t: t, stack: game}

	metadata, consent := claude.askToConnect()
	assert.Equal(t, adminOrigin+"/connect", consent.Scheme+"://"+consent.Host+consent.Path)
	request := consent.Query().Get("request")

	described, err := operator.describeConnection(request)
	require.NoError(t, err)
	assert.Equal(t, "Claude", described.GetClientName())
	assert.Equal(t, "claude.ai", described.GetRedirectHost())

	back, err := url.Parse(operator.approveConnection(request))
	require.NoError(t, err)
	assert.Equal(t, claudeCallback, back.Scheme+"://"+back.Host+back.Path)
	assert.Equal(t, "the state", back.Query().Get("state"))
	assert.Equal(t, metadata["issuer"], back.Query().Get("iss"))

	status, answer := claude.trade(metadata["token_endpoint"].(string), url.Values{
		"grant_type": {"authorization_code"}, "code": {back.Query().Get("code")}, "redirect_uri": {claudeCallback},
		"code_verifier": {verifier},
	})
	require.Equal(t, http.StatusOK, status, answer)
	assert.Equal(t, "Bearer", answer["token_type"])

	status, answer = claude.trade(metadata["token_endpoint"].(string), url.Values{
		"grant_type": {"authorization_code"}, "code": {back.Query().Get("code")}, "redirect_uri": {claudeCallback},
		"code_verifier": {verifier},
	})
	assert.Equal(t, http.StatusBadRequest, status, "a code is good once")
	assert.Equal(t, "invalid_grant", answer["error"])

	return claude, metadata
}

func TestClaudeConnectsWithTheOperatorsYesAndCallsAToolAsTheOperator(t *testing.T) {
	game := startGame(t)
	operator := newOperator(t, game)
	require.NoError(t, operator.adminSignIn("the-operator"))
	claude, _ := connectClaude(t, game, operator)

	res, answer := claude.mcp(claude.access, `{"jsonrpc":"2.0","id":1,"method":"initialize",
		"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"claude-ai","version":"0.1.0"}}}`)
	require.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "2025-06-18", answer["result"].(map[string]any)["protocolVersion"])

	_, answer = claude.mcp(claude.access, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	listed := map[string]bool{}
	for _, tool := range answer["result"].(map[string]any)["tools"].([]any) {
		annotations := tool.(map[string]any)["annotations"].(map[string]any)
		listed[tool.(map[string]any)["name"].(string)] = annotations["readOnlyHint"].(bool)
	}
	assert.True(t, listed["planet_top_players"])
	assert.False(t, listed["planet_grant_charges"])
	assert.NotContains(t, listed, "auth_approve_connection")

	_, answer = claude.mcp(claude.access, `{"jsonrpc":"2.0","id":3,"method":"tools/call",
		"params":{"name":"planet_grant_charges","arguments":{"accountId":"`+operator.account()+`","bomb":true}}}`)
	result := answer["result"].(map[string]any)
	require.Equal(t, false, result["isError"], result)
	var granted planetv1.GrantChargesResponse
	require.NoError(t, protojson.Unmarshal([]byte(result["content"].([]any)[0].(map[string]any)["text"].(string)), &granted))
	assert.True(t, granted.GetAfter().GetBomb(), "the operator's own account got its bomb")

	_, answer = claude.mcp(claude.access, `{"jsonrpc":"2.0","id":4,"method":"tools/call",
		"params":{"name":"planet_unban_player","arguments":{"accountId":"`+operator.account()+`"}}}`)
	result = answer["result"].(map[string]any)
	assert.Equal(t, true, result["isError"], "a refusal is the tool's result")
	assert.Contains(t, result["content"].([]any)[0].(map[string]any)["text"], "failed_precondition",
		"the antibot is off in this stack, and the call says so")
}

func TestARefreshTokenBuysTheNextPairOnce(t *testing.T) {
	game := startGame(t)
	operator := newOperator(t, game)
	require.NoError(t, operator.adminSignIn("the-operator"))
	claude, metadata := connectClaude(t, game, operator)
	first := claude.refresh

	status, _ := claude.trade(metadata["token_endpoint"].(string), url.Values{"grant_type": {"refresh_token"}, "refresh_token": {first}})
	require.Equal(t, http.StatusOK, status)
	assert.NotEqual(t, first, claude.refresh)

	res, _ := claude.mcp(claude.access, `{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	assert.Equal(t, http.StatusOK, res.StatusCode, "the new access token calls")

	status, answer := claude.trade(metadata["token_endpoint"].(string), url.Values{"grant_type": {"refresh_token"}, "refresh_token": {first}})
	assert.Equal(t, http.StatusBadRequest, status, "spent")
	assert.Equal(t, "invalid_grant", answer["error"])
}

func TestUnflaggingTheOperatorEndsTheConnection(t *testing.T) {
	game := startGame(t)
	operator := newOperator(t, game)
	require.NoError(t, operator.adminSignIn("the-operator"))
	claude, _ := connectClaude(t, game, operator)

	_, err := game.schema(t, "auth").ExecContext(t.Context(), `UPDATE accounts SET operator = false WHERE id = $1`, operator.account())
	require.NoError(t, err)

	res, _ := claude.mcp(claude.access, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	assert.Equal(t, http.StatusForbidden, res.StatusCode)
}

func TestAnAppsTokenCallsTheAdminServicesAndNeverLetsAnAppIn(t *testing.T) {
	game := startGame(t)
	operator := newOperator(t, game)
	require.NoError(t, operator.adminSignIn("the-operator"))
	claude, _ := connectClaude(t, game, operator)
	_, consent := (&app{t: t, stack: game}).askToConnect()

	approve := connect.NewRequest(&authv1.ApproveConnectionRequest{Request: consent.Query().Get("request")})
	approve.Header().Set("Authorization", "Bearer "+claude.access)
	_, err := operator.desk().ApproveConnection(t.Context(), approve)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "only the page's session lets an app in")

	top := connect.NewRequest(&planetv1.TopPlayersRequest{})
	top.Header().Set("Authorization", "Bearer "+claude.access)
	_, err = planetv1connect.NewAdminServiceClient(http.DefaultClient, game.adminURL).TopPlayers(t.Context(), top)
	require.NoError(t, err, "from any origin, as the operator")
}

func TestARequestNobodyApprovedLetsNothingIn(t *testing.T) {
	game := startGame(t)
	claude := &app{t: t, stack: game}
	metadata, _ := claude.askToConnect()

	status, answer := claude.trade(metadata["token_endpoint"].(string), url.Values{
		"grant_type": {"authorization_code"}, "code": {"made-up"}, "redirect_uri": {claudeCallback}, "code_verifier": {verifier},
	})
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "invalid_grant", answer["error"])

	res, read := claude.do(http.MethodPost, metadata["registration_endpoint"].(string), "application/json",
		bytes.NewReader([]byte(`{"client_name":"Evil","redirect_uris":["https://evil.example/cb"]}`)), nil)
	assert.Equal(t, http.StatusBadRequest, res.StatusCode, string(read))
}
