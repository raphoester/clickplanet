package rpc_scoring_results

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
)

type Scoring interface {
	GetRound(ctx context.Context, req *connect.Request[scoringv1.GetRoundRequest]) (*connect.Response[scoringv1.GetRoundResponse], error)
	GetTable(ctx context.Context, req *connect.Request[scoringv1.GetTableRequest]) (*connect.Response[scoringv1.GetTableResponse], error)
}

const askTimeout = time.Second

var errBookOpen = errors.New("the book is not closed yet")

func New(scoring Scoring) *Results {
	return &Results{scoring: scoring}
}

type Results struct {
	scoring Scoring
}

func (r *Results) Round(ctx context.Context, round announcements.RoundID) ([]announcements.Place, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.scoring.GetRound(ctx, connect.NewRequest(&scoringv1.GetRoundRequest{RoundId: round.String()}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask scoring for a round: %w", err)
	}

	standings := res.Msg.GetRound().GetStandings()
	places := make([]announcements.Place, 0, len(standings))
	for _, standing := range standings {
		places = append(places, announcements.PlaceOf(standing.GetCountryId(), standing.GetRank(), standing.GetPoints()))
	}
	return places, nil
}

func (r *Results) Table(ctx context.Context, book announcements.BookID) ([]announcements.Place, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.scoring.GetTable(ctx, connect.NewRequest(&scoringv1.GetTableRequest{BookId: book.String()}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask scoring for a table: %w", err)
	}

	if !res.Msg.GetClosed() {
		return nil, fmt.Errorf("%w: %q", errBookOpen, book)
	}

	table := res.Msg.GetTable()
	places := make([]announcements.Place, 0, len(table))
	for _, place := range table {
		places = append(places, announcements.PlaceOf(place.GetCountryId(), place.GetRank(), place.GetPoints()))
	}
	return places, nil
}
