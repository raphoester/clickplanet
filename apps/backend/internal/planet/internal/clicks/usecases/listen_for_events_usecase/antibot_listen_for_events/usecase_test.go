package antibot_listen_for_events_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase/antibot_listen_for_events"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type fakeGuard struct {
	scopes []string
}

func (g *fakeGuard) Listened(scope string) { g.scopes = append(g.scopes, scope) }

type stubUseCase struct {
	guard *fakeGuard
	seen  []string // what the guard held when the stream started
	err   error
}

func (s *stubUseCase) Execute(context.Context, listen_for_events_usecase.Sink) error {
	s.seen = slices.Clone(s.guard.scopes)
	return s.err
}

type discard struct{}

func (discard) Send(listen_for_events_usecase.Event) error { return nil }

func TestAStreamIsReportedBeforeItIsFollowed(t *testing.T) {
	guard := &fakeGuard{}
	inner := &stubUseCase{guard: guard}

	err := antibot_listen_for_events.New(inner, guard).
		Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), discard{})

	require.NoError(t, err)
	assert.Equal(t, []string{"2001:db8::/64"}, inner.seen, "a stream lasts hours: it counts when it opens")
}

func TestAStreamErrorIsWrapped(t *testing.T) {
	guard := &fakeGuard{}
	cause := errors.New("subscriber gone")

	err := antibot_listen_for_events.New(&stubUseCase{guard: guard, err: cause}, guard).
		Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), discard{})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, err.Error(), "failed to follow the planet")
	assert.Equal(t, []string{"203.0.113.7"}, guard.scopes, "an open that failed still opened")
}
