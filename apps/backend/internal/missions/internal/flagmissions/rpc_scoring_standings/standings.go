package rpc_scoring_standings

import (
	"context"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Scoring interface {
	GetTable(
		ctx context.Context,
		req *connect.Request[scoringv1.GetTableRequest],
	) (*connect.Response[scoringv1.GetTableResponse], error)
}

const (
	askTimeout = 5 * time.Second
	// While a round before the day is open, every read of a flag's missions and every take told again asks for a
	// draw: a table kept a second answers them all.
	keptFor = time.Second
)

func New(scoring Scoring, clock cptime.Clock) *Reader {
	return &Reader{scoring: scoring, clock: clock}
}

type Reader struct {
	scoring Scoring
	clock   cptime.Clock

	mu   sync.Mutex
	last answer
}

type answer struct {
	book  days.BookID
	at    time.Time
	table *scoringv1.GetTableResponse
}

var _ flagmissions.Standings = (*Reader)(nil)

// The table of the day's season in scoring. A book not opened yet has no points: a season's first day can be told
// before the gamemaster's open reached scoring. The table is settled when no round but the day's own is open.
func (r *Reader) Standing(ctx context.Context, day days.Day) (flagmissions.Standing, error) {
	table, err := r.table(ctx, day.Book())
	if err != nil {
		return flagmissions.Standing{}, err
	}

	snapshot := flagmissions.StandingSnapshot{Points: map[string]uint32{}, Settled: true}
	for _, place := range table.GetTable() {
		snapshot.Points[place.GetCountryId()] = place.GetPoints()
	}
	for _, round := range table.GetOpenRoundIds() {
		if round != day.String() {
			snapshot.Settled = false
		}
	}
	return flagmissions.StandingOf(snapshot), nil
}

func (r *Reader) table(ctx context.Context, book days.BookID) (*scoringv1.GetTableResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.clock.Now()
	if r.last.table != nil && r.last.book == book && now.Sub(r.last.at) < keptFor {
		return r.last.table, nil
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.scoring.GetTable(ctx, connect.NewRequest(&scoringv1.GetTableRequest{BookId: string(book)}))
	table := &scoringv1.GetTableResponse{}
	switch {
	case connect.CodeOf(err) == connect.CodeNotFound:
	case err != nil:
		return nil, fmt.Errorf("failed to ask scoring the table of %s: %w", book, err)
	default:
		table = res.Msg
	}
	r.last = answer{book: book, at: now, table: table}
	return table, nil
}
