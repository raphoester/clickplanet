package get_caller_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_caller_handler"
)

type stubQuery struct {
	answer *authv1.GetCallerResponse
	err    error
	asked  []string
}

func (s *stubQuery) Caller(_ context.Context, cookieHeader string) (*authv1.GetCallerResponse, error) {
	s.asked = append(s.asked, cookieHeader)
	return s.answer, s.err
}

func TestTheCookieGoesToTheQueryAndItsAnswerToTheCaller(t *testing.T) {
	query := &stubQuery{answer: &authv1.GetCallerResponse{AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"}}

	res, err := get_caller_handler.New(query).GetCaller(t.Context(),
		connect.NewRequest(&authv1.GetCallerRequest{Cookie: "cp_sid=token-1"}))

	require.NoError(t, err)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", res.Msg.GetAccountId())
	assert.Equal(t, []string{"cp_sid=token-1"}, query.asked)
}

func TestAQueryThatFailsIsLeftToTheErrorNet(t *testing.T) {
	_, err := get_caller_handler.New(&stubQuery{err: errors.New("postgres is down")}).GetCaller(t.Context(),
		connect.NewRequest(&authv1.GetCallerRequest{Cookie: "cp_sid=token-1"}))

	assert.ErrorContains(t, err, "postgres is down")
}
