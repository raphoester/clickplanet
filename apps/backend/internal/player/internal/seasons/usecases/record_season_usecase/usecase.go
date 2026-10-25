package record_season_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
)

type Records interface {
	Record(ctx context.Context, account players.AccountID) (seasons.Record, error)
	Save(ctx context.Context, record seasons.Record) error
}

type Publisher interface {
	Publish(event proto.Message)
}

type UseCase struct {
	records Records
	events  Publisher
}

func New(records Records, events Publisher) *UseCase {
	return &UseCase{records: records, events: events}
}

type In struct {
	Account players.AccountID
	Season  seasons.Season
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	record, err := u.records.Record(ctx, in.Account)
	if err != nil {
		return fmt.Errorf("failed to read the seasons played: %w", err)
	}
	if err := u.records.Save(ctx, record.With(in.Season)); err != nil {
		return fmt.Errorf("failed to record the season: %w", err)
	}

	u.events.Publish(&playerv1.SeasonRecorded{AccountId: in.Account.String()})
	return nil
}
