package rpc_scoring_rounds

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/pings"
)

type Scoring interface {
	GetRound(ctx context.Context, req *connect.Request[scoringv1.GetRoundRequest]) (*connect.Response[scoringv1.GetRoundResponse], error)
}

// GetRound reads the round's players and the book's table around it: more than one row.
const askTimeout = 5 * time.Second

// A round that runs answers its live standings, which no day scored.
var ErrNotScored = errors.New("scoring has not scored the round")

func New(scoring Scoring) *Rounds {
	return &Rounds{scoring: scoring}
}

// From scoring.v1.InternalService/GetRound: a scored round's results, and the book's scores after it.
type Rounds struct {
	scoring Scoring
}

func (r *Rounds) RankingAfter(ctx context.Context, round pings.RoundID) (pings.Ranking, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.scoring.GetRound(ctx, connect.NewRequest(&scoringv1.GetRoundRequest{RoundId: string(round)}))
	if err != nil {
		return pings.Ranking{}, fmt.Errorf("failed to ask scoring for the round %s: %w", round, err)
	}
	scored := res.Msg.GetRound()
	if scored.GetEndedAtUnixMs() == 0 {
		return pings.Ranking{}, fmt.Errorf("%w: %s", ErrNotScored, round)
	}

	won, err := placesOf(scored.GetStandings())
	if err != nil {
		return pings.Ranking{}, fmt.Errorf("the results of %s: %w", round, err)
	}
	table, err := placesOf(scored.GetAfter())
	if err != nil {
		return pings.Ranking{}, fmt.Errorf("the table after %s: %w", round, err)
	}
	return pings.Ranking{Won: won, Table: table}, nil
}

// A round's result and a book's score are both a place.
type placed interface {
	GetRank() uint32
	GetCountryId() string
	GetPoints() uint32
}

func placesOf[P placed](rows []P) ([]pings.Place, error) {
	places := make([]pings.Place, 0, len(rows))
	for _, row := range rows {
		flag, err := nations.FlagOf(row.GetCountryId())
		if err != nil {
			return nil, err //nolint:wrapcheck // the caller names the list.
		}
		places = append(places, pings.Place{Flag: flag, Rank: row.GetRank(), Points: row.GetPoints()})
	}
	return places, nil
}
