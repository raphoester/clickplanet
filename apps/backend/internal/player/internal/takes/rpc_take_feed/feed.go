package rpc_take_feed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

const (
	askTimeout = 5 * time.Second
	limit      = 1000
)

var errNoTime = errors.New("the take has no time")

type Feed struct {
	dial Dialer
}

func New(dial Dialer) *Feed {
	return &Feed{dial: dial}
}

func (f *Feed) Start(ctx context.Context) (takes.Position, error) {
	client, err := f.client()
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := client.GetFeedStart(ctx, connect.NewRequest(&planetv1.GetFeedStartRequest{}))
	if err != nil {
		return 0, fmt.Errorf("failed to call planet.v1.InternalService/GetFeedStart: %w", err)
	}
	return takes.Position(res.Msg.GetPosition()), nil
}

func (f *Feed) Batch(ctx context.Context, from takes.Position) (takes.Batch, error) {
	client, err := f.client()
	if err != nil {
		return takes.Batch{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := client.ReadLog(ctx, connect.NewRequest(&planetv1.ReadLogRequest{FromPosition: uint64(from), Limit: limit}))
	if err != nil {
		return takes.Batch{}, fmt.Errorf("failed to call planet.v1.InternalService/ReadLog: %w", err)
	}

	next := from
	var list []takes.Take
	for _, entry := range res.Msg.GetEntries() {
		position := takes.Position(entry.GetPosition())
		if position < next {
			return takes.Batch{}, fmt.Errorf("planet answered %w: %d after %d", takes.ErrDisordered, position, next)
		}
		next = position + 1

		answered, ok := entry.GetFact().(*planetv1.LogEntry_Take)
		if !ok {
			continue
		}
		take, err := takeOf(position, answered.Take)
		if err != nil {
			return takes.Batch{}, fmt.Errorf("planet answered the take at %d: %w", position, err)
		}
		list = append(list, take)
	}

	batch, err := takes.BatchOf(from, next, list)
	if err != nil {
		return takes.Batch{}, fmt.Errorf("planet answered: %w", err)
	}
	return batch, nil
}

func (f *Feed) client() (planetv1connect.InternalServiceClient, error) {
	client, baseURL, err := f.dial.Dial()
	if err != nil {
		return nil, fmt.Errorf("failed to reach the planet module: %w", err)
	}
	return planetv1connect.NewInternalServiceClient(client, baseURL), nil
}

func takeOf(position takes.Position, answered *planetv1.Take) (takes.Take, error) {
	account := cpsession.NoAccount
	if answered.GetAccountId() != "" {
		parsed, err := players.AccountIDOf(answered.GetAccountId())
		if err != nil {
			return takes.Take{}, err //nolint:wrapcheck // the sentinel names it, and the caller names the take.
		}
		account = parsed
	}

	if err := answered.GetTakenAt().CheckValid(); err != nil {
		return takes.Take{}, fmt.Errorf("%w: %w", errNoTime, err)
	}

	return takes.TakeOf(position, account, answered.GetCountry(), answered.GetTakenAt().AsTime(), answered.GetReverted()), nil
}
