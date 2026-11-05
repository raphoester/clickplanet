package rpc_planet_ledger

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts/usecases/make_cast_usecase"
)

type Planet interface {
	GetLedger(
		ctx context.Context,
		req *connect.Request[planetv1.GetLedgerRequest],
	) (*connect.Response[planetv1.GetLedgerResponse], error)
}

// A page of 20,000 events reads in under a second; 12 hours are about two.
const pageTimeout = 30 * time.Second

func New(planet Planet) *Ledger {
	return &Ledger{planet: planet}
}

type Ledger struct {
	planet Planet
}

var _ make_cast_usecase.Ledger = (*Ledger)(nil)

// Every change of the window an account made, page after page.
func (l *Ledger) Takes(ctx context.Context, window casts.Window) ([]casts.Take, error) {
	var (
		takes []casts.Take
		page  string
	)
	for {
		answer, err := l.page(ctx, window, page)
		if err != nil {
			return nil, err
		}
		for _, change := range answer.GetChanges() {
			if change.GetAccountId() == "" {
				continue
			}
			account, err := casts.AccountIDOf(change.GetAccountId())
			if err != nil {
				return nil, fmt.Errorf("the planet module answered a change by %q: %w", change.GetAccountId(), err)
			}
			update := change.GetUpdate()
			takes = append(takes, casts.Take{
				At: change.GetAt().AsTime(), Tile: update.GetTileId(), Country: update.GetCountryId(),
				Previous: update.GetPreviousCountryId(), Account: account,
			})
		}
		if page = answer.GetNextPage(); page == "" {
			return takes, nil
		}
	}
}

func (l *Ledger) page(ctx context.Context, window casts.Window, page string) (*planetv1.GetLedgerResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, pageTimeout)
	defer cancel()

	answer, err := l.planet.GetLedger(ctx, connect.NewRequest(&planetv1.GetLedgerRequest{
		Since: timestamppb.New(window.Since), Until: timestamppb.New(window.Until), Page: page,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask the planet module for its ledger: %w", err)
	}
	return answer.Msg, nil
}
