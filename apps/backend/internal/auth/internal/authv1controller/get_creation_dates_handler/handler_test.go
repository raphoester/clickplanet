package get_creation_dates_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_creation_dates_handler"
)

const accountID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubUseCase struct {
	dates map[accounts.AccountID]time.Time
	err   error
	asked []accounts.AccountID
}

func (s *stubUseCase) Execute(_ context.Context, asked []accounts.AccountID) (map[accounts.AccountID]time.Time, error) {
	s.asked = asked
	return s.dates, s.err
}

func getCreationDates(t *testing.T, useCase *stubUseCase, ids ...string) (*connect.Response[authv1.GetCreationDatesResponse], error) {
	t.Helper()

	return get_creation_dates_handler.New(useCase).GetCreationDates(t.Context(), //nolint:wrapcheck // the tests read the connect error.
		connect.NewRequest(&authv1.GetCreationDatesRequest{AccountIds: ids}))
}

func TestEachKnownAccountIsAnsweredWithItsDate(t *testing.T) {
	account, err := accounts.AccountIDOf(accountID)
	require.NoError(t, err)
	createdAt := time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC)
	useCase := &stubUseCase{dates: map[accounts.AccountID]time.Time{account: createdAt}}

	res, err := getCreationDates(t, useCase, accountID)

	require.NoError(t, err)
	assert.Equal(t, []accounts.AccountID{account}, useCase.asked)
	require.Len(t, res.Msg.GetDates(), 1)
	assert.Equal(t, accountID, res.Msg.GetDates()[0].GetAccountId())
	assert.Equal(t, createdAt.UnixMilli(), res.Msg.GetDates()[0].GetCreatedAtUnixMs())
}

func TestAnIDThatIsNotAnAccountIsInvalid(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := getCreationDates(t, useCase, accountID, "not-an-id")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Nil(t, useCase.asked, "nothing is read")
}

func TestAStoreFailureIsNotTheCallersFault(t *testing.T) {
	_, err := getCreationDates(t, &stubUseCase{err: errors.New("postgres is down")}, accountID)

	require.Error(t, err)
	assert.NotEqual(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
