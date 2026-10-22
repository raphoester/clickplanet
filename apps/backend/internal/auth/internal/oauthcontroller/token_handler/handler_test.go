package token_handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admintokens"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/exchange_code_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/refresh_tokens_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/oauthcontroller/token_handler"
)

func pair(t *testing.T) *admintokens.Pair {
	t.Helper()
	issued, err := admintokens.NewIssuer(&admintokens.SequentialSecrets{}).Pair(admintokens.Holder{Client: "Claude"}, time.Now())
	require.NoError(t, err)
	return issued
}

type exchange struct {
	pair  *admintokens.Pair
	asked *exchange_code_usecase.In
}

func (e exchange) Execute(_ context.Context, in exchange_code_usecase.In) (*admintokens.Pair, error) {
	*e.asked = in
	switch in.Code {
	case "spent":
		return nil, fmt.Errorf("%w: redeemed already", connections.ErrInvalidGrant)
	case "broken":
		return nil, errors.New("postgres is down")
	}
	return e.pair, nil
}

type refresh struct {
	pair  *admintokens.Pair
	asked *refresh_tokens_usecase.In
}

func (r refresh) Execute(_ context.Context, in refresh_tokens_usecase.In) (*admintokens.Pair, error) {
	*r.asked = in
	if in.RefreshToken == "cprt_spent" {
		return nil, admintokens.ErrUnknownToken
	}
	return r.pair, nil
}

type fixture struct {
	handler   http.Handler
	exchanged exchange_code_usecase.In
	refreshed refresh_tokens_usecase.In
}

func setUp(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{}
	issued := pair(t)
	f.handler = token_handler.New(exchange{pair: issued, asked: &f.exchanged}, refresh{pair: issued, asked: &f.refreshed},
		slog.New(slog.DiscardHandler))
	return f
}

func (f *fixture) post(t *testing.T, form url.Values, basic string) (int, http.Header, map[string]any) {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if basic != "" {
		request.SetBasicAuth(basic, "a secret nobody issued")
	}
	recorder := httptest.NewRecorder()
	f.handler.ServeHTTP(recorder, request)

	var answer map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))
	return recorder.Code, recorder.Header(), answer
}

func TestACodeIsTradedForABearerPairNobodyCaches(t *testing.T) {
	f := setUp(t)

	status, header, answer := f.post(t, url.Values{
		"grant_type": {"authorization_code"}, "code": {"the code"}, "client_id": {"the id"},
		"redirect_uri": {"https://claude.ai/cb"}, "code_verifier": {"the verifier"},
	}, "")

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, map[string]any{ //nolint:gosec // SequentialSecrets, not credentials.
		"access_token": "cpat_secret-1", "token_type": "Bearer", "expires_in": float64(3600), "refresh_token": "cprt_secret-2",
	}, answer)
	assert.Equal(t, "no-store", header.Get("Cache-Control"))
	assert.Equal(t, exchange_code_usecase.In{Code: "the code", Redemption: connections.Redemption{
		ClientID: "the id", RedirectURI: "https://claude.ai/cb", Verifier: "the verifier",
	}}, f.exchanged)
}

func TestARefreshTokenIsTradedAndTheClientMayNameItselfWithBasic(t *testing.T) {
	f := setUp(t)

	status, _, _ := f.post(t, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"cprt_live"}}, "the id")

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, refresh_tokens_usecase.In{RefreshToken: "cprt_live", ClientID: "the id"}, f.refreshed)
}

func TestEachRefusalIsTheOAuthErrorOfItsReason(t *testing.T) {
	f := setUp(t)

	for name, c := range map[string]struct {
		form   url.Values
		status int
		error  string
	}{
		"a spent code":     {url.Values{"grant_type": {"authorization_code"}, "code": {"spent"}}, http.StatusBadRequest, "invalid_grant"},
		"a spent refresh":  {url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"cprt_spent"}}, http.StatusBadRequest, "invalid_grant"},
		"a password":       {url.Values{"grant_type": {"password"}}, http.StatusBadRequest, "unsupported_grant_type"},
		"nothing":          {url.Values{}, http.StatusBadRequest, "unsupported_grant_type"},
		"postgres is down": {url.Values{"grant_type": {"authorization_code"}, "code": {"broken"}}, http.StatusInternalServerError, "server_error"},
	} {
		status, _, answer := f.post(t, c.form, "")
		assert.Equal(t, c.status, status, name)
		assert.Equal(t, c.error, answer["error"], name)
		assert.NotContains(t, answer["error_description"], "postgres", name)
	}
}
