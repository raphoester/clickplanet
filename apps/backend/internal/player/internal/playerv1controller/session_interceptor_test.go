package playerv1controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ada = cpsession.AccountID{15: 1}

type tokenVerifier map[string]cpsession.Claims

func (v tokenVerifier) Verify(_ context.Context, token string, _ string, _ time.Time) (*cpsession.Claims, error) {
	claims, ok := v[token]
	if !ok {
		return nil, errors.New("not minted here")
	}
	return &claims, nil
}

var verifier = tokenVerifier{
	"a-click-token":     {ID: "a-mint", Account: ada, Attested: true},
	"an-identity-token": {ID: "a-resume", Account: ada},
}

type fakeRequest struct {
	connect.AnyRequest
	spec connect.Spec
}

func (r fakeRequest) Spec() connect.Spec {
	return r.spec
}

func run(t *testing.T, procedure string, token string) (string, bool, error) {
	t.Helper()

	var (
		account string
		ran     bool
	)
	next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		account, ran = cpctx.GetAccount(ctx), true
		return connect.NewResponse(&playerv1.GetProfileResponse{}), nil
	})

	inner := connect.NewRequest(&playerv1.GetProfileRequest{})
	if token != "" {
		inner.Header().Set(cpconnect.SessionHeader, token)
	}
	interceptor := NewSessionInterceptor(verifier, cptime.SystemClock{}, prometheus.NewRegistry())
	_, err := interceptor.WrapUnary(next)(t.Context(), fakeRequest{AnyRequest: inner, spec: connect.Spec{Procedure: procedure}})

	return account, ran, err
}

func TestTheProceduresThatActForTheCallerNeedAClickToken(t *testing.T) {
	for _, procedure := range []string{
		playerv1connect.PlayerServiceSetNameProcedure,
		playerv1connect.PlayerServiceSetColorProcedure,
		playerv1connect.PlayerServiceAnnounceProcedure,
	} {
		account, ran, err := run(t, procedure, "a-click-token")
		require.NoError(t, err, procedure)
		require.True(t, ran, procedure)
		require.Equal(t, ada.String(), account, procedure)

		_, ran, err = run(t, procedure, "an-identity-token")
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), procedure)
		require.ErrorIs(t, err, ErrNoSession, procedure)
		require.False(t, ran, procedure)
	}
}

func TestTheReadsOfTheCallerTakeTheIdentityTokenOfAResumedSession(t *testing.T) {
	for _, procedure := range []string{
		playerv1connect.PlayerServiceGetProfileProcedure,
		playerv1connect.PlayerServiceLeaveProcedure,
	} {
		account, ran, err := run(t, procedure, "an-identity-token")
		require.NoError(t, err, procedure)
		require.True(t, ran, procedure)
		require.Equal(t, ada.String(), account, procedure)

		_, ran, err = run(t, procedure, "")
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), procedure)
		require.False(t, ran, procedure)
	}
}

func TestAnyonesPlayerTheRosterAndTheNameplatesNeedNoToken(t *testing.T) {
	for _, procedure := range []string{
		playerv1connect.PlayerServiceGetPlayerProcedure,
		playerv1connect.PlayerServiceGetRosterProcedure,
		playerv1connect.PlayerServiceGetNameplatesProcedure,
	} {
		for _, token := range []string{"", "forged"} {
			_, ran, err := run(t, procedure, token)
			require.NoError(t, err, procedure)
			require.True(t, ran, procedure)
		}
	}
}
