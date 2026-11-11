package open_book_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/open_book_handler"
)

type stubUseCase struct {
	command open_book_usecase.Command
	opened  bool
	err     error
}

func (s *stubUseCase) Execute(_ context.Context, command open_book_usecase.Command) (bool, error) {
	s.command = command
	return s.opened, s.err
}

var at = time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)

func open(t *testing.T, useCase *stubUseCase, req *scoringv1.OpenBookRequest) error {
	t.Helper()
	_, err := open_book_handler.New(useCase).OpenBook(t.Context(), connect.NewRequest(req))
	return err //nolint:wrapcheck // the test reads the code.
}

func TestABookOpenNowOrBeforeIsAnsweredOpen(t *testing.T) {
	for _, opened := range []bool{true, false} {
		useCase := &stubUseCase{opened: opened}

		require.NoError(t, open(t, useCase, &scoringv1.OpenBookRequest{BookId: "season-0", StartsAt: timestamppb.New(at)}))

		assert.Equal(t, open_book_usecase.Command{Book: "season-0", StartsAt: at}, useCase.command)
	}
}

func TestABadIDOrNoStartIsInvalid(t *testing.T) {
	for _, req := range []*scoringv1.OpenBookRequest{
		{StartsAt: timestamppb.New(at)},
		{BookId: string(make([]byte, 65)), StartsAt: timestamppb.New(at)},
		{BookId: "season-0"},
	} {
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(open(t, &stubUseCase{}, req)), "%v", req)
	}
}

func TestABookACloseMissedIsAFailedPreconditionAndAFailureIsLeftToTheErrorNet(t *testing.T) {
	req := &scoringv1.OpenBookRequest{BookId: "season-0", StartsAt: timestamppb.New(at)}
	err := open(t, &stubUseCase{err: fmt.Errorf("%w: book season-0: %w", books.ErrRefused, books.ErrMissed)}, req)
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

	down := errors.New("postgres is down")
	err = open(t, &stubUseCase{err: down}, req)
	require.ErrorIs(t, err, down)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
