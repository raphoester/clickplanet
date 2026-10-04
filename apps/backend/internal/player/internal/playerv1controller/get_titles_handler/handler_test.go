package get_titles_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	answer *playerv1.GetTitlesResponse
	err    error
	asked  []players.AccountID
}

func (s *stubQuery) Titles(_ context.Context, account players.AccountID) (*playerv1.GetTitlesResponse, error) {
	s.asked = append(s.asked, account)
	return s.answer, s.err
}

func TestTheCallersAnswerIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetTitlesResponse{Worn: &playerv1.Title{Id: "og", Name: "OG"}}}

	res, err := get_titles_handler.New(query).GetTitles(cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&playerv1.GetTitlesRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	account, err := players.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, []players.AccountID{account}, query.asked)
}

func TestACallerWithNoAccountIsUnauthenticatedAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	_, err := get_titles_handler.New(query).GetTitles(t.Context(), connect.NewRequest(&playerv1.GetTitlesRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, query.asked)
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := get_titles_handler.New(&stubQuery{err: failure}).GetTitles(cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&playerv1.GetTitlesRequest{}))

	assert.ErrorIs(t, err, failure)
}
