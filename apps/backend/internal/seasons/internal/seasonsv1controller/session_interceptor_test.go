package seasonsv1controller_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller"
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
		return connect.NewResponse(&seasonsv1.GetMySeasonResponse{}), nil
	})
	inner := connect.NewRequest(&seasonsv1.GetMySeasonRequest{})
	if token != "" {
		inner.Header().Set(cpconnect.SessionHeader, token)
	}

	interceptor := seasonsv1controller.NewSessionInterceptor(identityVerifier{}, cptime.SystemClock{}, prometheus.NewRegistry())
	_, err := interceptor.WrapUnary(next)(t.Context(), fakeRequest{AnyRequest: inner, spec: connect.Spec{Procedure: procedure}})

	return account, err
}

func TestMySeasonTakesAnIdentityTokenAndNamesItsAccount(t *testing.T) {
	account, err := ask(t, seasonsv1connect.SeasonServiceGetMySeasonProcedure, "an-identity-token")

	require.NoError(t, err)
	assert.Equal(t, holder.String(), account)
}

func TestMySeasonWithNoTokenIsUnauthenticated(t *testing.T) {
	_, err := ask(t, seasonsv1connect.SeasonServiceGetMySeasonProcedure, "")

	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.ErrorIs(t, err, seasonsv1controller.ErrNoSession)
}

func TestTheStandingsNeedNoToken(t *testing.T) {
	_, err := ask(t, seasonsv1connect.SeasonServiceGetStandingsProcedure, "")

	require.NoError(t, err)
}
