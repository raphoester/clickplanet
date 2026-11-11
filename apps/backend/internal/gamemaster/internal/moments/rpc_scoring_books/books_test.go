package rpc_scoring_books_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/rpc_scoring_books"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

const entry = "0192f5a0-7c00-7000-8000-000000000031"

type scoring struct {
	asked  []proto.Message
	round  *scoringv1.CloseRoundResponse
	book   *scoringv1.CloseBookResponse
	failed error
}

func (s *scoring) OpenBook(_ context.Context, req *connect.Request[scoringv1.OpenBookRequest],
) (*connect.Response[scoringv1.OpenBookResponse], error) {
	s.asked = append(s.asked, req.Msg)
	return connect.NewResponse(&scoringv1.OpenBookResponse{}), s.failed
}

func (s *scoring) OpenRound(_ context.Context, req *connect.Request[scoringv1.OpenRoundRequest],
) (*connect.Response[scoringv1.OpenRoundResponse], error) {
	s.asked = append(s.asked, req.Msg)
	return connect.NewResponse(&scoringv1.OpenRoundResponse{}), s.failed
}

func (s *scoring) CloseRound(_ context.Context, req *connect.Request[scoringv1.CloseRoundRequest],
) (*connect.Response[scoringv1.CloseRoundResponse], error) {
	s.asked = append(s.asked, req.Msg)
	return connect.NewResponse(s.round), s.failed
}

func (s *scoring) CloseBook(_ context.Context, req *connect.Request[scoringv1.CloseBookRequest],
) (*connect.Response[scoringv1.CloseBookResponse], error) {
	s.asked = append(s.asked, req.Msg)
	return connect.NewResponse(s.book), s.failed
}

func TestEachCommandIsOneCallInScoringsWords(t *testing.T) {
	at := timestamppb.New(finale)
	for _, each := range []struct {
		command moments.Command
		wire    proto.Message
	}{
		{
			moments.Command{Action: moments.OpenBook, Book: "season-0", At: finale},
			&scoringv1.OpenBookRequest{BookId: "season-0", StartsAt: at},
		},
		{
			moments.Command{Action: moments.OpenRound, Book: "season-0", Round: entry, Times: 2, At: finale, FromNothing: true, Ranking: moments.RankedByGain},
			&scoringv1.OpenRoundRequest{
				RoundId: entry, BookId: "season-0", Times: 2, StartsAt: at, FromNothing: true, Ranking: scoringv1.Ranking_RANKING_GAIN,
			},
		},
		{
			moments.Command{Action: moments.OpenRound, Book: "season-0", Round: "season-0/day-24", Times: 1, At: finale, Ranking: moments.RankedByHeldAndGain},
			&scoringv1.OpenRoundRequest{
				RoundId: "season-0/day-24", BookId: "season-0", Times: 1, StartsAt: at, Ranking: scoringv1.Ranking_RANKING_HELD_AND_GAIN,
			},
		},
		{
			moments.Command{Action: moments.CloseRound, Book: "season-0", Round: "season-0/day-24", At: finale},
			&scoringv1.CloseRoundRequest{RoundId: "season-0/day-24", EndsAt: at},
		},
		{
			moments.Command{Action: moments.CloseBook, Book: "season-0", Tiebreak: entry, At: finale},
			&scoringv1.CloseBookRequest{BookId: "season-0", EndsAt: at, TiebreakRoundId: entry},
		},
	} {
		asked := &scoring{
			round: &scoringv1.CloseRoundResponse{Outcome: &scoringv1.CloseRoundResponse_ScoredAt{ScoredAt: at}},
			book:  &scoringv1.CloseBookResponse{Outcome: &scoringv1.CloseBookResponse_ClosedAt{ClosedAt: at}},
		}

		_, err := rpc_scoring_books.New(asked).Ask(t.Context(), each.command)

		require.NoError(t, err)
		require.Len(t, asked.asked, 1)
		assert.True(t, proto.Equal(each.wire, asked.asked[0]), "%v", asked.asked[0])
	}
}

func TestACloseAnswersTheEndScoringScoredOrMissedAt(t *testing.T) {
	earlier := finale.Add(-time.Hour)
	asked := &scoring{
		round: &scoringv1.CloseRoundResponse{Outcome: &scoringv1.CloseRoundResponse_MissedAt{MissedAt: timestamppb.New(finale)}},
		book:  &scoringv1.CloseBookResponse{Outcome: &scoringv1.CloseBookResponse_ClosedAt{ClosedAt: timestamppb.New(earlier)}},
	}
	books := rpc_scoring_books.New(asked)

	round, err := books.Ask(t.Context(), moments.Command{Action: moments.CloseRound, Round: "season-0/day-24", At: finale})
	require.NoError(t, err)
	book, err := books.Ask(t.Context(), moments.Command{Action: moments.CloseBook, Book: "season-0", At: finale})
	require.NoError(t, err)

	assert.Equal(t, moments.Answer{Missed: true, EndedAt: finale}, round)
	assert.Equal(t, moments.Answer{EndedAt: earlier}, book, "a book closed before answers the end it closed at")
}

func TestARefusalIsForGoodAndAnythingElseIsNoAnswer(t *testing.T) {
	for code, refused := range map[connect.Code]bool{
		connect.CodeFailedPrecondition: true, connect.CodeInvalidArgument: true, connect.CodeUnavailable: false,
		connect.CodeDeadlineExceeded: false, connect.CodeInternal: false,
	} {
		_, err := rpc_scoring_books.New(&scoring{failed: connect.NewError(code, errors.New("no"))}).
			Ask(t.Context(), moments.Command{Action: moments.OpenBook, Book: "season-0", At: finale})

		require.Error(t, err)
		assert.Equal(t, refused, errors.Is(err, moments.ErrRefused), "%s", code)
	}
}

func TestACloseWithNoOutcomeIsNoAnswer(t *testing.T) {
	_, err := rpc_scoring_books.New(&scoring{round: &scoringv1.CloseRoundResponse{}}).
		Ask(t.Context(), moments.Command{Action: moments.CloseRound, Round: "season-0/day-24", At: finale})

	require.Error(t, err)
	assert.NotErrorIs(t, err, moments.ErrRefused)
}
