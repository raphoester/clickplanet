package scoringv1controller_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var holder = cpsession.AccountID{15: 1}

type identityVerifier struct{}

func (identityVerifier) Verify(context.Context, string, string, time.Time) (*cpsession.Claims, error) {
	return &cpsession.Claims{ID: "a-mint", Account: holder}, nil
}

type fakeRequest struct {
	connect.AnyRequest
	spec connect.Spec
}

func (r fakeRequest) Spec() connect.Spec {
	return r.spec
}

func ask(t *testing.T, procedure string, token string) (string, error) {
	t.Helper()

	var account string
	next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		account = cpctx.GetAccount(ctx)
		return connect.NewResponse(&scoringv1.GetMyTallyResponse{}), nil
	})
	inner := connect.NewRequest(&scoringv1.GetMyTallyRequest{})
	if token != "" {
		inner.Header().Set(cpconnect.SessionHeader, token)
	}

	interceptor := scoringv1controller.NewSessionInterceptor(identityVerifier{}, cptime.SystemClock{}, prometheus.NewRegistry())
	_, err := interceptor.WrapUnary(next)(t.Context(), fakeRequest{AnyRequest: inner, spec: connect.Spec{Procedure: procedure}})

	return account, err
}

func TestMyTallyTakesAnIdentityTokenAndNamesItsAccount(t *testing.T) {
	account, err := ask(t, scoringv1connect.ScoringServiceGetMyTallyProcedure, "an-identity-token")

	require.NoError(t, err)
	assert.Equal(t, holder.String(), account)
}

func TestMyTallyWithNoTokenIsUnauthenticated(t *testing.T) {
	_, err := ask(t, scoringv1connect.ScoringServiceGetMyTallyProcedure, "")

	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.ErrorIs(t, err, scoringv1controller.ErrNoSession)
}

func TestTheBoardNeedsNoToken(t *testing.T) {
	_, err := ask(t, scoringv1connect.ScoringServiceGetBoardProcedure, "")

	require.NoError(t, err)
}
