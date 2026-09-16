package auth_accounts_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/adapters/secondary/auth_accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
)

type fakeInternal struct {
	authv1connect.UnimplementedInternalServiceHandler

	response *authv1.ResolveAccountResponse
	delay    time.Duration
	asked    []*authv1.ResolveAccountRequest
}

func (f *fakeInternal) ResolveAccount(
	ctx context.Context,
	req *connect.Request[authv1.ResolveAccountRequest],
) (*connect.Response[authv1.ResolveAccountResponse], error) {
	f.asked = append(f.asked, req.Msg)

	select {
	case <-time.After(f.delay):
	case <-ctx.Done():
		return nil, connect.NewError(connect.CodeDeadlineExceeded, context.Cause(ctx))
	}

	return connect.NewResponse(f.response), nil
}

func accountsOver(t *testing.T, fake *fakeInternal, timeout time.Duration) *auth_accounts.Accounts {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(authv1connect.NewInternalServiceHandler(fake))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return auth_accounts.New(server.Client(), server.URL, auth_accounts.Config{Enabled: true, Timeout: timeout})
}

func TestTheCookieAndTheAskReachTheAuthModule(t *testing.T) {
	account := uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")
	fake := &fakeInternal{response: &authv1.ResolveAccountResponse{AccountId: account.String(), SetCookie: "cp_sid=x"}}

	resolution, err := accountsOver(t, fake, time.Second).Resolve(t.Context(), "cp_sid=abc", true)
	require.NoError(t, err)

	assert.Equal(t, account, resolution.Account)
	assert.Equal(t, "cp_sid=x", resolution.SetCookie)
	require.Len(t, fake.asked, 1)
	assert.Equal(t, "cp_sid=abc", fake.asked[0].GetCookieHeader())
	assert.True(t, fake.asked[0].GetCreate())
}

func TestAnEmptyAccountIDIsNoAccount(t *testing.T) {
	fake := &fakeInternal{response: &authv1.ResolveAccountResponse{}}

	resolution, err := accountsOver(t, fake, time.Second).Resolve(t.Context(), "", false)

	require.ErrorIs(t, err, domain.ErrNoAccount)
	assert.Nil(t, resolution)
}

func TestAnAccountIDThatIsNotAUUIDIsAnError(t *testing.T) {
	fake := &fakeInternal{response: &authv1.ResolveAccountResponse{AccountId: "bob"}}

	_, err := accountsOver(t, fake, time.Second).Resolve(t.Context(), "", true)

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrNoAccount)
}

func TestASlowAuthModuleIsGivenUpOn(t *testing.T) {
	fake := &fakeInternal{response: &authv1.ResolveAccountResponse{}, delay: time.Second}

	started := time.Now()
	_, err := accountsOver(t, fake, 50*time.Millisecond).Resolve(t.Context(), "", true)

	require.Error(t, err)
	assert.Less(t, time.Since(started), 500*time.Millisecond)
}
