package bans_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
)

type banner struct {
	t     *testing.T
	inner *bans.Banner
}

func (b banner) Flag(scope string) (bans.Sentence, bool) {
	b.t.Helper()
	sentence, accepted, err := b.inner.Flag(b.t.Context(), scope)
	require.NoError(b.t, err)
	return sentence, accepted
}

func (b banner) Ban(scope string, duration time.Duration) bans.Sentence {
	b.t.Helper()
	sentence, err := b.inner.Ban(b.t.Context(), scope, duration)
	require.NoError(b.t, err)
	return sentence
}

func (b banner) Unban(scope string) {
	b.t.Helper()
	require.NoError(b.t, b.inner.Unban(b.t.Context(), scope))
}

func (b banner) Sentence(scope string) (bans.Sentence, bool) {
	b.t.Helper()
	sentence, running, err := b.inner.Sentence(b.t.Context(), scope)
	require.NoError(b.t, err)
	return sentence, running
}

func (b banner) LastSentence(scope string) (bans.Sentence, bool) {
	b.t.Helper()
	sentence, found, err := b.inner.LastSentence(b.t.Context(), scope)
	require.NoError(b.t, err)
	return sentence, found
}

func (b banner) Banned(scope string) bool {
	b.t.Helper()
	_, banned, err := b.inner.EnforcedSentence(b.t.Context(), scope)
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

type book struct {
	t     *testing.T
	inner *bans.Book
}

func (b book) Flag(caller bans.Caller) (bans.Sentence, bool) {
	b.t.Helper()
	sentence, accepted, err := b.inner.Flag(b.t.Context(), caller)
	require.NoError(b.t, err)
	return sentence, accepted
}

func (b book) Ban(caller bans.Caller, duration time.Duration) bans.Sentence {
	b.t.Helper()
	sentence, err := b.inner.Ban(b.t.Context(), caller, duration)
	require.NoError(b.t, err)
	return sentence
}

func (b book) Unban(caller bans.Caller) {
	b.t.Helper()
	require.NoError(b.t, b.inner.Unban(b.t.Context(), caller))
}

func (b book) Sentence(caller bans.Caller) (bans.Sentence, bool) {
	b.t.Helper()
	sentence, running, err := b.inner.Sentence(b.t.Context(), caller)
	require.NoError(b.t, err)
	return sentence, running
}

func (b book) LastSentence(caller bans.Caller) (bans.Sentence, bool) {
	b.t.Helper()
	sentence, found, err := b.inner.LastSentence(b.t.Context(), caller)
	require.NoError(b.t, err)
	return sentence, found
}

func (b book) Banned(caller bans.Caller) bool {
	b.t.Helper()
	_, banned, err := b.inner.EnforcedSentence(b.t.Context(), caller)
	require.NoError(b.t, err)
	return banned
}

func (b book) Flagged() int {
	b.t.Helper()
	flagged, err := b.inner.Flagged(b.t.Context())
	require.NoError(b.t, err)
	return flagged
}
