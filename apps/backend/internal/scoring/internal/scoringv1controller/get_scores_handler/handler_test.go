package get_scores_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_scores_handler"
)

type stubQuery struct {
	asked *[]books.BookID
}

func (s stubQuery) Scores(_ context.Context, book books.BookID) (*scoringv1.Scores, error) {
	*s.asked = append(*s.asked, book)
	return &scoringv1.Scores{BookId: string(book)}, nil
}

func TestTheScoresOfTheBookAskedAreAnswered(t *testing.T) {
	var asked []books.BookID

	res, err := get_scores_handler.New(stubQuery{asked: &asked}).
		GetScores(t.Context(), connect.NewRequest(&scoringv1.GetScoresRequest{BookId: "season-0"}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&scoringv1.Scores{BookId: "season-0"}, res.Msg.GetScores()))
	assert.Equal(t, []books.BookID{"season-0"}, asked)
}

func TestNoBookIsInvalidArgument(t *testing.T) {
	var asked []books.BookID

	_, err := get_scores_handler.New(stubQuery{asked: &asked}).
		GetScores(t.Context(), connect.NewRequest(&scoringv1.GetScoresRequest{}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, asked)
}
