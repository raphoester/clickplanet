package get_table_handler_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_scores_handler/scores_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_table_handler"
)

type stubQuery struct {
	table *scoringv1.GetTableResponse
	err   error
}

func (s stubQuery) Table(context.Context, books.BookID) (*scoringv1.GetTableResponse, error) {
	return s.table, s.err
}

func TestTheTableOfTheBookAskedIsAnswered(t *testing.T) {
	res, err := get_table_handler.New(stubQuery{table: &scoringv1.GetTableResponse{Closed: true}}).
		GetTable(t.Context(), connect.NewRequest(&scoringv1.GetTableRequest{BookId: "season-0"}))

	require.NoError(t, err)
	assert.True(t, res.Msg.GetClosed())
}

func TestABookNeverOpenedIsNotFoundAndNoIDIsInvalid(t *testing.T) {
	_, err := get_table_handler.New(stubQuery{err: fmt.Errorf("%w: %q", scores_query.ErrNoBook, "season-9")}).
		GetTable(t.Context(), connect.NewRequest(&scoringv1.GetTableRequest{BookId: "season-9"}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = get_table_handler.New(stubQuery{}).GetTable(t.Context(), connect.NewRequest(&scoringv1.GetTableRequest{}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
