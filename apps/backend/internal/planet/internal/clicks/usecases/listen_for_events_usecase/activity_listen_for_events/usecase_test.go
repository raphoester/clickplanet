package activity_listen_for_events_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase/activity_listen_for_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const account = "00000000-0000-4000-8000-000000000001"

type stubUseCase struct {
	err error
}

func (s stubUseCase) Execute(context.Context, listen_for_events_usecase.Sink) error {
	return s.err
}

type discardSink struct{}

func (discardSink) Send(listen_for_events_usecase.Event) error { return nil }

func TestAStreamOpenedIsRecordedWithItsAccount(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	recorded := &activity.Recorded{}
	cause := errors.New("the client went away")
	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), account)

	err := activity_listen_for_events.New(stubUseCase{err: cause}, recorded, cptime.NewFixedClock(now)).
		Execute(ctx, discardSink{})

	require.ErrorIs(t, err, cause, "the answer is the inner one, untouched")
	assert.Equal(t, []activity.Event{{
		At: now, Kind: activity.KindStream,
		Caller: activity.Caller{Scope: "2001:db8::/64", Account: cpsession.AccountID(uuid.MustParse(account))},
	}}, recorded.Events())
}
