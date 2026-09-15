package get_me_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/resolve_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_me_handler"
)

type stubUseCase struct {
	out   resolve_account_usecase.Out
	asked []resolve_account_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in resolve_account_usecase.In) (resolve_account_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, nil
}

var account = uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")

func getMe(useCase *stubUseCase) (*connect.Response[authv1.GetMeResponse], error) {
	req := connect.NewRequest(&authv1.GetMeRequest{})
	req.Header().Set("Cookie", "cp_sid=abc")
	return get_me_handler.New(useCase).GetMe(context.Background(), req) //nolint:wrapcheck // the test reads the connect code.
}

func TestGetMeAnswersTheAccountAndPassesTheCookieOn(t *testing.T) {
	useCase := &stubUseCase{out: resolve_account_usecase.Out{Account: account, SetCookie: "cp_sid=renewed"}}

	res, err := getMe(useCase)
	require.NoError(t, err)

	assert.Equal(t, account.String(), res.Msg.GetAccountId())
	assert.Equal(t, "cp_sid=renewed", res.Header().Get("Set-Cookie"))
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestGetMeNeverCreatesAnAccount(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := getMe(useCase)

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Equal(t, []resolve_account_usecase.In{{CookieHeader: "cp_sid=abc", Create: false}}, useCase.asked)
}
