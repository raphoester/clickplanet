package open_round_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_round_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/open_round_handler"
)

type stubUseCase struct {
	command open_round_usecase.Command
	err     error
}

func (s *stubUseCase) Execute(_ context.Context, command open_round_usecase.Command) (bool, error) {
	s.command = command
	return true, s.err
}

var at = time.Date(2026, 11, 8, 21, 0, 0, 0, time.UTC)

func request() *scoringv1.OpenRoundRequest {
	return &scoringv1.OpenRoundRequest{
		RoundId: "season-1/day-1", BookId: "season-1", Times: 2, StartsAt: timestamppb.New(at),
		FromNothing: true, Ranking: scoringv1.Ranking_RANKING_HELD_AND_GAIN,
	}
}

func open(t *testing.T, useCase *stubUseCase, req *scoringv1.OpenRoundRequest) error {
	t.Helper()
	_, err := open_round_handler.New(useCase).OpenRound(t.Context(), connect.NewRequest(req))
	return err //nolint:wrapcheck // the test reads the code.
}

func TestTheRoundAskedReachesTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, open(t, useCase, request()))

	assert.Equal(t, open_round_usecase.Command{
		Round: "season-1/day-1", Book: "season-1", Times: 2, Ranking: books.HeldAndGainRanking{}, StartsAt: at, FromNothing: true,
	}, useCase.command)
	gain := request()
	gain.Ranking = scoringv1.Ranking_RANKING_GAIN
	require.NoError(t, open(t, useCase, gain))
	assert.Equal(t, books.GainRanking{}, useCase.command.Ranking)
}

func TestABadIDNoStartOrNoRankingIsInvalid(t *testing.T) {
	for _, change := range []func(*scoringv1.OpenRoundRequest){
		func(req *scoringv1.OpenRoundRequest) { req.RoundId = "" },
		func(req *scoringv1.OpenRoundRequest) { req.BookId = string(make([]byte, 65)) },
		func(req *scoringv1.OpenRoundRequest) { req.StartsAt = nil },
		func(req *scoringv1.OpenRoundRequest) { req.Ranking = scoringv1.Ranking_RANKING_UNSPECIFIED },
		func(req *scoringv1.OpenRoundRequest) { req.Ranking = scoringv1.Ranking(99) },
	} {
		req := request()
		change(req)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(open(t, &stubUseCase{}, req)), "%v", req)
	}
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for err, code := range map[error]connect.Code{
		fmt.Errorf("round %q: %w", "season-1/day-1", books.ErrInvalidTimes):                          connect.CodeInvalidArgument,
		fmt.Errorf("%w: round x: %w: season-1", books.ErrRefused, open_round_usecase.ErrBookNotOpen): connect.CodeFailedPrecondition,
		fmt.Errorf("%w: round x: %w", books.ErrRefused, books.ErrMissed):                             connect.CodeFailedPrecondition,
	} {
		assert.Equal(t, code, connect.CodeOf(open(t, &stubUseCase{err: err}, request())), err.Error())
	}

	down := errors.New("planet is down")
	err := open(t, &stubUseCase{err: down}, request())
	require.ErrorIs(t, err, down)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
