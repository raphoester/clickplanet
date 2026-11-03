package missions_paid_subscriber

import (
	"context"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type Streams interface {
	Publish(account missions.AccountID)
}

func New(streams Streams) Subscriber {
	return Subscriber{streams: streams}
}

type Subscriber struct {
	streams Streams
}

var _ cpredis.Handler[*missionsv1.MissionsPaid] = Subscriber{}

func (s Subscriber) Handle(_ context.Context, event *missionsv1.MissionsPaid) error {
	account, err := missions.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the sentinel already says what is wrong.
	}

	s.streams.Publish(account)
	return nil
}
