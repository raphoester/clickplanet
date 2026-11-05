package cpconnect_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type accountVerifier struct {
	account cpsession.AccountID
}

func (v accountVerifier) Verify(context.Context, string, string, time.Time) (*cpsession.Claims, error) {
	return &cpsession.Claims{ID: "a-mint", Account: v.account, Attested: true}, nil
}

type fakeRequest struct {
	connect.AnyRequest
}

func (fakeRequest) Spec() connect.Spec {
	return connect.Spec{Procedure: procedure}
}

func TestAnEnforcedSessionPutsTheAccountAndWhenItWasMadeOnTheContext(t *testing.T) {
	created := time.Date(2026, 10, 1, 13, 37, 26, 938_000_000, time.UTC)
	account := cpsession.AccountCreatedAt(created)
	var (
		named  string
		madeAt time.Time
	)
	next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		named, madeAt = cpctx.GetAccount(ctx), cpctx.GetAccountCreated(ctx)
		return connect.NewResponse(&emptypb.Empty{}), nil
	})
	inner := connect.NewRequest(&emptypb.Empty{})
	inner.Header().Set(cpconnect.SessionHeader, "a-token")

	interceptor := cpconnect.NewSessionInterceptor(accountVerifier{account: account}, cptime.SystemClock{},
		errors.New("no session"), true, func(cpconnect.SessionVerdict) {}, cpconnect.Attested(procedure))
	_, err := interceptor.WrapUnary(next)(t.Context(), fakeRequest{AnyRequest: inner})

	require.NoError(t, err)
	assert.Equal(t, account.String(), named)
	assert.Equal(t, created, madeAt)
}
