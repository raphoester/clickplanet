package close_book_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/close_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/close_book_handler"
)

type stubUseCase struct {
	command close_book_usecase.Command
	ending  books.Ending
	err     error
}

func (s *stubUseCase) Execute(_ context.Context, command close_book_usecase.Command) (books.Ending, error) {
	s.command = command
	return s.ending, s.err
}

const battle = "0192f5a0-7c00-7000-8000-000000000031"

var (
	at     = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	before = at.Add(-time.Hour)
)

func closeBook(t *testing.T, useCase *stubUseCase, req *scoringv1.CloseBookRequest) (*scoringv1.CloseBookResponse, error) {
	t.Helper()
	res, err := close_book_handler.New(useCase).CloseBook(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, err //nolint:wrapcheck // the test reads the code.
	}
	return res.Msg, nil
}

func TestABookClosedIsAnsweredWithTheEndItClosedAt(t *testing.T) {
	useCase := &stubUseCase{ending: books.Ending{At: before}}

	res, err := closeBook(t, useCase, &scoringv1.CloseBookRequest{BookId: "season-0", EndsAt: timestamppb.New(at), TiebreakRoundId: battle})

	require.NoError(t, err)
	assert.Equal(t, close_book_usecase.Command{Book: "season-0", EndsAt: at, Tiebreak: battle}, useCase.command)
	assert.Equal(t, before, res.GetClosedAt().AsTime())
	assert.Nil(t, res.GetMissedAt())

	_, err = closeBook(t, useCase, &scoringv1.CloseBookRequest{BookId: "season-0", EndsAt: timestamppb.New(at)})
	require.NoError(t, err)
	assert.Equal(t, close_book_usecase.Command{Book: "season-0", EndsAt: at}, useCase.command, "no tiebreak")
}

func TestABookMissedIsAnsweredWithTheEndOfTheCloseThatMissedIt(t *testing.T) {
	res, err := closeBook(t, &stubUseCase{ending: books.Ending{At: before, Missed: true}},
		&scoringv1.CloseBookRequest{BookId: "season-0", EndsAt: timestamppb.New(at)})

	require.NoError(t, err)
	assert.Equal(t, before, res.GetMissedAt().AsTime())
	assert.Nil(t, res.GetClosedAt())
}

func TestABadIDOrNoEndIsInvalidAndAFailureIsLeftToTheErrorNet(t *testing.T) {
	for _, req := range []*scoringv1.CloseBookRequest{
		{EndsAt: timestamppb.New(at)},
		{BookId: "season-0"},
		{BookId: "season-0", EndsAt: timestamppb.New(at), TiebreakRoundId: string(make([]byte, 65))},
	} {
		_, err := closeBook(t, &stubUseCase{}, req)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%v", req)
	}

	down := errors.New("postgres is down")
	_, err := closeBook(t, &stubUseCase{err: down}, &scoringv1.CloseBookRequest{BookId: "season-0", EndsAt: timestamppb.New(at)})
	require.ErrorIs(t, err, down)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
