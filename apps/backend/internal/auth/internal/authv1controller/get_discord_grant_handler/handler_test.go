package get_discord_grant_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_discord_grant_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
)

const grantID = "019a4a1e-8c2d-7b3f-9e41-5d6c7a8b9c0d"

var account = grants.AccountID{15: 7}

type stubUseCase struct {
	grant grants.Grant
	err   error
	asked []grants.GrantID
}

func (s *stubUseCase) Execute(_ context.Context, id grants.GrantID) (grants.Grant, error) {
	s.asked = append(s.asked, id)
	return s.grant, s.err
}

func getDiscordGrant(t *testing.T, useCase *stubUseCase, id string) (*authv1.GetDiscordGrantResponse, error) {
	t.Helper()

	res, err := get_discord_grant_handler.New(useCase).
		GetDiscordGrant(t.Context(), connect.NewRequest(&authv1.GetDiscordGrantRequest{GrantId: id}))
	if err != nil {
		return nil, fmt.Errorf("GetDiscordGrant failed: %w", err)
	}
	return res.Msg, nil
}

func TestALiveGrantAnswersItsAccountUserAndToken(t *testing.T) {
	id, err := grants.GrantIDOf(grantID)
	require.NoError(t, err)
	grant, err := grants.NewGrant(id, account, "4242", "the-access-token", time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	useCase := &stubUseCase{grant: grant}

	res, err := getDiscordGrant(t, useCase, grantID)

	require.NoError(t, err)
	assert.True(t, proto.Equal(&authv1.GetDiscordGrantResponse{
		AccountId: account.String(), DiscordUserId: "4242", AccessToken: "the-access-token",
	}, res), "got %v", res)
	assert.Equal(t, []grants.GrantID{id}, useCase.asked)
}

func TestAGrantNobodyKeepsIsNotFound(t *testing.T) {
	for name, err := range map[string]error{
		"never kept": grants.ErrGrantNotFound,
		"expired":    fmt.Errorf("failed to check the grant: %w", grants.ErrGrantExpired),
	} {
		t.Run(name, func(t *testing.T) {
			_, got := getDiscordGrant(t, &stubUseCase{err: err}, grantID)

			assert.Equal(t, connect.CodeNotFound, connect.CodeOf(got))
		})
	}
}

func TestAnIDThatIsNoGrantIsNotFoundAndNobodyIsAsked(t *testing.T) {
	for _, id := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
		useCase := &stubUseCase{}

		_, err := getDiscordGrant(t, useCase, id)

		assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "%q", id)
		assert.Empty(t, useCase.asked, "%q", id)
	}
}

func TestAFailedReadIsNoAnswer(t *testing.T) {
	_, err := getDiscordGrant(t, &stubUseCase{err: errors.New("postgres is down")}, grantID)

	require.Error(t, err)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err), "the error net answers it")
}
