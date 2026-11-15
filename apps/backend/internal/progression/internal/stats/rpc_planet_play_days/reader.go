package rpc_planet_play_days

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

type Planet interface {
	GetPlayDays(
		ctx context.Context,
		req *connect.Request[planetv1.GetPlayDaysRequest],
	) (*connect.Response[planetv1.GetPlayDaysResponse], error)
}

// A page is a grouped scan of its accounts' ledger rows: seconds, not milliseconds.
const askTimeout = time.Minute

func New(planet Planet) *Reader {
	return &Reader{planet: planet}
}

type Reader struct {
	planet Planet
}

// PlayDays is the days each account asked took a tile on; an account with none is left out.
func (r *Reader) PlayDays(ctx context.Context, asked []accounts.AccountID) (map[accounts.AccountID][]stats.Day, error) {
	played := make(map[accounts.AccountID][]stats.Day, len(asked))
	if len(asked) == 0 {
		return played, nil
	}
	ids := make([]string, len(asked))
	for i, account := range asked {
		ids[i] = account.String()
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()
	res, err := r.planet.GetPlayDays(ctx, connect.NewRequest(&planetv1.GetPlayDaysRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask planet the days each account took a tile on: %w", err)
	}

	for _, answered := range res.Msg.GetAccounts() {
		account, err := accounts.AccountIDOf(answered.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("planet answered %w", err)
		}
		days := make([]stats.Day, 0, len(answered.GetDays()))
		for _, value := range answered.GetDays() {
			day, err := stats.DayFrom(value)
			if err != nil {
				return nil, fmt.Errorf("planet answered %w", err)
			}
			days = append(days, day)
		}
		played[account] = days
	}
	return played, nil
}
