package rpc_scoring_podiums

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums"
)

type Scoring interface {
	GetBoard(ctx context.Context, req *connect.Request[scoringv1.GetBoardRequest]) (*connect.Response[scoringv1.GetBoardResponse], error)
}

const askTimeout = 5 * time.Second

func New(scoring Scoring) *Podiums {
	return &Podiums{scoring: scoring}
}

type Podiums struct {
	scoring Scoring
}

func (p *Podiums) Board(ctx context.Context, book finales.BookID) ([]podiums.Entry, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := p.scoring.GetBoard(ctx, connect.NewRequest(&scoringv1.GetBoardRequest{BookId: book.String()}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask scoring for a book's board: %w", err)
	}

	standings := res.Msg.GetStandings()
	entries := make([]podiums.Entry, 0, len(standings))
	for _, standing := range standings {
		account, err := players.AccountIDOf(standing.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("scoring answered a player of book %q: %w", book, err)
		}
		entries = append(entries, podiums.Entry{Account: account, Rank: standing.GetRank()})
	}
	return entries, nil
}
