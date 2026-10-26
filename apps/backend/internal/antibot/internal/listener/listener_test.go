package listener_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/listener"
)

var at = time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC)

type recorder struct {
	calls []string
	click guard.Click
}

func (h *recorder) Attempted(click guard.Click) {
	h.calls, h.click = append(h.calls, "attempted"), click
}

func (h *recorder) Inspect(_ context.Context, click guard.Click) {
	h.calls, h.click = append(h.calls, "inspect"), click
}

func (h *recorder) Committed(click guard.Click) {
	h.calls, h.click = append(h.calls, "committed"), click
}

func (h *recorder) Caught(scope string, after time.Duration) {
	h.calls = append(h.calls, "caught "+scope+" "+after.String())
}

func (h *recorder) Missed(scope string) { h.calls = append(h.calls, "missed "+scope) }

func (h *recorder) Foreign(scope string) { h.calls = append(h.calls, "foreign "+scope) }

func (h *recorder) Fetched(scope string, maps float64, offMap bool) {
	h.calls = append(h.calls, fmt.Sprintf("fetched %s %.2f off the map %t", scope, maps, offMap))
}

func (h *recorder) Listened(scope string) { h.calls = append(h.calls, "listened "+scope) }

func tried(outcome planetv1.ClickOutcome, when time.Time) *planetv1.ClickTried {
	return &planetv1.ClickTried{
		Scope: "2001:db8::/64", AccountId: "a-guest", TileId: 42, Country: "fr", Owner: "de",
		TriedAt: timestamppb.New(when), Outcome: outcome, Position: &planetv1.GlobePoint{X: 1}, Tempo: 3,
	}
}

func TestEachOutcomeReachesTheWatchdogsAsFarAsTheClickWent(t *testing.T) {
	for outcome, calls := range map[planetv1.ClickOutcome][]string{
		planetv1.ClickOutcome_CLICK_OUTCOME_REFUSED:    nil,
		planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED:  {"attempted"},
		planetv1.ClickOutcome_CLICK_OUTCOME_OTHER_FLAG: {"attempted"},
		planetv1.ClickOutcome_CLICK_OUTCOME_INVALID:    {"attempted", "inspect"},
		planetv1.ClickOutcome_CLICK_OUTCOME_FAILED:     {"attempted", "inspect"},
		planetv1.ClickOutcome_CLICK_OUTCOME_TAKEN:      {"attempted", "inspect", "committed"},
		planetv1.ClickOutcome_CLICK_OUTCOME_STRUCK:     {"attempted", "inspect", "committed"},
		planetv1.ClickOutcome_CLICK_OUTCOME_UNCHANGED:  {"attempted", "inspect", "committed"},
	} {
		t.Run(outcome.String(), func(t *testing.T) {
			guard := &recorder{}

			require.NoError(t, listener.New(guard, listener.NewMarks()).ClickTried().Handle(t.Context(), tried(outcome, at)))

			assert.Equal(t, calls, guard.calls)
		})
	}
}

func TestATryIsHeardInTheWatchdogsWords(t *testing.T) {
	heard := &recorder{}

	require.NoError(t, listener.New(heard, listener.NewMarks()).ClickTried().
		Handle(t.Context(), tried(planetv1.ClickOutcome_CLICK_OUTCOME_STRUCK, at)))

	assert.Equal(t, guard.Click{
		Scope: "2001:db8::/64", Account: "a-guest", Tile: 42, Country: "fr", At: at, Held: "de",
		Shielded: true, Position: guard.Point{X: 1}, Tempo: 3,
	}, heard.click)
}

func TestAnOutcomeTheAntibotDoesNotKnowIsAnError(t *testing.T) {
	err := listener.New(&recorder{}, listener.NewMarks()).ClickTried().
		Handle(t.Context(), tried(planetv1.ClickOutcome_CLICK_OUTCOME_UNSPECIFIED, at))

	assert.ErrorIs(t, err, listener.ErrUnknownOutcome)
}

func TestAFactHeardAlreadyIsHeardOnce(t *testing.T) {
	guard := &recorder{}
	tries := listener.New(guard, listener.NewMarks()).ClickTried()
	first := tried(planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED, at)
	second := tried(planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED, at.Add(time.Millisecond))

	for _, fact := range []*planetv1.ClickTried{first, second, first, second} {
		require.NoError(t, tries.Handle(t.Context(), fact))
	}

	assert.Equal(t, []string{"attempted", "attempted"}, guard.calls, "a redelivery after a crash feeds no watchdog twice")
}

func TestEachKindOfFactKeepsItsOwnMark(t *testing.T) {
	guard := &recorder{}
	heard := listener.New(guard, listener.NewMarks())

	require.NoError(t, heard.ClickTried().Handle(t.Context(), tried(planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED, at)))
	require.NoError(t, heard.StreamOpened().Handle(t.Context(),
		&planetv1.StreamOpened{Scope: "1.2.3.4", OpenedAt: timestamppb.New(at.Add(-time.Second))}))
	require.NoError(t, heard.MapRead().Handle(t.Context(),
		&planetv1.MapRead{Scope: "1.2.3.4", Maps: 0.5, OffMap: true, ReadAt: timestamppb.New(at.Add(-time.Second))}))

	assert.Equal(t, []string{"attempted", "listened 1.2.3.4", "fetched 1.2.3.4 0.50 off the map true"}, guard.calls,
		"a stream read behind another is not late: each kind is told in its own order")
}

func TestTheBoxesReachTheCatcher(t *testing.T) {
	guard := &recorder{}
	heard := listener.New(guard, listener.NewMarks())

	require.NoError(t, heard.BoxCaught().Handle(t.Context(),
		&planetv1.BoxCaught{Scope: "1.2.3.4", After: durationpb.New(800 * time.Millisecond), CaughtAt: timestamppb.New(at)}))
	require.NoError(t, heard.BoxLapsed().Handle(t.Context(), &planetv1.BoxLapsed{Scope: "1.2.3.4", LapsedAt: timestamppb.New(at)}))
	require.NoError(t, heard.ForeignBoxClaimed().Handle(t.Context(),
		&planetv1.ForeignBoxClaimed{Scope: "5.6.7.8", ClaimedAt: timestamppb.New(at)}))

	assert.Equal(t, []string{"caught 1.2.3.4 800ms", "missed 1.2.3.4", "foreign 5.6.7.8"}, guard.calls)
}

func TestTheMarksSurviveARestart(t *testing.T) {
	marks := listener.NewMarks()
	require.NoError(t, listener.New(&recorder{}, marks).ClickTried().
		Handle(t.Context(), tried(planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED, at)))
	saved, err := marks.Save()
	require.NoError(t, err)

	restarted := listener.NewMarks()
	require.NoError(t, restarted.Load(saved))
	guard := &recorder{}
	tries := listener.New(guard, restarted).ClickTried()

	require.NoError(t, tries.Handle(t.Context(), tried(planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED, at)))
	require.NoError(t, tries.Handle(t.Context(), tried(planetv1.ClickOutcome_CLICK_OUTCOME_THROTTLED, at.Add(time.Second))))

	assert.Equal(t, []string{"attempted"}, guard.calls, "what the last flush had heard is not heard again")
}
