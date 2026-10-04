package get_me_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler/me_query"
)

type stubQuery struct {
	me    *authv1.GetMeResponse
	err   error
	asked []string
}

func (s *stubQuery) Me(_ context.Context, cookieHeader string) (*authv1.GetMeResponse, error) {
	s.asked = append(s.asked, cookieHeader)
	return s.me, s.err
}

func getMe(query *stubQuery) (*connect.Response[authv1.GetMeResponse], error) {
	req := connect.NewRequest(&authv1.GetMeRequest{})
	req.Header().Set("Cookie", "cp_sid=abc")

	res, err := get_me_handler.New(query).GetMe(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("GetMe failed: %w", err)
	}
	return res, nil
}

func TestGetMeAnswersWhatTheQueryReadForTheCookieAndIsNeverCached(t *testing.T) {
	me := &authv1.GetMeResponse{
		AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", Kind: authv1.AccountKind_ACCOUNT_KIND_LINKED,
		Providers: []authv1.Provider{authv1.Provider_PROVIDER_DISCORD, authv1.Provider_PROVIDER_GOOGLE},
	}
	query := &stubQuery{me: me}

	res, err := getMe(query)
	require.NoError(t, err)

	assert.Equal(t, []string{"cp_sid=abc"}, query.asked)
	assert.True(t, proto.Equal(me, res.Msg), "got %v", res.Msg)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestNoAccountIsUnauthenticated(t *testing.T) {
	_, err := getMe(&stubQuery{err: fmt.Errorf("%w: no cookie", me_query.ErrNoAccount)})

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAFailureIsNotUnauthenticated(t *testing.T) {
	for _, err := range []error{errors.New("postgres is down"), me_query.ErrUnknownProvider} {
		_, err := getMe(&stubQuery{err: err})

		require.Error(t, err)
		assert.NotEqual(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	}
}
