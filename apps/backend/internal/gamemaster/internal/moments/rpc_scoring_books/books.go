package rpc_scoring_books

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/rpcrefusal"
)

type Scoring interface {
	OpenBook(ctx context.Context, req *connect.Request[scoringv1.OpenBookRequest],
	) (*connect.Response[scoringv1.OpenBookResponse], error)
	OpenRound(ctx context.Context, req *connect.Request[scoringv1.OpenRoundRequest],
	) (*connect.Response[scoringv1.OpenRoundResponse], error)
	CloseRound(ctx context.Context, req *connect.Request[scoringv1.CloseRoundRequest],
	) (*connect.Response[scoringv1.CloseRoundResponse], error)
	CloseBook(ctx context.Context, req *connect.Request[scoringv1.CloseBookRequest],
	) (*connect.Response[scoringv1.CloseBookResponse], error)
}

// A close reads the map to score a round: a slow scoring is answered at the next try.
const askTimeout = 10 * time.Second

func New(scoring Scoring) Books {
	return Books{scoring: scoring}
}

// Books asks scoring, in its words, over its internal service.
type Books struct {
	scoring Scoring
}

func (b Books) Ask(ctx context.Context, command moments.Command) (moments.Answer, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	answer, err := b.ask(ctx, command)
	if err != nil {
		subject := string(command.Book)
		if command.Round != "" {
			subject = string(command.Round)
		}
		return moments.Answer{}, rpcrefusal.Of(fmt.Errorf("failed to %s %s: %w", command.Action, subject, err)) //nolint:wrapcheck // wrapped here.
	}
	return answer, nil
}

func (b Books) ask(ctx context.Context, command moments.Command) (moments.Answer, error) {
	at := timestamppb.New(command.At)
	switch command.Action {
	case moments.OpenBook:
		_, err := b.scoring.OpenBook(ctx, connect.NewRequest(&scoringv1.OpenBookRequest{
			BookId: string(command.Book), StartsAt: at,
		}))
		return moments.Answer{}, err //nolint:wrapcheck // Ask names the command.
	case moments.OpenRound:
		ranking, err := rankingOf(command.Ranking)
		if err != nil {
			return moments.Answer{}, err
		}
		_, err = b.scoring.OpenRound(ctx, connect.NewRequest(&scoringv1.OpenRoundRequest{
			RoundId: string(command.Round), BookId: string(command.Book), Times: uint32(command.Times), StartsAt: at,
			FromNothing: command.FromNothing, Ranking: ranking,
		}))
		return moments.Answer{}, err //nolint:wrapcheck // Ask names the command.
	case moments.CloseRound:
		res, err := b.scoring.CloseRound(ctx, connect.NewRequest(&scoringv1.CloseRoundRequest{
			RoundId: string(command.Round), EndsAt: at,
		}))
		if err != nil {
			return moments.Answer{}, err //nolint:wrapcheck // Ask names the command.
		}
		return roundAnswerOf(res.Msg)
	case moments.CloseBook:
		res, err := b.scoring.CloseBook(ctx, connect.NewRequest(&scoringv1.CloseBookRequest{
			BookId: string(command.Book), EndsAt: at, TiebreakRoundId: string(command.Tiebreak),
		}))
		if err != nil {
			return moments.Answer{}, err //nolint:wrapcheck // Ask names the command.
		}
		return bookAnswerOf(res.Msg)
	}
	return moments.Answer{}, fmt.Errorf("scoring has no call to %s", command.Action)
}

func roundAnswerOf(res *scoringv1.CloseRoundResponse) (moments.Answer, error) {
	switch outcome := res.GetOutcome().(type) {
	case *scoringv1.CloseRoundResponse_ScoredAt:
		return moments.Answer{EndedAt: outcome.ScoredAt.AsTime()}, nil
	case *scoringv1.CloseRoundResponse_MissedAt:
		return moments.Answer{Missed: true, EndedAt: outcome.MissedAt.AsTime()}, nil
	case nil:
	}
	return moments.Answer{}, fmt.Errorf("scoring answered no outcome: %T", res.GetOutcome())
}

func bookAnswerOf(res *scoringv1.CloseBookResponse) (moments.Answer, error) {
	switch outcome := res.GetOutcome().(type) {
	case *scoringv1.CloseBookResponse_ClosedAt:
		return moments.Answer{EndedAt: outcome.ClosedAt.AsTime()}, nil
	case *scoringv1.CloseBookResponse_MissedAt:
		return moments.Answer{Missed: true, EndedAt: outcome.MissedAt.AsTime()}, nil
	case nil:
	}
	return moments.Answer{}, fmt.Errorf("scoring answered no outcome: %T", res.GetOutcome())
}

func rankingOf(ranking moments.Ranking) (scoringv1.Ranking, error) {
	switch ranking {
	case moments.RankedByGain:
		return scoringv1.Ranking_RANKING_GAIN, nil
	case moments.RankedByHeldAndGain:
		return scoringv1.Ranking_RANKING_HELD_AND_GAIN, nil
	}
	return scoringv1.Ranking_RANKING_UNSPECIFIED, fmt.Errorf("scoring has no ranking %q", ranking)
}
