package get_progress_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_progress_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_progress_handler/progress_query"
)

type stubQuery struct {
	answer *progressionv1.GetProgressResponse
	err    error
	asked  []accounts.AccountID
}

func (s *stubQuery) Progress(_ context.Context, account accounts.AccountID) (*progressionv1.GetProgressResponse, error) {
	s.asked = append(s.asked, account)
	return s.answer, s.err
}

func getProgress(t *testing.T, query *stubQuery, id string) (*connect.Response[progressionv1.GetProgressResponse], error) {
	t.Helper()

	return get_progress_handler.New(query).GetProgress(t.Context(), connect.NewRequest(&progressionv1.GetProgressRequest{AccountId: id})) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestAPlayersProgressIsCachedBriefly(t *testing.T) {
	query := &stubQuery{answer: &progressionv1.GetProgressResponse{Progress: &progressionv1.Progress{}}}

	res, err := getProgress(t, query, accounts.AccountID{15: 1}.String())

	require.NoError(t, err)
	assert.Equal(t, "public, max-age=10", res.Header().Get("Cache-Control"))
	assert.Equal(t, []accounts.AccountID{{15: 1}}, query.asked)
}

func TestAnIdThatIsNotAnAccountIsRefusedAndReadsNothing(t *testing.T) {
	for _, id := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		query := &stubQuery{}

		_, err := getProgress(t, query, id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
		assert.Empty(t, query.asked, id)
	}
}

func TestAnAccountTheOthersDoNotSeeIsNotFound(t *testing.T) {
	_, err := getProgress(t, &stubQuery{err: progress_query.ErrNotShown}, accounts.AccountID{15: 1}.String())

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := getProgress(t, &stubQuery{err: failure}, accounts.AccountID{15: 1}.String())

	assert.ErrorIs(t, err, failure)
}
