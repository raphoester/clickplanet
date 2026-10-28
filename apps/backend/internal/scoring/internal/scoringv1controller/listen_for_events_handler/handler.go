package listen_for_events_handler

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler/board_query"
)

type Boards interface {
	Subscribe(ctx context.Context, view board_query.View) <-chan *scoringv1.Board
}

type Scores interface {
	Subscribe(ctx context.Context, book books.BookID) <-chan *scoringv1.Scores
}

type Rounds interface {
	Subscribe(ctx context.Context, round books.RoundID) <-chan *scoringv1.Round
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

var (
	ErrUnknownCountry = errors.New("not a country")
	ErrTooManyRounds  = fmt.Errorf("a stream follows %d rounds at most", MaxRounds)
)

const (
	// Well under Cloudflare's ~125s idle cut on a silent stream.
	DefaultHeartbeat = 30 * time.Second
	MaxRounds        = 10
)

type Feeds struct {
	Boards Boards
	Scores Scores
	Rounds Rounds
}

func New(feeds Feeds, countries CountryChecker, heartbeat time.Duration) ListenForEventsHandler {
	if heartbeat <= 0 {
		heartbeat = DefaultHeartbeat
	}
	return ListenForEventsHandler{feeds: feeds, countries: countries, heartbeat: heartbeat}
}

type ListenForEventsHandler struct {
	feeds     Feeds
	countries CountryChecker
	heartbeat time.Duration
}

type asked struct {
	book    books.BookID
	country string
	rounds  []books.RoundID
}

func (h ListenForEventsHandler) askedOf(req *scoringv1.ListenForEventsRequest) (asked, error) {
	var (
		wanted = asked{country: req.GetCountryId()}
		err    error
	)
	if wanted.country != "" && !h.countries.CheckCountry(wanted.country) {
		return asked{}, fmt.Errorf("%w: %q", ErrUnknownCountry, wanted.country)
	}
	if req.GetBookId() != "" {
		if wanted.book, err = books.ParseBookID(req.GetBookId()); err != nil {
			return asked{}, err //nolint:wrapcheck // the id says why.
		}
	}
	if len(req.GetRoundIds()) > MaxRounds {
		return asked{}, ErrTooManyRounds
	}
	for _, id := range req.GetRoundIds() {
		round, err := books.ParseRoundID(id)
		if err != nil {
			return asked{}, err //nolint:wrapcheck // the id says why.
		}
		if !slices.Contains(wanted.rounds, round) {
			wanted.rounds = append(wanted.rounds, round)
		}
	}
	return wanted, nil
}

func (h ListenForEventsHandler) ListenForEvents(
	ctx context.Context,
	req *connect.Request[scoringv1.ListenForEventsRequest],
	stream *connect.ServerStream[scoringv1.ScoringEvent],
) error {
	wanted, err := h.askedOf(req.Msg)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		boards <-chan *scoringv1.Board
		scores <-chan *scoringv1.Scores
	)
	if wanted.book != "" {
		boards = h.feeds.Boards.Subscribe(ctx, board_query.View{Book: wanted.book, Country: wanted.country})
		scores = h.feeds.Scores.Subscribe(ctx, wanted.book)
	}
	rounds := h.rounds(ctx, wanted.rounds)

	heartbeat := time.NewTicker(h.heartbeat)
	defer heartbeat.Stop()

	for {
		var event *scoringv1.ScoringEvent
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeat.C:
			event = &scoringv1.ScoringEvent{Event: &scoringv1.ScoringEvent_Heartbeat{Heartbeat: &scoringv1.Heartbeat{}}}
		case board, open := <-boards:
			if !open {
				return nil
			}
			event = &scoringv1.ScoringEvent{Event: &scoringv1.ScoringEvent_Board{Board: board}}
		case scored, open := <-scores:
			if !open {
				return nil
			}
			event = &scoringv1.ScoringEvent{Event: &scoringv1.ScoringEvent_Scores{Scores: scored}}
		case round := <-rounds:
			event = &scoringv1.ScoringEvent{Event: &scoringv1.ScoringEvent_Round{Round: round}}
		}
		if err := stream.Send(event); err != nil {
			return err //nolint:wrapcheck // the stream's own error.
		}
	}
}

// rounds follows each round asked into one channel, until ctx ends.
func (h ListenForEventsHandler) rounds(ctx context.Context, ids []books.RoundID) <-chan *scoringv1.Round {
	merged := make(chan *scoringv1.Round, len(ids))
	for _, id := range ids {
		followed := h.feeds.Rounds.Subscribe(ctx, id)
		go func() {
			for round := range followed {
				select {
				case merged <- round:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	return merged
}
