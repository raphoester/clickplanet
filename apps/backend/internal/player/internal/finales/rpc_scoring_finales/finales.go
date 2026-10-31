package rpc_scoring_finales

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Scoring interface {
	GetTable(ctx context.Context, req *connect.Request[scoringv1.GetTableRequest]) (*connect.Response[scoringv1.GetTableResponse], error)
	GetRoundPlayers(ctx context.Context, req *connect.Request[scoringv1.GetRoundPlayersRequest]) (*connect.Response[scoringv1.GetRoundPlayersResponse], error)
}

const askTimeout = 5 * time.Second

var errBookOpen = errors.New("the book is not closed yet")

func New(scoring Scoring) *Finales {
	return &Finales{scoring: scoring}
}

type Finales struct {
	scoring Scoring
}

func (f *Finales) Table(ctx context.Context, book finales.BookID) ([]finales.Place, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := f.scoring.GetTable(ctx, connect.NewRequest(&scoringv1.GetTableRequest{BookId: book.String()}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask scoring for a table: %w", err)
	}
	if !res.Msg.GetClosed() {
		return nil, fmt.Errorf("%w: %q", errBookOpen, book)
	}

	table := res.Msg.GetTable()
	places := make([]finales.Place, 0, len(table))
	for _, place := range table {
		places = append(places, finales.Place{Country: finales.Country(place.GetCountryId()), Rank: place.GetRank()})
	}
	return places, nil
}

func (f *Finales) Fighters(ctx context.Context, round finales.RoundID) ([]finales.Fighter, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := f.scoring.GetRoundPlayers(ctx, connect.NewRequest(&scoringv1.GetRoundPlayersRequest{RoundId: round.String()}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask scoring for the players of a round: %w", err)
	}

	answered := res.Msg.GetPlayers()
	fighters := make([]finales.Fighter, 0, len(answered))
	for _, player := range answered {
		account, err := players.AccountIDOf(player.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("scoring answered a player of round %q: %w", round, err)
		}
		fighters = append(fighters, finales.Fighter{Account: account, Country: finales.Country(player.GetCountryId())})
	}
	return fighters, nil
}
