package authorize_handler_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/authorize_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/oauthcontroller/authorize_handler"
)

type stub struct {
	asked *authorize_usecase.In
}

func (s stub) Execute(_ context.Context, in authorize_usecase.In) (string, error) {
	*s.asked = in
	switch in.ClientID {
	case "refused":
		return "", &connections.Refusal{URL: "https://claude.ai/cb?error=invalid_request"}
	case "unknown":
		return "", fmt.Errorf("%w: not sealed here", connections.ErrInvalidClient)
	case "broken":
		return "", errors.New("the sealer is on fire")
	}
	return "https://backoffice.clickplanet.lol/connect?request=sealed", nil
}

func authorize(query string) (*httptest.ResponseRecorder, authorize_usecase.In) {
	var asked authorize_usecase.In
	recorder := httptest.NewRecorder()
	authorize_handler.New(stub{asked: &asked}, slog.New(slog.DiscardHandler)).
		ServeHTTP(recorder, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/oauth/authorize?"+query, nil))
	return recorder, asked
}

func TestTheBrowserIsSentToTheConsentPageWithWhatTheAppAsked(t *testing.T) {
	recorder, asked := authorize("response_type=code&client_id=the+id&redirect_uri=https%3A%2F%2Fclaude.ai%2Fcb" +
		"&code_challenge=the+challenge&code_challenge_method=S256&state=s&scope=ignored&resource=ignored")

	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, "https://backoffice.clickplanet.lol/connect?request=sealed", recorder.Header().Get("Location"))
	assert.Equal(t, authorize_usecase.In{ClientID: "the id", Ask: connections.Ask{
		ResponseType: "code", RedirectURI: "https://claude.ai/cb", Challenge: "the challenge", ChallengeMethod: "S256", State: "s",
	}}, asked)
}

func TestARefusalGoesBackToTheAppOnlyWhenTheAppIsKnown(t *testing.T) {
	recorder, _ := authorize("client_id=refused")
	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, "https://claude.ai/cb?error=invalid_request", recorder.Header().Get("Location"))

	recorder, _ = authorize("client_id=unknown&redirect_uri=https%3A%2F%2Fevil.example")
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Empty(t, recorder.Header().Get("Location"), "nobody vouched for that redirect")

	recorder, _ = authorize("client_id=broken")
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "on fire")
}
