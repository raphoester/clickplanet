package complete_admin_sign_in_usecase_test

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin/usecases/complete_admin_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin/usecases/start_admin_sign_in_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin/aes_flow_sealer"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start = time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	ada   = accounts.AccountID{15: 1}
	bob   = accounts.AccountID{15: 2}
)

type store struct {
	candidates map[string]adminsignin.Candidate
	saved      []adminsignin.Snapshot
}

func (s *store) Candidate(_ context.Context, provider string, subject string) (adminsignin.Candidate, error) {
	candidate, found := s.candidates[provider+":"+subject]
	if !found {
		return adminsignin.Candidate{}, adminsignin.ErrUnknownIdentity
	}
	return candidate, nil
}

func (s *store) SaveSession(_ context.Context, session adminsignin.Snapshot) error {
	s.saved = append(s.saved, session)
	return nil
}

func (s *store) DeleteSession(context.Context, accounts.TokenHash) error {
	return nil
}

type rig struct {
	google   *signin.FakeProvider
	store    *store
	start    *start_admin_sign_in_usecase.UseCase
	complete *complete_admin_sign_in_usecase.UseCase
}

func newRig(t *testing.T) rig {
	t.Helper()

	sealer, err := aes_flow_sealer.New(bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	google := signin.NewFakeProvider(signin.Google)
	providers := signin.Providers{signin.Google: google}
	clock := cptime.NewFixedClock(start)
	kept := &store{candidates: map[string]adminsignin.Candidate{
		"google:ada-sub": adminsignin.CandidateOf(ada, true),
		"google:bob-sub": adminsignin.CandidateOf(bob, false),
	}}

	return rig{
		google: google,
		store:  kept,
		start:  start_admin_sign_in_usecase.New(providers, &signin.SequentialSecrets{}, sealer, clock),
		complete: complete_admin_sign_in_usecase.New(providers, sealer, kept,
			adminsignin.NewIssuer(&accounts.SequentialTokens{}, adminsignin.DefaultTTL), clock),
	}
}

func (r rig) started(t *testing.T) (string, string) {
	t.Helper()

	out, err := r.start.Execute(t.Context())
	require.NoError(t, err)
	cookie, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	require.Equal(t, adminsignin.FlowCookieName, cookie.Name)
	authorization, err := url.Parse(out.AuthorizationURL)
	require.NoError(t, err)

	return cookie.Name + "=" + cookie.Value, authorization.Query().Get("state")
}

func TestAnOperatorSignedInWithGoogleGetsAnAdminSession(t *testing.T) {
	r := newRig(t)
	flow, state := r.started(t)
	r.google.Grant("code", accounts.ClaimOf("ada-sub", "ada@example.com", true))

	out, err := r.complete.Execute(t.Context(), complete_admin_sign_in_usecase.In{Code: "code", State: state, CookieHeader: flow})

	require.NoError(t, err)
	assert.Equal(t, ada, out.Account)
	cookie, err := http.ParseSetCookie(out.SetCookie)
	require.NoError(t, err)
	assert.Equal(t, adminsignin.CookieName, cookie.Name)
	require.Len(t, r.store.saved, 1)
	assert.Equal(t, accounts.TokenOf(cookie.Value).Hash(), r.store.saved[0].TokenHash)
	assert.Equal(t, start.Add(adminsignin.DefaultTTL), r.store.saved[0].ExpiresAt)
}

func TestAGoogleAccountThatDoesNotOperateIsRefusedAndKeepsNothing(t *testing.T) {
	for _, subject := range []string{"bob-sub", "stranger-sub"} {
		r := newRig(t)
		flow, state := r.started(t)
		r.google.Grant("code", accounts.ClaimOf(subject, "", false))

		_, err := r.complete.Execute(t.Context(), complete_admin_sign_in_usecase.In{Code: "code", State: state, CookieHeader: flow})

		require.ErrorIs(t, err, adminsignin.ErrNotAnOperator, subject)
		assert.Empty(t, r.store.saved, subject)
	}
}

func TestAFlowThisBrowserDidNotStartIsRefused(t *testing.T) {
	r := newRig(t)
	flow, state := r.started(t)
	r.google.Grant("code", accounts.ClaimOf("ada-sub", "", false))

	for name, in := range map[string]complete_admin_sign_in_usecase.In{
		"no flow cookie":       {Code: "code", State: state, CookieHeader: ""},
		"the game's flow name": {Code: "code", State: state, CookieHeader: "cp_oauth" + flow[len(adminsignin.FlowCookieName):]},
		"another state":        {Code: "code", State: "forged", CookieHeader: flow},
	} {
		_, err := r.complete.Execute(t.Context(), in)
		require.ErrorIs(t, err, signin.ErrFlowInvalid, name)
	}
	assert.Empty(t, r.store.saved)
}

func TestACodeGoogleDidNotGrantIsRefused(t *testing.T) {
	r := newRig(t)
	flow, state := r.started(t)

	_, err := r.complete.Execute(t.Context(), complete_admin_sign_in_usecase.In{Code: "made-up", State: state, CookieHeader: flow})

	require.ErrorIs(t, err, signin.ErrProviderRefused)
}

func TestWithNoGoogleClientTheAdminSignInIsOff(t *testing.T) {
	sealer, err := aes_flow_sealer.New(bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	clock := cptime.NewFixedClock(start)

	_, err = start_admin_sign_in_usecase.New(signin.Providers{}, &signin.SequentialSecrets{}, sealer, clock).Execute(t.Context())
	require.ErrorIs(t, err, adminsignin.ErrSignInOff)

	_, err = complete_admin_sign_in_usecase.New(signin.Providers{}, sealer, &store{},
		adminsignin.NewIssuer(&accounts.SequentialTokens{}, adminsignin.DefaultTTL), clock).
		Execute(t.Context(), complete_admin_sign_in_usecase.In{})
	require.ErrorIs(t, err, adminsignin.ErrSignInOff)
}
