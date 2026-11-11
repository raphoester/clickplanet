package close_round_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/close_round_handler"
)

type stubUseCase struct {
	command close_round_usecase.Command
	ending  books.Ending
	err     error
}

func (s *stubUseCase) Execute(_ context.Context, command close_round_usecase.Command) (books.Ending, error) {
	s.command = command
	return s.ending, s.err
}

var (
	at     = time.Date(2026, 11, 8, 21, 0, 0, 0, time.UTC)
	before = at.Add(-time.Hour)
)

func closeRound(t *testing.T, useCase *stubUseCase, req *scoringv1.CloseRoundRequest) (*scoringv1.CloseRoundResponse, error) {
	t.Helper()
	res, err := close_round_handler.New(useCase).CloseRound(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, err //nolint:wrapcheck // the test reads the code.
	}
	return res.Msg, nil
}

func TestARoundScoredIsAnsweredWithTheEndItWasScoredAt(t *testing.T) {
	useCase := &stubUseCase{ending: books.Ending{At: before}}

	res, err := closeRound(t, useCase, &scoringv1.CloseRoundRequest{RoundId: "season-1/day-1", EndsAt: timestamppb.New(at)})

	require.NoError(t, err)
	assert.Equal(t, close_round_usecase.Command{Round: "season-1/day-1", EndsAt: at}, useCase.command)
	assert.Equal(t, before, res.GetScoredAt().AsTime())
	assert.Nil(t, res.GetMissedAt())
}

func TestARoundMissedIsAnsweredWithTheEndOfTheCloseThatMissedIt(t *testing.T) {
	res, err := closeRound(t, &stubUseCase{ending: books.Ending{At: before, Missed: true}},
		&scoringv1.CloseRoundRequest{RoundId: "season-1/day-1", EndsAt: timestamppb.New(at)})

	require.NoError(t, err)
	assert.Equal(t, before, res.GetMissedAt().AsTime())
	assert.Nil(t, res.GetScoredAt())
}

func TestABadIDOrNoEndIsInvalidAndAFailureIsLeftToTheErrorNet(t *testing.T) {
	for _, req := range []*scoringv1.CloseRoundRequest{{EndsAt: timestamppb.New(at)}, {RoundId: "season-1/day-1"}} {
		_, err := closeRound(t, &stubUseCase{}, req)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%v", req)
	}

	down := errors.New("planet is down")
	_, err := closeRound(t, &stubUseCase{err: down}, &scoringv1.CloseRoundRequest{RoundId: "season-1/day-1", EndsAt: timestamppb.New(at)})
	require.ErrorIs(t, err, down)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
