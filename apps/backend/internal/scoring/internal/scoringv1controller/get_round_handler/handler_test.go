package get_round_handler_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_round_handler/round_query"
)

type stubQuery struct {
	round *scoringv1.Round
	err   error
}

func (s stubQuery) Round(context.Context, books.RoundID) (*scoringv1.Round, error) {
	return s.round, s.err
}

func TestARoundIsAnsweredByItsID(t *testing.T) {
	round := &scoringv1.Round{RoundId: "season-0/day-24", BookId: "season-0", Times: 1}

	res, err := get_round_handler.New(stubQuery{round: round}).
		GetRound(t.Context(), connect.NewRequest(&scoringv1.GetRoundRequest{RoundId: "season-0/day-24"}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(round, res.Msg.GetRound()))
}

func TestARoundNeverOpenedIsNotFoundAndNoIDIsInvalid(t *testing.T) {
	_, err := get_round_handler.New(stubQuery{err: fmt.Errorf("%w: %q", round_query.ErrNoRound, "x")}).
		GetRound(t.Context(), connect.NewRequest(&scoringv1.GetRoundRequest{RoundId: "x"}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))

	_, err = get_round_handler.New(stubQuery{}).GetRound(t.Context(), connect.NewRequest(&scoringv1.GetRoundRequest{}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
