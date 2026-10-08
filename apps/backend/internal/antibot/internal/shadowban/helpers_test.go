package shadowban_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
)

type banner struct {
	t     *testing.T
	inner *shadowban.Banner
}

func (b banner) Flag(scope string) (shadowban.Sentence, bool) {
	b.t.Helper()
	sentence, accepted, err := b.inner.Flag(b.t.Context(), scope)
	require.NoError(b.t, err)
	return sentence, accepted
}

func (b banner) Ban(scope string, duration time.Duration) shadowban.Sentence {
	b.t.Helper()
	sentence, err := b.inner.Ban(b.t.Context(), scope, duration)
	require.NoError(b.t, err)
	return sentence
}

func (b banner) Unban(scope string) {
	b.t.Helper()
	require.NoError(b.t, b.inner.Unban(b.t.Context(), scope))
}

func (b banner) Sentence(scope string) (shadowban.Sentence, bool) {
	b.t.Helper()
	sentence, running, err := b.inner.Sentence(b.t.Context(), scope)
	require.NoError(b.t, err)
	return sentence, running
}

func (b banner) Banned(scope string) bool {
	b.t.Helper()
	banned, err := b.inner.Banned(b.t.Context(), scope)
	require.NoError(b.t, err)
	return banned
}

func (b banner) Flagged() int {
	b.t.Helper()
	flagged, err := b.inner.Flagged(b.t.Context())
	require.NoError(b.t, err)
	return flagged
}

func (b banner) Enforcing() bool { return b.inner.Enforcing() }

type bans struct {
	t     *testing.T
	inner *shadowban.Bans
}

func (b bans) Flag(caller shadowban.Caller) (shadowban.Sentence, bool) {
	b.t.Helper()
	sentence, accepted, err := b.inner.Flag(b.t.Context(), caller)
	require.NoError(b.t, err)
	return sentence, accepted
}

func (b bans) Ban(caller shadowban.Caller, duration time.Duration) shadowban.Sentence {
	b.t.Helper()
	sentence, err := b.inner.Ban(b.t.Context(), caller, duration)
	require.NoError(b.t, err)
	return sentence
}

func (b bans) Unban(caller shadowban.Caller) {
	b.t.Helper()
	require.NoError(b.t, b.inner.Unban(b.t.Context(), caller))
}

func (b bans) Sentence(caller shadowban.Caller) (shadowban.Sentence, bool) {
	b.t.Helper()
	sentence, running, err := b.inner.Sentence(b.t.Context(), caller)
	require.NoError(b.t, err)
	return sentence, running
}

func (b bans) Banned(caller shadowban.Caller) bool {
	b.t.Helper()
	banned, err := b.inner.Banned(b.t.Context(), caller)
	require.NoError(b.t, err)
	return banned
}

func (b bans) Flagged() int {
	b.t.Helper()
	flagged, err := b.inner.Flagged(b.t.Context())
	require.NoError(b.t, err)
	return flagged
}
