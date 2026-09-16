package get_me_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
)

type stubUseCase struct {
	account uuid.UUID
	err     error
	asked   []string
}

func (s *stubUseCase) Execute(_ context.Context, cookieHeader string) (uuid.UUID, error) {
	s.asked = append(s.asked, cookieHeader)
	return s.account, s.err
}

func getMe(useCase *stubUseCase) (*connect.Response[authv1.GetMeResponse], error) {
	req := connect.NewRequest(&authv1.GetMeRequest{})
	req.Header().Set("Cookie", "cp_sid=abc")

	res, err := get_me_handler.New(useCase).GetMe(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("GetMe failed: %w", err)
	}
	return res, nil
}

func TestGetMeAnswersTheAccountOfTheCookieAndIsNeverCached(t *testing.T) {
	useCase := &stubUseCase{account: uuid.UUID{15: 1}}

	res, err := getMe(useCase)
	require.NoError(t, err)

	assert.Equal(t, []string{"cp_sid=abc"}, useCase.asked)
	assert.Equal(t, uuid.UUID{15: 1}.String(), res.Msg.GetAccountId())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestNoAccountIsUnauthenticated(t *testing.T) {
	_, err := getMe(&stubUseCase{err: accounts.ErrNoAccount})

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAFailureIsNotUnauthenticated(t *testing.T) {
	_, err := getMe(&stubUseCase{err: errors.New("postgres is down")})

	require.Error(t, err)
	assert.NotEqual(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
