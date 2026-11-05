package get_titles_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	answer *progressionv1.GetTitlesResponse
	err    error
	asked  []accounts.AccountID
}

func (s *stubQuery) Titles(_ context.Context, account accounts.AccountID) (*progressionv1.GetTitlesResponse, error) {
	s.asked = append(s.asked, account)
	return s.answer, s.err
}

func TestTheCallersAnswerIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &progressionv1.GetTitlesResponse{Worn: &progressionv1.Title{Id: "og", Name: "OG"}}}

	res, err := get_titles_handler.New(query).GetTitles(cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&progressionv1.GetTitlesRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	account, err := accounts.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, []accounts.AccountID{account}, query.asked)
}

func TestACallerWithNoAccountIsUnauthenticatedAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	_, err := get_titles_handler.New(query).GetTitles(t.Context(), connect.NewRequest(&progressionv1.GetTitlesRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, query.asked)
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := get_titles_handler.New(&stubQuery{err: failure}).GetTitles(cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&progressionv1.GetTitlesRequest{}))

	assert.ErrorIs(t, err, failure)
}
