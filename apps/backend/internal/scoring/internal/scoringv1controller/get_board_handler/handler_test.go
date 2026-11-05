package get_board_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler/board_query"
)

type stubQuery struct {
	board *scoringv1.GetBoardResponse
	err   error
	asked *[]board_query.View
}

func (s stubQuery) Board(_ context.Context, view board_query.View) (*scoringv1.GetBoardResponse, error) {
	*s.asked = append(*s.asked, view)
	return s.board, s.err
}

func TestGetBoardAnswersWhatTheQueryReadsForTheBookAndCountryAsked(t *testing.T) {
	board := &scoringv1.GetBoardResponse{Standings: []*scoringv1.Standing{{Rank: 1, Name: "Ada", CountryId: "fr", Tiles: 3}}}
	var asked []board_query.View

	res, err := get_board_handler.New(stubQuery{board: board, asked: &asked}).
		GetBoard(t.Context(), connect.NewRequest(&scoringv1.GetBoardRequest{BookId: "season-0", CountryId: "fr"}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(board, res.Msg))
	assert.Equal(t, []board_query.View{{Book: "season-0", Country: "fr"}}, asked)
}

func TestABookOrACountryThatIsNotOneIsInvalidArgument(t *testing.T) {
	var asked []board_query.View
	_, err := get_board_handler.New(stubQuery{
		err: fmt.Errorf("%w: %q", board_query.ErrUnknownCountry, "zz"), asked: &asked,
	}).GetBoard(t.Context(), connect.NewRequest(&scoringv1.GetBoardRequest{BookId: "season-0", CountryId: "zz"}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = get_board_handler.New(stubQuery{asked: &asked}).GetBoard(t.Context(), connect.NewRequest(&scoringv1.GetBoardRequest{}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Len(t, asked, 1, "no book is read for no id")
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	refused := errors.New("the database is down")

	_, err := get_board_handler.New(stubQuery{err: refused, asked: new([]board_query.View)}).
		GetBoard(t.Context(), connect.NewRequest(&scoringv1.GetBoardRequest{BookId: "season-0"}))

	require.ErrorIs(t, err, refused)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
