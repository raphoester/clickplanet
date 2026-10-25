package list_convictions_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/list_convictions_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 10, 31, 21, 15, 0, 0, time.UTC)

type stubGuard struct {
	off          bool
	since, until time.Time
	listed       []antibot.Conviction
	err          error
}

func (s *stubGuard) Enabled() bool { return !s.off }

func (s *stubGuard) Convictions(_ context.Context, since, until time.Time) ([]antibot.Conviction, error) {
	s.since, s.until = since, until
	return s.listed, s.err
}

func list(
	t *testing.T,
	guard *stubGuard,
	req *planetv1.ListConvictionsRequest,
) (*connect.Response[planetv1.ListConvictionsResponse], error) {
	t.Helper()
	//nolint:wrapcheck // the tests read the connect code.
	return list_convictions_handler.New(guard, cptime.NewFixedClock(at)).ListConvictions(t.Context(), connect.NewRequest(req))
}

func TestEachConvictionComesBackWhole(t *testing.T) {
	guard := &stubGuard{listed: []antibot.Conviction{{
		At: at.Add(-time.Minute), Scope: "2001:db8:1::/64", Account: "account-b", Alongside: "account-a",
		Flags: 2, Offence: 1, BannedUntil: at.Add(24 * time.Hour),
		Readings: []antibot.Reading{
			{Watchdog: "retaker", Level: "clear"},
			{Watchdog: "catcher", Level: "certain", Evidence: "catch median=400ms", At: at.Add(-2 * time.Minute)},
		},
		Clicks: 300, ActiveFor: time.Hour, LongestGap: 2 * time.Second, TopCountry: "dz", TopCountryClicks: 290,
	}}}

	res, err := list(t, guard, &planetv1.ListConvictionsRequest{Since: timestamppb.New(at.Add(-time.Hour))})
	require.NoError(t, err)

	assert.Equal(t, at.Add(-time.Hour), guard.since)
	assert.Equal(t, at, guard.until, "an unset until is now")
	assert.True(t, proto.Equal(&planetv1.ListConvictionsResponse{Convictions: []*planetv1.Conviction{{
		At: timestamppb.New(at.Add(-time.Minute)), Scope: "2001:db8:1::/64", AccountId: "account-b",
		AlongsideAccountId: "account-a", Flags: 2, Offence: 1, BannedUntil: timestamppb.New(at.Add(24 * time.Hour)),
		Readings: []*planetv1.WatchdogReading{
			{Watchdog: "retaker", Level: "clear"},
			{Watchdog: "catcher", Level: "certain", Evidence: "catch median=400ms", At: timestamppb.New(at.Add(-2 * time.Minute))},
		},
		Clicks: 300, ActiveFor: durationpb.New(time.Hour), LongestGap: durationpb.New(2 * time.Second),
		TopCountry: "dz", TopCountryClicks: 290,
	}}}, res.Msg), res.Msg.String())
}

func TestTheWindowAskedIsTheWindowRead(t *testing.T) {
	guard := &stubGuard{}

	_, err := list(t, guard, &planetv1.ListConvictionsRequest{
		Since: timestamppb.New(at.Add(-2 * time.Hour)), Until: timestamppb.New(at.Add(-time.Hour)),
	})
	require.NoError(t, err)

	assert.Equal(t, at.Add(-2*time.Hour), guard.since)
	assert.Equal(t, at.Add(-time.Hour), guard.until)
}

func TestAWindowWithNoSinceOrBackwardsIsTheCallersMistake(t *testing.T) {
	for name, req := range map[string]*planetv1.ListConvictionsRequest{
		"no since":  {},
		"backwards": {Since: timestamppb.New(at), Until: timestamppb.New(at.Add(-time.Hour))},
		"empty":     {Since: timestamppb.New(at), Until: timestamppb.New(at)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := list(t, &stubGuard{}, req)
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}
}

func TestTheAntibotOffIsAPrecondition(t *testing.T) {
	_, err := list(t, &stubGuard{off: true}, &planetv1.ListConvictionsRequest{Since: timestamppb.New(at.Add(-time.Hour))})

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestAStoreThatFailsIsNotTheCallersFault(t *testing.T) {
	cause := errors.New("postgres is down")

	_, err := list(t, &stubGuard{err: cause}, &planetv1.ListConvictionsRequest{Since: timestamppb.New(at.Add(-time.Hour))})

	require.ErrorIs(t, err, cause)
}
