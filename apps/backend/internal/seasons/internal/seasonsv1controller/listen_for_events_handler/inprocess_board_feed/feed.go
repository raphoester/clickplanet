package inprocess_board_feed

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Reader interface {
	Board(ctx context.Context, country string) (*seasonsv1.Board, []standings.AccountID, error)
}

const (
	Every   = time.Second
	AtLeast = 15 * time.Second

	tick  = 250 * time.Millisecond
	reads = 8
)

func New(reader Reader, clock cptime.Clock) *Feed {
	return &Feed{
		reader: reader,
		clock:  clock,
		wake:   make(chan struct{}, 1),
		views:  map[string]*view{},
	}
}

type Feed struct {
	reader Reader
	clock  cptime.Clock
	wake   chan struct{}

	mu    sync.Mutex
	views map[string]*view
}

type view struct {
	streams *cpcolls.Set[chan *seasonsv1.Board]
	board   *seasonsv1.Board
	listed  *cpcolls.Set[standings.AccountID]
	readAt  time.Time
	moved   bool
}

func (v *view) due(now time.Time) bool {
	elapsed := now.Sub(v.readAt)
	return v.readAt.IsZero() || elapsed >= AtLeast || (v.moved && elapsed >= Every)
}

func (f *Feed) Subscribe(ctx context.Context, country string) <-chan *seasonsv1.Board {
	boards := make(chan *seasonsv1.Board, 1)

	f.mu.Lock()
	shown, followed := f.views[country]
	if !followed {
		shown = &view{streams: cpcolls.NewSet[chan *seasonsv1.Board](), listed: cpcolls.NewSet[standings.AccountID]()}
		f.views[country] = shown
	}
	shown.streams.Add(boards)
	if shown.board != nil {
		boards <- shown.board
	}
	f.mu.Unlock()

	if !followed {
		select {
		case f.wake <- struct{}{}:
		default:
		}
	}

	go func() {
		<-ctx.Done()

		f.mu.Lock()
		defer f.mu.Unlock()

		shown.streams.Delete(boards)
		if shown.streams.Empty() {
			delete(f.views, country)
		}
		close(boards)
	}()

	return boards
}

func (f *Feed) MarkTaken(take standings.Take) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, country := range []string{"", string(take.Country)} {
		if shown, followed := f.views[country]; followed {
			shown.moved = true
		}
	}
}

func (f *Feed) MarkForgotten(account standings.AccountID) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, shown := range f.views {
		if shown.listed.Contains(account) {
			shown.moved = true
		}
	}
}

func (f *Feed) Name() string { return "seasons-boards" }

func (f *Feed) Run(ctx context.Context) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-f.wake:
		}
		f.Refresh(ctx)
	}
}

func (f *Feed) Refresh(ctx context.Context) {
	group := errgroup.Group{}
	group.SetLimit(reads)
	for _, country := range f.due() {
		group.Go(func() error {
			f.read(ctx, country)
			return nil
		})
	}
	_ = group.Wait()
}

func (f *Feed) due() []string {
	now := f.clock.Now()

	f.mu.Lock()
	defer f.mu.Unlock()

	var due []string
	for country, shown := range f.views {
		if shown.due(now) {
			shown.moved = false
			shown.readAt = now
			due = append(due, country)
		}
	}
	return due
}

func (f *Feed) read(ctx context.Context, country string) {
	board, accounts, err := f.reader.Board(ctx, country)

	f.mu.Lock()
	defer f.mu.Unlock()

	shown, followed := f.views[country]
	if !followed {
		return
	}
	if err != nil {
		shown.moved = true
		return
	}
	shown.listed = cpcolls.NewSet(accounts...)
	if shown.board != nil && proto.Equal(shown.board, board) {
		return
	}
	shown.board = board
	shown.streams.ForEach(func(stream chan *seasonsv1.Board) {
		select {
		case <-stream:
		default:
		}
		stream <- board
	})
}
