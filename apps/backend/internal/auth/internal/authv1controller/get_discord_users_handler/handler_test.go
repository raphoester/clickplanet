package get_discord_users_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_discord_users_handler"
)

const accountID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	found *authv1.GetDiscordUsersResponse
	err   error
	after *accounts.AccountID
	limit int
}

func (s *stubQuery) DiscordUsers(
	_ context.Context, after accounts.AccountID, limit int,
) (*authv1.GetDiscordUsersResponse, error) {
	s.after, s.limit = &after, limit
	return s.found, s.err
}

func getDiscordUsers(
	t *testing.T, query *stubQuery, req *authv1.GetDiscordUsersRequest,
) (*connect.Response[authv1.GetDiscordUsersResponse], error) {
	t.Helper()

	return get_discord_users_handler.New(query).GetDiscordUsers(t.Context(), //nolint:wrapcheck // the tests read the connect error.
		connect.NewRequest(req))
}

func TestThePageIsWhatTheQueryReadAfterTheAccount(t *testing.T) {
	account, err := accounts.AccountIDOf(accountID)
	require.NoError(t, err)
	found := &authv1.GetDiscordUsersResponse{Users: []*authv1.DiscordUser{
		{AccountId: "1c6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", DiscordUserId: "4242"},
	}}
	query := &stubQuery{found: found}

	res, err := getDiscordUsers(t, query, &authv1.GetDiscordUsersRequest{AfterAccountId: accountID, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, account, *query.after)
	assert.Equal(t, 20, query.limit)
	assert.True(t, proto.Equal(found, res.Msg), "got %v", res.Msg)
}

func TestNoAccountAsksFromTheFirstAndNoLimitIsTheMost(t *testing.T) {
	query := &stubQuery{found: &authv1.GetDiscordUsersResponse{}}

	_, err := getDiscordUsers(t, query, &authv1.GetDiscordUsersRequest{})

	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{}, *query.after)
	assert.Equal(t, get_discord_users_handler.MaxLimit, query.limit)
}

func TestALimitAboveTheMostIsTheMost(t *testing.T) {
	query := &stubQuery{found: &authv1.GetDiscordUsersResponse{}}

	_, err := getDiscordUsers(t, query, &authv1.GetDiscordUsersRequest{Limit: 100_000})

	require.NoError(t, err)
	assert.Equal(t, get_discord_users_handler.MaxLimit, query.limit)
}

func TestAnAfterThatIsNotAnAccountIsInvalid(t *testing.T) {
	query := &stubQuery{}

	_, err := getDiscordUsers(t, query, &authv1.GetDiscordUsersRequest{AfterAccountId: "not-an-id"})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Nil(t, query.after, "nothing is read")
}

func TestAFailedReadIsNotTheCallersFault(t *testing.T) {
	_, err := getDiscordUsers(t, &stubQuery{err: errors.New("postgres is down")}, &authv1.GetDiscordUsersRequest{})

	require.Error(t, err)
	assert.NotEqual(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
