package unmute_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/unmute_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

type stubUseCase struct {
	account messages.AccountID
	until   time.Time
	err     error
}

func (s *stubUseCase) Execute(_ context.Context, account messages.AccountID) (time.Time, error) {
	s.account = account
	return s.until, s.err
}

const bullyID = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

func unmute(t *testing.T, useCase *stubUseCase, accountID string) (*chatv1.UnmuteResponse, error) {
	t.Helper()

	res, err := unmute_handler.New(useCase).Unmute(t.Context(), connect.NewRequest(&chatv1.UnmuteRequest{AccountId: accountID}))
	if err != nil {
		return nil, fmt.Errorf("unmute refused: %w", err)
	}
	return res.Msg, nil
}

func TestTheAccountReachesTheUseCaseAndWhenItsMuteWouldHaveEndedComesBack(t *testing.T) {
	until := time.Date(2026, 10, 30, 21, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{until: until}

	res, err := unmute(t, useCase, bullyID)
	require.NoError(t, err)

	assert.Equal(t, messages.AccountIDOf(bullyID), useCase.account)
	assert.Equal(t, until, res.GetMutedUntil().AsTime())
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		mutes.ErrNoAccount: connect.CodeInvalidArgument,
		fmt.Errorf("%w: %s", mutes.ErrNotMuted, bullyID): connect.CodeNotFound,
		assert.AnError: connect.CodeUnknown,
	} {
		_, got := unmute(t, &stubUseCase{err: err}, bullyID)
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
