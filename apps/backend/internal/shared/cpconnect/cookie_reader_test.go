package cpconnect_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const (
	cookieProcedure = "/test.v1.Service/Read"
	otherProcedure  = "/test.v1.Service/Click"
)

type stubCallers struct {
	account cpsession.AccountID
	err     error
	asked   []string
}

func (s *stubCallers) Caller(_ context.Context, cookie string) (cpsession.AccountID, error) {
	s.asked = append(s.asked, cookie)
	return s.account, s.err
}

type fakeRequest struct {
	connect.AnyRequest
	spec connect.Spec
}

func (r fakeRequest) Spec() connect.Spec { return r.spec }

type fakeConn struct {
	connect.StreamingHandlerConn
	spec   connect.Spec
	header http.Header
}

func (c fakeConn) Spec() connect.Spec { return c.spec }

func (c fakeConn) RequestHeader() http.Header { return c.header }

func unaryCaller(ctx context.Context, callers *stubCallers, procedure string, cookie string) string {
	var account string
	next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		account = cpctx.GetAccount(ctx)
		return connect.NewResponse(&emptypb.Empty{}), nil
	})

	inner := connect.NewRequest(&emptypb.Empty{})
	if cookie != "" {
		inner.Header().Set(cpconnect.CookieHeader, cookie)
	}
	_, _ = cpconnect.NewCookieReaderInterceptor(callers, cookieProcedure).WrapUnary(next)(
		ctx, fakeRequest{AnyRequest: inner, spec: connect.Spec{Procedure: procedure}})

	return account
}

var ada = cpsession.AccountID(uuid.Must(uuid.NewV7()))

func TestTheCookieNamesTheCallerOfAProcedureThatTakesIt(t *testing.T) {
	callers := &stubCallers{account: ada}

	assert.Equal(t, ada.String(), unaryCaller(t.Context(), callers, cookieProcedure, "cp_sid=token-1"))
	assert.Equal(t, []string{"cp_sid=token-1"}, callers.asked)
}

func TestTheCookieAlsoSaysWhenTheAccountWasMade(t *testing.T) {
	var created bool
	next := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		created = !cpctx.GetAccountCreated(ctx).IsZero()
		return connect.NewResponse(&emptypb.Empty{}), nil
	})
	inner := connect.NewRequest(&emptypb.Empty{})
	inner.Header().Set(cpconnect.CookieHeader, "cp_sid=token-1")

	_, err := cpconnect.NewCookieReaderInterceptor(&stubCallers{account: ada}, cookieProcedure).WrapUnary(next)(
		t.Context(), fakeRequest{AnyRequest: inner, spec: connect.Spec{Procedure: cookieProcedure}})

	require.NoError(t, err)
	assert.True(t, created)
}

func TestAValidTokenWinsAndTheCookieIsNotLookedUp(t *testing.T) {
	callers := &stubCallers{account: ada}
	ctx := cpctx.AddAccountToContext(t.Context(), "from-the-token")

	assert.Equal(t, "from-the-token", unaryCaller(ctx, callers, cookieProcedure, "cp_sid=token-1"))
	assert.Empty(t, callers.asked)
}

func TestNoCookieIsNotLookedUp(t *testing.T) {
	callers := &stubCallers{account: ada}

	assert.Empty(t, unaryCaller(t.Context(), callers, cookieProcedure, ""))
	assert.Empty(t, callers.asked)
}

func TestAProcedureThatTakesNoCookieIsNotNamedByOne(t *testing.T) {
	callers := &stubCallers{account: ada}

	assert.Empty(t, unaryCaller(t.Context(), callers, otherProcedure, "cp_sid=token-1"))
	assert.Empty(t, callers.asked)
}

func TestACookieNobodyOwnsOrAnAuthThatCannotAnswerLeavesNoAccount(t *testing.T) {
	for _, callers := range []*stubCallers{{account: cpsession.NoAccount}, {account: ada, err: errors.New("auth is down")}} {
		assert.Empty(t, unaryCaller(t.Context(), callers, cookieProcedure, "cp_sid=token-1"))
	}
}

func TestAStreamIsNamedByItsCookieToo(t *testing.T) {
	var account string
	next := connect.StreamingHandlerFunc(func(ctx context.Context, _ connect.StreamingHandlerConn) error {
		account = cpctx.GetAccount(ctx)
		return nil
	})
	conn := fakeConn{spec: connect.Spec{Procedure: cookieProcedure}, header: http.Header{cpconnect.CookieHeader: {"cp_sid=token-1"}}}

	require.NoError(t, cpconnect.NewCookieReaderInterceptor(&stubCallers{account: ada}, cookieProcedure).
		WrapStreamingHandler(next)(t.Context(), conn))

	assert.Equal(t, ada.String(), account)
}
