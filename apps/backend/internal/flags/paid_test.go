package flags_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog/postgres_flag_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/commands/grant_time_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/commands/take_back_time_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/tell_streams_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/postgres_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/usecases/grant_time_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants/usecases/take_back_time_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/postgres_submission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/subscribers/told_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpoutbox"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis/cpredistest"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// The shop's word that a founder paid opens the approved submission's flag for every open page, a renewal told twice
// adds its month once, and a refund takes the first payment's time back.
func TestTheShopsPaymentsOpenAndKeepAFlagAndARefundTakesItsTimeBack(t *testing.T) {
	db := cppgtest.StartServer(t).OpenSchema(t, "flags", migrations.FS)
	redis := cpredistest.StartServer(t).Config()
	logger := slog.New(slog.DiscardHandler)
	streams := cpredis.New(redis, prometheus.NewRegistry())
	require.NoError(t, streams.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = streams.Close() })
	shop := cpredis.New(redis, prometheus.NewRegistry())
	require.NoError(t, shop.ConnectCtx(t.Context()))
	t.Cleanup(func() { _ = shop.Close() })

	now := time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	clock := cptime.NewFixedClock(now)
	flags, submitted := postgres_flag_store.New(db), postgres_submission_store.New(db)
	approved := submissions.TestWaiting(submissions.SubmissionID{15: 1}, submissions.AccountID{15: 1}, now)
	approved.State, approved.Held = submissions.Approved, false
	require.NoError(t, submitted.Save(t.Context(), submissions.Restored(approved)))

	board := feed.NewBoard(8)
	page, leave := board.Attend()
	t.Cleanup(leave)
	tell := tell_streams_usecase.New(feed.NewsSince(now.Add(-time.Hour)), flags, board)
	opened, err := cpredis.Subscribe(t.Context(), streams, "flags-streams-opened", told_subscriber.NewOpened(tell))
	require.NoError(t, err)
	closed, err := cpredis.Subscribe(t.Context(), streams, "flags-streams-closed", told_subscriber.NewClosed(tell))
	require.NoError(t, err)
	paidTime := postgres_grant_store.New(db)
	granted, err := cpredis.Handle(t.Context(), streams, grant_time_handler.New(grant_time_usecase.New(
		catalog.NewOpening(&catalog.Sequence{}, clock), clock, db,
		grant_time_usecase.Stores{Grants: paidTime, Submissions: submitted, Flags: flags}, cpoutbox.New(db),
	), logger))
	require.NoError(t, err)
	takenBack, err := cpredis.Handle(t.Context(), streams, take_back_time_handler.New(
		take_back_time_usecase.New(clock, db, paidTime, flags, cpoutbox.New(db)), logger))
	require.NoError(t, err)
	running(t, opened, closed, granted, takenBack, cpoutbox.NewRelay("flags-outbox", db, streams, logger))

	month := func(grant string) *flagsv1.GrantTime {
		return &flagsv1.GrantTime{
			GrantId: grant, Period: flagsv1.TimePeriod_TIME_PERIOD_MONTH, PaidAt: timestamppb.New(now),
			Target: &flagsv1.TimeTarget{Target: &flagsv1.TimeTarget_SubmissionId{
				SubmissionId: submissions.SubmissionID{15: 1}.String(),
			}},
		}
	}
	require.NoError(t, shop.Commands().Send(t.Context(), month("stripe:in_1")))
	flag := <-page
	assert.Equal(t, catalog.Open, flag.State, "the page hears it opened")
	assert.Equal(t, now.AddDate(0, 1, 0), flag.EndsAt)

	require.NoError(t, shop.Commands().Send(t.Context(), month("stripe:in_2"), month("stripe:in_2")))
	require.Eventually(t, func() bool {
		kept, err := flags.Flag(t.Context(), flag.ID)
		return err == nil && kept.TakeSnapshot().EndsAt.Equal(now.AddDate(0, 2, 0))
	}, 10*time.Second, 20*time.Millisecond, "the renewal adds its month, once")

	clock.Advance(time.Hour)
	require.NoError(t, shop.Commands().Send(t.Context(), &flagsv1.TakeBackTime{
		GrantId: "stripe:in_2", DecidedAt: timestamppb.New(clock.Now()),
	}))
	require.Eventually(t, func() bool {
		kept, err := flags.Flag(t.Context(), flag.ID)
		return err == nil && kept.TakeSnapshot().EndsAt.Equal(now.AddDate(0, 1, 0))
	}, 10*time.Second, 20*time.Millisecond, "the refund takes the renewal's month back")

	require.NoError(t, shop.Commands().Send(t.Context(), &flagsv1.TakeBackTime{
		GrantId: "stripe:in_1", DecidedAt: timestamppb.New(clock.Now()),
	}))
	closedFlag := <-page
	assert.Equal(t, catalog.Closed, closedFlag.State, "a refund of the first month closes the flag it opened")
}
