package try_click_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/try_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	at      = time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC)
	created = at.Add(-time.Hour)
)

type answer struct {
	out click_usecase.Out
	err error
}

func (a answer) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	return a.out, a.err
}

type owners map[uint32]string

func (o owners) Owner(tile uint32) (string, bool) {
	owner, ok := o[tile]
	return owner, ok
}

func tempo(pace float64) func() clicks.Throttle {
	return func() clicks.Throttle { return clicks.Throttle{Tempo: pace} }
}

type positions map[uint32]clicks.Vec3

func (p positions) Position(tile uint32) (clicks.Vec3, bool) {
	at, ok := p[tile]
	return at, ok
}

func try(t *testing.T, ctx context.Context, inner answer, tile uint32) (*planetv1.ClickTried, click_usecase.Out, error) {
	t.Helper()
	log := facts.NewLog(cptime.NewFixedClock(at))

	out, err := try_click.New(inner, owners{42: "de"}, positions{42: {X: 1, Y: 0, Z: 0}}, tempo(3), log).
		Execute(ctx, click_usecase.In{TileID: tile, CountryID: "fr"})

	told := log.Take()
	require.Len(t, told, 1, "every try is told once")
	tried, ok := told[0].(*planetv1.ClickTried)
	require.True(t, ok)
	return tried, out, err
}

func TestATryNamesWhoClickedWhatAndWhen(t *testing.T) {
	ctx := cpctx.AddAccountCreatedToContext(cpctx.AddLinkedToContext(cpctx.AddAccountToContext(
		cpctx.AddIPToContext(t.Context(), "2001:db8::9"), "a-player")), created)
	inner := answer{out: click_usecase.Out{Outcome: clicks.Taken, Owner: "de", Limited: true}}

	tried, out, err := try(t, ctx, inner, 42)

	require.NoError(t, err)
	assert.Equal(t, inner.out, out, "the answer is the inner one, untouched")
	assert.True(t, proto.Equal(&planetv1.ClickTried{
		Scope: "2001:db8::/64", AccountId: "a-player", Linked: true, AccountCreatedAt: timestamppb.New(created),
		TileId: 42, Country: "fr", TriedAt: timestamppb.New(at), Outcome: planetv1.ClickOutcome_CLICK_OUTCOME_TAKEN,
		Owner: "de", Position: &planetv1.GlobePoint{X: 1}, Tempo: 3,
	}, tried), "got %v", tried)
}

func TestEachAnswerIsAnOutcome(t *testing.T) {
	for name, tc := range map[string]struct {
		inner   answer
		outcome planetv1.ClickOutcome
	}{
		"taken":     {answer{out: click_usecase.Out{Outcome: clicks.Taken}}, planetv1.ClickOutcome_CLICK_OUTCOME_TAKEN},
		"struck":    {answer{out: click_usecase.Out{Outcome: clicks.Shielded}}, planetv1.ClickOutcome_CLICK_OUTCOME_STRUCK},
		"unchanged": {answer{out: click_usecase.Out{Outcome: clicks.Unchanged}}, planetv1.ClickOutcome_CLICK_OUTCOME_UNCHANGED},
		"refused":   {answer{err: clicks.NewRefusal(at.Add(time.Hour))}, planetv1.ClickOutcome_CLICK_OUTCOME_REFUSED},
		"throttled": {answer{err: clicks.ErrThrottled}, planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED},
		"another flag": {
			answer{err: fmt.Errorf("may not play: %w", allegiances.Refusal{Flag: "de", Reason: allegiances.ErrOtherFlag})},
			planetv1.ClickOutcome_CLICK_OUTCOME_OTHER_FLAG,
		},
		"no flag": {
			answer{err: fmt.Errorf("may not play: %w", allegiances.Refusal{Reason: allegiances.ErrNoFlag})},
			planetv1.ClickOutcome_CLICK_OUTCOME_OTHER_FLAG,
		},
		"unknown flag":      {answer{err: fmt.Errorf("%w: zz", clicks.ErrUnknownCountry)}, planetv1.ClickOutcome_CLICK_OUTCOME_INVALID},
		"off the map":       {answer{err: clicks.ErrTileOutOfRange}, planetv1.ClickOutcome_CLICK_OUTCOME_INVALID},
		"bonuses together":  {answer{err: clicks.ErrBonusesTogether}, planetv1.ClickOutcome_CLICK_OUTCOME_INVALID},
		"the map failed it": {answer{err: errors.New("disk on fire")}, planetv1.ClickOutcome_CLICK_OUTCOME_FAILED},
	} {
		t.Run(name, func(t *testing.T) {
			tried, _, err := try(t, cpctx.AddIPToContext(t.Context(), "203.0.113.7"), tc.inner, 42)

			assert.Equal(t, tc.inner.err, err, "the answer is the inner one, untouched")
			assert.Equal(t, tc.outcome, tried.GetOutcome())
		})
	}
}

func TestATryTheGameAnsweredWithAnErrorIsToldWithTheFlagTheTileWears(t *testing.T) {
	tried, _, _ := try(t, cpctx.AddIPToContext(t.Context(), "203.0.113.7"), answer{err: clicks.ErrThrottled}, 42)

	assert.Equal(t, "de", tried.GetOwner())
}

func TestATryOffTheMapHasNoPosition(t *testing.T) {
	tried, _, _ := try(t, cpctx.AddIPToContext(t.Context(), "203.0.113.7"), answer{err: clicks.ErrTileOutOfRange}, 0)

	assert.Nil(t, tried.GetPosition())
	assert.Empty(t, tried.GetOwner())
	assert.Nil(t, tried.GetAccountCreatedAt(), "a click with no account")
}
